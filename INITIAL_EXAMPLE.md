## FEATURE:

Build a full-stack Daily Fantasy Sports (DFS) lineup optimizer using **100% FREE sports data APIs**. This system replicates SaberSim's core functionality without any API costs.

**Core Features:**
1. **Multi-Sport Support with Free APIs**:
   - ESPN Hidden API for live data (NFL, NBA, MLB, NHL)
   - TheSportsDB for supplementary team/player info
   - BALLDONTLIE for specialized NBA statistics
   - MySportsFeeds free tier for advanced analytics

2. **Smart Data Aggregation Engine**:
   - Intelligent API rotation to avoid rate limits
   - Cross-source data validation
   - Local caching with Redis for performance
   - Automatic failover between API sources
   - Historical data accumulation

3. **Lineup Optimization Engine**: 
   - Knapsack algorithm for salary cap optimization
   - Correlation/stacking based on free game data
   - Position constraints per DraftKings/FanDuel
   - Multi-lineup generation (up to 150 lineups)
   - Diversity controls using cached data

4. **Monte Carlo Simulation Engine**:
   - Uses free historical stats for projections
   - Player performance distributions from ESPN data
   - Correlation calculations from game logs
   - Contest simulation without paid data
   - Real-time progress visualization

5. **Free Data Management**:
   - ESPN API for real-time scores/stats
   - Player projections from historical averages
   - Ownership projections from social sentiment
   - CSV import/export for manual adjustments
   - Automated data refresh within rate limits

**Technical Architecture:**
- Backend: Go with Gin, PostgreSQL, Redis, WebSockets
- Frontend: React with TypeScript, TailwindCSS, Recharts
- API Gateway: Custom middleware for rate limit management
- Caching: Aggressive Redis caching to minimize API calls
- Queue: Background jobs for data aggregation

## EXAMPLES:

Place these examples in the `examples/` folder:

1. **espn_api_client.go** - ESPN Hidden API integration with endpoints:
   ```go
   // Live scores: site.api.espn.com/apis/site/v2/sports/basketball/nba/scoreboard
   // Player stats: sports.core.api.espn.com/v2/sports/basketball/leagues/nba/athletes/{id}
   // Team rosters: site.api.espn.com/apis/site/v2/sports/basketball/nba/teams/{id}/roster
   ```

2. **api_aggregator.go** - Multi-source data aggregation:
   ```go
   // Combines ESPN, TheSportsDB, and BALLDONTLIE data
   // Implements intelligent fallback and validation
   // Manages rate limits across all sources
   ```

3. **free_data_projections.go** - Generate projections from free data:
   ```go
   // Calculate player projections from last 10 games (ESPN)
   // Apply matchup adjustments from historical data
   // Factor in pace and defensive ratings
   ```

4. **rate_limit_manager.go** - Smart rate limiting:
   ```go
   // Track API usage per source
   // Implement exponential backoff
   // Rotate between sources based on limits
   ```

5. **cache_strategy.go** - Redis caching patterns:
   ```go
   // Cache player stats for 1 hour
   // Cache game results permanently
   // Implement cache warming strategies
   ```

6. **correlation_from_free_data.go** - Build correlations without paid data:
   ```go
   // Calculate team stacking correlations from game logs
   // Identify correlated player performances
   // Use historical teammate data
   ```

## DOCUMENTATION:

### **Free API Integration Guide**

1. **ESPN Hidden API**:
   - **No authentication required**
   - Base URLs:
     - NFL: `site.api.espn.com/apis/site/v2/sports/football/nfl/`
     - NBA: `site.api.espn.com/apis/site/v2/sports/basketball/nba/`
     - MLB: `site.api.espn.com/apis/site/v2/sports/baseball/mlb/`
     - NHL: `site.api.espn.com/apis/site/v2/sports/hockey/nhl/`
   - Key endpoints: `/scoreboard`, `/teams`, `/athletes`
   - Rate limits: Unofficial but stable for moderate use

2. **TheSportsDB** (Free Tier):
   - API Key: Register at thesportsdb.com
   - Base URL: `https://www.thesportsdb.com/api/v1/json/{API_KEY}/`
   - Use for: Team logos, player images, league standings
   - Rate limits: Very generous, no strict limits

3. **BALLDONTLIE**:
   - Free tier: 5 requests/minute
   - Base URL: `https://api.balldontlie.io/v1/`
   - Specialized NBA statistics and averages
   - TypeScript SDK available

4. **MySportsFeeds**:
   - Free tier: 500 API calls/day
   - Requires registration for API key
   - Advanced statistics and historical data
   - Best for batch data updates

### **Implementation Priorities**

**Phase 1: Core Free Data (Week 1)**
- Integrate ESPN API for live scores/stats
- Set up Redis caching infrastructure
- Build basic projection model from historical data
- Implement rate limit management

**Phase 2: Enhanced Features (Week 2)**
- Add TheSportsDB for media assets
- Integrate BALLDONTLIE for NBA deep stats
- Build correlation matrices from free data
- Implement API failover system

**Phase 3: Advanced Analytics (Week 3)**
- Add MySportsFeeds for historical analysis
- Build ownership projection models
- Implement advanced caching strategies
- Create data validation layer

## OTHER CONSIDERATIONS:

1. **Rate Limit Strategies**:
   - Cache everything aggressively (1-24 hour TTL)
   - Batch API requests during off-peak hours
   - Use webhooks where available
   - Implement request queuing system
   - Monitor usage dashboards

2. **Data Quality Assurance**:
   - Cross-validate critical stats across sources
   - Flag statistical anomalies
   - Maintain audit logs of data sources
   - Build confidence scores for projections

3. **Free API Limitations & Workarounds**:
   - **No real-time ownership**: Build social sentiment analyzer
   - **Limited injury data**: Scrape team reports (ethically)
   - **No betting lines**: Use historical scoring patterns
   - **Delayed updates**: Implement smart polling strategies

4. **Scalability with Free APIs**:
   - Design for 100-1000 concurrent users initially
   - Implement user-based rate limiting
   - Use CDN for static assets
   - Consider peer-to-peer data sharing

5. **Monetization Options**:
   - Premium features with faster updates
   - Advanced analytics using cached data
   - White-label solution for DFS groups
   - Data export tools for power users

6. **Legal Compliance**:
   - Respect all API terms of service
   - Implement user agent headers
   - No aggressive scraping
   - Clear data attribution

7. **MVP Focus with Free Data**:
   - Start with NBA (best free data availability)
   - Focus on last 10 games for projections
   - Simple correlation model
   - 20 lineup maximum initially
   - Manual ownership input option

## EXAMPLE API RESPONSES:

### ESPN NBA Scoreboard
```json
{
  "leagues": [{
    "id": "46",
    "name": "National Basketball Association",
    "abbreviation": "NBA",
    "events": [{
      "id": "401585000",
      "date": "2024-01-15T00:00Z",
      "competitions": [{
        "competitors": [{
          "id": "5",
          "team": {
            "id": "5",
            "abbreviation": "CLE",
            "displayName": "Cleveland Cavaliers"
          },
          "score": "95"
        }]
      }]
    }]
  }]
}
```

### BALLDONTLIE Player Stats
```json
{
  "data": [{
    "id": 237,
    "player": {
      "id": 237,
      "first_name": "LeBron",
      "last_name": "James",
      "position": "F",
      "team_id": 14
    },
    "pts": 27.1,
    "reb": 7.5,
    "ast": 7.4,
    "games_played": 55
  }]
}
```

This approach creates a sustainable, cost-free DFS optimizer that can compete with paid services while maintaining zero operational API costs.