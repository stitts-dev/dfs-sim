# PRP: RapidAPI Live Golf Data Integration

## Overview
Replace the mock ESPN Golf provider with RapidAPI's Live Golf Data API to provide real-time PGA Tour and LIV Tour data including tournaments, leaderboards, player stats, and scorecards. This integration will follow existing provider patterns while adding comprehensive caching and robust error handling.

## Objectives
- Create `RapidAPIGolfClient` implementing the `dfs.Provider` interface
- Integrate all available golf endpoints (tournaments, leaderboards, players, stats, etc.)
- Implement Redis caching with appropriate TTLs
- Add proper rate limiting and error handling
- Map RapidAPI data to existing DFS optimizer models
- Create comprehensive test coverage

## Implementation Blueprint

### Provider Structure (Pseudocode)
```go
// backend/internal/providers/rapidapi_golf.go
type RapidAPIGolfClient struct {
    httpClient      *http.Client      // With 30s timeout
    cache           dfs.CacheProvider // Redis cache
    logger          *logrus.Logger    
    apiKey          string            // From env: RAPIDAPI_KEY
    apiHost         string            // "live-golf-data.p.rapidapi.com"
    baseURL         string            // "https://live-golf-data.p.rapidapi.com"
    rateLimiter     *time.Ticker      // 1 request per 3 seconds (safe for 20/day)
    retryAttempts   int               // Max 3 with exponential backoff
    requestTracker  *RateLimitTracker // Track daily/monthly usage
    espnFallback    *ESPNGolfClient   // Fallback when limit reached
}

// Implement Provider interface methods:
// - GetPlayers(sport, date) -> CACHE FIRST, then /leaderboard (not /players - save requests)
// - GetPlayer(sport, externalID) -> Return from cached leaderboard data
// - GetTeamRoster(sport, teamID) -> not applicable for golf

// Additional golf-specific methods (prioritized by importance):
// - GetCurrentTournament() -> /tournament (1 req/day max)
// - GetLeaderboard(tournamentID) -> /leaderboard (2-3 req/day during tournament)
// - GetTournamentSchedule() -> /schedule (1 req/week)
// - GetPlayerStats(playerID, year) -> ONLY if cached data expired
// - GetScorecard(tournamentID, playerID) -> ONLY for specific user requests
// - GetPoints(tournamentID) -> Cache for entire tournament
// - GetEarnings(tournamentID) -> Cache for entire tournament
```

### Cache Strategy (Optimized for Basic Plan)
```
Key Format: "rapidapi:golf:{entity}:{identifier}"

EXTENDED TTLs for Basic Plan (20 requests/day):
- Tournament data: 24 hours (1 request/day max)
- Player stats: 7 days (rarely changing data)
- Leaderboard: 2 hours (active), 24 hours (completed)
- Schedule: 7 days (changes infrequently)
- Organizations: 30 days (static data)
- Points/Earnings: 6 hours (active), 24 hours (completed)
- Scorecard: 1 hour (active), 7 days (completed)

Cache-First Strategy:
1. ALWAYS check cache before API call
2. Return stale data with warning if daily limit reached
3. Implement cache warming during off-peak hours
4. Use ESPN Golf as fallback when RapidAPI limit reached
```

### Error Handling Pattern
```go
// Implement exponential backoff with max 3 attempts
// Handle specific status codes:
// - 403: Invalid API credentials
// - 429: Rate limit exceeded (check headers)
// - 500+: Server errors (retry with backoff)
// Log errors but return cached data when available
```

## Implementation Tasks

### 1. Environment Configuration
- Add `RAPIDAPI_KEY` to `.env` and `.env.example`
- Update `backend/pkg/config/config.go` to load RapidAPI key
- Document the new environment variable in CLAUDE.md

### 2. Create RapidAPI Golf Provider
- Create `backend/internal/providers/rapidapi_golf.go`
- Follow pattern from `backend/internal/providers/espn_golf.go:26-36` for client initialization
- Implement RapidAPI authentication headers as shown in examples
- Add strict rate limiting for Basic plan:
  - Initialize with 20 daily request budget
  - Track each request against daily/monthly limits
  - Return cached data or ESPN fallback when limit reached
  - Log warnings when approaching limits (15/20 requests)

### 3. Implement Provider Interface Methods
- `GetPlayers()`: Fetch from leaderboard endpoint, map to `dfs.PlayerData`
- `GetPlayer()`: Direct player lookup via `/players` endpoint
- `GetTeamRoster()`: Return error - not applicable for golf

### 4. Add Golf-Specific Methods
- Tournament data retrieval (schedule, details)
- Live leaderboard with real-time updates
- Player statistics and historical data
- Hole-by-hole scorecards
- FedEx Cup points and rankings
- Prize money distribution

### 5. Data Mapping Implementation
- Map RapidAPI player IDs to DFS platform IDs
- Convert tournament IDs for contest creation
- Map stat IDs to meaningful categories
- Handle organization IDs (1 = PGA Tour, etc.)

### 6. Cache Implementation
- Use patterns from `backend/internal/services/cache.go`
- Implement cache key generators for each entity type
- Add appropriate TTLs based on data volatility
- Ensure cache failures don't break the flow

### 7. Testing
- Unit tests for provider methods (mock HTTP responses)
- Integration tests with test API key
- Cache behavior tests (TTL expiration, fallback)
- Rate limit handling tests
- Error scenario tests (API down, invalid responses)

### 8. Frontend Caching
- Update `frontend/src/services/api.ts` to use React Query
- Implement stale-while-revalidate for golf data
- Add loading states for real-time updates

## Key Context & References

### RapidAPI Authentication (CRITICAL)
```go
// Required headers for EVERY request:
req.Header.Set("X-RapidAPI-Key", r.apiKey)  // From env: RAPIDAPI_KEY
req.Header.Set("X-RapidAPI-Host", "live-golf-data.p.rapidapi.com")
```

### Available Endpoints (from INITIAL_EXAMPLE_GOLF.md)
- `/tournament` - Tournament details
- `/players` - Player information  
- `/leaderboard` - Live leaderboard data
- `/schedule` - Tournament schedule
- `/stats` - Player statistics
- `/points` - FedEx Cup points
- `/earnings` - Prize money
- `/organizations` - Golf tours
- `/scorecard` - Hole-by-hole scores

### Rate Limit Handling

**Current Plan: Basic (Free)**
- 20 requests/day total (Hard limit - no overages)
- 250 requests/month total (Hard limit)
- Rate limit: 60 requests per minute

**CRITICAL: Basic Plan Limitations**
```go
// With only 20 requests/day, aggressive caching is ESSENTIAL
// Priority order for API calls:
// 1. Current tournament info (once per day)
// 2. Active leaderboard (2-3 times during tournament)
// 3. Player data (only when absolutely needed)

// Check response headers:
limit := resp.Header.Get("x-ratelimit-requests-limit")
remaining := resp.Header.Get("x-ratelimit-requests-remaining")
reset := resp.Header.Get("x-ratelimit-requests-reset")

// Implement daily request counter
type RateLimitTracker struct {
    dailyCount    int
    monthlyCount  int
    lastReset     time.Time
}

// Basic plan: 20/day, Pro: 2,000/day, Ultra: 7,500/day, Mega: Unlimited
```

**Future Scaling Plans:**
- Pro ($20/mo): 2,000 requests/day for regular tournament updates
- Ultra ($50/mo): 7,500 requests/day for real-time during tournaments
- Mega ($100/mo): Unlimited for production with multiple tournaments

### Existing Patterns to Follow
1. **Provider Registration**: Add to `backend/internal/services/data_fetcher.go`
2. **Error Handling**: Use `backend/pkg/utils/response.go` patterns
3. **Logging**: Use structured logging with logrus
4. **HTTP Client**: 30-second timeout standard
5. **Cache Keys**: Format `provider:sport:entity:id`

### Common Gotchas
- RapidAPI requires HTTPS for all requests
- Basic plan has HARD limits - no overages allowed
- Rate limits reset daily at midnight UTC
- With only 20 req/day, prioritize tournament/leaderboard over individual player data
- Player IDs differ between RapidAPI and DFS platforms
- Tournament status affects cache TTL
- Leaderboard updates in real-time but we can only check 2-3 times/day
- Must implement request counting to avoid hitting limits

### Testing Approach
Follow pattern from `backend/tests/golf_integration_test.go`:
- Setup test environment with test database
- Mock external API calls for unit tests
- Use test API key for integration tests
- Cleanup test data after runs

## Validation Gates

```bash
# Backend validation
cd backend

# 1. Ensure environment variable is set
grep RAPIDAPI_KEY .env || echo "ERROR: RAPIDAPI_KEY not set"

# 2. Run linter
golangci-lint run

# 3. Run unit tests
go test ./internal/providers/... -v

# 4. Run integration tests (requires test DB)
go test ./tests/... -run TestRapidAPIGolf -v

# 5. Test specific provider methods
go test -run TestRapidAPIGolfClient ./internal/providers/ -v

# 6. Verify cache behavior
go test -run TestRapidAPICache ./internal/services/ -v

# Frontend validation
cd ../frontend

# 7. Type checking
npm run type-check

# 8. Linting
npm run lint

# 9. Test golf components
npm test -- --testNamePattern="Golf"
```

## Success Criteria
- All Provider interface methods implemented
- Stays within 20 requests/day limit on Basic plan
- Proper caching reduces API calls by 95%+ (critical for Basic plan)
- Seamless fallback to ESPN Golf when limit reached
- Request tracking with daily/monthly counters
- Tests achieve 80%+ coverage
- No hardcoded API keys in code
- Clear upgrade path documentation for Pro/Ultra plans

## Additional Resources
- RapidAPI Live Golf Data: https://rapidapi.com/slashgolf/api/live-golf-data
- RapidAPI Auth Docs: https://docs.rapidapi.com/docs/configuring-api-authentication
- Go HTTP Best Practices: https://www.digitalocean.com/community/tutorials/how-to-make-http-requests-in-go
- Existing ESPN Golf Provider: `backend/internal/providers/espn_golf.go`

## Basic Plan Implementation Strategy

Given the severe limitations of 20 requests/day:

1. **Morning Cache Warm (2-3 requests)**
   - Fetch current tournament info
   - Get initial leaderboard
   - Cache for entire day

2. **Afternoon Update (2-3 requests)**
   - Update leaderboard if tournament active
   - Only during tournament days

3. **User-Triggered (remaining requests)**
   - Specific player lookups
   - Scorecard requests
   - But always check cache first!

4. **ESPN Fallback**
   - When daily limit reached
   - For non-critical data
   - Log when fallback is used

## PRP Confidence Score: 9/10

The PRP provides comprehensive context for successful one-pass implementation with Basic plan constraints. The implementation prioritizes aggressive caching and intelligent fallback strategies to work within the 20 request/day limit.