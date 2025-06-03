# Main Makefile - includes all sub-makefiles
include scripts/make/build.mk
include scripts/make/dev.mk
include scripts/make/test.mk
include scripts/make/db.mk

.DEFAULT_GOAL := help

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@echo ''
	@echo 'Build:'
	@awk '/^[a-zA-Z_-]+:.*?## .*Build/ { printf "  %-18s %s\n", $$1, $$3 }' scripts/make/build.mk
	@echo ''
	@echo 'Development:'
	@awk '/^[a-zA-Z_-]+:.*?## / { printf "  %-18s %s\n", $$1, substr($$0, index($$0, "## ") + 3) }' scripts/make/dev.mk
	@echo ''
	@echo 'Testing & Quality:'
	@awk '/^[a-zA-Z_-]+:.*?## / { printf "  %-18s %s\n", $$1, substr($$0, index($$0, "## ") + 3) }' scripts/make/test.mk
	@echo ''
	@echo 'Database:'
	@awk '/^[a-zA-Z_-]+:.*?## / { printf "  %-18s %s\n", $$1, substr($$0, index($$0, "## ") + 3) }' scripts/make/db.mk
	@echo ''
	@echo 'Examples:'
	@echo '  make build              # Build all services'
	@echo '  make test               # Run all tests'
	@echo '  make run-cron           # Run cron service'
	@echo '  DATABASE_URL=postgres://... make migrate  # Run migrations'