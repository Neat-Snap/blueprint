package middleware

import (
	"net/http"
	"strings"
)

func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var allowed []string
			for _, o := range strings.Split(allowedOrigin, ",") {
				o = strings.TrimSpace(o)
				if o != "" {
					allowed = append(allowed, o)
				}
			}

			reqOrigin := r.Header.Get("Origin")
			if reqOrigin != "" {
				for _, ao := range allowed {
					if strings.EqualFold(reqOrigin, ao) {
						w.Header().Set("Access-Control-Allow-Origin", reqOrigin)
						break
					}
				}
			}

			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Expose-Headers", "Link")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
