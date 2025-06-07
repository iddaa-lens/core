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

-- name: EnrichTeamWithAPIFootball :one
UPDATE teams 
SET 
    api_football_id = sqlc.arg(api_football_id),
    team_code = sqlc.arg(team_code),
    founded_year = sqlc.arg(founded_year),
    is_national_team = sqlc.arg(is_national_team),
    venue_id = sqlc.arg(venue_id),
    venue_name = sqlc.arg(venue_name),
    venue_address = sqlc.arg(venue_address),
    venue_city = sqlc.arg(venue_city),
    venue_capacity = sqlc.arg(venue_capacity),
    venue_surface = sqlc.arg(venue_surface),
    venue_image_url = sqlc.arg(venue_image_url),
    api_enrichment_data = sqlc.arg(api_enrichment_data),
    last_api_update = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetTeamsNeedingEnrichment :many
SELECT * FROM teams 
WHERE api_football_id IS NULL 
   OR last_api_update IS NULL 
   OR last_api_update < (CURRENT_TIMESTAMP - INTERVAL '7 days')
ORDER BY last_api_update ASC NULLS FIRST
LIMIT sqlc.arg(limit_count);

-- name: GetTeamsByAPIFootballID :one
SELECT * FROM teams WHERE api_football_id = sqlc.arg(api_football_id);

-- name: SearchTeamsByCode :many
SELECT * FROM teams 
WHERE team_code ILIKE '%' || sqlc.arg(code_search) || '%' 
ORDER BY name
LIMIT sqlc.arg(limit_count);

-- name: GetNationalTeams :many
SELECT * FROM teams 
WHERE is_national_team = true
ORDER BY name;

-- name: GetTeamsByFoundedRange :many
SELECT * FROM teams 
WHERE founded_year >= sqlc.arg(min_year) 
  AND founded_year <= sqlc.arg(max_year)
ORDER BY founded_year DESC, name;

-- name: GetTeamsByVenueCapacity :many
SELECT * FROM teams 
WHERE venue_capacity >= sqlc.arg(min_capacity)
ORDER BY venue_capacity DESC, name
LIMIT sqlc.arg(limit_count);

-- name: SearchTeams :many
SELECT * FROM teams 
WHERE name ILIKE '%' || sqlc.arg(search_term) || '%' 
ORDER BY name
LIMIT sqlc.arg(limit_count);