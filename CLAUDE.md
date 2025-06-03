# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository hosts the backend services for iddaa-related features. Iddaa is the betting platform of Turkey. The project consists of two main services:

1. **REST API** (`cmd/api`) - Provides HTTP endpoints for odds history and AI predictions
2. **Cron Jobs** (`cmd/cron`) - Fetches data from iddaa API and stores in database

## Technology Stack

- **Language**: Go 1.21+
- **Database**: PostgreSQL with sqlc for type-safe queries
- **Web Framework**: Gin
- **External API**: Iddaa sportsbook API
- **Database Migrations**: golang-migrate
- **Cron Jobs**: robfig/cron

## Development Commands

The project uses modular Makefiles organized in `scripts/make/` directory:

```bash
# Build commands
make build           # Build all services  
make build-cron      # Build cron service only
make clean           # Clean build artifacts

# Development commands
make deps            # Download and organize dependencies
make sqlc            # Generate sqlc code (run after modifying SQL queries)
make run-cron        # Run cron jobs service

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

# Help
make help           # Show all available commands
```

## Database Schema

- `sports` - Sport types (football, basketball, etc.)
- `competitions` - Leagues and tournaments from iddaa API
- `teams` - Team information
- `events` - Matches/games
- `odds` - Historical odds data
- `predictions` - AI model predictions
- `market_types` - Betting market types (1X2, Over/Under, etc.)

## External API Integration

The system fetches data from iddaa API endpoints:
- `GET /sportsbook/competitions` - Competition list
- `GET /sportsbook/competitions/{id}/events` - Events for competition  
- `GET /sportsbook/events/{id}/odds` - Odds for event
- `GET https://contentv2.iddaa.com/appconfig?platform=WEB` - Platform configuration

## Important Notes

- All SQL queries use `sqlc.arg(param_name)` instead of `$1, $2` etc.
- Database migrations are idempotent using `CREATE TABLE IF NOT EXISTS` and `ON CONFLICT`
- Database schema is in `sql/schema/` and migrations in `migrations/`
- Iddaa API responses use specific field names: `i` (id), `cid` (country), `si` (sport), etc.
- Competitions sync runs every 6 hours, config sync every 4 hours via cron jobs
- Use `UpsertCompetition` and `UpsertConfig` to handle existing data updates
- Configuration data is stored as JSONB for flexible querying