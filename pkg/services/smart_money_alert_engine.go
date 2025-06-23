package services

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/iddaa-lens/core/pkg/database/generated"
)

// calculateSmartMoneyIndicators analyzes an odds movement for smart money patterns
func (smt *SmartMoneyTracker) calculateSmartMoneyIndicators(ctx context.Context, oddsHistory generated.OddsHistory, changePercent, multiplier float64) SmartMoneyIndicators {
	indicators := SmartMoneyIndicators{}

	// Get event information for timing analysis
	if oddsHistory.EventID != nil {
		event, err := smt.db.GetEventByID(ctx, *oddsHistory.EventID)
		if err == nil && event.EventDate.Valid {
			minutesToKickoff := time.Until(event.EventDate.Time).Minutes()

			// Time proximity factor (closer to game = higher confidence)
			if minutesToKickoff <= 0 {
				indicators.TimeProximity = 1.0 // Live
			} else if minutesToKickoff <= 60 {
				indicators.TimeProximity = 0.9 // Within 1 hour
			} else if minutesToKickoff <= 360 {
				indicators.TimeProximity = 0.7 // Within 6 hours
			} else if minutesToKickoff <= 1440 {
				indicators.TimeProximity = 0.5 // Within 24 hours
			} else {
				indicators.TimeProximity = 0.3 // More than 24 hours
			}
		} else {
			indicators.TimeProximity = 0.5 // Default if event not found
		}
	} else {
		indicators.TimeProximity = 0.5 // Default if no event ID
	}

	// Calculate confidence score
	confidence := 0.0

	// Large movements often indicate informed betting
	if math.Abs(changePercent) >= 30 {
		confidence += 0.3
	} else if math.Abs(changePercent) >= 20 {
		confidence += 0.2
	}

	// Multiplier factor
	if multiplier >= 2.5 {
		confidence += 0.3
	} else if multiplier >= 2.0 {
		confidence += 0.2
	}

	// Time factor (closer to game = more informed)
	confidence += indicators.TimeProximity * 0.3

	// Counter-public movement patterns
	isCounterPublic := smt.isCounterPublicMovement(ctx, oddsHistory, changePercent)
	if isCounterPublic {
		confidence += 0.2
		indicators.IsReverseMovement = true
	}

	// Check against betting volume data for reverse line movement
	indicators.VolumeDirection = smt.analyzeVolumeDirection(ctx, oddsHistory, changePercent)
	if indicators.VolumeDirection == "against_movement" {
		indicators.IsReverseMovement = true
		confidence += 0.3
	}

	// Analyze public bias using outcome distributions
	indicators.PublicBias = smt.calculatePublicBias(ctx, oddsHistory)
	if indicators.PublicBias > 20 {
		confidence += 0.2
	}

	// Cap confidence at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	indicators.ConfidenceScore = confidence
	return indicators
}

// isCounterPublicMovement checks if movement goes against typical public patterns
func (smt *SmartMoneyTracker) isCounterPublicMovement(ctx context.Context, oddsHistory generated.OddsHistory, changePercent float64) bool {
	if oddsHistory.PreviousValue == nil {
		return false
	}

	prevOdds := *oddsHistory.PreviousValue
	if prevOdds == 0 {
		return false
	}

	// Favorites getting longer odds (public loves favorites)
	if prevOdds < 2.0 && changePercent > 0 {
		return true
	}

	// Big underdogs getting shorter odds (public avoids big underdogs)
	if prevOdds > 5.0 && changePercent < 0 {
		return true
	}

	// Market-specific patterns for different bet types
	if oddsHistory.MarketTypeID != nil {
		// Get market type to check for specific patterns
		marketType, err := smt.db.GetMarketTypeByID(ctx, *oddsHistory.MarketTypeID)
		if err == nil {
			marketCode := strings.ToLower(marketType.Code)

			// Over/Under patterns - public typically loves overs
			if strings.Contains(marketCode, "over") || strings.Contains(marketCode, "alt") {
				if prevOdds > 1.9 && changePercent > 0 { // Over getting longer odds
					return true
				}
			}

			// Under patterns - public typically avoids unders
			if strings.Contains(marketCode, "under") {
				if prevOdds > 2.5 && changePercent < 0 { // Under getting shorter odds
					return true
				}
			}

			// Draw patterns - public typically avoids draws in close matches
			if strings.Contains(marketCode, "draw") || oddsHistory.Outcome == "X" {
				if prevOdds > 3.0 && changePercent < 0 { // Draw getting shorter odds
					return true
				}
			}

			// Double chance patterns - public loves "safer" bets
			if strings.Contains(marketCode, "double") {
				if prevOdds > 1.8 && changePercent > 0 { // Double chance getting longer odds
					return true
				}
			}
		}
	}

	return false
}

// analyzeVolumeDirection compares odds movement with volume patterns
func (smt *SmartMoneyTracker) analyzeVolumeDirection(ctx context.Context, oddsHistory generated.OddsHistory, changePercent float64) string {
	if oddsHistory.EventID == nil || oddsHistory.MarketTypeID == nil {
		return "neutral"
	}

	// Get outcome distribution to analyze public betting patterns
	distribution, err := smt.db.GetLatestOutcomeDistribution(ctx, generated.GetLatestOutcomeDistributionParams{
		EventID:  oddsHistory.EventID,
		MarketID: *oddsHistory.MarketTypeID,
		Outcome:  oddsHistory.Outcome,
	})

	if err != nil {
		return "neutral" // No distribution data available
	}

	// Analyze betting percentage vs odds movement
	// If public is betting heavily (>60%) but odds are moving away from them, it's reverse line movement

	// High public betting but odds moving against them (sharp money)
	if distribution.BetPercentage > 60 && changePercent > 0 { // Public loves it, odds getting worse
		return "against_movement"
	}
	if distribution.BetPercentage < 30 && changePercent < 0 { // Public avoids it, odds getting better
		return "against_movement"
	}

	// Movement with public sentiment
	if distribution.BetPercentage > 60 && changePercent < 0 { // Public loves it, odds getting better
		return "with_movement"
	}
	if distribution.BetPercentage < 30 && changePercent > 0 { // Public avoids it, odds getting worse
		return "with_movement"
	}

	return "neutral"
}

// calculatePublicBias compares betting percentages with implied probability
func (smt *SmartMoneyTracker) calculatePublicBias(ctx context.Context, oddsHistory generated.OddsHistory) float64 {
	if oddsHistory.EventID == nil || oddsHistory.MarketTypeID == nil {
		return 0.0
	}

	// Get outcome distribution data
	distribution, err := smt.db.GetLatestOutcomeDistribution(ctx, generated.GetLatestOutcomeDistributionParams{
		EventID:  oddsHistory.EventID,
		MarketID: *oddsHistory.MarketTypeID,
		Outcome:  oddsHistory.Outcome,
	})

	if err != nil {
		return 0.0 // No distribution data available
	}

	// Calculate bias: betting percentage - implied probability
	bias := float64(distribution.BetPercentage) - float64(*distribution.ImpliedProbability)
	return bias
}

// createBigMoverAlert creates an alert for significant odds movements
func (smt *SmartMoneyTracker) createBigMoverAlert(ctx context.Context, oddsHistoryID int64, oddsHistory generated.OddsHistory, changePercent, multiplier float64) error {
	severity := "medium"
	if math.Abs(changePercent) >= smt.massiveMoverThreshold || multiplier >= 3.0 {
		severity = "high"
	}

	// Get team names for alert message
	teamInfo := teamInfo{}
	if oddsHistory.EventID != nil {
		teamInfo = smt.getTeamInfo(ctx, *oddsHistory.EventID)
	}

	direction := "up"
	emoji := "📈"
	if changePercent < 0 {
		direction = "down"
		emoji = "📉"
	}

	title := fmt.Sprintf("Big Mover: %s", teamInfo.matchName)
	message := fmt.Sprintf("%s %s - %s odds moved %s %.1f%% (%.2fx)",
		emoji, teamInfo.matchName, oddsHistory.Outcome, direction, math.Abs(changePercent), multiplier)

	// Calculate minutes to kickoff
	minutesToKickoff := smt.getMinutesToKickoff(ctx, *oddsHistory.EventID)

	return smt.createAlert(ctx, oddsHistoryID, "big_mover", severity, title, message, changePercent, multiplier, 0.5, minutesToKickoff)
}

// createReverseLineAlert creates an alert for reverse line movements
func (smt *SmartMoneyTracker) createReverseLineAlert(ctx context.Context, oddsHistoryID int64, oddsHistory generated.OddsHistory, indicators SmartMoneyIndicators) error {
	teamInfo := teamInfo{}
	if oddsHistory.EventID != nil {
		teamInfo = smt.getTeamInfo(ctx, *oddsHistory.EventID)
	}

	title := fmt.Sprintf("Reverse Line Movement: %s", teamInfo.matchName)
	message := fmt.Sprintf("🔄 %s - %s moving against public money (%.0f%% confidence)",
		teamInfo.matchName, oddsHistory.Outcome, indicators.ConfidenceScore*100)

	minutesToKickoff := 0
	if oddsHistory.EventID != nil {
		minutesToKickoff = smt.getMinutesToKickoff(ctx, *oddsHistory.EventID)
	}
	// Extract change percentage
	changePercent := 0.0
	if oddsHistory.ChangePercentage != nil {
		changePercent = float64(*oddsHistory.ChangePercentage)
	}

	multiplier := 1.0
	if oddsHistory.Multiplier != nil {
		multiplier = *oddsHistory.Multiplier
	}

	return smt.createAlert(ctx, oddsHistoryID, "reverse_line", "high", title, message, changePercent, multiplier, indicators.ConfidenceScore, minutesToKickoff)
}

// createSharpMoneyAlert creates an alert for high-confidence sharp money movements
func (smt *SmartMoneyTracker) createSharpMoneyAlert(ctx context.Context, oddsHistoryID int64, oddsHistory generated.OddsHistory, indicators SmartMoneyIndicators) error {
	teamInfo := teamInfo{}
	if oddsHistory.EventID != nil {
		teamInfo = smt.getTeamInfo(ctx, *oddsHistory.EventID)
	}

	title := fmt.Sprintf("Sharp Money Detected: %s", teamInfo.matchName)
	message := fmt.Sprintf("🎯 %s - %s shows sharp money activity (%.0f%% confidence)",
		teamInfo.matchName, oddsHistory.Outcome, indicators.ConfidenceScore*100)

	minutesToKickoff := 0
	if oddsHistory.EventID != nil {
		minutesToKickoff = smt.getMinutesToKickoff(ctx, *oddsHistory.EventID)
	}
	// Extract change percentage
	changePercent := 0.0
	if oddsHistory.ChangePercentage != nil {
		changePercent = float64(*oddsHistory.ChangePercentage)
	}

	multiplier := 1.0
	if oddsHistory.Multiplier != nil {
		multiplier = *oddsHistory.Multiplier
	}

	return smt.createAlert(ctx, oddsHistoryID, "sharp_money", "critical", title, message, changePercent, multiplier, indicators.ConfidenceScore, minutesToKickoff)
}

// createValueSpotAlert creates an alert for value betting opportunities
func (smt *SmartMoneyTracker) createValueSpotAlert(ctx context.Context, oddsHistoryID int64, oddsHistory generated.OddsHistory, indicators SmartMoneyIndicators) error {
	teamInfo := teamInfo{}
	if oddsHistory.EventID != nil {
		teamInfo = smt.getTeamInfo(ctx, *oddsHistory.EventID)
	}

	title := fmt.Sprintf("Value Spot: %s", teamInfo.matchName)
	message := fmt.Sprintf("💰 %s - %s shows value opportunity (Public bias: %.1f%%)",
		teamInfo.matchName, oddsHistory.Outcome, indicators.PublicBias)

	minutesToKickoff := 0
	if oddsHistory.EventID != nil {
		minutesToKickoff = smt.getMinutesToKickoff(ctx, *oddsHistory.EventID)
	}
	// Extract change percentage
	changePercent := 0.0
	if oddsHistory.ChangePercentage != nil {
		changePercent = float64(*oddsHistory.ChangePercentage)
	}

	multiplier := 1.0
	if oddsHistory.Multiplier != nil {
		multiplier = *oddsHistory.Multiplier
	}

	return smt.createAlert(ctx, oddsHistoryID, "value_spot", "medium", title, message, changePercent, multiplier, indicators.ConfidenceScore, minutesToKickoff)
}

// createAlert is the core method that inserts alerts into the database
func (smt *SmartMoneyTracker) createAlert(
	ctx context.Context,
	oddsHistoryID int64,
	alertType,
	severity, title, message string,
	changePercent, multiplier, confidence float64,
	minutesToKickoff int,
) error {
	// Create the alert in the database
	alert, err := smt.db.CreateMovementAlert(ctx, generated.CreateMovementAlertParams{
		OddsHistoryID:    int32(oddsHistoryID),
		AlertType:        alertType,
		Severity:         severity,
		Title:            title,
		Message:          message,
		ChangePercentage: float32(changePercent),
		Multiplier:       multiplier,
		ConfidenceScore:  float32(confidence),
		MinutesToKickoff: func() *int32 {
			if minutesToKickoff > 0 {
				m := int32(minutesToKickoff)
				return &m
			}
			return nil
		}(),
	})

	if err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	smt.logger.Info().
		Int32("alert_id", alert.ID).
		Int64("odds_history_id", oddsHistoryID).
		Str("alert_type", alertType).
		Str("severity", severity).
		Str("title", title).
		Float64("change_percent", changePercent).
		Float64("multiplier", multiplier).
		Float64("confidence", confidence).
		Int("minutes_to_kickoff", minutesToKickoff).
		Msg("Smart money alert created successfully")

	return nil
}

// Helper types and methods for team information
type teamInfo struct {
	matchName string
	homeTeam  string
	awayTeam  string
}

// getTeamInfo retrieves team names for building alert messages
func (smt *SmartMoneyTracker) getTeamInfo(ctx context.Context, eventID int32) teamInfo {
	info := teamInfo{matchName: "Unknown Match"}

	if eventID == 0 {
		return info
	}

	event, err := smt.db.GetEventByID(ctx, eventID)
	if err != nil {
		return info
	}

	// Get team names
	var homeTeamName, awayTeamName string

	if event.HomeTeamID != nil {
		homeTeam, err := smt.db.GetTeam(ctx, *event.HomeTeamID)
		if err == nil {
			homeTeamName = homeTeam.Name
		}
	}

	if event.AwayTeamID != nil {
		awayTeam, err := smt.db.GetTeam(ctx, *event.AwayTeamID)
		if err == nil {
			awayTeamName = awayTeam.Name
		}
	}

	if homeTeamName != "" && awayTeamName != "" {
		info.matchName = fmt.Sprintf("%s vs %s", homeTeamName, awayTeamName)
		info.homeTeam = homeTeamName
		info.awayTeam = awayTeamName
	}

	return info
}

// getMinutesToKickoff calculates minutes until the event starts
func (smt *SmartMoneyTracker) getMinutesToKickoff(ctx context.Context, eventID int32) int {
	if eventID == 0 {
		return 0
	}

	event, err := smt.db.GetEventByID(ctx, eventID)
	if err != nil || !event.EventDate.Valid {
		return 0
	}

	minutesToKickoff := time.Until(event.EventDate.Time).Minutes()
	if minutesToKickoff < 0 {
		return 0 // Live or finished
	}

	return int(minutesToKickoff)
}
