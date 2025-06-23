-- ============================================================================
-- VIEWS AND MATERIALIZED VIEWS
-- ============================================================================
-- ============================================================================
-- BIG MOVERS VIEW
-- Purpose: Identifies significant odds movements that may indicate smart money activity
-- or important information entering the market.
-- 
-- Use Cases:
-- - Spot sharp money moves (large odds changes)
-- - Identify potential value before lines adjust further
-- - Track market volatility for specific events
--
-- Key Metrics:
-- - Movement > 20% or odds multiplier > 2x
-- - Shows both shortening (price decrease) and drifting (price increase)
-- - Includes time context (how close to kickoff)
-- ============================================================================
CREATE MATERIALIZED VIEW big_movers AS
SELECT
    e.id as event_id,
    e.slug as event_slug,
    e.external_id as event_external_id,
    s.name as sport_name,
    s.slug as sport_slug,
    l.name as league_name,
    l.slug as league_slug,
    ht.name as home_team,
    ht.slug as home_team_slug,
    at.name as away_team,
    at.slug as away_team_slug,
    e.event_date,
    e.status,
    e.is_live,
    EXTRACT(
        EPOCH
        FROM
            (e.event_date - NOW())
    ) / 3600 as hours_to_kickoff,
    mt.code as market_code,
    mt.name as market_name,
    mt.slug as market_slug,
    co.outcome,
    co.opening_value,
    co.odds_value as current_value,
    co.highest_value,
    co.lowest_value,
    co.movement_percentage,
    co.total_movement,
    co.odds_value / NULLIF(co.opening_value, 0) as multiplier,
    CASE
        WHEN co.odds_value > co.opening_value THEN 'DRIFTING'
        WHEN co.odds_value < co.opening_value THEN 'SHORTENING'
        ELSE 'STABLE'
    END as trend_direction,
    CASE
        WHEN ABS(co.movement_percentage) >= 50 THEN 'EXTREME'
        WHEN ABS(co.movement_percentage) >= 30 THEN 'SIGNIFICANT'
        WHEN ABS(co.movement_percentage) >= 20 THEN 'NOTABLE'
        ELSE 'MODERATE'
    END as movement_strength,
    e.betting_volume_percentage,
    e.volume_rank,
    co.last_updated
FROM
    current_odds co
    JOIN events e ON co.event_id = e.id
    JOIN teams ht ON e.home_team_id = ht.id
    JOIN teams at ON e.away_team_id = at.id
    JOIN market_types mt ON co.market_type_id = mt.id
    JOIN leagues l ON e.league_id = l.id
    JOIN sports s ON l.sport_id = s.id
WHERE
    (
        ABS(co.movement_percentage) > 20
        OR co.odds_value / NULLIF(co.opening_value, 0) > 2
        OR co.opening_value / NULLIF(co.odds_value, 0) > 2
    )
    AND e.status IN ('scheduled', 'live')
    AND e.event_date > NOW() - INTERVAL '2 hours' -- Include recently started matches
ORDER BY
    ABS(co.movement_percentage) DESC;

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_current_odds_movement_percentage ON current_odds(ABS(movement_percentage) DESC);

CREATE INDEX IF NOT EXISTS idx_big_movers_event_id ON big_movers(event_id);

CREATE INDEX IF NOT EXISTS idx_big_movers_movement ON big_movers(movement_percentage DESC);

CREATE INDEX IF NOT EXISTS idx_big_movers_sport ON big_movers(sport_slug);

-- ============================================================================
-- CONTRARIAN BETS MATERIALIZED VIEW
-- Purpose: Identifies opportunities to bet against heavy public backing,
-- based on the theory that the public often overvalues popular teams.
--
-- Use Cases:
-- - Fade heavily backed favorites
-- - Spot value on unpopular underdogs
-- - Identify public bias in betting markets
--
-- Key Metrics:
-- - Public backing > 60% on one outcome
-- - Overbet percentage (difference between public % and implied probability)
-- - Combines with odds movement for stronger signals
-- ============================================================================
CREATE MATERIALIZED VIEW contrarian_bets AS
SELECT
    e.id as event_id,
    e.slug as event_slug,
    e.external_id as event_external_id,
    s.name as sport_name,
    s.slug as sport_slug,
    l.name as league_name,
    l.slug as league_slug,
    l.country,
    ht.name as home_team,
    ht.slug as home_team_slug,
    at.name as away_team,
    at.slug as away_team_slug,
    ht.name || ' vs ' || at.name as match_name,
    e.event_date,
    EXTRACT(
        EPOCH
        FROM
            (e.event_date - NOW())
    ) / 3600 as hours_to_kickoff,
    mt.name as market_name,
    mt.slug as market_slug,
    od.outcome as public_choice,
    od.bet_percentage as public_percentage,
    co.odds_value as current_odds,
    co.opening_value as opening_odds,
    co.movement_percentage as odds_movement,
    -- Calculate how much the public is overbetting this outcome
    (od.bet_percentage - (100.0 / co.odds_value))::REAL as overbet_percentage,
    od.value_indicator,
    -- Categorize the strength of the contrarian signal
    CASE
        WHEN od.bet_percentage > 80
        AND co.movement_percentage < -5 THEN 'EXTREME_CONTRARIAN'
        WHEN od.bet_percentage > 75 THEN 'STRONG_CONTRARIAN'
        WHEN od.bet_percentage > 65 THEN 'MODERATE_CONTRARIAN'
        ELSE 'MILD_CONTRARIAN'
    END as signal_strength,
    -- Provide opposite outcome for contrarian bet
    CASE
        WHEN od.outcome = '1' THEN 'Bet Draw (X) or Away (2)'
        WHEN od.outcome = '2' THEN 'Bet Draw (X) or Home (1)'
        WHEN od.outcome = 'X' THEN 'Bet Home (1) or Away (2)'
        WHEN od.outcome = 'Over' THEN 'Bet Under'
        WHEN od.outcome = 'Under' THEN 'Bet Over'
        ELSE 'Bet opposite of ' || od.outcome
    END as contrarian_play,
    NOW() as last_refreshed
FROM
    outcome_distributions od
    JOIN current_odds co ON od.event_id = co.event_id
    AND od.market_type_id = co.market_type_id
    AND od.outcome = co.outcome
    JOIN events e ON od.event_id = e.id
    JOIN teams ht ON e.home_team_id = ht.id
    JOIN teams at ON e.away_team_id = at.id
    JOIN leagues l ON e.league_id = l.id
    JOIN sports s ON l.sport_id = s.id
    JOIN market_types mt ON od.market_type_id = mt.id
WHERE
    od.bet_percentage > 60
    AND e.event_date > NOW()
    AND e.status = 'scheduled'
    AND (od.bet_percentage - (100.0 / co.odds_value)) > 15
ORDER BY
    od.bet_percentage DESC,
    (od.bet_percentage - (100.0 / co.odds_value)) DESC;

-- Create indexes for materialized view
CREATE INDEX IF NOT EXISTS idx_contrarian_bets_event_id ON contrarian_bets(event_id);

CREATE INDEX IF NOT EXISTS idx_contrarian_bets_signal_strength ON contrarian_bets(signal_strength);

CREATE INDEX IF NOT EXISTS idx_contrarian_bets_sport ON contrarian_bets(sport_slug);

-- ============================================================================
-- SHARP MONEY MOVEMENTS VIEW
-- Purpose: Tracks odds movements that indicate professional/sharp bettor activity,
-- especially reverse line movements where odds move against public betting.
--
-- Use Cases:
-- - Follow smart money patterns
-- - Identify insider betting activity
-- - Spot value before lines adjust
--
-- Key Metrics:
-- - Sharp money indicator (0-1 scale)
-- - Reverse line movements
-- - Significance levels (normal, high, extreme)
-- ============================================================================
CREATE MATERIALIZED VIEW sharp_money_moves AS
SELECT
    e.id as event_id,
    e.slug as event_slug,
    e.external_id as event_external_id,
    s.name as sport_name,
    s.slug as sport_slug,
    l.name as league_name,
    l.slug as league_slug,
    ht.name || ' vs ' || at.name as match_name,
    ht.slug as home_team_slug,
    at.slug as away_team_slug,
    e.event_date,
    e.status,
    mt.name as market_name,
    mt.slug as market_slug,
    oh.outcome,
    oh.odds_value as current_odds,
    oh.previous_value as previous_odds,
    oh.change_percentage,
    oh.sharp_money_indicator,
    oh.is_reverse_movement,
    oh.significance_level,
    oh.minutes_to_kickoff,
    -- Categorize sharp money confidence
    CASE
        WHEN oh.sharp_money_indicator >= 0.9 THEN 'EXTREME_CONFIDENCE'
        WHEN oh.sharp_money_indicator >= 0.8 THEN 'VERY_HIGH_CONFIDENCE'
        WHEN oh.sharp_money_indicator >= 0.7 THEN 'HIGH_CONFIDENCE'
        WHEN oh.sharp_money_indicator >= 0.5 THEN 'MODERATE_CONFIDENCE'
        ELSE 'LOW_CONFIDENCE'
    END as sharp_confidence,
    -- Explain the signal
    CASE
        WHEN oh.is_reverse_movement
        AND oh.sharp_money_indicator >= 0.8 THEN 'Strong reverse movement with high sharp indicator'
        WHEN oh.is_reverse_movement THEN 'Line moving against public money'
        WHEN oh.significance_level = 'extreme' THEN 'Extreme significance movement detected'
        WHEN oh.sharp_money_indicator >= 0.8 THEN 'High sharp money activity'
        ELSE 'Notable sharp activity'
    END as signal_description,
    oh.recorded_at
FROM
    odds_history oh
    JOIN events e ON oh.event_id = e.id
    JOIN teams ht ON e.home_team_id = ht.id
    JOIN teams at ON e.away_team_id = at.id
    JOIN market_types mt ON oh.market_type_id = mt.id
    JOIN leagues l ON e.league_id = l.id
    JOIN sports s ON l.sport_id = s.id
WHERE
    (
        oh.sharp_money_indicator > 0.5
        OR oh.is_reverse_movement = true
        OR oh.significance_level IN ('high', 'extreme')
    )
    AND e.event_date > NOW()
    AND oh.recorded_at > NOW() - INTERVAL '24 hours'
ORDER BY
    oh.sharp_money_indicator DESC,
    oh.recorded_at DESC;

-- ============================================================================
-- LIVE BETTING OPPORTUNITIES VIEW
-- Purpose: Identifies valuable in-play betting opportunities based on live
-- odds movements and betting patterns during matches.
--
-- Use Cases:
-- - React to in-game developments
-- - Spot overreactions in live markets
-- - Track momentum shifts during matches
--
-- Key Metrics:
-- - Live odds movements > 10%
-- - Current match situation (score, minute)
-- - Public betting shifts during the game
-- ============================================================================
CREATE MATERIALIZED VIEW live_opportunities AS
SELECT
    e.id as event_id,
    e.slug as event_slug,
    e.external_id as event_external_id,
    s.name as sport_name,
    s.slug as sport_slug,
    l.name as league_name,
    l.slug as league_slug,
    ht.name as home_team,
    ht.slug as home_team_slug,
    at.name as away_team,
    at.slug as away_team_slug,
    e.home_score,
    e.away_score,
    e.minute_of_match,
    e.half,
    e.status,
    mt.name as market_name,
    mt.slug as market_slug,
    co.outcome,
    co.odds_value as current_odds,
    co.opening_value as pre_match_odds,
    co.movement_percentage as total_movement,
    -- Calculate in-play specific movement
    (
        (
            (co.odds_value - co.opening_value) / NULLIF(co.opening_value, 0)
        ) * 100
    )::REAL as live_movement_pct,
    od.bet_percentage as current_backing,
    e.betting_volume_percentage,
    -- Categorize opportunity type
    CASE
        WHEN e.home_score > e.away_score
        AND co.outcome = '2'
        AND co.movement_percentage > 20 THEN 'Away team value (losing but odds drifting)'
        WHEN e.away_score > e.home_score
        AND co.outcome = '1'
        AND co.movement_percentage > 20 THEN 'Home team value (losing but odds drifting)'
        WHEN e.minute_of_match < 30
        AND ABS(co.movement_percentage) > 25 THEN 'Early overreaction'
        WHEN e.minute_of_match > 70
        AND ABS(co.movement_percentage) > 15 THEN 'Late game opportunity'
        ELSE 'Live value detected'
    END as opportunity_type,
    co.last_updated
FROM
    events e
    JOIN current_odds co ON e.id = co.event_id
    JOIN outcome_distributions od ON e.id = od.event_id
    AND co.market_type_id = od.market_type_id
    AND co.outcome = od.outcome
    JOIN teams ht ON e.home_team_id = ht.id
    JOIN teams at ON e.away_team_id = at.id
    JOIN market_types mt ON co.market_type_id = mt.id
    JOIN leagues l ON e.league_id = l.id
    JOIN sports s ON l.sport_id = s.id
WHERE
    e.is_live = true
    AND e.status = 'live'
    AND ABS(co.movement_percentage) > 10
    AND co.last_updated > NOW() - INTERVAL '5 minutes' -- Recent movements only
ORDER BY
    ABS(co.movement_percentage) DESC;

-- ============================================================================
-- VALUE BETTING SPOTS VIEW
-- Purpose: Combines multiple indicators to identify the highest-value betting
-- opportunities across all markets.
--
-- Use Cases:
-- - Find best overall betting value
-- - Combine sharp money, public bias, and odds movement
-- - Prioritize bets by expected value
--
-- Key Metrics:
-- - Public bias (bet % vs implied probability)
-- - Sharp money indicators
-- - Odds movement patterns
-- - Combined value score
-- ============================================================================
CREATE MATERIALIZED VIEW value_spots AS
SELECT
    e.id as event_id,
    e.slug as event_slug,
    e.external_id as event_external_id,
    s.name as sport_name,
    s.slug as sport_slug,
    l.name as league_name,
    l.slug as league_slug,
    ht.name || ' vs ' || at.name as match_name,
    ht.slug as home_team_slug,
    at.slug as away_team_slug,
    e.event_date,
    EXTRACT(
        EPOCH
        FROM
            (e.event_date - NOW())
    ) / 3600 as hours_to_kickoff,
    mt.code as market_code,
    mt.name as market_name,
    mt.slug as market_slug,
    co.outcome,
    co.odds_value as current_odds,
    co.opening_value as opening_odds,
    co.movement_percentage,
    od.bet_percentage,
    od.implied_probability,
    -- Key value indicators
    (od.bet_percentage - od.implied_probability) as public_bias,
    COALESCE(
        (
            SELECT
                MAX(sharp_money_indicator)
            FROM
                odds_history
            WHERE
                event_id = e.id
                AND market_type_id = mt.id
                AND outcome = co.outcome
                AND recorded_at > NOW() - INTERVAL '6 hours'
        ),
        0
    ) as max_sharp_indicator,
    -- Calculate composite value score
    (
        CASE
            WHEN (od.bet_percentage - od.implied_probability) < -10 -- Underbet
            AND co.movement_percentage > 5 -- Odds drifting
            THEN 0.8
            WHEN (od.bet_percentage - od.implied_probability) > 15 -- Overbet
            AND co.movement_percentage < -5 -- Odds shortening against public
            THEN 0.9
            ELSE 0.5
        END * 100
    )::INTEGER as value_score,
    -- Recommend action
    CASE
        WHEN (od.bet_percentage - od.implied_probability) < -10 THEN 'BET (Undervalued by public)'
        WHEN (od.bet_percentage - od.implied_probability) > 15 THEN 'FADE (Overvalued by public)'
        ELSE 'MONITOR'
    END as recommendation,
    co.last_updated
FROM
    current_odds co
    JOIN events e ON co.event_id = e.id
    JOIN teams ht ON e.home_team_id = ht.id
    JOIN teams at ON e.away_team_id = at.id
    JOIN market_types mt ON co.market_type_id = mt.id
    JOIN leagues l ON e.league_id = l.id
    JOIN sports s ON l.sport_id = s.id
    LEFT JOIN outcome_distributions od ON (
        co.event_id = od.event_id
        AND co.market_type_id = od.market_type_id
        AND co.outcome = od.outcome
    )
WHERE
    e.event_date > NOW()
    AND e.status = 'scheduled'
    AND od.bet_percentage IS NOT NULL
    AND (
        ABS(od.bet_percentage - od.implied_probability) > 10
        OR ABS(co.movement_percentage) > 15
    )
ORDER BY
    ABS(od.bet_percentage - od.implied_probability) DESC,
    ABS(co.movement_percentage) DESC;

-- ============================================================================
-- HIGH VOLUME EVENTS VIEW
-- Purpose: Tracks events with unusually high betting volume, which often
-- indicates important matches or insider activity.
--
-- Use Cases:
-- - Focus on most popular/important matches
-- - Spot unusual betting interest
-- - Track volume trends
--
-- Key Metrics:
-- - Betting volume percentage and rank
-- - Volume changes over time
-- - Comparison to average volume
-- ============================================================================
CREATE MATERIALIZED VIEW high_volume_events AS
SELECT
    e.id as event_id,
    e.slug as event_slug,
    e.external_id as event_external_id,
    s.name as sport_name,
    s.slug as sport_slug,
    l.name as league_name,
    l.slug as league_slug,
    ht.name as home_team,
    ht.slug as home_team_slug,
    at.name as away_team,
    at.slug as away_team_slug,
    e.event_date,
    e.status,
    e.betting_volume_percentage,
    e.volume_rank,
    -- Get latest volume change
    (
        SELECT
            bvh1.volume_percentage - bvh2.volume_percentage
        FROM
            betting_volume_history bvh1
            JOIN betting_volume_history bvh2 ON bvh1.event_id = bvh2.event_id
        WHERE
            bvh1.event_id = e.id
            AND bvh1.recorded_at = (
                SELECT
                    MAX(recorded_at)
                FROM
                    betting_volume_history
                WHERE
                    event_id = e.id
            )
            AND bvh2.recorded_at = (
                SELECT
                    MAX(recorded_at)
                FROM
                    betting_volume_history
                WHERE
                    event_id = e.id
                    AND recorded_at < bvh1.recorded_at
            )
    ) as recent_volume_change,
    e.volume_updated_at,
    -- Categorize volume level
    CASE
        WHEN e.volume_rank <= 5 THEN 'TOP_5_VOLUME'
        WHEN e.volume_rank <= 10 THEN 'TOP_10_VOLUME'
        WHEN e.betting_volume_percentage > 5 THEN 'HIGH_VOLUME'
        WHEN e.betting_volume_percentage > 2 THEN 'ABOVE_AVERAGE_VOLUME'
        ELSE 'NORMAL_VOLUME'
    END as volume_category
FROM
    events e
    JOIN teams ht ON e.home_team_id = ht.id
    JOIN teams at ON e.away_team_id = at.id
    JOIN leagues l ON e.league_id = l.id
    JOIN sports s ON l.sport_id = s.id
WHERE
    e.betting_volume_percentage > 2
    AND e.event_date > NOW()
    AND e.status IN ('scheduled', 'live')
ORDER BY
    e.betting_volume_percentage DESC;

-- Create necessary indexes for view performance
CREATE INDEX IF NOT EXISTS idx_events_volume_rank ON events(volume_rank)
WHERE
    volume_rank IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_events_betting_volume ON events(betting_volume_percentage DESC)
WHERE
    betting_volume_percentage IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_odds_history_sharp_money ON odds_history(sharp_money_indicator DESC)
WHERE
    sharp_money_indicator > 0;

-- Indexes for sharp_money_moves materialized view
CREATE INDEX IF NOT EXISTS idx_sharp_money_moves_event_id ON sharp_money_moves(event_id);

CREATE INDEX IF NOT EXISTS idx_sharp_money_moves_indicator ON sharp_money_moves(sharp_money_indicator DESC);

CREATE INDEX IF NOT EXISTS idx_sharp_money_moves_sport ON sharp_money_moves(sport_slug);

CREATE INDEX IF NOT EXISTS idx_odds_history_reverse_movement ON odds_history(is_reverse_movement)
WHERE
    is_reverse_movement = true;

CREATE INDEX IF NOT EXISTS idx_outcome_distributions_public_bias ON outcome_distributions((bet_percentage - implied_probability));

-- Indexes for remaining materialized views
CREATE INDEX IF NOT EXISTS idx_live_opportunities_event_id ON live_opportunities(event_id);

CREATE INDEX IF NOT EXISTS idx_live_opportunities_movement ON live_opportunities(total_movement DESC);

CREATE INDEX IF NOT EXISTS idx_live_opportunities_sport ON live_opportunities(sport_slug);

CREATE INDEX IF NOT EXISTS idx_value_spots_event_id ON value_spots(event_id);

CREATE INDEX IF NOT EXISTS idx_value_spots_value_score ON value_spots(value_score DESC);

CREATE INDEX IF NOT EXISTS idx_value_spots_sport ON value_spots(sport_slug);

CREATE INDEX IF NOT EXISTS idx_high_volume_events_event_id ON high_volume_events(event_id);

CREATE INDEX IF NOT EXISTS idx_high_volume_events_volume ON high_volume_events(betting_volume_percentage DESC);

CREATE INDEX IF NOT EXISTS idx_high_volume_events_sport ON high_volume_events(sport_slug);