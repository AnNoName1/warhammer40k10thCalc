# Warhammer 40k 10th Calc API

**Version:** 1.0

A high-performance backend engine for simulating combat outcomes in Warhammer 40,000 10th Edition. This project follows the [Standard Go Project Layout](https://github.com/golang-standards/project-layout) and serves as the backend for the "Industrial Programming Technologies" course.

## Architecture & Features

* **API Specification:** OpenAPI documentation auto-generated via **Swagger**.
* **Observability:**
* **Logging Middleware:** Automatically assigns UUIDs to requests (see `internal/middleware`).
* **Profiling:** Integrated `pprof` support (see `internal/app/app.go`).


* **Testing:**
* Unit tests for calculation logic (`internal/calculator`).
* Integration tests for handlers (`internal/middleware`).



## Getting Started

### Prerequisites

* Go 1.2x+
* `pre-commit` (optional, for development hooks)

### Build & Run

```bash
# Install dependencies
go mod tidy

# Build the binary
go build cmd/WarhammerCalcServer/main.go

# Run the server
go run cmd/WarhammerCalcServer/main.go

```

### Verification

To verify the server is running and middleware is active:

```bash
curl -v http://localhost:8080/health
# Expect 'X-Request-Id' header in response

```

## Development

### Testing

```bash
# Run all tests with race detection and coverage
go test -v -race -cover ./...

```

### Documentation (Swagger)

```bash
# Format Swagger comments
swag fmt

# Generate OpenAPI docs
swag init -g cmd/WarhammerCalcServer/main.go

```

### Code Quality & Licensing

```bash
# Add license headers (MIT)
addlicense -c "Olbutov Aleksandr" -l mit -ignore **/docs/** .

# Install Git hooks
pre-commit install --hook-type pre-push
pre-commit install --hook-type commit-msg

```