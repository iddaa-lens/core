// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: bulk_operations_safe.sql

package generated

import (
	"context"
)

const bulkInsertOddsHistorySafe = `-- name: BulkInsertOddsHistorySafe :exec
WITH input_data AS (
    SELECT 
        event_id,
        market_type_id,
        outcome,
        odds_value,
        previous_value,
        change_amount,
        change_percentage,
        multiplier,
        is_reverse_movement,
        significance_level,
        minutes_to_kickoff,
        market_params,
        ROW_NUMBER() OVER () as row_num
    FROM (
        SELECT
            unnest($1::int[]) as event_id,
            unnest($2::int[]) as market_type_id,
            unnest($3::text[]) as outcome,
            unnest($4::float8[]) as odds_value,
            unnest($5::float8[]) as previous_value,
            unnest($6::float8[]) as change_amount,
            unnest($7::float8[]) as change_percentage,
            unnest($8::float8[]) as multiplier,
            unnest($9::boolean[]) as is_reverse_movement,
            unnest($10::text[]) as significance_level,
            unnest($11::int[]) as minutes_to_kickoff,
            unnest($12::jsonb[]) as market_params
    ) t
)
INSERT INTO
    odds_history (
        event_id,
        market_type_id,
        outcome,
        odds_value,
        previous_value,
        change_amount,
        change_percentage,
        multiplier,
        is_reverse_movement,
        significance_level,
        minutes_to_kickoff,
        market_params,
        recorded_at
    )
SELECT
    event_id,
    market_type_id,
    outcome,
    odds_value,
    previous_value,
    change_amount,
    change_percentage,
    multiplier,
    is_reverse_movement,
    significance_level,
    minutes_to_kickoff,
    market_params,
    NOW()
FROM
    input_data
WHERE
    -- Validate data integrity
    odds_value > 0
    AND previous_value > 0
    AND NOT (odds_value = 'NaN'::float8 OR previous_value = 'NaN'::float8)
    AND NOT (odds_value = 'Infinity'::float8 OR previous_value = 'Infinity'::float8)
ORDER BY row_num
`

type BulkInsertOddsHistorySafeParams struct {
	EventIds           []int32   `db:"event_ids" json:"event_ids"`
	MarketTypeIds      []int32   `db:"market_type_ids" json:"market_type_ids"`
	Outcomes           []string  `db:"outcomes" json:"outcomes"`
	OddsValues         []float64 `db:"odds_values" json:"odds_values"`
	PreviousValues     []float64 `db:"previous_values" json:"previous_values"`
	ChangeAmounts      []float64 `db:"change_amounts" json:"change_amounts"`
	ChangePercentages  []float64 `db:"change_percentages" json:"change_percentages"`
	Multipliers        []float64 `db:"multipliers" json:"multipliers"`
	IsReverseMovements []bool    `db:"is_reverse_movements" json:"is_reverse_movements"`
	SignificanceLevels []string  `db:"significance_levels" json:"significance_levels"`
	MinutesToKickoffs  []int32   `db:"minutes_to_kickoffs" json:"minutes_to_kickoffs"`
	MarketParams       [][]byte  `db:"market_params" json:"market_params"`
}

// This version ensures array ordering is preserved and validates data
func (q *Queries) BulkInsertOddsHistorySafe(ctx context.Context, arg BulkInsertOddsHistorySafeParams) error {
	_, err := q.db.Exec(ctx, bulkInsertOddsHistorySafe,
		arg.EventIds,
		arg.MarketTypeIds,
		arg.Outcomes,
		arg.OddsValues,
		arg.PreviousValues,
		arg.ChangeAmounts,
		arg.ChangePercentages,
		arg.Multipliers,
		arg.IsReverseMovements,
		arg.SignificanceLevels,
		arg.MinutesToKickoffs,
		arg.MarketParams,
	)
	return err
}

const bulkUpsertCurrentOddsSafe = `-- name: BulkUpsertCurrentOddsSafe :exec
WITH input_data AS (
    SELECT 
        event_id,
        market_type_id,
        outcome,
        odds_value,
        market_params,
        ROW_NUMBER() OVER () as row_num
    FROM (
        SELECT
            unnest($1::int[]) as event_id,
            unnest($2::int[]) as market_type_id,
            unnest($3::text[]) as outcome,
            unnest($4::float8[]) as odds_value,
            unnest($5::jsonb[]) as market_params
    ) t
)
INSERT INTO
    current_odds (
        event_id,
        market_type_id,
        outcome,
        odds_value,
        opening_value,
        highest_value,
        lowest_value,
        market_params,
        created_at,
        updated_at
    )
SELECT
    event_id,
    market_type_id,
    outcome,
    odds_value,
    odds_value,
    odds_value,
    odds_value,
    market_params,
    NOW(),
    NOW()
FROM
    input_data
ORDER BY row_num  -- Ensure insertion order matches array order
ON CONFLICT (event_id, market_type_id, outcome) DO
UPDATE
SET
    odds_value = EXCLUDED.odds_value,
    highest_value = GREATEST(current_odds.highest_value, EXCLUDED.odds_value),
    lowest_value = LEAST(current_odds.lowest_value, EXCLUDED.odds_value),
    updated_at = NOW()
WHERE
    current_odds.odds_value IS DISTINCT FROM EXCLUDED.odds_value
`

type BulkUpsertCurrentOddsSafeParams struct {
	EventIds      []int32   `db:"event_ids" json:"event_ids"`
	MarketTypeIds []int32   `db:"market_type_ids" json:"market_type_ids"`
	Outcomes      []string  `db:"outcomes" json:"outcomes"`
	OddsValues    []float64 `db:"odds_values" json:"odds_values"`
	MarketParams  [][]byte  `db:"market_params" json:"market_params"`
}

// This version ensures array ordering is preserved by using ROW_NUMBER()
func (q *Queries) BulkUpsertCurrentOddsSafe(ctx context.Context, arg BulkUpsertCurrentOddsSafeParams) error {
	_, err := q.db.Exec(ctx, bulkUpsertCurrentOddsSafe,
		arg.EventIds,
		arg.MarketTypeIds,
		arg.Outcomes,
		arg.OddsValues,
		arg.MarketParams,
	)
	return err
}

const bulkUpsertCurrentOddsWithOrdinality = `-- name: BulkUpsertCurrentOddsWithOrdinality :exec

INSERT INTO
    current_odds (
        event_id,
        market_type_id,
        outcome,
        odds_value,
        opening_value,
        highest_value,
        lowest_value,
        market_params,
        created_at,
        updated_at
    )
SELECT
    event_id,
    market_type_id,
    outcome,
    odds_value,
    odds_value,
    odds_value,
    odds_value,
    market_params,
    NOW(),
    NOW()
FROM (
    SELECT 
        e.value as event_id,
        m.value as market_type_id,
        o.value as outcome,
        v.value as odds_value,
        p.value as market_params
    FROM 
        unnest($1::int[]) WITH ORDINALITY as e(value, ord)
        JOIN unnest($2::int[]) WITH ORDINALITY as m(value, ord) ON e.ord = m.ord
        JOIN unnest($3::text[]) WITH ORDINALITY as o(value, ord) ON e.ord = o.ord
        JOIN unnest($4::float8[]) WITH ORDINALITY as v(value, ord) ON e.ord = v.ord
        JOIN unnest($5::jsonb[]) WITH ORDINALITY as p(value, ord) ON e.ord = p.ord
) AS ordered_data
ON CONFLICT (event_id, market_type_id, outcome) DO
UPDATE
SET
    odds_value = EXCLUDED.odds_value,
    highest_value = GREATEST(current_odds.highest_value, EXCLUDED.odds_value),
    lowest_value = LEAST(current_odds.lowest_value, EXCLUDED.odds_value),
    updated_at = NOW()
WHERE
    current_odds.odds_value IS DISTINCT FROM EXCLUDED.odds_value
`

type BulkUpsertCurrentOddsWithOrdinalityParams struct {
	EventIds      []int32   `db:"event_ids" json:"event_ids"`
	MarketTypeIds []int32   `db:"market_type_ids" json:"market_type_ids"`
	Outcomes      []string  `db:"outcomes" json:"outcomes"`
	OddsValues    []float64 `db:"odds_values" json:"odds_values"`
	MarketParams  [][]byte  `db:"market_params" json:"market_params"`
}

// Ensure insertion order matches array order
// Alternative approach using WITH ORDINALITY for PostgreSQL 9.4+
func (q *Queries) BulkUpsertCurrentOddsWithOrdinality(ctx context.Context, arg BulkUpsertCurrentOddsWithOrdinalityParams) error {
	_, err := q.db.Exec(ctx, bulkUpsertCurrentOddsWithOrdinality,
		arg.EventIds,
		arg.MarketTypeIds,
		arg.Outcomes,
		arg.OddsValues,
		arg.MarketParams,
	)
	return err
}
