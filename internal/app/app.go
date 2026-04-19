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
	"os"
	"strings"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/AnNoName1/warhammer40k10thCalc/docs"

	calculator "github.com/AnNoName1/warhammer40k10thCalc/internal/calculator"
	"github.com/AnNoName1/warhammer40k10thCalc/internal/middleware"
	handler "github.com/AnNoName1/warhammer40k10thCalc/pkg/handler"
)

// HealthCheck godoc
//
//	@Summary		Health Check
//	@Description	Confirm the server is up and responding.
//	@Tags			System
//	@Produce		plain
//	@Success		200	{string}	string	"OK"
//	@Router			/alive [get]
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// ReadinessCheck godoc
//
//	@Summary		Readiness Check
//	@Description	Confirm the server is ready to receive traffic (currently same as alive).
//	@Tags			System
//	@Produce		plain
//	@Success		200	{string}	string	"OK"
//	@Router			/ready [get]
func ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	// TODO: When DB integration is added, check database connectivity here.
	// For now, it's identical to HealthCheck.
	w.WriteHeader(http.StatusOK)
}

// Run initializes the application and starts the HTTP server.
func Run() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/alive", HealthCheck)
	mux.HandleFunc("/ready", ReadinessCheck) // New line

	calcCore := &calculator.DamageCalculatorImpl{}

	mux.HandleFunc("/api/damage/calculate", handler.CalculateDamageHandler(calcCore))

	// This serves the documentation at /swagger/index.html
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	origins := parseOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))

	// Wrap the mux with middleware
	handler := middleware.CORSMiddleware(origins)(
		middleware.RecoverMiddleware(
			middleware.LoggingMiddleware(mux),
		),
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on http://localhost:%s\n", port)
	log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html\n", port)
	// Start the server with middleware-wrapped handler
	return http.ListenAndServe(":"+port, handler)
}

func parseOrigins(env string) map[string]bool {
	m := make(map[string]bool)
	for _, o := range strings.Split(env, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			m[o] = true
		}
	}
	return m
}
