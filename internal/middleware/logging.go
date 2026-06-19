// Copyright (c) 2025 Olbutov Aleksandr
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
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

// GetRequestID extracts the Request UUID from the context.
// It returns an empty string if the ID is not found.
func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(RequestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// LoggingMiddleware returns a middleware that logs every request with structured zap fields.
// Debug: request start; Info/Warn/Error: request end based on status code (2xx/4xx/5xx).
func LoggingMiddleware(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get("X-Request-ID")
			if reqID == "" {
				reqID = newRequestID()
			}

			r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, reqID))
			w.Header().Set("X-Request-ID", reqID)

			start := time.Now()
			rw := &responseWriter{ResponseWriter: w}

			log.Debug("request started",
				zap.String("request_id", reqID),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
			)

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			status := rw.status
			if status == 0 {
				status = http.StatusOK
			}

			fields := []zap.Field{
				zap.String("request_id", reqID),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status_code", status),
				zap.Duration("duration", duration),
				zap.String("remote_addr", r.RemoteAddr),
				zap.Int("response_bytes", rw.bytes),
			}

			switch {
			case status >= 500:
				log.Error("request completed", fields...)
			case status >= 400:
				log.Warn("request completed", fields...)
			default:
				log.Info("request completed", fields...)
			}
		})
	}
}

type ctxKey string

// RequestIDKey is the context key where the request UUID is stored.
const RequestIDKey ctxKey = "request_id"

// newRequestID generates a V4-style UUID without external dependencies.
func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	// Set version (4) and variant bits per RFC 4122
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
