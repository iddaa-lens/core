# Volume + Odds Movement Insights

## The Power of Combining Volume and Odds Data

By tracking both betting volume (what % of bets each event receives) and odds movements, we can identify:

### 1. **Hot Movers** 🔥
High volume + big odds movement = Public money causing market shifts
```sql
-- Example: Fenerbahce vs Galatasaray
-- Volume: 8.01% (rank #1)
-- Home odds: 1.20 → 3.70 (208% drift)
-- Insight: "Public hammering the away team, causing home to drift"
```

### 2. **Hidden Gems** 💎
Low volume + big odds movement = Sharp money moving quietly
```sql
-- Example: Antalyaspor vs Konyaspor
-- Volume: 0.07% (rank #91)
-- Draw odds: 3.20 → 4.50 (40% drift)
-- Insight: "Sharps quietly backing home/away, draw drifting"
```

### 3. **Stable Favorites** 🛡️
High volume + stable odds = Efficient market consensus
```sql
-- Example: Real Madrid vs Getafe
-- Volume: 4.49% (rank #2)
-- Home odds: 1.15 → 1.17 (1.7% drift)
-- Insight: "Market agrees on heavy favorite"
```

### 4. **Value Traps** ⚠️
Low volume + stable odds = Potentially overlooked value
```sql
-- Example: Segunda Division match
-- Volume: 0.13% (rank #75)
-- All odds stable
-- Insight: "Market ignoring - check for value"
```

## Key Patterns to Display

### Pattern 1: Volume Surge Detection
```javascript
// When an event suddenly gets popular
const detectVolumeSurge = (currentVolume, historicalAvg) => {
  const surgeMultiplier = currentVolume / historicalAvg;
  
  if (surgeMultiplier > 3) {
    return {
      alert: "VOLUME SURGE",
      message: `${surgeMultiplier.toFixed(1)}x normal betting volume`,
      action: "Check for team news or insider activity"
    };
  }
};
```

### Pattern 2: Sharp vs Public Money
```typescript
interface MoneyTypeIndicator {
  volumeRank: number;
  oddsMovement: number;
  diagnosis: 'SHARP' | 'PUBLIC' | 'MIXED';
  confidence: number;
}

// Low volume + big movement = likely sharp
// High volume + big movement = likely public
// High volume + no movement = balanced sharp/public
```

### Pattern 3: Pre-Match Patterns
```sql
-- Track how volume/odds evolve as match approaches
WITH time_patterns AS (
  SELECT 
    EXTRACT(HOURS FROM (event_date - NOW())) as hours_to_match,
    AVG(betting_volume_percentage) as avg_volume,
    AVG(ABS(movement_percentage)) as avg_movement
  FROM events e
  JOIN current_odds co ON co.event_id = e.id
  WHERE event_date BETWEEN NOW() AND NOW() + INTERVAL '48 hours'
  GROUP BY hours_to_match
)
-- Shows that movement typically peaks 2-4 hours before match
```

## API Endpoints

### 1. Smart Money Dashboard
```http
GET /api/smart-money/dashboard
```

Response:
```json
{
  "hot_movers": [
    {
      "match": "Fenerbahce vs Galatasaray",
      "volume_rank": 1,
      "volume_percentage": 8.01,
      "key_movement": {
        "market": "1X2",
        "outcome": "1",
        "change": "1.20 → 3.70 (+208%)"
      },
      "insight": "Public money on away team",
      "suggested_action": "Consider home value at 3.70"
    }
  ],
  "hidden_gems": [
    {
      "match": "Antalyaspor vs Konyaspor",
      "volume_rank": 91,
      "volume_percentage": 0.07,
      "key_movement": {
        "market": "1X2",
        "outcome": "X",
        "change": "3.20 → 4.50 (+40%)"
      },
      "insight": "Sharp money detected",
      "confidence": 0.82
    }
  ],
  "volume_surges": [
    {
      "match": "Barcelona vs Atletico",
      "current_volume": 4.2,
      "hourly_change": "+280%",
      "trigger": "Messi injury news"
    }
  ]
}
```

### 2. Event Intelligence Report
```http
GET /api/events/{slug}/intelligence
```

Response:
```json
{
  "event": "fenerbahce-vs-galatasaray-2025-06-05",
  "intelligence": {
    "volume_metrics": {
      "current_percentage": 8.01,
      "rank": 1,
      "percentile": 99.0,
      "trend": "INCREASING"
    },
    "money_type_analysis": {
      "sharp_probability": 0.15,
      "public_probability": 0.85,
      "evidence": [
        "High volume rank (#1)",
        "Favorite drifting significantly",
        "Movement started after team news"
      ]
    },
    "historical_patterns": {
      "similar_volume_events_roi": -3.2,
      "similar_movement_events_roi": 12.4,
      "pattern_match": "PUBLIC_OVERREACTION"
    },
    "recommendations": [
      {
        "action": "BACK_DRIFT",
        "market": "1X2",
        "selection": "1",
        "reasoning": "Public overreaction to minor news",
        "confidence": 0.75
      }
    ]
  }
}
```

## Visual Components

### 1. Volume vs Movement Scatter Plot
```
Movement % 
   ↑
200|            • (Hidden Gem)
   |
100|
   |  
 50|     •••• (Hot Movers)
   |  •••••••
  0|••••••••••••••(Stable)
   |______|______|______→ Volume %
   0      5      10
```

### 2. Money Flow Indicator
```
SHARP MONEY ◀━━━━━━━━━━█━━━━━━━━━━▶ PUBLIC MONEY
                       65% Public

Based on: High volume + Favorite drifting
```

### 3. Time-based Volume Heat Map
```
Hours to KO:  48  24  12   6   3   1   0
Volume:       ░░  ▒▒  ▓▓  ██  ██  ██  ██
Movement:     ░░  ░░  ▒▒  ▓▓  ██  ▓▓  ▒▒
```

## Implementation Priority

1. **Phase 1**: Basic volume tracking + hot movers identification
2. **Phase 2**: Historical patterns + sharp vs public classification  
3. **Phase 3**: ML model for outcome prediction based on volume/odds patterns
4. **Phase 4**: Real-time alerts and automated betting suggestions

## Why This Works

1. **Volume reveals interest**: High volume = public attention
2. **Movement reveals information**: Big moves = new information entering market
3. **Combination reveals opportunity**: 
   - High volume + movement = fade the public
   - Low volume + movement = follow the sharps
   - Patterns repeat = profitable systematic approach

This creates a complete betting intelligence system that transforms raw data into actionable insights!