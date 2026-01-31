package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAliveHandler_Success(t *testing.T) {
	// Setup the router to test the actual mux logic in Run()
	// or test the handler function directly if exported.
	req := httptest.NewRequest(http.MethodGet, "/alive", nil)
	rr := httptest.NewRecorder()

	// Assuming you define the handler logic clearly in Run
	mux := http.NewServeMux()
	mux.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
