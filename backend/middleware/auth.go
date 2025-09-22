package middleware

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/Neat-Snap/blueprint-backend/services"
	"gorm.io/gorm"
)

type contextKey string

const (
	UserEmailContextKey  contextKey = "userEmail"
	UserObjectContextKey contextKey = "userObject"
)

type MiddlewareSkipper func(*http.Request) bool

func DefaultSkipper(r *http.Request) bool {
	if r.Method == http.MethodOptions {
		return true
	}

	path := r.URL.Path
	switch {
	case path == "/health",
		strings.HasPrefix(path, "/auth/login"),
		strings.HasPrefix(path, "/auth/signup"),
		strings.HasPrefix(path, "/auth/callback"),
		strings.HasPrefix(path, "/auth/password/reset"),
		strings.HasPrefix(path, "/auth/password/confirm"),
		strings.HasPrefix(path, "/auth/resend-email"):
		return true
	}
	return false
}

func AuthMiddlewareBuilder(logger logger.MultiLogger, conn *db.Connection, workos *services.WorkOSAuthService, secureCookies bool, skipFunc MiddlewareSkipper) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skipFunc != nil && skipFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			tokenCookie, err := r.Cookie(services.AccessTokenCookieName)
			if err != nil || tokenCookie == nil || strings.TrimSpace(tokenCookie.Value) == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			claims, err := workos.ParseAccessToken(tokenCookie.Value)
			if err != nil {
				clearAuthCookies(w, secureCookies)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			identity, err := conn.Auth.FindAuthIdentity(r.Context(), services.WorkOSProvider, claims.Subject)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, context.Canceled) {
					clearAuthCookies(w, secureCookies)
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				logger.Warn("auth: failed to load auth identity", "error", err)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			refreshToken := ""
			if identity.RefreshToken != nil {
				refreshToken = strings.TrimSpace(*identity.RefreshToken)
			}
			if refreshToken == "" {
				clearAuthCookies(w, secureCookies)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if time.Until(claims.ExpiresAt) <= 0 {
				refreshed, err := workos.AuthenticateWithRefreshToken(r.Context(), refreshToken, clientIP(r), r.UserAgent())
				if err != nil {
					clearAuthCookies(w, secureCookies)
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}

				newClaims, err := workos.ParseAccessToken(refreshed.AccessToken)
				if err != nil {
					clearAuthCookies(w, secureCookies)
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				if err := conn.Auth.UpdateIdentityTokens(r.Context(), identity.ID, refreshed.AccessToken, refreshed.RefreshToken); err != nil {
					logger.Warn("auth: failed to persist refreshed tokens", "error", err)
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				setAuthCookies(w, refreshed.AccessToken, newClaims, secureCookies)
				claims = newClaims
			}

			user, err := conn.Auth.FindUserByAuthIdentity(r.Context(), identity)
			if err != nil {
				logger.Warn("auth: failed to load user", "error", err)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			email := ""
			if user.Email != nil {
				email = *user.Email
			}

			ctx := context.WithValue(r.Context(), UserEmailContextKey, email)
			ctx = context.WithValue(ctx, UserObjectContextKey, user)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func setAuthCookies(w http.ResponseWriter, token string, claims services.AccessTokenClaims, secure bool) {
	maxAge := int(time.Until(claims.ExpiresAt).Seconds())
	if maxAge <= 0 {
		maxAge = 3600
	}
	accessCookie := &http.Cookie{
		Name:     services.AccessTokenCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	}
	if !claims.ExpiresAt.IsZero() {
		accessCookie.Expires = claims.ExpiresAt
	}
	http.SetCookie(w, accessCookie)

	if claims.SessionID != "" {
		sessionCookie := &http.Cookie{
			Name:     services.SessionIDCookieName,
			Value:    claims.SessionID,
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   maxAge,
		}
		if !claims.ExpiresAt.IsZero() {
			sessionCookie.Expires = claims.ExpiresAt
		}
		http.SetCookie(w, sessionCookie)
	}
}

func clearAuthCookies(w http.ResponseWriter, secure bool) {
	expired := time.Unix(0, 0)
	http.SetCookie(w, &http.Cookie{
		Name:     services.AccessTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  expired,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     services.SessionIDCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  expired,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
