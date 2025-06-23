-- name: GetMarketType :one
SELECT
    *
FROM
    market_types
WHERE
    code = sqlc.arg(code)
LIMIT
    1;

-- name: GetMarketTypeByID :one
SELECT
    *
FROM
    market_types
WHERE
    id = sqlc.arg(id)
LIMIT
    1;

-- name: UpsertMarketType :one
INSERT INTO
    market_types (
        code,
        name,
        slug,
        description,
        iddaa_market_id,
        is_live,
        market_type,
        min_market_default_value,
        max_market_limit_value,
        priority,
        sport_type,
        market_sub_type,
        min_default_value,
        max_limit_value,
        is_active
    )
VALUES
    (
        sqlc.arg(code),
        sqlc.arg(name),
        sqlc.arg(slug),
        sqlc.arg(description),
        sqlc.arg(iddaa_market_id),
        sqlc.arg(is_live),
        sqlc.arg(market_type),
        sqlc.arg(min_market_default_value),
        sqlc.arg(max_market_limit_value),
        sqlc.arg(priority),
        sqlc.arg(sport_type),
        sqlc.arg(market_sub_type),
        sqlc.arg(min_default_value),
        sqlc.arg(max_limit_value),
        sqlc.arg(is_active)
    ) ON CONFLICT (code) DO
UPDATE
SET
    name = EXCLUDED.name,
    slug = EXCLUDED.slug,
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
    updated_at = CURRENT_TIMESTAMP RETURNING *;

-- name: UpsertMarketTypeByExternalID :one
INSERT INTO
    market_types (code, name, description)
VALUES
    (
        sqlc.arg(external_id)::text,
        sqlc.arg(name),
        sqlc.arg(description)
    ) ON CONFLICT (code) DO
UPDATE
SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_at = CURRENT_TIMESTAMP RETURNING *;

-- name: ListMarketTypes :many
SELECT
    *
FROM
    market_types
ORDER BY
    code;

-- name: BulkUpsertMarketTypes :exec
WITH input_data AS (
    SELECT
        unnest(sqlc.arg(codes)::text[]) as code,
        unnest(sqlc.arg(names)::text[]) as name,
        unnest(sqlc.arg(slugs)::text[]) as slug,
        unnest(sqlc.arg(descriptions)::text[]) as description,
        unnest(sqlc.arg(iddaa_market_ids)::int[]) as iddaa_market_id,
        unnest(sqlc.arg(is_lives)::boolean[]) as is_live,
        unnest(sqlc.arg(market_types)::int[]) as market_type,
        unnest(sqlc.arg(min_market_default_values)::int[]) as min_market_default_value,
        unnest(sqlc.arg(max_market_limit_values)::int[]) as max_market_limit_value,
        unnest(sqlc.arg(priorities)::int[]) as priority,
        unnest(sqlc.arg(sport_types)::int[]) as sport_type,
        unnest(sqlc.arg(market_sub_types)::int[]) as market_sub_type,
        unnest(sqlc.arg(min_default_values)::int[]) as min_default_value,
        unnest(sqlc.arg(max_limit_values)::int[]) as max_limit_value,
        unnest(sqlc.arg(is_actives)::boolean[]) as is_active
)
INSERT INTO
    market_types (
        code,
        name,
        slug,
        description,
        iddaa_market_id,
        is_live,
        market_type,
        min_market_default_value,
        max_market_limit_value,
        priority,
        sport_type,
        market_sub_type,
        min_default_value,
        max_limit_value,
        is_active,
        created_at,
        updated_at
    )
SELECT
    code,
    name,
    slug,
    NULLIF(description, ''),
    iddaa_market_id,
    is_live,
    market_type,
    min_market_default_value,
    max_market_limit_value,
    priority,
    sport_type,
    market_sub_type,
    min_default_value,
    max_limit_value,
    is_active,
    NOW(),
    NOW()
FROM input_data
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    slug = EXCLUDED.slug,
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
    updated_at = CURRENT_TIMESTAMP;