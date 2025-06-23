# Database related targets

.PHONY: migrate migrate-up migrate-down migrate-status db-create db-drop

migrate: migrate-up ## Run database migrations

migrate-up: ## Run database migrations up
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down: ## Run one migration down
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-status: ## Check migration status
	migrate -path migrations -database "$(DATABASE_URL)" version

db-create: ## Create database (requires DATABASE_URL with admin privileges)
	@echo "Creating generated..."
	@# Extract database name from URL for creation
	@DB_NAME=$$(echo "$(DATABASE_URL)" | sed -n 's/.*\/\([^?]*\).*/\1/p'); \
	ADMIN_URL=$$(echo "$(DATABASE_URL)" | sed 's/\/[^?]*/?/'); \
	psql "$${ADMIN_URL%?}postgres" -c "CREATE DATABASE $$DB_NAME;"

db-drop: ## Drop database (requires DATABASE_URL with admin privileges)  
	@echo "Dropping generated..."
	@DB_NAME=$$(echo "$(DATABASE_URL)" | sed -n 's/.*\/\([^?]*\).*/\1/p'); \
	ADMIN_URL=$$(echo "$(DATABASE_URL)" | sed 's/\/[^?]*/?/'); \
	psql "$${ADMIN_URL%?}postgres" -c "DROP DATABASE IF EXISTS $$DB_NAME;"