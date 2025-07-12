name: "Golf Tournament Data Integration - Slash Golf API"
description: |

## Purpose
Implement comprehensive golf tournament data integration using Slash Golf API (RapidAPI) to enhance the DFS Optimizer with real-time tournament schedules, player fields, leaderboards, and scoring data while intelligently managing API rate limits.

## Core Principles
1. **Cache-First Architecture**: Minimize API calls with aggressive caching (20 req/day limit)
2. **Fallback Strategy**: Seamless ESPN fallback when RapidAPI limits are reached
3. **Data Freshness**: Balance between real-time updates and API conservation
4. **Integration Pattern**: Follow existing provider patterns in the codebase
5. **Global Rules**: Adhere to all rules in CLAUDE.md

---

## Goal
Enhance the existing RapidAPI Golf provider to fetch and manage tournament schedules, provide comprehensive tournament data for the dashboard, and ensure the optimizer has access to complete player field information while working within the Basic plan's 20 requests/day limit.

## Why
- **User Value**: Display upcoming tournaments on dashboard for planning DFS entries
- **Data Completeness**: Ensure optimizer has full tournament field data
- **Rate Limit Management**: Optimize API usage to stay within Basic plan limits
- **Real-time Updates**: Provide current leaderboard data during live tournaments

## What
Implement tournament schedule fetching, enhance tournament details retrieval, and create dashboard endpoints to display tournament information with intelligent caching strategies.

### Success Criteria
- [ ] Tournament schedule endpoint returns next 4 upcoming tournaments
- [ ] Current tournament details include complete player field
- [ ] Cache strategy keeps API calls under 20/day limit
- [ ] Dashboard displays tournament data with freshness indicators
- [ ] All existing golf tests continue to pass
- [ ] Integration test validates new endpoints

## All Needed Context

### Documentation & References
```yaml
# MUST READ - Include these in your context window
- url: https://slashgolf.dev/docs.html
  why: Complete endpoint documentation for /schedule, /tournament, /leaderboard
  
- url: https://slashgolf.dev/quickstart
  why: Authentication setup and rate limiting information
  
- url: https://rapidapi.com/slashgolf/api/live-golf-data
  why: Interactive API testing and subscription plan details
  
- file: backend/internal/providers/rapidapi_golf.go
  why: Current implementation with rate limiting, caching patterns
  
- file: backend/internal/providers/espn_golf.go
  why: Fallback provider pattern to follow
  
- file: backend/internal/api/handlers/golf.go
  why: Existing API handler patterns for golf endpoints
  
- file: backend/scripts/test-golf-integration.sh
  why: Test patterns and validation requirements
  
- file: backend/internal/models/golf.go
  why: Golf data models that may need enhancement
```

### Current Codebase Structure
```bash
backend/
├── internal/
│   ├── providers/
│   │   ├── rapidapi_golf.go          # Main provider to enhance
│   │   └── espn_golf.go              # Fallback provider
│   ├── api/
│   │   └── handlers/
│   │       └── golf.go               # API endpoints
│   ├── models/
│   │   └── golf.go                   # Data models
│   └── services/
│       └── golf_projections.go       # Projection service
└── scripts/
    └── test-golf-integration.sh      # Integration tests
```

### Desired Additions
```bash
# No new files needed - enhance existing files
backend/
├── internal/
│   ├── providers/
│   │   └── rapidapi_golf.go          # Add GetTournamentSchedule()
│   └── api/
│       └── handlers/
│           └── golf.go               # Add schedule endpoint
```

### Known Gotchas & Library Quirks
```go
// CRITICAL: RapidAPI requires both headers on EVERY request
req.Header.Set("X-RapidAPI-Key", apiKey)
req.Header.Set("X-RapidAPI-Host", "live-golf-data.p.rapidapi.com")

// GOTCHA: Basic plan = 20 requests/day, 250/month total
// PATTERN: Always check cache first, then rate limit, then API
// GOTCHA: Tournament IDs are numeric strings (e.g., "475")
// CRITICAL: Player status values: "complete", "cut", "wd"
// PATTERN: Use 24h cache for tournament data, 7d for schedule
```

## Implementation Blueprint

### Data Models Enhancement
```go
// Verify if GolfTournamentScheduleItem is needed or use existing GolfTournament
type GolfTournamentScheduleItem struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    StartDate    time.Time `json:"start_date"`
    EndDate      time.Time `json:"end_date"`
    Purse        float64   `json:"purse"`
    FedexPoints  int       `json:"fedex_points"`
    CourseName   string    `json:"course_name"`
    Status       string    `json:"status"` // scheduled, in_progress, completed
}
```

### List of Tasks to Complete

```yaml
Task 1: Add Tournament Schedule Fetching to RapidAPI Provider
MODIFY backend/internal/providers/rapidapi_golf.go:
  - ADD method: GetTournamentSchedule(year int) ([]GolfTournamentData, error)
  - IMPLEMENT cache-first pattern with 7-day TTL
  - HANDLE rate limiting and ESPN fallback
  - MAP RapidAPI response to GolfTournamentData

Task 2: Add Schedule Response Structures
MODIFY backend/internal/providers/rapidapi_golf.go:
  - ADD struct: rapidAPIScheduleResponse
  - ADD struct: rapidAPIScheduleItem
  - FOLLOW existing response structure patterns

Task 3: Enhance Tournament Details Fetching
MODIFY backend/internal/providers/rapidapi_golf.go:
  - UPDATE GetCurrentTournament() to accept tournamentID parameter
  - MODIFY to fetch any tournament by ID, not just current
  - PRESERVE backward compatibility

Task 4: Add Tournament Schedule API Endpoint
MODIFY backend/internal/api/handlers/golf.go:
  - ADD endpoint: GET /api/v1/golf/tournaments/schedule
  - IMPLEMENT year parameter (default current year)
  - RETURN next 4 upcoming tournaments with cache metadata

Task 5: Update Golf Routes
MODIFY backend/internal/api/handlers/golf.go or routes file:
  - REGISTER new schedule endpoint
  - ENSURE proper route ordering

Task 6: Add Integration Tests
MODIFY backend/scripts/test-golf-integration.sh:
  - ADD test: test_tournament_schedule()
  - VERIFY schedule endpoint returns tournaments
  - CHECK cache behavior and data freshness

Task 7: Update Existing Tests
MODIFY backend/internal/providers/rapidapi_golf_test.go:
  - ADD test cases for GetTournamentSchedule
  - MOCK schedule API responses
  - TEST cache hit/miss scenarios
```

### Task 1 Pseudocode - Tournament Schedule Implementation
```go
// Task 1 - Add to RapidAPIGolfClient
func (c *RapidAPIGolfClient) GetTournamentSchedule(year int) ([]GolfTournamentData, error) {
    // PATTERN: Cache-first approach
    cacheKey := fmt.Sprintf("rapidapi:golf:schedule:%d", year)
    
    // Check cache with 7-day TTL for schedule data
    var cachedSchedule []GolfTournamentData
    if err := c.cache.GetSimple(cacheKey, &cachedSchedule); err == nil {
        c.logger.Info("Returning cached tournament schedule")
        return cachedSchedule, nil
    }
    
    // CRITICAL: Check daily rate limit
    if c.isOverDailyLimit() {
        c.logger.Warn("RapidAPI limit reached, using ESPN fallback")
        // ESPN doesn't have schedule, return error or empty
        return nil, fmt.Errorf("rate limit exceeded, schedule not available from fallback")
    }
    
    // PATTERN: Use makeRequest helper with retry logic
    url := fmt.Sprintf("%s/schedule?year=%d&orgId=1", c.baseURL, year)
    var response rapidAPIScheduleResponse
    
    if err := c.makeRequest(url, &response); err != nil {
        return nil, err
    }
    
    // Map response to our data structure
    tournaments := make([]GolfTournamentData, 0)
    for _, item := range response.Results.Schedule {
        tournament := GolfTournamentData{
            ID:         strconv.Itoa(item.TournID),
            Name:       item.Name,
            StartDate:  c.parseDate(item.StartDate),
            EndDate:    c.parseDate(item.EndDate),
            Status:     "scheduled",
            CourseName: item.Courses[0].CourseName, // First course
            Purse:      float64(item.Purse),
        }
        tournaments = append(tournaments, tournament)
    }
    
    // CRITICAL: Cache for 7 days (schedule rarely changes)
    c.cache.SetSimple(cacheKey, tournaments, 7*24*time.Hour)
    c.logger.WithField("count", len(tournaments)).Info("Cached tournament schedule")
    
    return tournaments, nil
}
```

### Task 4 Pseudocode - API Handler
```go
// Task 4 - Add to golf.go handlers
func (h *GolfHandler) GetTournamentSchedule(c *gin.Context) {
    yearStr := c.DefaultQuery("year", strconv.Itoa(time.Now().Year()))
    year, err := strconv.Atoi(yearStr)
    if err != nil {
        c.JSON(400, gin.H{"error": "invalid year parameter"})
        return
    }
    
    // Get schedule from provider
    schedule, err := h.golfProvider.GetTournamentSchedule(year)
    if err != nil {
        h.logger.WithError(err).Error("Failed to fetch tournament schedule")
        c.JSON(500, gin.H{"error": "failed to fetch schedule"})
        return
    }
    
    // Filter to next 4 upcoming tournaments
    now := time.Now()
    upcoming := make([]GolfTournamentData, 0, 4)
    for _, tournament := range schedule {
        if tournament.StartDate.After(now) && len(upcoming) < 4 {
            upcoming = append(upcoming, tournament)
        }
    }
    
    // Include cache metadata
    c.JSON(200, gin.H{
        "tournaments": upcoming,
        "cached_at": time.Now().Add(-6 * time.Hour), // Estimate based on cache strategy
        "next_update": time.Now().Add(18 * time.Hour),
    })
}
```

### Integration Points
```yaml
DATABASE:
  - No migration needed - using existing golf_tournaments table
  
ROUTES:
  - ADD to: golf router in RegisterRoutes()
  - pattern: "golf.GET("/tournaments/schedule", h.GetTournamentSchedule)"
  
CACHE:
  - Keys: "rapidapi:golf:schedule:{year}"
  - TTL: 7 days for schedule, 24 hours for tournament details
  
PROVIDER:
  - Interface method: GetTournamentSchedule(year int) ([]GolfTournamentData, error)
```

## Validation Loop

### Level 1: Syntax & Style
```bash
# Run from backend directory
cd backend

# Check Go syntax and formatting
go fmt ./internal/providers/rapidapi_golf.go
go fmt ./internal/api/handlers/golf.go

# Run linter if available
golangci-lint run ./internal/providers/... ./internal/api/...

# Expected: No errors or warnings
```

### Level 2: Unit Tests
```bash
# Run provider tests
go test ./internal/providers -run TestRapidAPIGolf -v

# Run handler tests  
go test ./internal/api/handlers -run TestGolf -v

# Expected: All tests pass, including new schedule tests
```

### Level 3: Integration Test
```bash
# Start backend server
cd backend
go run cmd/server/main.go

# In another terminal, run integration tests
cd backend/scripts
./test-golf-integration.sh

# Test new endpoint manually
curl -X GET "http://localhost:8080/api/v1/golf/tournaments/schedule?year=2025"

# Expected response:
# {
#   "tournaments": [...],
#   "cached_at": "2025-01-01T12:00:00Z",
#   "next_update": "2025-01-02T06:00:00Z"
# }
```

## Final Validation Checklist
- [ ] All existing golf tests pass: `go test ./... -tags=integration`
- [ ] No linting errors: `golangci-lint run`
- [ ] Schedule endpoint returns data: `curl localhost:8080/api/v1/golf/tournaments/schedule`
- [ ] Rate limiting prevents > 20 API calls/day
- [ ] Cache TTLs are appropriate (7d schedule, 24h tournament)
- [ ] ESPN fallback works when rate limit exceeded
- [ ] Integration test script passes all checks
- [ ] API responses include cache metadata

---

## Anti-Patterns to Avoid
- ❌ Don't fetch individual tournament details in a loop
- ❌ Don't ignore cache on startup - always check first
- ❌ Don't make unnecessary API calls - batch when possible
- ❌ Don't hardcode year values - use parameters
- ❌ Don't skip rate limit checks - enforce strictly
- ❌ Don't modify existing method signatures - maintain compatibility

## Confidence Score: 9/10

The implementation path is clear with existing patterns to follow. The main complexity is managing the rate limits effectively, but the current caching infrastructure and patterns make this straightforward. The only uncertainty is around potential edge cases in tournament data mapping.