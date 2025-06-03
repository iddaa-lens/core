package services

import "context"

// CompetitionSyncer defines the interface for competition synchronization
type CompetitionSyncer interface {
	SyncCompetitions(ctx context.Context) error
}
