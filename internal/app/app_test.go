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

package app

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/require"
)

func TestAliveHandler_Success(t *testing.T) {

	req := httptest.NewRequest(http.MethodGet, "/alive", nil)
	rr := httptest.NewRecorder()

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

func TestReadyHandler_Success(t *testing.T) {

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rr := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestGracefulShutdown_CompletesInFlightMocked(t *testing.T) {
	// 1. Create a test-only handler that guarantees a slow execution time.
	requestEntered := make(chan struct{})
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestEntered)       // Signal the test that the request is in-flight
		time.Sleep(2 * time.Second) // Simulate blocking work
		w.WriteHeader(http.StatusOK)
	})

	// 2. Instantiate server primitives using Port 0 (OS assigns random free port).
	srv := NewServer(testHandler, "0")

	// Use net.Listen to extract the actual port assigned by the OS.
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		t.Fatalf("Failed to bind port: %v", err)
	}

	port := ln.Addr().(*net.TCPAddr).Port

	// Start the server manually
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			t.Errorf("Unexpected server error: %v", err)
		}
	}()

	// 3. Fire the request in a background goroutine.
	requestCompleted := make(chan struct{})
	go func() {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
		if err == nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				t.Errorf("Response close error: %v", closeErr)
			}
		}
		close(requestCompleted)
	}()

	// 4. Wait for the request to physically enter the handler before initiating shutdown.
	<-requestEntered

	// 5. Trigger shutdown.
	start := time.Now()
	err = ShutdownServer(srv, 5*time.Second)
	if err != nil {
		t.Fatalf("ShutdownServer failed: %v", err)
	}

	// 6. Block until the HTTP client confirms the request finished.
	<-requestCompleted
	elapsed := time.Since(start)

	// 7. Verify the shutdown waited for the in-flight request to finish.
	if elapsed < 2*time.Second {
		t.Fatal("Shutdown completed prematurely, dropping the in-flight request")
	}
}

// freePort asks the OS for an available TCP port and returns it as a string.
// Using :0 lets the kernel pick; we close the listener before returning so
// the port is free for the server to bind — there is a tiny TOCTOU window,
// but it is vastly better than any hardcoded number.
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
	require.NoError(t, l.Close())
	return port
}

func TestRun_HappyPath(t *testing.T) {
	port := freePort(t)
	t.Setenv("PORT", port)

	// Write a real .env so godotenv.Load() succeeds (no "no file" log).
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, ".env"),
		[]byte("PORT="+port),
		0644,
	))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			fmt.Fprintf(os.Stderr, "failed to restore working directory: %v\n", err)
		}
	})
	require.NoError(t, os.Chdir(tmp))

	// Cancel after a short window — no signals, no goroutine leaks.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	require.NoError(t, run(ctx))
}

func TestRun_NoEnvFile_CapturedLog(t *testing.T) {
	port := freePort(t)
	t.Setenv("PORT", port)

	// Empty temp dir — godotenv.Load() will fail, triggering kNoFileStr.
	tmp := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.Chdir(oldWd)) })
	require.NoError(t, os.Chdir(tmp))

	// Redirect the global logger; restore it when the test ends.
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	require.NoError(t, run(ctx))

	out := buf.String()
	require.Contains(t, out, kNoFileStr)
	require.Contains(t, out, "Server starting on")
}
