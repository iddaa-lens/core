package jobs

import "context"

// Job represents a schedulable job that can be executed by the cron service
type Job interface {
	// Execute runs the job with the given context
	Execute(ctx context.Context) error

	// Name returns a human-readable name for the job
	Name() string

	// Schedule returns the cron schedule expression for this job
	// Format: "minute hour day month weekday" or "@every duration"
	// Examples: "0 */6 * * *" (every 6 hours), "@every 1h" (every hour)
	Schedule() string
}

// JobManager manages and schedules multiple jobs
type JobManager interface {
	// RegisterJob adds a job to the manager
	RegisterJob(job Job) error

	// Start begins executing all registered jobs according to their schedules
	Start()

	// Stop gracefully shuts down the job manager
	Stop()

	// GetJobs returns all registered jobs
	GetJobs() []Job
}
