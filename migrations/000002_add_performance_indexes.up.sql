-- Add missing indexes for performance optimization

-- Additional Teams table indexes (name index missing from initial migration)
CREATE INDEX IF NOT EXISTS idx_teams_name ON teams(name);

-- Additional Events table indexes (missing from initial migration)
CREATE INDEX IF NOT EXISTS idx_events_external_id ON events(external_id);
CREATE INDEX IF NOT EXISTS idx_events_home_team_id ON events(home_team_id);
CREATE INDEX IF NOT EXISTS idx_events_away_team_id ON events(away_team_id);
CREATE INDEX IF NOT EXISTS idx_events_sport_id ON events(sport_id);
CREATE INDEX IF NOT EXISTS idx_events_composite ON events(sport_id, status, event_date);

-- Additional Market types table indexes (missing from initial migration)
CREATE INDEX IF NOT EXISTS idx_market_types_market_type ON market_types(market_type);
CREATE INDEX IF NOT EXISTS idx_market_types_market_sub_type ON market_types(market_sub_type);

-- Additional Current odds table indexes (missing from initial migration)
CREATE INDEX IF NOT EXISTS idx_current_odds_event_id ON current_odds(event_id);
CREATE INDEX IF NOT EXISTS idx_current_odds_market_type_id ON current_odds(market_type_id);
-- Composite index for GetCurrentOddsByMarket query performance
CREATE INDEX IF NOT EXISTS idx_current_odds_event_market_composite ON current_odds(event_id, market_type_id);

-- Odds history table indexes (missing from initial migration)
CREATE INDEX IF NOT EXISTS idx_odds_history_event_id ON odds_history(event_id);
CREATE INDEX IF NOT EXISTS idx_odds_history_market_type_id ON odds_history(market_type_id);
CREATE INDEX IF NOT EXISTS idx_odds_history_composite ON odds_history(event_id, market_type_id, outcome);

-- Team mappings indexes (missing from initial migration)
CREATE INDEX IF NOT EXISTS idx_team_mappings_internal_team_id ON team_mappings(internal_team_id);
CREATE INDEX IF NOT EXISTS idx_team_mappings_football_api_team_id ON team_mappings(football_api_team_id);