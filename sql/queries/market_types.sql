-- name: GetMarketType :one
SELECT * FROM market_types
WHERE code = sqlc.arg(code) LIMIT 1;

-- name: GetMarketTypeByID :one
SELECT * FROM market_types
WHERE id = sqlc.arg(id) LIMIT 1;

-- name: UpsertMarketType :one
INSERT INTO market_types (code, name, description)
VALUES (sqlc.arg(code), sqlc.arg(name), sqlc.arg(description))
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
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