-- Performance indexes for high-frequency queries on events, teams, leagues, current_odds, and odds_history
-- ====================
-- EVENTS TABLE INDEXES
-- ====================
-- 1. Index for GetActiveEventsForDetailedSync and GetAllActiveEventsForDetailedSync
-- Partial index for active events (without time-based filtering due to immutability requirement)
CREATE INDEX IF NOT EXISTS idx_events_active_sync ON events(status, event_date)
WHERE status IN ('live', 'scheduled');

-- 2. Composite index for ListEventsFiltered with multiple filter conditions
CREATE INDEX IF NOT EXISTS idx_events_filter_composite ON events(event_date, status, sport_id, league_id);

-- 3. Index for GetEventsByTeam queries (both home and away)
CREATE INDEX IF NOT EXISTS idx_events_home_team_date ON events(home_team_id, event_date DESC);

CREATE INDEX IF NOT EXISTS idx_events_away_team_date ON events(away_team_id, event_date DESC);

-- 4. Index for GetEventByExternalID (frequently used in upserts)
CREATE INDEX IF NOT EXISTS idx_events_external_id ON events(external_id);

-- 5. Index for live events filtering
CREATE INDEX IF NOT EXISTS idx_events_live_status ON events(is_live, status, event_date)
WHERE
  is_live = true;

-- 6. Index for events by date (supports ListEventsByDate)
CREATE INDEX IF NOT EXISTS idx_events_date_only ON events(DATE(event_date));

-- ====================
-- TEAMS TABLE INDEXES
-- ====================
-- 7. Index for GetTeamByExternalID (used in bulk upserts)
CREATE INDEX IF NOT EXISTS idx_teams_external_id ON teams(external_id);

-- 8. Index for team name searches
CREATE INDEX IF NOT EXISTS idx_teams_name_pattern ON teams(name varchar_pattern_ops);

-- 9. Index for API Football ID lookups
CREATE INDEX IF NOT EXISTS idx_teams_api_football_id ON teams(api_football_id)
WHERE
  api_football_id IS NOT NULL;

-- 10. Index for teams needing enrichment
CREATE INDEX IF NOT EXISTS idx_teams_enrichment_needed ON teams(last_api_update NULLS FIRST)
WHERE
  api_football_id IS NULL
  OR last_api_update IS NULL;

-- ====================
-- LEAGUES TABLE INDEXES
-- ====================
-- 11. Index for GetLeagueByExternalID
CREATE INDEX IF NOT EXISTS idx_leagues_external_id ON leagues(external_id);

-- 12. Index for leagues by sport
CREATE INDEX IF NOT EXISTS idx_leagues_sport_active ON leagues(sport_id, is_active)
WHERE
  is_active = true;

-- 13. Index for API Football league lookups
CREATE INDEX IF NOT EXISTS idx_leagues_api_football_id ON leagues(api_football_id)
WHERE
  api_football_id IS NOT NULL;

-- 14. Index for leagues needing enrichment
CREATE INDEX IF NOT EXISTS idx_leagues_enrichment_needed ON leagues(last_api_update NULLS FIRST);

-- ====================
-- CURRENT_ODDS TABLE INDEXES
-- ====================
-- 15. Composite index for BulkGetCurrentOddsForComparison
-- This is the most critical index for odds processing performance
CREATE INDEX IF NOT EXISTS idx_current_odds_bulk_lookup ON current_odds(event_id, market_type_id, outcome) INCLUDE (
  odds_value,
  opening_value,
  highest_value,
  lowest_value
);

-- 16. Index for odds movements tracking
CREATE INDEX IF NOT EXISTS idx_current_odds_movement ON current_odds(
  movement_percentage DESC NULLS LAST,
  last_updated DESC
)
WHERE
  movement_percentage IS NOT NULL;

-- 17. Index for recent odds updates
CREATE INDEX IF NOT EXISTS idx_current_odds_last_updated ON current_odds(last_updated DESC);

-- ====================
-- ODDS_HISTORY TABLE INDEXES
-- ====================
-- 18. Composite index for GetOddsMovements and event-specific history
CREATE INDEX IF NOT EXISTS idx_odds_history_event_time ON odds_history(event_id, recorded_at DESC) INCLUDE (
  market_type_id,
  outcome,
  odds_value,
  change_percentage
);

-- 19. Index for GetBigMovers and significant odds changes
CREATE INDEX IF NOT EXISTS idx_odds_history_big_movers ON odds_history(abs(change_percentage) DESC, recorded_at DESC)
WHERE
  abs(change_percentage) > 5.0;

-- 20. Index for GetRecentOddsHistory with change percentage filter
CREATE INDEX IF NOT EXISTS idx_odds_history_recent_changes ON odds_history(recorded_at DESC, abs(change_percentage))
WHERE
  abs(change_percentage) >= 5.0;

-- 21. Index for sharp money detection
CREATE INDEX IF NOT EXISTS idx_odds_history_sharp_money ON odds_history(sharp_money_indicator DESC, recorded_at DESC)
WHERE
  sharp_money_indicator > 0.5;

-- 22. Index for reverse movements
CREATE INDEX IF NOT EXISTS idx_odds_history_reverse_movements ON odds_history(event_id, is_reverse_movement, recorded_at DESC)
WHERE
  is_reverse_movement = true;

-- 23. Index for significance level filtering
CREATE INDEX IF NOT EXISTS idx_odds_history_significance ON odds_history(significance_level, recorded_at DESC)
WHERE
  significance_level IN ('high', 'extreme');

-- 24. Composite index for full history lookup pattern
CREATE INDEX IF NOT EXISTS idx_odds_history_composite ON odds_history(
  event_id,
  market_type_id,
  outcome,
  recorded_at DESC
);

-- ====================
-- ADDITIONAL SUPPORTING INDEXES
-- ====================
-- 25. Market types code lookup (used in odds processing)
CREATE INDEX IF NOT EXISTS idx_market_types_code ON market_types(code);

-- 26. League mappings for unmapped leagues query
CREATE INDEX IF NOT EXISTS idx_league_mappings_internal ON league_mappings(internal_league_id);

-- 27. Team mappings for unmapped teams query
CREATE INDEX IF NOT EXISTS idx_team_mappings_internal ON team_mappings(internal_team_id);

-- Update table statistics for query planner optimization
ANALYZE events;

ANALYZE teams;

ANALYZE leagues;

ANALYZE current_odds;

ANALYZE odds_history;

ANALYZE market_types;

ANALYZE league_mappings;

ANALYZE team_mappings;