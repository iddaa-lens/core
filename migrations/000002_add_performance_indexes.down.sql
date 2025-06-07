-- Drop additional performance indexes

-- Team mappings indexes
DROP INDEX IF EXISTS idx_team_mappings_football_api_team_id;
DROP INDEX IF EXISTS idx_team_mappings_internal_team_id;

-- Odds history table indexes
DROP INDEX IF EXISTS idx_odds_history_composite;
DROP INDEX IF EXISTS idx_odds_history_market_type_id;
DROP INDEX IF EXISTS idx_odds_history_event_id;

-- Current odds table indexes
DROP INDEX IF EXISTS idx_current_odds_market_type_id;
DROP INDEX IF EXISTS idx_current_odds_event_id;

-- Market types table indexes
DROP INDEX IF EXISTS idx_market_types_market_sub_type;
DROP INDEX IF EXISTS idx_market_types_market_type;

-- Events table indexes
DROP INDEX IF EXISTS idx_events_composite;
DROP INDEX IF EXISTS idx_events_sport_id;
DROP INDEX IF EXISTS idx_events_away_team_id;
DROP INDEX IF EXISTS idx_events_home_team_id;
DROP INDEX IF EXISTS idx_events_external_id;

-- Teams table indexes
DROP INDEX IF EXISTS idx_teams_name;