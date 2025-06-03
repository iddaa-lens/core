# Docker Guide

This guide covers how to run the Iddaa Core services using Docker and Docker Compose.

## 🐳 Quick Start

### Prerequisites

- Docker 20.0+
- Docker Compose 2.0+

### Run with Docker Compose

1. **Clone and setup environment**:
   ```bash
   git clone <repository-url>
   cd iddaa-core
   cp .env.example .env
   ```

2. **Start all services**:
   ```bash
   make docker-up
   ```

3. **View logs**:
   ```bash
   make docker-logs
   ```

4. **Stop services**:
   ```bash
   make docker-down
   ```

## 📁 Services

### PostgreSQL Database
- **Container**: `iddaa-postgres`
- **Port**: `5432`
- **Database**: `iddaa_core`
- **User/Password**: `iddaa/iddaa123`

### Cron Service
- **Container**: `iddaa-cron`
- **Purpose**: Background job processing
- **Jobs**: Competition sync, config sync

### API Service
- **Container**: `iddaa-api`
- **Port**: `8080`
- **Purpose**: REST API endpoints

## 🔧 Configuration

### Environment Variables

Create a `.env` file (copy from `.env.example`):

```bash
# Database
DATABASE_URL=postgres://iddaa:iddaa123@postgres:5432/iddaa_core?sslmode=disable

# API
PORT=8080
HOST=0.0.0.0

# External APIs
EXTERNAL_API_TIMEOUT=30
```

### Docker Compose Override

For development, use the dev override:

```bash
make docker-dev
```

This uses `docker compose.dev.yml` which:
- Mounts source code as volumes
- Uses `go run` for hot reloading
- Exposes additional ports

## 🚀 Development Workflow

### 1. Start Development Environment

```bash
# Start with development overrides
make docker-dev

# Or manually
docker compose -f docker compose.yml -f docker compose.dev.yml up --build
```

### 2. View Service Logs

```bash
# All services
make docker-logs

# Specific service
docker compose logs -f cron
docker compose logs -f api
docker compose logs -f postgres
```

### 3. Execute Commands in Containers

```bash
# Connect to postgres
docker compose exec postgres psql -U iddaa -d iddaa_core

# Run migrations manually
docker compose exec cron migrate -path migrations -database $DATABASE_URL up

# Check container health
docker compose ps
```

### 4. Rebuild After Changes

```bash
# Rebuild and restart
make docker-build
make docker-up

# Or rebuild specific service
docker compose build cron
docker compose up -d cron
```

## 🗄️ Database Management

### Accessing PostgreSQL

```bash
# Via docker compose
docker compose exec postgres psql -U iddaa -d iddaa_core

# Via host (if port 5432 is exposed)
psql -h localhost -U iddaa -d iddaa_core
```

### Running Migrations

Migrations run automatically on container startup, but you can run them manually:

```bash
docker compose exec cron migrate -path migrations -database $DATABASE_URL up
```

### Database Backup

```bash
# Create backup
docker compose exec postgres pg_dump -U iddaa iddaa_core > backup.sql

# Restore backup
docker compose exec -T postgres psql -U iddaa -d iddaa_core < backup.sql
```

## 📊 Monitoring

### Health Checks

The API service includes health checks:

```bash
# Check API health
curl http://localhost:8080/health

# View container health status
docker compose ps
```

### Resource Usage

```bash
# View resource usage
docker stats

# View container processes
docker compose top
```

## 🐛 Troubleshooting

### Common Issues

1. **Port conflicts**:
   ```bash
   # Check what's using port 5432
   lsof -i :5432
   
   # Stop conflicting services
   brew services stop postgresql
   ```

2. **Database connection errors**:
   ```bash
   # Check if postgres is healthy
   docker compose exec postgres pg_isready -U iddaa
   
   # View postgres logs
   docker compose logs postgres
   ```

3. **Migration failures**:
   ```bash
   # Check migration status
   docker compose exec cron migrate -path migrations -database $DATABASE_URL version
   
   # Force migration version
   docker compose exec cron migrate -path migrations -database $DATABASE_URL force 1
   ```

### Reset Everything

```bash
# Stop and remove all containers, networks, and volumes
make docker-clean

# Start fresh
make docker-up
```

## 🔒 Production Considerations

### Security

1. **Change default passwords**:
   ```bash
   # Generate secure passwords
   openssl rand -base64 32
   ```

2. **Use secrets management**:
   ```yaml
   # docker compose.prod.yml
   services:
     postgres:
       environment:
         POSTGRES_PASSWORD_FILE: /run/secrets/postgres_password
       secrets:
         - postgres_password
   ```

### Performance

1. **Resource limits**:
   ```yaml
   services:
     cron:
       deploy:
         resources:
           limits:
             memory: 512M
             cpus: "0.5"
   ```

2. **Volume optimization**:
   ```yaml
   volumes:
     postgres_data:
       driver_opts:
         type: none
         device: /data/postgres
         o: bind
   ```

## 📝 Docker Commands Reference

```bash
# Building
make docker-build              # Build all images
docker compose build --no-cache  # Force rebuild

# Running
make docker-up                 # Start detached
make docker-dev                # Start development mode
docker compose up             # Start with logs

# Monitoring
make docker-logs              # View all logs
docker compose logs -f cron   # Follow specific service
docker compose ps             # List containers

# Maintenance
make docker-down              # Stop services
make docker-clean             # Clean up everything
docker compose restart cron   # Restart specific service
```

---

This Docker setup provides a complete development and deployment environment for the Iddaa Core services with proper service isolation, health checks, and easy management.

**Next**: [Development Guide](development.md)