# Build stage
FROM golang:1.23-alpine AS builder

# Install git and other dependencies
RUN apk --no-cache add git ca-certificates tzdata

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build cron service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/cron ./cmd/cron

# Build api service (create placeholder if it doesn't exist)
RUN if [ -d "./cmd/api" ] && [ -n "$(find ./cmd/api -name '*.go' -print -quit)" ]; then \
        CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api ./cmd/api; \
    else \
        echo '#!/bin/sh' > bin/api && \
        echo 'echo "API service not implemented yet"' >> bin/api && \
        echo 'sleep infinity' >> bin/api && \
        chmod +x bin/api; \
    fi

# Production stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/bin/cron .
COPY --from=builder /app/bin/api .

# Copy migrations
COPY migrations ./migrations

# Install migrate tool
RUN wget -O migrate.tar.gz https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz && \
    tar -xzf migrate.tar.gz && \
    mv migrate /usr/local/bin/ && \
    rm migrate.tar.gz

# Create logs directory
RUN mkdir -p logs

# Create non-root user
RUN adduser -D -s /bin/sh appuser
RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 8080

# Default command (can be overridden in docker-compose)
CMD ["./cron"]