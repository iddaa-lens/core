-- name: GetMarketType :one
SELECT * FROM market_types
WHERE code = sqlc.arg(code) LIMIT 1;

-- name: GetMarketTypeByID :one
SELECT * FROM market_types
WHERE id = sqlc.arg(id) LIMIT 1;

-- name: UpsertMarketType :one
INSERT INTO market_types (code, name, description, iddaa_market_id, is_live, market_type, 
                         min_market_default_value, max_market_limit_value, priority, sport_type,
                         market_sub_type, min_default_value, max_limit_value, is_active)
VALUES (sqlc.arg(code), sqlc.arg(name), sqlc.arg(description), sqlc.arg(iddaa_market_id),
        sqlc.arg(is_live), sqlc.arg(market_type), sqlc.arg(min_market_default_value),
        sqlc.arg(max_market_limit_value), sqlc.arg(priority), sqlc.arg(sport_type),
        sqlc.arg(market_sub_type), sqlc.arg(min_default_value), sqlc.arg(max_limit_value),
        sqlc.arg(is_active))
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    iddaa_market_id = EXCLUDED.iddaa_market_id,
    is_live = EXCLUDED.is_live,
    market_type = EXCLUDED.market_type,
    min_market_default_value = EXCLUDED.min_market_default_value,
    max_market_limit_value = EXCLUDED.max_market_limit_value,
    priority = EXCLUDED.priority,
    sport_type = EXCLUDED.sport_type,
    market_sub_type = EXCLUDED.market_sub_type,
    min_default_value = EXCLUDED.min_default_value,
    max_limit_value = EXCLUDED.max_limit_value,
    is_active = EXCLUDED.is_active,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpsertMarketTypeByExternalID :one
INSERT INTO market_types (code, name, description)
VALUES (sqlc.arg(external_id)::text, sqlc.arg(name), sqlc.arg(description))
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: ListMarketTypes :many
SELECT * FROM market_types
ORDER BY code;