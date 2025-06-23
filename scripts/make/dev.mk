# Development related targets

.PHONY: run-api run-cron deps sqlc docker-build docker-up docker-down docker-logs docker-dev test-cron-competitions test-cron-config test-cron-local inspect-db docker-clean

run-api: ## Run the REST API service
	go run ./cmd/api

run-cron: ## Run the cron jobs service
	go run ./cmd/cron

deps: ## Download dependencies
	go mod download
	go mod tidy

sqlc: ## Generate sqlc code
	echo "Generating SQL code with sqlc..."
	echo "Deleting old generated code..."
	rm -rf pkg/database/generated
	echo "Running sqlc generate..."
	sqlc generate

# Docker targets
docker-build: ## Build Docker images
	docker compose build

docker-up: ## Start all services with Docker Compose
	docker compose up -d

docker-down: ## Stop all services
	docker compose down

docker-logs: ## View logs from all services
	docker compose logs -f

docker-dev: ## Start development environment
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build

docker-clean: ## Clean up Docker resources
	docker compose down -v
	docker system prune -f

# Cron testing targets
test-cron-competitions: ## Test competitions sync job once
	docker compose up -d postgres
	@echo "Waiting for generated..."
	@until docker compose exec postgres pg_isready -U iddaa -d iddaa_core >/dev/null 2>&1; do sleep 1; done
	docker compose run --rm cron ./cron --job=competitions --once

test-cron-config: ## Test config sync job once  
	docker compose up -d postgres
	@echo "Waiting for generated..."
	@until docker compose exec postgres pg_isready -U iddaa -d iddaa_core >/dev/null 2>&1; do sleep 1; done
	docker compose run --rm cron ./cron --job=config --once

test-cron-local: ## Test cron jobs locally (without Docker)
	@echo "Testing competitions sync..."
	go run ./cmd/cron --job=competitions --once
	@echo "Testing config sync..."
	go run ./cmd/cron --job=config --once

inspect-db: ## Inspect database after cron runs
	./scripts/inspect-db.sh