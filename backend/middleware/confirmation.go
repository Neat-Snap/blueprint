package middleware

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/db"
	mail "github.com/Neat-Snap/blueprint-backend/utils/email"
)

func Confirmation(cfg config.Config, rds *mail.Redis) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userObj, ok := r.Context().Value(UserObjectContextKey).(*db.User)
			if !ok {
				http.Error(w, "user is not authenticated", http.StatusUnauthorized)
				return
			}

			if userObj.EmailVerified {
				next.ServeHTTP(w, r)
				return
			}

			if userObj.Email == nil || *userObj.Email == "" {
				http.Error(w, "user email missing", http.StatusForbidden)
				return
			}

			email := *userObj.Email
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()

			id, err := rds.GetIdByEmail(ctx, mail.VerifyPurpose, email)
			if err != nil {
				if err == mail.ErrNotFound {
					var code string
					id, code, err = rds.Create(ctx, []byte(cfg.REDIS_SECRET), mail.VerifyPurpose, email, 6, 15*time.Minute, 6)
					_ = code
				}
			}
			if err != nil || id == "" {
				http.Error(w, "could not initiate verification", http.StatusInternalServerError)
				return
			}

			v := url.Values{}
			v.Set("cid", id)
			v.Set("email", email)
			http.Redirect(w, r, cfg.APP_URL+"/auth/verify?"+v.Encode(), http.StatusFound)
		})
	}
}
