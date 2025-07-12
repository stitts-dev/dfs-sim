## FEATURE:

Implement comprehensive golf tournament data integration using Slash Golf API (RapidAPI) for the DFS Optimizer. This includes properly fetching tournament schedules, player fields, leaderboards, and scoring data with intelligent caching to work within API rate limits. The integration should provide real-time tournament data for both the dashboard display and the optimizer functionality.

## EXAMPLES:

### Example 1: Tournament Schedule Request
```bash
# Get current year tournament schedule
curl -X GET "https://live-golf-data.p.rapidapi.com/schedule?year=2025&orgId=1" \
  -H "X-RapidAPI-Key: YOUR_API_KEY" \
  -H "X-RapidAPI-Host: live-golf-data.p.rapidapi.com"
```

### Example 2: Tournament Details Response Structure
```json
{
  "tournId": "475",
  "name": "Valspar Championship",
  "year": 2024,
  "purse": 8400000,
  "fedexCupPoints": 500,
  "startDate": "2024-03-21",
  "endDate": "2024-03-24",
  "courses": [{
    "courseId": "665",
    "courseName": "Innisbrook Resort - Copperhead Course",
    "par": 71,
    "holes": [4,5,3,4,4,5,3,4,4,3,4,4,4,3,5,4,4,4]
  }],
  "players": [{
    "playerId": "28237",
    "firstName": "Jordan",
    "lastName": "Spieth",
    "status": "complete",
    "isAmateur": false
  }],
  "currentRound": 4,
  "timezone": "America/New_York"
}
```

### Example 3: Caching Strategy Implementation
```go
// Cache with appropriate TTLs for Basic plan (20 req/day)
cacheKey := fmt.Sprintf("rapidapi:golf:tournament:%s:%d", tournId, year)
cacheTTL := 24 * time.Hour // Tournament data rarely changes

// Check cache first
var tournament TournamentData
if err := cache.Get(cacheKey, &tournament); err == nil {
    return &tournament, nil
}

// Only make API call if cache miss
tournament, err := fetchTournamentFromAPI(tournId, year)
if err == nil {
    cache.Set(cacheKey, tournament, cacheTTL)
}
```

## DOCUMENTATION:

### Slash Golf API Documentation
1. **API Reference**: https://slashgolf.dev/docs.html
   - Complete endpoint documentation
   - Response structure definitions
   - Parameter specifications

2. **Quick Start Guide**: https://slashgolf.dev/quickstart
   - Authentication setup
   - Basic usage examples
   - Rate limiting information

3. **RapidAPI Portal**: https://rapidapi.com/slashgolf/api/live-golf-data
   - Interactive API testing
   - Subscription plan details
   - Request/response examples

### Key Endpoints to Implement
- `/schedule` - Get tournament schedule for a year
- `/tournament` - Get detailed tournament information
- `/leaderboard` - Get live leaderboard data
- `/players` - Get player information (use sparingly)
- `/scorecard` - Get hole-by-hole scores
- `/points` - Get FedEx Cup points
- `/earnings` - Get prize money distribution

### Tournament Response Structure Analysis
Based on the Valspar Championship example, the API provides:
- **Tournament Metadata**: ID, name, year, purse, FedEx Cup points
- **Course Information**: Detailed hole-by-hole par values
- **Player Field**: Complete list with status (complete/cut/wd)
- **Timing Data**: Current round, timezone, dates

## OTHER CONSIDERATIONS:

### API Rate Limiting Strategy
- **Basic Plan**: 20 requests/day, 250/month total
- **Priority Order**: 
  1. Tournament schedule (1/week)
  2. Current tournament details (1/day)
  3. Leaderboard updates (2-3/day during tournaments)
- **Cache Everything**: Use Redis with extended TTLs
- **Fallback**: Use ESPN Golf when RapidAPI limit reached

### Critical Implementation Details
1. **Authentication Headers** (REQUIRED for every request):
   ```go
   req.Header.Set("X-RapidAPI-Key", apiKey)
   req.Header.Set("X-RapidAPI-Host", "live-golf-data.p.rapidapi.com")
   ```

2. **Tournament ID Mapping**:
   - Tournament IDs are numeric (e.g., "475" for Valspar)
   - Must track tournament IDs for schedule → details mapping
   - Store mapping in database for quick lookup

3. **Player Status Handling**:
   - "complete" = finished all rounds
   - "cut" = missed cut after round 2
   - "wd" = withdrew from tournament
   - Filter out cut/wd players for optimizer

4. **Data Freshness Requirements**:
   - Schedule: Update weekly (tournaments don't change often)
   - Tournament details: Update daily during tournament week
   - Leaderboard: Update 2-3x daily during live play (Thurs-Sun)
   - Player stats: Cache for entire tournament

5. **Error Handling**:
   - 403: Invalid API key
   - 429: Rate limit exceeded (check headers)
   - 404: Tournament not found
   - Always return cached data on API errors

6. **Dashboard Display Requirements**:
   - Show next 4 upcoming tournaments
   - Display current tournament if active
   - Include: Name, dates, purse, field size, last updated
   - Visual indicator for data freshness

7. **Optimizer Integration**:
   - Only show tournaments with complete player data
   - Map player IDs to DFS platform IDs
   - Calculate fantasy projections from stats
   - Handle late withdrawals before lineups lock

### Common Pitfalls to Avoid
- Don't fetch individual players - get all from leaderboard
- Don't poll during off-hours (tournaments run Thurs-Sun)
- Don't ignore cache on startup - check first
- Don't make duplicate requests - batch when possible
- Don't trust player status - always verify before optimizer

### Testing Considerations
- Mock API responses for unit tests
- Use test tournament IDs (past tournaments)
- Implement request counting for limit tracking
- Test cache fallback scenarios
- Verify timezone handling for tee times