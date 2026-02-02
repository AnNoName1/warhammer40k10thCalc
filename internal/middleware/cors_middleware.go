package middleware

import (
	"net/http"
	"strings"
)

func CORSMiddleware(allowedOrigins map[string]bool) func(http.Handler) http.Handler {
	// Normalize map for O(1) lookup
	normalizedAllowed := make(map[string]bool)
	for k, v := range allowedOrigins {
		normalized := strings.TrimSuffix(strings.ToLower(k), "/")
		normalizedAllowed[normalized] = v
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawOrigin := r.Header.Get("Origin")

			// 1. Missing Origin: Pass through (Server-to-Server / Same-Origin)
			if rawOrigin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// 2. Normalize and Check
			origin := strings.TrimSuffix(strings.ToLower(rawOrigin), "/")

			if normalizedAllowed[origin] {
				// Match found: Decorate response
				w.Header().Set("Access-Control-Allow-Origin", rawOrigin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "86400")

				// Handle Preflight for allowed origins only
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}

			// 3. Unauthorized / Non-Matching: Pass through
			// The browser will block the response due to missing CORS headers.
			next.ServeHTTP(w, r)
		})
	}
}
