package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
)

// ProductionJobManager extends the regular job manager with production features
type ProductionJobManager struct {
	cron        *cron.Cron
	jobs        []Job
	logger      *logger.Logger
	lockManager JobLockManager

	// Production features
	enableLocking bool
	defaultConfig *ProductionJobConfig
}

// ProductionJobManagerConfig holds configuration for the production job manager
type ProductionJobManagerConfig struct {
	EnableLocking bool                 // Enable distributed locking for all jobs
	DefaultConfig *ProductionJobConfig // Default configuration for wrapped jobs
}

// NewProductionJobManager creates a production-ready job manager with distributed locking
func NewProductionJobManager(db generated.DBTX, config *ProductionJobManagerConfig) JobManager {
	if config == nil {
		config = &ProductionJobManagerConfig{
			EnableLocking: true,
			DefaultConfig: DefaultProductionJobConfig(),
		}
	}

	lockManager := NewPostgreSQLLockManager(db)

	return &ProductionJobManager{
		cron:          cron.New(cron.WithLocation(time.UTC)),
		jobs:          make([]Job, 0),
		logger:        logger.New("production-job-manager"),
		lockManager:   lockManager,
		enableLocking: config.EnableLocking,
		defaultConfig: config.DefaultConfig,
	}
}

// RegisterJob adds a job to the manager, automatically wrapping it with production features
func (m *ProductionJobManager) RegisterJob(job Job) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	// Wrap job with production features if locking is enabled
	finalJob := job
	if m.enableLocking {
		// Check if job is already a production job
		if _, isProduction := job.(*ProductionJob); !isProduction {
			finalJob = NewProductionJob(job, m.lockManager, m.defaultConfig)
			m.logger.Info().
				Str("job_name", job.Name()).
				Str("action", "wrap_with_locking").
				Msg("Wrapping job with distributed locking")
		}
	}

	m.logger.Info().
		Str("action", "register_job").
		Str("job_name", finalJob.Name()).
		Str("schedule", finalJob.Schedule()).
		Bool("locking_enabled", m.enableLocking).
		Msg("Registering production job")

	_, err := m.cron.AddFunc(finalJob.Schedule(), func() {
		// Create unique request ID for job execution
		requestID := uuid.New().String()
		jobLogger := m.logger.WithRequestID(requestID).WithJob(finalJob.Name())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		// Add logger to context
		ctx = jobLogger.ToContext(ctx)

		jobLogger.LogJobStart(finalJob.Name(), finalJob.Schedule())
		start := time.Now()

		if err := finalJob.Execute(ctx); err != nil {
			jobLogger.Error().
				Err(err).
				Str("action", "job_failed").
				Dur("duration", time.Since(start)).
				Msg("Production job execution failed")
		} else {
			duration := time.Since(start)
			jobLogger.LogJobComplete(finalJob.Name(), duration, 0, 0)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to schedule job %s: %w", finalJob.Name(), err)
	}

	m.jobs = append(m.jobs, finalJob)
	return nil
}

// RegisterJobWithConfig registers a job with custom production configuration
func (m *ProductionJobManager) RegisterJobWithConfig(job Job, config *ProductionJobConfig) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	// Always wrap with the specified configuration
	productionJob := NewProductionJob(job, m.lockManager, config)

	m.logger.Info().
		Str("action", "register_job_with_config").
		Str("job_name", job.Name()).
		Str("schedule", job.Schedule()).
		Dur("lock_timeout", config.LockTimeout).
		Bool("skip_if_locked", config.SkipIfLocked).
		Bool("retry_on_error", config.RetryOnError).
		Int("max_retries", config.MaxRetries).
		Msg("Registering job with custom production configuration")

	// Temporarily disable auto-wrapping for this registration
	origLocking := m.enableLocking
	m.enableLocking = false
	err := m.RegisterJob(productionJob)
	m.enableLocking = origLocking

	return err
}

// RegisterJobWithoutLocking registers a job without distributed locking
func (m *ProductionJobManager) RegisterJobWithoutLocking(job Job) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	m.logger.Info().
		Str("action", "register_job_without_locking").
		Str("job_name", job.Name()).
		Str("schedule", job.Schedule()).
		Msg("Registering job without distributed locking")

	// Temporarily disable auto-wrapping for this registration
	origLocking := m.enableLocking
	m.enableLocking = false
	err := m.RegisterJob(job)
	m.enableLocking = origLocking

	return err
}

// Start begins executing all registered jobs according to their schedules
func (m *ProductionJobManager) Start() {
	m.logger.Info().
		Str("action", "start").
		Int("job_count", len(m.jobs)).
		Bool("locking_enabled", m.enableLocking).
		Msg("Starting production job manager")

	// Run startup jobs immediately for better initial data sync
	m.runStartupJobs()

	m.cron.Start()
}

// runStartupJobs executes critical jobs once on startup for immediate data sync
func (m *ProductionJobManager) runStartupJobs() {
	startupJobs := []string{
		"config_sync",
		"sports_sync",
	}

	for _, job := range m.jobs {
		jobName := job.Name()

		// Check if this is a startup job
		isStartupJob := false
		for _, startupJobName := range startupJobs {
			if jobName == startupJobName {
				isStartupJob = true
				break
			}
		}

		if !isStartupJob {
			continue
		}

		m.logger.Info().
			Str("job_name", jobName).
			Str("action", "startup_job_start").
			Msg("Running job on startup")

		// Create context for startup job execution
		requestID := uuid.New().String()
		jobLogger := m.logger.WithRequestID(requestID).WithJob(jobName)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		ctx = jobLogger.ToContext(ctx)

		start := time.Now()
		if err := job.Execute(ctx); err != nil {
			jobLogger.Error().
				Err(err).
				Str("action", "startup_job_failed").
				Dur("duration", time.Since(start)).
				Msg("Startup job execution failed")
		} else {
			duration := time.Since(start)
			jobLogger.LogJobComplete(jobName, duration, 0, 0)
		}

		cancel()
	}
}

// Stop gracefully shuts down the job manager
func (m *ProductionJobManager) Stop() {
	m.logger.Info().
		Str("action", "stop_initiated").
		Msg("Stopping production job manager")

	// Stop scheduling new jobs
	ctx := m.cron.Stop()

	// Wait for running jobs to complete
	<-ctx.Done()

	m.logger.Info().
		Str("action", "stopped").
		Msg("Production job manager stopped")
}

// GetJobs returns all registered jobs
func (m *ProductionJobManager) GetJobs() []Job {
	return m.jobs
}

// GetLockManager returns the distributed lock manager
func (m *ProductionJobManager) GetLockManager() JobLockManager {
	return m.lockManager
}

// IsJobLocked checks if a specific job is currently locked
func (m *ProductionJobManager) IsJobLocked(ctx context.Context, jobName string) (bool, error) {
	return m.lockManager.IsLocked(ctx, jobName)
}

// GetJobStatus returns status information for all jobs
func (m *ProductionJobManager) GetJobStatus(ctx context.Context) (map[string]JobStatus, error) {
	status := make(map[string]JobStatus)

	for _, job := range m.jobs {
		jobName := job.Name()
		isLocked, err := m.lockManager.IsLocked(ctx, jobName)
		if err != nil {
			return nil, fmt.Errorf("failed to check lock status for job %s: %w", jobName, err)
		}

		status[jobName] = JobStatus{
			Name:     jobName,
			Schedule: job.Schedule(),
			IsLocked: isLocked,
			// TODO: Add more status fields (last run time, success/failure counts, etc.)
		}
	}

	return status, nil
}

// JobStatus represents the current status of a job
type JobStatus struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	IsLocked bool   `json:"is_locked"`
	// Future: LastRunTime, SuccessCount, FailureCount, etc.
}
