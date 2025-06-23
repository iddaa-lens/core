-- name: GetCurrentOdds :many
SELECT
    co.*,
    mt.name as market_name,
    mt.code as market_code
FROM
    current_odds co
    JOIN market_types mt ON co.market_type_id = mt.id
WHERE
    co.event_id = sqlc.arg(event_id)::int;

-- name: GetCurrentOddsByMarket :many
SELECT
    co.*,
    mt.name as market_name,
    mt.code as market_code
FROM
    current_odds co
    JOIN market_types mt ON co.market_type_id = mt.id
WHERE
    co.event_id = sqlc.arg(event_id)::int
    AND co.market_type_id = sqlc.arg(market_type_id)::int;

-- name: GetCurrentOddsByOutcome :one
SELECT
    co.*,
    mt.name as market_name,
    mt.code as market_code
FROM
    current_odds co
    JOIN market_types mt ON co.market_type_id = mt.id
WHERE
    co.event_id = sqlc.arg(event_id)::int
    AND co.market_type_id = sqlc.arg(market_type_id)::int
    AND co.outcome = sqlc.arg(outcome)::text;

-- name: UpsertCurrentOdds :one
INSERT INTO
    current_odds (
        event_id,
        market_type_id,
        outcome,
        odds_value,
        opening_value,
        highest_value,
        lowest_value,
        winning_odds,
        total_movement,
        movement_percentage,
        market_params
    )
VALUES
    (
        sqlc.arg(event_id)::int,
        sqlc.arg(market_type_id)::int,
        sqlc.arg(outcome)::text,
        sqlc.arg(odds_value)::decimal,
        sqlc.arg(opening_value)::decimal,
        sqlc.arg(highest_value)::decimal,
        sqlc.arg(lowest_value)::decimal,
        sqlc.arg(winning_odds)::decimal,
        0,
        -- First time, no movement
        0,
        -- First time, no movement percentage
        sqlc.arg(market_params)::jsonb
    ) ON CONFLICT (event_id, market_type_id, outcome) DO
UPDATE
SET
    odds_value = EXCLUDED.odds_value,
    winning_odds = EXCLUDED.winning_odds,
    highest_value = GREATEST(current_odds.highest_value, EXCLUDED.odds_value),
    lowest_value = LEAST(current_odds.lowest_value, EXCLUDED.odds_value),
    -- Calculate movement: new_odds - opening_odds
    total_movement = EXCLUDED.odds_value - current_odds.opening_value,
    -- Calculate movement percentage: ((new_odds - opening_odds) / opening_odds) * 100
    movement_percentage = CASE
        WHEN current_odds.opening_value > 0 THEN (
            (
                (EXCLUDED.odds_value - current_odds.opening_value) / current_odds.opening_value * 100
            )
        )::REAL
        ELSE 0
    END,
    market_params = EXCLUDED.market_params,
    last_updated = CURRENT_TIMESTAMP RETURNING *;

-- name: CreateOddsHistory :one
INSERT INTO
    odds_history (
        event_id,
        market_type_id,
        outcome,
        odds_value,
        previous_value,
        winning_odds,
        change_amount,
        change_percentage,
        multiplier,
        market_params
    )
VALUES
    (
        sqlc.arg(event_id)::int,
        sqlc.arg(market_type_id)::int,
        sqlc.arg(outcome)::text,
        sqlc.arg(odds_value)::decimal,
        sqlc.arg(previous_value)::decimal,
        sqlc.arg(winning_odds)::decimal,
        -- Calculate change amount: new_odds - previous_odds
        sqlc.arg(odds_value)::decimal - sqlc.arg(previous_value)::decimal,
        -- Calculate change percentage: ((new_odds - previous_odds) / previous_odds) * 100
        CASE
            WHEN sqlc.arg(previous_value)::decimal > 0 THEN (
                (
                    (
                        sqlc.arg(odds_value)::decimal - sqlc.arg(previous_value)::decimal
                    ) / sqlc.arg(previous_value)::decimal * 100
                )
            )::REAL
            ELSE 0
        END,
        -- Calculate multiplier: new_odds / previous_odds
        CASE
            WHEN sqlc.arg(previous_value)::decimal > 0 THEN (
                sqlc.arg(odds_value)::decimal / sqlc.arg(previous_value)::decimal
            )::DOUBLE PRECISION
            ELSE 1
        END,
        sqlc.arg(market_params)::jsonb
    ) RETURNING *;

-- name: GetOddsMovements :many
SELECT
    oh.*,
    mt.name as market_name,
    mt.code as market_code
FROM
    odds_history oh
    JOIN market_types mt ON oh.market_type_id = mt.id
WHERE
    oh.event_id = sqlc.arg(event_id)::int
ORDER BY
    oh.recorded_at DESC
LIMIT
    sqlc.arg(limit_count)::int;

-- name: GetOddsHistoryByID :one
SELECT
    *
FROM
    odds_history
WHERE
    id = sqlc.arg(id)::bigint;

-- name: GetBigMovers :many
SELECT
    oh.*,
    e.slug as event_slug,
    mt.code as market_code
FROM
    odds_history oh
    JOIN events e ON oh.event_id = e.id
    JOIN market_types mt ON oh.market_type_id = mt.id
WHERE
    ABS(oh.change_percentage) > sqlc.arg(min_change_pct)::float8
    AND oh.recorded_at > sqlc.arg(since_time)::timestamp
ORDER BY
    ABS(oh.change_percentage) DESC
LIMIT
    sqlc.arg(limit_count)::int;

-- name: GetRecentOddsHistory :many
SELECT
    oh.*,
    e.event_date,
    e.is_live,
    mt.name as market_name,
    mt.code as market_code
FROM
    odds_history oh
    JOIN events e ON oh.event_id = e.id
    JOIN market_types mt ON oh.market_type_id = mt.id
WHERE
    oh.recorded_at >= sqlc.arg(since_time)::timestamp
    AND e.event_date > NOW()
    AND ABS(oh.change_percentage) >= sqlc.arg(min_change_pct)::float8
ORDER BY
    oh.recorded_at DESC
LIMIT
    sqlc.arg(limit_count)::int;

-- name: BatchGetCurrentOdds :many
SELECT
    *
FROM
    current_odds
WHERE
    event_id = sqlc.arg(event_id)::int
    AND market_type_id = ANY(sqlc.arg(market_type_ids)::int)
    AND outcome = ANY(sqlc.arg(outcomes)::text[]);

-- name: BulkUpsertCurrentOdds :exec
WITH input_data AS (
    SELECT
        unnest(sqlc.arg(event_ids)::int[]) as event_id,
        unnest(sqlc.arg(market_type_ids)::int[]) as market_type_id,
        unnest(sqlc.arg(outcomes)::text[]) as outcome,
        unnest(sqlc.arg(odds_values)::float8[]) as odds_value,
        unnest(sqlc.arg(market_params)::jsonb[]) as market_params
)
INSERT INTO
    current_odds (
        event_id,
        market_type_id,
        outcome,
        odds_value,
        opening_value,
        highest_value,
        lowest_value,
        market_params,
        last_updated
    )
SELECT
    event_id,
    market_type_id,
    outcome,
    odds_value,
    odds_value,
    odds_value,
    odds_value,
    market_params,
    NOW()
FROM
    input_data ON CONFLICT (event_id, market_type_id, outcome) DO
UPDATE
SET
    odds_value = EXCLUDED.odds_value,
    highest_value = GREATEST(current_odds.highest_value, EXCLUDED.odds_value),
    lowest_value = LEAST(current_odds.lowest_value, EXCLUDED.odds_value),
    last_updated = NOW()
WHERE
    current_odds.odds_value IS DISTINCT
FROM
    EXCLUDED.odds_value;

-- name: BulkInsertOddsHistory :exec
WITH input_data AS (
    SELECT
        unnest(sqlc.arg(event_ids)::int[]) as event_id,
        unnest(sqlc.arg(market_type_ids)::int[]) as market_type_id,
        unnest(sqlc.arg(outcomes)::text[]) as outcome,
        unnest(sqlc.arg(odds_values)::float8[]) as odds_value,
        unnest(sqlc.arg(previous_values)::float8[]) as previous_value,
        unnest(sqlc.arg(change_amounts)::float8[]) as change_amount,
        unnest(sqlc.arg(change_percentages)::float8[]) as change_percentage,
        unnest(sqlc.arg(multipliers)::float8[]) as multiplier,
        unnest(sqlc.arg(is_reverse_movements)::boolean[]) as is_reverse_movement,
        unnest(sqlc.arg(significance_levels)::text[]) as significance_level,
        unnest(sqlc.arg(minutes_to_kickoffs)::int[]) as minutes_to_kickoff,
        unnest(sqlc.arg(market_params)::jsonb[]) as market_params
)
INSERT INTO
    odds_history (
        event_id,
        market_type_id,
        outcome,
        odds_value,
        previous_value,
        change_amount,
        change_percentage,
        multiplier,
        is_reverse_movement,
        significance_level,
        minutes_to_kickoff,
        market_params,
        recorded_at
    )
SELECT
    event_id,
    market_type_id,
    outcome,
    odds_value,
    previous_value,
    change_amount,
    change_percentage,
    multiplier,
    is_reverse_movement,
    significance_level,
    minutes_to_kickoff,
    market_params,
    NOW()
FROM
    input_data;

-- Helper query to get current odds for comparison
-- name: BulkGetCurrentOddsForComparison :many
SELECT
    co.event_id,
    co.market_type_id,
    co.outcome,
    co.odds_value,
    co.opening_value,
    co.highest_value,
    co.lowest_value,
    e.event_date
FROM
    current_odds co
    JOIN events e ON e.id = co.event_id
WHERE
    (co.event_id, co.market_type_id, co.outcome) IN (
        SELECT
            unnest(sqlc.arg(event_ids)::int[]),
            unnest(sqlc.arg(market_type_ids)::int[]),
            unnest(sqlc.arg(outcomes)::text[])
    );