package jobs

import (
	"context"
	"time"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/services"
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

// Schedule returns the cron schedule - every 15 minutes
func (j *SmartMoneyProcessorJob) Schedule() string {
	return "*/15 * * * *" // Every 15 minutes
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

	// Process recent movements with smart money detection using real betting data
	err := j.tracker.ProcessRecentMovements(ctx, 1) // Process last hour
	if err != nil {
		log.Error().Err(err).Msg("Failed to process recent movements")
		return err
	}

	log.Info().
		Str("action", "processing_complete").
		Msg("Smart money processing completed successfully")

	// Clean up expired alerts
	err = j.cleanupExpiredAlerts(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to cleanup expired alerts")
	}

	duration := time.Since(start)
	log.Info().
		Str("action", "processor_complete").
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

	// Process recent movements to catch new alerts
	err := tracker.ProcessRecentMovements(ctx, 1)
	if err != nil {
		log.Error().
			Err(err).
			Int64("odds_history_id", oddsHistoryID).
			Msg("Failed to process odds movement for smart money alerts")
	}
}
