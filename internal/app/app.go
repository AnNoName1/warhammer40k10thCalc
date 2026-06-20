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

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/AnNoName1/warhammer40k10thCalc/docs"
	"github.com/AnNoName1/warhammer40k10thCalc/pkg/handler"

	"github.com/joho/godotenv"

	calculator "github.com/AnNoName1/warhammer40k10thCalc/internal/calculator"
	"github.com/AnNoName1/warhammer40k10thCalc/internal/middleware"
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

const kNoFileStr string = "No env file"

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

type Middleware func(http.Handler) http.Handler

func Apply(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// BuildPublicHandler собирает и оборачивает публичную ветку
func BuildPublicHandler(middlewares ...Middleware) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/alive", HealthCheck)
	mux.HandleFunc("/ready", ReadinessCheck)
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	return Apply(mux, middlewares...)
}

func BuildProtectedHandler(calc *calculator.DamageCalculatorImpl, log *zap.Logger, middlewares ...Middleware) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/damage/calculate", handler.CalculateDamageHandler(calc, log))

	return Apply(mux, middlewares...)
}

func BuildRootHandler(public, protected http.Handler, globalMW ...Middleware) http.Handler {
	root := http.NewServeMux()

	root.Handle("/alive", public)
	root.Handle("/ready", public)
	root.Handle("/swagger/", public)

	root.Handle("/api/", protected)

	return Apply(root, globalMW...)
}

type Config struct {
	Port     string
	Origins  map[string]bool
	LogLevel string
}

func LoadConfig(getenv func(string) string) Config {
	port := getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logLevel := getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	return Config{
		Port:     port,
		Origins:  parseOrigins(getenv("CORS_ALLOWED_ORIGINS")),
		LogLevel: logLevel,
	}
}

func NewLogger(levelStr string) (*zap.Logger, error) {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		level = zapcore.InfoLevel // safe fallback
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)

	return cfg.Build()
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
// Run is the production entry point. It wires OS signals to a context
// and delegates to the testable inner function.
func Run() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()
	return run(ctx)
}

// run is the testable core: it starts the server and exits when ctx is
// cancelled or the server itself errors out.
func run(ctx context.Context) error {
	if err := godotenv.Load(); err != nil {
		log.Print(kNoFileStr)
	}
	cfg := LoadConfig(os.Getenv)

	logger, err := NewLogger(cfg.LogLevel)
	if err != nil {
		return err
	}
	defer logger.Sync() //nolint:errcheck

	calcCore := &calculator.DamageCalculatorImpl{}

	// 1. Initialize Public-facing middleware (Auth-free zones like Healthchecks or Docs)
	publicMW := []Middleware{
		// middleware.RateLimitMiddleware,
	}

	// 2. Initialize Protected middleware (Business logic, logging, and panic recovery)
	protectedMW := []Middleware{
		middleware.RecoverMiddleware(logger),
		middleware.LoggingMiddleware(logger),
	}

	// 3. Initialize Global middleware (Applied to every single incoming request)
	globalMW := []Middleware{
		middleware.CORSMiddleware(cfg.Origins),
	}

	// 4. Build isolated route branches
	publicHandler := BuildPublicHandler(publicMW...)
	protectedHandler := BuildProtectedHandler(calcCore, logger, protectedMW...)

	// 5. Assemble the root router with global middleware wrapper
	handler := BuildRootHandler(publicHandler, protectedHandler, globalMW...)

	logger.Info("server starting",
		zap.String("addr", "http://localhost:"+cfg.Port),
		zap.String("swagger", "http://localhost:"+cfg.Port+"/swagger/index.html"),
	)

	srv := NewServer(handler, cfg.Port)

	errCh := make(chan error, 1)
	StartServer(srv, errCh)

	select {
	case <-ctx.Done(): // clean cancellation from tests or OS signal
	case err := <-errCh: // server failed to bind / crashed
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
