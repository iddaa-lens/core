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
SELECT
    oh.*,
    e.external_id as event_external_id,
    e.event_date,
    e.home_team_id,
    e.away_team_id,
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
    oh.recorded_at >= sqlc.arg(since_time)
    AND e.event_date > NOW() -- Look for movements against typical patterns
    AND (
        (
            oh.previous_value < 2.0
            AND oh.change_percentage > 15
        )
        OR -- Favorites drifting
        (
            oh.previous_value > 4.0
            AND oh.change_percentage < -15
        ) -- Big underdogs shortening
    )
ORDER BY
    ABS(oh.change_percentage) DESC
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