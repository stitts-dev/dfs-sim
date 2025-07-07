## FEATURE: Golf Sport Addition to DFS Optimizer

Add PGA Tour golf as a supported sport in the DFS lineup optimizer using **100% FREE golf data APIs**, enabling fantasy golf lineup optimization with the same core features as other sports.

**Core Features:**
1. **Free Golf API Integration**:
   - ESPN Hidden API for live PGA Tour leaderboards and player stats
   - Free Golf API (FreeWebApi) for world rankings and detailed scorecards
   - Live Golf API for real-time tournament scoring
   - GolfCourseAPI for course-specific insights (30,000+ courses)

2. **Golf-Specific Data Engine**:
   - Tournament leaderboard tracking with real-time updates
   - Player statistics aggregation (scoring averages, recent form)
   - Course history analysis for player-venue compatibility
   - Weather impact correlation from historical data
   - Shot-by-shot data for advanced analytics

3. **Golf Lineup Optimization**:
   - DraftKings/FanDuel golf roster rules (6 players)
   - Salary cap optimization for golf contests
   - Cut probability calculations from historical data
   - Ownership projections based on betting odds/popularity
   - Tournament finish distribution modeling

4. **Golf Simulation Features**:
   - Monte Carlo simulations for tournament outcomes
   - Player scoring distributions by course type
   - Round-by-round variance modeling
   - Weather-adjusted projections
   - Field strength considerations

5. **Golf Data Management**:
   - Live leaderboard updates during tournaments
   - Historical tournament results database
   - Player form tracking (last 5/10 tournaments)
   - Course characteristics database
   - Strokes gained statistics where available

**Technical Architecture:**
- Extends existing Go backend with golf-specific services
- New golf API client package with rate limiting
- Golf-specific database tables for players/tournaments
- Frontend components for golf contest types
- Redis caching optimized for 4-day tournament data

## EXAMPLES:

Place these examples in the `examples/golf/` folder:

1. **espn_golf_client.go** - ESPN Golf API integration:
   ```go
   // Tournament leaderboard: site.api.espn.com/apis/site/v2/sports/golf/pga/scoreboard
   // Player stats: site.web.api.espn.com/apis/common/v3/sports/golf/athletes/{player_id}/stats
   // Tournament history: sports.core.api.espn.com/v2/sports/golf/leagues/pga/seasons/{year}/athletes/{player_id}/eventlog
   // Player search: site.web.api.espn.com/apis/common/v3/search?query={name}&type=player
   ```

2. **free_golf_api_client.go** - FreeWebApi Golf integration:
   ```go
   // World rankings: GET /world-ranking/{year}
   // Tournament leaderboards: GET /leaderboards/{tournId}/{year}/{roundId}
   // Detailed scorecards: GET /scorecards/{tournId}/{year}/{playerId}/{roundId}
   // Player statistics: GET /players
   // Tournament earnings: GET /tournaments/{tournId}/{year}
   ```

3. **golf_projections.go** - Generate golf-specific projections:
   ```go
   // Calculate expected tournament score from recent form
   // Apply course history adjustments
   // Factor in field strength and cut lines
   // Weather impact modeling
   ```

4. **golf_correlation_builder.go** - Golf-specific correlations:
   ```go
   // Same tournament finish correlations
   // Playing partner correlations
   // Country/region based correlations
   // Tee time wave correlations
   ```

5. **golf_lineup_rules.go** - Platform-specific constraints:
   ```go
   // DraftKings: 6 golfers, $50,000 salary cap
   // FanDuel: 6 golfers, $60,000 salary cap
   // Tournament cut considerations
   // Scoring system differences
   ```

6. **golf_course_analyzer.go** - Course insights integration:
   ```go
   // Course length impact on player types
   // Historical scoring averages by course
   // Weather patterns for tournament week
   // Course setup trends
   ```

## DOCUMENTATION:

### **Free Golf API Integration Guide**

1. **ESPN Golf Hidden API** (Primary Source):
   - **No authentication required**
   - Base URL: `site.api.espn.com/apis/site/v2/sports/golf/`
   - Tournaments: `/pga/scoreboard?dates=YYYYMMDD`
   - Players: `/athletes/{id}/stats`
   - Rate limits: Unofficial, implement 1-second delays
   - Coverage: PGA Tour events with historical data

2. **Free Golf API (FreeWebApi)**:
   - **Free with signup** at freewebapi.com
   - World rankings updated weekly
   - Shot-by-shot scorecards available
   - FedExCup standings and points
   - No strict rate limits documented

3. **Live Golf API**:
   - **Completely free** at livegolfapi.com
   - Real-time scoring during tournaments
   - Multiple tour coverage (PGA, DP World, LIV)
   - REST API with JSON responses
   - Updates every minute during events

4. **GolfCourseAPI**:
   - **Free with email** registration
   - 30,000+ golf courses globally
   - Course characteristics and layouts
   - Historical scoring data
   - Weather pattern integration

### **Implementation Priorities**

**Phase 1: Core Golf Data (Week 1)**
- Integrate ESPN Golf API for tournaments/players
- Create golf-specific database schema
- Build basic projection model from last 10 events
- Implement tournament status tracking

**Phase 2: Enhanced Analytics (Week 2)**
- Add Free Golf API for detailed statistics
- Integrate course data for venue analysis
- Build cut probability models
- Create golf-specific caching strategy

**Phase 3: Advanced Features (Week 3)**
- Add Live Golf API for real-time updates
- Build ownership projection models
- Implement weather adjustments
- Create golf-specific UI components

## OTHER CONSIDERATIONS:

1. **Golf-Specific Challenges**:
   - **4-day events**: Different from daily sports
   - **Cut risk**: Half the field eliminated after 2 days
   - **Weather delays**: Dynamic tournament scheduling
   - **International players**: Time zone considerations
   - **Multiple tours**: Focus on PGA Tour initially

2. **Data Quality for Golf**:
   - Cross-validate scores across ESPN and Live Golf API
   - Handle tournament status changes (delays, cancellations)
   - Track official/unofficial status of rounds
   - Monitor player withdrawals in real-time

3. **Free API Optimization**:
   - Cache tournament data aggressively (update hourly)
   - Pre-fetch player stats before tournaments
   - Use WebSocket connections where available
   - Batch player lookups to reduce API calls

4. **Golf Scoring Considerations**:
   - **DraftKings scoring**: Emphasizes birdies/eagles
   - **FanDuel scoring**: More balanced approach
   - **Placement points**: Significant for top finishes
   - **Streak bonuses**: Consecutive birdies
   - **Bogey penalties**: Negative scoring impact

5. **MVP Golf Features**:
   - Start with PGA Tour only
   - Support top 20 tournaments initially
   - 20 lineup maximum
   - Basic cut projection model
   - Manual ownership override option

6. **Future Enhancements**:
   - DP World Tour support
   - LPGA Tour integration
   - Amateur tournament data
   - Betting market integration
   - Social sentiment analysis

7. **Golf-Specific Optimizations**:
   - **Stars and scrubs**: Popular golf DFS strategy
   - **Balanced builds**: Even salary distribution
   - **Cut protection**: Mixing safe/risky plays
   - **Ownership leverage**: Fade popular players
   - **Late swap strategy**: Thursday/Friday adjustments

## EXAMPLE API RESPONSES:

### ESPN Golf Leaderboard
```json
{
  "events": [{
    "id": "401580866",
    "name": "The American Express",
    "competitions": [{
      "competitors": [{
        "id": "10404",
        "athlete": {
          "displayName": "Jon Rahm",
          "id": "10404"
        },
        "score": "-23",
        "status": {
          "position": {
            "displayName": "1"
          }
        },
        "statistics": [{
          "name": "R1",
          "displayValue": "64"
        }]
      }]
    }]
  }]
}
```

### Free Golf API Player Stats
```json
{
  "player": {
    "id": "47959",
    "name": "Scottie Scheffler",
    "country": "USA",
    "worldRank": 1,
    "fedexRank": 1,
    "stats": {
      "scoringAverage": 68.63,
      "drivingDistance": 312.5,
      "drivingAccuracy": 62.3,
      "greensInRegulation": 71.2,
      "puttingAverage": 1.731
    }
  }
}
```

### Golf Course API Response
```json
{
  "course": {
    "id": "1234",
    "name": "TPC Sawgrass",
    "location": "Ponte Vedra Beach, FL",
    "par": 72,
    "yardage": 7189,
    "characteristics": {
      "type": "Pete Dye Design",
      "grass": "Bermuda",
      "difficulty": 9.2,
      "weatherHistory": {
        "avgWindSpeed": 12.3,
        "rainProbability": 0.15
      }
    }
  }
}
```

This golf feature leverages completely free APIs to deliver professional-grade fantasy golf optimization, maintaining the zero-cost operational model while expanding the platform's sport coverage.