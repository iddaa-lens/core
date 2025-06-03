# Odds Timeline Graph API

## Overview
Provide time-series data for rendering interactive odds movement graphs.

## API Endpoints

### 1. Get Odds Timeline for Event

```http
GET /api/events/{slug}/odds-timeline?market={market_code}&outcomes={outcomes}&interval={interval}
```

#### Parameters:
- `slug`: Event slug (e.g., "fenerbahce-vs-galatasaray-2025-06-05")
- `market`: Market code (e.g., "1X2", "OU_2_5")
- `outcomes`: Comma-separated outcomes (e.g., "1,2" or "all")
- `interval`: Data point interval (e.g., "5m", "30m", "1h", "raw")

#### Response:
```json
{
  "event": {
    "slug": "fenerbahce-vs-galatasaray-2025-06-05",
    "home_team": "Fenerbahce",
    "away_team": "Galatasaray",
    "event_date": "2025-06-05T18:00:00Z",
    "current_status": "scheduled"
  },
  "market": {
    "code": "1X2",
    "name": "Match Result"
  },
  "timeline": {
    "start_time": "2025-06-01T10:00:00Z",
    "end_time": "2025-06-02T16:30:00Z",
    "data_points": [
      {
        "timestamp": "2025-06-01T10:00:00Z",
        "values": {
          "1": 1.20,
          "X": 5.50,
          "2": 12.00
        },
        "event_marker": null
      },
      {
        "timestamp": "2025-06-01T14:00:00Z",
        "values": {
          "1": 1.22,
          "X": 5.40,
          "2": 11.50
        },
        "event_marker": null
      },
      {
        "timestamp": "2025-06-02T14:00:00Z",
        "values": {
          "1": 3.70,
          "X": 3.40,
          "2": 2.10
        },
        "event_marker": {
          "type": "sharp_movement",
          "description": "208% drift on home win",
          "severity": "high"
        }
      }
    ],
    "statistics": {
      "1": {
        "opening": 1.20,
        "current": 3.70,
        "highest": 3.70,
        "lowest": 1.20,
        "change_percent": 208.33,
        "volatility": 0.85
      },
      "X": {
        "opening": 5.50,
        "current": 3.40,
        "highest": 5.50,
        "lowest": 3.40,
        "change_percent": -38.18,
        "volatility": 0.42
      },
      "2": {
        "opening": 12.00,
        "current": 2.10,
        "highest": 12.00,
        "lowest": 2.10,
        "change_percent": -82.50,
        "volatility": 0.93
      }
    }
  }
}
```

### 2. SQL Query for Timeline Data

```sql
-- Function to get timeline data with configurable intervals
CREATE OR REPLACE FUNCTION get_odds_timeline(
    p_event_slug VARCHAR,
    p_market_code VARCHAR,
    p_interval INTERVAL DEFAULT '30 minutes'
) RETURNS TABLE (
    time_bucket TIMESTAMP,
    outcome VARCHAR,
    odds_value DECIMAL,
    change_from_previous DECIMAL,
    is_significant_change BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    WITH time_series AS (
        SELECT 
            date_trunc('minute', 
                generate_series(
                    (SELECT MIN(recorded_at) FROM odds_history oh 
                     JOIN events e ON oh.event_id = e.id 
                     WHERE e.slug = p_event_slug),
                    (SELECT MAX(recorded_at) FROM odds_history oh 
                     JOIN events e ON oh.event_id = e.id 
                     WHERE e.slug = p_event_slug),
                    p_interval
                )
            ) as time_bucket
    ),
    odds_data AS (
        SELECT 
            date_trunc('minute', oh.recorded_at) as recorded_time,
            oh.outcome,
            oh.odds_value,
            oh.previous_value,
            ABS(oh.change_percentage) > 10 as is_significant
        FROM odds_history oh
        JOIN events e ON oh.event_id = e.id
        JOIN market_types mt ON oh.market_type_id = mt.id
        WHERE e.slug = p_event_slug
          AND mt.code = p_market_code
    )
    SELECT 
        ts.time_bucket,
        od.outcome,
        -- Get the most recent odds value for each time bucket
        LAST_VALUE(od.odds_value) OVER (
            PARTITION BY od.outcome 
            ORDER BY od.recorded_time
            RANGE BETWEEN UNBOUNDED PRECEDING AND ts.time_bucket FOLLOWING
        ) as odds_value,
        od.odds_value - od.previous_value as change_from_previous,
        od.is_significant
    FROM time_series ts
    CROSS JOIN LATERAL (
        SELECT DISTINCT outcome FROM odds_data
    ) outcomes
    LEFT JOIN odds_data od ON od.outcome = outcomes.outcome
        AND od.recorded_time <= ts.time_bucket
        AND od.recorded_time > ts.time_bucket - p_interval
    ORDER BY ts.time_bucket, od.outcome;
END;
$$ LANGUAGE plpgsql;
```

### 3. Optimized Query for Real-time Updates

```sql
-- Materialized view for fast graph rendering
CREATE MATERIALIZED VIEW odds_timeline_cache AS
WITH intervals AS (
    SELECT 
        e.id as event_id,
        e.slug as event_slug,
        date_trunc('minute', oh.recorded_at) as time_point,
        mt.code as market_code,
        oh.outcome,
        oh.odds_value,
        oh.previous_value,
        oh.change_percentage,
        CASE 
            WHEN ABS(oh.change_percentage) > 50 THEN 'extreme'
            WHEN ABS(oh.change_percentage) > 20 THEN 'high'
            WHEN ABS(oh.change_percentage) > 10 THEN 'moderate'
            ELSE 'normal'
        END as movement_severity
    FROM odds_history oh
    JOIN events e ON oh.event_id = e.id
    JOIN market_types mt ON oh.market_type_id = mt.id
    WHERE e.event_date > NOW() - INTERVAL '7 days'
)
SELECT 
    event_id,
    event_slug,
    market_code,
    time_point,
    json_object_agg(
        outcome,
        json_build_object(
            'value', odds_value,
            'previous', previous_value,
            'change_pct', change_percentage,
            'severity', movement_severity
        )
    ) as odds_data
FROM intervals
GROUP BY event_id, event_slug, market_code, time_point;

CREATE INDEX idx_timeline_cache_lookup 
ON odds_timeline_cache (event_slug, market_code, time_point DESC);
```

## Frontend Implementation Examples

### 1. React Component with Recharts

```typescript
interface OddsTimelineProps {
  eventSlug: string;
  marketCode: string;
  outcomes?: string[];
}

const OddsTimelineGraph: React.FC<OddsTimelineProps> = ({ 
  eventSlug, 
  marketCode, 
  outcomes = ['1', 'X', '2'] 
}) => {
  const { data, isLoading } = useQuery({
    queryKey: ['odds-timeline', eventSlug, marketCode],
    queryFn: () => fetchOddsTimeline(eventSlug, marketCode, outcomes),
    refetchInterval: 30000 // Refresh every 30 seconds
  });

  if (isLoading) return <Skeleton />;

  // Transform data for Recharts
  const chartData = data.timeline.data_points.map(point => ({
    time: new Date(point.timestamp).toLocaleTimeString(),
    ...point.values,
    marker: point.event_marker
  }));

  return (
    <Card>
      <CardHeader>
        <h3>{data.event.home_team} vs {data.event.away_team}</h3>
        <p>{data.market.name}</p>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={400}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="time" />
            <YAxis domain={['dataMin - 0.5', 'dataMax + 0.5']} />
            <Tooltip 
              content={<CustomTooltip />}
              formatter={(value) => value.toFixed(2)}
            />
            <Legend />
            
            {/* Lines for each outcome */}
            <Line 
              type="monotone" 
              dataKey="1" 
              stroke="#2563eb" 
              name="Home Win"
              strokeWidth={2}
              dot={(props) => <CustomDot {...props} />}
            />
            <Line 
              type="monotone" 
              dataKey="X" 
              stroke="#16a34a" 
              name="Draw"
              strokeWidth={2}
            />
            <Line 
              type="monotone" 
              dataKey="2" 
              stroke="#dc2626" 
              name="Away Win"
              strokeWidth={2}
            />
            
            {/* Reference lines for significant events */}
            {chartData
              .filter(d => d.marker?.severity === 'high')
              .map((d, i) => (
                <ReferenceLine 
                  key={i}
                  x={d.time} 
                  stroke="#ff6b6b"
                  strokeDasharray="5 5"
                  label={d.marker.description}
                />
              ))
            }
          </LineChart>
        </ResponsiveContainer>
        
        {/* Statistics Summary */}
        <div className="grid grid-cols-3 gap-4 mt-4">
          {Object.entries(data.timeline.statistics).map(([outcome, stats]) => (
            <StatCard 
              key={outcome}
              outcome={outcome}
              stats={stats}
            />
          ))}
        </div>
      </CardContent>
    </Card>
  );
};

// Custom dot to highlight significant changes
const CustomDot = (props) => {
  const { cx, cy, payload } = props;
  if (payload.marker?.severity === 'high') {
    return (
      <circle 
        cx={cx} 
        cy={cy} 
        r={6} 
        fill="#ff6b6b" 
        stroke="#fff" 
        strokeWidth={2}
      />
    );
  }
  return null;
};
```

### 2. Advanced Features

```typescript
// Interactive timeline with zoom and pan
const InteractiveOddsTimeline = () => {
  const [timeRange, setTimeRange] = useState<[Date, Date]>([
    subDays(new Date(), 7),
    new Date()
  ]);
  
  const [selectedOutcome, setSelectedOutcome] = useState<string | null>(null);
  const [annotations, setAnnotations] = useState<Annotation[]>([]);

  return (
    <div>
      {/* Time range selector */}
      <TimeRangeSelector 
        value={timeRange}
        onChange={setTimeRange}
        presets={[
          { label: 'Last 24h', value: [subDays(new Date(), 1), new Date()] },
          { label: 'Last 7d', value: [subDays(new Date(), 7), new Date()] },
          { label: 'All time', value: null }
        ]}
      />
      
      {/* Main graph with zoom */}
      <ZoomableLineChart
        data={filteredData}
        xDomain={timeRange}
        onZoom={setTimeRange}
        highlightLine={selectedOutcome}
      >
        {/* Annotations for events */}
        {annotations.map(ann => (
          <Annotation
            key={ann.id}
            x={ann.timestamp}
            y={ann.value}
            label={ann.label}
            color={ann.color}
          />
        ))}
      </ZoomableLineChart>
      
      {/* Movement indicators */}
      <MovementIndicators
        data={data}
        onOutcomeSelect={setSelectedOutcome}
      />
    </div>
  );
};
```

### 3. Real-time Updates with WebSocket

```typescript
// WebSocket connection for live updates
const useLiveOddsTimeline = (eventSlug: string, marketCode: string) => {
  const [data, setData] = useState<TimelineData | null>(null);
  
  useEffect(() => {
    const ws = new WebSocket(`wss://api.example.com/live/odds/${eventSlug}/${marketCode}`);
    
    ws.onmessage = (event) => {
      const update = JSON.parse(event.data);
      
      setData(prev => {
        if (!prev) return null;
        
        // Add new data point
        const newPoint = {
          timestamp: update.timestamp,
          values: update.values,
          event_marker: update.significant ? {
            type: 'live_update',
            description: `${update.outcome}: ${update.old_value} → ${update.new_value}`,
            severity: update.change_pct > 20 ? 'high' : 'normal'
          } : null
        };
        
        return {
          ...prev,
          timeline: {
            ...prev.timeline,
            data_points: [...prev.timeline.data_points, newPoint].slice(-100) // Keep last 100 points
          }
        };
      });
    };
    
    return () => ws.close();
  }, [eventSlug, marketCode]);
  
  return data;
};
```

## Graph Visualization Options

### 1. **Line Chart** (Shown above)
- Best for: Continuous tracking over time
- Shows: Trends and patterns clearly

### 2. **Candlestick Chart**
```typescript
// Show open/high/low/close for each time period
const OddsCandlestickChart = ({ data }) => (
  <CandlestickChart
    data={data.map(period => ({
      date: period.timestamp,
      open: period.open_value,
      high: period.high_value,
      low: period.low_value,
      close: period.close_value,
      volume: period.change_count // Number of changes
    }))}
  />
);
```

### 3. **Area Chart with Confidence Bands**
```typescript
// Show volatility as shaded areas
const OddsConfidenceChart = ({ data }) => (
  <AreaChart data={data}>
    <Area
      dataKey="value"
      stroke="#2563eb"
      fill="#2563eb"
      fillOpacity={0.3}
    />
    <Area
      dataKey="upper_bound"
      stroke="none"
      fill="#2563eb"
      fillOpacity={0.1}
    />
    <Area
      dataKey="lower_bound"
      stroke="none"
      fill="#2563eb"
      fillOpacity={0.1}
    />
  </AreaChart>
);
```

### 4. **Heat Map Timeline**
```typescript
// Show all markets/outcomes as a heat map
const OddsHeatMap = ({ eventSlug }) => (
  <HeatMapGrid
    rows={['1X2', 'OU_2.5', 'BTTS']}
    columns={timeIntervals}
    getValue={(market, time) => getMovementIntensity(market, time)}
    colorScale={['#22c55e', '#eab308', '#ef4444']} // Green to red
  />
);
```

## Performance Optimization

```sql
-- Aggregate data for different zoom levels
CREATE TABLE odds_timeline_aggregates (
    event_id INTEGER,
    market_type_id INTEGER,
    outcome VARCHAR(100),
    time_bucket TIMESTAMP,
    bucket_size INTERVAL,
    open_value DECIMAL(10,3),
    close_value DECIMAL(10,3),
    high_value DECIMAL(10,3),
    low_value DECIMAL(10,3),
    change_count INTEGER,
    PRIMARY KEY (event_id, market_type_id, outcome, time_bucket, bucket_size)
);

-- Pre-aggregate for common intervals
INSERT INTO odds_timeline_aggregates
SELECT 
    event_id,
    market_type_id,
    outcome,
    date_trunc('hour', recorded_at) as time_bucket,
    '1 hour'::INTERVAL as bucket_size,
    FIRST_VALUE(odds_value) OVER w as open_value,
    LAST_VALUE(odds_value) OVER w as close_value,
    MAX(odds_value) OVER w as high_value,
    MIN(odds_value) OVER w as low_value,
    COUNT(*) OVER w as change_count
FROM odds_history
WINDOW w AS (
    PARTITION BY event_id, market_type_id, outcome, date_trunc('hour', recorded_at)
    ORDER BY recorded_at
    ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING
);
```

This gives you a complete solution for displaying odds timelines with various visualization options!