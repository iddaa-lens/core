# Iddaa Backend Deployment

This directory contains Kubernetes deployment manifests for the **Iddaa Backend** service, which consists of two main components:

1. **API Service** - REST API for odds history and AI predictions
2. **Cron Service** - Background jobs for data synchronization with iddaa API and Football API

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐
│   API Service   │    │  Cron Service   │
│   (Port 8080)   │    │  (Background)   │
│                 │    │                 │
│ • REST API      │    │ • Data Sync     │
│ • Odds History  │    │ • Football API  │
│ • Predictions   │    │ • Scheduled Jobs│
└─────────────────┘    └─────────────────┘
         │                       │
         └───────────┬───────────┘
                     │
            ┌─────────────────┐
            │   PostgreSQL    │
            │   Database      │
            └─────────────────┘
```

## 🚀 Quick Deployment

### Prerequisites
- Azure CLI installed and logged in
- kubectl configured for your AKS cluster
- Docker installed for building images

### 1. Build and Deploy
```bash
# From the project root directory
cd /path/to/iddaa-core

# Build Docker images
make build-images ORG=iddaa-backend TAG=latest

# Push to registry
make push-images ORG=iddaa-backend TAG=latest

# Deploy to Kubernetes
cd deploy/iddaa-backend
./deploy.sh
```

### 2. Manual Step-by-Step Deployment

```bash
# 1. Create namespace
kubectl apply -f namespace.yaml

# 2. Create secrets (update with your values first!)
kubectl apply -f secrets.yaml

# 3. Deploy services
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f ingress.yaml
```

## 🔐 Configuration

### Required Secrets
Update `secrets.yaml` with your actual values:

- **DATABASE_URL**: PostgreSQL connection string
- **FOOTBALL_API_KEY**: API key for football data enrichment

### Environment Variables
- **PORT**: API service port (default: 8080)
- **HOST**: Bind host (default: 0.0.0.0)
- **EXTERNAL_API_TIMEOUT**: API timeout in seconds (default: 30)
- **FOOTBALL_API_TIMEOUT**: Football API timeout in seconds (default: 30)

## 🌐 Endpoints

After deployment, services will be available at:

- **API Service**: https://iddaa-api.betslib.com
- **Cron Service**: https://iddaa-cron.betslib.com (monitoring only)

## 📊 Monitoring

### Health Checks
- API: `GET /health` - Service health
- API: `GET /ready` - Readiness probe

### Pod Status
```bash
# Check pod status
kubectl get pods -n iddaa-backend

# Check deployment status
kubectl get deployments -n iddaa-backend

# View logs
kubectl logs -f deployment/iddaa-backend-api -n iddaa-backend
kubectl logs -f deployment/iddaa-backend-cron -n iddaa-backend
```

## 🔧 Troubleshooting

### Common Issues

1. **Pods not starting**
   ```bash
   kubectl describe pod <pod-name> -n iddaa-backend
   ```

2. **Database connection issues**
   - Verify DATABASE_URL in secrets
   - Check network connectivity to database

3. **Image pull errors**
   - Ensure you're logged into the container registry
   - Verify image tags are correct

### Scaling

```bash
# Scale API service
kubectl scale deployment iddaa-backend-api --replicas=3 -n iddaa-backend

# Scale cron service (keep at 1 to avoid duplicate jobs)
kubectl scale deployment iddaa-backend-cron --replicas=1 -n iddaa-backend
```

## 📋 Deployment Checklist

- [ ] Update `secrets.yaml` with production values
- [ ] Build and push Docker images
- [ ] Run database migrations
- [ ] Apply Kubernetes manifests
- [ ] Verify pod health
- [ ] Test API endpoints
- [ ] Monitor cron job execution

## 🔄 Updates

To update the deployment:

```bash
# Build new images
make build-images ORG=iddaa-backend TAG=v1.1.0

# Push new images
make push-images ORG=iddaa-backend TAG=v1.1.0

# Update deployment with new tag
kubectl set image deployment/iddaa-backend-api iddaa-backend-api=omercr.azurecr.io/iddaa-backend/api:v1.1.0 -n iddaa-backend
kubectl set image deployment/iddaa-backend-cron iddaa-backend-cron=omercr.azurecr.io/iddaa-backend/cron:v1.1.0 -n iddaa-backend
```