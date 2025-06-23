-- Drop all views and materialized views in reverse order of creation
-- Also drop associated indexes

-- Drop indexes first
DROP INDEX IF EXISTS idx_outcome_distributions_public_bias;
DROP INDEX IF EXISTS idx_odds_history_reverse_movement;
DROP INDEX IF EXISTS idx_odds_history_sharp_money;
DROP INDEX IF EXISTS idx_events_betting_volume;
DROP INDEX IF EXISTS idx_events_volume_rank;
DROP INDEX IF EXISTS idx_current_odds_movement_percentage;

-- Drop materialized view indexes
DROP INDEX IF EXISTS idx_big_movers_event_id;
DROP INDEX IF EXISTS idx_big_movers_movement;
DROP INDEX IF EXISTS idx_big_movers_sport;
DROP INDEX IF EXISTS idx_contrarian_bets_sport;
DROP INDEX IF EXISTS idx_contrarian_bets_signal_strength;
DROP INDEX IF EXISTS idx_contrarian_bets_event_id;
DROP INDEX IF EXISTS idx_sharp_money_moves_event_id;
DROP INDEX IF EXISTS idx_sharp_money_moves_indicator;
DROP INDEX IF EXISTS idx_sharp_money_moves_sport;
DROP INDEX IF EXISTS idx_live_opportunities_event_id;
DROP INDEX IF EXISTS idx_live_opportunities_movement;
DROP INDEX IF EXISTS idx_live_opportunities_sport;
DROP INDEX IF EXISTS idx_value_spots_event_id;
DROP INDEX IF EXISTS idx_value_spots_value_score;
DROP INDEX IF EXISTS idx_value_spots_sport;
DROP INDEX IF EXISTS idx_high_volume_events_event_id;
DROP INDEX IF EXISTS idx_high_volume_events_volume;
DROP INDEX IF EXISTS idx_high_volume_events_sport;

-- Drop all materialized views
DROP MATERIALIZED VIEW IF EXISTS high_volume_events CASCADE;
DROP MATERIALIZED VIEW IF EXISTS value_spots CASCADE;
DROP MATERIALIZED VIEW IF EXISTS live_opportunities CASCADE;
DROP MATERIALIZED VIEW IF EXISTS sharp_money_moves CASCADE;
DROP MATERIALIZED VIEW IF EXISTS big_movers CASCADE;
DROP MATERIALIZED VIEW IF EXISTS contrarian_bets CASCADE;