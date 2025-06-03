-- name: GetCurrentOdds :many
SELECT co.*, mt.name as market_name, mt.code as market_code
FROM current_odds co
JOIN market_types mt ON co.market_type_id = mt.id
WHERE co.event_id = sqlc.arg(event_id);

-- name: GetCurrentOddsByMarket :many
SELECT co.*, mt.name as market_name, mt.code as market_code
FROM current_odds co
JOIN market_types mt ON co.market_type_id = mt.id
WHERE co.event_id = sqlc.arg(event_id) 
AND co.market_type_id = sqlc.arg(market_type_id);

-- name: UpsertCurrentOdds :one
INSERT INTO current_odds (
    event_id, 
    market_type_id, 
    outcome, 
    odds_value, 
    opening_value, 
    highest_value, 
    lowest_value,
    winning_odds
) VALUES (
    sqlc.arg(event_id), 
    sqlc.arg(market_type_id), 
    sqlc.arg(outcome), 
    sqlc.arg(odds_value), 
    sqlc.arg(opening_value), 
    sqlc.arg(highest_value), 
    sqlc.arg(lowest_value),
    sqlc.arg(winning_odds)
)
ON CONFLICT (event_id, market_type_id, outcome) DO UPDATE SET
    odds_value = EXCLUDED.odds_value,
    winning_odds = EXCLUDED.winning_odds,
    highest_value = GREATEST(current_odds.highest_value, EXCLUDED.odds_value),
    lowest_value = LEAST(current_odds.lowest_value, EXCLUDED.odds_value),
    last_updated = CURRENT_TIMESTAMP
RETURNING *;

-- name: CreateOddsHistory :one
INSERT INTO odds_history (
    event_id, 
    market_type_id, 
    outcome, 
    odds_value, 
    previous_value,
    winning_odds
) VALUES (
    sqlc.arg(event_id), 
    sqlc.arg(market_type_id), 
    sqlc.arg(outcome), 
    sqlc.arg(odds_value), 
    sqlc.arg(previous_value),
    sqlc.arg(winning_odds)
)
RETURNING *;

-- name: GetOddsMovements :many
SELECT oh.*, mt.name as market_name, mt.code as market_code
FROM odds_history oh
JOIN market_types mt ON oh.market_type_id = mt.id
WHERE oh.event_id = sqlc.arg(event_id)
ORDER BY oh.recorded_at DESC
LIMIT sqlc.arg(limit_count);

-- name: GetBigMovers :many
SELECT 
    oh.*,
    e.slug as event_slug,
    mt.code as market_code
FROM odds_history oh
JOIN events e ON oh.event_id = e.id
JOIN market_types mt ON oh.market_type_id = mt.id
WHERE ABS(oh.change_percentage) > sqlc.arg(min_change_pct)
AND oh.recorded_at > sqlc.arg(since_time)
ORDER BY ABS(oh.change_percentage) DESC
LIMIT sqlc.arg(limit_count);