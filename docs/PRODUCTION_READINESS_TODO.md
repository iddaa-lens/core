# Production Readiness TODO - Cron Jobs

Based on comprehensive review by Copilot Opus, this document outlines critical improvements needed for production-ready cron job execution.

## Critical Issues Identified

### 🚨 Priority 1 - Immediate (Data Integrity Risk)

#### 1. Distributed Locking

**Problem**: Multiple Kubernetes replicas can run the same job simultaneously, causing:

- Data corruption
- Duplicate processing  
- Race conditions
- Inconsistent state

**Current Risk**: HIGH - Production has multiple pod replicas
**Implementation**: PostgreSQL advisory locks

#### 2. Monitoring & Metrics Blind Spots

**Problem**: No observability into job execution:

- No metrics collection
- No health checks per job
- No alerting on failures
- No performance tracking

**Current Risk**: HIGH - Flying blind in production
**Implementation**: Prometheus metrics + health checks

#### 3. Memory Issues in High-Volume Jobs

**Problem**: Jobs load all data into memory:

- `detailed_odds_sync.go` loads all events at once
- Potential OOM in production
- Poor scalability

**Current Risk**: MEDIUM - Could cause pod crashes
**Implementation**: Pagination and streaming

### ⚠️ Priority 2 - High (Reliability & Performance)

#### 4. Incomplete Circuit Breaker Coverage

**Problem**: External API failures can cascade:

- Iddaa API (partially addressed)
- OpenAI API (no protection)
- No comprehensive failure handling

**Current Risk**: MEDIUM - API outages affect multiple jobs
**Implementation**: Circuit breaker pattern for all external APIs

#### 5. Database Connection Management

**Problem**: No connection limiting per job:

- No proper transaction management in batch operations
- Potential connection pool exhaustion
- No transaction boundaries for data consistency

**Current Risk**: MEDIUM - Could affect database performance
**Implementation**: Connection limiting + transaction boundaries

#### 6. Idempotency Gaps

**Problem**: Some operations lack idempotency guarantees:

- Jobs can create duplicate data if run multiple times
- Retry logic may cause inconsistency

**Current Risk**: MEDIUM - Data consistency issues
**Implementation**: Idempotency checks and keys

### 📈 Priority 3 - Medium (Operational Excellence)

#### 7. Race Conditions Between Jobs

**Problem**: Multiple jobs updating same data without coordination:

- `volume_sync` and `distribution_sync` can conflict
- `detailed_odds_sync` and `events_sync` can race

**Current Risk**: LOW - Mostly handled by database constraints
**Implementation**: Job dependency management

#### 8. Advanced Error Handling

**Problem**: Jobs continue processing after errors:

- Can lead to partial data states
- Insufficient error boundaries

**Current Risk**: LOW - Recently improved significantly
**Implementation**: Enhanced error boundaries and rollback

#### 9. Graceful Shutdown

**Problem**: Jobs don't handle shutdown gracefully:

- No proper cleanup on termination
- Potential data loss on pod restart

**Current Risk**: LOW - Jobs are generally idempotent
**Implementation**: Graceful shutdown with timeout

## Implementation Plan

### Phase 1: Critical Infrastructure (Week 1-2)

**Goal**: Prevent data corruption and add basic observability

1. **Distributed Locking System** ✅ COMPLETED
   - [x] Create `JobLockManager` interface using PostgreSQL advisory locks
   - [x] Implement lock acquisition/release with timeouts
   - [x] Add lock wrapper to existing jobs (`ProductionJob`)
   - [x] Test with multiple replicas (concurrent execution prevented)
   - [x] Add `--production-mode` flag for safe rollout

1. **Basic Metrics Collection**
   - [ ] Add Prometheus metrics endpoint
   - [ ] Implement job duration tracking
   - [ ] Add success/failure counters
   - [ ] Create basic dashboard

1. **Health Check System**
   - [ ] Create `HealthCheckableJob` interface
   - [ ] Implement per-job health endpoints
   - [ ] Add health check aggregation
   - [ ] Configure liveness/readiness probes

### Phase 2: Reliability Improvements (Week 3-4)

**Goal**: Improve reliability and performance under load

1. **Comprehensive Circuit Breakers**
   - [ ] Extend circuit breaker to all external APIs
   - [ ] Add OpenAI API circuit breaker
   - [ ] Implement API quota management
   - [ ] Add circuit breaker metrics

1. **Memory & Performance Optimization**
   - [ ] Implement pagination in `detailed_odds_sync`
   - [ ] Add streaming for high-volume jobs
   - [ ] Optimize memory usage patterns
   - [ ] Add memory usage metrics

1. **Database Transaction Management**
   - [ ] Add transaction boundaries for batch operations
   - [ ] Implement connection limiting per job
   - [ ] Add database connection metrics
   - [ ] Optimize concurrent database access

### Phase 3: Operational Excellence (Week 5-6)

**Goal**: Production-grade operational capabilities

1. **Advanced Monitoring & Alerting**
   - [ ] Create comprehensive alerting rules
   - [ ] Add job dependency tracking
   - [ ] Implement performance benchmarking
   - [ ] Create operational runbooks

1. **Enhanced Error Recovery**
   - [ ] Implement job dependency management
   - [ ] Add advanced retry logic with exponential backoff
   - [ ] Create error categorization system
   - [ ] Add automated recovery mechanisms

1. **Production Hardening**
   - [ ] Implement graceful shutdown with timeout
   - [ ] Add data validation before processing
   - [ ] Create job scheduling optimization
   - [ ] Add capacity planning metrics

## Recently Completed ✅

- **API Rate Limiting**: Football API with exponential backoff and caching
- **Timeout Handling**: Context deadline management with fallback mechanisms
- **Smart Money Tracker**: Fixed duplicate alert creation errors
- **Database Performance**: Added composite indexes for query optimization
- **Distributed Locking**: Complete PostgreSQL advisory lock implementation with production wrapper
  - Prevents concurrent job execution across Kubernetes replicas
  - RAII-style lock management with automatic cleanup
  - Production-ready with configurable timeouts and skip behavior
  - Backward compatible with `--production-mode` flag

## Architecture Components to Implement

### 1. Production Job Wrapper

```go
type ProductionJob struct {
    job Job
    lockManager *JobLockManager
    metrics *JobMetrics
    circuitBreaker *CircuitBreaker
    healthChecker *JobHealthChecker
}
```

### 2. Distributed Lock Manager

```go
type JobLockManager interface {
    AcquireLock(ctx context.Context, jobName string) (bool, error)
    ReleaseLock(ctx context.Context, jobName string) error
    IsLocked(ctx context.Context, jobName string) (bool, error)
}
```

### 3. Job Metrics System

```go
type JobMetrics struct {
    Duration prometheus.Histogram
    Successes prometheus.Counter
    Failures prometheus.Counter
    ActiveJobs prometheus.Gauge
}
```

### 4. Health Check Interface

```go
type HealthCheckableJob interface {
    Job
    HealthCheck(ctx context.Context) error
    GetStatus(ctx context.Context) JobStatus
}
```

## Success Criteria

### Phase 1 Complete When

- [x] Zero data corruption from concurrent job execution (✅ Distributed locking implemented)
- [ ] Basic job metrics visible in monitoring dashboard
- [ ] Health checks prevent deployment of broken jobs

### Phase 2 Complete When

- [ ] Jobs handle external API outages gracefully
- [ ] Memory usage stable under high load
- [ ] Database connection exhaustion eliminated

### Phase 3 Complete When

- [ ] Proactive alerting prevents most production issues
- [ ] Jobs recover automatically from most failure scenarios
- [ ] System provides production-grade operational visibility

## References

- Original Review: Copilot Opus Cron Job Analysis
- Job Manager Implementation: `pkg/jobs/manager.go`
- Current Job Interfaces: `pkg/jobs/interface.go`
- Database Connection: `pkg/database/db.go`
- Monitoring Endpoint: `pkg/handlers/health/handler.go`

---

**Next Action**: Begin Phase 1 implementation with distributed locking system
