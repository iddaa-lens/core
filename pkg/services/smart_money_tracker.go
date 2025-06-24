package services

import (
	"context"
	"fmt"
	"time"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/jackc/pgx/v5/pgtype"
)

// SmartMoneyTracker analyzes odds movements using real betting distribution data
type SmartMoneyTracker struct {
	db     *generated.Queries
	logger *logger.Logger
}

// NewSmartMoneyTracker creates a new smart money tracker
func NewSmartMoneyTracker(db *generated.Queries) *SmartMoneyTracker {
	return &SmartMoneyTracker{
		db:     db,
		logger: logger.New("smart-money-tracker"),
	}
}

// ProcessRecentMovements analyzes recent odds movements for smart money patterns
func (smt *SmartMoneyTracker) ProcessRecentMovements(ctx context.Context, hours int) error {
	sinceTime := pgtype.Timestamp{
		Time:  time.Now().Add(-time.Duration(hours) * time.Hour),
		Valid: true,
	}

	// 1. Process reverse line movements using real betting data
	reverseMovements, err := smt.db.GetReverseLineMovements(ctx, generated.GetReverseLineMovementsParams{
		SinceTime:  sinceTime,
		LimitCount: 100,
	})
	if err != nil {
		return fmt.Errorf("failed to get reverse line movements: %w", err)
	}

	smt.logger.Info().
		Int("count", len(reverseMovements)).
		Msg("Found reverse line movements")

	for _, movement := range reverseMovements {
		if err := smt.createReverseLineAlert(ctx, movement); err != nil {
			smt.logger.Error().Err(err).
				Int32("odds_history_id", movement.ID).
				Msg("Failed to create reverse line alert")
		}
	}

	// 2. Process sharp money indicators
	sharpIndicators, err := smt.db.GetSharpMoneyIndicators(ctx, generated.GetSharpMoneyIndicatorsParams{
		SinceTime:  sinceTime,
		LimitCount: 100,
	})
	if err != nil {
		return fmt.Errorf("failed to get sharp money indicators: %w", err)
	}

	smt.logger.Info().
		Int("count", len(sharpIndicators)).
		Msg("Found sharp money indicators")

	for _, indicator := range sharpIndicators {
		// Only create alerts for high-confidence sharp money (score > 60)
		if indicator.SharpMoneyScore > 60 {
			if err := smt.createSharpMoneyAlert(ctx, indicator); err != nil {
				smt.logger.Error().Err(err).
					Int32("odds_history_id", indicator.ID).
					Msg("Failed to create sharp money alert")
			}
		}
	}

	// 3. Process steam moves
	steamMoves, err := smt.db.GetSteamMoves(ctx, generated.GetSteamMovesParams{
		SinceTime:  sinceTime,
		LimitCount: 50,
	})
	if err != nil {
		return fmt.Errorf("failed to get steam moves: %w", err)
	}

	smt.logger.Info().
		Int("count", len(steamMoves)).
		Msg("Found steam moves")

	for _, steam := range steamMoves {
		if err := smt.createSteamMoveAlert(ctx, steam); err != nil {
			smt.logger.Error().Err(err).
				Int32("odds_history_id", steam.ID).
				Msg("Failed to create steam move alert")
		}
	}

	// 4. Process value spots
	valueSpots, err := smt.db.GetValueSpots(ctx, generated.GetValueSpotsParams{
		SinceTime:      sinceTime,
		MinBiasPct:     15.0, // At least 15% public bias
		MinMovementPct: 5.0,  // At least 5% movement
		LimitCount:     50,
	})
	if err != nil {
		return fmt.Errorf("failed to get value spots: %w", err)
	}

	smt.logger.Info().
		Int("count", len(valueSpots)).
		Msg("Found value spots")

	for _, value := range valueSpots {
		if err := smt.createValueSpotAlert(ctx, value); err != nil {
			smt.logger.Error().Err(err).
				Int32("odds_history_id", value.ID).
				Msg("Failed to create value spot alert")
		}
	}

	// 5. Deactivate expired alerts
	if err := smt.db.DeactivateExpiredAlerts(ctx); err != nil {
		smt.logger.Error().Err(err).Msg("Failed to deactivate expired alerts")
	}

	return nil
}

// createReverseLineAlert creates an alert for true reverse line movements
func (smt *SmartMoneyTracker) createReverseLineAlert(ctx context.Context, movement generated.GetReverseLineMovementsRow) error {
	// Build match name
	matchName := "Unknown Match"
	if movement.HomeTeamName != nil && movement.AwayTeamName != nil {
		matchName = fmt.Sprintf("%s vs %s", *movement.HomeTeamName, *movement.AwayTeamName)
	}

	// Calculate confidence based on reverse strength
	confidence := float32(movement.ReverseStrength) / 100.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	title := fmt.Sprintf("Reverse Line Movement: %s", matchName)

	var movementType string
	switch movement.MovementType {
	case "public_heavy_odds_worse":
		movementType = "Public betting heavy but odds getting worse"
	case "public_light_odds_better":
		movementType = "Public avoiding but odds getting better"
	default:
		movementType = "Unknown"
	}

	message := fmt.Sprintf("🔄 %s - %s: %s (Public: %.0f%%, Strength: %d)",
		matchName, movement.Outcome, movementType,
		*movement.BetPercentage, movement.ReverseStrength)

	changePercent := float32(0)
	if movement.ChangePercentage != nil {
		changePercent = *movement.ChangePercentage
	}

	multiplier := float64(1)
	if movement.Multiplier != nil {
		multiplier = *movement.Multiplier
	}

	_, err := smt.db.CreateMovementAlert(ctx, generated.CreateMovementAlertParams{
		OddsHistoryID:    movement.ID,
		AlertType:        "reverse_line",
		Severity:         smt.calculateSeverity(confidence),
		Title:            title,
		Message:          message,
		ChangePercentage: changePercent,
		Multiplier:       multiplier,
		ConfidenceScore:  confidence,
		MinutesToKickoff: movement.MinutesToKickoff,
	})
	return err
}

// createSharpMoneyAlert creates an alert for sharp money indicators
func (smt *SmartMoneyTracker) createSharpMoneyAlert(ctx context.Context, indicator generated.GetSharpMoneyIndicatorsRow) error {
	// Build match name
	matchName := "Unknown Match"
	if indicator.HomeTeamName != nil && indicator.AwayTeamName != nil {
		matchName = fmt.Sprintf("%s vs %s", *indicator.HomeTeamName, *indicator.AwayTeamName)
	}

	confidence := float32(indicator.SharpMoneyScore) / 100.0

	title := fmt.Sprintf("Sharp Money Detected: %s", matchName)

	publicInfo := ""
	if indicator.BetPercentage != nil {
		publicInfo = fmt.Sprintf(" (Public: %.0f%%)", *indicator.BetPercentage)
	}

	message := fmt.Sprintf("🎯 %s - %s shows sharp activity (Score: %d/100)%s",
		matchName, indicator.Outcome, indicator.SharpMoneyScore, publicInfo)

	changePercent := float32(0)
	if indicator.ChangePercentage != nil {
		changePercent = *indicator.ChangePercentage
	}

	multiplier := float64(1)
	if indicator.Multiplier != nil {
		multiplier = *indicator.Multiplier
	}

	_, err := smt.db.CreateMovementAlert(ctx, generated.CreateMovementAlertParams{
		OddsHistoryID:    indicator.ID,
		AlertType:        "sharp_money",
		Severity:         smt.calculateSeverity(confidence),
		Title:            title,
		Message:          message,
		ChangePercentage: changePercent,
		Multiplier:       multiplier,
		ConfidenceScore:  confidence,
		MinutesToKickoff: indicator.MinutesToKickoff,
	})
	return err
}

// createSteamMoveAlert creates an alert for steam moves
func (smt *SmartMoneyTracker) createSteamMoveAlert(ctx context.Context, steam generated.GetSteamMovesRow) error {
	// Build match name
	matchName := "Unknown Match"
	if steam.HomeTeamName != nil && steam.AwayTeamName != nil {
		matchName = fmt.Sprintf("%s vs %s", *steam.HomeTeamName, *steam.AwayTeamName)
	}

	// Base confidence on movement count and speed
	confidence := float32(0.5)
	if steam.MovementsLastHour > 5 {
		confidence = 0.8
	} else if steam.MovementsLastHour > 3 {
		confidence = 0.7
	}

	title := fmt.Sprintf("Steam Move: %s", matchName)
	message := fmt.Sprintf("⚡ %s - %s moving rapidly (%d moves/hour, last: %.0fs ago)",
		matchName, steam.Outcome, steam.MovementsLastHour, steam.SecondsSinceLastMove)

	changePercent := float32(0)
	if steam.ChangePercentage != nil {
		changePercent = *steam.ChangePercentage
	}

	multiplier := float64(1)
	if steam.Multiplier != nil {
		multiplier = *steam.Multiplier
	}

	_, err := smt.db.CreateMovementAlert(ctx, generated.CreateMovementAlertParams{
		OddsHistoryID:    steam.ID,
		AlertType:        "steam_move",
		Severity:         "high",
		Title:            title,
		Message:          message,
		ChangePercentage: changePercent,
		Multiplier:       multiplier,
		ConfidenceScore:  confidence,
		MinutesToKickoff: steam.MinutesToKickoff,
	})
	return err
}

// createValueSpotAlert creates an alert for value betting opportunities
func (smt *SmartMoneyTracker) createValueSpotAlert(ctx context.Context, value generated.GetValueSpotsRow) error {
	// Build match name
	matchName := "Unknown Match"
	if value.HomeTeamName != nil && value.AwayTeamName != nil {
		matchName = fmt.Sprintf("%s vs %s", *value.HomeTeamName, *value.AwayTeamName)
	}

	// Confidence based on public bias
	confidence := float32(value.PublicBias) / 100.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	title := fmt.Sprintf("Value Spot: %s", matchName)
	message := fmt.Sprintf("💰 %s - %s overbet by public (Bet: %.0f%%, Fair: %.0f%%)",
		matchName, value.Outcome, *value.BetPercentage, *value.ImpliedProbability)

	changePercent := float32(0)
	if value.ChangePercentage != nil {
		changePercent = *value.ChangePercentage
	}

	multiplier := float64(1)
	if value.Multiplier != nil {
		multiplier = *value.Multiplier
	}

	_, err := smt.db.CreateMovementAlert(ctx, generated.CreateMovementAlertParams{
		OddsHistoryID:    value.ID,
		AlertType:        "value_spot",
		Severity:         smt.calculateSeverity(confidence),
		Title:            title,
		Message:          message,
		ChangePercentage: changePercent,
		Multiplier:       multiplier,
		ConfidenceScore:  confidence,
		MinutesToKickoff: value.MinutesToKickoff,
	})
	return err
}

// calculateSeverity determines alert severity based on confidence score
func (smt *SmartMoneyTracker) calculateSeverity(confidence float32) string {
	if confidence >= 0.8 {
		return "critical"
	} else if confidence >= 0.6 {
		return "high"
	} else if confidence >= 0.4 {
		return "medium"
	}
	return "low"
}

// GetActiveAlerts retrieves active smart money alerts
func (smt *SmartMoneyTracker) GetActiveAlerts(ctx context.Context, alertType, minSeverity string, limit int) ([]generated.GetActiveAlertsRow, error) {
	return smt.db.GetActiveAlerts(ctx, generated.GetActiveAlertsParams{
		AlertType:   alertType,
		MinSeverity: minSeverity,
		LimitCount:  int64(limit),
	})
}
