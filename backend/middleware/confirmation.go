package middleware

import (
	"net/http"

	"github.com/Neat-Snap/blueprint-backend/logger"
	"github.com/Neat-Snap/blueprint-backend/workos"
	"github.com/workos/workos-go/v4/pkg/usermanagement"
)

// ConfirmationConfig configures the email confirmation middleware.
type ConfirmationConfig struct {
	RequireVerifiedEmail bool
	WorkOSClient         *usermanagement.Client
	Logger               logger.MultiLogger
}

// Confirmation ensures that the authenticated WorkOS user has a verified email address.
func Confirmation(cfg ConfirmationConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			workosData, ok := r.Context().Value(WorkOSUserContextKey).(*workos.ValidationResult)
			if !ok || workosData == nil {
				http.Error(w, "user is not authenticated", http.StatusUnauthorized)
				return
			}

			if userObj.EmailVerified {
				next.ServeHTTP(w, r)
				return
			}

			if workosData.EmailVerified {
				next.ServeHTTP(w, r)
				return
			}

			if cfg.WorkOSClient != nil && workosData.UserID != "" {
				if _, err := cfg.WorkOSClient.SendVerificationEmail(r.Context(), usermanagement.SendVerificationEmailOpts{User: workosData.UserID}); err != nil {
					cfg.Logger.Warn("confirmation: failed to send verification email", "error", err)
				}
			}

			http.Error(w, "email not verified", http.StatusForbidden)
		})
	}
}
