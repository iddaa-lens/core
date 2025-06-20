package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/logger"
)

// SmartMoneyTracker analyzes odds movements and generates alerts using existing data
type SmartMoneyTracker struct {
	db     *database.Queries
	logger *logger.Logger

	// Alert thresholds
	bigMoverThreshold     float64 // 20% default
	massiveMoverThreshold float64 // 50% default
	multiplierThreshold   float64 // 2.0x default
	sharpMoneyThreshold   float64 // 0.6 confidence default
}

// NewSmartMoneyTracker creates a new smart money tracker service
func NewSmartMoneyTracker(db *database.Queries) *SmartMoneyTracker {
	return &SmartMoneyTracker{
		db:                    db,
		logger:                logger.New("smart-money-tracker"),
		bigMoverThreshold:     20.0,
		massiveMoverThreshold: 50.0,
		multiplierThreshold:   2.0,
		sharpMoneyThreshold:   0.6,
	}
}

// MovementAlert represents a smart money alert
type MovementAlert struct {
	ID               int64
	OddsHistoryID    int64
	AlertType        string // 'big_mover', 'reverse_line', 'sharp_money', 'value_spot'
	Severity         string // 'low', 'medium', 'high', 'critical'
	Title            string
	Message          string
	ChangePercent    float64
	Multiplier       float64
	ConfidenceScore  float64
	MinutesToKickoff int
	CreatedAt        time.Time
	ExpiresAt        time.Time
	IsActive         bool
}

// SmartMoneyIndicators represents calculated indicators from movement analysis
type SmartMoneyIndicators struct {
	IsReverseMovement bool
	ConfidenceScore   float64
	VolumeDirection   string  // 'with_movement', 'against_movement', 'neutral'
	TimeProximity     float64 // 0.0 to 1.0 based on time to kickoff
	PublicBias        float64 // difference between public % and implied probability
}

// AnalyzeOddsHistoryForAlerts processes new odds history records and creates alerts
func (smt *SmartMoneyTracker) AnalyzeOddsHistoryForAlerts(ctx context.Context, oddsHistoryID int64) error {
	// Get the odds history record
	oddsHistory, err := smt.db.GetOddsHistoryByID(ctx, oddsHistoryID)
	if err != nil {
		return fmt.Errorf("failed to get odds history record %d: %w", oddsHistoryID, err)
	}

	// Skip if no meaningful change
	changePercent := 0.0
	if oddsHistory.ChangePercentage.Valid {
		if changePercentFloat, err := oddsHistory.ChangePercentage.Float64Value(); err == nil && changePercentFloat.Valid {
			changePercent = changePercentFloat.Float64
		}
	}

	if math.Abs(changePercent) < 5.0 {
		return nil // Skip small movements
	}

	multiplier := 1.0
	if oddsHistory.Multiplier.Valid {
		if multiplierFloat, err := oddsHistory.Multiplier.Float64Value(); err == nil && multiplierFloat.Valid {
			multiplier = multiplierFloat.Float64
		}
	}

	// Check for big mover alert
	if math.Abs(changePercent) >= smt.bigMoverThreshold || multiplier >= smt.multiplierThreshold {
		err = smt.createBigMoverAlert(ctx, oddsHistoryID, oddsHistory, changePercent, multiplier)
		if err != nil {
			smt.logger.Error().Err(err).Msg("Failed to create big mover alert")
		}
	}

	// Calculate smart money indicators
	indicators := smt.calculateSmartMoneyIndicators(ctx, oddsHistory, changePercent, multiplier)

	// Check for reverse line movement alert
	if indicators.IsReverseMovement {
		err = smt.createReverseLineAlert(ctx, oddsHistoryID, oddsHistory, indicators)
		if err != nil {
			smt.logger.Error().Err(err).Msg("Failed to create reverse line alert")
		}
	}

	// Check for sharp money alert
	if indicators.ConfidenceScore >= smt.sharpMoneyThreshold {
		err = smt.createSharpMoneyAlert(ctx, oddsHistoryID, oddsHistory, indicators)
		if err != nil {
			smt.logger.Error().Err(err).Msg("Failed to create sharp money alert")
		}
	}

	// Check for value spot alert
	if indicators.PublicBias > 15.0 { // Public overweight by >15%
		err = smt.createValueSpotAlert(ctx, oddsHistoryID, oddsHistory, indicators)
		if err != nil {
			smt.logger.Error().Err(err).Msg("Failed to create value spot alert")
		}
	}

	return nil
}

// ProcessOddsUpdate is called whenever odds are updated to detect movements
func (smt *SmartMoneyTracker) ProcessOddsUpdate(ctx context.Context, eventID int, marketTypeID int, outcome string, newOdds float64) error {
	// For now, just log that we would process this
	smt.logger.Debug().
		Int("event_id", eventID).
		Int("market_type_id", marketTypeID).
		Str("outcome", outcome).
		Float64("new_odds", newOdds).
		Msg("Would analyze odds movement")

	return nil
}
