# Production Deployment Checklist

This document outlines the comprehensive steps required to deploy the iddaa-core backend services to production.

## Pre-Deployment Validation

### Code Quality & Testing

- [ ] **Run final comprehensive tests** (`make test`, `make lint`, `make sqlc`)
- [ ] **Test database migrations up/down cycle** to ensure stability
- [ ] **Verify all cron jobs work correctly** with new anti-bot headers
  - Sports sync
  - Events sync  
  - Volume sync
  - Distributions sync
- [ ] **Test Iddaa API integration** with production-ready headers to prevent blacklisting

## Infrastructure Setup

### Database & Environment

- [ ] **Set up production PostgreSQL database** with proper credentials and sizing
- [ ] **Create Kubernetes namespace and secrets** for database credentials and API keys
- [ ] **Build and push Docker images** for both API and cron services

  ```bash
  make build-images ORG=iddaa-backend TAG=latest
  make push-images ORG=iddaa-backend TAG=latest
  ```

- [ ] **Deploy database migrations** to production database

  ```bash
  export DATABASE_URL="postgresql://[prod-credentials]"
  make migrate
  ```

## Service Deployment

### Core Services

- [ ] **Deploy cron service to Kubernetes** with proper resource limits and scheduling
- [ ] **Deploy API service to Kubernetes** with health checks and horizontal pod autoscaling
- [ ] **Set up ingress and load balancer** for API service external access

## Monitoring & Validation

### Observability

- [ ] **Configure logging and monitoring** for both services (Prometheus/Grafana)
- [ ] **Set up alerts** for cron job failures and API downtime
- [ ] **Monitor Iddaa API rate limits** and response times to prevent blocking

### Functional Testing

- [ ] **Verify cron jobs are running on schedule** and fetching data successfully
- [ ] **Check database tables are being populated** correctly with real data
- [ ] **Test API health endpoint** is responding correctly (`GET /health`)

## Security & Compliance

### Production Security

- [ ] **Review and secure all environment variables** and secrets
- [ ] **Ensure database connections use SSL/TLS** in production
- [ ] **Verify anti-bot headers** are working and preventing Iddaa blacklisting

## Documentation & Future Planning

### Knowledge Transfer

- [ ] **Document known limitations** (Football API league matching incomplete - only 38/X leagues mapped)
- [ ] **Plan next phase** improvements:
  - AI bulk league matching enhancements
  - More comprehensive Football API integration
  - Advanced analytics and predictions

## Critical Production Notes

### High Priority Items

1. **Anti-Bot Headers**: The Iddaa client now includes comprehensive browser-like headers to prevent bot detection and blacklisting. This is critical for production stability.

2. **League Mappings**: Currently only partial league mappings exist. The AI translation service is available but needs further tuning to improve matching accuracy.

3. **Database Migrations**: All migrations have been consolidated into a single file for development. Production should use the consolidated migration.

4. **External API Dependencies**:
   - Iddaa API (primary data source)
   - Football API (league/team mapping)
   - OpenAI API (AI translation service)

### Service Architecture

- **API Service**: Lightweight health check service (no database dependency)
- **Cron Service**: Data fetching and processing service (database dependent)
- **Database**: PostgreSQL with time-series data for odds tracking

### Deployment Commands

```bash
# Build and deploy
cd deploy/iddaa-backend
./deploy.sh

# Monitor deployment
kubectl get pods -n iddaa-backend
kubectl logs -f deployment/iddaa-cron -n iddaa-backend
```

---

**Last Updated**: January 2025  
**Version**: 1.0  
**Status**: Ready for Production Deployment
