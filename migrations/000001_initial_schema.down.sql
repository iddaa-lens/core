-- Drop functions
DROP FUNCTION IF EXISTS analyze_betting_patterns(INTEGER);
DROP FUNCTION IF EXISTS get_biggest_movers(INTEGER, DECIMAL);

-- Drop materialized views
DROP MATERIALIZED VIEW IF EXISTS contrarian_bets;
DROP MATERIALIZED VIEW IF EXISTS volume_trends;

-- Drop views
DROP VIEW IF EXISTS value_opportunities;
DROP VIEW IF EXISTS popular_events_with_movements;
DROP VIEW IF EXISTS suspicious_movements;
DROP VIEW IF EXISTS big_movers;

-- Drop triggers
DROP TRIGGER IF EXISTS update_app_config_updated_at ON app_config;
DROP TRIGGER IF EXISTS update_market_types_updated_at ON market_types;
DROP TRIGGER IF EXISTS update_events_updated_at ON events;
DROP TRIGGER IF EXISTS update_teams_updated_at ON teams;
DROP TRIGGER IF EXISTS update_competitions_updated_at ON competitions;
DROP TRIGGER IF EXISTS update_sports_updated_at ON sports;

DROP TRIGGER IF EXISTS generate_prediction_slug_trigger ON predictions;
DROP TRIGGER IF EXISTS generate_market_type_slug_trigger ON market_types;
DROP TRIGGER IF EXISTS generate_event_slug_trigger ON events;
DROP TRIGGER IF EXISTS generate_competition_slug_trigger ON competitions;
DROP TRIGGER IF EXISTS generate_team_slug_trigger ON teams;
DROP TRIGGER IF EXISTS generate_sport_slug_trigger ON sports;

-- Drop trigger functions
DROP FUNCTION IF EXISTS update_updated_at();
DROP FUNCTION IF EXISTS trigger_generate_prediction_slug();
DROP FUNCTION IF EXISTS trigger_generate_market_type_slug();
DROP FUNCTION IF EXISTS trigger_generate_event_slug();
DROP FUNCTION IF EXISTS trigger_generate_competition_slug();
DROP FUNCTION IF EXISTS trigger_generate_team_slug();
DROP FUNCTION IF EXISTS trigger_generate_sport_slug();
DROP FUNCTION IF EXISTS generate_slug(TEXT);

-- Drop tables in correct order (due to foreign keys)
DROP TABLE IF EXISTS outcome_distribution_history CASCADE;
DROP TABLE IF EXISTS outcome_distributions CASCADE;
DROP TABLE IF EXISTS betting_volume_history CASCADE;
DROP TABLE IF EXISTS odds_history CASCADE;
DROP TABLE IF EXISTS current_odds CASCADE;
DROP TABLE IF EXISTS predictions CASCADE;
DROP TABLE IF EXISTS market_types CASCADE;
DROP TABLE IF EXISTS events CASCADE;
DROP TABLE IF EXISTS teams CASCADE;
DROP TABLE IF EXISTS competitions CASCADE;
DROP TABLE IF EXISTS sports CASCADE;
DROP TABLE IF EXISTS app_config CASCADE;

-- Drop extension
DROP EXTENSION IF EXISTS unaccent;