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
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

func NewServer(handler http.Handler, port string) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

func BuildMux(calc *calculator.DamageCalculatorImpl) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/alive", HealthCheck)
	mux.HandleFunc("/ready", ReadinessCheck)
	mux.HandleFunc("/api/damage/calculate", handler.CalculateDamageHandler(calc))
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	return mux
}

func BuildHandler(mux *http.ServeMux, origins map[string]bool) http.Handler {
	return middleware.CORSMiddleware(origins)(
		middleware.RecoverMiddleware(
			middleware.LoggingMiddleware(mux),
		),
	)
}

type Config struct {
	Port    string
	Origins map[string]bool
}

func LoadConfig(getenv func(string) string) Config {
	port := getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{
		Port:    port,
		Origins: parseOrigins(getenv("CORS_ALLOWED_ORIGINS")),
	}
}

func StartServer(srv *http.Server, errCh chan<- error) {
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
}

func ShutdownServer(srv *http.Server, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return srv.Shutdown(ctx)
}

// Run initializes the application and starts the HTTP server.
func Run() error {
	cfg := LoadConfig(os.Getenv)

	calcCore := &calculator.DamageCalculatorImpl{}

	mux := BuildMux(calcCore)
	handler := BuildHandler(mux, cfg.Origins)

	log.Printf("Server starting on http://localhost:%s\n", cfg.Port)
	log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html\n", cfg.Port)

	srv := NewServer(handler, cfg.Port)

	errCh := make(chan error, 1)
	StartServer(srv, errCh)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
	case err := <-errCh:
		return err
	}

	return ShutdownServer(srv, 30*time.Second)
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
