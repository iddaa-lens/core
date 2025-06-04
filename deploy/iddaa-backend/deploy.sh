#!/bin/bash

# Enable error handling
set -e          # Exit immediately if a command exits with a non-zero status
set -o pipefail # Return value of a pipeline is the value of the last command to exit with non-zero status

# Function for logging
log() {
  echo "$(date +'%Y-%m-%d %H:%M:%S') - $1"
}

# Error handling function
handle_error() {
  local exit_code=$?
  log "❌ Error occurred in script at line: $1, exit code: $exit_code"
  exit $exit_code
}

# Set up error handling
trap 'handle_error $LINENO' ERR

# Configuration
NAMESPACE="iddaa-backend"
RESOURCE_GROUP="omer-test"
AKS_CLUSTER="omer-test-aks"
ORG="iddaa-backend"
TAG="latest"

# Start deployment
log "🚀 Starting deployment process for Iddaa Backend services"

cd ../../

log "🔄 Updating dependencies..."
go mod tidy

log "🔨 Building both services (API + Cron)..."
make build

log "🔐 Logging into container registry..."
az acr login --name omercr || {
  log "❌ Failed to login to ACR"
  exit 1
}

log "🏗️ Building Docker images for both services..."
make build-images ORG=$ORG TAG=$TAG

log "📤 Pushing Docker images to container registry..."
make push-images ORG=$ORG TAG=$TAG

# Run migrations
log "🗃️ Running database migrations..."
export DATABASE_URL="postgresql://iddaa:iddaa123@iddaa-db.postgres.database.azure.com/iddaa_core?sslmode=require"
make migrate

log "🔄 Setting up Kubernetes context..."
az account set --subscription affefc30-fc74-4468-bfde-d54995f061ab
az aks get-credentials --resource-group $RESOURCE_GROUP --name $AKS_CLUSTER --overwrite-existing --admin

cd deploy/iddaa-backend

# Check if namespace exists
if kubectl get namespace $NAMESPACE >/dev/null 2>&1; then
  log "🌐 Namespace $NAMESPACE already exists. Skipping creation."
else
  log "🌐 Creating namespace $NAMESPACE."
  kubectl apply -f namespace.yaml
fi

# Apply secrets (make sure this exists)
log "🔐 Applying secrets..."
if [ -f "secrets.yaml" ]; then
  kubectl apply -f secrets.yaml -n $NAMESPACE
else
  log "⚠️ Warning: secrets.yaml not found. Please create it manually with your secrets."
fi

# Deploy the applications
log "🚀 Deploying Kubernetes resources (API + Cron services)..."
kubectl apply -f deployment.yaml -n $NAMESPACE

# Restart deployments to apply latest images
log "♻️ Restarting deployments to apply latest images..."
kubectl rollout restart deployment/iddaa-backend-api -n $NAMESPACE
kubectl rollout restart deployment/iddaa-backend-cron -n $NAMESPACE

# Apply service configurations
log "🔌 Applying service configurations..."
kubectl apply -f service.yaml -n $NAMESPACE

# Apply ingress configuration
log "🌍 Applying ingress configuration..."
kubectl apply -f ingress.yaml -n $NAMESPACE

# Wait for deployments to be available
log "⏳ Waiting for deployments to be ready..."
kubectl rollout status deployment/iddaa-backend-api -n $NAMESPACE --timeout=300s ||
  log "⚠️ Warning: API deployment not ready within timeout"
kubectl rollout status deployment/iddaa-backend-cron -n $NAMESPACE --timeout=300s ||
  log "⚠️ Warning: Cron deployment not ready within timeout"

# Check deployment status
API_READY=$(kubectl get deployment iddaa-backend-api -n $NAMESPACE -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
CRON_READY=$(kubectl get deployment iddaa-backend-cron -n $NAMESPACE -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")

if [ "$API_READY" -gt 0 ] && [ "$CRON_READY" -gt 0 ]; then
  log "✅ Deployment completed successfully!"
  log "🌐 Services are accessible at:"
  log "   - Iddaa API: https://iddaa-api.betslib.com"
  log "   - Iddaa Cron Service: https://iddaa-cron.betslib.com"
else
  log "⚠️ Deployment completed but some pods may not be ready yet. Please check status."
  log "   API Ready Replicas: $API_READY"
  log "   Cron Ready Replicas: $CRON_READY"
fi

# Show pod status
log "📊 Current pod status:"
kubectl get pods -n $NAMESPACE -l project=iddaa

# Add deployment tags
log "🏷️ Adding deployment metadata..."
az resource tag --resource-group $RESOURCE_GROUP --name $AKS_CLUSTER \
  --resource-type Microsoft.ContainerService/managedClusters \
  --tags Environment=Production Application=IddaaBackend DeployDate="$(date +'%Y-%m-%d')" >/dev/null

log "🔄 Deployment script completed for Iddaa Backend (API + Cron services)."