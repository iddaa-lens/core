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

-- Events (matches/games) with volume tracking and live statistics
CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    external_id VARCHAR(255) UNIQUE NOT NULL,
    competition_id INTEGER REFERENCES competitions(id),
    home_team_id INTEGER REFERENCES teams(id),
    away_team_id INTEGER REFERENCES teams(id),
    slug VARCHAR(500) UNIQUE NOT NULL,
    event_date TIMESTAMP NOT NULL,
    status VARCHAR(50) NOT NULL,
    home_score INTEGER,
    away_score INTEGER,
    is_live BOOLEAN DEFAULT FALSE,
    minute_of_match INTEGER,
    half INTEGER DEFAULT 0,
    betting_volume_percentage DECIMAL(5, 2),
    volume_rank INTEGER,
    volume_updated_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

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

-- Current odds (latest values for fast lookup)
CREATE TABLE IF NOT EXISTS current_odds (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(100) NOT NULL,
    odds_value DECIMAL(10, 3) NOT NULL,
    opening_value DECIMAL(10, 3) NOT NULL,
    highest_value DECIMAL(10, 3) NOT NULL,
    lowest_value DECIMAL(10, 3) NOT NULL,
    winning_odds DECIMAL(10, 3),
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

-- Odds history (only changes)
CREATE TABLE IF NOT EXISTS odds_history (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(100) NOT NULL,
    odds_value DECIMAL(10, 3) NOT NULL,
    previous_value DECIMAL(10, 3),
    winning_odds DECIMAL(10, 3),
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

-- Betting volume history
CREATE TABLE IF NOT EXISTS betting_volume_history (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    volume_percentage DECIMAL(5, 2) NOT NULL,
    rank_position INTEGER,
    total_events_tracked INTEGER,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Outcome betting distributions
CREATE TABLE IF NOT EXISTS outcome_distributions (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_id INTEGER NOT NULL,
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(10) NOT NULL,
    bet_percentage DECIMAL(5, 2) NOT NULL,
    implied_probability DECIMAL(5, 2),
    value_indicator DECIMAL(5, 2) GENERATED ALWAYS AS (
        CASE 
            WHEN implied_probability > 0 AND bet_percentage > 0
            THEN (implied_probability - bet_percentage)
            ELSE 0
        END
    ) STORED,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(event_id, market_id, outcome)
);

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

-- AI predictions
CREATE TABLE IF NOT EXISTS predictions (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id),
    market_type_id INTEGER REFERENCES market_types(id),
    slug VARCHAR(500) UNIQUE NOT NULL,
    predicted_outcome VARCHAR(100) NOT NULL,
    confidence_score DECIMAL(5, 4) NOT NULL,
    model_version VARCHAR(50) NOT NULL,
    features_used TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

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
WHERE ABS(co.movement_percentage) > 20
   OR co.odds_value / NULLIF(co.opening_value, 0) > 2
   OR co.opening_value / NULLIF(co.odds_value, 0) > 2
ORDER BY ABS(co.movement_percentage) DESC;

-- Materialized views for performance
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
    WHERE od.bet_percentage > 60
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

-- Match statistics table for detailed match statistics
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

-- Match events table for individual match events (goals, cards, etc.)
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