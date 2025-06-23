-- name: BulkCreateLeagueMappings :exec
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
    match_score,
    created_at,
    updated_at
)
SELECT
    unnest(sqlc.arg(internal_league_ids)::int[]),
    unnest(sqlc.arg(football_api_league_ids)::int[]),
    unnest(sqlc.arg(confidences)::float4[]),
    unnest(sqlc.arg(mapping_methods)::text[]),
    unnest(sqlc.arg(translated_league_names)::text[]),
    unnest(sqlc.arg(translated_countries)::text[]),
    unnest(sqlc.arg(original_league_names)::text[]),
    unnest(sqlc.arg(original_countries)::text[]),
    unnest(sqlc.arg(match_factors)::jsonb[]),
    unnest(sqlc.arg(needs_review)::boolean[]),
    unnest(sqlc.arg(ai_translation_used)::boolean[]),
    unnest(sqlc.arg(normalization_applied)::boolean[]),
    unnest(sqlc.arg(match_scores)::float4[]),
    NOW(),
    NOW()
ON CONFLICT (internal_league_id) DO UPDATE SET
    football_api_league_id = EXCLUDED.football_api_league_id,
    confidence = EXCLUDED.confidence,
    mapping_method = EXCLUDED.mapping_method,
    translated_league_name = EXCLUDED.translated_league_name,
    translated_country = EXCLUDED.translated_country,
    match_factors = EXCLUDED.match_factors,
    needs_review = EXCLUDED.needs_review,
    match_score = EXCLUDED.match_score,
    updated_at = NOW();