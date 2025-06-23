package jobs

import (
	"context"
	"time"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
)

type AnalyticsRefreshJob struct {
	db *generated.Queries
}

func NewAnalyticsRefreshJob(db *generated.Queries) Job {
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

	// List of all materialized views to refresh
	views := []struct {
		name    string
		refresh func(context.Context) error
	}{
		{"contrarian_bets", j.db.RefreshContrarianBets},
		{"big_movers", j.db.RefreshBigMovers},
		{"sharp_money_moves", j.db.RefreshSharpMoneyMoves},
		{"live_opportunities", j.db.RefreshLiveOpportunities},
		{"value_spots", j.db.RefreshValueSpots},
		{"high_volume_events", j.db.RefreshHighVolumeEvents},
	}

	// Refresh each materialized view
	for _, view := range views {
		viewStart := time.Now()
		log.Debug().
			Str("action", "view_refresh_start").
			Str("view_name", view.name).
			Msg("Refreshing materialized view")

		err := view.refresh(ctx)
		if err != nil {
			errorCount++
			log.Error().
				Err(err).
				Str("action", "view_refresh_failed").
				Str("view_name", view.name).
				Dur("duration", time.Since(viewStart)).
				Msg("Failed to refresh materialized view")
			// Continue with other views even if one fails
		} else {
			refreshedViews++
			log.Debug().
				Str("action", "view_refresh_complete").
				Str("view_name", view.name).
				Dur("duration", time.Since(viewStart)).
				Msg("Materialized view refreshed successfully")
		}
	}

	// Log summary
	log.Info().
		Str("action", "refresh_summary").
		Int("total_views", len(views)).
		Int("refreshed_successfully", refreshedViews).
		Int("failed_refreshes", errorCount).
		Msg("Analytics refresh completed")

	duration := time.Since(start)
	log.LogJobComplete("analytics_refresh", duration, refreshedViews, errorCount)

	// Return nil even if some views failed - we logged the errors above
	// This allows the job to continue running on schedule
	return nil
}

func (j *AnalyticsRefreshJob) Schedule() string {
	// Run every 5 minutes to keep materialized views fresh
	return "*/5 * * * *"
}
