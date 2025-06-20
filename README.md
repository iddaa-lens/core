# Iddaa Core Backend

Core backend services for Iddaa data platform - REST API, cron jobs, and database management for comprehensive betting data analysis.

## 🚀 Quick Start

### Prerequisites

- Go 1.23+
- Docker (optional)

### Installation

```bash
# Clone the repository
git clone https://github.com/iddaa-lens/core.git
cd core

# Install dependencies
make deps

# Build service
make build

# Run locally
make run
```

## 📁 Project Structure

```text
├── cmd/
│   ├── api/              # REST API service
│   └── cron/             # Background job scheduler
├── pkg/
│   ├── database/         # Database queries and models
│   ├── jobs/             # Cron job implementations
│   ├── services/         # Business logic services
│   └── models/           # Data models
├── migrations/           # Database migrations
├── docs/                # Documentation
├── deploy/              # Kubernetes deployment configs
└── CLAUDE.md           # AI assistant guidance
```

## 🔧 Development

### Local Development

```bash
# Run the service
make run

# Build the service
make build

# Run tests
make test

# Run linting
make lint
```

### Docker

```bash
# Build Docker image
make build-image

# Build tagged image
make build-image-tagged ORG=iddaa-backend TAG=v1.0.0

# Push image
make push-image ORG=iddaa-backend TAG=v1.0.0
```

## 📊 Services

### API Service (`cmd/api`)

- `GET /health` - Health check endpoint returning JSON status
- `GET /` - Simple root endpoint returning text response

### Cron Service (`cmd/cron`)

- **Sports Sync**: Fetches sport types from Iddaa API
- **Leagues Sync**: Syncs leagues and teams (hourly)
- **Events Sync**: Fetches matches and odds (every 5 minutes)
- **Config Sync**: Updates market configurations
- **Statistics Sync**: Collects match statistics

### Health Endpoint Response

```json
{
  "status": "ok",
  "timestamp": "2024-06-03T10:30:45Z"
}
```

## 🚀 Deployment

Deploy to Kubernetes:

```bash
cd deploy/iddaa-backend
./deploy.sh
```

Deployment includes:

- Kubernetes namespace
- API service deployment (2 replicas)
- ClusterIP service
- Ingress for external access

## 🛠️ Make Commands

```bash
# Build
make build              # Build API service
make clean              # Clean build artifacts

# Development
make deps               # Install dependencies
make run                # Run API service locally

# Testing & Quality
make test               # Run all tests
make test-race          # Run tests with race detection
make lint               # Run linting tools

# Docker
make build-image        # Build Docker image
make build-image-tagged # Build tagged image (requires ORG and TAG)
make push-image         # Push image (requires ORG and TAG)

# Help
make help               # Show all commands
```

## 🔧 Configuration

Environment variables:

```bash
# Server
PORT=8080               # Server port (default: 8080)
```

## 📄 License

This project is part of the Iddaa Lens platform for sports betting data analysis.

---

**Comprehensive backend infrastructure for betting data intelligence**
