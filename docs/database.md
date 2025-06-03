# Database Guide

This guide covers the database schema, migrations, and query patterns used in the Iddaa Core system.

## 📊 Schema Overview

### Core Tables

#### `sports`
```sql
CREATE TABLE sports (
    id INTEGER PRIMARY KEY,           -- Sport ID from Iddaa API
    name VARCHAR(100) NOT NULL,      -- Display name (e.g., "Football")
    code VARCHAR(20) NOT NULL        -- Internal code (e.g., "FOOTBALL")
);
```

**Purpose**: Maps sport IDs to human-readable names and internal codes.

#### `competitions`
```sql
CREATE TABLE competitions (
    id SERIAL PRIMARY KEY,
    iddaa_id INTEGER UNIQUE NOT NULL,    -- Iddaa API competition ID
    external_ref INTEGER,               -- External reference ID
    country_code VARCHAR(10),           -- Country code (e.g., "TR", "IT")
    parent_id INTEGER,                  -- Parent competition ID
    sport_id INTEGER REFERENCES sports(id),
    short_name VARCHAR(100),            -- Short name (e.g., "Süper Lig")
    full_name VARCHAR(255) NOT NULL,    -- Full name (e.g., "Türkiye Süper Lig")
    icon_url TEXT,                      -- Competition logo URL
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Purpose**: Stores leagues, tournaments, and competitions from Iddaa API.

#### `teams`
```sql
CREATE TABLE teams (
    id SERIAL PRIMARY KEY,
    external_id VARCHAR(255) UNIQUE NOT NULL,  -- External team ID
    name VARCHAR(255) NOT NULL,               -- Team name
    short_name VARCHAR(100),                  -- Abbreviated name
    country VARCHAR(100),                     -- Team's country
    logo_url TEXT,                           -- Team logo URL
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Purpose**: Team information for matches and events.

#### `events`
```sql
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    external_id VARCHAR(255) UNIQUE NOT NULL,   -- External event ID
    competition_id INTEGER REFERENCES competitions(id),
    home_team_id INTEGER REFERENCES teams(id),
    away_team_id INTEGER REFERENCES teams(id),
    event_date TIMESTAMP NOT NULL,             -- Match date/time
    status VARCHAR(50) NOT NULL,               -- scheduled, live, finished, cancelled
    home_score INTEGER,                        -- Final home score
    away_score INTEGER,                        -- Final away score
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Purpose**: Individual matches/games within competitions.

#### `market_types`
```sql
CREATE TABLE market_types (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,          -- Market code (e.g., "1X2", "OU")
    name VARCHAR(255) NOT NULL,               -- Display name
    description TEXT                          -- Market description
);
```

**Purpose**: Betting market types (1X2, Over/Under, etc.).

#### `odds`
```sql
CREATE TABLE odds (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id),
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(100) NOT NULL,            -- '1', 'X', '2', 'Over 2.5', etc.
    odds_value DECIMAL(10, 3) NOT NULL,       -- Odds value (e.g., 1.850)
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Purpose**: Historical odds data for events and markets.

#### `predictions`
```sql
CREATE TABLE predictions (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id),
    market_type_id INTEGER REFERENCES market_types(id),
    predicted_outcome VARCHAR(100) NOT NULL,  -- Predicted outcome
    confidence_score DECIMAL(5, 4) NOT NULL,  -- 0.0000 to 1.0000
    model_version VARCHAR(50) NOT NULL,       -- AI model version
    features_used TEXT,                       -- JSON of features used
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Purpose**: AI-generated predictions for events.

#### `app_config`
```sql
CREATE TABLE app_config (
    id SERIAL PRIMARY KEY,
    platform VARCHAR(50) UNIQUE NOT NULL,     -- Platform (WEB, MOBILE)
    config_data JSONB NOT NULL,               -- Full config as JSON
    sportoto_program_name VARCHAR(255),       -- Current program name
    payin_end_date TIMESTAMP,                 -- Payin deadline
    next_draw_expected_win DECIMAL(15, 2),    -- Expected winnings
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Purpose**: Platform configuration and system settings.

## 🗂️ Indexes

### Performance Indexes
```sql
-- Competitions
CREATE INDEX idx_competitions_sport_country ON competitions (sport_id, country_code);
CREATE INDEX idx_competitions_iddaa_id ON competitions (iddaa_id);

-- Odds
CREATE INDEX idx_odds_event_market ON odds (event_id, market_type_id);
CREATE INDEX idx_odds_recorded_at ON odds (recorded_at);

-- Predictions  
CREATE INDEX idx_predictions_event_market ON predictions (event_id, market_type_id);
CREATE INDEX idx_predictions_confidence ON predictions (confidence_score);

-- Configuration
CREATE INDEX idx_app_config_updated_at ON app_config (updated_at);
```

## 🔄 Migrations

### Migration Strategy

We use **idempotent migrations** that can be safely run multiple times:

```sql
-- Safe table creation
CREATE TABLE IF NOT EXISTS table_name (...);

-- Safe index creation  
CREATE INDEX IF NOT EXISTS index_name ON table_name (...);

-- Safe data insertion
INSERT INTO table_name (...) VALUES (...)
ON CONFLICT (unique_column) DO UPDATE SET
    column = EXCLUDED.column;
```

### Migration Commands

```bash
# Run migrations
make migrate

# Rollback one migration
make migrate-down

# Check migration status
make migrate-status

# Create new migration
migrate create -ext sql -dir migrations -seq migration_name
```

### Migration Best Practices

1. **Always backup before migrations in production**
2. **Test migrations on production-like data**
3. **Use transactions for complex migrations**
4. **Make migrations idempotent**
5. **Keep migrations small and focused**

## 📝 Query Patterns

### Using sqlc

All database queries use sqlc for type safety:

```sql
-- sql/queries/competitions.sql

-- name: GetCompetitionByIddaaID :one
SELECT c.*, s.name as sport_name, s.code as sport_code
FROM competitions c
JOIN sports s ON c.sport_id = s.id
WHERE c.iddaa_id = sqlc.arg(iddaa_id);

-- name: UpsertCompetition :one
INSERT INTO competitions (iddaa_id, external_ref, country_code, ...)
VALUES (sqlc.arg(iddaa_id), sqlc.arg(external_ref), ...)
ON CONFLICT (iddaa_id) DO UPDATE SET
    external_ref = EXCLUDED.external_ref,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;
```

### Generated Go Code

```go
// pkg/database/competitions.sql.go (generated)

func (q *Queries) GetCompetitionByIddaaID(ctx context.Context, iddaaID int32) (GetCompetitionByIddaaIDRow, error) {
    // Generated implementation
}

func (q *Queries) UpsertCompetition(ctx context.Context, arg UpsertCompetitionParams) (Competition, error) {
    // Generated implementation  
}
```

### Service Layer Usage

```go
// pkg/services/competition_service.go

func (s *CompetitionService) GetCompetition(ctx context.Context, iddaaID int32) (*database.Competition, error) {
    comp, err := s.db.GetCompetitionByIddaaID(ctx, iddaaID)
    if err != nil {
        if err == pgx.ErrNoRows {
            return nil, fmt.Errorf("competition %d not found", iddaaID)
        }
        return nil, fmt.Errorf("failed to get competition: %w", err)
    }
    return &comp, nil
}
```

## 🔍 Common Queries

### Competition Queries

```sql
-- Get all competitions for a sport
SELECT * FROM competitions c
JOIN sports s ON c.sport_id = s.id  
WHERE s.code = 'FOOTBALL'
ORDER BY c.full_name;

-- Get competitions by country
SELECT * FROM competitions 
WHERE country_code = 'TR'
ORDER BY full_name;
```

### Odds Queries

```sql
-- Get latest odds for an event
SELECT o.*, mt.name as market_name
FROM odds o
JOIN market_types mt ON o.market_type_id = mt.id
WHERE o.event_id = $1
AND o.recorded_at = (
    SELECT MAX(recorded_at) 
    FROM odds o2 
    WHERE o2.event_id = o.event_id 
    AND o2.market_type_id = o.market_type_id
    AND o2.outcome = o.outcome
);

-- Get odds history for time range
SELECT * FROM odds 
WHERE event_id = $1 
AND market_type_id = $2
AND recorded_at BETWEEN $3 AND $4
ORDER BY recorded_at DESC;
```

### Prediction Queries

```sql
-- Get prediction accuracy by model
SELECT 
    model_version,
    COUNT(*) as total_predictions,
    AVG(confidence_score) as avg_confidence,
    COUNT(CASE WHEN /* prediction matches result */ THEN 1 END) as correct
FROM predictions p
JOIN events e ON p.event_id = e.id
WHERE e.status = 'finished'
GROUP BY model_version;
```

### Configuration Queries

```sql
-- Get latest config for platform
SELECT * FROM app_config 
WHERE platform = 'WEB'
ORDER BY updated_at DESC 
LIMIT 1;

-- Extract specific config values
SELECT config_data->>'globalConfig'->>'coupon'->>'maxPrice' as max_price
FROM app_config 
WHERE platform = 'WEB';
```

## 🔧 Database Management

### Development Setup

```bash
# Create database
createdb iddaa_core

# Set connection string
export DATABASE_URL="postgres://user:pass@localhost:5432/iddaa_core?sslmode=disable"

# Run migrations
make migrate

# Verify setup
psql $DATABASE_URL -c "\dt"
```

### Production Considerations

1. **Connection Pooling**: Use pgxpool for connection management
2. **Read Replicas**: Consider read replicas for analytical queries
3. **Backup Strategy**: Regular backups of odds and prediction data
4. **Monitoring**: Track query performance and slow queries
5. **Partitioning**: Consider partitioning odds table by date

### Performance Tuning

```sql
-- Analyze query performance
EXPLAIN ANALYZE SELECT ...;

-- Update table statistics
ANALYZE competitions;

-- Monitor index usage
SELECT schemaname, tablename, indexname, idx_tup_read, idx_tup_fetch 
FROM pg_stat_user_indexes;
```

## 🛠️ Tools and Utilities

### sqlc Configuration

```yaml
# sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "sql/queries"
    schema: "sql/schema"
    gen:
      go:
        package: "database"
        out: "pkg/database"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_db_tags: true
        emit_prepared_queries: true
        emit_interface: true
```

### Migration Tools

```bash
# Install golang-migrate
brew install golang-migrate

# Create migration
migrate create -ext sql -dir migrations -seq add_new_table

# Force version (if needed)
migrate -path migrations -database $DATABASE_URL force 1
```

---

This database design supports high-frequency odds updates, historical analysis, and AI prediction workflows while maintaining data integrity and query performance.

**Next**: [API Documentation](api.md)