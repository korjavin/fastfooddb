package auth

import (
	"net/http"
	"strings"
)

// ParseAPIKeys parses a comma-separated list of API keys, trimming whitespace
// and ignoring empty entries.
func ParseAPIKeys(raw string) []string {
	var keys []string
	for _, k := range strings.Split(raw, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

// APIKeyMiddleware returns a middleware that validates the X-API-Key header.
// Also accepts api_key as a query parameter as a fallback.
// If no keys are configured, all requests are allowed through.
func APIKeyMiddleware(validKeys []string) func(http.Handler) http.Handler {
	keySet := make(map[string]struct{}, len(validKeys))
	for _, k := range validKeys {
		keySet[k] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(keySet) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get("X-API-Key")
			if key == "" {
				key = r.URL.Query().Get("api_key")
			}

			if _, ok := keySet[key]; !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
