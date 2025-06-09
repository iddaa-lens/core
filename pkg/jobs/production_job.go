package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/iddaa-lens/core/pkg/logger"
)

// ProductionJob wraps a regular job with production-ready features
type ProductionJob struct {
	job         Job
	lockManager JobLockManager
	logger      *logger.Logger

	// Configuration
	lockTimeout  time.Duration
	skipIfLocked bool
	retryOnError bool
	maxRetries   int
}

// ProductionJobConfig holds configuration for production job wrapper
type ProductionJobConfig struct {
	LockTimeout  time.Duration // How long to wait for lock acquisition
	SkipIfLocked bool          // Skip execution if lock can't be acquired
	RetryOnError bool          // Retry job execution on failure
	MaxRetries   int           // Maximum retry attempts
}

// DefaultProductionJobConfig returns sensible defaults for production jobs
func DefaultProductionJobConfig() *ProductionJobConfig {
	return &ProductionJobConfig{
		LockTimeout:  30 * time.Second, // Wait up to 30 seconds for lock
		SkipIfLocked: true,             // Skip if another instance is running
		RetryOnError: false,            // Don't retry by default (cron will reschedule)
		MaxRetries:   0,                // No retries by default
	}
}

// NewProductionJob creates a production-ready job wrapper
func NewProductionJob(job Job, lockManager JobLockManager, config *ProductionJobConfig) *ProductionJob {
	if config == nil {
		config = DefaultProductionJobConfig()
	}

	return &ProductionJob{
		job:          job,
		lockManager:  lockManager,
		logger:       logger.New("production-job"),
		lockTimeout:  config.LockTimeout,
		skipIfLocked: config.SkipIfLocked,
		retryOnError: config.RetryOnError,
		maxRetries:   config.MaxRetries,
	}
}

// Name returns the underlying job name
func (p *ProductionJob) Name() string {
	return p.job.Name()
}

// Schedule returns the underlying job schedule
func (p *ProductionJob) Schedule() string {
	return p.job.Schedule()
}

// Execute runs the job with distributed locking and error handling
func (p *ProductionJob) Execute(ctx context.Context) error {
	jobName := p.job.Name()
	startTime := time.Now()

	// Create lock guard for automatic cleanup
	lockGuard := NewLockGuard(p.lockManager, jobName)

	p.logger.Info().
		Str("job_name", jobName).
		Str("action", "job_start").
		Dur("lock_timeout", p.lockTimeout).
		Bool("skip_if_locked", p.skipIfLocked).
		Msg("Starting production job execution")

	// Attempt to acquire distributed lock
	var acquired bool
	var err error

	if p.lockTimeout > 0 {
		acquired, err = lockGuard.AcquireWithTimeout(ctx, p.lockTimeout)
	} else {
		acquired, err = lockGuard.Acquire(ctx)
	}

	if err != nil {
		p.logger.Error().
			Err(err).
			Str("job_name", jobName).
			Str("action", "lock_acquisition_error").
			Msg("Failed to acquire distributed lock")
		return fmt.Errorf("failed to acquire lock for job %s: %w", jobName, err)
	}

	if !acquired {
		if p.skipIfLocked {
			p.logger.Info().
				Str("job_name", jobName).
				Str("action", "job_skipped_locked").
				Msg("Job skipped - another instance is running")
			return nil // Not an error - another instance is handling it
		} else {
			return fmt.Errorf("could not acquire lock for job %s within timeout", jobName)
		}
	}

	// Ensure lock is released when we exit
	defer func() {
		if releaseErr := lockGuard.Release(ctx); releaseErr != nil {
			p.logger.Error().
				Err(releaseErr).
				Str("job_name", jobName).
				Str("action", "lock_release_error").
				Msg("Failed to release distributed lock")
		}
	}()

	p.logger.Info().
		Str("job_name", jobName).
		Str("action", "lock_acquired").
		Msg("Successfully acquired distributed lock, executing job")

	// Execute the job with retry logic if configured
	err = p.executeWithRetry(ctx)

	duration := time.Since(startTime)

	if err != nil {
		p.logger.Error().
			Err(err).
			Str("job_name", jobName).
			Str("action", "job_failed").
			Dur("duration", duration).
			Msg("Production job execution failed")
		return err
	}

	p.logger.Info().
		Str("job_name", jobName).
		Str("action", "job_completed").
		Dur("duration", duration).
		Msg("Production job execution completed successfully")

	return nil
}

// executeWithRetry executes the job with retry logic if configured
func (p *ProductionJob) executeWithRetry(ctx context.Context) error {
	var lastErr error
	maxAttempts := p.maxRetries + 1 // maxRetries + initial attempt

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			p.logger.Warn().
				Int("attempt", attempt).
				Int("max_attempts", maxAttempts).
				Err(lastErr).
				Str("job_name", p.job.Name()).
				Str("action", "job_retry").
				Msg("Retrying job execution after failure")

			// Add exponential backoff between retries
			backoffDuration := time.Duration(1<<uint(attempt-2)) * time.Second
			select {
			case <-time.After(backoffDuration):
				// Continue with retry
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Execute the actual job
		err := p.job.Execute(ctx)
		if err == nil {
			if attempt > 1 {
				p.logger.Info().
					Int("attempt", attempt).
					Str("job_name", p.job.Name()).
					Str("action", "job_retry_success").
					Msg("Job succeeded after retry")
			}
			return nil // Success!
		}

		lastErr = err

		if !p.retryOnError {
			break // Don't retry if retries are disabled
		}

		// Check if we should retry this type of error
		if !p.shouldRetryError(err) {
			p.logger.Warn().
				Err(err).
				Str("job_name", p.job.Name()).
				Str("action", "error_not_retryable").
				Msg("Error is not retryable, failing immediately")
			break
		}
	}

	return lastErr
}

// shouldRetryError determines if an error should trigger a retry
func (p *ProductionJob) shouldRetryError(err error) bool {
	// For now, we retry most errors except context cancellation
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	// Could add more sophisticated error classification here
	// For example, don't retry on validation errors, but retry on network errors

	return true
}

// JobWithLockTimeout creates a production job with custom lock timeout
func JobWithLockTimeout(job Job, lockManager JobLockManager, timeout time.Duration) *ProductionJob {
	config := DefaultProductionJobConfig()
	config.LockTimeout = timeout
	return NewProductionJob(job, lockManager, config)
}

// JobWithRetry creates a production job with retry logic
func JobWithRetry(job Job, lockManager JobLockManager, maxRetries int) *ProductionJob {
	config := DefaultProductionJobConfig()
	config.RetryOnError = true
	config.MaxRetries = maxRetries
	return NewProductionJob(job, lockManager, config)
}

// JobWithCustomConfig creates a production job with custom configuration
func JobWithCustomConfig(job Job, lockManager JobLockManager, config *ProductionJobConfig) *ProductionJob {
	return NewProductionJob(job, lockManager, config)
}

// MustAcquireLock creates a production job that fails if lock cannot be acquired
func MustAcquireLock(job Job, lockManager JobLockManager) *ProductionJob {
	config := DefaultProductionJobConfig()
	config.SkipIfLocked = false // Fail if lock can't be acquired
	return NewProductionJob(job, lockManager, config)
}
