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
