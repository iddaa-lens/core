package jobs

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockJob struct {
	name        string
	schedule    string
	executeFunc func(ctx context.Context) error
	executed    bool
}

func (m *mockJob) Execute(ctx context.Context) error {
	m.executed = true
	if m.executeFunc != nil {
		return m.executeFunc(ctx)
	}
	return nil
}

func (m *mockJob) Name() string {
	return m.name
}

func (m *mockJob) Schedule() string {
	return m.schedule
}

func TestJobManager_RegisterJob(t *testing.T) {
	manager := NewJobManager()

	tests := []struct {
		name    string
		job     Job
		wantErr bool
	}{
		{
			name: "valid job",
			job: &mockJob{
				name:     "test-job",
				schedule: "@every 1s",
			},
			wantErr: false,
		},
		{
			name:    "nil job",
			job:     nil,
			wantErr: true,
		},
		{
			name: "invalid schedule",
			job: &mockJob{
				name:     "invalid-job",
				schedule: "invalid-cron",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.RegisterJob(tt.job)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobManager_GetJobs(t *testing.T) {
	manager := NewJobManager()

	// Initially should have no jobs
	jobs := manager.GetJobs()
	if len(jobs) != 0 {
		t.Errorf("Expected 0 jobs initially, got %d", len(jobs))
	}

	// Add a job
	testJob := &mockJob{
		name:     "test-job",
		schedule: "@every 1s",
	}

	err := manager.RegisterJob(testJob)
	if err != nil {
		t.Fatalf("Failed to register job: %v", err)
	}

	// Should now have 1 job
	jobs = manager.GetJobs()
	if len(jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(jobs))
	}

	if jobs[0].Name() != "test-job" {
		t.Errorf("Expected job name 'test-job', got '%s'", jobs[0].Name())
	}
}

func TestJobManager_StartStop(t *testing.T) {
	manager := NewJobManager()

	// Test starting and stopping without jobs
	manager.Start()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Stop should complete without hanging
	done := make(chan bool, 1)
	go func() {
		manager.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() took too long")
	}
}

func TestJobExecution(t *testing.T) {
	manager := NewJobManager()

	testJob := &mockJob{
		name:     "test-execution",
		schedule: "@every 100ms",
		executeFunc: func(ctx context.Context) error {
			return nil
		},
	}

	err := manager.RegisterJob(testJob)
	if err != nil {
		t.Fatalf("Failed to register job: %v", err)
	}

	manager.Start()
	defer manager.Stop()

	// Wait for job to execute
	time.Sleep(200 * time.Millisecond)

	if !testJob.executed {
		t.Error("Job was not executed")
	}
}

func TestJobExecutionError(t *testing.T) {
	manager := NewJobManager()

	testError := errors.New("test error")
	testJob := &mockJob{
		name:     "test-error",
		schedule: "@every 100ms",
		executeFunc: func(ctx context.Context) error {
			return testError
		},
	}

	err := manager.RegisterJob(testJob)
	if err != nil {
		t.Fatalf("Failed to register job: %v", err)
	}

	manager.Start()
	defer manager.Stop()

	// Wait for job to execute
	time.Sleep(200 * time.Millisecond)

	// Job should still be executed despite error (error should be logged but not break execution)
	if !testJob.executed {
		t.Error("Job was not executed even though it should run despite errors")
	}
}
