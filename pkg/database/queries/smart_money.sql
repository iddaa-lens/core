-- Smart Money Tracker queries
-- name: GetRecentBigMovers :many
SELECT
    oh.*,
    e.external_id as event_external_id,
    e.event_date,
    e.home_team_id,
    e.away_team_id,
    e.is_live,
    ht.name as home_team_name,
    at.name as away_team_name,
    mt.code as market_code,
    mt.name as market_name
FROM
    odds_history oh
    JOIN events e ON oh.event_id = e.id
    LEFT JOIN teams ht ON e.home_team_id = ht.id
    LEFT JOIN teams at ON e.away_team_id = at.id
    JOIN market_types mt ON oh.market_type_id = mt.id
WHERE
    (
        ABS(oh.change_percentage) >= sqlc.arg(min_change_pct)::float8
        OR oh.multiplier >= sqlc.arg(min_multiplier)::float8
    )
    AND oh.recorded_at >= sqlc.arg(since_time)::timestamp
    AND e.event_date > NOW()
ORDER BY
    oh.recorded_at DESC
LIMIT
    sqlc.arg(limit_count)::int;

-- name: GetReverseLineMovements :many
-- Detect TRUE reverse line movements where odds move against public betting percentages
SELECT
    oh.*,
    e.external_id as event_external_id,
    e.event_date,
    e.home_team_id,
    e.away_team_id,
    e.betting_volume_percentage,
    ht.name as home_team_name,
    at.name as away_team_name,
    mt.code as market_code,
    mt.name as market_name,
    od.bet_percentage,
    od.implied_probability,
    -- Calculate the divergence between public betting and odds movement
    CASE 
        WHEN od.bet_percentage > 60 AND oh.change_percentage > 0 THEN 'public_heavy_odds_worse'
        WHEN od.bet_percentage < 40 AND oh.change_percentage < 0 THEN 'public_light_odds_better'
        ELSE 'normal'
    END as movement_type,
    -- Strength of reverse movement (public % * abs(odds change %))
    (od.bet_percentage * ABS(oh.change_percentage) / 100) as reverse_strength
FROM
    odds_history oh
    JOIN events e ON oh.event_id = e.id
    LEFT JOIN teams ht ON e.home_team_id = ht.id
    LEFT JOIN teams at ON e.away_team_id = at.id
    JOIN market_types mt ON oh.market_type_id = mt.id
    LEFT JOIN outcome_distributions od ON (
        oh.event_id = od.event_id 
        AND oh.market_type_id = od.market_type_id
        AND oh.outcome = od.outcome
    )
WHERE
    oh.recorded_at >= sqlc.arg(since_time)
    AND e.event_date > NOW()
    AND od.bet_percentage IS NOT NULL
    -- True reverse line movements:
    AND (
        -- Heavy public betting (>65%) but odds getting worse (going up)
        (od.bet_percentage > 65 AND oh.change_percentage > 5)
        OR
        -- Light public betting (<35%) but odds getting better (going down)
        (od.bet_percentage < 35 AND oh.change_percentage < -5)
    )
    -- Only significant movements
    AND ABS(oh.change_percentage) >= 5
ORDER BY
    (od.bet_percentage * ABS(oh.change_percentage) / 100) DESC
LIMIT
    sqlc.arg(limit_count);

-- name: GetValueSpots :many
SELECT
    oh.*,
    od.bet_percentage,
    od.implied_probability,
    e.external_id as event_external_id,
    e.event_date,
    e.home_team_id,
    e.away_team_id,
    ht.name as home_team_name,
    at.name as away_team_name,
    mt.code as market_code,
    mt.name as market_name,
    (od.bet_percentage - od.implied_probability) as public_bias
FROM
    odds_history oh
    JOIN events e ON oh.event_id = e.id
    LEFT JOIN teams ht ON e.home_team_id = ht.id
    LEFT JOIN teams at ON e.away_team_id = at.id
    JOIN market_types mt ON oh.market_type_id = mt.id
    LEFT JOIN outcome_distributions od ON (
        oh.event_id = od.event_id
        AND oh.market_type_id = od.market_type_id
        AND oh.outcome = od.outcome
    )
WHERE
    oh.recorded_at >= sqlc.arg(since_time)
    AND e.event_date > NOW()
    AND od.bet_percentage > od.implied_probability + sqlc.arg(min_bias_pct)::float8
    AND ABS(oh.change_percentage) >= sqlc.arg(min_movement_pct)::float8
ORDER BY
    (od.bet_percentage - od.implied_probability) DESC
LIMIT
    sqlc.arg(limit_count);

-- name: CreateMovementAlert :one
INSERT INTO
    movement_alerts (
        odds_history_id,
        alert_type,
        severity,
        title,
        message,
        change_percentage,
        multiplier,
        confidence_score,
        minutes_to_kickoff,
        expires_at
    )
VALUES
    (
        sqlc.arg(odds_history_id),
        sqlc.arg(alert_type),
        sqlc.arg(severity),
        sqlc.arg(title),
        sqlc.arg(message),
        sqlc.arg(change_percentage),
        sqlc.arg(multiplier),
        sqlc.arg(confidence_score),
        sqlc.arg(minutes_to_kickoff),
        CURRENT_TIMESTAMP + INTERVAL '24 hours'
    ) ON CONFLICT (odds_history_id, alert_type) DO
UPDATE
SET
    severity = EXCLUDED.severity,
    title = EXCLUDED.title,
    message = EXCLUDED.message,
    change_percentage = EXCLUDED.change_percentage,
    multiplier = EXCLUDED.multiplier,
    confidence_score = EXCLUDED.confidence_score,
    minutes_to_kickoff = EXCLUDED.minutes_to_kickoff,
    expires_at = CURRENT_TIMESTAMP + INTERVAL '24 hours',
    updated_at = CURRENT_TIMESTAMP RETURNING *;

-- name: GetActiveAlerts :many
SELECT
    ma.*,
    oh.event_id,
    oh.market_type_id,
    oh.outcome,
    e.external_id as event_external_id,
    e.event_date,
    e.home_team_id,
    e.away_team_id,
    ht.name as home_team_name,
    at.name as away_team_name,
    mt.code as market_code,
    mt.name as market_name
FROM
    movement_alerts ma
    JOIN odds_history oh ON ma.odds_history_id = oh.id
    JOIN events e ON oh.event_id = e.id
    LEFT JOIN teams ht ON e.home_team_id = ht.id
    LEFT JOIN teams at ON e.away_team_id = at.id
    JOIN market_types mt ON oh.market_type_id = mt.id
WHERE
    ma.is_active = true
    AND ma.expires_at > NOW()
    AND (
        sqlc.arg(alert_type) = ''
        OR ma.alert_type = sqlc.arg(alert_type)
    )
    AND (
        sqlc.arg(min_severity) = ''
        OR (
            ma.severity = 'critical'
            AND sqlc.arg(min_severity) IN ('low', 'medium', 'high', 'critical')
        )
        OR (
            ma.severity = 'high'
            AND sqlc.arg(min_severity) IN ('low', 'medium', 'high')
        )
        OR (
            ma.severity = 'medium'
            AND sqlc.arg(min_severity) IN ('low', 'medium')
        )
        OR (
            ma.severity = 'low'
            AND sqlc.arg(min_severity) = 'low'
        )
    )
ORDER BY
    CASE
        ma.severity
        WHEN 'critical' THEN 4
        WHEN 'high' THEN 3
        WHEN 'medium' THEN 2
        ELSE 1
    END DESC,
    ma.created_at DESC
LIMIT
    sqlc.arg(limit_count);

-- COMMENTED OUT: Requires smart_money_preferences table
-- -- name: GetAlertsByUser :many
-- SELECT 
--     ma.*,
--     oh.event_id,
--     oh.outcome,
--     e.external_id as event_external_id,
--     e.event_date,
--     ht.name as home_team_name,
--     at.name as away_team_name,
--     mt.name as market_name
-- FROM movement_alerts ma
-- JOIN odds_history oh ON ma.odds_history_id = oh.id
-- JOIN events e ON oh.event_id = e.id
-- LEFT JOIN teams ht ON e.home_team_id = ht.id
-- LEFT JOIN teams at ON e.away_team_id = at.id
-- JOIN market_types mt ON oh.market_type_id = mt.id
-- JOIN smart_money_preferences smp ON smp.user_id = sqlc.arg(user_id)
-- WHERE 
--     ma.is_active = true
--     AND ma.expires_at > NOW()
--     AND ma.change_percentage >= smp.min_change_percentage
--     AND ma.multiplier >= smp.min_multiplier
--     AND ma.confidence_score >= smp.min_confidence_score
--     AND (
--         (ma.alert_type = 'big_mover' AND smp.big_mover_alerts = true) OR
--         (ma.alert_type = 'reverse_line' AND smp.reverse_line_alerts = true) OR
--         (ma.alert_type = 'sharp_money' AND smp.sharp_money_alerts = true) OR
--         (ma.alert_type = 'value_spot' AND smp.value_spot_alerts = true)
--     )
--     -- Filter by preferred sports/leagues if specified
--     AND (
--         smp.preferred_sports = '[]'::jsonb OR 
--         e.sport_id::text = ANY(SELECT jsonb_array_elements_text(smp.preferred_sports))
--     )
-- ORDER BY ma.created_at DESC
-- LIMIT sqlc.arg(limit_count);
-- name: MarkAlertViewed :exec
UPDATE
    movement_alerts
SET
    views = views + 1
WHERE
    id = sqlc.arg(alert_id);

-- name: MarkAlertClicked :exec
UPDATE
    movement_alerts
SET
    clicks = clicks + 1
WHERE
    id = sqlc.arg(alert_id);

-- name: DeactivateExpiredAlerts :exec
UPDATE
    movement_alerts
SET
    is_active = false
WHERE
    expires_at <= NOW()
    AND is_active = true;

-- name: GetSteamMoves :many
-- Detect rapid odds movements across multiple bookmakers (steam moves)
SELECT
    oh.*,
    e.external_id as event_external_id,
    e.event_date,
    e.betting_volume_percentage,
    ht.name as home_team_name,
    at.name as away_team_name,
    mt.code as market_code,
    mt.name as market_name,
    -- Time since last movement
    EXTRACT(EPOCH FROM (oh.recorded_at - LAG(oh.recorded_at) OVER (
        PARTITION BY oh.event_id, oh.market_type_id, oh.outcome 
        ORDER BY oh.recorded_at
    ))) as seconds_since_last_move,
    -- Running total of movements in last hour
    COUNT(*) OVER (
        PARTITION BY oh.event_id, oh.market_type_id, oh.outcome 
        ORDER BY oh.recorded_at 
        RANGE BETWEEN INTERVAL '1 hour' PRECEDING AND CURRENT ROW
    ) as movements_last_hour
FROM
    odds_history oh
    JOIN events e ON oh.event_id = e.id
    LEFT JOIN teams ht ON e.home_team_id = ht.id
    LEFT JOIN teams at ON e.away_team_id = at.id
    JOIN market_types mt ON oh.market_type_id = mt.id
WHERE
    oh.recorded_at >= sqlc.arg(since_time)
    AND e.event_date > NOW()
    -- Significant movement
    AND ABS(oh.change_percentage) >= 3
    -- Multiple movements in short time indicates steam
    AND oh.event_id IN (
        SELECT event_id 
        FROM odds_history 
        WHERE recorded_at >= sqlc.arg(since_time)
        GROUP BY event_id, market_type_id, outcome
        HAVING COUNT(*) >= 3 -- At least 3 movements
    )
ORDER BY
    oh.recorded_at DESC
LIMIT
    sqlc.arg(limit_count);

-- name: GetSharpMoneyIndicators :many
-- Comprehensive sharp money detection combining multiple factors
SELECT
    oh.*,
    e.external_id as event_external_id,
    e.event_date,
    e.betting_volume_percentage,
    e.volume_rank,
    ht.name as home_team_name,
    at.name as away_team_name,
    mt.code as market_code,
    mt.name as market_name,
    od.bet_percentage,
    od.implied_probability,
    -- Sharp money score calculation
    (
        -- Reverse line movement factor (0-40 points)
        CASE 
            WHEN od.bet_percentage > 70 AND oh.change_percentage > 5 THEN 40
            WHEN od.bet_percentage > 60 AND oh.change_percentage > 3 THEN 30
            WHEN od.bet_percentage < 30 AND oh.change_percentage < -5 THEN 40
            WHEN od.bet_percentage < 40 AND oh.change_percentage < -3 THEN 30
            ELSE 0
        END +
        -- Volume factor (0-20 points) - lower volume with movement = sharper
        CASE
            WHEN e.betting_volume_percentage < 1 AND ABS(oh.change_percentage) > 10 THEN 20
            WHEN e.betting_volume_percentage < 2 AND ABS(oh.change_percentage) > 7 THEN 15
            WHEN e.betting_volume_percentage < 5 AND ABS(oh.change_percentage) > 5 THEN 10
            ELSE 0
        END +
        -- Timing factor (0-20 points) - late movement = sharper
        CASE
            WHEN EXTRACT(EPOCH FROM (e.event_date - oh.recorded_at)) / 3600 < 2 THEN 20
            WHEN EXTRACT(EPOCH FROM (e.event_date - oh.recorded_at)) / 3600 < 6 THEN 15
            WHEN EXTRACT(EPOCH FROM (e.event_date - oh.recorded_at)) / 3600 < 24 THEN 10
            ELSE 5
        END +
        -- Movement size factor (0-20 points)
        CASE
            WHEN ABS(oh.change_percentage) > 20 THEN 20
            WHEN ABS(oh.change_percentage) > 15 THEN 15
            WHEN ABS(oh.change_percentage) > 10 THEN 10
            WHEN ABS(oh.change_percentage) > 5 THEN 5
            ELSE 0
        END
    ) as sharp_money_score
FROM
    odds_history oh
    JOIN events e ON oh.event_id = e.id
    LEFT JOIN teams ht ON e.home_team_id = ht.id
    LEFT JOIN teams at ON e.away_team_id = at.id
    JOIN market_types mt ON oh.market_type_id = mt.id
    LEFT JOIN outcome_distributions od ON (
        oh.event_id = od.event_id 
        AND oh.market_type_id = od.market_type_id
        AND oh.outcome = od.outcome
    )
WHERE
    oh.recorded_at >= sqlc.arg(since_time)
    AND e.event_date > NOW()
    AND ABS(oh.change_percentage) >= 5
ORDER BY
    sharp_money_score DESC
LIMIT
    sqlc.arg(limit_count);

-- COMMENTED OUT: Requires smart_money_preferences table
-- -- name: GetUserSmartMoneyPreferences :one
-- SELECT * FROM smart_money_preferences WHERE user_id = sqlc.arg(user_id);
-- -- name: UpsertUserSmartMoneyPreferences :one
-- INSERT INTO smart_money_preferences (
--     user_id,
--     min_change_percentage,
--     min_multiplier,
--     min_confidence_score,
--     big_mover_alerts,
--     reverse_line_alerts,
--     sharp_money_alerts,
--     value_spot_alerts,
--     preferred_sports,
--     preferred_leagues,
--     max_daily_alerts,
--     push_notifications,
--     quiet_hours_start,
--     quiet_hours_end
-- ) VALUES (
--     sqlc.arg(user_id),
--     sqlc.arg(min_change_percentage),
--     sqlc.arg(min_multiplier),
--     sqlc.arg(min_confidence_score),
--     sqlc.arg(big_mover_alerts),
--     sqlc.arg(reverse_line_alerts),
--     sqlc.arg(sharp_money_alerts),
--     sqlc.arg(value_spot_alerts),
--     sqlc.arg(preferred_sports),
--     sqlc.arg(preferred_leagues),
--     sqlc.arg(max_daily_alerts),
--     sqlc.arg(push_notifications),
--     sqlc.arg(quiet_hours_start),
--     sqlc.arg(quiet_hours_end)
-- )
-- ON CONFLICT (user_id) DO UPDATE SET
--     min_change_percentage = EXCLUDED.min_change_percentage,
--     min_multiplier = EXCLUDED.min_multiplier,
--     min_confidence_score = EXCLUDED.min_confidence_score,
--     big_mover_alerts = EXCLUDED.big_mover_alerts,
--     reverse_line_alerts = EXCLUDED.reverse_line_alerts,
--     sharp_money_alerts = EXCLUDED.sharp_money_alerts,
--     value_spot_alerts = EXCLUDED.value_spot_alerts,
--     preferred_sports = EXCLUDED.preferred_sports,
--     preferred_leagues = EXCLUDED.preferred_leagues,
--     max_daily_alerts = EXCLUDED.max_daily_alerts,
--     push_notifications = EXCLUDED.push_notifications,
--     quiet_hours_start = EXCLUDED.quiet_hours_start,
--     quiet_hours_end = EXCLUDED.quiet_hours_end,
--     updated_at = CURRENT_TIMESTAMP
-- RETURNING *;