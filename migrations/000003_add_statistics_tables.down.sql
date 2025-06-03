-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_match_statistics_updated_at ON match_statistics;
DROP FUNCTION IF EXISTS update_match_statistics_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_events_is_live;
DROP INDEX IF EXISTS idx_match_events_event_type;
DROP INDEX IF EXISTS idx_match_events_event_id;
DROP INDEX IF EXISTS idx_match_statistics_event_id;

-- Drop tables
DROP TABLE IF EXISTS match_events;
DROP TABLE IF EXISTS match_statistics;

-- Remove columns from events table
ALTER TABLE events 
DROP COLUMN IF EXISTS half,
DROP COLUMN IF EXISTS minute_of_match,
DROP COLUMN IF EXISTS is_live;