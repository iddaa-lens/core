-- Drop all objects in reverse order

-- Drop views
DROP VIEW IF EXISTS contrarian_bets;
DROP VIEW IF EXISTS big_movers;

-- Drop triggers
DROP TRIGGER IF EXISTS update_team_mappings_updated_at ON team_mappings;
DROP TRIGGER IF EXISTS update_league_mappings_updated_at ON league_mappings;
DROP TRIGGER IF EXISTS update_match_statistics_updated_at ON match_statistics;
DROP TRIGGER IF EXISTS update_app_config_updated_at ON app_config;
DROP TRIGGER IF EXISTS update_market_types_updated_at ON market_types;
DROP TRIGGER IF EXISTS update_events_updated_at ON events;
DROP TRIGGER IF EXISTS update_teams_updated_at ON teams;
DROP TRIGGER IF EXISTS update_leagues_updated_at ON leagues;

DROP TRIGGER IF EXISTS auto_generate_market_type_slug ON market_types;
DROP TRIGGER IF EXISTS auto_generate_event_slug ON events;
DROP TRIGGER IF EXISTS auto_generate_team_slug ON teams;
DROP TRIGGER IF EXISTS auto_generate_league_slug ON leagues;

-- Drop functions
DROP FUNCTION IF EXISTS update_updated_at();
DROP FUNCTION IF EXISTS generate_market_type_slug();
DROP FUNCTION IF EXISTS generate_event_slug();
DROP FUNCTION IF EXISTS generate_team_slug();
DROP FUNCTION IF EXISTS generate_league_slug();

-- Drop tables (in reverse dependency order)
DROP TABLE IF EXISTS team_mappings CASCADE;
DROP TABLE IF EXISTS league_mappings CASCADE;
DROP TABLE IF EXISTS app_config CASCADE;
DROP TABLE IF EXISTS match_statistics CASCADE;
DROP TABLE IF EXISTS match_events CASCADE;
DROP TABLE IF EXISTS predictions CASCADE;
DROP TABLE IF EXISTS outcome_distribution_history CASCADE;
DROP TABLE IF EXISTS outcome_distributions CASCADE;
DROP TABLE IF EXISTS betting_volume_history CASCADE;
DROP TABLE IF EXISTS odds_history CASCADE;
DROP TABLE IF EXISTS current_odds CASCADE;
DROP TABLE IF EXISTS market_types CASCADE;
DROP TABLE IF EXISTS events CASCADE;
DROP TABLE IF EXISTS teams CASCADE;
DROP TABLE IF EXISTS leagues CASCADE;
DROP TABLE IF EXISTS sports CASCADE;