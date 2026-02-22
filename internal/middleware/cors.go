package middleware

import (
	"net/http"
	"strings"
)

// CORS returns a middleware that sets CORS headers.
// origins can be "*" to allow all, or a comma-separated list of allowed origins.
func CORS(origins string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origins == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				for _, allowed := range strings.Split(origins, ",") {
					if strings.TrimSpace(allowed) == origin {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
