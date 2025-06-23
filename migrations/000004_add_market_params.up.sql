-- Add market parameters to odds tables for dynamic market names

-- Add to current_odds table
ALTER TABLE current_odds 
ADD COLUMN IF NOT EXISTS market_params JSONB DEFAULT '{}';

-- Add to odds_history table  
ALTER TABLE odds_history
ADD COLUMN IF NOT EXISTS market_params JSONB DEFAULT '{}';

-- Example data structure in market_params:
-- For "Ev Sahibi Toplam Korner Altı/Üstü {0}":
-- {"line": 5.5}
--
-- For "{0} - {1} dk. Kart Sayısı Altı/Üstü {2}":
-- {"start_minute": 15, "end_minute": 30, "card_line": 2.5}
--
-- For generic markets:
-- {"params": ["value1", "value2"]}

-- Add index for common queries
CREATE INDEX IF NOT EXISTS idx_current_odds_market_params 
ON current_odds USING GIN (market_params);

CREATE INDEX IF NOT EXISTS idx_odds_history_market_params 
ON odds_history USING GIN (market_params);