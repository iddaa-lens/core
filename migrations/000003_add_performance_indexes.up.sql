-- Performance indexes for high-frequency queries

-- Events indexes
CREATE INDEX IF NOT EXISTS idx_events_event_date ON events(event_date);
CREATE INDEX IF NOT EXISTS idx_events_status ON events(status);
CREATE INDEX IF NOT EXISTS idx_events_league_id ON events(league_id);
CREATE INDEX IF NOT EXISTS idx_events_composite ON events(event_date, status, league_id);
CREATE INDEX IF NOT EXISTS idx_events_live ON events(is_live) WHERE is_live = true;

-- Odds history indexes for sharp money tracking
CREATE INDEX IF NOT EXISTS idx_odds_history_event_id ON odds_history(event_id);
CREATE INDEX IF NOT EXISTS idx_odds_history_recorded_at ON odds_history(recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_odds_history_composite ON odds_history(event_id, market_type_id, outcome, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_odds_history_sharp_indicator ON odds_history(sharp_money_indicator DESC) WHERE sharp_money_indicator > 0.5;
CREATE INDEX IF NOT EXISTS idx_odds_history_reverse ON odds_history(event_id, is_reverse_movement) WHERE is_reverse_movement = true;

-- Current odds indexes
CREATE INDEX IF NOT EXISTS idx_current_odds_event_id ON current_odds(event_id);
CREATE INDEX IF NOT EXISTS idx_current_odds_last_updated ON current_odds(last_updated DESC);
CREATE INDEX IF NOT EXISTS idx_current_odds_movement ON current_odds(movement_percentage DESC);

-- Outcome distributions indexes
CREATE INDEX IF NOT EXISTS idx_outcome_dist_event_id ON outcome_distributions(event_id);
CREATE INDEX IF NOT EXISTS idx_outcome_dist_bet_percentage ON outcome_distributions(bet_percentage DESC);

-- Movement alerts indexes
CREATE INDEX IF NOT EXISTS idx_movement_alerts_created_at ON movement_alerts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_movement_alerts_active ON movement_alerts(is_active, expires_at) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_movement_alerts_type ON movement_alerts(alert_type, severity);

-- Team and league lookup indexes
CREATE INDEX IF NOT EXISTS idx_teams_slug ON teams(slug);
CREATE INDEX IF NOT EXISTS idx_leagues_slug ON leagues(slug);
CREATE INDEX IF NOT EXISTS idx_leagues_sport_id ON leagues(sport_id);

-- Betting volume indexes
CREATE INDEX IF NOT EXISTS idx_betting_volume_event_id ON betting_volume_history(event_id);
CREATE INDEX IF NOT EXISTS idx_betting_volume_recorded_at ON betting_volume_history(recorded_at DESC);

-- API Football mapping indexes
CREATE INDEX IF NOT EXISTS idx_league_mappings_confidence ON league_mappings(confidence DESC);
CREATE INDEX IF NOT EXISTS idx_team_mappings_confidence ON team_mappings(confidence DESC);