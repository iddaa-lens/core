package jobs

import (
	"context"
	"time"

	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/logger"
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
	log := logger.WithContext(ctx, "analytics-refresh")
	start := time.Now()

	log.Info().
		Str("action", "refresh_start").
		Msg("Starting analytics refresh job")

	errorCount := 0
	refreshedViews := 0

	// Refresh materialized views for better performance
	viewStart := time.Now()
	log.Debug().
		Str("action", "view_refresh_start").
		Str("view_name", "contrarian_bets").
		Msg("Refreshing materialized view")

	err := j.db.RefreshContrarianBets(ctx)
	if err != nil {
		errorCount++
		log.Error().
			Err(err).
			Str("action", "view_refresh_failed").
			Str("view_name", "contrarian_bets").
			Dur("duration", time.Since(viewStart)).
			Msg("Failed to refresh contrarian bets view")
		// Continue with other refreshes even if one fails
	} else {
		refreshedViews++
		log.Debug().
			Str("action", "view_refresh_complete").
			Str("view_name", "contrarian_bets").
			Dur("duration", time.Since(viewStart)).
			Msg("Contrarian bets view refreshed")
	}

	// Note: volume_trends materialized view doesn't have a specific refresh function
	// in our current queries, but we could add one if needed

	duration := time.Since(start)
	log.LogJobComplete("analytics_refresh", duration, refreshedViews, errorCount)
	return nil
}

func (j *AnalyticsRefreshJob) Schedule() string {
	// Run every 6 hours to refresh materialized views and analytics
	return "0 */6 * * *"
}
