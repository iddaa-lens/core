-- Add columns to events table for live match tracking
ALTER TABLE events 
ADD COLUMN IF NOT EXISTS is_live BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS minute_of_match INTEGER,
ADD COLUMN IF NOT EXISTS half INTEGER DEFAULT 0;

-- Create match_statistics table for detailed match statistics
CREATE TABLE IF NOT EXISTS match_statistics (
    id SERIAL PRIMARY KEY,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    is_home BOOLEAN NOT NULL, -- true for home team, false for away team
    shots INTEGER DEFAULT 0,
    shots_on_target INTEGER DEFAULT 0,
    possession INTEGER DEFAULT 0, -- percentage
    corners INTEGER DEFAULT 0,
    yellow_cards INTEGER DEFAULT 0,
    red_cards INTEGER DEFAULT 0,
    fouls INTEGER DEFAULT 0,
    offsides INTEGER DEFAULT 0,
    free_kicks INTEGER DEFAULT 0,
    throw_ins INTEGER DEFAULT 0,
    goal_kicks INTEGER DEFAULT 0,
    saves INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(event_id, is_home) -- One row per team per match
);

-- Create match_events table for individual match events (goals, cards, etc.)
CREATE TABLE IF NOT EXISTS match_events (
    id SERIAL PRIMARY KEY,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    minute INTEGER NOT NULL,
    event_type VARCHAR(50) NOT NULL, -- 'goal', 'yellow_card', 'red_card', 'substitution', etc.
    team VARCHAR(255) NOT NULL,
    player VARCHAR(255),
    description TEXT NOT NULL,
    is_home BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(event_id, minute, event_type, team, player) -- Prevent duplicate events
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_match_statistics_event_id ON match_statistics(event_id);
CREATE INDEX IF NOT EXISTS idx_match_events_event_id ON match_events(event_id);
CREATE INDEX IF NOT EXISTS idx_match_events_event_type ON match_events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_is_live ON events(is_live);

-- Add trigger to update updated_at for match_statistics
CREATE OR REPLACE FUNCTION update_match_statistics_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_match_statistics_updated_at
    BEFORE UPDATE ON match_statistics
    FOR EACH ROW
    EXECUTE FUNCTION update_match_statistics_updated_at();