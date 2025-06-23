-- Remove market parameters from odds tables

-- Drop indexes
DROP INDEX IF EXISTS idx_odds_history_market_params;
DROP INDEX IF EXISTS idx_current_odds_market_params;

-- Remove columns
ALTER TABLE odds_history DROP COLUMN IF EXISTS market_params;
ALTER TABLE current_odds DROP COLUMN IF EXISTS market_params;