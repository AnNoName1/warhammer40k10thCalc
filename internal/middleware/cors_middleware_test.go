package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware_Standard(t *testing.T) {
	allowed := map[string]bool{
		"https://example.com": true,
	}

	innerCalled := false
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	mw := CORSMiddleware(allowed)(innerHandler)

	tests := []struct {
		name           string
		method         string
		origin         string
		expectedStatus int
		expectInner    bool // Logic should execute?
		expectHeaders  bool // CORS headers should exist?
	}{
		// --- HAPPY PATHS ---
		{
			name:           "Allowed Origin - GET",
			method:         http.MethodGet,
			origin:         "https://example.com",
			expectedStatus: http.StatusOK,
			expectInner:    true,
			expectHeaders:  true,
		},
		{
			name:           "Allowed Origin - OPTIONS (Preflight)",
			method:         http.MethodOptions,
			origin:         "https://example.com",
			expectedStatus: http.StatusNoContent,
			expectInner:    false, // Middleware handles this
			expectHeaders:  true,
		},

		// --- PASSTHROUGH CASES (Firewall removed) ---
		{
			name:           "Unauthorized Origin - GET",
			method:         http.MethodGet,
			origin:         "https://malicious.com",
			expectedStatus: http.StatusOK, // Passed to inner handler
			expectInner:    true,          // Logic executed
			expectHeaders:  false,         // Browser blocks response
		},
		{
			name:           "Unauthorized Origin - OPTIONS",
			method:         http.MethodOptions,
			origin:         "https://malicious.com",
			expectedStatus: http.StatusOK, // Passed to inner handler
			expectInner:    true,          // Logic executed
			expectHeaders:  false,         // Browser blocks response
		},
		{
			name:           "Missing Origin Header",
			method:         http.MethodGet,
			origin:         "",
			expectedStatus: http.StatusOK,
			expectInner:    true,
			expectHeaders:  false,
		},
		{
			name:           "Normalization - Trailing Slash",
			method:         http.MethodGet,
			origin:         "https://example.com/",
			expectedStatus: http.StatusOK,
			expectInner:    true,
			expectHeaders:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			innerCalled = false
			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			mw.ServeHTTP(rec, req)

			// 1. Verify Status
			if rec.Code != tt.expectedStatus {
				t.Errorf("Status mismatch: got %d, want %d", rec.Code, tt.expectedStatus)
			}

			// 2. Verify Execution Flow
			if innerCalled != tt.expectInner {
				t.Errorf("Inner handler execution: got %v, want %v", innerCalled, tt.expectInner)
			}

			// 3. Verify Header Presence
			header := rec.Header().Get("Access-Control-Allow-Origin")
			if tt.expectHeaders && header == "" {
				t.Error("Expected CORS headers, got none")
			} else if !tt.expectHeaders && header != "" {
				t.Errorf("Expected NO CORS headers, got %s", header)
			}
		})
	}
}
