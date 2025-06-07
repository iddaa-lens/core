# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache ca-certificates git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build cron service with enhanced optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -buildid=" -trimpath -o bin/cron ./cmd/cron

# Build api service with enhanced optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -buildid=" -trimpath -o bin/api ./cmd/api

# Production stage - use distroless static for smallest possible image
FROM gcr.io/distroless/static-debian12:latest

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/bin/cron /cron
COPY --from=builder /app/bin/api /api

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Set timezone to UTC (distroless default)
ENV TZ=UTC

# Run as non-root user (distroless uses uid 65534 by default)
USER 65534

EXPOSE 8080

# Default command (can be overridden in docker-compose)
ENTRYPOINT ["/cron"]