package middleware

import (
	"net/http"
	"net/url"

	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/db"
	mail "github.com/Neat-Snap/blueprint-backend/utils/email"
)

func Confirmation(cfg config.Config, _ *mail.Redis) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userObj, ok := r.Context().Value(UserObjectContextKey).(*db.User)
			if !ok {
				http.Error(w, "user is not authenticated", http.StatusUnauthorized)
				return
			}

			if userObj.EmailVerifiedAt != nil {
				next.ServeHTTP(w, r)
				return
			}

			if userObj.Email == nil || *userObj.Email == "" {
				http.Error(w, "user email missing", http.StatusForbidden)
				return
			}
			if userObj.WorkOSUserID == nil || *userObj.WorkOSUserID == "" {
				http.Error(w, "user identity missing", http.StatusForbidden)
				return
			}

			email := *userObj.Email

			v := url.Values{}
			v.Set("cid", *userObj.WorkOSUserID)
			v.Set("email", email)
			http.Redirect(w, r, cfg.APP_URL+"/auth/verify?"+v.Encode(), http.StatusFound)
		})
	}
}
