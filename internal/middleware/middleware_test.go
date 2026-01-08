package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareChain_Panic(t *testing.T) {
	handler := RecoverMiddleware(
		LoggingMiddleware(
			http.HandlerFunc(panicHandler),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	if got := rec.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("expected X-Request-ID header to be set")
	}
}
