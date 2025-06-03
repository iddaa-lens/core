-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "unaccent";

-- Function to generate slugs
CREATE OR REPLACE FUNCTION generate_slug(input_text TEXT) 
RETURNS TEXT AS $$
BEGIN
    -- Convert to lowercase, remove accents, replace spaces and special chars with hyphens
    RETURN TRIM(
        REGEXP_REPLACE(
            REGEXP_REPLACE(
                REGEXP_REPLACE(
                    REGEXP_REPLACE(
                        LOWER(unaccent(input_text)),
                        '[^a-zA-Z0-9\s-]', '', 'g'  -- Remove special characters
                    ),
                    '\s+', '-', 'g'  -- Replace spaces with hyphens
                ),
                '-+', '-', 'g'  -- Replace multiple hyphens with single hyphen
            ),
            '^-+|-+$', '', 'g'  -- Remove leading and trailing hyphens
        ),
        '-'  -- Trim any remaining hyphens from both ends
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Sports mapping
CREATE TABLE IF NOT EXISTS sports (
    id INTEGER PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(20) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    live_count INTEGER DEFAULT 0,
    upcoming_count INTEGER DEFAULT 0,
    events_count INTEGER DEFAULT 0,
    odds_count INTEGER DEFAULT 0,
    has_results BOOLEAN DEFAULT false,
    has_king_odd BOOLEAN DEFAULT false,
    has_digital_content BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Competitions (leagues, tournaments)
CREATE TABLE IF NOT EXISTS competitions (
    id SERIAL PRIMARY KEY,
    iddaa_id INTEGER UNIQUE NOT NULL,
    external_ref INTEGER,
    country_code VARCHAR(10),
    parent_id INTEGER,
    sport_id INTEGER REFERENCES sports(id),
    short_name VARCHAR(100),
    full_name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    icon_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_competitions_sport_country ON competitions (sport_id, country_code);
CREATE INDEX IF NOT EXISTS idx_competitions_iddaa_id ON competitions (iddaa_id);
CREATE INDEX IF NOT EXISTS idx_competitions_slug ON competitions (slug);

-- Teams
CREATE TABLE IF NOT EXISTS teams (
    id SERIAL PRIMARY KEY,
    external_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    short_name VARCHAR(100),
    country VARCHAR(100),
    logo_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_teams_slug ON teams (slug);

-- Events (matches/games) with volume tracking
CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    external_id VARCHAR(255) UNIQUE NOT NULL,
    competition_id INTEGER REFERENCES competitions(id),
    home_team_id INTEGER REFERENCES teams(id),
    away_team_id INTEGER REFERENCES teams(id),
    slug VARCHAR(500) UNIQUE NOT NULL, -- Longer for team-vs-team-date format
    event_date TIMESTAMP NOT NULL,
    status VARCHAR(50) NOT NULL, -- scheduled, live, finished, cancelled
    home_score INTEGER,
    away_score INTEGER,
    betting_volume_percentage DECIMAL(5, 2),
    volume_rank INTEGER,
    volume_updated_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_events_slug ON events (slug);
CREATE INDEX IF NOT EXISTS idx_events_date ON events (event_date);
CREATE INDEX IF NOT EXISTS idx_events_status ON events (status);

-- Market types (1X2, Over/Under, etc.)
CREATE TABLE IF NOT EXISTS market_types (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_market_types_slug ON market_types (slug);

-- Current odds (latest values for fast lookup)
CREATE TABLE IF NOT EXISTS current_odds (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(100) NOT NULL,
    odds_value DECIMAL(10, 3) NOT NULL,
    opening_value DECIMAL(10, 3) NOT NULL, -- First recorded odds
    highest_value DECIMAL(10, 3) NOT NULL,  -- Track max for movement detection
    lowest_value DECIMAL(10, 3) NOT NULL,   -- Track min for movement detection
    total_movement DECIMAL(10, 3) GENERATED ALWAYS AS (highest_value - lowest_value) STORED,
    movement_percentage DECIMAL(10, 2) GENERATED ALWAYS AS (
        CASE 
            WHEN opening_value > 0 THEN ((odds_value - opening_value) / opening_value * 100)
            ELSE 0
        END
    ) STORED,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(event_id, market_type_id, outcome)
);

CREATE INDEX IF NOT EXISTS idx_current_odds_movement ON current_odds (total_movement DESC);
CREATE INDEX IF NOT EXISTS idx_current_odds_movement_pct ON current_odds (ABS(movement_percentage) DESC);

-- Odds history (only changes)
CREATE TABLE IF NOT EXISTS odds_history (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(100) NOT NULL,
    odds_value DECIMAL(10, 3) NOT NULL,
    previous_value DECIMAL(10, 3),
    change_amount DECIMAL(10, 3) GENERATED ALWAYS AS (odds_value - COALESCE(previous_value, odds_value)) STORED,
    change_percentage DECIMAL(10, 2) GENERATED ALWAYS AS (
        CASE 
            WHEN previous_value IS NOT NULL AND previous_value > 0 
            THEN ((odds_value - previous_value) / previous_value * 100)
            ELSE 0
        END
    ) STORED,
    multiplier DECIMAL(10, 2) GENERATED ALWAYS AS (
        CASE 
            WHEN previous_value IS NOT NULL AND previous_value > 0 
            THEN (odds_value / previous_value)
            ELSE 1
        END
    ) STORED,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_odds_history_event_time ON odds_history (event_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_odds_history_big_changes ON odds_history (ABS(change_percentage) DESC) WHERE ABS(change_percentage) > 20;
CREATE INDEX IF NOT EXISTS idx_odds_history_multiplier ON odds_history (multiplier DESC) WHERE multiplier > 1.5;

-- Betting volume history
CREATE TABLE IF NOT EXISTS betting_volume_history (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    volume_percentage DECIMAL(5, 2) NOT NULL,
    rank_position INTEGER,
    total_events_tracked INTEGER,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_volume_history_event_time 
ON betting_volume_history (event_id, recorded_at DESC);

CREATE INDEX IF NOT EXISTS idx_volume_history_high_volume 
ON betting_volume_history (volume_percentage DESC) 
WHERE volume_percentage > 5.0;

-- Outcome betting distributions
CREATE TABLE IF NOT EXISTS outcome_distributions (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_id INTEGER NOT NULL, -- The market ID from iddaa (e.g., 40641924)
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(10) NOT NULL, -- '1', '2', '3', etc.
    bet_percentage DECIMAL(5, 2) NOT NULL, -- Percentage of bets on this outcome
    implied_probability DECIMAL(5, 2), -- Calculated from current odds
    value_indicator DECIMAL(5, 2) GENERATED ALWAYS AS (
        CASE 
            WHEN implied_probability > 0 AND bet_percentage > 0
            THEN (implied_probability - bet_percentage)
            ELSE 0
        END
    ) STORED, -- Positive = potential value (less bet than probability suggests)
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(event_id, market_id, outcome)
);

CREATE INDEX idx_outcome_dist_event ON outcome_distributions (event_id);
CREATE INDEX idx_outcome_dist_value ON outcome_distributions (ABS(value_indicator) DESC);

-- Historical tracking of distribution changes
CREATE TABLE IF NOT EXISTS outcome_distribution_history (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_id INTEGER NOT NULL,
    outcome VARCHAR(10) NOT NULL,
    bet_percentage DECIMAL(5, 2) NOT NULL,
    previous_percentage DECIMAL(5, 2),
    change_amount DECIMAL(5, 2) GENERATED ALWAYS AS (bet_percentage - COALESCE(previous_percentage, bet_percentage)) STORED,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_outcome_dist_history_event_time ON outcome_distribution_history (event_id, recorded_at DESC);

-- AI predictions
CREATE TABLE IF NOT EXISTS predictions (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id),
    market_type_id INTEGER REFERENCES market_types(id),
    slug VARCHAR(500) UNIQUE NOT NULL,
    predicted_outcome VARCHAR(100) NOT NULL,
    confidence_score DECIMAL(5, 4) NOT NULL, -- 0.0000 to 1.0000
    model_version VARCHAR(50) NOT NULL,
    features_used TEXT, -- JSON of features used
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_predictions_event_market ON predictions (event_id, market_type_id);
CREATE INDEX IF NOT EXISTS idx_predictions_confidence ON predictions (confidence_score);
CREATE INDEX IF NOT EXISTS idx_predictions_slug ON predictions (slug);

-- Configuration storage
CREATE TABLE IF NOT EXISTS app_config (
    id SERIAL PRIMARY KEY,
    platform VARCHAR(50) UNIQUE NOT NULL,
    config_data JSONB NOT NULL,
    sportoto_program_name VARCHAR(255),
    payin_end_date TIMESTAMP,
    next_draw_expected_win DECIMAL(15, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_app_config_updated_at ON app_config (updated_at);

-- Triggers for automatic slug generation
CREATE OR REPLACE FUNCTION trigger_generate_sport_slug() RETURNS TRIGGER AS $$
BEGIN
    NEW.slug := generate_slug(NEW.name);
    -- Ensure uniqueness by appending ID if needed
    IF EXISTS (SELECT 1 FROM sports WHERE slug = NEW.slug AND id != NEW.id) THEN
        NEW.slug := NEW.slug || '-' || NEW.id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION trigger_generate_team_slug() RETURNS TRIGGER AS $$
BEGIN
    NEW.slug := generate_slug(NEW.name);
    -- Ensure uniqueness by appending a counter if needed
    DECLARE
        counter INTEGER := 1;
        temp_slug TEXT;
    BEGIN
        temp_slug := NEW.slug;
        WHILE EXISTS (SELECT 1 FROM teams WHERE slug = temp_slug AND id != COALESCE(NEW.id, -1)) LOOP
            temp_slug := NEW.slug || '-' || counter;
            counter := counter + 1;
        END LOOP;
        NEW.slug := temp_slug;
    END;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION trigger_generate_competition_slug() RETURNS TRIGGER AS $$
BEGIN
    NEW.slug := generate_slug(NEW.full_name);
    -- Ensure uniqueness
    DECLARE
        counter INTEGER := 1;
        temp_slug TEXT;
    BEGIN
        temp_slug := NEW.slug;
        WHILE EXISTS (SELECT 1 FROM competitions WHERE slug = temp_slug AND id != COALESCE(NEW.id, -1)) LOOP
            temp_slug := NEW.slug || '-' || counter;
            counter := counter + 1;
        END LOOP;
        NEW.slug := temp_slug;
    END;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION trigger_generate_event_slug() RETURNS TRIGGER AS $$
DECLARE
    home_team_name TEXT;
    away_team_name TEXT;
    date_str TEXT;
BEGIN
    -- Get team names
    SELECT name INTO home_team_name FROM teams WHERE id = NEW.home_team_id;
    SELECT name INTO away_team_name FROM teams WHERE id = NEW.away_team_id;
    
    -- Format date as YYYY-MM-DD
    date_str := TO_CHAR(NEW.event_date, 'YYYY-MM-DD');
    
    -- Generate slug: home-team-vs-away-team-2025-06-05
    NEW.slug := generate_slug(home_team_name || ' vs ' || away_team_name || ' ' || date_str);
    
    -- Ensure uniqueness
    DECLARE
        counter INTEGER := 1;
        temp_slug TEXT;
    BEGIN
        temp_slug := NEW.slug;
        WHILE EXISTS (SELECT 1 FROM events WHERE slug = temp_slug AND id != COALESCE(NEW.id, -1)) LOOP
            temp_slug := NEW.slug || '-' || counter;
            counter := counter + 1;
        END LOOP;
        NEW.slug := temp_slug;
    END;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION trigger_generate_market_type_slug() RETURNS TRIGGER AS $$
BEGIN
    NEW.slug := generate_slug(NEW.name);
    -- Ensure uniqueness
    DECLARE
        counter INTEGER := 1;
        temp_slug TEXT;
    BEGIN
        temp_slug := NEW.slug;
        WHILE EXISTS (SELECT 1 FROM market_types WHERE slug = temp_slug AND id != COALESCE(NEW.id, -1)) LOOP
            temp_slug := NEW.slug || '-' || counter;
            counter := counter + 1;
        END LOOP;
        NEW.slug := temp_slug;
    END;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION trigger_generate_prediction_slug() RETURNS TRIGGER AS $$
DECLARE
    event_slug TEXT;
    market_slug TEXT;
BEGIN
    -- Get related slugs
    SELECT slug INTO event_slug FROM events WHERE id = NEW.event_id;
    SELECT slug INTO market_slug FROM market_types WHERE id = NEW.market_type_id;
    
    -- Generate slug: event-slug-market-slug-timestamp
    NEW.slug := event_slug || '-' || market_slug || '-' || EXTRACT(EPOCH FROM NEW.created_at)::INTEGER;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers
CREATE TRIGGER generate_sport_slug_trigger
    BEFORE INSERT OR UPDATE OF name ON sports
    FOR EACH ROW
    EXECUTE FUNCTION trigger_generate_sport_slug();

CREATE TRIGGER generate_team_slug_trigger
    BEFORE INSERT OR UPDATE OF name ON teams
    FOR EACH ROW
    EXECUTE FUNCTION trigger_generate_team_slug();

CREATE TRIGGER generate_competition_slug_trigger
    BEFORE INSERT OR UPDATE OF full_name ON competitions
    FOR EACH ROW
    EXECUTE FUNCTION trigger_generate_competition_slug();

CREATE TRIGGER generate_event_slug_trigger
    BEFORE INSERT OR UPDATE OF home_team_id, away_team_id, event_date ON events
    FOR EACH ROW
    EXECUTE FUNCTION trigger_generate_event_slug();

CREATE TRIGGER generate_market_type_slug_trigger
    BEFORE INSERT OR UPDATE OF name ON market_types
    FOR EACH ROW
    EXECUTE FUNCTION trigger_generate_market_type_slug();

CREATE TRIGGER generate_prediction_slug_trigger
    BEFORE INSERT ON predictions
    FOR EACH ROW
    EXECUTE FUNCTION trigger_generate_prediction_slug();

-- Update existing data triggers
CREATE OR REPLACE FUNCTION update_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_sports_updated_at BEFORE UPDATE ON sports FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER update_competitions_updated_at BEFORE UPDATE ON competitions FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER update_teams_updated_at BEFORE UPDATE ON teams FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER update_events_updated_at BEFORE UPDATE ON events FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER update_market_types_updated_at BEFORE UPDATE ON market_types FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER update_app_config_updated_at BEFORE UPDATE ON app_config FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Insert initial sports mapping
INSERT INTO sports (id, name, code) VALUES
(1, 'Football', 'FOOTBALL'),
(2, 'Basketball', 'BASKETBALL'),
(4, 'Ice Hockey', 'ICE_HOCKEY'),
(5, 'Tennis', 'TENNIS'),
(6, 'Handball', 'HANDBALL'),
(11, 'Formula 1', 'FORMULA1'),
(23, 'Other', 'OTHER')
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    code = EXCLUDED.code;

-- Insert common market types
INSERT INTO market_types (code, name, description) VALUES
('1X2', 'Match Result', 'Home Win, Draw, Away Win'),
('OU', 'Over/Under', 'Total goals over or under a threshold'),
('BTTS', 'Both Teams To Score', 'Both teams score yes/no'),
('HT', 'Half Time Result', 'Half time result'),
('DC', 'Double Chance', 'Two outcomes combined')
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description;

-- Views for finding big movements
CREATE OR REPLACE VIEW big_movers AS
SELECT 
    e.id as event_id,
    e.slug as event_slug,
    ht.name as home_team,
    at.name as away_team,
    e.event_date,
    e.status,
    mt.code as market_code,
    mt.name as market_name,
    co.outcome,
    co.opening_value,
    co.odds_value as current_value,
    co.highest_value,
    co.lowest_value,
    co.movement_percentage,
    co.odds_value / NULLIF(co.opening_value, 0) as multiplier,
    CASE 
        WHEN co.odds_value > co.opening_value THEN 'DRIFTING'
        WHEN co.odds_value < co.opening_value THEN 'SHORTENING'
        ELSE 'STABLE'
    END as trend_direction,
    co.last_updated
FROM current_odds co
JOIN events e ON co.event_id = e.id
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
JOIN market_types mt ON co.market_type_id = mt.id
WHERE ABS(co.movement_percentage) > 20  -- More than 20% movement
   OR co.odds_value / NULLIF(co.opening_value, 0) > 2  -- Odds doubled
   OR co.opening_value / NULLIF(co.odds_value, 0) > 2  -- Odds halved
ORDER BY ABS(co.movement_percentage) DESC;

-- View for tracking suspicious movements (potential insider activity)
CREATE OR REPLACE VIEW suspicious_movements AS
SELECT 
    e.slug as event_slug,
    mt.code as market_code,
    oh.outcome,
    oh.previous_value,
    oh.odds_value,
    oh.change_percentage,
    oh.multiplier,
    oh.recorded_at,
    COUNT(*) OVER (
        PARTITION BY oh.event_id, oh.market_type_id, oh.outcome 
        ORDER BY oh.recorded_at 
        RANGE BETWEEN INTERVAL '1 hour' PRECEDING AND CURRENT ROW
    ) as changes_last_hour
FROM odds_history oh
JOIN events e ON oh.event_id = e.id
JOIN market_types mt ON oh.market_type_id = mt.id
WHERE oh.multiplier > 1.5  -- 50% increase
   OR oh.multiplier < 0.67  -- 33% decrease
   OR ABS(oh.change_percentage) > 30  -- 30% swing
ORDER BY oh.recorded_at DESC;

-- View for popular events with odds movements
CREATE OR REPLACE VIEW popular_events_with_movements AS
SELECT 
    e.id,
    e.slug,
    ht.name || ' vs ' || at.name as match_name,
    e.event_date,
    e.betting_volume_percentage,
    e.volume_rank,
    co.movement_stats,
    CASE 
        WHEN e.betting_volume_percentage > 5 THEN 'HOT'
        WHEN e.betting_volume_percentage > 2 THEN 'POPULAR'
        WHEN e.betting_volume_percentage > 1 THEN 'MODERATE'
        ELSE 'COLD'
    END as popularity_level,
    CASE 
        WHEN e.betting_volume_percentage > 5 AND co.max_movement > 50 THEN 'HOT_MOVER'
        WHEN e.betting_volume_percentage < 1 AND co.max_movement > 50 THEN 'HIDDEN_GEM'
        WHEN e.betting_volume_percentage > 5 AND co.max_movement < 10 THEN 'STABLE_FAVORITE'
        ELSE 'NORMAL'
    END as event_type
FROM events e
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
LEFT JOIN LATERAL (
    SELECT 
        json_object_agg(
            mt.code || '_' || co.outcome,
            json_build_object(
                'current', co.odds_value,
                'opening', co.opening_value,
                'movement_pct', co.movement_percentage,
                'direction', CASE 
                    WHEN co.odds_value > co.opening_value THEN 'DRIFT'
                    WHEN co.odds_value < co.opening_value THEN 'SHORTEN'
                    ELSE 'STABLE'
                END
            )
        ) as movement_stats,
        MAX(ABS(co.movement_percentage)) as max_movement
    FROM current_odds co
    JOIN market_types mt ON co.market_type_id = mt.id
    WHERE co.event_id = e.id
    GROUP BY co.event_id
) co ON true
WHERE e.event_date > NOW()
ORDER BY e.betting_volume_percentage DESC;

-- View to identify value bets based on public bias
CREATE OR REPLACE VIEW value_opportunities AS
WITH odds_probabilities AS (
    SELECT 
        co.event_id,
        co.market_type_id,
        co.outcome,
        co.odds_value,
        -- Convert odds to implied probability
        CASE 
            WHEN co.odds_value > 0 THEN ROUND((1.0 / co.odds_value * 100)::numeric, 2)
            ELSE 0
        END as implied_probability
    FROM current_odds co
),
distribution_analysis AS (
    SELECT 
        od.event_id,
        od.market_id,
        od.outcome,
        od.bet_percentage,
        op.implied_probability,
        op.odds_value,
        od.bet_percentage - op.implied_probability as public_bias,
        CASE 
            WHEN od.bet_percentage > op.implied_probability + 10 THEN 'OVERBET'
            WHEN od.bet_percentage < op.implied_probability - 10 THEN 'UNDERBET'
            ELSE 'FAIR'
        END as bet_assessment
    FROM outcome_distributions od
    JOIN odds_probabilities op ON od.event_id = op.event_id 
        AND od.outcome = op.outcome
)
SELECT 
    e.slug as event_slug,
    ht.name || ' vs ' || at.name as match_name,
    e.event_date,
    'Market ' || da.market_id as market_name,
    da.outcome,
    da.odds_value as current_odds,
    da.implied_probability || '%' as implied_prob,
    da.bet_percentage || '%' as public_bets,
    da.public_bias as bias_percentage,
    da.bet_assessment,
    CASE 
        WHEN da.bet_assessment = 'UNDERBET' THEN 'Value opportunity - less bet than probability suggests'
        WHEN da.bet_assessment = 'OVERBET' THEN 'Avoid - public overvaluing this outcome'
        ELSE 'Market aligned with probabilities'
    END as recommendation
FROM distribution_analysis da
JOIN events e ON da.event_id = e.id
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
WHERE e.event_date > NOW()
  AND ABS(da.public_bias) > 5
ORDER BY ABS(da.public_bias) DESC;

-- Materialized views for performance
CREATE MATERIALIZED VIEW volume_trends AS
WITH hourly_volumes AS (
    SELECT 
        event_id,
        date_trunc('hour', recorded_at) as hour,
        AVG(volume_percentage) as avg_volume,
        MAX(volume_percentage) as max_volume,
        MIN(volume_percentage) as min_volume,
        COUNT(*) as measurements
    FROM betting_volume_history
    WHERE recorded_at > NOW() - INTERVAL '7 days'
    GROUP BY event_id, date_trunc('hour', recorded_at)
),
volume_changes AS (
    SELECT 
        event_id,
        hour,
        avg_volume,
        avg_volume - LAG(avg_volume) OVER (PARTITION BY event_id ORDER BY hour) as volume_change,
        (avg_volume - LAG(avg_volume) OVER (PARTITION BY event_id ORDER BY hour)) / 
            NULLIF(LAG(avg_volume) OVER (PARTITION BY event_id ORDER BY hour), 0) * 100 as volume_change_pct
    FROM hourly_volumes
)
SELECT 
    e.slug,
    vc.*,
    CASE 
        WHEN volume_change_pct > 100 THEN 'SURGE'
        WHEN volume_change_pct > 50 THEN 'INCREASING'
        WHEN volume_change_pct < -50 THEN 'DROPPING'
        ELSE 'STABLE'
    END as trend
FROM volume_changes vc
JOIN events e ON vc.event_id = e.id;

CREATE INDEX idx_volume_trends_lookup ON volume_trends (slug, hour DESC);

-- Materialized view for contrarian opportunities
CREATE MATERIALIZED VIEW contrarian_bets AS
WITH public_favorites AS (
    SELECT 
        od.event_id,
        od.market_id,
        od.outcome,
        od.bet_percentage,
        co.odds_value,
        od.bet_percentage - (100.0 / co.odds_value) as overbet_amount
    FROM outcome_distributions od
    JOIN current_odds co ON od.event_id = co.event_id 
        AND od.outcome = co.outcome
    WHERE od.bet_percentage > 60 -- Heavy public backing
)
SELECT 
    e.slug,
    ht.name || ' vs ' || at.name as match_name,
    'Market ' || pf.market_id as market,
    pf.outcome as public_choice,
    pf.bet_percentage || '% of bets' as public_backing,
    pf.odds_value as odds,
    ROUND(pf.overbet_amount, 1) || '%' as overbet_by,
    'Fade the public - bet opposite' as strategy
FROM public_favorites pf
JOIN events e ON pf.event_id = e.id
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
WHERE e.event_date > NOW()
  AND pf.overbet_amount > 15
ORDER BY pf.overbet_amount DESC;

CREATE INDEX idx_contrarian_bets_refresh ON contrarian_bets (match_name);

-- Functions for advanced analytics
CREATE OR REPLACE FUNCTION get_biggest_movers(
    hours_back INTEGER DEFAULT 24,
    min_movement_pct DECIMAL DEFAULT 20
) RETURNS TABLE (
    event_slug VARCHAR,
    home_team VARCHAR,
    away_team VARCHAR,
    event_date TIMESTAMP,
    market_name VARCHAR,
    outcome VARCHAR,
    opening_odds DECIMAL,
    current_odds DECIMAL,
    lowest_odds DECIMAL,
    highest_odds DECIMAL,
    movement_pct DECIMAL,
    multiplier DECIMAL,
    trend VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        e.slug,
        ht.name,
        at.name,
        e.event_date,
        mt.name,
        co.outcome,
        co.opening_value,
        co.odds_value,
        co.lowest_value,
        co.highest_value,
        co.movement_percentage,
        ROUND((co.odds_value / NULLIF(co.opening_value, 0))::numeric, 2),
        CASE 
            WHEN co.odds_value > co.opening_value THEN 'DRIFTING'
            WHEN co.odds_value < co.opening_value THEN 'SHORTENING'
            ELSE 'STABLE'
        END
    FROM current_odds co
    JOIN events e ON co.event_id = e.id
    JOIN teams ht ON e.home_team_id = ht.id
    JOIN teams at ON e.away_team_id = at.id
    JOIN market_types mt ON co.market_type_id = mt.id
    WHERE co.last_updated > NOW() - (hours_back || ' hours')::INTERVAL
      AND ABS(co.movement_percentage) >= min_movement_pct
      AND e.event_date > NOW()  -- Only future events
    ORDER BY ABS(co.movement_percentage) DESC;
END;
$$ LANGUAGE plpgsql;

-- Function to analyze betting patterns
CREATE OR REPLACE FUNCTION analyze_betting_patterns(p_event_id INTEGER)
RETURNS TABLE (
    pattern_type VARCHAR,
    description TEXT,
    confidence DECIMAL,
    evidence JSONB
) AS $$
DECLARE
    v_volume_pct DECIMAL;
    v_max_bias DECIMAL;
    v_outcome_count INTEGER;
    v_overbet_count INTEGER;
BEGIN
    -- Get event volume
    SELECT betting_volume_percentage INTO v_volume_pct
    FROM events WHERE id = p_event_id;
    
    -- Get distribution stats
    SELECT 
        MAX(ABS(od.bet_percentage - op.implied_probability)),
        COUNT(DISTINCT od.outcome),
        COUNT(CASE WHEN od.bet_percentage > op.implied_probability + 10 THEN 1 END)
    INTO v_max_bias, v_outcome_count, v_overbet_count
    FROM outcome_distributions od
    LEFT JOIN (
        SELECT event_id, outcome, 
               ROUND((1.0 / odds_value * 100)::numeric, 2) as implied_probability
        FROM current_odds
        WHERE event_id = p_event_id
    ) op ON od.event_id = op.event_id AND od.outcome = op.outcome
    WHERE od.event_id = p_event_id;
    
    -- Pattern detection logic
    IF v_volume_pct > 5 AND v_max_bias > 20 THEN
        RETURN QUERY
        SELECT 
            'PUBLIC_BIAS'::VARCHAR,
            'Heavy public betting causing market inefficiency'::TEXT,
            0.85::DECIMAL,
            jsonb_build_object(
                'volume_rank', v_volume_pct,
                'max_bias', v_max_bias,
                'overbet_outcomes', v_overbet_count
            );
    ELSIF v_volume_pct < 1 AND v_max_bias > 15 THEN
        RETURN QUERY
        SELECT 
            'SHARP_ACTION'::VARCHAR,
            'Low volume but significant distribution skew suggests sharp money'::TEXT,
            0.75::DECIMAL,
            jsonb_build_object(
                'volume_rank', v_volume_pct,
                'distribution_skew', v_max_bias
            );
    ELSIF v_overbet_count = 0 AND v_outcome_count > 2 THEN
        RETURN QUERY
        SELECT 
            'EFFICIENT_MARKET'::VARCHAR,
            'Betting aligns with probabilities - limited value'::TEXT,
            0.90::DECIMAL,
            jsonb_build_object(
                'aligned_outcomes', v_outcome_count
            );
    ELSE
        RETURN QUERY
        SELECT 
            'MIXED_SIGNALS'::VARCHAR,
            'No clear pattern detected'::TEXT,
            0.50::DECIMAL,
            jsonb_build_object();
    END IF;
END;
$$ LANGUAGE plpgsql;