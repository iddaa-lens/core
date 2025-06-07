-- Drop Smart Money Tracker schema

-- Drop indexes
DROP INDEX IF EXISTS idx_user_alert_preferences_user_id;
DROP INDEX IF EXISTS idx_movement_alerts_active;
DROP INDEX IF EXISTS idx_movement_alerts_severity;
DROP INDEX IF EXISTS idx_movement_alerts_type;
DROP INDEX IF EXISTS idx_volume_patterns_recorded_at;
DROP INDEX IF EXISTS idx_volume_patterns_event_id;
DROP INDEX IF EXISTS idx_odds_movements_composite;
DROP INDEX IF EXISTS idx_odds_movements_sharp_money;
DROP INDEX IF EXISTS idx_odds_movements_significance;
DROP INDEX IF EXISTS idx_odds_movements_detected_at;
DROP INDEX IF EXISTS idx_odds_movements_event_id;

-- Drop views
DROP VIEW IF EXISTS sharp_money_spots;
DROP VIEW IF EXISTS reverse_line_movements;
DROP VIEW IF EXISTS big_movers;

-- Drop tables
DROP TABLE IF EXISTS user_alert_preferences;
DROP TABLE IF EXISTS movement_alerts;
DROP TABLE IF EXISTS volume_patterns;
DROP TABLE IF EXISTS odds_movements;