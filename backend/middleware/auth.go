package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/Neat-Snap/blueprint-backend/workos"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type contextKey string

const (
	UserEmailContextKey  contextKey = "userEmail"
	UserObjectContextKey contextKey = "userObject"
	WorkOSUserContextKey contextKey = "workosUser"
	defaultCookieName               = "workos_session"
)

const workOSProvider = "workos"

type MiddlewareSkipper func(*http.Request) bool

func DefaultSkipper(r *http.Request) bool {
	if r.Method == http.MethodOptions {
		return true
	}

	path := r.URL.Path
	switch {
	case path == "/health",
		strings.HasPrefix(path, "/auth/workos/login"),
		strings.HasPrefix(path, "/auth/workos/callback"):
		return true
	}
	return false
}

// AuthMiddlewareConfig configures the WorkOS authentication middleware.
type AuthMiddlewareConfig struct {
	Logger     logger.MultiLogger
	Connection *db.Connection
	Validator  *workos.Validator

	Skipper MiddlewareSkipper

	CookieName     string
	CookieSecure   bool
	CookieSameSite http.SameSite
}

// AuthMiddlewareBuilder constructs a middleware that validates WorkOS sessions and
// loads the corresponding local user from the database.
func AuthMiddlewareBuilder(cfg AuthMiddlewareConfig) func(http.Handler) http.Handler {
	cookieName := cfg.CookieName
	if cookieName == "" {
		cookieName = defaultCookieName
	}

	sameSite := cfg.CookieSameSite
	if sameSite == 0 {
		sameSite = http.SameSiteLaxMode
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skipper != nil && cfg.Skipper(r) {
				next.ServeHTTP(w, r)
				return
			}

			if cfg.Validator == nil || cfg.Connection == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			cookie, err := r.Cookie(cookieName)
			if err != nil || cookie == nil || cookie.Value == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			result, parseErr := cfg.Validator.ParseAccessToken(r.Context(), cookie.Value)
			if parseErr != nil && !errors.Is(parseErr, jwt.ErrTokenExpired) {
				cfg.Logger.Debug("auth: failed to parse WorkOS access token", "error", parseErr)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if result == nil {
				cfg.Logger.Debug("auth: validator returned no result")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			authIdentity, err := cfg.Connection.Auth.FindAuthIdentity(r.Context(), workOSProvider, result.UserID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					cfg.Logger.Debug("auth: workos identity not found", "user_id", result.UserID)
				} else {
					cfg.Logger.Debug("auth: failed to load workos identity", "error", err)
				}
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			dbUser := authIdentity.User
			if dbUser == nil {
				dbUser, err = cfg.Connection.Auth.FindUserByAuthIdentity(r.Context(), authIdentity)
				if err != nil {
					cfg.Logger.Debug("auth: failed to load user for identity", "error", err)
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
			}

			if errors.Is(parseErr, jwt.ErrTokenExpired) {
				if authIdentity.RefreshToken == nil || *authIdentity.RefreshToken == "" {
					cfg.Logger.Debug("auth: missing refresh token for expired session", "user_id", result.UserID)
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}

				refreshed, refreshErr := cfg.Validator.Refresh(r.Context(), *authIdentity.RefreshToken)
				if refreshErr != nil {
					cfg.Logger.Debug("auth: failed to refresh token", "error", refreshErr)
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}

				result = refreshed
				if err := cfg.Connection.DBConn.WithContext(r.Context()).Model(authIdentity).Updates(map[string]interface{}{
					"access_token":  refreshed.AccessToken,
					"refresh_token": refreshed.RefreshToken,
				}).Error; err != nil {
					cfg.Logger.Warn("auth: failed to persist refreshed tokens", "error", err)
				} else {
					authIdentity.AccessToken = &refreshed.AccessToken
					authIdentity.RefreshToken = &refreshed.RefreshToken
				}

				setSessionCookie(w, cookieName, refreshed.AccessToken, refreshed.ExpiresAt, cfg.CookieSecure, sameSite)
			} else {
				setSessionCookie(w, cookieName, result.AccessToken, result.ExpiresAt, cfg.CookieSecure, sameSite)
			}

			if shouldUpdateUser(dbUser, result) {
				if err := cfg.Connection.Users.Update(r.Context(), dbUser); err != nil {
					cfg.Logger.Warn("auth: failed to sync user details", "error", err)
				}
			}

			ctx := context.WithValue(r.Context(), UserEmailContextKey, result.Email)
			ctx = context.WithValue(ctx, UserObjectContextKey, dbUser)
			ctx = context.WithValue(ctx, WorkOSUserContextKey, result)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func shouldUpdateUser(u *db.User, res *workos.ValidationResult) bool {
	if u == nil || res == nil {
		return false
	}

	updated := false
	if res.Email != "" {
		if u.Email == nil || *u.Email != res.Email {
			email := res.Email
			u.Email = &email
			updated = true
		}
	}

	if res.EmailVerified && u.EmailVerifiedAt == nil {
		now := time.Now()
		u.EmailVerifiedAt = &now
		updated = true
	}

	return updated
}

func setSessionCookie(w http.ResponseWriter, name, value string, expiresAt time.Time, secure bool, sameSite http.SameSite) {
	if value == "" {
		return
	}

	maxAge := 0
	if !expiresAt.IsZero() {
		if delta := time.Until(expiresAt); delta > 0 {
			maxAge = int(delta.Seconds())
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   maxAge,
		Expires:  expiresAt,
	})
}
