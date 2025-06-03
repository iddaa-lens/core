-- name: GetCompetition :one
SELECT c.*, s.name as sport_name, s.code as sport_code
FROM competitions c
JOIN sports s ON c.sport_id = s.id
WHERE c.id = sqlc.arg(id);

-- name: GetCompetitionByIddaaID :one
SELECT c.*, s.name as sport_name, s.code as sport_code
FROM competitions c
JOIN sports s ON c.sport_id = s.id
WHERE c.iddaa_id = sqlc.arg(iddaa_id);

-- name: CreateCompetition :one
INSERT INTO competitions (iddaa_id, external_ref, country_code, parent_id, sport_id, short_name, full_name, icon_url)
VALUES (sqlc.arg(iddaa_id), sqlc.arg(external_ref), sqlc.arg(country_code), sqlc.arg(parent_id), sqlc.arg(sport_id), sqlc.arg(short_name), sqlc.arg(full_name), sqlc.arg(icon_url))
RETURNING *;

-- name: UpdateCompetition :one
UPDATE competitions 
SET external_ref = sqlc.arg(external_ref), country_code = sqlc.arg(country_code), parent_id = sqlc.arg(parent_id), sport_id = sqlc.arg(sport_id), 
    short_name = sqlc.arg(short_name), full_name = sqlc.arg(full_name), icon_url = sqlc.arg(icon_url), updated_at = CURRENT_TIMESTAMP
WHERE iddaa_id = sqlc.arg(iddaa_id)
RETURNING *;

-- name: UpsertCompetition :one
INSERT INTO competitions (iddaa_id, external_ref, country_code, parent_id, sport_id, short_name, full_name, icon_url)
VALUES (sqlc.arg(iddaa_id), sqlc.arg(external_ref), sqlc.arg(country_code), sqlc.arg(parent_id), sqlc.arg(sport_id), sqlc.arg(short_name), sqlc.arg(full_name), sqlc.arg(icon_url))
ON CONFLICT (iddaa_id) DO UPDATE SET
    external_ref = EXCLUDED.external_ref,
    country_code = EXCLUDED.country_code,
    parent_id = EXCLUDED.parent_id,
    sport_id = EXCLUDED.sport_id,
    short_name = EXCLUDED.short_name,
    full_name = EXCLUDED.full_name,
    icon_url = EXCLUDED.icon_url,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: ListCompetitionsBySport :many
SELECT c.*, s.name as sport_name, s.code as sport_code
FROM competitions c
JOIN sports s ON c.sport_id = s.id
WHERE c.sport_id = sqlc.arg(sport_id)
ORDER BY c.full_name;

-- name: ListCompetitionsByCountry :many
SELECT c.*, s.name as sport_name, s.code as sport_code
FROM competitions c
JOIN sports s ON c.sport_id = s.id
WHERE c.country_code = sqlc.arg(country_code)
ORDER BY c.full_name;