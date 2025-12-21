package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"time"
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

// LoggingMiddleware logs request info and ensures a request ID in context and response header.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prefer client-provided X-Request-ID, otherwise generate one
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = newRequestID()
		}

		// attach to context and response header
		r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, reqID))
		w.Header().Set("X-Request-ID", reqID)

		start := time.Now()
		rw := &responseWriter{ResponseWriter: w}

		// log before
		remote := r.RemoteAddr
		log.Printf("[%s] START %s %s from %s", reqID, r.Method, r.URL.Path, remote)

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// log after
		log.Printf("[%s] END %s %s %d %dB %s %s", reqID, r.Method, r.URL.Path, rw.status, rw.bytes, duration, remote)
	})
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

// Prefer client-provided X-Request-ID, otherwise generate one
