package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type cronJobManager struct {
	cron   *cron.Cron
	jobs   []Job
	logger *logger.Logger
}

// NewJobManager creates a new job manager
func NewJobManager() JobManager {
	return &cronJobManager{
		cron:   cron.New(cron.WithLocation(time.UTC)),
		jobs:   make([]Job, 0),
		logger: logger.New("job-manager"),
	}
}

func (m *cronJobManager) RegisterJob(job Job) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	m.logger.Info().
		Str("action", "register_job").
		Str("job_name", job.Name()).
		Str("schedule", job.Schedule()).
		Msg("Registering job")

	_, err := m.cron.AddFunc(job.Schedule(), func() {
		// Create unique request ID for job execution
		requestID := uuid.New().String()
		jobLogger := m.logger.WithRequestID(requestID).WithJob(job.Name())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		// Add logger to context
		ctx = jobLogger.ToContext(ctx)

		jobLogger.LogJobStart(job.Name(), job.Schedule())
		start := time.Now()

		if err := job.Execute(ctx); err != nil {
			jobLogger.Error().
				Err(err).
				Str("action", "job_failed").
				Dur("duration", time.Since(start)).
				Msg("Job execution failed")
		} else {
			duration := time.Since(start)
			jobLogger.LogJobComplete(job.Name(), duration, 0, 0) // TODO: Pass actual metrics
		}
	})

	if err != nil {
		return fmt.Errorf("failed to schedule job %s: %w", job.Name(), err)
	}

	m.jobs = append(m.jobs, job)
	return nil
}

func (m *cronJobManager) Start() {
	m.logger.Info().
		Str("action", "start").
		Int("job_count", len(m.jobs)).
		Msg("Starting job manager")

	// Run all jobs once on startup
	for _, job := range m.jobs {
		go func(j Job) {
			// Create unique request ID for startup job execution
			requestID := uuid.New().String()
			jobLogger := m.logger.WithRequestID(requestID).WithJob(j.Name())

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()

			// Add logger to context
			ctx = jobLogger.ToContext(ctx)

			jobLogger.Info().
				Str("action", "startup_job_start").
				Str("job_name", j.Name()).
				Msg("Running job on startup")

			start := time.Now()

			if err := j.Execute(ctx); err != nil {
				jobLogger.Error().
					Err(err).
					Str("action", "startup_job_failed").
					Dur("duration", time.Since(start)).
					Msg("Startup job execution failed")
			} else {
				duration := time.Since(start)
				jobLogger.Info().
					Str("action", "startup_job_complete").
					Str("job_name", j.Name()).
					Dur("duration", duration).
					Msg("Startup job completed successfully")
			}
		}(job)
	}

	m.cron.Start()
}

func (m *cronJobManager) Stop() {
	m.logger.Info().
		Str("action", "stop_initiated").
		Msg("Stopping job manager")

	ctx := m.cron.Stop()
	<-ctx.Done()

	m.logger.Info().
		Str("action", "stopped").
		Msg("Job manager stopped")
}

func (m *cronJobManager) GetJobs() []Job {
	return append([]Job(nil), m.jobs...)
}
