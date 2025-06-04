# BetsLib Kubernetes Deployment

This directory contains Kubernetes manifests and deployment scripts for the complete BetsLib platform including application services and logging infrastructure.

## Structure

```
deploy/
├── deploy-all.sh           # Complete deployment script
├── betslib/               # Application services
│   ├── deployment.yaml    # Core API and Cron services
│   ├── service.yaml       # Kubernetes services
│   ├── ingress.yaml       # External routing (includes Kibana)
│   ├── namespace.yaml     # Namespace definition
│   └── secrets.yaml       # Sensitive configuration (DO NOT COMMIT)
└── logging/               # ELK Stack for log management
    ├── deploy-elk.sh      # ELK-only deployment script
    ├── elasticsearch.yaml # Log storage and search
    ├── kibana.yaml        # Log visualization UI
    ├── fluent-bit.yaml    # Log collection daemon
    └── README.md          # Detailed logging documentation
```

## Quick Deployment

### Option 1: Deploy Everything (Recommended)
```bash
cd deploy
./deploy-all.sh
```

### Option 2: Deploy Only Application Services
```bash
cd deploy
./deploy-all.sh false
```

### Option 3: Deploy Only Logging Stack
```bash
cd deploy/logging
./deploy-elk.sh
```

## Services

### Application Services (namespace: `betslib-test`)

| Service | Purpose | Port | External URL |
|---------|---------|------|--------------|
| `betslib-core` | REST API server | 8080 | https://api.betslib.com |
| `betslib-cron` | Background jobs | 8081 | https://cron.betslib.com |

### Logging Services (namespace: `betslib-test`)

| Service | Purpose | Port | External URL |
|---------|---------|------|--------------|
| `elasticsearch` | Log storage | 9200 | Internal only |
| `kibana` | Log visualization | 5601 | https://logs.betslib.com |
| `fluent-bit` | Log collection | 2020 | DaemonSet (no external access) |

## Configuration Management

### Secrets
**Important**: The `betslib/secrets.yaml` file contains sensitive API keys and credentials. 

- ✅ **DO**: Create this file manually in your environment
- ❌ **DON'T**: Commit this file to version control
- 🔄 **UPDATE**: Keep secrets in sync across environments

### Environment Variables
Your deployments now use Kubernetes secrets instead of hardcoded values:

```yaml
# Before (insecure)
- name: OPENAI_API_KEY
  value: "sk-proj-..."

# After (secure)
- name: OPENAI_API_KEY
  valueFrom:
    secretKeyRef:
      name: betslib-secrets
      key: OPENAI_API_KEY
```

## Monitoring Your Services

### Application Health
```bash
# Check all pods
kubectl get pods -n betslib-test

# Check specific service logs
kubectl logs -f deployment/betslib-core -n betslib-test
kubectl logs -f deployment/betslib-cron -n betslib-test

# Health checks
curl https://api.betslib.com/health
curl https://cron.betslib.com/health
```

### Manual Cron Triggers
Your cron service exposes HTTP endpoints for manual job triggering:

```bash
# Trigger prediction generation
curl -X POST https://cron.betslib.com/api/triggers/predictions

# Trigger score updates
curl -X POST https://cron.betslib.com/api/triggers/scores

# List all available triggers
curl https://cron.betslib.com/api/triggers
```

### Log Analysis
With the ELK stack deployed, you can analyze your enhanced prediction logging:

1. **Access Kibana**: https://logs.betslib.com
2. **Create index pattern**: `betslib-*`
3. **Search examples**:
   ```kql
   # View prediction statistics
   service: "betslib-cron" AND message: "prediction_candidates"
   
   # Monitor prediction generation results
   service: "betslib-cron" AND message: "predictions_generated"
   
   # Track failed predictions
   service: "betslib-cron" AND message: "prediction_failures"
   
   # View API requests
   service: "betslib-core" AND message: "HTTP"
   ```

## Security Considerations

### TLS/SSL
- All external services use HTTPS with Let's Encrypt certificates
- `cert-manager` automatically manages certificate renewal
- Ingress controller enforces SSL redirect

### Network Policies
- Services communicate within the cluster network
- Only necessary ports are exposed externally
- Elasticsearch is not directly accessible from outside

### Secret Management
- API keys stored in Kubernetes secrets
- Secrets mounted as environment variables
- No sensitive data in container images or deployment files

## Scaling

### Horizontal Scaling
```bash
# Scale API service
kubectl scale deployment betslib-core --replicas=3 -n betslib-test

# Scale cron service (usually keep at 1)
kubectl scale deployment betslib-cron --replicas=1 -n betslib-test
```

### Resource Limits
Current resource configuration:
- **betslib-core**: 100m-500m CPU, 256Mi-512Mi memory
- **betslib-cron**: 100m-500m CPU, 256Mi-512Mi memory
- **elasticsearch**: 250m-500m CPU, 512Mi-1Gi memory
- **kibana**: 250m-500m CPU, 256Mi-512Mi memory

## Troubleshooting

### Common Issues

**Pods stuck in Pending:**
```bash
kubectl describe pod <pod-name> -n betslib-test
# Check for resource constraints or node availability
```

**Service not accessible:**
```bash
kubectl get ingress -n betslib-test
kubectl describe ingress betslib-services -n betslib-test
```

**Database connection issues:**
```bash
kubectl logs deployment/betslib-core -n betslib-test | grep -i database
kubectl exec -it deployment/betslib-core -n betslib-test -- env | grep DATABASE_URL
```

**Logging not working:**
```bash
kubectl logs daemonset/fluent-bit -n betslib-test
kubectl port-forward svc/elasticsearch 9200:9200 -n betslib-test
curl http://localhost:9200/_cluster/health
```

### Support Commands
```bash
# Get all resources
kubectl get all -n betslib-test

# Debug networking
kubectl exec -it deployment/betslib-core -n betslib-test -- nslookup elasticsearch

# Check resource usage
kubectl top pods -n betslib-test
```

## Development Workflow

1. **Make changes** to your Go code
2. **Build images** and push to your container registry
3. **Update deployment** with new image tags
4. **Apply changes**: `kubectl apply -f betslib/deployment.yaml`
5. **Monitor logs** via Kibana or kubectl

Your BetsLib platform is now ready for production with comprehensive monitoring and logging capabilities!