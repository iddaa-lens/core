-- name: ListUnmappedLeagues :many
SELECT l.* FROM leagues l 
LEFT JOIN league_mappings lm ON l.id = lm.internal_league_id 
WHERE lm.id IS NULL;

-- name: ListUnmappedFootballLeagues :many
SELECT l.* FROM leagues l 
LEFT JOIN league_mappings lm ON l.id = lm.internal_league_id 
WHERE lm.id IS NULL AND l.sport_id = 1;

-- name: CreateLeagueMapping :one
INSERT INTO league_mappings (
    internal_league_id, 
    football_api_league_id, 
    confidence, 
    mapping_method
) VALUES (
    sqlc.arg(internal_league_id), sqlc.arg(football_api_league_id), sqlc.arg(confidence), sqlc.arg(mapping_method)
) RETURNING *;

-- name: ListLeagueMappings :many
SELECT * FROM league_mappings ORDER BY confidence DESC;

-- name: GetLeagueMapping :one
SELECT * FROM league_mappings 
WHERE internal_league_id = sqlc.arg(internal_league_id);

-- name: ListTeamsByLeague :many
SELECT t.* FROM teams t
INNER JOIN events e ON (t.id = e.home_team_id OR t.id = e.away_team_id)
WHERE e.league_id = sqlc.arg(league_id)
GROUP BY t.id, t.external_id, t.name, t.country, t.logo_url, t.is_active, t.slug, t.created_at, t.updated_at;

-- name: GetTeamMapping :one
SELECT * FROM team_mappings 
WHERE internal_team_id = sqlc.arg(internal_team_id);

-- name: CreateTeamMapping :one
INSERT INTO team_mappings (
    internal_team_id, 
    football_api_team_id, 
    confidence, 
    mapping_method
) VALUES (
    sqlc.arg(internal_team_id), sqlc.arg(football_api_team_id), sqlc.arg(confidence), sqlc.arg(mapping_method)
) RETURNING *;

-- name: ListTeamMappings :many
SELECT * FROM team_mappings ORDER BY confidence DESC;

-- name: GetLeague :one
SELECT * FROM leagues WHERE id = sqlc.arg(id);

-- name: GetLeagueByExternalID :one
SELECT * FROM leagues WHERE external_id = sqlc.arg(external_id);

-- name: ListLeagues :many
SELECT * FROM leagues ORDER BY name;

-- name: UpsertLeague :one
INSERT INTO leagues (external_id, name, country, sport_id, is_active)
VALUES (sqlc.arg(external_id), sqlc.arg(name), sqlc.arg(country), sqlc.arg(sport_id), sqlc.arg(is_active))
ON CONFLICT (external_id) DO UPDATE SET
    name = EXCLUDED.name,
    country = EXCLUDED.country,
    sport_id = EXCLUDED.sport_id,
    is_active = EXCLUDED.is_active,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpdateLeague :one
UPDATE leagues 
SET name = sqlc.arg(name), country = sqlc.arg(country), sport_id = sqlc.arg(sport_id), is_active = sqlc.arg(is_active), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteLeague :exec
DELETE FROM leagues WHERE id = sqlc.arg(id);