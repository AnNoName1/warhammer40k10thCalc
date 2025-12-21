package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggingMiddleware_GeneratesAndPropagatesRequestID(t *testing.T) {
	// next handler echoes the request id from context to the response body
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.Context().Value(RequestIDKey); v != nil {
			if s, ok := v.(string); ok {
				w.Write([]byte(s))
				return
			}
		}
		http.Error(w, "no id", http.StatusInternalServerError)
	})

	h := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	// header must contain X-Request-ID
	id := res.Header.Get("X-Request-ID")
	if id == "" {
		t.Fatalf("expected X-Request-ID header, got empty")
	}

	// body should equal the id (echoed by next handler)
	b, _ := io.ReadAll(res.Body)
	if string(b) != id {
		t.Fatalf("body (%s) != header id (%s)", string(b), id)
	}
}

func TestLoggingMiddleware_PreservesClientRequestID(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.Context().Value(RequestIDKey); v != nil {
			if s, ok := v.(string); ok {
				w.Write([]byte(s))
				return
			}
		}
		http.Error(w, "no id", http.StatusInternalServerError)
	})

	h := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	clientID := "client-provided-id-123"
	req.Header.Set("X-Request-ID", clientID)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	id := res.Header.Get("X-Request-ID")
	if id != clientID {
		t.Fatalf("expected header id %s, got %s", clientID, id)
	}

	b, _ := io.ReadAll(res.Body)
	if string(b) != clientID {
		t.Fatalf("body (%s) != client id (%s)", string(b), clientID)
	}
}
