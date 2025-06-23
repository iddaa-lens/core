# Top 5 Features Using Volume & Distribution Data

## 1. **Sharp vs Public Money Divergence Alerts** 🎯

### Overview

Identifies when betting distribution (where people are betting) diverges significantly from betting volume (how much money is being bet). This is the classic "fade the public" strategy used by professional bettors.

### Key Metrics

- **Sharp Money Indicator**: High volume + Low distribution = Professionals betting big
- **Public Trap Indicator**: Low volume + High distribution = Casual bettors falling for popular picks
- **Divergence Score**: The gap between money percentage and bet percentage

### Example Implementation

```sql
-- Find games with sharp/public divergence
WITH divergence_analysis AS (
  SELECT 
    e.id,
    e.slug,
    e.betting_volume_percentage,
    od.outcome,
    od.bet_percentage,
    od.implied_probability,
    co.odds_value,
    ABS(e.betting_volume_percentage - od.bet_percentage) as divergence,
    CASE 
      WHEN e.betting_volume_percentage > od.bet_percentage + 20 THEN 'SHARP_MONEY'
      WHEN od.bet_percentage > e.betting_volume_percentage + 20 THEN 'PUBLIC_TRAP'
      ELSE 'BALANCED'
    END as money_type
  FROM events e
  JOIN outcome_distributions od ON od.event_id = e.id
  JOIN current_odds co ON co.event_id = e.id AND co.outcome = od.outcome
  WHERE e.event_date > NOW() 
    AND e.event_date < NOW() + INTERVAL '24 hours'
    AND od.market_id = 1 -- Match Result market
)
SELECT * FROM divergence_analysis 
WHERE money_type != 'BALANCED'
ORDER BY divergence DESC;
```

### User Interface

```text
⚡ SHARP MONEY ALERT: Fenerbahçe vs Galatasaray
├─ 73% of bets on Galatasaray (public favorite)
├─ 68% of money on Fenerbahçe (sharp side)
├─ Divergence Score: 41 points
├─ Odds Movement: 2.10 → 1.95 (following sharp money)
└─ Confidence: ⭐⭐⭐⭐⭐
```

---

## 2. **Reverse Line Movement Detector** 📈

### Overview

Tracks games where odds move in the opposite direction of public betting percentages. This is the strongest indicator that sharp money is influencing the market.

### Signal Strength Levels

- **Strong Reverse**: 70%+ public betting, odds move opposite direction
- **Moderate Reverse**: 60-70% public betting, odds move opposite
- **Confirmed Reverse**: Multiple odds movements against public sentiment

### Implementation

```sql
-- Detect reverse line movements with volume context
WITH recent_movements AS (
  SELECT 
    oh.event_id,
    oh.outcome,
    oh.odds_value,
    oh.previous_value,
    oh.change_percentage,
    oh.minutes_to_kickoff,
    ROW_NUMBER() OVER (PARTITION BY oh.event_id, oh.outcome 
                       ORDER BY oh.recorded_at DESC) as rn
  FROM odds_history oh
  WHERE oh.recorded_at > NOW() - INTERVAL '4 hours'
    AND ABS(oh.change_percentage) > 2.0
)
SELECT 
  e.slug,
  e.betting_volume_percentage,
  od.outcome,
  od.bet_percentage as public_bet_pct,
  rm.odds_value as current_odds,
  rm.previous_value as previous_odds,
  rm.change_percentage as odds_movement_pct,
  CASE 
    WHEN od.bet_percentage > 70 AND rm.change_percentage > 0 
      AND od.outcome IN ('1', 'Home') THEN 'STRONG_REVERSE_HOME'
    WHEN od.bet_percentage > 70 AND rm.change_percentage < 0 
      AND od.outcome IN ('2', 'Away') THEN 'STRONG_REVERSE_AWAY'
    WHEN od.bet_percentage BETWEEN 60 AND 70 AND 
      SIGN(od.bet_percentage - 50) != SIGN(rm.change_percentage) THEN 'MODERATE_REVERSE'
    ELSE 'NORMAL_MOVEMENT'
  END as movement_type,
  e.volume_rank
FROM outcome_distributions od
JOIN events e ON e.id = od.event_id
JOIN recent_movements rm ON rm.event_id = od.event_id 
  AND rm.outcome = od.outcome AND rm.rn = 1
WHERE e.event_date > NOW()
  AND od.bet_percentage > 60
  AND ABS(rm.change_percentage) > 2.0
ORDER BY ABS(od.bet_percentage - 50) * ABS(rm.change_percentage) DESC;
```

### Alert Example

```text
🔄 REVERSE LINE MOVEMENT DETECTED

Match: Beşiktaş vs Trabzonspor
├─ Public: 78% on Beşiktaş
├─ Line Move: Beşiktaş 1.75 → 1.85 (+5.7%)
├─ Volume Rank: #8 (high interest)
├─ Sharp Money Confidence: 85%
└─ Historical Success Rate: 67% (45/67 games)

💡 The books are making Beşiktaş less attractive despite heavy public betting.
   This suggests sharp money is on Trabzonspor.
```

---

## 3. **Late Money Steam Tracker** 💨

### Overview

Monitors dramatic volume and distribution changes in the final hours before kickoff. Late money moves often represent the most informed betting action.

### Time Windows

- **Early Steam**: 4-2 hours before kickoff
- **Prime Steam**: 2 hours to 30 minutes before
- **Late Steam**: Final 30 minutes (most valuable)

### Implementation

```sql
-- Track late money movements with historical comparison
WITH volume_trends AS (
  SELECT 
    vn.event_id,
    vn.volume_percentage as current_vol,
    vn.rank_position as current_rank,
    vh.volume_percentage as hist_vol,
    vh.rank_position as hist_rank,
    vn.recorded_at,
    (vn.volume_percentage - vh.volume_percentage) as vol_change,
    (vh.rank_position - vn.rank_position) as rank_improvement
  FROM betting_volume_history vn
  JOIN betting_volume_history vh ON vh.event_id = vn.event_id
  WHERE vn.recorded_at > NOW() - INTERVAL '30 minutes'
    AND vh.recorded_at BETWEEN NOW() - INTERVAL '2 hours' 
                           AND NOW() - INTERVAL '1 hour 50 minutes'
),
distribution_shifts AS (
  SELECT 
    dn.event_id,
    dn.outcome,
    dn.bet_percentage as current_dist,
    dh.bet_percentage as hist_dist,
    (dn.bet_percentage - dh.bet_percentage) as dist_shift
  FROM outcome_distribution_history dn
  JOIN outcome_distribution_history dh 
    ON dh.event_id = dn.event_id AND dh.outcome = dn.outcome
  WHERE dn.recorded_at > NOW() - INTERVAL '30 minutes'
    AND dh.recorded_at BETWEEN NOW() - INTERVAL '2 hours' 
                           AND NOW() - INTERVAL '1 hour 50 minutes'
)
SELECT 
  e.slug,
  ht.name || ' vs ' || at.name as match_name,
  e.event_date,
  vt.current_vol,
  vt.vol_change,
  vt.rank_improvement,
  ds.outcome,
  ds.current_dist,
  ds.dist_shift,
  co.odds_value,
  CASE 
    WHEN vt.vol_change > 2.0 AND vt.rank_improvement > 20 THEN 'HOT_STEAM'
    WHEN vt.vol_change > 1.0 AND ds.dist_shift > 15 THEN 'MODERATE_STEAM'
    WHEN vt.rank_improvement > 30 THEN 'VOLUME_SPIKE'
    ELSE 'NORMAL'
  END as steam_level
FROM volume_trends vt
JOIN events e ON e.id = vt.event_id
JOIN teams ht ON ht.id = e.home_team_id
JOIN teams at ON at.id = e.away_team_id
LEFT JOIN distribution_shifts ds ON ds.event_id = vt.event_id
LEFT JOIN current_odds co ON co.event_id = e.id AND co.outcome = ds.outcome
WHERE e.event_date BETWEEN NOW() AND NOW() + INTERVAL '3 hours'
  AND (vt.vol_change > 0.5 OR vt.rank_improvement > 10 OR ABS(ds.dist_shift) > 10)
ORDER BY vt.vol_change DESC, vt.rank_improvement DESC;
```

### Real-time Dashboard

```
⏰ LATE MONEY STEAM MOVES (Last 30 Minutes)

🔥 HOT STEAM:
1. Antalyaspor vs Konyaspor
   ├─ Volume: 0.5% → 2.3% (+360%!)
   ├─ Rank: #47 → #12 (↑35 spots)
   ├─ Over 2.5: 45% → 68% (+23%)
   ├─ Odds: 1.85 → 1.75 (-5.4%)
   └─ Action: STRONG BUY on Over 2.5

📈 MODERATE STEAM:
2. Göztepe vs Hatayspor
   ├─ Volume: 1.1% → 1.8% (+64%)
   ├─ Home Win: 55% → 71% (+16%)
   ├─ Odds: 2.20 → 2.05 (-6.8%)
   └─ Action: Consider Home Win
```

---

## 4. **Public Fade System** 🎪

### Overview

Systematically identifies and tracks opportunities to bet against overwhelming public consensus, especially in high-profile matches where casual money inflates lines.

### Key Indicators

- **Public Consensus**: 75%+ betting on one side
- **Volume Context**: Low volume rank despite high-profile match
- **Line Value**: Odds inflated by public action
- **Historical Performance**: Track success rate of fading public in similar spots

### Implementation

```sql
-- Advanced public fade opportunities
WITH public_sides AS (
  SELECT 
    od.event_id,
    od.outcome,
    od.bet_percentage,
    od.market_id,
    MAX(od.bet_percentage) OVER (PARTITION BY od.event_id, od.market_id) as max_bet_pct,
    CASE 
      WHEN od.bet_percentage = MAX(od.bet_percentage) 
           OVER (PARTITION BY od.event_id, od.market_id) THEN 'PUBLIC_SIDE'
      ELSE 'FADE_SIDE'
    END as side_type
  FROM outcome_distributions od
  WHERE od.market_id IN (1, 2, 3) -- Match Result, Over/Under, Handicap
),
fade_opportunities AS (
  SELECT 
    e.id,
    e.slug,
    e.betting_volume_percentage,
    e.volume_rank,
    ps.outcome as public_outcome,
    ps.bet_percentage as public_pct,
    ps_fade.outcome as fade_outcome,
    ps_fade.bet_percentage as fade_pct,
    co_pub.odds_value as public_odds,
    co_fade.odds_value as fade_odds,
    -- Calculate expected value
    (100.0 / co_fade.odds_value) as implied_prob,
    ps_fade.bet_percentage as actual_prob,
    ((100.0 - ps.bet_percentage) / co_fade.odds_value) - 1 as expected_value
  FROM events e
  JOIN public_sides ps ON ps.event_id = e.id AND ps.side_type = 'PUBLIC_SIDE'
  JOIN public_sides ps_fade ON ps_fade.event_id = e.id 
    AND ps_fade.market_id = ps.market_id 
    AND ps_fade.side_type = 'FADE_SIDE'
  JOIN current_odds co_pub ON co_pub.event_id = e.id 
    AND co_pub.outcome = ps.outcome
  JOIN current_odds co_fade ON co_fade.event_id = e.id 
    AND co_fade.outcome = ps_fade.outcome
  WHERE ps.bet_percentage > 70
    AND e.event_date > NOW()
    AND e.event_date < NOW() + INTERVAL '48 hours'
)
SELECT 
  f.*,
  CASE 
    WHEN f.public_pct > 80 AND f.expected_value > 0.05 THEN 'STRONG_FADE'
    WHEN f.public_pct > 75 AND f.expected_value > 0.02 THEN 'MODERATE_FADE'
    WHEN f.public_pct > 70 AND f.volume_rank > 50 THEN 'VALUE_FADE'
    ELSE 'MONITOR'
  END as fade_strength,
  l.name as league_name,
  ht.name as home_team,
  at.name as away_team
FROM fade_opportunities f
JOIN events e ON e.id = f.id
JOIN leagues l ON l.id = e.league_id
JOIN teams ht ON ht.id = e.home_team_id
JOIN teams at ON at.id = e.away_team_id
WHERE f.expected_value > 0
ORDER BY f.public_pct DESC, f.expected_value DESC;
```

### Fade Alert Example

```text
🎪 PUBLIC FADE OPPORTUNITY

Match: Galatasaray vs Fenerbahçe (Derby)
├─ Public: 83% on Galatasaray (1.65 odds)
├─ Fade: Fenerbahçe Draw/Win @ 2.20
├─ Expected Value: +8.5%
├─ Volume Rank: #2 (massive public interest)
├─ Historical Derby Fades: 19-12 (+24.5 units)
└─ Confidence: ⭐⭐⭐⭐

💡 Classic spot where casual money creates value on unpopular side
```

---

## 5. **Smart Money Confidence Score** 🧠

### Overview

Combines multiple factors to create a single confidence score (0-100) for each betting opportunity, helping users quickly identify the strongest plays.

### Score Components

- **Volume Spike Score** (0-25): Recent volume changes
- **Distribution Divergence** (0-25): Sharp vs public money gap  
- **Line Movement** (0-25): Odds movement aligned with sharp money
- **Timing Score** (0-25): How early the sharp money appeared

### Master Algorithm

```sql
-- Calculate comprehensive smart money confidence scores
WITH volume_scores AS (
  SELECT 
    e.id,
    e.betting_volume_percentage,
    e.volume_rank,
    -- Volume spike score (0-25)
    LEAST(25, 
      CASE 
        WHEN e.volume_rank <= 10 THEN 20
        WHEN e.volume_rank <= 25 THEN 15
        WHEN e.volume_rank <= 50 THEN 10
        ELSE 5
      END +
      CASE 
        WHEN e.betting_volume_percentage > 5 THEN 5
        WHEN e.betting_volume_percentage > 3 THEN 3
        ELSE 0
      END
    ) as volume_score
  FROM events e
),
divergence_scores AS (
  SELECT 
    od.event_id,
    od.outcome,
    od.bet_percentage,
    e.betting_volume_percentage,
    -- Divergence score (0-25)
    LEAST(25,
      GREATEST(0, ABS(od.bet_percentage - e.betting_volume_percentage) / 2)
    ) as divergence_score
  FROM outcome_distributions od
  JOIN events e ON e.id = od.event_id
  WHERE od.market_id = 1 -- Match result
),
movement_scores AS (
  SELECT 
    oh.event_id,
    oh.outcome,
    SUM(ABS(oh.change_percentage)) as total_movement,
    -- Line movement score (0-25)
    LEAST(25,
      SUM(CASE 
        WHEN oh.significance_level = 'extreme' THEN 10
        WHEN oh.significance_level = 'high' THEN 5
        ELSE 2
      END)
    ) as movement_score,
    -- Timing score based on when movement happened (0-25)
    LEAST(25,
      AVG(CASE 
        WHEN oh.minutes_to_kickoff > 120 THEN 20  -- Early movement
        WHEN oh.minutes_to_kickoff > 60 THEN 15   -- Mid movement
        WHEN oh.minutes_to_kickoff > 30 THEN 10   -- Late movement
        ELSE 5                                     -- Very late
      END)
    ) as timing_score
  FROM odds_history oh
  WHERE oh.recorded_at > NOW() - INTERVAL '24 hours'
  GROUP BY oh.event_id, oh.outcome
)
SELECT 
  e.slug,
  ht.name || ' vs ' || at.name as match,
  COALESCE(ds.outcome, 'Draw') as sharp_pick,
  ROUND(
    COALESCE(vs.volume_score, 0) +
    COALESCE(ds.divergence_score, 0) +
    COALESCE(ms.movement_score, 0) +
    COALESCE(ms.timing_score, 0)
  ) as confidence_score,
  -- Detailed breakdown
  COALESCE(vs.volume_score, 0) as volume_component,
  COALESCE(ds.divergence_score, 0) as divergence_component,
  COALESCE(ms.movement_score, 0) as movement_component,
  COALESCE(ms.timing_score, 0) as timing_component,
  -- Context
  e.betting_volume_percentage,
  e.volume_rank,
  ds.bet_percentage as public_percentage,
  co.odds_value as current_odds,
  e.event_date
FROM events e
JOIN teams ht ON ht.id = e.home_team_id
JOIN teams at ON at.id = e.away_team_id
LEFT JOIN volume_scores vs ON vs.id = e.id
LEFT JOIN divergence_scores ds ON ds.event_id = e.id
LEFT JOIN movement_scores ms ON ms.event_id = e.id AND ms.outcome = ds.outcome
LEFT JOIN current_odds co ON co.event_id = e.id AND co.outcome = ds.outcome
WHERE e.event_date > NOW()
  AND e.event_date < NOW() + INTERVAL '48 hours'
  AND (
    COALESCE(vs.volume_score, 0) +
    COALESCE(ds.divergence_score, 0) +
    COALESCE(ms.movement_score, 0) +
    COALESCE(ms.timing_score, 0)
  ) >= 40  -- Minimum confidence threshold
ORDER BY confidence_score DESC
LIMIT 10;
```

### Smart Money Dashboard

```
🧠 TOP SMART MONEY PLAYS (Next 48 Hours)

1. ⭐ 92/100 - Konyaspor vs Alanyaspor
   ├─ Pick: Under 2.5 Goals @ 1.85
   ├─ Volume: 95th percentile (25/25)
   ├─ Divergence: 31% public vs 72% money (23/25)
   ├─ Movement: 2.10 → 1.85 in 3 moves (22/25)
   ├─ Timing: Sharp money 3 hours early (22/25)
   └─ Action: MAXIMUM CONFIDENCE

2. ⭐ 87/100 - Sivasspor vs Giresunspor  
   ├─ Pick: Home Win @ 2.05
   ├─ Volume: 89th percentile (22/25)
   ├─ Divergence: 41% public vs 65% money (20/25)
   ├─ Movement: 2.35 → 2.05 steady (20/25)
   ├─ Timing: Early sharp action (25/25)
   └─ Action: STRONG PLAY

3. ⭐ 76/100 - Başakşehir vs Antalyaspor
   ├─ Pick: Away +1.5 @ 1.75
   ├─ Volume: 71st percentile (15/25)
   ├─ Divergence: 25% public vs 48% money (18/25)
   ├─ Movement: 1.90 → 1.75 late (18/25)
   ├─ Timing: Mixed timing (25/25)
   └─ Action: SOLID VALUE
```

## Implementation Priority

1. **Phase 1**: Sharp vs Public Divergence (easiest to implement, immediate value)
2. **Phase 2**: Smart Money Confidence Score (combines everything, great UX)
3. **Phase 3**: Reverse Line Movement (requires good odds history)
4. **Phase 4**: Late Money Steam (needs real-time processing)
5. **Phase 5**: Public Fade System (requires historical performance tracking)

## Technical Considerations

### Performance Optimization

- Create materialized views for real-time dashboards
- Use Redis for caching confidence scores (5-minute TTL)
- Implement WebSocket connections for live steam tracking

### Data Requirements

- Minimum 2 weeks of historical data for patterns
- Real-time odds updates (at least every 5 minutes)
- Volume data updates every 10-20 minutes

### Monitoring & Alerts

- Set up alerts for extreme divergences (>40 points)
- Monitor query performance on high-traffic days
- Track feature accuracy with backtesting

## Revenue Potential

These features could support a premium subscription model:

- **Basic**: Public fade alerts only ($9.99/month)
- **Pro**: All features with 10 alerts/day ($29.99/month)
- **Sharp**: Unlimited access + API ($99.99/month)

The combination of volume and distribution data creates unique insights that most betting sites don't provide, giving you a strong competitive advantage.
