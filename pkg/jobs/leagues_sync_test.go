package jobs

import (
	"context"
	"testing"
	"time"
)

func TestLeaguesSyncJob_Name(t *testing.T) {
	job := &LeaguesSyncJob{}
	expected := "leagues_sync"
	if got := job.Name(); got != expected {
		t.Errorf("Name() = %v, want %v", got, expected)
	}
}

func TestLeaguesSyncJob_Description(t *testing.T) {
	job := &LeaguesSyncJob{}
	expected := "Syncs leagues and teams with Football API"
	if got := job.Description(); got != expected {
		t.Errorf("Description() = %v, want %v", got, expected)
	}
}

func TestLeaguesSyncJob_Schedule(t *testing.T) {
	job := &LeaguesSyncJob{}
	expected := "0 2 * * *" // Daily at 2 AM
	if got := job.Schedule(); got != expected {
		t.Errorf("Schedule() = %v, want %v", got, expected)
	}
}

func TestLeaguesSyncJob_Timeout(t *testing.T) {
	job := &LeaguesSyncJob{}
	expected := 10 * time.Minute
	if got := job.Timeout(); got != expected {
		t.Errorf("Timeout() = %v, want %v", got, expected)
	}
}

func TestLeaguesSyncJob_Execute(t *testing.T) {
	// This test requires a database connection and API key
	// For now, we'll just test that the job doesn't panic
	job := &LeaguesSyncJob{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// This should fail due to missing database and API key, but not panic
	err := job.Execute(ctx)
	if err == nil {
		t.Error("Expected error due to missing dependencies, but got nil")
	} else {
		// Check that we get a proper error (not a panic)
		if err.Error() != "leagues service is not initialized" {
			// It's OK if we get a different error (e.g., database connection error)
			// The important thing is that we don't panic
			t.Logf("Got expected error: %v", err)
		}
	}
}
