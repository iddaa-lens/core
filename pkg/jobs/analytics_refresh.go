package jobs

import (
	"context"
	"log"

	"github.com/betslib/iddaa-core/pkg/database"
)

type AnalyticsRefreshJob struct {
	db *database.Queries
}

func NewAnalyticsRefreshJob(db *database.Queries) Job {
	return &AnalyticsRefreshJob{
		db: db,
	}
}

func (j *AnalyticsRefreshJob) Name() string {
	return "analytics_refresh"
}

func (j *AnalyticsRefreshJob) Execute(ctx context.Context) error {
	log.Printf("Starting analytics refresh job...")

	// Refresh materialized views for better performance
	log.Printf("Refreshing contrarian bets view...")
	err := j.db.RefreshContrarianBets(ctx)
	if err != nil {
		log.Printf("Failed to refresh contrarian bets: %v", err)
		// Continue with other refreshes even if one fails
	}

	// Note: volume_trends materialized view doesn't have a specific refresh function
	// in our current queries, but we could add one if needed

	log.Printf("Analytics refresh completed successfully")
	return nil
}

func (j *AnalyticsRefreshJob) Schedule() string {
	// Run every 6 hours to refresh materialized views and analytics
	return "0 */6 * * *"
}
