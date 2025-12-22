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

package app

import (
	"log"
	"net/http"
	"net/http/pprof"
	"os"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/AnNoName1/warhammer40k10thCalc/docs"

	"github.com/AnNoName1/warhammer40k10thCalc/internal/middleware"
	handler "github.com/AnNoName1/warhammer40k10thCalc/pkg/handler"
)

// Run initializes the application and starts the HTTP server.
// It returns an error if the server fails to start or encounters an issue.
func Run() error {
	mux := http.NewServeMux()

	// Register pprof handlers manually to your custom mux
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/api/damage/calculate", handler.CalculateDamageHandler)

	// This serves the documentation at /swagger/index.html
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	// Wrap the mux with logging middleware
	handlerWithMiddleware := middleware.LoggingMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on http://localhost:%s\n", port)
	log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html\n", port)
	// Start the server with middleware-wrapped handler
	return http.ListenAndServe(":"+port, handlerWithMiddleware)
}
