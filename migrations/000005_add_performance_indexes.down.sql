-- Remove performance indexes

-- Events table indexes
DROP INDEX IF EXISTS idx_events_active_sync;
DROP INDEX IF EXISTS idx_events_filter_composite;
DROP INDEX IF EXISTS idx_events_home_team_date;
DROP INDEX IF EXISTS idx_events_away_team_date;
DROP INDEX IF EXISTS idx_events_external_id;
DROP INDEX IF EXISTS idx_events_live_status;
DROP INDEX IF EXISTS idx_events_date_only;

-- Teams table indexes
DROP INDEX IF EXISTS idx_teams_external_id;
DROP INDEX IF EXISTS idx_teams_name_pattern;
DROP INDEX IF EXISTS idx_teams_api_football_id;
DROP INDEX IF EXISTS idx_teams_enrichment_needed;

-- Leagues table indexes
DROP INDEX IF EXISTS idx_leagues_external_id;
DROP INDEX IF EXISTS idx_leagues_sport_active;
DROP INDEX IF EXISTS idx_leagues_api_football_id;
DROP INDEX IF EXISTS idx_leagues_enrichment_needed;

-- Current odds table indexes
DROP INDEX IF EXISTS idx_current_odds_bulk_lookup;
DROP INDEX IF EXISTS idx_current_odds_movement;
DROP INDEX IF EXISTS idx_current_odds_last_updated;

-- Odds history table indexes
DROP INDEX IF EXISTS idx_odds_history_event_time;
DROP INDEX IF EXISTS idx_odds_history_big_movers;
DROP INDEX IF EXISTS idx_odds_history_recent_changes;
DROP INDEX IF EXISTS idx_odds_history_sharp_money;
DROP INDEX IF EXISTS idx_odds_history_reverse_movements;
DROP INDEX IF EXISTS idx_odds_history_significance;
DROP INDEX IF EXISTS idx_odds_history_composite;

-- Additional supporting indexes
DROP INDEX IF EXISTS idx_market_types_code;
DROP INDEX IF EXISTS idx_league_mappings_internal;
DROP INDEX IF EXISTS idx_team_mappings_internal;