package pool

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config represents optimized database connection pool settings
type Config struct {
	// MaxConns is the maximum number of connections in the pool
	MaxConns int32
	// MinConns is the minimum number of connections in the pool
	MinConns int32
	// MaxConnLifetime is the maximum lifetime of a connection
	MaxConnLifetime time.Duration
	// MaxConnIdleTime is the maximum idle time for a connection
	MaxConnIdleTime time.Duration
	// HealthCheckPeriod is the interval between health checks
	HealthCheckPeriod time.Duration
	// ConnectTimeout is the timeout for establishing new connections
	ConnectTimeout time.Duration
}

// DefaultConfig returns optimized pool configuration for production
func DefaultConfig() *Config {
	return &Config{
		MaxConns:          50,               // Increased for high-performance
		MinConns:          10,               // Keep warm connections ready
		MaxConnLifetime:   30 * time.Minute, // Rotate connections every 30 minutes
		MaxConnIdleTime:   5 * time.Minute,  // Release idle connections after 5 minutes
		HealthCheckPeriod: 30 * time.Second, // Health check every 30 seconds
		ConnectTimeout:    10 * time.Second, // 10 second timeout for new connections
	}
}

// AzureConfig returns optimized pool configuration for Azure PostgreSQL
func AzureConfig() *Config {
	return &Config{
		MaxConns:          25, // Conservative for Azure limits
		MinConns:          5,  // Keep some warm connections
		MaxConnLifetime:   30 * time.Minute,
		MaxConnIdleTime:   10 * time.Minute,
		HealthCheckPeriod: 60 * time.Second,
		ConnectTimeout:    15 * time.Second, // Longer timeout for cloud DB
	}
}

// New creates a new database connection pool with optimized settings
func New(ctx context.Context, databaseURL string, cfg *Config) (*pgxpool.Pool, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Parse the connection string
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Apply optimized settings
	config.MaxConns = cfg.MaxConns
	config.MinConns = cfg.MinConns
	config.MaxConnLifetime = cfg.MaxConnLifetime
	config.MaxConnIdleTime = cfg.MaxConnIdleTime
	config.HealthCheckPeriod = cfg.HealthCheckPeriod
	config.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	// Additional performance optimizations
	config.ConnConfig.RuntimeParams = map[string]string{
		// Optimize for bulk operations
		"work_mem":                        "256MB",
		"maintenance_work_mem":            "512MB",
		"effective_cache_size":            "4GB",
		"random_page_cost":                "1.1", // SSD optimized
		"effective_io_concurrency":        "200", // For parallel queries
		"max_parallel_workers_per_gather": "4",
		"max_parallel_workers":            "8",
		// Statement timeout for safety
		"statement_timeout":                   "300000", // 5 minutes
		"idle_in_transaction_session_timeout": "60000",  // 1 minute
	}

	// Create the pool
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// Stats returns current pool statistics for monitoring
type Stats struct {
	AcquireCount         int64
	AcquiredConns        int32
	CanceledAcquireCount int64
	EmptyAcquireCount    int64
	IdleConns            int32
	MaxConns             int32
	TotalConns           int32
}

// GetStats returns current pool statistics
func GetStats(pool *pgxpool.Pool) Stats {
	stats := pool.Stat()
	return Stats{
		AcquireCount:         stats.AcquireCount(),
		AcquiredConns:        stats.AcquiredConns(),
		CanceledAcquireCount: stats.CanceledAcquireCount(),
		EmptyAcquireCount:    stats.EmptyAcquireCount(),
		IdleConns:            stats.IdleConns(),
		MaxConns:             stats.MaxConns(),
		TotalConns:           stats.TotalConns(),
	}
}
