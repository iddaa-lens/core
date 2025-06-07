-- Smart Money Tracker: View-Based Design
-- Leverages existing odds_history, current_odds, outcome_distributions, and betting_volume_history tables

-- Add smart money analysis columns to existing odds_history table
ALTER TABLE odds_history ADD COLUMN IF NOT EXISTS sharp_money_indicator DECIMAL(3, 2) DEFAULT 0.0;
ALTER TABLE odds_history ADD COLUMN IF NOT EXISTS is_reverse_movement BOOLEAN DEFAULT false;
ALTER TABLE odds_history ADD COLUMN IF NOT EXISTS significance_level VARCHAR(20) DEFAULT 'normal';
ALTER TABLE odds_history ADD COLUMN IF NOT EXISTS minutes_to_kickoff INTEGER;

-- Minimal additions to existing schema - only for alert system and user preferences

-- Table for movement alerts (references odds_history IDs)
CREATE TABLE IF NOT EXISTS movement_alerts (
    id SERIAL PRIMARY KEY,
    odds_history_id INTEGER NOT NULL REFERENCES odds_history(id) ON DELETE CASCADE,
    
    -- Alert details
    alert_type VARCHAR(50) NOT NULL, -- 'big_mover', 'reverse_line', 'sharp_money', 'value_spot'
    severity VARCHAR(20) NOT NULL, -- 'low', 'medium', 'high', 'critical'
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    
    -- Movement context (calculated at alert time)
    change_percentage DECIMAL(10, 2) NOT NULL,
    multiplier DECIMAL(10, 3) NOT NULL,
    confidence_score DECIMAL(3, 2) DEFAULT 0.5,
    minutes_to_kickoff INTEGER,
    
    -- Alert metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP + INTERVAL '24 hours'),
    is_active BOOLEAN DEFAULT true,
    
    -- Engagement tracking
    views INTEGER DEFAULT 0,
    clicks INTEGER DEFAULT 0,
    
    -- Prevent duplicate alerts for same movement
    UNIQUE(odds_history_id, alert_type)
);

-- User preferences for smart money alerts
CREATE TABLE IF NOT EXISTS smart_money_preferences (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL, -- Will integrate with user system later
    
    -- Movement thresholds
    min_change_percentage DECIMAL(5, 2) DEFAULT 20.0,
    min_multiplier DECIMAL(4, 2) DEFAULT 2.0,
    min_confidence_score DECIMAL(3, 2) DEFAULT 0.5,
    
    -- Alert types enabled
    big_mover_alerts BOOLEAN DEFAULT true,
    reverse_line_alerts BOOLEAN DEFAULT true,
    sharp_money_alerts BOOLEAN DEFAULT false, -- Premium feature
    value_spot_alerts BOOLEAN DEFAULT false,  -- Premium feature
    
    -- Targeting preferences
    preferred_sports JSONB DEFAULT '[]', -- array of sport IDs
    preferred_leagues JSONB DEFAULT '[]', -- array of league IDs
    max_daily_alerts INTEGER DEFAULT 50,
    
    -- Notification settings
    push_notifications BOOLEAN DEFAULT true,
    quiet_hours_start TIME DEFAULT '23:00',
    quiet_hours_end TIME DEFAULT '07:00',
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(user_id)
);

-- Drop existing views if they exist to avoid conflicts
DROP VIEW IF EXISTS big_movers CASCADE;
DROP VIEW IF EXISTS reverse_line_movements CASCADE;
DROP VIEW IF EXISTS sharp_money_spots CASCADE;
DROP VIEW IF EXISTS value_spots CASCADE;

-- Views for quick analytics using existing odds_history table

-- Big movers view (movements > 20% or 2x multiplier) 
CREATE VIEW big_movers AS
SELECT 
    oh.*,
    e.external_id as event_external_id,
    e.home_team_id,
    e.away_team_id, 
    e.event_date,
    e.status as event_status,
    e.is_live,
    mt.code as market_code,
    mt.name as market_name,
    l.name as league_name,
    s.name as sport_name,
    ht.name as home_team_name,
    at.name as away_team_name,
    EXTRACT(EPOCH FROM (e.event_date - oh.recorded_at))/60 as minutes_to_kickoff_calc,
    'Big odds movement detected' as alert_message
FROM odds_history oh
JOIN events e ON oh.event_id = e.id
JOIN market_types mt ON oh.market_type_id = mt.id
LEFT JOIN leagues l ON e.league_id = l.id
LEFT JOIN sports s ON e.sport_id = s.id
LEFT JOIN teams ht ON e.home_team_id = ht.id
LEFT JOIN teams at ON e.away_team_id = at.id
WHERE 
    (ABS(oh.change_percentage) >= 20 OR oh.multiplier >= 2.0)
    AND oh.recorded_at >= NOW() - INTERVAL '24 hours'
    AND e.event_date > NOW()
ORDER BY oh.recorded_at DESC;

-- Reverse line movements view
CREATE VIEW reverse_line_movements AS
SELECT 
    oh.*,
    e.external_id as event_external_id,
    e.event_date,
    e.is_live,
    mt.code as market_code,
    mt.name as market_name,
    ht.name as home_team_name,
    at.name as away_team_name,
    'Reverse line movement detected' as alert_message
FROM odds_history oh
JOIN events e ON oh.event_id = e.id
JOIN market_types mt ON oh.market_type_id = mt.id
LEFT JOIN teams ht ON e.home_team_id = ht.id
LEFT JOIN teams at ON e.away_team_id = at.id
WHERE 
    oh.is_reverse_movement = true
    AND oh.recorded_at >= NOW() - INTERVAL '6 hours'
    AND e.event_date > NOW()
ORDER BY oh.sharp_money_indicator DESC, oh.recorded_at DESC;

-- Sharp money indicators view
CREATE VIEW sharp_money_spots AS
SELECT 
    oh.*,
    e.external_id as event_external_id,
    e.event_date,
    e.is_live,
    mt.code as market_code,
    mt.name as market_name,
    ht.name as home_team_name,
    at.name as away_team_name,
    CASE 
        WHEN oh.sharp_money_indicator >= 0.8 THEN 'Very High Confidence'
        WHEN oh.sharp_money_indicator >= 0.6 THEN 'High Confidence'
        WHEN oh.sharp_money_indicator >= 0.4 THEN 'Medium Confidence'
        ELSE 'Low Confidence'
    END as confidence_level,
    'Sharp money activity detected' as alert_message
FROM odds_history oh
JOIN events e ON oh.event_id = e.id
JOIN market_types mt ON oh.market_type_id = mt.id
LEFT JOIN teams ht ON e.home_team_id = ht.id
LEFT JOIN teams at ON e.away_team_id = at.id
WHERE 
    oh.sharp_money_indicator >= 0.4
    AND oh.recorded_at >= NOW() - INTERVAL '12 hours'
    AND e.event_date > NOW()
ORDER BY oh.sharp_money_indicator DESC, oh.recorded_at DESC;

-- Value betting opportunities (combining odds movement with betting distributions)
CREATE VIEW value_spots AS
SELECT 
    oh.*,
    od.bet_percentage,
    od.implied_probability,
    e.external_id as event_external_id,
    e.event_date,
    mt.code as market_code,
    mt.name as market_name,
    ht.name as home_team_name,
    at.name as away_team_name,
    (od.bet_percentage - od.implied_probability) as public_bias,
    'Value betting opportunity detected' as alert_message
FROM odds_history oh
JOIN events e ON oh.event_id = e.id
JOIN market_types mt ON oh.market_type_id = mt.id
LEFT JOIN teams ht ON e.home_team_id = ht.id
LEFT JOIN teams at ON e.away_team_id = at.id
LEFT JOIN outcome_distributions od ON (
    oh.event_id = od.event_id 
    AND oh.market_type_id = od.market_type_id 
    AND oh.outcome = od.outcome
)
WHERE 
    oh.recorded_at >= NOW() - INTERVAL '6 hours'
    AND e.event_date > NOW()
    AND od.bet_percentage > od.implied_probability + 10
    AND ABS(oh.change_percentage) >= 15
ORDER BY (od.bet_percentage - od.implied_probability) DESC;

-- Indexes for performance on new columns
CREATE INDEX IF NOT EXISTS idx_odds_history_sharp_money ON odds_history(sharp_money_indicator DESC) WHERE sharp_money_indicator > 0;
CREATE INDEX IF NOT EXISTS idx_odds_history_reverse_movement ON odds_history(is_reverse_movement) WHERE is_reverse_movement = true;
CREATE INDEX IF NOT EXISTS idx_odds_history_significance ON odds_history(significance_level) WHERE significance_level != 'normal';

CREATE INDEX IF NOT EXISTS idx_movement_alerts_odds_history_id ON movement_alerts(odds_history_id);
CREATE INDEX IF NOT EXISTS idx_movement_alerts_type ON movement_alerts(alert_type);
CREATE INDEX IF NOT EXISTS idx_movement_alerts_severity ON movement_alerts(severity);
CREATE INDEX IF NOT EXISTS idx_movement_alerts_active ON movement_alerts(is_active, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_smart_money_preferences_user_id ON smart_money_preferences(user_id);