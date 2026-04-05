# Stage 1: Build
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Compile the binary (disable CGO for static linking)
RUN CGO_ENABLED=0 GOOS=linux go build -o /warhammer-server cmd/WarhammerCalcServer/main.go

# Stage 2: Final image
FROM alpine:latest
WORKDIR /

# Copy only the built binary
COPY --from=builder /warhammer-server /warhammer-server

# Specify the port your application listens on (assume 8080)
EXPOSE 8080 

ENTRYPOINT ["/warhammer-server"]