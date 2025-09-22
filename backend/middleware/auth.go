package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/Neat-Snap/blueprint-backend/workosclient"
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
		strings.HasPrefix(path, "/auth/refresh"),
		strings.HasPrefix(path, "/auth/password/reset"),
		strings.HasPrefix(path, "/auth/password/confirm"),
		strings.HasPrefix(path, "/auth/verify/resend"),
		strings.HasPrefix(path, "/auth/logout"):
		return true
	}
	return false
}

func AuthMiddlewareBuilder(workos *workosclient.Client, logger logger.MultiLogger, conn *db.Connection, skipFunc MiddlewareSkipper) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Debug("auth: processing auth header with middleware")
			if skipFunc != nil && skipFunc(r) {
				logger.Debug("auth: skipping auth middleware")
				next.ServeHTTP(w, r)
				return
			}

			token := workos.AccessTokenFromRequest(r)
			if token == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			claims, err := workos.ParseAndValidateAccessToken(r.Context(), token)
			if err != nil {
				logger.Debug("auth: token validation failed", "error", err)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			sub, _ := claims["sub"].(string)
			if strings.TrimSpace(sub) == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			dbUser, err := workos.EnsureLocalUserByID(r.Context(), conn, sub)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					logger.Debug("auth: user not found while ensuring local record", "workos_id", sub)
				} else {
					logger.Debug("auth: error occured in the middleware", "error", err)
				}
				logger.Debug("auth: error occured in the middleware")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			email := ""
			if dbUser.Email != nil {
				email = *dbUser.Email
			}

			ctx := context.WithValue(r.Context(), UserEmailContextKey, email)
			ctx = context.WithValue(ctx, UserObjectContextKey, dbUser)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
