name: "DFS Lineup Optimizer - Critical Features Implementation"
description: |

## Purpose
Fix critical infrastructure issues and implement essential features for the DFS Lineup Optimizer, including API routing fix, external data integration from free sports APIs, drag-and-drop UI, and real-time WebSocket functionality.

## Core Principles
1. **Context is King**: Include ALL necessary documentation, examples, and caveats
2. **Validation Loops**: Provide executable tests/lints the AI can run and fix
3. **Information Dense**: Use keywords and patterns from the codebase
4. **Progressive Success**: Start simple, validate, then enhance
5. **Global rules**: Be sure to follow all rules in CLAUDE.md

---

## Goal
Transform the DFS Lineup Optimizer from a mock-data prototype to a production-ready application with real sports data integration, functional UI features, and real-time optimization progress updates.

## Why
- **Business value**: Enable actual DFS lineup optimization with real player data and projections
- **Integration**: Connect to ESPN, TheSportsDB, and BALLDONTLIE APIs for comprehensive sports data
- **Problems solved**: Users can create optimized lineups with real data, drag-and-drop players, and see real-time progress
- **Current blockers**: API routing prevents frontend-backend communication; no real data sources

## What
A fully functional DFS optimizer where:
- Frontend successfully communicates with backend via fixed `/api/v1` routes
- Real player data flows from ESPN Hidden API, TheSportsDB, and BALLDONTLIE
- Users can drag-and-drop players into lineup positions
- WebSocket provides real-time optimization progress
- Data updates automatically on schedule

### Success Criteria
- [ ] Frontend can call backend API endpoints at `/api/v1/*`
- [ ] Real player data populates from external APIs
- [ ] Drag-and-drop lineup builder works smoothly
- [ ] WebSocket shows optimization progress in real-time
- [ ] All tests pass and code meets quality standards

## All Needed Context

### Documentation & References
```yaml
# MUST READ - Include these in your context window
- url: https://gist.github.com/akeaswaran/b48b02f1c94f873c6655e7129910fc3b
  why: ESPN Hidden API endpoints documentation - critical for player stats
  
- url: https://gist.github.com/nntrn/ee26cb2a0716de0947a0a4e9a157bc1c
  why: Complete list of NFL API endpoints from ESPN
  
- url: https://www.thesportsdb.com/documentation
  why: TheSportsDB API for team/player images and metadata
  
- url: https://docs.balldontlie.io/
  why: BALLDONTLIE NBA API for real-time game data and stats
  
- url: https://dndkit.com/docs/introduction
  why: DnD Kit documentation for implementing drag-and-drop
  
- file: backend/internal/api/router.go
  why: Current routing issue - routes not prefixed with /api/v1
  
- file: backend/cmd/server/main.go
  why: Shows how router is mounted at /api/v1 (line 79)
  
- file: frontend/src/services/api.ts
  why: Frontend expects /api/v1 prefix for all API calls
  
- file: backend/internal/api/handlers/player.go
  why: Shows mock data implementation to replace (lines 120-135, 189-211)
  
- file: frontend/src/components/LineupBuilder/index.tsx
  why: Current UI without drag-and-drop functionality
```

### Current Codebase tree (relevant parts)
```bash
backend/
├── cmd/
│   └── server/main.go                 # API server with routing issue
├── internal/
│   ├── api/
│   │   ├── router.go                  # Routes without /api/v1 prefix
│   │   └── handlers/
│   │       └── player.go              # Mock data to replace
│   ├── models/
│   │   └── player.go                  # Player model structure
│   └── services/
│       ├── cache.go                   # Redis caching
│       └── websocket.go               # WebSocket hub ready
frontend/
├── src/
│   ├── components/
│   │   └── LineupBuilder/
│   │       └── index.tsx              # Needs drag-and-drop
│   └── services/
│       └── api.ts                     # Expects /api/v1 prefix
```

### Desired Codebase tree with files to be added
```bash
backend/
├── internal/
│   ├── api/
│   │   └── router.go                  # FIXED: Proper route grouping
│   ├── providers/                     # NEW: External API clients
│   │   ├── espn.go                    # ESPN Hidden API client
│   │   ├── thesportsdb.go            # TheSportsDB client
│   │   ├── balldontlie.go            # BALLDONTLIE client
│   │   └── aggregator.go              # Data aggregation service
│   └── services/
│       └── data_fetcher.go            # NEW: Scheduled data updates
frontend/
├── src/
│   ├── components/
│   │   ├── LineupBuilder/
│   │   │   └── index.tsx              # ENHANCED: With drag-and-drop
│   │   └── SimulationProgress/        # NEW: Real-time progress
│   │       └── index.tsx
```

### Known Gotchas & Library Quirks
```go
// CRITICAL: Gin router grouping issue
// Current code uses gin.WrapH which causes routing problems
// Must use proper Gin route grouping for /api/v1 prefix

// CRITICAL: ESPN API has no official docs
// Rate limits unknown - implement exponential backoff
// Data shape varies between endpoints

// CRITICAL: DnD Kit with TypeScript
// Must use proper type definitions for drag events
// Requires context providers wrapping components

// CRITICAL: WebSocket authentication
// JWT token must be passed in query string for WS
// Cannot use Authorization header in WebSocket
```

## Implementation Blueprint

### Data models and structure

```go
// backend/internal/providers/types.go
type PlayerData struct {
    ExternalID   string                 `json:"external_id"`
    Name         string                 `json:"name"`
    Team         string                 `json:"team"`
    Position     string                 `json:"position"`
    Stats        map[string]float64     `json:"stats"`
    LastUpdated  time.Time             `json:"last_updated"`
    Source       string                 `json:"source"` // "espn", "thesportsdb", "balldontlie"
}

type AggregatedPlayer struct {
    Player              models.Player
    ESPNData           *PlayerData
    TheSportsDBData    *PlayerData
    BallDontLieData    *PlayerData
    ProjectedPoints    float64
    Confidence         float64  // Based on data availability
}
```

### List of tasks to be completed in order

```yaml
Task 1: Fix API Routing (CRITICAL - Blocks everything else)
MODIFY backend/internal/api/router.go:
  - Change router setup to use groups properly
  - Ensure all routes have relative paths (no /api/v1 prefix)
  - Return *gin.Engine instead of http.Handler

MODIFY backend/cmd/server/main.go:
  - Remove gin.WrapH usage
  - Mount API routes properly using RouterGroup
  - Test with: curl http://localhost:8080/api/v1/health

Task 2: Implement ESPN API Client
CREATE backend/internal/providers/espn.go:
  - MIRROR pattern from: backend/internal/services/cache.go (for service structure)
  - Implement methods: GetNBAPlayers, GetNFLPlayers, GetMLBPlayers
  - Add exponential backoff for rate limiting
  - Cache responses for 15 minutes

Task 3: Implement TheSportsDB Client
CREATE backend/internal/providers/thesportsdb.go:
  - Use free API endpoints (no key required initially)
  - Focus on player images and team metadata
  - Implement search by player name for image matching

Task 4: Implement BALLDONTLIE Client  
CREATE backend/internal/providers/balldontlie.go:
  - Register for free API key
  - Implement GetNBAStats with 60 req/min rate limit
  - Focus on current season stats

Task 5: Create Data Aggregator Service
CREATE backend/internal/providers/aggregator.go:
  - Combine data from all three sources
  - Calculate projections based on recent performance
  - Handle missing data gracefully
  - Update player records in database

Task 6: Create Scheduled Data Fetcher
CREATE backend/internal/services/data_fetcher.go:
  - Use cron-like scheduling
  - Fetch data every 2 hours during season
  - Update cache after successful fetch
  - Log all API calls for monitoring

Task 7: Update Player Handler
MODIFY backend/internal/api/handlers/player.go:
  - Remove mock data (lines 120-135, 189-211)
  - Inject data aggregator service
  - Return real stats and news from providers

Task 8: Implement Drag-and-Drop UI
MODIFY frontend/src/components/LineupBuilder/index.tsx:
  - Import DndContext, DragOverlay from @dnd-kit/core
  - Import SortableContext from @dnd-kit/sortable  
  - Wrap component in DndContext
  - Make player cards draggable
  - Make position slots droppable
  - Handle drag end events

Task 9: Create WebSocket Progress Component
CREATE frontend/src/components/SimulationProgress/index.tsx:
  - Connect to WebSocket using connectWebSocket from api.ts
  - Display progress bar for optimization
  - Show intermediate results as they arrive
  - Handle connection errors gracefully

Task 10: Integration Testing
  - Test full flow: fetch real data → optimize → export
  - Verify drag-and-drop works across browsers
  - Ensure WebSocket reconnects on disconnect
```

### Per task pseudocode

```go
// Task 1: Fix API Routing
// backend/internal/api/router.go
func NewRouter(db *database.DB, cache *services.CacheService, wsHub *services.WebSocketHub, cfg *config.Config) *gin.Engine {
    router := gin.New()
    router.Use(gin.Recovery())
    router.Use(middleware.Logger())
    router.Use(middleware.CORS(cfg.CorsOrigins))
    
    // API v1 routes - NO prefix here, will be added in main.go
    // Group all routes together
    playerHandler := handlers.NewPlayerHandler(db, cache)
    
    // Public routes
    router.GET("/contests", contestHandler.ListContests)
    router.GET("/contests/:id", contestHandler.GetContest)
    // ... rest of routes without /api/v1 prefix
    
    return router
}

// Task 2: ESPN API Client
// backend/internal/providers/espn.go
type ESPNClient struct {
    httpClient *http.Client
    cache      *services.CacheService
    logger     *logrus.Logger
}

func (c *ESPNClient) GetNBAPlayers(date string) ([]PlayerData, error) {
    // PATTERN: Check cache first (see handlers/player.go:40-44)
    cacheKey := fmt.Sprintf("espn:nba:players:%s", date)
    
    // GOTCHA: ESPN API has no auth but unknown rate limits
    url := fmt.Sprintf("https://site.api.espn.com/apis/site/v2/sports/basketball/nba/scoreboard?dates=%s", date)
    
    // CRITICAL: Implement retry with exponential backoff
    var resp *http.Response
    for attempt := 0; attempt < 3; attempt++ {
        resp, err = c.httpClient.Get(url)
        if err == nil && resp.StatusCode == 200 {
            break
        }
        time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
    }
    
    // Parse response and extract player data
    // Transform to our PlayerData structure
    // Cache for 15 minutes
}

// Task 8: Drag-and-Drop Implementation
// frontend/src/components/LineupBuilder/index.tsx
import { DndContext, DragEndEvent, DragOverlay } from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'

function LineupBuilder() {
    const [activeId, setActiveId] = useState<string | null>(null)
    
    const handleDragEnd = (event: DragEndEvent) => {
        const { active, over } = event
        
        // PATTERN: Check if dropped on valid position
        if (over && active.id !== over.id) {
            // Update lineup based on drag result
            const draggedPlayer = allPlayers.find(p => p.id === active.id)
            const targetPosition = over.id // position slot ID
            
            // Validate position eligibility
            if (isValidPosition(draggedPlayer, targetPosition)) {
                onLineupChange(/* updated lineup */)
            }
        }
    }
    
    return (
        <DndContext onDragEnd={handleDragEnd}>
            <SortableContext items={positionSlots} strategy={verticalListSortingStrategy}>
                {/* Render position slots as droppable */}
            </SortableContext>
            <DragOverlay>
                {activeId ? <PlayerCard player={...} /> : null}
            </DragOverlay>
        </DndContext>
    )
}
```

### Integration Points
```yaml
DATABASE:
  - migration: "Add columns to players: espn_id, sportsdb_id, balldontlie_id, last_fetched"
  - index: "CREATE INDEX idx_players_external_ids ON players(espn_id, sportsdb_id)"
  
CONFIG:
  - add to: backend/pkg/config/config.go
  - pattern: |
      ESPNRateLimit      int    `envconfig:"ESPN_RATE_LIMIT" default:"10"`
      BallDontLieAPIKey  string `envconfig:"BALLDONTLIE_API_KEY"`
      DataFetchInterval  string `envconfig:"DATA_FETCH_INTERVAL" default:"2h"`
  
ROUTES:
  - fix in: backend/internal/api/router.go
  - remove all "/api/v1" prefixes from route definitions
  
MAIN:
  - fix in: backend/cmd/server/main.go
  - pattern: |
      apiV1 := router.Group("/api/v1")
      apiRoutes := api.NewRouter(db, cacheService, webSocketHub, cfg)
      apiV1.Any("/*path", gin.WrapH(apiRoutes))
```

## Validation Loop

### Level 1: Backend Syntax & Build
```bash
# Run these FIRST - fix any errors before proceeding
cd backend
go mod tidy
go fmt ./...
golangci-lint run ./...

# Build should succeed
go build -o bin/server cmd/server/main.go

# Expected: No errors. If errors, READ the error and fix.
```

### Level 2: API Routing Test
```bash
# Start the backend server
cd backend
go run cmd/server/main.go

# Test the API routes are accessible
curl -v http://localhost:8080/api/v1/health
# Expected: {"status":"ok","time":"..."}

curl -v http://localhost:8080/api/v1/contests  
# Expected: {"data":[...],"success":true}

# If 404, the routing fix didn't work - debug the router setup
```

### Level 3: Frontend Build & Type Check
```bash
cd frontend
npm install
npm run type-check
npm run lint

# Expected: No errors
```

### Level 4: Integration Test
```bash
# Start backend (if not running)
cd backend && go run cmd/server/main.go &

# Start frontend
cd frontend && npm run dev &

# Open browser to http://localhost:5173
# Click through to Optimizer page
# Should see player data loading (even if mock initially)
# Open Network tab - API calls should succeed (200 status)
```

### Level 5: External API Test
```bash
# Test ESPN API directly
curl "https://site.api.espn.com/apis/site/v2/sports/basketball/nba/scoreboard"
# Should return current NBA games

# Test TheSportsDB (no auth needed)
curl "https://www.thesportsdb.com/api/v1/json/3/searchplayers.php?p=Lebron%20James"
# Should return player data

# After implementation, test our aggregated endpoint
curl http://localhost:8080/api/v1/contests/1/players
# Should return real player data, not mock
```

### Level 6: Drag-and-Drop Test
```
Manual Test Steps:
1. Navigate to Optimizer page
2. Select a contest
3. Try dragging a player from the pool
4. Drop on a valid position slot
5. Verify player appears in lineup
6. Try invalid position (e.g., QB in RB slot)
7. Verify rejection with visual feedback
8. Remove player with X button
9. Verify salary updates correctly
```

### Level 7: WebSocket Test
```bash
# Use wscat or similar tool
npm install -g wscat
wscat -c ws://localhost:8080/ws

# In another terminal, trigger optimization
curl -X POST http://localhost:8080/api/v1/optimize \
  -H "Content-Type: application/json" \
  -d '{"contest_id":1,"num_lineups":5}'

# WebSocket should receive progress messages
```

## Final Validation Checklist
- [ ] API routes work at `/api/v1/*` endpoints
- [ ] Real player data loads from external APIs  
- [ ] Drag-and-drop lineup building works smoothly
- [ ] WebSocket shows optimization progress
- [ ] No Go linting errors: `golangci-lint run`
- [ ] No TypeScript errors: `npm run type-check`
- [ ] Manual test of full workflow succeeds
- [ ] Data updates on schedule (check logs)
- [ ] Error cases handled gracefully
- [ ] Performance acceptable (< 2s page loads)

---

## Anti-Patterns to Avoid
- ❌ Don't use gin.WrapH for route groups - causes routing issues
- ❌ Don't skip rate limiting on external APIs - will get blocked
- ❌ Don't store API keys in code - use environment variables  
- ❌ Don't ignore TypeScript errors in drag-and-drop - causes runtime issues
- ❌ Don't fetch all player data on every request - use caching
- ❌ Don't assume external API availability - implement fallbacks

## Phase 2 Considerations (Future)
After these critical features are working:
- Implement more sophisticated projection algorithms
- Add historical performance tracking
- Create ownership projection models  
- Add more sports beyond NBA/NFL/MLB
- Implement user accounts and saved lineups
- Add tournament simulation features

## Confidence Score: 8/10
This PRP provides comprehensive context for fixing the critical routing issue, integrating real data sources, and implementing the missing UI features. The validation gates ensure each component works before moving to the next. The main risk is external API reliability, which is mitigated by caching and fallback strategies.