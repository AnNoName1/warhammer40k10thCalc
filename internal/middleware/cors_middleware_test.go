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
