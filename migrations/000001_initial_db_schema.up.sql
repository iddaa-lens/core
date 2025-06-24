-- iddaa-core database schema
-- Simplified development version with essential tables only
-- Sports (football, basketball, etc.)
CREATE TABLE IF NOT EXISTS sports (
    id INTEGER PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(20) NOT NULL UNIQUE,
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

-- Leagues (competitions/tournaments)
CREATE TABLE IF NOT EXISTS leagues (
    id SERIAL PRIMARY KEY,
    external_id VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    country VARCHAR(100),
    sport_id INTEGER REFERENCES sports(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    slug VARCHAR(255) UNIQUE NOT NULL,
    -- API-Football enrichment fields
    api_football_id INTEGER,
    league_type VARCHAR(50),
    -- 'League', 'Cup', etc.
    logo_url TEXT,
    country_code VARCHAR(10),
    -- 'GB', 'TR', etc.
    country_flag_url TEXT,
    has_standings BOOLEAN DEFAULT false,
    has_fixtures BOOLEAN DEFAULT false,
    has_players BOOLEAN DEFAULT false,
    has_top_scorers BOOLEAN DEFAULT false,
    has_injuries BOOLEAN DEFAULT false,
    has_predictions BOOLEAN DEFAULT false,
    has_odds BOOLEAN DEFAULT false,
    current_season_year INTEGER,
    current_season_start DATE,
    current_season_end DATE,
    api_enrichment_data JSONB,
    -- Store full API response for flexibility
    last_api_update TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Teams
CREATE TABLE IF NOT EXISTS teams (
    id SERIAL PRIMARY KEY,
    external_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    country VARCHAR(100),
    logo_url TEXT,
    is_active BOOLEAN DEFAULT true,
    slug VARCHAR(255) UNIQUE NOT NULL,
    -- API-Football enrichment fields
    api_football_id INTEGER,
    team_code VARCHAR(10),
    -- 'MUN', 'FCB', etc.
    founded_year INTEGER,
    is_national_team BOOLEAN DEFAULT false,
    venue_id INTEGER,
    venue_name VARCHAR(255),
    venue_address TEXT,
    venue_city VARCHAR(100),
    venue_capacity INTEGER,
    venue_surface VARCHAR(50),
    -- 'grass', 'artificial', etc.
    venue_image_url TEXT,
    api_enrichment_data JSONB,
    -- Store full API response for flexibility
    last_api_update TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Events (matches/games)
CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    external_id VARCHAR(255) UNIQUE NOT NULL,
    league_id INTEGER REFERENCES leagues(id) ON DELETE CASCADE,
    home_team_id INTEGER REFERENCES teams(id),
    away_team_id INTEGER REFERENCES teams(id),
    slug VARCHAR(500) UNIQUE NOT NULL,
    event_date TIMESTAMP NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'scheduled',
    home_score INTEGER,
    away_score INTEGER,
    is_live BOOLEAN DEFAULT false,
    minute_of_match INTEGER,
    half INTEGER,
    betting_volume_percentage REAL CHECK (
        betting_volume_percentage >= 0
        AND betting_volume_percentage <= 100
    ),
    -- Changed from DECIMAL(5,2) to REAL
    volume_rank INTEGER,
    volume_updated_at TIMESTAMP,
    -- Additional Iddaa-specific fields
    bulletin_id BIGINT,
    -- bri: Iddaa bulletin/program ID
    version BIGINT,
    -- v: Event version for change tracking
    sport_id INTEGER REFERENCES sports(id),
    -- sid: Sport ID for validation
    bet_program INTEGER,
    -- bp: Betting program identifier
    mbc INTEGER,
    -- mbc: Market betting category
    has_king_odd BOOLEAN DEFAULT false,
    -- kOdd: King odds availability
    odds_count INTEGER DEFAULT 0,
    -- oc: Number of available odds
    has_combine BOOLEAN DEFAULT false,
    -- hc: Combination betting availability
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
    -- Iddaa-specific market configuration fields
    iddaa_market_id INTEGER,
    -- i: Market ID from Iddaa
    is_live BOOLEAN DEFAULT false,
    -- il: Is live market
    market_type INTEGER,
    -- mt: Market type
    min_market_default_value INTEGER,
    -- mmdv: Min market default value
    max_market_limit_value INTEGER,
    -- mmlv: Max market limit value
    priority INTEGER DEFAULT 0,
    -- p: Priority
    sport_type INTEGER,
    -- st: Sport type
    market_sub_type INTEGER,
    -- mst: Market sub type
    min_default_value INTEGER,
    -- mdv: Min default value
    max_limit_value INTEGER,
    -- mlv: Max limit value
    is_active BOOLEAN DEFAULT true,
    -- in: Is active
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Current odds (latest values)
CREATE TABLE IF NOT EXISTS current_odds (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(100) NOT NULL,
    odds_value DOUBLE PRECISION NOT NULL,
    -- Changed from DECIMAL(10,3)
    opening_value DOUBLE PRECISION,
    -- Changed from DECIMAL(10,3)
    highest_value DOUBLE PRECISION,
    -- Changed from DECIMAL(10,3)
    lowest_value DOUBLE PRECISION,
    -- Changed from DECIMAL(10,3)
    winning_odds DOUBLE PRECISION,
    -- Changed from DECIMAL(10,3)
    total_movement DOUBLE PRECISION,
    -- Changed from DECIMAL(10,3)
    movement_percentage REAL,
    -- Changed from DECIMAL(10,2)
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    market_params JSONB,
    -- Added from your previous requirements
    UNIQUE(event_id, market_type_id, outcome)
);

-- Odds history (changes over time)
CREATE TABLE IF NOT EXISTS odds_history (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(100) NOT NULL,
    odds_value DOUBLE PRECISION NOT NULL,
    -- Changed from DECIMAL(10,3)
    previous_value DOUBLE PRECISION,
    -- Changed from DECIMAL(10,3)
    winning_odds DOUBLE PRECISION,
    -- Changed from DECIMAL(10,3)
    change_amount DOUBLE PRECISION,
    -- Changed from DECIMAL(10,3)
    change_percentage REAL,
    -- Changed from DECIMAL(10,2)
    multiplier DOUBLE PRECISION,
    -- Changed from DECIMAL(10,2)
    sharp_money_indicator REAL DEFAULT 0 CHECK (
        sharp_money_indicator >= 0
        AND sharp_money_indicator <= 1
    ),
    -- Changed from DECIMAL(3,2)
    is_reverse_movement BOOLEAN DEFAULT FALSE,
    significance_level VARCHAR(20) DEFAULT 'normal' CHECK (
        significance_level IN ('normal', 'high', 'extreme')
    ),
    minutes_to_kickoff INTEGER,
    market_params JSONB,
    -- Added from your previous requirements
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Betting volume history
CREATE TABLE IF NOT EXISTS betting_volume_history (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    volume_percentage REAL NOT NULL CHECK (
        volume_percentage >= 0
        AND volume_percentage <= 100
    ),
    -- Changed from DECIMAL(5,2)
    rank_position INTEGER,
    total_events_tracked INTEGER,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Outcome distributions (betting percentages)
CREATE TABLE IF NOT EXISTS outcome_distributions (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_id INTEGER NOT NULL,
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(10) NOT NULL,
    bet_percentage REAL NOT NULL CHECK (
        bet_percentage >= 0
        AND bet_percentage <= 100
    ),
    -- Changed from DECIMAL(5,2)
    implied_probability REAL CHECK (
        implied_probability >= 0
        AND implied_probability <= 100
    ),
    -- Changed from DECIMAL(5,2)
    value_indicator REAL,
    -- Changed from DECIMAL(5,2)
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(event_id, market_id, outcome)
);

-- Outcome distribution history
CREATE TABLE IF NOT EXISTS outcome_distribution_history (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_id INTEGER NOT NULL,
    outcome VARCHAR(10) NOT NULL,
    bet_percentage REAL NOT NULL CHECK (
        bet_percentage >= 0
        AND bet_percentage <= 100
    ),
    -- Changed from DECIMAL(5,2)
    previous_percentage REAL,
    -- Changed from DECIMAL(5,2)
    change_amount REAL,
    -- Changed from DECIMAL(5,2)
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Match events (goals, cards, etc.)
CREATE TABLE IF NOT EXISTS match_events (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    minute INTEGER NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    team VARCHAR(10) NOT NULL,
    player VARCHAR(255),
    description VARCHAR(255) NOT NULL,
    is_home BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Match statistics
CREATE TABLE IF NOT EXISTS match_statistics (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    is_home BOOLEAN NOT NULL,
    shots INTEGER,
    shots_on_target INTEGER,
    possession INTEGER,
    corners INTEGER,
    yellow_cards INTEGER,
    red_cards INTEGER,
    fouls INTEGER,
    offsides INTEGER,
    free_kicks INTEGER,
    throw_ins INTEGER,
    goal_kicks INTEGER,
    saves INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Football API mappings (for external league/team data)
CREATE TABLE IF NOT EXISTS league_mappings (
    id SERIAL PRIMARY KEY,
    internal_league_id INTEGER NOT NULL REFERENCES leagues(id) ON DELETE CASCADE,
    football_api_league_id INTEGER NOT NULL,
    confidence REAL NOT NULL CHECK (
        confidence >= 0
        AND confidence <= 1
    ),
    -- Changed from DECIMAL(3,2)
    mapping_method VARCHAR(50) NOT NULL,
    -- Enhanced translation tracking fields
    translated_league_name TEXT,
    translated_country TEXT,
    original_league_name TEXT,
    original_country TEXT,
    match_factors JSONB,
    needs_review BOOLEAN DEFAULT FALSE,
    ai_translation_used BOOLEAN DEFAULT FALSE,
    normalization_applied BOOLEAN DEFAULT FALSE,
    match_score REAL DEFAULT 0.0 CHECK (
        match_score >= 0.0
        AND match_score <= 1.0
    ),
    -- Changed from DECIMAL(5,4)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(internal_league_id) -- Removed UNIQUE(football_api_league_id) to allow multiple Turkish leagues -> same Football API league
);

CREATE TABLE IF NOT EXISTS team_mappings (
    id SERIAL PRIMARY KEY,
    internal_team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    football_api_team_id INTEGER NOT NULL,
    confidence REAL NOT NULL CHECK (
        confidence >= 0
        AND confidence <= 1
    ),
    -- Changed from DECIMAL(3,2)
    mapping_method VARCHAR(50) NOT NULL,
    -- Enhanced translation tracking fields
    translated_team_name TEXT,
    translated_country TEXT,
    translated_league TEXT,
    original_team_name TEXT,
    original_country TEXT,
    original_league TEXT,
    match_factors JSONB,
    needs_review BOOLEAN DEFAULT FALSE,
    ai_translation_used BOOLEAN DEFAULT FALSE,
    normalization_applied BOOLEAN DEFAULT FALSE,
    match_score REAL DEFAULT 0.0 CHECK (
        match_score >= 0.0
        AND match_score <= 1.0
    ),
    -- Changed from DECIMAL(5,4)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(internal_team_id),
    UNIQUE(football_api_team_id)
);

-- Configuration storage
CREATE TABLE IF NOT EXISTS app_config (
    id SERIAL PRIMARY KEY,
    platform VARCHAR(50) UNIQUE NOT NULL,
    config_data JSONB NOT NULL,
    sportoto_program_name VARCHAR(255),
    payin_end_date TIMESTAMP,
    next_draw_expected_win DECIMAL(15, 2),
    -- Keep as DECIMAL for money values
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for movement alerts (references odds_history IDs)
CREATE TABLE IF NOT EXISTS movement_alerts (
    id SERIAL PRIMARY KEY,
    odds_history_id INTEGER NOT NULL REFERENCES odds_history(id) ON DELETE CASCADE,
    -- Alert details
    alert_type VARCHAR(50) NOT NULL CHECK (
        alert_type IN (
            'big_mover',
            'reverse_line',
            'sharp_money',
            'value_spot'
        )
    ),
    severity VARCHAR(20) NOT NULL CHECK (
        severity IN ('low', 'medium', 'high', 'critical')
    ),
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    -- Movement context (calculated at alert time)
    change_percentage REAL NOT NULL,
    -- Changed from DECIMAL(10,2)
    multiplier DOUBLE PRECISION NOT NULL CHECK (multiplier > 0),
    -- Changed from DECIMAL(10,3)
    confidence_score REAL NOT NULL DEFAULT 0.5 CHECK (
        confidence_score >= 0
        AND confidence_score <= 1
    ),
    -- Changed from DECIMAL(3,2)
    minutes_to_kickoff INTEGER,
    -- Alert metadata
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    expires_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP + INTERVAL '24 hours'),
    is_active BOOLEAN NOT NULL DEFAULT true,
    -- Engagement tracking
    views INTEGER NOT NULL DEFAULT 0 CHECK (views >= 0),
    clicks INTEGER NOT NULL DEFAULT 0 CHECK (clicks >= 0),
    -- Prevent duplicate alerts for same movement
    UNIQUE(odds_history_id, alert_type)
);

-- Update timestamp triggers
CREATE
OR REPLACE FUNCTION update_updated_at() RETURNS TRIGGER AS $$ BEGIN NEW.updated_at = CURRENT_TIMESTAMP;
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_leagues_updated_at BEFORE
UPDATE
    ON leagues FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_teams_updated_at BEFORE
UPDATE
    ON teams FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_events_updated_at BEFORE
UPDATE
    ON events FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_market_types_updated_at BEFORE
UPDATE
    ON market_types FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_app_config_updated_at BEFORE
UPDATE
    ON app_config FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_match_statistics_updated_at BEFORE
UPDATE
    ON match_statistics FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_league_mappings_updated_at BEFORE
UPDATE
    ON league_mappings FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_team_mappings_updated_at BEFORE
UPDATE
    ON team_mappings FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Create indexes for better performance
CREATE INDEX idx_current_odds_event_id ON current_odds(event_id);

CREATE INDEX idx_current_odds_market_type_id ON current_odds(market_type_id);

CREATE INDEX idx_current_odds_event_market ON current_odds(event_id, market_type_id);

CREATE INDEX idx_odds_history_event_id ON odds_history(event_id);

CREATE INDEX idx_odds_history_recorded_at ON odds_history(recorded_at DESC);

CREATE INDEX idx_odds_history_event_market_outcome ON odds_history(event_id, market_type_id, outcome);

CREATE INDEX idx_events_event_date ON events(event_date);

CREATE INDEX idx_events_status ON events(status);

CREATE INDEX idx_events_league_id ON events(league_id);