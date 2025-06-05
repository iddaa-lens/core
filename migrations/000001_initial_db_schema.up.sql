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
    slug VARCHAR(255),
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
    slug VARCHAR(255),
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
    betting_volume_percentage DECIMAL(5, 2),
    volume_rank INTEGER,
    volume_updated_at TIMESTAMP,
    -- Additional Iddaa-specific fields
    bulletin_id BIGINT,                     -- bri: Iddaa bulletin/program ID
    version BIGINT,                         -- v: Event version for change tracking
    sport_id INTEGER REFERENCES sports(id), -- sid: Sport ID for validation
    bet_program INTEGER,                    -- bp: Betting program identifier
    mbc INTEGER,                           -- mbc: Market betting category
    has_king_odd BOOLEAN DEFAULT false,     -- kOdd: King odds availability
    odds_count INTEGER DEFAULT 0,          -- oc: Number of available odds
    has_combine BOOLEAN DEFAULT false,      -- hc: Combination betting availability
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
    iddaa_market_id INTEGER,                -- i: Market ID from Iddaa
    is_live BOOLEAN DEFAULT false,          -- il: Is live market
    market_type INTEGER,                    -- mt: Market type
    min_market_default_value INTEGER,       -- mmdv: Min market default value
    max_market_limit_value INTEGER,         -- mmlv: Max market limit value
    priority INTEGER DEFAULT 0,            -- p: Priority
    sport_type INTEGER,                     -- st: Sport type
    market_sub_type INTEGER,                -- mst: Market sub type
    min_default_value INTEGER,              -- mdv: Min default value
    max_limit_value INTEGER,                -- mlv: Max limit value
    is_active BOOLEAN DEFAULT true,         -- in: Is active
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Current odds (latest values)
CREATE TABLE IF NOT EXISTS current_odds (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(100) NOT NULL,
    odds_value DECIMAL(10, 3) NOT NULL,
    opening_value DECIMAL(10, 3),
    highest_value DECIMAL(10, 3),
    lowest_value DECIMAL(10, 3),
    winning_odds DECIMAL(10, 3),
    total_movement DECIMAL(10, 3),
    movement_percentage DECIMAL(10, 2),
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(event_id, market_type_id, outcome)
);

-- Odds history (changes over time)
CREATE TABLE IF NOT EXISTS odds_history (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(100) NOT NULL,
    odds_value DECIMAL(10, 3) NOT NULL,
    previous_value DECIMAL(10, 3),
    winning_odds DECIMAL(10, 3),
    change_amount DECIMAL(10, 3),
    change_percentage DECIMAL(10, 2),
    multiplier DECIMAL(10, 2),
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

-- Outcome distributions (betting percentages)
CREATE TABLE IF NOT EXISTS outcome_distributions (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_id INTEGER NOT NULL,
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(10) NOT NULL,
    bet_percentage DECIMAL(5, 2) NOT NULL,
    implied_probability DECIMAL(5, 2),
    value_indicator DECIMAL(5, 2),
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(event_id, market_id, outcome)
);

-- Outcome distribution history
CREATE TABLE IF NOT EXISTS outcome_distribution_history (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    market_id INTEGER NOT NULL,
    outcome VARCHAR(10) NOT NULL,
    bet_percentage DECIMAL(5, 2) NOT NULL,
    previous_percentage DECIMAL(5, 2),
    change_amount DECIMAL(5, 2),
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
    confidence DECIMAL(3,2) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    mapping_method VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(internal_league_id)
    -- Removed UNIQUE(football_api_league_id) to allow multiple Turkish leagues -> same Football API league
);

CREATE TABLE IF NOT EXISTS team_mappings (
    id SERIAL PRIMARY KEY,
    internal_team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    football_api_team_id INTEGER NOT NULL,
    confidence DECIMAL(3,2) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    mapping_method VARCHAR(50) NOT NULL,
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
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_leagues_sport_id ON leagues(sport_id);
CREATE INDEX IF NOT EXISTS idx_leagues_external_id ON leagues(external_id);
CREATE INDEX IF NOT EXISTS idx_leagues_is_active ON leagues(is_active);
CREATE INDEX IF NOT EXISTS idx_leagues_slug ON leagues(slug);

CREATE INDEX IF NOT EXISTS idx_teams_external_id ON teams(external_id);
CREATE INDEX IF NOT EXISTS idx_teams_slug ON teams(slug);

CREATE INDEX IF NOT EXISTS idx_events_league_id ON events(league_id);
CREATE INDEX IF NOT EXISTS idx_events_date ON events(event_date);
CREATE INDEX IF NOT EXISTS idx_events_status ON events(status);
CREATE INDEX IF NOT EXISTS idx_events_slug ON events(slug);

CREATE INDEX IF NOT EXISTS idx_market_types_code ON market_types(code);
CREATE INDEX IF NOT EXISTS idx_market_types_slug ON market_types(slug);

CREATE INDEX IF NOT EXISTS idx_current_odds_event_market ON current_odds(event_id, market_type_id);
CREATE INDEX IF NOT EXISTS idx_current_odds_movement ON current_odds(ABS(movement_percentage) DESC);

CREATE INDEX IF NOT EXISTS idx_odds_history_event_time ON odds_history(event_id, recorded_at DESC);

CREATE INDEX IF NOT EXISTS idx_volume_history_event_time ON betting_volume_history(event_id, recorded_at DESC);

CREATE INDEX IF NOT EXISTS idx_outcome_dist_event ON outcome_distributions(event_id);

CREATE INDEX IF NOT EXISTS idx_league_mappings_internal_league_id ON league_mappings(internal_league_id);
CREATE INDEX IF NOT EXISTS idx_league_mappings_football_api_league_id ON league_mappings(football_api_league_id);
CREATE INDEX IF NOT EXISTS idx_league_mappings_confidence ON league_mappings(confidence);

CREATE INDEX IF NOT EXISTS idx_team_mappings_internal_team_id ON team_mappings(internal_team_id);
CREATE INDEX IF NOT EXISTS idx_team_mappings_football_api_team_id ON team_mappings(football_api_team_id);
CREATE INDEX IF NOT EXISTS idx_team_mappings_confidence ON team_mappings(confidence);

-- Create improved slug normalization function
-- This matches the gosimple/slug library behavior for consistent results
CREATE OR REPLACE FUNCTION normalize_slug(input_text TEXT)
RETURNS TEXT AS $$
DECLARE
    normalized_text TEXT;
BEGIN
    IF input_text IS NULL OR input_text = '' THEN
        RETURN '';
    END IF;

    -- Start with the input text
    normalized_text := input_text;
    
    -- Turkish characters
    normalized_text := REPLACE(normalized_text, 'ı', 'i');
    normalized_text := REPLACE(normalized_text, 'İ', 'I');
    normalized_text := REPLACE(normalized_text, 'ğ', 'g');
    normalized_text := REPLACE(normalized_text, 'Ğ', 'G');
    normalized_text := REPLACE(normalized_text, 'ü', 'u');
    normalized_text := REPLACE(normalized_text, 'Ü', 'U');
    normalized_text := REPLACE(normalized_text, 'ş', 's');
    normalized_text := REPLACE(normalized_text, 'Ş', 'S');
    normalized_text := REPLACE(normalized_text, 'ö', 'o');
    normalized_text := REPLACE(normalized_text, 'Ö', 'O');
    normalized_text := REPLACE(normalized_text, 'ç', 'c');
    normalized_text := REPLACE(normalized_text, 'Ç', 'C');
    
    -- German/European characters
    normalized_text := REPLACE(normalized_text, 'ä', 'a');
    normalized_text := REPLACE(normalized_text, 'Ä', 'A');
    normalized_text := REPLACE(normalized_text, 'ß', 'ss');
    
    -- Nordic characters
    normalized_text := REPLACE(normalized_text, 'ø', 'o');
    normalized_text := REPLACE(normalized_text, 'Ø', 'O');
    normalized_text := REPLACE(normalized_text, 'å', 'a');
    normalized_text := REPLACE(normalized_text, 'Å', 'A');
    normalized_text := REPLACE(normalized_text, 'æ', 'ae');
    normalized_text := REPLACE(normalized_text, 'Æ', 'AE');
    
    -- Spanish/French characters
    normalized_text := REPLACE(normalized_text, 'ñ', 'n');
    normalized_text := REPLACE(normalized_text, 'Ñ', 'N');
    normalized_text := REPLACE(normalized_text, 'é', 'e');
    normalized_text := REPLACE(normalized_text, 'É', 'E');
    normalized_text := REPLACE(normalized_text, 'è', 'e');
    normalized_text := REPLACE(normalized_text, 'È', 'E');
    normalized_text := REPLACE(normalized_text, 'ê', 'e');
    normalized_text := REPLACE(normalized_text, 'Ê', 'E');
    normalized_text := REPLACE(normalized_text, 'ë', 'e');
    normalized_text := REPLACE(normalized_text, 'Ë', 'E');
    normalized_text := REPLACE(normalized_text, 'á', 'a');
    normalized_text := REPLACE(normalized_text, 'Á', 'A');
    normalized_text := REPLACE(normalized_text, 'à', 'a');
    normalized_text := REPLACE(normalized_text, 'À', 'A');
    normalized_text := REPLACE(normalized_text, 'â', 'a');
    normalized_text := REPLACE(normalized_text, 'Â', 'A');
    normalized_text := REPLACE(normalized_text, 'í', 'i');
    normalized_text := REPLACE(normalized_text, 'Í', 'I');
    normalized_text := REPLACE(normalized_text, 'ì', 'i');
    normalized_text := REPLACE(normalized_text, 'Ì', 'I');
    normalized_text := REPLACE(normalized_text, 'î', 'i');
    normalized_text := REPLACE(normalized_text, 'Î', 'I');
    normalized_text := REPLACE(normalized_text, 'ï', 'i');
    normalized_text := REPLACE(normalized_text, 'Ï', 'I');
    normalized_text := REPLACE(normalized_text, 'ó', 'o');
    normalized_text := REPLACE(normalized_text, 'Ó', 'O');
    normalized_text := REPLACE(normalized_text, 'ò', 'o');
    normalized_text := REPLACE(normalized_text, 'Ò', 'O');
    normalized_text := REPLACE(normalized_text, 'ô', 'o');
    normalized_text := REPLACE(normalized_text, 'Ô', 'O');
    normalized_text := REPLACE(normalized_text, 'ú', 'u');
    normalized_text := REPLACE(normalized_text, 'Ú', 'U');
    normalized_text := REPLACE(normalized_text, 'ù', 'u');
    normalized_text := REPLACE(normalized_text, 'Ù', 'U');
    normalized_text := REPLACE(normalized_text, 'û', 'u');
    normalized_text := REPLACE(normalized_text, 'Û', 'U');

    -- Convert to lowercase
    normalized_text := lower(normalized_text);

    -- Replace non-alphanumeric characters with hyphens
    normalized_text := regexp_replace(normalized_text, '[^a-z0-9]+', '-', 'g');

    -- Remove leading and trailing hyphens
    normalized_text := trim(both '-' from normalized_text);

    -- Replace multiple consecutive hyphens with single hyphen
    normalized_text := regexp_replace(normalized_text, '-+', '-', 'g');

    RETURN normalized_text;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Auto-generate slugs for leagues
CREATE OR REPLACE FUNCTION generate_league_slug()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.slug IS NULL OR NEW.slug = '' THEN
        NEW.slug := normalize_slug(NEW.name || ' ' || COALESCE(NEW.country, ''));
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Auto-generate slugs for teams
CREATE OR REPLACE FUNCTION generate_team_slug()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.slug IS NULL OR NEW.slug = '' THEN
        NEW.slug := normalize_slug(NEW.name);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Auto-generate slugs for events
CREATE OR REPLACE FUNCTION generate_event_slug()
RETURNS TRIGGER AS $$
DECLARE
    home_team_name TEXT;
    away_team_name TEXT;
    base_slug TEXT;
    final_slug TEXT;
    counter INTEGER := 0;
BEGIN
    IF NEW.slug IS NULL OR NEW.slug = '' THEN
        -- Get team names
        SELECT name INTO home_team_name FROM teams WHERE id = NEW.home_team_id;
        SELECT name INTO away_team_name FROM teams WHERE id = NEW.away_team_id;
        
        -- Create base slug from team names and external_id using improved normalization
        base_slug := normalize_slug(
            COALESCE(home_team_name, 'team') || ' vs ' || 
            COALESCE(away_team_name, 'team') || ' ' || 
            NEW.external_id
        );
        
        -- Ensure uniqueness by appending counter if needed
        final_slug := base_slug;
        WHILE EXISTS (SELECT 1 FROM events WHERE slug = final_slug AND id != COALESCE(NEW.id, -1)) LOOP
            counter := counter + 1;
            final_slug := base_slug || '-' || counter;
        END LOOP;
        
        NEW.slug := final_slug;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Auto-generate slugs for market types
CREATE OR REPLACE FUNCTION generate_market_type_slug()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.slug IS NULL OR NEW.slug = '' THEN
        NEW.slug := normalize_slug(NEW.code || ' ' || NEW.name);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers
CREATE TRIGGER auto_generate_league_slug
    BEFORE INSERT OR UPDATE ON leagues
    FOR EACH ROW
    EXECUTE FUNCTION generate_league_slug();

CREATE TRIGGER auto_generate_team_slug
    BEFORE INSERT OR UPDATE ON teams
    FOR EACH ROW
    EXECUTE FUNCTION generate_team_slug();

CREATE TRIGGER auto_generate_event_slug
    BEFORE INSERT OR UPDATE ON events
    FOR EACH ROW
    EXECUTE FUNCTION generate_event_slug();

CREATE TRIGGER auto_generate_market_type_slug
    BEFORE INSERT OR UPDATE ON market_types
    FOR EACH ROW
    EXECUTE FUNCTION generate_market_type_slug();

-- Update timestamp triggers
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_leagues_updated_at BEFORE UPDATE ON leagues FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER update_teams_updated_at BEFORE UPDATE ON teams FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER update_events_updated_at BEFORE UPDATE ON events FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER update_market_types_updated_at BEFORE UPDATE ON market_types FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER update_app_config_updated_at BEFORE UPDATE ON app_config FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER update_match_statistics_updated_at BEFORE UPDATE ON match_statistics FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER update_league_mappings_updated_at BEFORE UPDATE ON league_mappings FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER update_team_mappings_updated_at BEFORE UPDATE ON team_mappings FOR EACH ROW EXECUTE FUNCTION update_updated_at();


-- Create view for big odds movements
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

-- Create view for contrarian betting opportunities
CREATE OR REPLACE VIEW contrarian_bets AS
SELECT 
    e.slug,
    ht.name || ' vs ' || at.name as match_name,
    'Market ' || od.market_id as market,
    od.outcome as public_choice,
    od.bet_percentage || '% of bets' as public_backing,
    co.odds_value as odds,
    ROUND((od.bet_percentage - (100.0 / co.odds_value)), 1) || '%' as overbet_by,
    'Fade the public - bet opposite' as strategy
FROM outcome_distributions od
JOIN current_odds co ON od.event_id = co.event_id AND od.outcome = co.outcome
JOIN events e ON od.event_id = e.id
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
WHERE od.bet_percentage > 60
  AND e.event_date > NOW()
  AND (od.bet_percentage - (100.0 / co.odds_value)) > 15
ORDER BY (od.bet_percentage - (100.0 / co.odds_value)) DESC;