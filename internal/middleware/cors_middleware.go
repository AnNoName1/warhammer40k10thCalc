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
	normalizedAllowed := normalizeOriginSet(allowedOrigins)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawOrigin := r.Header.Get("Origin")

			// Server-to-Server / Same-Origin requests carry no Origin header.
			if rawOrigin == "" {
				next.ServeHTTP(w, r)
				return
			}

			origin := strings.TrimSuffix(strings.ToLower(rawOrigin), "/")

			if normalizedAllowed[origin] {
				setCORSHeaders(w, rawOrigin)

				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}

			// The browser will block the response due to missing CORS headers.
			next.ServeHTTP(w, r)
		})
	}
}

func normalizeOriginSet(allowedOrigins map[string]bool) map[string]bool {
	normalized := make(map[string]bool)
	for k, v := range allowedOrigins {
		normalized[strings.TrimSuffix(strings.ToLower(k), "/")] = v
	}
	return normalized
}

func setCORSHeaders(w http.ResponseWriter, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Max-Age", "86400")
}
