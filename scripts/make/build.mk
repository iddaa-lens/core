# Build related targets

.PHONY: build build-api build-cron clean

build: build-api build-cron ## Build all services

build-api: ## Build the REST API service
	@mkdir -p bin
	go build -o bin/api ./cmd/api

build-cron: ## Build the cron jobs service
	@mkdir -p bin
	go build -o bin/cron ./cmd/cron

clean: ## Clean build artifacts
	rm -rf bin/