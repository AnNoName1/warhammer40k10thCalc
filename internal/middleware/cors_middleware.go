// Copyright (c) 2026 Olbutov Aleksandr
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
