package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/Neat-Snap/blueprint-backend/utils"
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
		strings.HasPrefix(path, "/auth/confirm-email"),

		strings.HasPrefix(path, "/auth/google"),
		strings.HasPrefix(path, "/auth/github"),
		strings.HasPrefix(path, "/auth/azuread"):
		return true
	}
	return false
}

func AuthMiddlewareBuilder(secret string, logger logger.MultiLogger, conn *db.Connection, skipFunc MiddlewareSkipper) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Debug("auth: processing auth header with middleware")
			if skipFunc != nil && skipFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			var token string
			if c, err := r.Cookie("token"); err == nil && c != nil && c.Value != "" {
				token = c.Value
			} else {
				authz := r.Header.Get("Authorization")
				parts := strings.SplitN(authz, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					token = parts[1]
				}
			}
			if token == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			email, err := utils.DecodeJWT([]byte(secret), token)
			if err != nil {
				logger.Error("auth: error during jwt decoding", "error", err)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			dbUser, err := conn.Users.ByEmail(r.Context(), email)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					logger.Debug("auth: record for email was not found in the database during middleware check", "email", email)
				} else {
					logger.Error("auth: error occured in the middleware", "error", err)
				}
				logger.Error("error")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserEmailContextKey, email)
			ctx = context.WithValue(ctx, UserObjectContextKey, dbUser)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
