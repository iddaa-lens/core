#!/bin/bash

# Production Deployment Validation Script
# Run this script after infrastructure is set up to validate the deployment

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="iddaa-backend"
API_SERVICE="iddaa-api"
CRON_SERVICE="iddaa-cron"

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

# Function to check if command exists
check_command() {
    if ! command -v $1 &> /dev/null; then
        error "$1 could not be found. Please install it."
        exit 1
    fi
}

# Check prerequisites
log "🔍 Checking prerequisites..."
check_command kubectl
check_command az
check_command psql

# Check Azure login
log "🔐 Checking Azure authentication..."
if ! az account show &> /dev/null; then
    error "Not logged in to Azure. Please run 'az login'"
    exit 1
fi

# Check Kubernetes context
log "⚙️ Checking Kubernetes context..."
if ! kubectl cluster-info &> /dev/null; then
    error "Cannot connect to Kubernetes cluster. Please check your context."
    exit 1
fi

# Check if namespace exists
log "🌐 Checking namespace..."
if ! kubectl get namespace $NAMESPACE &> /dev/null; then
    error "Namespace $NAMESPACE does not exist. Please create it first."
    exit 1
fi

# Validate PostgreSQL connection
log "🗃️ Validating database connection..."
if [ -z "$DATABASE_URL" ]; then
    error "DATABASE_URL environment variable is not set"
    exit 1
fi

if ! psql "$DATABASE_URL" -c "SELECT 1;" &> /dev/null; then
    error "Cannot connect to production database"
    exit 1
fi

log "✅ Database connection successful"

# Check if services are deployed
log "🚀 Checking service deployments..."

# Check API deployment
if kubectl get deployment $API_SERVICE -n $NAMESPACE &> /dev/null; then
    API_READY=$(kubectl get deployment $API_SERVICE -n $NAMESPACE -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    API_DESIRED=$(kubectl get deployment $API_SERVICE -n $NAMESPACE -o jsonpath='{.spec.replicas}')
    
    if [ "$API_READY" -eq "$API_DESIRED" ]; then
        log "✅ API service: $API_READY/$API_DESIRED pods ready"
    else
        warn "⚠️ API service: $API_READY/$API_DESIRED pods ready"
    fi
else
    error "❌ API deployment not found"
fi

# Check Cron deployment
if kubectl get deployment $CRON_SERVICE -n $NAMESPACE &> /dev/null; then
    CRON_READY=$(kubectl get deployment $CRON_SERVICE -n $NAMESPACE -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    CRON_DESIRED=$(kubectl get deployment $CRON_SERVICE -n $NAMESPACE -o jsonpath='{.spec.replicas}')
    
    if [ "$CRON_READY" -eq "$CRON_DESIRED" ]; then
        log "✅ Cron service: $CRON_READY/$CRON_DESIRED pods ready"
    else
        warn "⚠️ Cron service: $CRON_READY/$CRON_DESIRED pods ready"
    fi
else
    error "❌ Cron deployment not found"
fi

# Check API health endpoint
log "🏥 Testing API health endpoint..."
API_SERVICE_IP=$(kubectl get service iddaa-api-service -n $NAMESPACE -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")

if [ -n "$API_SERVICE_IP" ]; then
    if curl -s -f "http://$API_SERVICE_IP:8080/health" > /dev/null; then
        log "✅ API health endpoint responding"
    else
        warn "⚠️ API health endpoint not responding"
    fi
else
    log "ℹ️ API service IP not yet assigned, using port-forward to test..."
    kubectl port-forward service/iddaa-api-service 8080:8080 -n $NAMESPACE &
    PORTFORWARD_PID=$!
    sleep 5
    
    if curl -s -f "http://localhost:8080/health" > /dev/null; then
        log "✅ API health endpoint responding via port-forward"
    else
        warn "⚠️ API health endpoint not responding via port-forward"
    fi
    
    kill $PORTFORWARD_PID 2>/dev/null || true
fi

# Check database tables
log "📊 Validating database schema..."
TABLES=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';")
if [ "$TABLES" -gt 10 ]; then
    log "✅ Database schema deployed: $TABLES tables found"
else
    warn "⚠️ Database schema may be incomplete: only $TABLES tables found"
fi

# Check if data is being populated
log "📈 Checking data population..."
SPORTS_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM sports;" 2>/dev/null || echo "0")
EVENTS_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM events;" 2>/dev/null || echo "0")
MARKETS_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM market_types;" 2>/dev/null || echo "0")

log "📊 Data Summary:"
echo "   - Sports: $SPORTS_COUNT"
echo "   - Events: $EVENTS_COUNT" 
echo "   - Market Types: $MARKETS_COUNT"

if [ "$SPORTS_COUNT" -gt 0 ] && [ "$MARKETS_COUNT" -gt 100 ]; then
    log "✅ Core data populated successfully"
else
    warn "⚠️ Core data may not be fully populated yet"
fi

# Check cron job logs
log "📋 Checking recent cron job activity..."
kubectl logs --tail=20 deployment/$CRON_SERVICE -n $NAMESPACE

log "🎉 Deployment validation completed!"
log "📊 Summary:"
echo "   - Infrastructure: Ready"
echo "   - API Service: $([ "$API_READY" -eq "$API_DESIRED" ] && echo "✅ Ready" || echo "⚠️ Partial")"
echo "   - Cron Service: $([ "$CRON_READY" -eq "$CRON_DESIRED" ] && echo "✅ Ready" || echo "⚠️ Partial")"
echo "   - Database: $([ "$SPORTS_COUNT" -gt 0 ] && echo "✅ Populated" || echo "⚠️ Empty")"

log "🔗 Access URLs:"
echo "   - API Health: http://$API_SERVICE_IP:8080/health"
echo "   - Kubernetes Dashboard: kubectl proxy"
echo "   - Logs: kubectl logs -f deployment/$CRON_SERVICE -n $NAMESPACE"