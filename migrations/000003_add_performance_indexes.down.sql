-- Drop performance indexes

-- Movement alerts indexes
DROP INDEX IF EXISTS idx_movement_alerts_type;
DROP INDEX IF EXISTS idx_movement_alerts_active;
DROP INDEX IF EXISTS idx_movement_alerts_created_at;

-- API Football mapping indexes
DROP INDEX IF EXISTS idx_team_mappings_confidence;
DROP INDEX IF EXISTS idx_league_mappings_confidence;

-- Betting volume indexes
DROP INDEX IF EXISTS idx_betting_volume_recorded_at;
DROP INDEX IF EXISTS idx_betting_volume_event_id;

-- Team and league lookup indexes
DROP INDEX IF EXISTS idx_leagues_sport_id;
DROP INDEX IF EXISTS idx_leagues_slug;
DROP INDEX IF EXISTS idx_teams_slug;

-- Outcome distributions indexes
DROP INDEX IF EXISTS idx_outcome_dist_bet_percentage;
DROP INDEX IF EXISTS idx_outcome_dist_event_id;

-- Current odds indexes
DROP INDEX IF EXISTS idx_current_odds_movement;
DROP INDEX IF EXISTS idx_current_odds_last_updated;
DROP INDEX IF EXISTS idx_current_odds_event_id;

-- Odds history indexes
DROP INDEX IF EXISTS idx_odds_history_reverse;
DROP INDEX IF EXISTS idx_odds_history_sharp_indicator;
DROP INDEX IF EXISTS idx_odds_history_composite;
DROP INDEX IF EXISTS idx_odds_history_recorded_at;
DROP INDEX IF EXISTS idx_odds_history_event_id;

-- Events indexes
DROP INDEX IF EXISTS idx_events_live;
DROP INDEX IF EXISTS idx_events_composite;
DROP INDEX IF EXISTS idx_events_league_id;
DROP INDEX IF EXISTS idx_events_status;
DROP INDEX IF EXISTS idx_events_event_date;