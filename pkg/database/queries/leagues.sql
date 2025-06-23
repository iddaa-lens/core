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

-- name: CreateEnhancedLeagueMapping :one
INSERT INTO league_mappings (
    internal_league_id,
    football_api_league_id,
    confidence,
    mapping_method,
    translated_league_name,
    translated_country,
    original_league_name,
    original_country,
    match_factors,
    needs_review,
    ai_translation_used,
    normalization_applied,
    match_score
) VALUES (
    sqlc.arg(internal_league_id),
    sqlc.arg(football_api_league_id),
    sqlc.arg(confidence),
    sqlc.arg(mapping_method),
    sqlc.arg(translated_league_name),
    sqlc.arg(translated_country),
    sqlc.arg(original_league_name),
    sqlc.arg(original_country),
    sqlc.arg(match_factors),
    sqlc.arg(needs_review),
    sqlc.arg(ai_translation_used),
    sqlc.arg(normalization_applied),
    sqlc.arg(match_score)
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

-- name: CreateEnhancedTeamMapping :one
INSERT INTO team_mappings (
    internal_team_id,
    football_api_team_id,
    confidence,
    mapping_method,
    translated_team_name,
    translated_country,
    translated_league,
    original_team_name,
    original_country,
    original_league,
    match_factors,
    needs_review,
    ai_translation_used,
    normalization_applied,
    match_score
) VALUES (
    sqlc.arg(internal_team_id),
    sqlc.arg(football_api_team_id),
    sqlc.arg(confidence),
    sqlc.arg(mapping_method),
    sqlc.arg(translated_team_name),
    sqlc.arg(translated_country),
    sqlc.arg(translated_league),
    sqlc.arg(original_team_name),
    sqlc.arg(original_country),
    sqlc.arg(original_league),
    sqlc.arg(match_factors),
    sqlc.arg(needs_review),
    sqlc.arg(ai_translation_used),
    sqlc.arg(normalization_applied),
    sqlc.arg(match_score)
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

-- name: BulkUpsertLeagues :execrows
INSERT INTO leagues (external_id, name, country, sport_id, is_active)
VALUES (
    unnest(sqlc.arg(external_ids)::text[]),
    unnest(sqlc.arg(names)::text[]),
    unnest(sqlc.arg(countries)::text[]),
    unnest(sqlc.arg(sport_ids)::int[]),
    unnest(sqlc.arg(is_actives)::boolean[])
)
ON CONFLICT (external_id) DO UPDATE SET
    name = EXCLUDED.name,
    country = EXCLUDED.country,
    sport_id = EXCLUDED.sport_id,
    is_active = EXCLUDED.is_active,
    updated_at = CURRENT_TIMESTAMP;

-- name: UpdateLeague :one
UPDATE leagues 
SET name = sqlc.arg(name), country = sqlc.arg(country), sport_id = sqlc.arg(sport_id), is_active = sqlc.arg(is_active), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteLeague :exec
DELETE FROM leagues WHERE id = sqlc.arg(id);

-- name: EnrichLeagueWithAPIFootball :one
UPDATE leagues SET
    api_football_id = sqlc.arg(api_football_id),
    league_type = sqlc.arg(league_type),
    logo_url = sqlc.arg(logo_url),
    country_code = sqlc.arg(country_code),
    country_flag_url = sqlc.arg(country_flag_url),
    has_standings = sqlc.arg(has_standings),
    has_fixtures = sqlc.arg(has_fixtures),
    has_players = sqlc.arg(has_players),
    has_top_scorers = sqlc.arg(has_top_scorers),
    has_injuries = sqlc.arg(has_injuries),
    has_predictions = sqlc.arg(has_predictions),
    has_odds = sqlc.arg(has_odds),
    current_season_year = sqlc.arg(current_season_year),
    current_season_start = sqlc.arg(current_season_start),
    current_season_end = sqlc.arg(current_season_end),
    api_enrichment_data = sqlc.arg(api_enrichment_data),
    last_api_update = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetLeaguesByAPIFootballID :many
SELECT * FROM leagues WHERE api_football_id = sqlc.arg(api_football_id);

-- name: ListLeaguesForAPIEnrichment :many
SELECT l.* FROM leagues l
INNER JOIN league_mappings lm ON l.id = lm.internal_league_id
WHERE l.last_api_update IS NULL 
   OR l.last_api_update < NOW() - INTERVAL '7 days'
ORDER BY l.updated_at ASC
LIMIT sqlc.arg(limit_count);

-- name: UpdateLeagueApiFootballID :exec
UPDATE leagues 
SET api_football_id = sqlc.arg(api_football_id), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id);

-- name: UpsertLeagueMapping :one
INSERT INTO league_mappings (
    internal_league_id, 
    football_api_league_id, 
    confidence, 
    mapping_method
) VALUES (
    sqlc.arg(internal_league_id), 
    sqlc.arg(football_api_league_id), 
    sqlc.arg(confidence), 
    sqlc.arg(mapping_method)
) 
ON CONFLICT (internal_league_id) 
DO UPDATE SET
    football_api_league_id = EXCLUDED.football_api_league_id,
    confidence = EXCLUDED.confidence,
    mapping_method = EXCLUDED.mapping_method,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;