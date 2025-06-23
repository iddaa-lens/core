-- name: GetOutcomeDistribution :one
SELECT
  *
FROM
  outcome_distributions
WHERE
  event_id = sqlc.arg(event_id)
  AND market_id = sqlc.arg(market_id)
  AND outcome = sqlc.arg(outcome);

-- name: UpsertOutcomeDistribution :one
INSERT INTO
  outcome_distributions (
    event_id,
    market_id,
    outcome,
    bet_percentage,
    implied_probability
  )
VALUES
  (
    sqlc.arg(event_id),
    sqlc.arg(market_id),
    sqlc.arg(outcome),
    sqlc.arg(bet_percentage),
    sqlc.arg(implied_probability)::float8
  ) ON CONFLICT (event_id, market_id, outcome) DO
UPDATE
SET
  bet_percentage = EXCLUDED.bet_percentage,
  implied_probability = EXCLUDED.implied_probability,
  last_updated = CURRENT_TIMESTAMP RETURNING *;

-- name: CreateDistributionHistory :one
INSERT INTO
  outcome_distribution_history (
    event_id,
    market_id,
    outcome,
    bet_percentage,
    previous_percentage
  )
VALUES
  (
    sqlc.arg(event_id),
    sqlc.arg(market_id),
    sqlc.arg(outcome),
    sqlc.arg(bet_percentage),
    sqlc.arg(previous_percentage)
  ) RETURNING *;

-- name: GetCurrentOddsForOutcome :many
SELECT
  co.*
FROM
  current_odds co
WHERE
  co.event_id = sqlc.arg(event_id)
  AND co.outcome = sqlc.arg(outcome);

-- name: RefreshContrarianBets :exec
REFRESH MATERIALIZED VIEW contrarian_bets;

-- name: RefreshBigMovers :exec
REFRESH MATERIALIZED VIEW big_movers;

-- name: RefreshSharpMoneyMoves :exec
REFRESH MATERIALIZED VIEW sharp_money_moves;

-- name: RefreshLiveOpportunities :exec
REFRESH MATERIALIZED VIEW live_opportunities;

-- name: RefreshValueSpots :exec
REFRESH MATERIALIZED VIEW value_spots;

-- name: RefreshHighVolumeEvents :exec
REFRESH MATERIALIZED VIEW high_volume_events;

-- name: GetLatestOutcomeDistribution :one
SELECT
  *
FROM
  outcome_distributions
WHERE
  event_id = sqlc.arg(event_id)
  AND market_id = sqlc.arg(market_id)
  AND outcome = sqlc.arg(outcome)
ORDER BY
  last_updated DESC
LIMIT
  1;

-- name: GetEventsByExternalIDs :many
-- Bulk fetch events by external IDs
SELECT id, external_id
FROM events
WHERE external_id = ANY(sqlc.arg(external_ids)::text[]);

-- name: GetAllDistributionsForEvents :many
-- Bulk fetch all distributions for multiple events
SELECT 
  od.*,
  e.external_id as event_external_id
FROM outcome_distributions od
JOIN events e ON e.id = od.event_id
WHERE e.external_id = ANY(sqlc.arg(external_ids)::text[]);

-- name: BulkUpsertDistributions :execrows
-- Bulk upsert distributions with database-side calculations
WITH input_data AS (
  SELECT 
    unnest(sqlc.arg(external_ids)::text[]) as external_id,
    unnest(sqlc.arg(market_ids)::int4[]) as market_id,
    unnest(sqlc.arg(outcomes)::text[]) as outcome,
    unnest(sqlc.arg(bet_percentages)::float4[]) as bet_percentage,
    unnest(sqlc.arg(implied_probabilities)::float8[]) as implied_probability
)
INSERT INTO outcome_distributions (
  event_id,
  market_id,
  outcome,
  bet_percentage,
  implied_probability
)
SELECT 
  e.id,
  i.market_id,
  i.outcome,
  i.bet_percentage,
  i.implied_probability
FROM input_data i
JOIN events e ON e.external_id = i.external_id
ON CONFLICT (event_id, market_id, outcome) DO UPDATE
SET 
  bet_percentage = EXCLUDED.bet_percentage,
  implied_probability = EXCLUDED.implied_probability,
  last_updated = CURRENT_TIMESTAMP;

-- name: BulkInsertDistributionHistory :execrows
-- Bulk insert distribution history for changed values
WITH input_data AS (
  SELECT 
    unnest(sqlc.arg(external_ids)::text[]) as external_id,
    unnest(sqlc.arg(market_ids)::int4[]) as market_id,
    unnest(sqlc.arg(outcomes)::text[]) as outcome,
    unnest(sqlc.arg(bet_percentages)::float4[]) as bet_percentage,
    unnest(sqlc.arg(previous_percentages)::float4[]) as previous_percentage
)
INSERT INTO outcome_distribution_history (
  event_id,
  market_id,
  outcome,
  bet_percentage,
  previous_percentage
)
SELECT
  e.id,
  i.market_id,
  i.outcome,
  i.bet_percentage,
  i.previous_percentage
FROM input_data i
JOIN events e ON e.external_id = i.external_id;

-- name: GetCurrentOddsForEvents :many
-- Bulk fetch current odds for implied probability calculation
SELECT 
  co.event_id,
  co.outcome,
  co.odds_value,
  e.external_id
FROM current_odds co
JOIN events e ON e.id = co.event_id
WHERE e.external_id = ANY(sqlc.arg(external_ids)::text[])
  AND co.market_type_id = 1; -- Match Result market for simplicity