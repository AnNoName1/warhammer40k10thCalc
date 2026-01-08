package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func panicHandler(w http.ResponseWriter, r *http.Request) {
	panic("boom")
}
func TestRecoverMiddleware_Panic(t *testing.T) {
	handler := RecoverMiddleware(http.HandlerFunc(panicHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.StatusCode)
	}
}
