// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0

package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type Querier interface {
	// Analyze correlation between volume and odds movement
	AnalyzeVolumeOddsPattern(ctx context.Context) ([]byte, error)
	CountEventsFiltered(ctx context.Context, arg CountEventsFilteredParams) (int64, error)
	CreateConfig(ctx context.Context, arg CreateConfigParams) (AppConfig, error)
	CreateDistributionHistory(ctx context.Context, arg CreateDistributionHistoryParams) (OutcomeDistributionHistory, error)
	CreateEnhancedLeagueMapping(ctx context.Context, arg CreateEnhancedLeagueMappingParams) (LeagueMapping, error)
	CreateEnhancedTeamMapping(ctx context.Context, arg CreateEnhancedTeamMappingParams) (TeamMapping, error)
	CreateEvent(ctx context.Context, arg CreateEventParams) (Event, error)
	CreateLeagueMapping(ctx context.Context, arg CreateLeagueMappingParams) (LeagueMapping, error)
	CreateMatchEvent(ctx context.Context, arg CreateMatchEventParams) (MatchEvent, error)
	CreateMovementAlert(ctx context.Context, arg CreateMovementAlertParams) (MovementAlert, error)
	CreateOddsHistory(ctx context.Context, arg CreateOddsHistoryParams) (OddsHistory, error)
	CreatePrediction(ctx context.Context, arg CreatePredictionParams) (Prediction, error)
	CreateTeam(ctx context.Context, arg CreateTeamParams) (Team, error)
	CreateTeamMapping(ctx context.Context, arg CreateTeamMappingParams) (TeamMapping, error)
	CreateVolumeHistory(ctx context.Context, arg CreateVolumeHistoryParams) (BettingVolumeHistory, error)
	DeactivateExpiredAlerts(ctx context.Context) error
	DeleteLeague(ctx context.Context, id int32) error
	EnrichLeagueWithAPIFootball(ctx context.Context, arg EnrichLeagueWithAPIFootballParams) (League, error)
	EnrichTeamWithAPIFootball(ctx context.Context, arg EnrichTeamWithAPIFootballParams) (Team, error)
	GetActiveAlerts(ctx context.Context, arg GetActiveAlertsParams) ([]GetActiveAlertsRow, error)
	GetActiveEventsForDetailedSync(ctx context.Context, limitCount int32) ([]Event, error)
	GetAlertsByUser(ctx context.Context, arg GetAlertsByUserParams) ([]GetAlertsByUserRow, error)
	GetBigMovers(ctx context.Context, arg GetBigMoversParams) ([]GetBigMoversRow, error)
	GetContrarianBets(ctx context.Context) ([]ContrarianBet, error)
	GetCurrentOdds(ctx context.Context, eventID pgtype.Int4) ([]GetCurrentOddsRow, error)
	GetCurrentOddsByMarket(ctx context.Context, arg GetCurrentOddsByMarketParams) ([]GetCurrentOddsByMarketRow, error)
	GetCurrentOddsForOutcome(ctx context.Context, arg GetCurrentOddsForOutcomeParams) ([]CurrentOdd, error)
	GetDistributionHistory(ctx context.Context, arg GetDistributionHistoryParams) ([]OutcomeDistributionHistory, error)
	GetEvent(ctx context.Context, id int32) (GetEventRow, error)
	// Analyze all betting distributions for an event
	GetEventBettingPatterns(ctx context.Context, eventID pgtype.Int4) ([]GetEventBettingPatternsRow, error)
	GetEventByExternalID(ctx context.Context, externalID string) (GetEventByExternalIDRow, error)
	GetEventByExternalIDSimple(ctx context.Context, externalID string) (Event, error)
	GetEventByID(ctx context.Context, id int32) (Event, error)
	GetEventDistributions(ctx context.Context, eventID pgtype.Int4) ([]OutcomeDistribution, error)
	GetEventStatisticsSummary(ctx context.Context, eventID int32) (GetEventStatisticsSummaryRow, error)
	GetEventsByTeam(ctx context.Context, arg GetEventsByTeamParams) ([]GetEventsByTeamRow, error)
	// Find low-volume events with big movements (potential sharp money)
	GetHiddenGems(ctx context.Context, arg GetHiddenGemsParams) ([]GetHiddenGemsRow, error)
	// Find events with high betting volume AND significant odds movement
	GetHotMovers(ctx context.Context, arg GetHotMoversParams) ([]GetHotMoversRow, error)
	GetLatestConfig(ctx context.Context, platform string) (AppConfig, error)
	GetLatestOutcomeDistribution(ctx context.Context, arg GetLatestOutcomeDistributionParams) (OutcomeDistribution, error)
	GetLatestPredictions(ctx context.Context, eventID pgtype.Int4) ([]GetLatestPredictionsRow, error)
	GetLeague(ctx context.Context, id int32) (League, error)
	GetLeagueByExternalID(ctx context.Context, externalID string) (League, error)
	GetLeagueMapping(ctx context.Context, internalLeagueID int32) (LeagueMapping, error)
	GetLeaguesByAPIFootballID(ctx context.Context, apiFootballID pgtype.Int4) ([]League, error)
	GetLiveEvents(ctx context.Context) ([]GetLiveEventsRow, error)
	GetMarketType(ctx context.Context, code string) (MarketType, error)
	GetMarketTypeByID(ctx context.Context, id int32) (MarketType, error)
	GetMatchEvents(ctx context.Context, eventID pgtype.Int4) ([]MatchEvent, error)
	GetMatchStatistics(ctx context.Context, eventID pgtype.Int4) ([]MatchStatistic, error)
	GetNationalTeams(ctx context.Context) ([]Team, error)
	// Get odds changes for a specific market
	GetOddsChangesByMarket(ctx context.Context, arg GetOddsChangesByMarketParams) ([]GetOddsChangesByMarketRow, error)
	// Get full odds history for a specific event
	GetOddsHistory(ctx context.Context, eventID pgtype.Int4) ([]GetOddsHistoryRow, error)
	GetOddsHistoryByID(ctx context.Context, id int32) (OddsHistory, error)
	GetOddsMovements(ctx context.Context, arg GetOddsMovementsParams) ([]GetOddsMovementsRow, error)
	GetOutcomeDistribution(ctx context.Context, arg GetOutcomeDistributionParams) (OutcomeDistribution, error)
	GetPredictionAccuracy(ctx context.Context, sinceDate pgtype.Timestamp) ([]GetPredictionAccuracyRow, error)
	GetPredictionsByEvent(ctx context.Context, eventID pgtype.Int4) ([]GetPredictionsByEventRow, error)
	// Smart Money Tracker queries
	GetRecentBigMovers(ctx context.Context, arg GetRecentBigMoversParams) ([]GetRecentBigMoversRow, error)
	// Get recent significant odds movements across all events
	GetRecentMovements(ctx context.Context, arg GetRecentMovementsParams) ([]GetRecentMovementsRow, error)
	GetRecentOddsHistory(ctx context.Context, arg GetRecentOddsHistoryParams) ([]GetRecentOddsHistoryRow, error)
	GetReverseLineMovements(ctx context.Context, arg GetReverseLineMovementsParams) ([]GetReverseLineMovementsRow, error)
	GetSport(ctx context.Context, id int32) (Sport, error)
	// Get potentially suspicious odds movements (sharp money indicators)
	GetSuspiciousMovements(ctx context.Context, arg GetSuspiciousMovementsParams) ([]GetSuspiciousMovementsRow, error)
	GetTeam(ctx context.Context, id int32) (Team, error)
	GetTeamByExternalID(ctx context.Context, externalID string) (Team, error)
	GetTeamMapping(ctx context.Context, internalTeamID int32) (TeamMapping, error)
	GetTeamsByAPIFootballID(ctx context.Context, apiFootballID pgtype.Int4) (Team, error)
	GetTeamsByFoundedRange(ctx context.Context, arg GetTeamsByFoundedRangeParams) ([]Team, error)
	GetTeamsByVenueCapacity(ctx context.Context, arg GetTeamsByVenueCapacityParams) ([]Team, error)
	GetTeamsNeedingEnrichment(ctx context.Context, limitCount int32) ([]Team, error)
	GetTopDistributions(ctx context.Context, limitCount int32) ([]OutcomeDistribution, error)
	// Get current top events by betting volume
	GetTopVolumeEvents(ctx context.Context) ([]GetTopVolumeEventsRow, error)
	GetUserSmartMoneyPreferences(ctx context.Context, userID string) (SmartMoneyPreference, error)
	GetValueSpots(ctx context.Context, arg GetValueSpotsParams) ([]GetValueSpotsRow, error)
	// Get volume history for a specific event
	GetVolumeHistory(ctx context.Context, eventID pgtype.Int4) ([]GetVolumeHistoryRow, error)
	ListEventsByDate(ctx context.Context, eventDate interface{}) ([]ListEventsByDateRow, error)
	ListEventsFiltered(ctx context.Context, arg ListEventsFilteredParams) ([]ListEventsFilteredRow, error)
	ListLeagueMappings(ctx context.Context) ([]LeagueMapping, error)
	ListLeagues(ctx context.Context) ([]League, error)
	ListLeaguesForAPIEnrichment(ctx context.Context, limitCount int32) ([]League, error)
	ListMarketTypes(ctx context.Context) ([]MarketType, error)
	ListSports(ctx context.Context) ([]Sport, error)
	ListTeamMappings(ctx context.Context) ([]TeamMapping, error)
	ListTeamsByLeague(ctx context.Context, leagueID pgtype.Int4) ([]Team, error)
	ListUnmappedFootballLeagues(ctx context.Context) ([]League, error)
	ListUnmappedLeagues(ctx context.Context) ([]League, error)
	ListUnmappedTeams(ctx context.Context) ([]Team, error)
	MarkAlertClicked(ctx context.Context, alertID int32) error
	MarkAlertViewed(ctx context.Context, alertID int32) error
	RefreshContrarianBets(ctx context.Context) error
	SearchTeams(ctx context.Context, arg SearchTeamsParams) ([]Team, error)
	SearchTeamsByCode(ctx context.Context, arg SearchTeamsByCodeParams) ([]Team, error)
	UpdateEventLiveData(ctx context.Context, arg UpdateEventLiveDataParams) (Event, error)
	UpdateEventStatus(ctx context.Context, arg UpdateEventStatusParams) (Event, error)
	UpdateEventVolume(ctx context.Context, arg UpdateEventVolumeParams) (Event, error)
	UpdateLeague(ctx context.Context, arg UpdateLeagueParams) (League, error)
	UpdateLeagueApiFootballID(ctx context.Context, arg UpdateLeagueApiFootballIDParams) error
	UpdateSport(ctx context.Context, arg UpdateSportParams) (Sport, error)
	UpdateTeam(ctx context.Context, arg UpdateTeamParams) (Team, error)
	UpdateTeamApiFootballID(ctx context.Context, arg UpdateTeamApiFootballIDParams) error
	UpsertConfig(ctx context.Context, arg UpsertConfigParams) (AppConfig, error)
	UpsertCurrentOdds(ctx context.Context, arg UpsertCurrentOddsParams) (CurrentOdd, error)
	UpsertEvent(ctx context.Context, arg UpsertEventParams) (Event, error)
	UpsertLeague(ctx context.Context, arg UpsertLeagueParams) (League, error)
	UpsertLeagueMapping(ctx context.Context, arg UpsertLeagueMappingParams) (LeagueMapping, error)
	UpsertMarketType(ctx context.Context, arg UpsertMarketTypeParams) (MarketType, error)
	UpsertMarketTypeByExternalID(ctx context.Context, arg UpsertMarketTypeByExternalIDParams) (MarketType, error)
	UpsertMatchStatistics(ctx context.Context, arg UpsertMatchStatisticsParams) (MatchStatistic, error)
	UpsertOutcomeDistribution(ctx context.Context, arg UpsertOutcomeDistributionParams) (OutcomeDistribution, error)
	UpsertSport(ctx context.Context, arg UpsertSportParams) (Sport, error)
	UpsertTeam(ctx context.Context, arg UpsertTeamParams) (Team, error)
	UpsertTeamMapping(ctx context.Context, arg UpsertTeamMappingParams) (TeamMapping, error)
	UpsertUserSmartMoneyPreferences(ctx context.Context, arg UpsertUserSmartMoneyPreferencesParams) (SmartMoneyPreference, error)
}

var _ Querier = (*Queries)(nil)
