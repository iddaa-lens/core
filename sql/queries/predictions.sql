-- name: GetPredictionsByEvent :many
SELECT p.*, mt.name as market_name, mt.code as market_code
FROM predictions p
JOIN market_types mt ON p.market_type_id = mt.id
WHERE p.event_id = sqlc.arg(event_id)
ORDER BY p.confidence_score DESC;

-- name: GetLatestPredictions :many
SELECT p.*, mt.name as market_name, mt.code as market_code
FROM predictions p
JOIN market_types mt ON p.market_type_id = mt.id
WHERE p.event_id = sqlc.arg(event_id)
AND p.created_at = (
    SELECT MAX(created_at) 
    FROM predictions p2 
    WHERE p2.event_id = p.event_id 
    AND p2.market_type_id = p.market_type_id
);

-- name: CreatePrediction :one
INSERT INTO predictions (event_id, market_type_id, predicted_outcome, confidence_score, model_version, features_used)
VALUES (sqlc.arg(event_id), sqlc.arg(market_type_id), sqlc.arg(predicted_outcome), sqlc.arg(confidence_score), sqlc.arg(model_version), sqlc.arg(features_used))
RETURNING *;

-- name: GetPredictionAccuracy :many
SELECT 
    p.market_type_id,
    mt.name as market_name,
    p.model_version,
    COUNT(*) as total_predictions,
    SUM(CASE 
        WHEN e.status = 'finished' AND (
            (p.predicted_outcome = '1' AND e.home_score > e.away_score) OR
            (p.predicted_outcome = 'X' AND e.home_score = e.away_score) OR
            (p.predicted_outcome = '2' AND e.home_score < e.away_score)
        ) THEN 1 
        ELSE 0 
    END) as correct_predictions,
    AVG(p.confidence_score) as avg_confidence
FROM predictions p
JOIN market_types mt ON p.market_type_id = mt.id
JOIN events e ON p.event_id = e.id
WHERE e.status = 'finished'
AND p.created_at >= sqlc.arg(since_date)
GROUP BY p.market_type_id, mt.name, p.model_version;