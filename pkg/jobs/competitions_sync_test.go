package jobs

import (
	"context"
	"errors"
	"testing"
)

type mockCompetitionService struct {
	syncFunc func(ctx context.Context) error
}

func (m *mockCompetitionService) SyncCompetitions(ctx context.Context) error {
	if m.syncFunc != nil {
		return m.syncFunc(ctx)
	}
	return nil
}

func TestCompetitionsSyncJob_Name(t *testing.T) {
	service := &mockCompetitionService{}
	job := NewCompetitionsSyncJob(service)

	expectedName := "Competitions Sync"
	if job.Name() != expectedName {
		t.Errorf("Expected name '%s', got '%s'", expectedName, job.Name())
	}
}

func TestCompetitionsSyncJob_Schedule(t *testing.T) {
	service := &mockCompetitionService{}
	job := NewCompetitionsSyncJob(service)

	expectedSchedule := "0 */6 * * *"
	if job.Schedule() != expectedSchedule {
		t.Errorf("Expected schedule '%s', got '%s'", expectedSchedule, job.Schedule())
	}
}

func TestCompetitionsSyncJob_Execute_Success(t *testing.T) {
	executed := false
	service := &mockCompetitionService{
		syncFunc: func(ctx context.Context) error {
			executed = true
			return nil
		},
	}

	job := NewCompetitionsSyncJob(service)

	err := job.Execute(context.Background())

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !executed {
		t.Error("Service sync method was not called")
	}
}

func TestCompetitionsSyncJob_Execute_Error(t *testing.T) {
	expectedError := errors.New("sync failed")
	service := &mockCompetitionService{
		syncFunc: func(ctx context.Context) error {
			return expectedError
		},
	}

	job := NewCompetitionsSyncJob(service)

	err := job.Execute(context.Background())

	if err == nil {
		t.Error("Expected error, but got none")
	}

	if err != expectedError {
		t.Errorf("Expected error '%v', got '%v'", expectedError, err)
	}
}

func TestCompetitionsSyncJob_Execute_ContextCancellation(t *testing.T) {
	service := &mockCompetitionService{
		syncFunc: func(ctx context.Context) error {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		},
	}

	job := NewCompetitionsSyncJob(service)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := job.Execute(ctx)

	if err == nil {
		t.Error("Expected context cancellation error, but got none")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}
