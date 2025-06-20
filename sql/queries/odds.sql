-- name: GetCurrentOdds :many
SELECT co.*, mt.name as market_name, mt.code as market_code
FROM current_odds co
JOIN market_types mt ON co.market_type_id = mt.id
WHERE co.event_id = sqlc.arg('event_id')::int;

-- name: GetCurrentOddsByMarket :many
SELECT co.*, mt.name as market_name, mt.code as market_code
FROM current_odds co
JOIN market_types mt ON co.market_type_id = mt.id
WHERE co.event_id = sqlc.arg('event_id')::int 
AND co.market_type_id = sqlc.arg('market_type_id')::int;

-- name: GetCurrentOddsByOutcome :one
SELECT co.*, mt.name as market_name, mt.code as market_code
FROM current_odds co
JOIN market_types mt ON co.market_type_id = mt.id
WHERE co.event_id = sqlc.arg('event_id')::int 
AND co.market_type_id = sqlc.arg('market_type_id')::int
AND co.outcome = sqlc.arg('outcome')::text;

-- name: UpsertCurrentOdds :one
INSERT INTO current_odds (
    event_id, 
    market_type_id, 
    outcome, 
    odds_value, 
    opening_value, 
    highest_value, 
    lowest_value,
    winning_odds,
    total_movement,
    movement_percentage
) VALUES (
    sqlc.arg('event_id')::int, 
    sqlc.arg('market_type_id')::int, 
    sqlc.arg('outcome')::text, 
    sqlc.arg('odds_value')::decimal, 
    sqlc.arg('opening_value')::decimal, 
    sqlc.arg('highest_value')::decimal, 
    sqlc.arg('lowest_value')::decimal,
    sqlc.arg('winning_odds')::decimal,
    0, -- First time, no movement
    0  -- First time, no movement percentage
)
ON CONFLICT (event_id, market_type_id, outcome) DO UPDATE SET
    odds_value = EXCLUDED.odds_value,
    winning_odds = EXCLUDED.winning_odds,
    highest_value = GREATEST(current_odds.highest_value, EXCLUDED.odds_value),
    lowest_value = LEAST(current_odds.lowest_value, EXCLUDED.odds_value),
    -- Calculate movement: new_odds - opening_odds
    total_movement = EXCLUDED.odds_value - current_odds.opening_value,
    -- Calculate movement percentage: ((new_odds - opening_odds) / opening_odds) * 100
    movement_percentage = CASE 
        WHEN current_odds.opening_value > 0 THEN 
            ROUND(((EXCLUDED.odds_value - current_odds.opening_value) / current_odds.opening_value * 100)::numeric, 2)
        ELSE 0 
    END,
    last_updated = CURRENT_TIMESTAMP
RETURNING *;

-- name: CreateOddsHistory :one
INSERT INTO odds_history (
    event_id, 
    market_type_id, 
    outcome, 
    odds_value, 
    previous_value,
    winning_odds,
    change_amount,
    change_percentage,
    multiplier
) VALUES (
    sqlc.arg('event_id')::int, 
    sqlc.arg('market_type_id')::int, 
    sqlc.arg('outcome')::text, 
    sqlc.arg('odds_value')::decimal, 
    sqlc.arg('previous_value')::decimal,
    sqlc.arg('winning_odds')::decimal,
    -- Calculate change amount: new_odds - previous_odds
    sqlc.arg('odds_value')::decimal - sqlc.arg('previous_value')::decimal,
    -- Calculate change percentage: ((new_odds - previous_odds) / previous_odds) * 100
    CASE 
        WHEN sqlc.arg('previous_value')::decimal > 0 THEN 
            ROUND(((sqlc.arg('odds_value')::decimal - sqlc.arg('previous_value')::decimal) / sqlc.arg('previous_value')::decimal * 100), 2)
        ELSE 0 
    END,
    -- Calculate multiplier: new_odds / previous_odds
    CASE 
        WHEN sqlc.arg('previous_value')::decimal > 0 THEN 
            ROUND((sqlc.arg('odds_value')::decimal / sqlc.arg('previous_value')::decimal), 3)
        ELSE 1 
    END
)
RETURNING *;

-- name: GetOddsMovements :many
SELECT oh.*, mt.name as market_name, mt.code as market_code
FROM odds_history oh
JOIN market_types mt ON oh.market_type_id = mt.id
WHERE oh.event_id = sqlc.arg('event_id')::int
ORDER BY oh.recorded_at DESC
LIMIT sqlc.arg('limit_count')::int;

-- name: GetOddsHistoryByID :one
SELECT * FROM odds_history WHERE id = sqlc.arg('id')::bigint;

-- name: GetBigMovers :many
SELECT 
    oh.*,
    e.slug as event_slug,
    mt.code as market_code
FROM odds_history oh
JOIN events e ON oh.event_id = e.id
JOIN market_types mt ON oh.market_type_id = mt.id
WHERE ABS(oh.change_percentage) > sqlc.arg('min_change_pct')::decimal
AND oh.recorded_at > sqlc.arg('since_time')::timestamp
ORDER BY ABS(oh.change_percentage) DESC
LIMIT sqlc.arg('limit_count')::int;

-- name: GetRecentOddsHistory :many
SELECT oh.*, e.event_date, e.is_live, mt.name as market_name, mt.code as market_code
FROM odds_history oh
JOIN events e ON oh.event_id = e.id
JOIN market_types mt ON oh.market_type_id = mt.id
WHERE oh.recorded_at >= sqlc.arg('since_time')::timestamp
AND e.event_date > NOW()
AND ABS(oh.change_percentage) >= sqlc.arg('min_change_pct')::decimal
ORDER BY oh.recorded_at DESC
LIMIT sqlc.arg('limit_count')::int;