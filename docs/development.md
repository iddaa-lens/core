# Development Guide

This guide covers local development setup, workflows, and best practices for the Iddaa Core backend services.

## 🛠️ Local Development Setup

### Prerequisites

1. **Go 1.23+**
   ```bash
   # macOS
   brew install go
   
   # Verify installation
   go version
   ```

2. **PostgreSQL 12+**
   ```bash
   # macOS
   brew install postgresql
   brew services start postgresql
   
   # Create development database
   createdb iddaa_core
   ```

3. **Development Tools**
   ```bash
   # Install migration tool
   brew install golang-migrate
   
   # Install sqlc
   go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
   
   # Install linting tools (optional, will auto-install via make lint)
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

### Environment Setup

Create a `.env` file in the project root:

```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_username
DB_PASSWORD=your_password
DB_NAME=iddaa_core
DB_SSLMODE=disable

# Server Configuration
PORT=8080
HOST=localhost

# External API Configuration
EXTERNAL_API_TIMEOUT=30

# Composite Database URL
DATABASE_URL=postgres://your_username:your_password@localhost:5432/iddaa_core?sslmode=disable
```

### Initial Setup

```bash
# Clone and setup
git clone <repository-url>
cd iddaa-core

# Install dependencies
make deps

# Generate database code
make sqlc

# Run database migrations
make migrate

# Build services
make build

# Run tests
make test
```

## 🔄 Development Workflow

### 1. Database Changes

When modifying the database schema:

```bash
# 1. Edit schema files
vim sql/schema/001_initial.sql

# 2. Update migration files  
cp sql/schema/001_initial.sql migrations/000001_initial.up.sql

# 3. Regenerate sqlc code
make sqlc

# 4. Run migrations
make migrate

# 5. Test changes
make test
```

### 2. Adding New Cron Jobs

```bash
# 1. Create job implementation
cat > pkg/jobs/new_job.go << 'EOF'
package jobs

import "context"

type NewJob struct {
    service SomeService
}

func NewNewJob(service SomeService) Job {
    return &NewJob{service: service}
}

func (j *NewJob) Execute(ctx context.Context) error {
    // Implementation
    return nil
}

func (j *NewJob) Name() string {
    return "New Job"
}

func (j *NewJob) Schedule() string {
    return "0 */2 * * *" // Every 2 hours
}
EOF

# 2. Register job in cmd/cron/main.go
# 3. Add tests
# 4. Build and test
make build-cron
make test
```

### 3. Adding New SQL Queries

```bash
# 1. Add queries to appropriate file
vim sql/queries/competitions.sql

# 2. Regenerate sqlc code
make sqlc

# 3. Use in service layer
vim pkg/services/competition_service.go

# 4. Add tests
make test
```

## 🧪 Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with race detection
make test-race

# Run specific package tests
go test -v ./pkg/models

# Run specific test
go test -v ./pkg/jobs -run TestJobManager
```

### Writing Tests

Follow these patterns:

```go
// Service tests with mocks
func TestServiceMethod(t *testing.T) {
    mockDB := &MockDB{}
    service := NewService(mockDB)
    
    result, err := service.Method(context.Background())
    
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}

// HTTP client tests with test server
func TestAPIClient(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(mockResponse)
    }))
    defer server.Close()
    
    client := NewClient(config)
    client.baseURL = server.URL
    
    result, err := client.GetData()
    assert.NoError(t, err)
}
```

## 🔍 Code Quality

### Linting

```bash
# Run all linting tools
make lint

# Individual tools
go vet ./...
go fmt ./...
golangci-lint run
```

### Code Organization

- **`pkg/`** - Public packages that can be imported
- **`internal/`** - Private packages, not importable
- **`cmd/`** - Application entry points
- **`scripts/`** - Build and deployment scripts

### Best Practices

1. **Error Handling**
   ```go
   // Wrap errors with context
   if err != nil {
       return fmt.Errorf("failed to fetch competitions: %w", err)
   }
   ```

2. **Logging**
   ```go
   // Structured logging
   log.Printf("Starting job: %s", job.Name())
   log.Printf("Job %s completed in %v", job.Name(), duration)
   ```

3. **Context Usage**
   ```go
   // Always pass context, especially for database operations
   func (s *Service) Method(ctx context.Context) error {
       return s.db.Query(ctx, params)
   }
   ```

4. **Configuration**
   ```go
   // Use environment variables with defaults
   func getEnv(key, defaultValue string) string {
       if value := os.Getenv(key); value != "" {
           return value
       }
       return defaultValue
   }
   ```

## 🐛 Debugging

### Common Issues

1. **Database Connection Errors**
   ```bash
   # Check PostgreSQL is running
   brew services list | grep postgresql
   
   # Verify database exists
   psql -l | grep iddaa_core
   
   # Test connection
   psql $DATABASE_URL -c "SELECT 1"
   ```

2. **Migration Errors**
   ```bash
   # Check migration status
   make migrate-status
   
   # Rollback if needed
   make migrate-down
   
   # Re-run migrations
   make migrate
   ```

3. **sqlc Generation Issues**
   ```bash
   # Check sqlc configuration
   cat sqlc.yaml
   
   # Verify SQL syntax
   psql $DATABASE_URL -f sql/schema/001_initial.sql
   
   # Clean and regenerate
   rm -rf pkg/database/*
   make sqlc
   ```

### Debugging Tools

```bash
# Database inspection
psql $DATABASE_URL
\dt                    # List tables
\d competitions        # Describe table
SELECT COUNT(*) FROM competitions;

# Log analysis
./bin/cron 2>&1 | tee cron.log

# Performance monitoring
go test -bench=. -benchmem ./...
```

## 🚀 Local Testing

### Running Services Locally

```bash
# Terminal 1: Run cron service
export DATABASE_URL="postgres://user:pass@localhost:5432/iddaa_core?sslmode=disable"
make run-cron

# Terminal 2: Monitor logs
tail -f cron.log

# Terminal 3: Check database
psql $DATABASE_URL -c "SELECT COUNT(*) FROM competitions"
```

### Testing External API Integration

```bash
# Test competition API directly
curl -s "https://sportsbookv2.iddaa.com/sportsbook/competitions" | jq '.data | length'

# Test config API
curl -s "https://contentv2.iddaa.com/appconfig?platform=WEB" | jq '.data.platform'

# Monitor network requests
tcpdump -i en0 host sportsbookv2.iddaa.com
```

## 📝 Documentation

When adding new features:

1. Update relevant markdown files in `docs/`
2. Add inline code comments for complex logic
3. Update `CLAUDE.md` with new patterns or commands
4. Add examples to `README.md` if user-facing

## 🔄 Git Workflow

```bash
# Feature development
git checkout -b feature/new-feature
git add .
git commit -m "feat: add new feature"
git push origin feature/new-feature

# Create pull request with:
# - Clear description
# - Test results
# - Any breaking changes noted
```

---

Happy coding! 🎉