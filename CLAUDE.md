# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the core backend service for IddaaLens, a comprehensive sports betting analysis platform focused on the Turkish Iddaa betting system. The service provides:

1. **REST API** (`cmd/api`) - HTTP service for data access and analysis
2. **Cron Jobs** (`cmd/cron`) - Automated data synchronization from Iddaa API and API-Football

## Architecture

The codebase follows clean architecture principles:

- `/cmd` - Service entry points (api, cron)
- `/pkg` - Core business logic organized by domain
  - `database/` - SQLC-generated type-safe database code
  - `handlers/` - HTTP request handlers with middleware
  - `services/` - Business logic and external API integrations
  - `jobs/` - Cron job implementations with distributed locking
  - `models/` - Domain models and data structures
- `/sql` - SQL query definitions for SQLC
- `/migrations` - Database migration files
- `/deploy` - Kubernetes deployment configurations

## Development Commands

```bash
# Database setup (required first)
export DATABASE_URL="postgresql://iddaa:iddaa123@localhost:5433/iddaa_core?sslmode=disable"
make migrate         # Run database migrations

# Build and run
make deps            # Install dependencies
make build           # Build all services
make run-api         # Run API service (port 8080)
make run-cron        # Run cron jobs service
make docker-dev      # Start full dev environment

# Database operations
make sqlc            # Regenerate database code after SQL changes
make migrate-down    # Rollback one migration
make migrate-status  # Check migration status

# Testing and quality
make test            # Run all tests
make test-coverage   # Generate coverage report
make lint            # Run linters (golangci-lint, go vet, go fmt)

# Single job testing
go run ./cmd/cron --job=events --once
go run ./cmd/cron --job=smart_money_processor --once
go run ./cmd/cron --job=api_football_team_matching --once

# Docker operations
make build-images ORG=iddaalens TAG=latest
make push-images ORG=iddaalens TAG=latest
```

## API Endpoints

### Core Endpoints

- `GET /health` - Health check (no database dependency)
- `GET /api/v1/events` - List events with filters (sport_id, league_id, date range)
- `GET /api/v1/events/upcoming` - Next 7 days of matches
- `GET /api/v1/events/daily` - Today's matches
- `GET /api/v1/events/live` - Currently live matches
- `GET /api/v1/events/:id` - Single event details
- `GET /api/v1/events/:id/odds` - Odds history for an event

### Odds & Analysis

- `GET /api/v1/odds` - List odds with filters
- `GET /api/v1/odds/big-movers` - Significant odds movements
- `GET /api/v1/smart-money/alerts` - Smart money detection alerts
- `GET /api/v1/smart-money/summary` - Overview of smart money activity

### Reference Data

- `GET /api/v1/teams` - List teams (optional league_id filter)
- `GET /api/v1/teams/:id` - Single team details
- `GET /api/v1/leagues` - List leagues (optional sport_id filter)
- `GET /api/v1/leagues/:id` - Single league details

## Database Operations

The project uses SQLC for type-safe database queries:

1. **Modifying queries**: Edit SQL files in `/sql/queries/`
2. **Parameter syntax**: Use `sqlc.arg(param_name)` instead of `$1, $2`
3. **Regenerate code**: Run `make sqlc` after changes
4. **Test changes**: Run `make test`

Key database features:

- Upsert operations for idempotent data sync
- JSONB fields for flexible configuration storage
- Optimized indexes for high-frequency queries
- Distributed job locking for production environments

## External API Integration

### Iddaa API

- Turkish sports betting platform
- Requires API key and anti-bot headers (critical for production)
- Field mappings: `i` (id), `cid` (country), `si` (sport)
- Rate limits must be respected

### API-Football

- Provides team/league enrichment data
- Uses 70%+ confidence threshold for matching
- Separate jobs for matching and enrichment
- Currently 38 leagues successfully mapped

### OpenAI API

- Used for Turkish to English translations
- Helps improve team/league matching accuracy
- Optional but recommended for better data quality

## Cron Jobs

All jobs implement the `jobs.Job` interface with distributed locking support:

- **config** - System configuration sync
- **sports** - Sport types synchronization
- **events** - Match events and basic odds (every 5 minutes)
- **leagues** - League and team data sync
- **detailed_odds** - High-frequency odds tracking
- **smart_money_processor** - Detect odds movements and betting patterns
- **api_football_team_matching** - Match teams with API-Football (daily)
- **api_football_league_matching** - Match leagues with API-Football (daily)
- **volume** - Betting volume collection
- **distribution** - Betting distribution analytics

Production mode: `--production-mode` flag enables distributed locking

## Important Production Considerations

1. **Anti-bot Headers**: Required for all Iddaa API calls to prevent blacklisting
2. **Context Timeouts**: 30-minute timeout per job execution, proper deadline handling required
3. **Rate Limiting**: Respect external API limits in job scheduling
4. **Database Pooling**: Optimized connection pooling for high-frequency operations
5. **Distributed Locking**: Prevents concurrent job execution in multi-instance deployments
6. **Error Handling**: Comprehensive error logging with request IDs for tracing

## Environment Variables

Required for production:

- `DATABASE_URL` - PostgreSQL connection string
- `IDDAA_API_KEY` - Turkish betting platform API key
- `API_FOOTBALL_KEY` - Football data API key
- `OPENAI_API_KEY` - Translation service API key (optional)
- `PORT` - API server port (default: 8080)
- `LOG_LEVEL` - Logging level (default: info)

## Deployment

The service deploys to Kubernetes using Azure Container Registry:

```bash
# Build and push images
cd core
make build-images ORG=iddaalens TAG=latest
make push-images ORG=iddaalens TAG=latest

# Deploy to Kubernetes
cd deploy/iddaa-backend
./deploy.sh
```

Key deployment files:

- `deployment.yaml` - Separate deployments for API and Cron
- `configmap.yaml` - Environment configuration
- `secrets.yaml` - Sensitive credentials
- `service.yaml` - Load balancer configuration

## Testing Strategy

- Unit tests for critical business logic
- Integration tests for API endpoints
- Mock external API responses for reliable testing
- Use `make test-race` before production deployments
- Minimum 70% code coverage target

## Feature Implementation Status

### Smart Money Tracker ✓

- Real-time odds movement detection
- Confidence scoring algorithm
- Alert generation for significant movements

### Value Hunter ✓

- Betting distribution analysis
- Public vs sharp money identification
- Value bet detection

### Bankroll Boss (In Progress)

- Kelly criterion calculations
- Bet tracking and analytics
- ROI performance metrics
