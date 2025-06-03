# Iddaa Core Backend Services

A comprehensive backend system for the Turkish betting platform Iddaa, built with Go, PostgreSQL, and modern development practices.

## 🏗️ Architecture Overview

The system consists of two main services:

- **🔄 Cron Jobs Service** (`cmd/cron`) - Automated data synchronization from Iddaa APIs
- **🌐 REST API Service** (`cmd/api`) - HTTP endpoints for odds history and AI predictions *(coming soon)*

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 12+
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI tool
- [sqlc](https://sqlc.dev/) for type-safe SQL code generation

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd iddaa-core

# Install dependencies
make deps

# Generate database code
make sqlc

# Set up database
export DATABASE_URL="postgres://user:password@localhost:5432/iddaa_core?sslmode=disable"
make db-create    # Create database
make migrate      # Run migrations

# Build services
make build

# Run cron service
./bin/cron
```

## 📁 Project Structure

```
├── cmd/
│   ├── api/                 # REST API service (future)
│   └── cron/               # Cron jobs service
├── pkg/
│   ├── database/           # Generated sqlc code
│   ├── services/           # Business logic services
│   ├── jobs/              # Cron job implementations
│   └── models/            # Data models
├── internal/config/       # Configuration management
├── sql/
│   ├── schema/            # Database schema definitions
│   └── queries/           # SQL queries for sqlc
├── migrations/            # Database migrations
├── scripts/make/         # Modular Makefiles
└── docs/                 # Documentation
```

## 🔧 Development

See detailed development guides:

- [Development Setup](docs/development.md) - Local setup and workflows
- [Database Guide](docs/database.md) - Schema, migrations, and queries
- [API Documentation](docs/api.md) - External API integration details
- [Deployment Guide](docs/deployment.md) - Production deployment

## 📊 Features

### ✅ Implemented

- **Data Synchronization**: Automated fetching of competitions and configuration
- **Type-Safe Database**: sqlc-generated Go code with PostgreSQL
- **Idempotent Migrations**: Safe, repeatable database schema changes
- **Extensible Job System**: Interface-based cron job management
- **Comprehensive Testing**: Unit tests with mocks and edge cases
- **Configuration Management**: Environment-based config with validation

### 🔮 Planned

- REST API endpoints for odds history
- AI prediction system integration
- Real-time odds updates via WebSockets
- Event and team data synchronization
- Monitoring and alerting

## 🛠️ Make Commands

```bash
# Build
make build              # Build all services
make build-cron         # Build cron service only

# Development
make deps               # Install dependencies
make sqlc               # Generate database code
make run-cron           # Run cron service

# Testing & Quality
make test               # Run all tests
make test-coverage      # Generate coverage report
make lint               # Run linting tools

# Database
make migrate            # Run migrations
make migrate-down       # Rollback one migration
make db-create          # Create database
make db-drop            # Drop database

# Help
make help               # Show all commands
```

## 🗄️ Database Schema

The system uses PostgreSQL with the following main tables:

- `sports` - Sport types (football, basketball, etc.)
- `competitions` - Leagues and tournaments from Iddaa API
- `teams` - Team information
- `events` - Matches/games
- `odds` - Historical odds data
- `predictions` - AI model predictions
- `app_config` - Platform configuration (JSONB)

## 🔗 External APIs

- **Competitions**: `GET https://sportsbookv2.iddaa.com/sportsbook/competitions`
- **Configuration**: `GET https://contentv2.iddaa.com/appconfig?platform=WEB`
- **Events**: `GET /sportsbook/competitions/{id}/events` *(future)*
- **Odds**: `GET /sportsbook/events/{id}/odds` *(future)*

## 🔧 Configuration

Environment variables:

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=iddaa
DB_PASSWORD=secret
DB_NAME=iddaa_core
DB_SSLMODE=disable

# Server
PORT=8080
HOST=localhost

# External API
EXTERNAL_API_TIMEOUT=30
```

## 📈 Monitoring

The cron service provides structured logging for:

- Job execution status and duration
- API request success/failure rates
- Database operation metrics
- Error reporting with context

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Run `make lint` and `make test`
5. Submit a pull request

## 📄 License

This project is proprietary software for Betslib/Iddaa integration.

---

**Built with ❤️ for reliable sports betting data management**