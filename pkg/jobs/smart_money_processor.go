package jobs

import (
	"context"
	"time"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/services"
	"github.com/jackc/pgx/v5/pgtype"
)

// SmartMoneyProcessorJob processes odds movements for smart money alerts
type SmartMoneyProcessorJob struct {
	queries *generated.Queries
	tracker *services.SmartMoneyTracker
}

// NewSmartMoneyProcessorJob creates a new smart money processor job
func NewSmartMoneyProcessorJob(queries *generated.Queries, tracker *services.SmartMoneyTracker) *SmartMoneyProcessorJob {
	return &SmartMoneyProcessorJob{
		queries: queries,
		tracker: tracker,
	}
}

// Name returns the job name for CLI execution
func (j *SmartMoneyProcessorJob) Name() string {
	return "smart_money_processor"
}

// Schedule returns the cron schedule - every minute for quick alert generation
func (j *SmartMoneyProcessorJob) Schedule() string {
	return "* * * * *" // Every minute
}

// Execute runs the smart money processing
func (j *SmartMoneyProcessorJob) Execute(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Second) // 50 seconds to avoid overlap
	defer cancel()
	ctx = timeoutCtx

	log := logger.WithContext(ctx, "smart-money-processor")
	start := time.Now()

	log.Info().
		Str("action", "processor_start").
		Msg("Starting smart money processor job")

	// Get recent odds history records (last 2 minutes to catch new movements)
	since := time.Now().Add(-2 * time.Minute)

	recentMovements, err := j.queries.GetRecentOddsHistory(ctx, generated.GetRecentOddsHistoryParams{
		SinceTime:    pgtype.Timestamp{Time: since, Valid: true},
		MinChangePct: 5.0, // 5% minimum change
		LimitCount:   100, // Process up to 100 movements per run
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to get recent odds history")
		return err
	}

	log.Info().
		Str("action", "processing_recent_movements").
		Int("movement_count", len(recentMovements)).
		Msg("Processing recent odds movements for smart money alerts")

	// Process each new odds movement
	processedCount := 0
	alertsCreated := 0

	// Process each odds movement for smart money alerts
	for _, movement := range recentMovements {
		err = j.tracker.AnalyzeOddsHistoryForAlerts(ctx, int64(movement.ID))
		if err != nil {
			log.Error().
				Err(err).
				Int32("odds_history_id", movement.ID).
				Msg("Failed to analyze odds movement")
			continue
		}
		processedCount++

		// Count as alert created if movement was significant enough
		if movement.ChangePercentage != nil && *movement.ChangePercentage >= 20.0 {
			alertsCreated++
		}
	}

	// Clean up expired alerts
	err = j.cleanupExpiredAlerts(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to cleanup expired alerts")
	}

	duration := time.Since(start)
	log.Info().
		Str("action", "processor_complete").
		Int("processed_count", processedCount).
		Int("alerts_created", alertsCreated).
		Dur("duration", duration).
		Msg("Smart money processor job completed")

	return nil
}

// cleanupExpiredAlerts removes expired alerts from the database
func (j *SmartMoneyProcessorJob) cleanupExpiredAlerts(ctx context.Context) error {
	err := j.queries.DeactivateExpiredAlerts(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Integration point for events service
// This method should be called from the events service when new odds history is created
func ProcessNewOddsMovement(ctx context.Context, tracker *services.SmartMoneyTracker, oddsHistoryID int64) {
	log := logger.WithContext(ctx, "process-new-movement")

	// Analyze the movement for potential alerts
	err := tracker.AnalyzeOddsHistoryForAlerts(ctx, oddsHistoryID)
	if err != nil {
		log.Error().
			Err(err).
			Int64("odds_history_id", oddsHistoryID).
			Msg("Failed to analyze odds movement for smart money alerts")
	}
}
