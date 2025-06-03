# Deployment Guide

This guide covers production deployment strategies, infrastructure requirements, and operational procedures for the Iddaa Core services.

## 🏗️ Infrastructure Requirements

### Minimum System Requirements

#### Application Servers
- **CPU**: 2 cores minimum, 4 cores recommended
- **Memory**: 2GB minimum, 4GB recommended  
- **Storage**: 10GB for application, logs, and temporary files
- **Network**: Reliable internet connection for API calls

#### Database Server
- **CPU**: 4 cores minimum, 8 cores recommended
- **Memory**: 8GB minimum, 16GB recommended
- **Storage**: SSD storage, 100GB minimum
  - Data: 50GB (grows with historical odds)
  - WAL: 10GB
  - Temp: 10GB
  - Backups: 30GB

### Production Architecture

```
┌─────────────────┐    ┌─────────────────┐
│   Load Balancer │    │   Monitoring    │
│   (nginx/HAProxy)│    │  (Prometheus)   │
└─────────┬───────┘    └─────────────────┘
          │                     │
          ▼                     │
┌─────────────────┐             │
│   API Server    │◄────────────┘
│   (Port 8080)   │
└─────────┬───────┘
          │
          ▼
┌─────────────────┐    ┌─────────────────┐
│  Cron Service   │    │   PostgreSQL    │
│  (Background)   │◄──►│   (Primary)     │
└─────────────────┘    └─────────┬───────┘
                                 │
                        ┌─────────▼───────┐
                        │   PostgreSQL    │
                        │  (Read Replica) │
                        └─────────────────┘
```

## 🐳 Docker Deployment

### Multi-Stage Dockerfile

```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/cron ./cmd/cron
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/api ./cmd/api

# Production stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy binaries
COPY --from=builder /app/bin/cron .
COPY --from=builder /app/bin/api .

# Copy migrations
COPY migrations ./migrations

# Install migrate tool
RUN apk add --no-cache curl && \
    curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz && \
    mv migrate /usr/local/bin/

EXPOSE 8080

# Default to cron service
CMD ["./cron"]
```

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: iddaa_core
      POSTGRES_USER: iddaa
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U iddaa"]
      interval: 30s
      timeout: 10s
      retries: 3

  cron:
    build: .
    command: ["./cron"]
    environment:
      - DATABASE_URL=postgres://iddaa:${DB_PASSWORD}@postgres:5432/iddaa_core?sslmode=disable
      - EXTERNAL_API_TIMEOUT=30
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  api:
    build: .
    command: ["./api"]
    environment:
      - DATABASE_URL=postgres://iddaa:${DB_PASSWORD}@postgres:5432/iddaa_core?sslmode=disable
      - PORT=8080
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

volumes:
  postgres_data:
```

### Environment Configuration

```bash
# .env.production
DB_PASSWORD=your_secure_password_here
DATABASE_URL=postgres://iddaa:${DB_PASSWORD}@postgres:5432/iddaa_core?sslmode=disable

# Application
PORT=8080
HOST=0.0.0.0
EXTERNAL_API_TIMEOUT=30

# Monitoring
ENABLE_METRICS=true
METRICS_PORT=9090
```

## ☸️ Kubernetes Deployment

### Namespace and ConfigMap

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: iddaa-core

---
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: iddaa-config
  namespace: iddaa-core
data:
  EXTERNAL_API_TIMEOUT: "30"
  PORT: "8080"
  HOST: "0.0.0.0"
```

### Secrets

```yaml
# k8s/secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: iddaa-secrets
  namespace: iddaa-core
type: Opaque
data:
  DATABASE_URL: <base64-encoded-database-url>
```

### Deployments

```yaml
# k8s/cron-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: iddaa-cron
  namespace: iddaa-core
spec:
  replicas: 1
  selector:
    matchLabels:
      app: iddaa-cron
  template:
    metadata:
      labels:
        app: iddaa-cron
    spec:
      containers:
      - name: cron
        image: iddaa-core:latest
        command: ["./cron"]
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: iddaa-secrets
              key: DATABASE_URL
        envFrom:
        - configMapRef:
            name: iddaa-config
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "1Gi" 
            cpu: "500m"
        livenessProbe:
          exec:
            command:
            - /bin/sh
            - -c
            - "pgrep cron"
          initialDelaySeconds: 30
          periodSeconds: 30

---
# k8s/api-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: iddaa-api
  namespace: iddaa-core
spec:
  replicas: 2
  selector:
    matchLabels:
      app: iddaa-api
  template:
    metadata:
      labels:
        app: iddaa-api
    spec:
      containers:
      - name: api
        image: iddaa-core:latest
        command: ["./api"]
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: iddaa-secrets
              key: DATABASE_URL
        envFrom:
        - configMapRef:
            name: iddaa-config
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "500m"
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 30
```

### Services

```yaml
# k8s/services.yaml
apiVersion: v1
kind: Service
metadata:
  name: iddaa-api-service
  namespace: iddaa-core
spec:
  selector:
    app: iddaa-api
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP
```

## 🗄️ Database Setup

### Production Database Configuration

```sql
-- Create production database
CREATE DATABASE iddaa_core;
CREATE USER iddaa WITH ENCRYPTED PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE iddaa_core TO iddaa;

-- Performance tuning
ALTER SYSTEM SET shared_buffers = '2GB';
ALTER SYSTEM SET effective_cache_size = '6GB';
ALTER SYSTEM SET maintenance_work_mem = '512MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
ALTER SYSTEM SET random_page_cost = 1.1;
ALTER SYSTEM SET effective_io_concurrency = 200;

-- Reload configuration
SELECT pg_reload_conf();
```

### Migration Strategy

```bash
#!/bin/bash
# scripts/migrate.sh

set -e

echo "Starting database migration..."

# Check database connectivity
psql $DATABASE_URL -c "SELECT 1" > /dev/null

# Backup before migration (production only)
if [ "$ENVIRONMENT" = "production" ]; then
    echo "Creating backup..."
    pg_dump $DATABASE_URL > "backup_$(date +%Y%m%d_%H%M%S).sql"
fi

# Run migrations
migrate -path migrations -database $DATABASE_URL up

echo "Migration completed successfully"
```

### Backup Strategy

```bash
#!/bin/bash
# scripts/backup.sh

BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/iddaa_core_${DATE}.sql"

# Create backup directory
mkdir -p $BACKUP_DIR

# Full database backup
pg_dump $DATABASE_URL > $BACKUP_FILE

# Compress backup
gzip $BACKUP_FILE

# Upload to cloud storage (optional)
# aws s3 cp ${BACKUP_FILE}.gz s3://your-backup-bucket/

# Cleanup old backups (keep last 7 days)
find $BACKUP_DIR -name "*.sql.gz" -mtime +7 -delete

echo "Backup completed: ${BACKUP_FILE}.gz"
```

## 📊 Monitoring and Logging

### Application Metrics

```go
// pkg/metrics/metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    JobExecutions = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "iddaa_job_executions_total",
            Help: "Total number of job executions",
        },
        []string{"job_name", "status"},
    )

    JobDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "iddaa_job_duration_seconds",
            Help: "Job execution duration",
        },
        []string{"job_name"},
    )

    APIRequests = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "iddaa_api_requests_total",
            Help: "Total number of API requests",
        },
        []string{"endpoint", "status"},
    )
)
```

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'iddaa-api'
    static_configs:
      - targets: ['iddaa-api-service:9090']
  
  - job_name: 'iddaa-cron'
    static_configs:
      - targets: ['iddaa-cron-service:9090']

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres:9187']
```

### Grafana Dashboard

Key metrics to monitor:
- Job execution success rate
- Job execution duration
- API response times
- Database connection pool usage
- Competition sync frequency
- Error rates by service

## 🚨 Alerting

### Alert Rules

```yaml
# alerts.yml
groups:
- name: iddaa-core
  rules:
  - alert: JobExecutionFailed
    expr: increase(iddaa_job_executions_total{status="failed"}[5m]) > 0
    for: 0m
    labels:
      severity: warning
    annotations:
      summary: "Job execution failed"
      description: "Job {{ $labels.job_name }} has failed"

  - alert: HighJobDuration
    expr: iddaa_job_duration_seconds > 300
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Job taking too long"
      description: "Job {{ $labels.job_name }} taking longer than 5 minutes"

  - alert: DatabaseConnectionError
    expr: up{job="postgres"} == 0
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "Database is down"
      description: "PostgreSQL database is not responding"
```

## 🔐 Security

### Network Security

```yaml
# k8s/network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: iddaa-network-policy
  namespace: iddaa-core
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 5432  # Database
    - protocol: TCP
      port: 443   # HTTPS APIs
```

### RBAC Configuration

```yaml
# k8s/rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: iddaa-service-account
  namespace: iddaa-core

---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: iddaa-role
  namespace: iddaa-core
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: iddaa-role-binding
  namespace: iddaa-core
subjects:
- kind: ServiceAccount
  name: iddaa-service-account
  namespace: iddaa-core
roleRef:
  kind: Role
  name: iddaa-role
  apiGroup: rbac.authorization.k8s.io
```

## 🔄 CI/CD Pipeline

### GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test
          POSTGRES_DB: iddaa_core_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.23

    - name: Run tests
      run: |
        export DATABASE_URL="postgres://postgres:test@localhost:5432/iddaa_core_test?sslmode=disable"
        make deps
        make sqlc
        make migrate
        make test
        make lint

  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Build and push Docker image
      run: |
        docker build -t iddaa-core:${{ github.sha }} .
        docker tag iddaa-core:${{ github.sha }} iddaa-core:latest
        # Push to container registry

  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment: production
    steps:
    - name: Deploy to Kubernetes
      run: |
        kubectl set image deployment/iddaa-cron cron=iddaa-core:${{ github.sha }}
        kubectl set image deployment/iddaa-api api=iddaa-core:${{ github.sha }}
        kubectl rollout status deployment/iddaa-cron
        kubectl rollout status deployment/iddaa-api
```

## 📋 Deployment Checklist

### Pre-Deployment

- [ ] Run full test suite
- [ ] Verify database migrations
- [ ] Check configuration files
- [ ] Backup production database
- [ ] Verify monitoring setup
- [ ] Test rollback procedure

### Deployment

- [ ] Deploy to staging environment
- [ ] Run smoke tests
- [ ] Deploy to production
- [ ] Verify service health
- [ ] Check metrics and logs
- [ ] Validate data synchronization

### Post-Deployment

- [ ] Monitor error rates
- [ ] Verify job executions
- [ ] Check database performance
- [ ] Update documentation
- [ ] Communicate deployment status

---

This deployment guide ensures reliable, scalable, and secure production operations for the Iddaa Core services with proper monitoring, alerting, and maintenance procedures.

**Previous**: [API Documentation](api.md) | **Next**: [Development Guide](development.md)