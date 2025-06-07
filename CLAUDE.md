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
make docker-up       # Start all services with Docker Compose
make docker-down     # Stop all services
make docker-dev      # Start development environment

# Single Job Testing
go run ./cmd/cron --job=config --once           # Run config sync once
go run ./cmd/cron --job=sports --once           # Run sports sync once  
go run ./cmd/cron --job=events --once           # Run events sync once
go run ./cmd/cron --job=leagues --once          # Run leagues sync once
go run ./cmd/cron --job=detailed_odds --once    # Run detailed odds sync once

# Help
make help           # Show all available commands
```

## Database Schema

- `sports` - Sport types (football, basketball, etc.)
- `leagues` - Leagues and tournaments with API-Football enrichment fields (logo_url, country_code, etc.)
- `teams` - Team information with API-Football enrichment (team_code, founded_year, venue details, etc.)
- `events` - Matches/games
- `odds` - Historical odds data
- `predictions` - AI model predictions
- `market_types` - Betting market types (1X2, Over/Under, etc.)
- `league_mappings` - Maps internal leagues to Football API leagues with translation tracking
- `team_mappings` - Maps internal teams to Football API teams with confidence scoring

## External API Integration

The system fetches data from multiple APIs:
- **Iddaa API**: `GET /sportsbook/competitions`, events, odds
- **API-Football**: Comprehensive league and team data for mapping and enrichment
  - `/leagues` - Get leagues by various criteria (country, season, type, etc.)
  - `/teams` - Get teams by ID, name, league, country, venue, etc.
  - Rate-limited client with proper error handling and retry logic
- **Configuration**: `GET https://contentv2.iddaa.com/appconfig?platform=WEB`
- **OpenAI API**: Turkish to English translation for team/league names

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

## Cron Job Architecture

The cron service uses a robust job manager with the following features:
- **Centralized Scheduling**: All jobs registered in `cmd/cron/main.go`
- **Startup Execution**: All jobs run once on service startup for immediate data sync
- **Contextual Logging**: Each job execution gets unique request ID and structured logging
- **Timeout Management**: 30-minute timeout per job execution
- **Graceful Shutdown**: Jobs stop cleanly on SIGINT/SIGTERM

### Available Cron Jobs:
- `config` - Sync market configurations from Iddaa API
- `sports` - Fetch sport types
- `events` - Sync matches and basic odds (every 5 minutes)
- `leagues` - Sync leagues and teams with Football API mapping
- `detailed_odds` - High-frequency detailed odds tracking
- `volume` - Betting volume data collection
- `distribution` - Betting distribution analytics
- `statistics` - Match statistics collection
- `analytics` - Refresh analytics views
- `market_config` - Market type configurations
- `api_football_league_matching` - Match Turkish leagues with API-Football data (daily at 3 AM)
- `api_football_team_matching` - Match Turkish teams with API-Football data (daily at 4 AM)
- `api_football_league_enrichment` - Enrich leagues with API-Football metadata (weekly)
- `api_football_team_enrichment` - Enrich teams with API-Football metadata (weekly)

## Important Notes

- All SQL queries use `sqlc.arg(param_name)` instead of `$1, $2` etc.
- Database migrations are idempotent using `CREATE TABLE IF NOT EXISTS` and `ON CONFLICT`
- Database schema is in `sql/schema/` and migrations in `migrations/`
- Iddaa API responses use specific field names: `i` (id), `cid` (country), `si` (sport), etc.
- Football API integration uses similarity matching with 70%+ confidence threshold
- Use `UpsertLeague` and `UpsertConfig` to handle existing data updates
- Configuration data is stored as JSONB for flexible querying
- API service has no database dependency - cron service handles all data operations
- All jobs implement the `jobs.Job` interface with `Name()`, `Schedule()`, and `Execute(ctx)` methods