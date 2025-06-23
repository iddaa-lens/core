package jobs

import (
	"context"
	"crypto/md5"
	"fmt"
	"time"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
)

// JobLockManager provides distributed locking for job execution
type JobLockManager interface {
	// AcquireLock attempts to acquire a distributed lock for the given job
	// Returns true if lock was acquired, false if already locked by another instance
	AcquireLock(ctx context.Context, jobName string) (bool, error)

	// ReleaseLock releases the distributed lock for the given job
	ReleaseLock(ctx context.Context, jobName string) error

	// IsLocked checks if a job is currently locked
	IsLocked(ctx context.Context, jobName string) (bool, error)

	// AcquireLockWithTimeout attempts to acquire a lock with a timeout
	AcquireLockWithTimeout(ctx context.Context, jobName string, timeout time.Duration) (bool, error)
}

// PostgreSQLLockManager implements distributed locking using PostgreSQL advisory locks
type PostgreSQLLockManager struct {
	db     generated.DBTX
	logger *logger.Logger
}

// NewPostgreSQLLockManager creates a new PostgreSQL-based lock manager
func NewPostgreSQLLockManager(db generated.DBTX) JobLockManager {
	return &PostgreSQLLockManager{
		db:     db,
		logger: logger.New("job-lock-manager"),
	}
}

// generateLockID creates a consistent numeric lock ID from job name
// PostgreSQL advisory locks require int64 keys
func (p *PostgreSQLLockManager) generateLockID(jobName string) int64 {
	// Use MD5 hash to create consistent lock ID from job name
	hash := md5.Sum([]byte(jobName))

	// Convert first 8 bytes of hash to int64
	lockID := int64(0)
	for i := 0; i < 8; i++ {
		lockID = lockID<<8 + int64(hash[i])
	}

	// Ensure positive value (PostgreSQL advisory locks work with any int64)
	if lockID < 0 {
		lockID = -lockID
	}

	return lockID
}

// AcquireLock attempts to acquire a distributed lock for the given job
func (p *PostgreSQLLockManager) AcquireLock(ctx context.Context, jobName string) (bool, error) {
	lockID := p.generateLockID(jobName)

	p.logger.Debug().
		Str("job_name", jobName).
		Int64("lock_id", lockID).
		Str("action", "acquire_lock_attempt").
		Msg("Attempting to acquire distributed lock")

	// Use pg_try_advisory_lock which returns immediately
	// Returns true if lock acquired, false if already locked
	query := "SELECT pg_try_advisory_lock($1)"

	var acquired bool
	err := p.db.QueryRow(ctx, query, lockID).Scan(&acquired)
	if err != nil {
		p.logger.Error().
			Err(err).
			Str("job_name", jobName).
			Int64("lock_id", lockID).
			Str("action", "acquire_lock_failed").
			Msg("Failed to acquire distributed lock")
		return false, fmt.Errorf("failed to acquire lock for job %s: %w", jobName, err)
	}

	if acquired {
		p.logger.Info().
			Str("job_name", jobName).
			Int64("lock_id", lockID).
			Str("action", "lock_acquired").
			Msg("Successfully acquired distributed lock")
	} else {
		p.logger.Debug().
			Str("job_name", jobName).
			Int64("lock_id", lockID).
			Str("action", "lock_already_held").
			Msg("Lock already held by another instance")
	}

	return acquired, nil
}

// ReleaseLock releases the distributed lock for the given job
func (p *PostgreSQLLockManager) ReleaseLock(ctx context.Context, jobName string) error {
	lockID := p.generateLockID(jobName)

	p.logger.Debug().
		Str("job_name", jobName).
		Int64("lock_id", lockID).
		Str("action", "release_lock_attempt").
		Msg("Attempting to release distributed lock")

	// Use pg_advisory_unlock to release the lock
	// Returns true if lock was held and released, false if not held
	query := "SELECT pg_advisory_unlock($1)"

	var released bool
	err := p.db.QueryRow(ctx, query, lockID).Scan(&released)
	if err != nil {
		p.logger.Error().
			Err(err).
			Str("job_name", jobName).
			Int64("lock_id", lockID).
			Str("action", "release_lock_failed").
			Msg("Failed to release distributed lock")
		return fmt.Errorf("failed to release lock for job %s: %w", jobName, err)
	}

	if released {
		p.logger.Info().
			Str("job_name", jobName).
			Int64("lock_id", lockID).
			Str("action", "lock_released").
			Msg("Successfully released distributed lock")
	} else {
		p.logger.Warn().
			Str("job_name", jobName).
			Int64("lock_id", lockID).
			Str("action", "lock_not_held").
			Msg("Attempted to release lock that was not held")
	}

	return nil
}

// IsLocked checks if a job is currently locked
func (p *PostgreSQLLockManager) IsLocked(ctx context.Context, jobName string) (bool, error) {
	lockID := p.generateLockID(jobName)

	// Check if the advisory lock is currently held
	// We can try to acquire and immediately release if successful
	query := "SELECT pg_try_advisory_lock($1)"

	var canAcquire bool
	err := p.db.QueryRow(ctx, query, lockID).Scan(&canAcquire)
	if err != nil {
		return false, fmt.Errorf("failed to check lock status for job %s: %w", jobName, err)
	}

	if canAcquire {
		// We acquired it, so it wasn't locked - release immediately
		releaseQuery := "SELECT pg_advisory_unlock($1)"
		_, err = p.db.Exec(ctx, releaseQuery, lockID)
		if err != nil {
			p.logger.Warn().
				Err(err).
				Str("job_name", jobName).
				Msg("Failed to release lock after check")
		}
		return false, nil // Was not locked
	}

	return true, nil // Was locked (couldn't acquire)
}

// AcquireLockWithTimeout attempts to acquire a lock with polling and timeout
func (p *PostgreSQLLockManager) AcquireLockWithTimeout(ctx context.Context, jobName string, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond) // Poll every 100ms
	defer ticker.Stop()

	// Try to acquire immediately first
	acquired, err := p.AcquireLock(ctx, jobName)
	if err != nil {
		return false, err
	}
	if acquired {
		return true, nil
	}

	p.logger.Debug().
		Str("job_name", jobName).
		Dur("timeout", timeout).
		Str("action", "lock_wait_start").
		Msg("Lock not available, waiting with timeout")

	// Poll until timeout or acquisition
	for {
		select {
		case <-ctx.Done():
			p.logger.Debug().
				Str("job_name", jobName).
				Dur("timeout", timeout).
				Str("action", "lock_wait_timeout").
				Msg("Lock acquisition timed out")
			return false, ctx.Err()
		case <-ticker.C:
			acquired, err := p.AcquireLock(ctx, jobName)
			if err != nil {
				return false, err
			}
			if acquired {
				p.logger.Debug().
					Str("job_name", jobName).
					Str("action", "lock_acquired_after_wait").
					Msg("Successfully acquired lock after waiting")
				return true, nil
			}
		}
	}
}

// LockGuard provides RAII-style lock management
type LockGuard struct {
	lockManager JobLockManager
	jobName     string
	acquired    bool
	logger      *logger.Logger
}

// NewLockGuard creates a new lock guard that automatically releases on defer
func NewLockGuard(lockManager JobLockManager, jobName string) *LockGuard {
	return &LockGuard{
		lockManager: lockManager,
		jobName:     jobName,
		acquired:    false,
		logger:      logger.New("lock-guard"),
	}
}

// Acquire attempts to acquire the lock
func (lg *LockGuard) Acquire(ctx context.Context) (bool, error) {
	acquired, err := lg.lockManager.AcquireLock(ctx, lg.jobName)
	if err != nil {
		return false, err
	}
	lg.acquired = acquired
	return acquired, nil
}

// AcquireWithTimeout attempts to acquire the lock with timeout
func (lg *LockGuard) AcquireWithTimeout(ctx context.Context, timeout time.Duration) (bool, error) {
	acquired, err := lg.lockManager.AcquireLockWithTimeout(ctx, lg.jobName, timeout)
	if err != nil {
		return false, err
	}
	lg.acquired = acquired
	return acquired, nil
}

// Release releases the lock if it was acquired
func (lg *LockGuard) Release(ctx context.Context) error {
	if !lg.acquired {
		return nil
	}

	err := lg.lockManager.ReleaseLock(ctx, lg.jobName)
	if err != nil {
		lg.logger.Error().
			Err(err).
			Str("job_name", lg.jobName).
			Msg("Failed to release lock in guard")
		return err
	}

	lg.acquired = false
	return nil
}

// IsAcquired returns whether the lock is currently held by this guard
func (lg *LockGuard) IsAcquired() bool {
	return lg.acquired
}
