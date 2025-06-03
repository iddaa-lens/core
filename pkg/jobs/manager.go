package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

type cronJobManager struct {
	cron *cron.Cron
	jobs []Job
}

// NewJobManager creates a new job manager
func NewJobManager() JobManager {
	return &cronJobManager{
		cron: cron.New(cron.WithLocation(time.UTC)),
		jobs: make([]Job, 0),
	}
}

func (m *cronJobManager) RegisterJob(job Job) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	log.Printf("Registering job: %s with schedule: %s", job.Name(), job.Schedule())

	_, err := m.cron.AddFunc(job.Schedule(), func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		log.Printf("Starting job: %s", job.Name())
		start := time.Now()

		if err := job.Execute(ctx); err != nil {
			log.Printf("Job %s failed: %v", job.Name(), err)
		} else {
			duration := time.Since(start)
			log.Printf("Job %s completed successfully in %v", job.Name(), duration)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to schedule job %s: %w", job.Name(), err)
	}

	m.jobs = append(m.jobs, job)
	return nil
}

func (m *cronJobManager) Start() {
	log.Printf("Starting job manager with %d registered jobs", len(m.jobs))
	m.cron.Start()
}

func (m *cronJobManager) Stop() {
	log.Println("Stopping job manager...")
	ctx := m.cron.Stop()
	<-ctx.Done()
	log.Println("Job manager stopped")
}

func (m *cronJobManager) GetJobs() []Job {
	return append([]Job(nil), m.jobs...)
}
