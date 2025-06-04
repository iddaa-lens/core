-- name: GetTeam :one
SELECT * FROM teams WHERE id = sqlc.arg(id);

-- name: GetTeamByExternalID :one
SELECT * FROM teams WHERE external_id = sqlc.arg(external_id);

-- name: CreateTeam :one
INSERT INTO teams (external_id, name, country, logo_url)
VALUES (sqlc.arg(external_id), sqlc.arg(name), sqlc.arg(country), sqlc.arg(logo_url))
RETURNING *;

-- name: UpdateTeam :one
UPDATE teams 
SET name = sqlc.arg(name), country = sqlc.arg(country), logo_url = sqlc.arg(logo_url), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpsertTeam :one
INSERT INTO teams (external_id, name, country, logo_url)
VALUES (sqlc.arg(external_id), sqlc.arg(name), sqlc.arg(country), sqlc.arg(logo_url))
ON CONFLICT (external_id) DO UPDATE SET
    name = EXCLUDED.name,
    country = EXCLUDED.country,
    logo_url = EXCLUDED.logo_url,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: SearchTeams :many
SELECT * FROM teams 
WHERE name ILIKE '%' || sqlc.arg(search_term) || '%' 
ORDER BY name
LIMIT sqlc.arg(limit_count);