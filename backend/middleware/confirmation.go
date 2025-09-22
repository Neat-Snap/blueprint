package middleware

import (
	"net/http"
	"net/url"

	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/db"
)

func Confirmation(cfg config.Config) func(http.Handler) http.Handler {
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

			email := ""
			if userObj.Email != nil {
				email = *userObj.Email
			}
			v := url.Values{}
			if email != "" {
				v.Set("email", email)
			}
			http.Redirect(w, r, cfg.APP_URL+"/auth/verify?"+v.Encode(), http.StatusFound)
		})
	}
}
