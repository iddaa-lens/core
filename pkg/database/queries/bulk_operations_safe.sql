-- name: BulkUpsertCurrentOddsSafe :exec
-- This version ensures array ordering is preserved by using ROW_NUMBER()
WITH input_data AS (
    SELECT 
        event_id,
        market_type_id,
        outcome,
        odds_value,
        market_params,
        ROW_NUMBER() OVER () as row_num
    FROM (
        SELECT
            unnest(sqlc.arg(event_ids)::int[]) as event_id,
            unnest(sqlc.arg(market_type_ids)::int[]) as market_type_id,
            unnest(sqlc.arg(outcomes)::text[]) as outcome,
            unnest(sqlc.arg(odds_values)::float8[]) as odds_value,
            unnest(sqlc.arg(market_params)::jsonb[]) as market_params
    ) t
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
        created_at,
        updated_at
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
    NOW(),
    NOW()
FROM
    input_data
ORDER BY row_num  -- Ensure insertion order matches array order
ON CONFLICT (event_id, market_type_id, outcome) DO
UPDATE
SET
    odds_value = EXCLUDED.odds_value,
    highest_value = GREATEST(current_odds.highest_value, EXCLUDED.odds_value),
    lowest_value = LEAST(current_odds.lowest_value, EXCLUDED.odds_value),
    updated_at = NOW()
WHERE
    current_odds.odds_value IS DISTINCT FROM EXCLUDED.odds_value;

-- name: BulkInsertOddsHistorySafe :exec
-- This version ensures array ordering is preserved and validates data
WITH input_data AS (
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
        ROW_NUMBER() OVER () as row_num
    FROM (
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
    ) t
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
    input_data
WHERE
    -- Validate data integrity
    odds_value > 0
    AND previous_value > 0
    AND NOT (odds_value = 'NaN'::float8 OR previous_value = 'NaN'::float8)
    AND NOT (odds_value = 'Infinity'::float8 OR previous_value = 'Infinity'::float8)
ORDER BY row_num;  -- Ensure insertion order matches array order

-- name: BulkUpsertCurrentOddsWithOrdinality :exec
-- Alternative approach using WITH ORDINALITY for PostgreSQL 9.4+
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
        created_at,
        updated_at
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
    NOW(),
    NOW()
FROM (
    SELECT 
        e.value as event_id,
        m.value as market_type_id,
        o.value as outcome,
        v.value as odds_value,
        p.value as market_params
    FROM 
        unnest(sqlc.arg(event_ids)::int[]) WITH ORDINALITY as e(value, ord)
        JOIN unnest(sqlc.arg(market_type_ids)::int[]) WITH ORDINALITY as m(value, ord) ON e.ord = m.ord
        JOIN unnest(sqlc.arg(outcomes)::text[]) WITH ORDINALITY as o(value, ord) ON e.ord = o.ord
        JOIN unnest(sqlc.arg(odds_values)::float8[]) WITH ORDINALITY as v(value, ord) ON e.ord = v.ord
        JOIN unnest(sqlc.arg(market_params)::jsonb[]) WITH ORDINALITY as p(value, ord) ON e.ord = p.ord
) AS ordered_data
ON CONFLICT (event_id, market_type_id, outcome) DO
UPDATE
SET
    odds_value = EXCLUDED.odds_value,
    highest_value = GREATEST(current_odds.highest_value, EXCLUDED.odds_value),
    lowest_value = LEAST(current_odds.lowest_value, EXCLUDED.odds_value),
    updated_at = NOW()
WHERE
    current_odds.odds_value IS DISTINCT FROM EXCLUDED.odds_value;