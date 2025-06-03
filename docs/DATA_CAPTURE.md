# Iddaa Core - Data Capture Overview

This document provides a comprehensive overview of all data captured by the Iddaa Core system as of the current implementation.

## System Overview

The Iddaa Core system is a comprehensive betting intelligence platform that automatically captures, processes, and analyzes data from Turkish betting platform Iddaa. The system runs automated jobs every few minutes to build a complete picture of the betting landscape.

## Data Sources & APIs

### Primary Iddaa APIs

- **Events API**: `GET /sportsbook/events?st=1&type=0&version=0`
- **Single Event API**: `GET /sportsbook/event/{id}`
- **Competitions API**: `GET /sportsbook/competitions`
- **Sport Info API**: `GET /sportsbook/info`
- **Market Config API**: `GET /sportsbook/get_market_config`
- **Volume API**: `GET /sportsbook/played-event-percentage?sportType=1`
- **Distribution API**: `GET /sportsbook/outcome-play-percentages?sportType=1`
- **Statistics API**: `GET https://statisticsv2.iddaa.com/broadage/getEventListCache?SportId=1&SearchDate=YYYY-MM-DD`
- **Config API**: `GET https://contentv2.iddaa.com/appconfig?platform=WEB`

## Captured Data Categories

## 1. Sports & Competitions

### Sports Information

- **Sport metadata**: ID, name, code, slug
- **Live counts**: Current live events, upcoming events
- **Event statistics**: Total events, odds count
- **Feature flags**: Has results, king odds, digital content
- **Update tracking**: Created/updated timestamps

### Competitions (Leagues & Tournaments)

- **Basic info**: Iddaa ID, external reference, names (short/full)
- **Hierarchy**: Country code, parent competition relationships
- **Sport associations**: Which sport the competition belongs to
- **Metadata**: Icon URLs, slugs for web integration
- **Auto-generated**: URL-friendly slugs for API endpoints

## 2. Teams & Events

### Team Data

- **Identifiers**: External ID, internal ID, slug
- **Names**: Full name, short name variations
- **Geographic**: Country information
- **Branding**: Logo URLs
- **Tracking**: Creation and update timestamps

### Event (Match) Information

- **Core data**: External ID, competition, home/away teams
- **Scheduling**: Event date and time
- **Status tracking**: Scheduled, live, finished, cancelled
- **Live match data**:
  - Current scores (home/away)
  - Match minute and half
  - Live status indicator
- **Betting metadata**:
  - Volume percentage and ranking
  - Volume update timestamps
- **URL structure**: Auto-generated slugs for web access

## 3. Comprehensive Odds Data

### Current Odds (Live State)

- **Market identification**: Event, market type, outcome
- **Pricing data**:
  - Current odds value
  - Opening odds (starting value)
  - Highest/lowest values seen
  - Winning odds (special odds type)
- **Movement tracking**:
  - Total movement (highest - lowest)
  - Movement percentage from opening
  - Last update timestamp
- **Market context**: Market type, outcome names

### Historical Odds Tracking

- **Change detection**: Only stores when odds actually change
- **Price history**: Previous value, new value, change amount
- **Temporal data**: Exact timestamp of each change
- **Movement analysis**: Percentage change calculations
- **Trend identification**: Direction and magnitude of movements

### Market Types & Configuration

- **Market definitions**: Code, name, description
- **Display configuration**: Slugs, sort order
- **Betting rules**: Market-specific configurations
- **Platform integration**: Display names, descriptions
- **Auto-generation**: URL-friendly market slugs

## 4. Betting Intelligence

### Volume Analytics

- **Event popularity**: Betting volume percentage per event
- **Ranking system**: Volume-based event rankings
- **Historical tracking**: Volume changes over time
- **Popularity indicators**: High/medium/low categorization
- **Sharp detection**: Volume vs odds movement analysis

### Outcome Distribution Analysis

- **Public betting patterns**: Percentage of bets per outcome
- **Historical changes**: Distribution shifts over time
- **Market efficiency**: Public vs implied probability
- **Contrarian opportunities**: Over/under-bet identification
- **Value detection**: Bias percentage calculations

### Advanced Analytics (Materialized Views)

- **Contrarian betting opportunities**:
  - Heavily backed favorites to fade
  - Public vs sharp money indicators
  - Overbet percentage calculations
  - Strategy recommendations
- **Volume trend analysis**:
  - Hot movers (high volume + movement)
  - Hidden gems (low volume + movement)
  - Popularity categorization
- **Big mover detection**:
  - Significant odds movements
  - Movement thresholds and alerts
  - Time-based movement tracking

## 5. Live Match Statistics

### Real-Time Match Data

- **Game state**: Live indicator, match minute, half
- **Scoring**: Real-time score updates
- **Match events timeline**:
  - Goals, cards, substitutions
  - Minute-by-minute event tracking
  - Player-specific events
  - Team attribution (home/away)

### Detailed Match Statistics

- **Possession**: Ball possession percentages
- **Shooting**: Shots, shots on target
- **Set pieces**: Corners, free kicks, throw-ins
- **Disciplinary**: Yellow cards, red cards, fouls
- **Goalkeeping**: Saves, goal kicks
- **Advanced metrics**: Offsides tracking
- **Team-level aggregation**: Home vs away statistics

### Match Event Timeline

- **Event types**: Goals, cards, substitutions, etc.
- **Temporal tracking**: Exact minute of occurrence
- **Player attribution**: Specific player involvement
- **Team context**: Home vs away team events
- **Duplicate prevention**: Unique event constraints
- **Description details**: Event-specific descriptions

## 6. Prediction & AI Integration

### Prediction Framework

- **Event associations**: Predictions linked to specific events
- **Model tracking**: Prediction model identification
- **Confidence scoring**: Probability/confidence levels
- **Outcome prediction**: Specific predicted outcomes
- **Temporal tracking**: When predictions were made
- **Accuracy analysis**: Prediction vs actual result tracking

## Automation Schedule

### High Frequency (Live Data)

- **Events & Odds Sync**: Every 5 minutes
  - Bulk events API + individual detailed calls
  - Rate-limited (10 requests/second)
  - Comprehensive market coverage
- **Statistics Sync**: Every 15 minutes (8 AM - 11 PM)
  - Live scores and match events
  - Today + yesterday's events

### Medium Frequency (Intelligence)

- **Volume Sync**: Every 20 minutes
  - Betting volume percentages
  - Event popularity rankings
- **Distribution Sync**: Every hour
  - Outcome betting distributions
  - Public betting patterns

### Lower Frequency (Configuration)

- **Analytics Refresh**: Every 6 hours
  - Materialized view updates
  - Performance optimization
- **Market Config**: Daily at 6 AM
  - Market type definitions
  - Betting rule updates
- **Competitions**: Daily at 8 AM
  - League/tournament updates
  - New competition discovery
- **Platform Config**: Weekly (Mondays)
  - Platform configuration changes
  - Feature flag updates

## Data Relationships

### Core Entity Relationships

```
Sports ←→ Competitions ←→ Events ←→ Teams
  ↓           ↓           ↓        ↓
Market Types → Current Odds → Odds History
  ↓           ↓           ↓        ↓
Predictions ← Volume Data ← Distribution Data
  ↓           ↓           ↓        ↓
Match Stats ← Match Events ← Live Data
```

### Advanced Analytics Views

- **Big Movers**: Events with significant odds movement
- **Contrarian Bets**: Public vs sharp money opportunities
- **Volume Trends**: Popularity and movement correlation
- **Value Opportunities**: Market inefficiency detection

## Technical Implementation

### Database Architecture

- **PostgreSQL**: Primary database with advanced features
- **Materialized Views**: Pre-computed analytics for performance
- **Generated Columns**: Automatic calculations (movement %)
- **Triggers**: Auto-updating timestamps and slugs
- **Indexes**: Optimized for fast lookups and analytics

### API Integration

- **Rate Limiting**: Respectful API usage patterns
- **Error Handling**: Graceful failure recovery
- **Batch Processing**: Efficient bulk operations
- **Hybrid Approach**: Bulk + detailed API strategies

### Data Quality

- **Duplicate Prevention**: Unique constraints
- **Data Validation**: Type safety and bounds checking
- **Relationship Integrity**: Foreign key constraints
- **Audit Trail**: Complete change tracking

## Use Cases & Applications

### Betting Intelligence

- **Line shopping**: Find best odds across markets
- **Movement alerts**: Detect significant odds changes
- **Value betting**: Identify market inefficiencies
- **Contrarian strategies**: Fade public betting patterns

### Analytics & Insights

- **Market analysis**: Volume vs movement correlation
- **Public sentiment**: Betting distribution patterns
- **Sharp money detection**: Low volume, high movement
- **Performance tracking**: Prediction accuracy

### Live Betting

- **Real-time scores**: Live match tracking
- **In-play opportunities**: Live odds monitoring
- **Match context**: Statistics-informed betting
- **Event timeline**: Match progression tracking

### Research & Development

- **Historical analysis**: Trend identification
- **Model training**: ML/AI prediction models
- **Market research**: Betting behavior analysis
- **Strategy backtesting**: Historical performance validation

## Data Retention & Performance

### Storage Optimization

- **Incremental updates**: Only store changes
- **Materialized views**: Pre-computed heavy queries
- **Indexing strategy**: Optimized for common access patterns
- **Archival policies**: Historical data management

### Performance Features

- **Connection pooling**: Efficient database usage
- **Batch operations**: Bulk inserts and updates
- **Query optimization**: Efficient SQL patterns
- **Caching layers**: Reduced API load

## Security & Compliance

### Data Protection

- **No sensitive data**: Public betting information only
- **API rate limiting**: Respectful third-party usage
- **Error boundaries**: Isolated failure handling
- **Audit logging**: Complete operation tracking

This comprehensive data capture system provides a complete foundation for advanced betting analytics, market intelligence, and automated trading strategies.
