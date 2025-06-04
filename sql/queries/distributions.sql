-- name: GetOutcomeDistribution :one
SELECT * FROM outcome_distributions
WHERE event_id = sqlc.arg(event_id)
  AND market_id = sqlc.arg(market_id)
  AND outcome = sqlc.arg(outcome);

-- name: UpsertOutcomeDistribution :one
INSERT INTO outcome_distributions (
    event_id,
    market_id,
    outcome,
    bet_percentage,
    implied_probability
) VALUES (
    sqlc.arg(event_id),
    sqlc.arg(market_id),
    sqlc.arg(outcome),
    sqlc.arg(bet_percentage),
    sqlc.arg(implied_probability)
)
ON CONFLICT (event_id, market_id, outcome) DO UPDATE SET
    bet_percentage = EXCLUDED.bet_percentage,
    implied_probability = EXCLUDED.implied_probability,
    last_updated = CURRENT_TIMESTAMP
RETURNING *;

-- name: CreateDistributionHistory :one
INSERT INTO outcome_distribution_history (
    event_id,
    market_id,
    outcome,
    bet_percentage,
    previous_percentage
) VALUES (
    sqlc.arg(event_id),
    sqlc.arg(market_id),
    sqlc.arg(outcome),
    sqlc.arg(bet_percentage),
    sqlc.arg(previous_percentage)
)
RETURNING *;

-- name: GetCurrentOddsForOutcome :many
SELECT co.* 
FROM current_odds co
WHERE co.event_id = sqlc.arg(event_id)
  AND co.outcome = sqlc.arg(outcome);

-- name: RefreshContrarianBets :exec
REFRESH MATERIALIZED VIEW contrarian_bets;

-- name: GetContrarianBets :many
SELECT * FROM contrarian_bets;

-- name: GetEventBettingPatterns :many
-- Analyze all betting distributions for an event
SELECT 
    od.market_id,
    od.outcome,
    od.bet_percentage,
    co.odds_value,
    ROUND((1.0 / co.odds_value * 100)::numeric, 2) as implied_probability,
    od.bet_percentage - ROUND((1.0 / co.odds_value * 100)::numeric, 2) as bias,
    CASE 
        WHEN od.bet_percentage > 70 THEN 'HEAVY_FAVORITE'
        WHEN od.bet_percentage > 50 THEN 'FAVORITE'
        WHEN od.bet_percentage < 20 THEN 'LONGSHOT'
        ELSE 'BALANCED'
    END as betting_pattern
FROM outcome_distributions od
LEFT JOIN current_odds co ON od.event_id = co.event_id 
    AND od.outcome = co.outcome
WHERE od.event_id = sqlc.arg(event_id)
ORDER BY od.market_id, od.outcome;

-- name: GetEventDistributions :many
SELECT * FROM outcome_distributions
WHERE event_id = sqlc.arg(event_id)
ORDER BY market_id, outcome;

-- name: GetTopDistributions :many
SELECT * FROM outcome_distributions
ORDER BY bet_percentage DESC
LIMIT sqlc.arg(limit_count);

-- name: GetDistributionHistory :many
SELECT * FROM outcome_distribution_history
WHERE event_id = sqlc.arg(event_id)
  AND market_id = sqlc.arg(market_id)
  AND outcome = sqlc.arg(outcome)
ORDER BY recorded_at DESC;