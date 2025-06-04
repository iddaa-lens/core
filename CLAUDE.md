# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository hosts the backend services for iddaa-related features. Iddaa is the betting platform of Turkey. The project consists of two main services:

1. **REST API** (`cmd/api`) - Simple HTTP service with health check endpoint
2. **Cron Jobs** (`cmd/cron`) - Fetches data from iddaa API, stores in database, and syncs with Football API

## Technology Stack

- **Language**: Go 1.23+
- **Database**: PostgreSQL with sqlc for type-safe queries
- **Web Framework**: Standard library net/http (API), Gin framework available
- **External APIs**: Iddaa sportsbook API, Football API for team/league mapping
- **Database Migrations**: golang-migrate
- **Cron Jobs**: robfig/cron

## Development Commands

The project uses modular Makefiles organized in `scripts/make/` directory:

```bash
# Build commands
make build           # Build all services  
make build-cron      # Build cron service only
make build-api       # Build API service only
make clean           # Clean build artifacts

# Development commands
make deps            # Download and organize dependencies
make sqlc            # Generate sqlc code (run after modifying SQL queries)
make run-cron        # Run cron jobs service
make run-api         # Run API service locally

# Testing & Quality
make test            # Run all tests
make test-coverage   # Run tests with coverage report
make test-race       # Run tests with race detection
make lint            # Run linting tools (golangci-lint, go vet, go fmt)

# Database commands  
export DATABASE_URL="postgresql://iddaa:iddaa123@localhost:5433/iddaa_core?sslmode=disable"
make migrate         # Run database migrations up
make migrate-down    # Run one migration down
make migrate-status  # Check migration status
make db-create       # Create database
make db-drop         # Drop database

# Docker commands
make build-images ORG=iddaa-backend TAG=latest    # Build all Docker images
make push-images ORG=iddaa-backend TAG=latest     # Push all Docker images
make build-api-image                              # Build API image only
make build-cron-image                             # Build cron image only

# Help
make help           # Show all available commands
```

## Database Schema

- `sports` - Sport types (football, basketball, etc.)
- `leagues` - Leagues and tournaments (renamed from competitions)
- `teams` - Team information
- `events` - Matches/games
- `odds` - Historical odds data
- `predictions` - AI model predictions
- `market_types` - Betting market types (1X2, Over/Under, etc.)
- `league_mappings` - Maps internal leagues to Football API leagues
- `team_mappings` - Maps internal teams to Football API teams

## External API Integration

The system fetches data from multiple APIs:
- **Iddaa API**: `GET /sportsbook/competitions`, events, odds
- **Football API**: League and team data for mapping and enrichment
- **Configuration**: `GET https://contentv2.iddaa.com/appconfig?platform=WEB`

## API Endpoints

- `GET /health` - Health check endpoint returning JSON status
- `GET /` - Simple root endpoint returning text response

## Deployment

Kubernetes deployment files are available in `deploy/iddaa-backend/`:
- `namespace.yaml` - Kubernetes namespace
- `deployment.yaml` - Both API and cron service deployments
- `service.yaml` - Kubernetes service for API
- `ingress.yaml` - Ingress configuration
- `secrets.yaml` - Database and API key secrets

Deploy with:
```bash
cd deploy/iddaa-backend
./deploy.sh
```

## Important Notes

- All SQL queries use `sqlc.arg(param_name)` instead of `$1, $2` etc.
- Database migrations are idempotent using `CREATE TABLE IF NOT EXISTS` and `ON CONFLICT`
- Database schema is in `sql/schema/` and migrations in `migrations/`
- Iddaa API responses use specific field names: `i` (id), `cid` (country), `si` (sport), etc.
- Football API integration uses similarity matching with 70%+ confidence threshold
- Competitions sync runs every 6 hours, config sync every 4 hours, Football API sync daily at 2 AM
- Use `UpsertLeague` and `UpsertConfig` to handle existing data updates
- Configuration data is stored as JSONB for flexible querying
- API service has no database dependency - cron service handles all data operations