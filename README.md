# Iddaa API Service

A simple API service with health check endpoint for Iddaa-related features.

## 🚀 Quick Start

### Prerequisites

- Go 1.23+
- Docker (optional)

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd iddaa-core

# Install dependencies
make deps

# Build service
make build

# Run locally
make run
```

## 📁 Project Structure

```
├── cmd/api/               # API service
├── docker/               # Docker files
├── deploy/iddaa-backend/ # Kubernetes deployment
├── Makefile              # Build and development commands
└── CLAUDE.md            # Project documentation
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

## 📊 API Endpoints

- `GET /health` - Health check endpoint returning JSON status
- `GET /` - Simple root endpoint returning text response

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
- Ingress for api.iddaa.betslib.com

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

This project is proprietary software for Betslib/Iddaa integration.

---

**Simple, reliable API service for health monitoring**