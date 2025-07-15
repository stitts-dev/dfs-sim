name: "DFS Lineup Optimizer - Full Stack Implementation"
description: |

## Purpose
Complete implementation of a DFS (Daily Fantasy Sports) lineup optimizer with Go backend and React frontend, featuring Monte Carlo simulations, optimization algorithms with correlation/stacking, and multi-sport support.

## Core Principles
1. **Context is King**: Include ALL necessary documentation, examples, and caveats
2. **Validation Loops**: Provide executable tests/lints the AI can run and fix
3. **Information Dense**: Use keywords and patterns from the codebase
4. **Progressive Success**: Start simple, validate, then enhance
5. **Global rules**: Be sure to follow all rules in CLAUDE.md

---

## Goal
Build a full-stack DFS lineup optimizer that replicates SaberSim's core functionality:
- Monte Carlo game simulations for outcome predictions
- Lineup optimization with salary cap, position constraints, and stacking rules
- Real-time updates via WebSocket
- Export functionality for DraftKings/FanDuel platforms
- MVP: NBA only, then expand to NFL, MLB, NHL

## Why
- **Business Value**: Professional DFS players need sophisticated tools for lineup optimization
- **User Impact**: Improve win rates through algorithmic lineup generation
- **Market Need**: SaberSim alternative with modern tech stack and open architecture
- **Problems Solved**: Manual lineup creation is time-consuming and suboptimal

## What
### User-Visible Behavior
- Dashboard for sport/contest selection
- Player pool with stats, projections, and salaries
- Drag-and-drop lineup builder with real-time validation
- Optimizer with correlation slider and stacking options
- Simulation visualizer showing lineup performance distributions
- Export lineups as CSV for platform upload

### Technical Requirements
- Go backend with Gin framework for high-performance API
- React + TypeScript frontend with TailwindCSS
- PostgreSQL database with GORM ORM
- Redis caching for optimization results
- WebSocket for real-time updates
- Docker containerization

### Success Criteria
- [ ] Generate valid lineups under salary cap with position constraints
- [ ] Implement correlation-based stacking (game stacks, team stacks)
- [ ] Run 10,000+ Monte Carlo simulations in <5 seconds
- [ ] Export lineups in DraftKings/FanDuel CSV format
- [ ] Support 150+ concurrent lineup generation requests
- [ ] 80%+ test coverage on critical paths

## All Needed Context

### Documentation & References
```yaml
# MUST READ - Include these in your context window
- url: https://developers.google.com/optimization
  why: OR-Tools documentation for linear programming optimization
  
- url: https://github.com/gin-gonic/gin
  why: Gin framework patterns for REST API implementation
  
- url: https://github.com/gorilla/websocket
  why: WebSocket implementation patterns for real-time updates
  
- url: https://github.com/atlassian/react-beautiful-dnd
  why: Drag-and-drop implementation for lineup builder
  
- url: https://redis.io/docs/latest/develop/clients/go/
  why: Redis caching patterns with go-redis client

- url: https://gorm.io/docs/
  why: GORM ORM patterns for database operations

- file: /Users/jstittsworth/fun/CLAUDE.md
  why: Project architecture, conventions, and commands
  
- file: /Users/jstittsworth/fun/INITIAL.md
  why: Detailed feature requirements and MVP scope

- doc: https://github.com/pydfs-lineup-optimizer/pydfs-lineup-optimizer
  section: Optimization algorithms
  critical: Reference implementation of DFS optimization algorithms

- doc: https://github.com/bcanfield/southpaw
  section: FanDuel integration
  critical: Unofficial FanDuel API patterns (for reference only)
```

### Current Codebase tree
```bash
/Users/jstittsworth/fun/
├── CLAUDE.md
├── INITIAL.md
├── LICENSE
├── PRPs/
│   ├── dfs-lineup-optimizer.md (this file)
│   └── templates/
│       └── prp_base.md
└── README.md
```

### Desired Codebase tree with files to be added
```bash
/Users/jstittsworth/fun/
├── backend/
│   ├── cmd/
│   │   ├── server/
│   │   │   └── main.go              # API server entry point
│   │   └── migrate/
│   │       └── main.go              # Database migration tool
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handlers/           # HTTP request handlers
│   │   │   │   ├── lineup.go       # Lineup CRUD operations
│   │   │   │   ├── optimizer.go    # Optimization endpoints
│   │   │   │   ├── player.go       # Player data endpoints
│   │   │   │   └── simulation.go   # Simulation endpoints
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go         # Authentication middleware
│   │   │   │   └── cors.go         # CORS configuration
│   │   │   └── router.go           # Route definitions
│   │   ├── models/
│   │   │   ├── contest.go          # Contest model
│   │   │   ├── lineup.go           # Lineup model
│   │   │   ├── player.go           # Player model
│   │   │   └── simulation.go       # Simulation results model
│   │   ├── optimizer/
│   │   │   ├── algorithm.go        # Core optimization algorithm
│   │   │   ├── constraints.go      # Position/salary constraints
│   │   │   ├── correlation.go      # Player correlation matrix
│   │   │   └── stacking.go         # Stacking rules implementation
│   │   ├── simulator/
│   │   │   ├── monte_carlo.go      # Monte Carlo simulation engine
│   │   │   ├── distributions.go    # Player performance distributions
│   │   │   └── contest.go          # Contest simulation (GPP/Cash)
│   │   └── services/
│   │       ├── cache.go            # Redis caching service
│   │       ├── export.go           # CSV export service
│   │       ├── lineup.go           # Lineup business logic
│   │       └── websocket.go        # WebSocket hub
│   ├── pkg/
│   │   ├── config/
│   │   │   └── config.go           # Configuration management
│   │   ├── database/
│   │   │   └── connection.go       # Database connection pool
│   │   └── utils/
│   │       ├── errors.go           # Error handling utilities
│   │       └── response.go         # Standard response format
│   ├── migrations/                  # SQL migration files
│   ├── go.mod                      # Go dependencies
│   ├── go.sum                      # Go dependency checksums
│   └── Dockerfile                  # Backend container config
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── LineupBuilder/      # Drag-drop lineup component
│   │   │   ├── OptimizerControls/  # Optimization settings UI
│   │   │   ├── PlayerCard/         # Player display component
│   │   │   ├── PlayerPool/         # Player list with filters
│   │   │   └── SimulationViz/      # Monte Carlo visualization
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx       # Main dashboard
│   │   │   ├── Optimizer.tsx       # Optimization page
│   │   │   └── Lineups.tsx         # Saved lineups page
│   │   ├── hooks/
│   │   │   ├── useWebSocket.ts     # WebSocket connection hook
│   │   │   ├── useOptimizer.ts     # Optimizer API hook
│   │   │   └── usePlayers.ts       # Player data hook
│   │   ├── services/
│   │   │   ├── api.ts              # API client
│   │   │   ├── optimizer.ts        # Optimizer service
│   │   │   └── export.ts           # Export service
│   │   ├── store/
│   │   │   ├── lineup.ts           # Lineup state management
│   │   │   └── player.ts           # Player state management
│   │   ├── types/
│   │   │   ├── contest.ts          # Contest types
│   │   │   ├── lineup.ts           # Lineup types
│   │   │   └── player.ts           # Player types
│   │   ├── App.tsx                 # Main app component
│   │   └── main.tsx                # Entry point
│   ├── package.json                # Node dependencies
│   ├── tsconfig.json               # TypeScript config
│   ├── vite.config.ts              # Vite build config
│   ├── tailwind.config.js          # TailwindCSS config
│   └── Dockerfile                  # Frontend container config
├── examples/
│   ├── optimizer_algorithm.go       # Basic knapsack implementation
│   ├── correlation_matrix.go        # Player correlation example
│   ├── monte_carlo_sim.go          # Simple Monte Carlo demo
│   ├── lineup_builder_component.tsx # React drag-drop example
│   └── api_client.ts               # TypeScript API client example
├── docker-compose.yml              # Multi-service orchestration
├── .env.example                    # Environment variables template
└── .gitignore                      # Git ignore patterns
```

### Known Gotchas & Library Quirks
```go
// CRITICAL: Gin requires handler functions to accept *gin.Context
// Example: func GetLineups(c *gin.Context) not func GetLineups(w http.ResponseWriter, r *http.Request)

// CRITICAL: GORM v2 has different API from v1 - use gorm.io/gorm not github.com/jinzhu/gorm
// Example: db.Where("sport = ?", sport).Find(&players) not db.Where("sport = ?", sport).Find(&players).Error

// CRITICAL: WebSocket connections require upgrade from HTTP
// Must handle connection lifecycle: upgrade, read pump, write pump

// CRITICAL: React Beautiful DnD requires stable IDs for draggable items
// Use database IDs, not array indices which change on re-render

// CRITICAL: Redis go-redis v9 uses context for all operations
// Example: rdb.Set(ctx, "key", "value", 0) not rdb.Set("key", "value", 0)

// CRITICAL: DraftKings CSV has 500 lineup limit per file
// Must implement pagination for large lineup exports

// CRITICAL: Optimizer constraints must validate NFL has different positions than NBA
// NBA: PG, SG, SF, PF, C, G, F, UTIL
// NFL: QB, RB, WR, TE, FLEX, DST
```

## Implementation Blueprint

### Data models and structure

```go
// internal/models/player.go
type Player struct {
    ID           uint      `gorm:"primaryKey"`
    Name         string    `gorm:"not null"`
    Team         string    `gorm:"not null"`
    Position     string    `gorm:"not null"`
    Salary       int       `gorm:"not null"`
    ProjectedFP  float64   `gorm:"not null"`
    Sport        string    `gorm:"not null"`
    ContestID    uint      `gorm:"not null"`
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// internal/models/lineup.go
type Lineup struct {
    ID           uint      `gorm:"primaryKey"`
    UserID       uint      `gorm:"not null"`
    ContestID    uint      `gorm:"not null"`
    Players      []Player  `gorm:"many2many:lineup_players;"`
    TotalSalary  int       `gorm:"not null"`
    ProjectedFP  float64   `gorm:"not null"`
    ActualFP     *float64  // Null until contest completes
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// internal/models/contest.go
type Contest struct {
    ID           uint      `gorm:"primaryKey"`
    Platform     string    `gorm:"not null"` // "draftkings" or "fanduel"
    Sport        string    `gorm:"not null"` // "nba", "nfl", etc
    ContestType  string    `gorm:"not null"` // "gpp" or "cash"
    SalaryCap    int       `gorm:"not null"`
    StartTime    time.Time `gorm:"not null"`
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### List of tasks to complete the PRP in order

```yaml
Task 1: Setup Backend Infrastructure
CREATE backend/go.mod:
  - Initialize Go module: github.com/jstittsworth/dfs-optimizer
  - Add dependencies: gin, gorm, go-redis, gorilla/websocket

CREATE backend/cmd/server/main.go:
  - MIRROR pattern from: Gin quick start guide
  - Setup database connection with GORM
  - Initialize Redis client
  - Start HTTP server on port 8080

CREATE backend/internal/pkg/config/config.go:
  - Environment variable loading with viper
  - Configuration struct with all settings
  - Validation of required configs

Task 2: Database Models and Migrations
CREATE backend/internal/models/*.go:
  - Define all GORM models with proper tags
  - Add indexes for frequent queries
  - Setup model relationships

CREATE backend/cmd/migrate/main.go:
  - GORM AutoMigrate for all models
  - Seed data for testing (sample players)

Task 3: Core Optimization Algorithm
CREATE backend/internal/optimizer/algorithm.go:
  - Implement knapsack algorithm with position constraints
  - Dynamic programming approach for efficiency
  - Return top N lineups sorted by projected points

CREATE backend/internal/optimizer/constraints.go:
  - Position requirements per sport (NBA: 2PG, 2SG, etc)
  - Salary cap validation
  - Min/max player exposure settings

CREATE backend/internal/optimizer/correlation.go:
  - Player correlation matrix calculation
  - Team stacking bonuses
  - Game stacking implementation

Task 4: Monte Carlo Simulation Engine
CREATE backend/internal/simulator/monte_carlo.go:
  - Generate N game simulations
  - Apply player correlations
  - Calculate lineup scores for each simulation

CREATE backend/internal/simulator/distributions.go:
  - Normal distribution for player performance
  - Injury/DNP probability modeling
  - Boom/bust variance by position

Task 5: API Endpoints
CREATE backend/internal/api/handlers/*.go:
  - CRUD operations for lineups
  - Optimization endpoint with parameters
  - Player data endpoints with filtering
  - WebSocket upgrade for real-time updates

CREATE backend/internal/api/router.go:
  - Route definitions with middleware
  - CORS configuration
  - Request validation middleware

Task 6: Frontend Foundation
CREATE frontend/package.json:
  - React 18+ with TypeScript
  - Vite for build tooling
  - Dependencies: react-beautiful-dnd, axios, recharts

CREATE frontend/src/App.tsx:
  - Router setup with main pages
  - Global state provider
  - Theme/styling setup with TailwindCSS

Task 7: Player Pool and Lineup Builder
CREATE frontend/src/components/PlayerPool/:
  - Player list with virtual scrolling
  - Filters: position, team, salary range
  - Search by player name
  - Sort by projection, salary, value

CREATE frontend/src/components/LineupBuilder/:
  - Drag source (player pool)
  - Drop targets (lineup positions)
  - Real-time salary/projection totals
  - Position validation highlighting

Task 8: Optimizer UI and Controls
CREATE frontend/src/components/OptimizerControls/:
  - Number of lineups slider
  - Correlation strength control
  - Stacking options (team/game stacks)
  - Player lock/exclude toggles

CREATE frontend/src/pages/Optimizer.tsx:
  - Integrate all optimizer components
  - WebSocket connection for progress
  - Results display with sorting

Task 9: Export and Platform Integration
CREATE backend/internal/services/export.go:
  - Generate CSV in DraftKings format
  - Generate CSV in FanDuel format
  - Validate lineup rules per platform

CREATE frontend/src/services/export.ts:
  - Download CSV from API
  - Multiple lineup selection
  - Platform format selection

Task 10: Testing and Validation
CREATE backend tests:
  - Unit tests for optimizer algorithm
  - Integration tests for API endpoints
  - Benchmark tests for performance

CREATE frontend tests:
  - Component tests with React Testing Library
  - E2E tests for critical flows
  - Accessibility tests
```

### Per task pseudocode

```go
// Task 3: Core Optimization Algorithm
// backend/internal/optimizer/algorithm.go
func OptimizeLineups(players []models.Player, config OptimizeConfig) []models.Lineup {
    // PATTERN: Use dynamic programming for knapsack
    dp := make([][][]bool, len(players)+1)
    for i := range dp {
        dp[i] = make([][]bool, config.SalaryCap+1)
        // Initialize for each position constraint
    }
    
    // GOTCHA: Must track position counts not just total players
    positionCounts := map[string]int{
        "PG": 2, "SG": 2, "SF": 2, "PF": 2, "C": 1, // NBA
    }
    
    // CRITICAL: Apply correlation bonuses during optimization
    for i, player := range players {
        for salary := player.Salary; salary <= config.SalaryCap; salary++ {
            // Check if adding player maintains position constraints
            if canAddPlayer(dp[i][salary], player.Position, positionCounts) {
                // Calculate value with correlation bonus
                value := player.ProjectedFP + getCorrelationBonus(player, currentLineup)
                // Update DP table
            }
        }
    }
    
    // PATTERN: Generate multiple lineups with diversity
    lineups := make([]models.Lineup, 0, config.NumLineups)
    for i := 0; i < config.NumLineups; i++ {
        lineup := extractLineup(dp, players, config)
        // Apply diversity constraint
        if meetsDiversityRequirement(lineup, lineups, config.MinDifferentPlayers) {
            lineups = append(lineups, lineup)
        }
    }
    
    return lineups
}

// Task 4: Monte Carlo Simulation
// backend/internal/simulator/monte_carlo.go
func SimulateContest(lineups []models.Lineup, config SimConfig) SimulationResult {
    // PATTERN: Use goroutines for parallel simulation
    results := make(chan SimulationRun, config.NumSimulations)
    
    // Worker pool pattern
    numWorkers := runtime.NumCPU()
    for w := 0; w < numWorkers; w++ {
        go simulationWorker(lineups, config, results)
    }
    
    // CRITICAL: Apply correlations in each simulation
    for i := 0; i < config.NumSimulations; i++ {
        // Generate correlated player outcomes
        playerOutcomes := generateCorrelatedOutcomes(lineups[0].Players)
        
        // Calculate lineup scores
        for _, lineup := range lineups {
            score := calculateLineupScore(lineup, playerOutcomes)
            // Track placement and payout
        }
    }
    
    // Aggregate results
    return aggregateSimulations(results)
}

// Task 7: Drag and Drop Implementation
// frontend/src/components/LineupBuilder/index.tsx
const LineupBuilder: React.FC = () => {
    // PATTERN: Use react-beautiful-dnd for drag-drop
    const onDragEnd = (result: DropResult) => {
        // CRITICAL: Validate position compatibility
        if (!isValidPosition(result.source, result.destination)) {
            // Show error toast
            return;
        }
        
        // PATTERN: Immutable state updates
        const newLineup = {
            ...lineup,
            [result.destination.droppableId]: players[result.draggableId]
        };
        
        // GOTCHA: Recalculate totals after each change
        const totals = calculateTotals(newLineup);
        if (totals.salary > SALARY_CAP) {
            // Highlight over-cap in red
        }
        
        setLineup(newLineup);
    };
    
    return (
        <DragDropContext onDragEnd={onDragEnd}>
            {/* Player pool as drag source */}
            {/* Lineup slots as drop targets */}
        </DragDropContext>
    );
};
```

### Integration Points
```yaml
DATABASE:
  - migration: "Create tables: players, lineups, contests, lineup_players"
  - indexes: 
    - "CREATE INDEX idx_players_contest_sport ON players(contest_id, sport)"
    - "CREATE INDEX idx_lineups_user_contest ON lineups(user_id, contest_id)"
  
CONFIG:
  - add to: backend/internal/pkg/config/config.go
  - pattern: |
      type Config struct {
          DatabaseURL string `mapstructure:"DATABASE_URL"`
          RedisURL    string `mapstructure:"REDIS_URL"`
          Port        string `mapstructure:"PORT"`
          CorsOrigins []string `mapstructure:"CORS_ORIGINS"`
      }
  
ROUTES:
  - add to: backend/internal/api/router.go
  - pattern: |
      v1 := r.Group("/api/v1")
      {
          v1.GET("/players", handlers.GetPlayers)
          v1.POST("/optimize", handlers.OptimizeLineups)
          v1.GET("/lineups", handlers.GetLineups)
          v1.POST("/simulate", handlers.SimulateContest)
          v1.GET("/ws", handlers.WebSocketUpgrade)
      }

FRONTEND_API:
  - add to: frontend/src/services/api.ts
  - pattern: |
      const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';
      export const optimizerAPI = {
          optimize: (config: OptimizeConfig) => 
              axios.post(`${API_BASE}/api/v1/optimize`, config),
          getPlayers: (contestId: string) => 
              axios.get(`${API_BASE}/api/v1/players?contest=${contestId}`)
      };
```

## Validation Loop

### Level 1: Syntax & Style (Backend)
```bash
# Navigate to backend directory
cd backend

# Run Go fmt and vet
go fmt ./...
go vet ./...

# Run golangci-lint (install if needed: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
golangci-lint run

# Expected: No errors. If errors, READ and fix.
```

### Level 1: Syntax & Style (Frontend)
```bash
# Navigate to frontend directory
cd frontend

# Run ESLint
npm run lint

# Run TypeScript check
npm run type-check

# Expected: No errors. Fix any issues before proceeding.
```

### Level 2: Unit Tests (Backend)
```go
// CREATE backend/internal/optimizer/algorithm_test.go
func TestOptimizeLineups(t *testing.T) {
    // Test basic lineup generation
    players := []models.Player{
        {Name: "Player1", Position: "PG", Salary: 5000, ProjectedFP: 30},
        // ... more test players
    }
    
    config := OptimizeConfig{
        SalaryCap: 50000,
        NumLineups: 5,
    }
    
    lineups := OptimizeLineups(players, config)
    
    assert.Equal(t, 5, len(lineups))
    for _, lineup := range lineups {
        assert.LessOrEqual(t, lineup.TotalSalary, 50000)
        assert.Equal(t, 9, len(lineup.Players)) // NBA roster size
    }
}

func TestPositionConstraints(t *testing.T) {
    // Test that position requirements are met
    // Test invalid lineups are rejected
}

func TestCorrelationBonus(t *testing.T) {
    // Test stacking bonuses applied correctly
}
```

```bash
# Run backend tests
cd backend
go test ./... -v -cover

# Expected: All tests pass with >80% coverage on critical paths
```

### Level 2: Unit Tests (Frontend)
```typescript
// CREATE frontend/src/components/LineupBuilder/LineupBuilder.test.tsx
import { render, screen } from '@testing-library/react';
import { DragDropContext } from 'react-beautiful-dnd';
import LineupBuilder from './index';

describe('LineupBuilder', () => {
  it('validates position constraints on drop', () => {
    // Test PG can only drop in PG slots
  });
  
  it('updates salary total on player add', () => {
    // Test salary calculation
  });
  
  it('prevents over-cap lineups', () => {
    // Test salary cap validation
  });
});
```

```bash
# Run frontend tests
cd frontend
npm test

# Expected: All tests pass
```

### Level 3: Integration Test
```bash
# Start all services
docker-compose up -d

# Wait for services to be ready
sleep 10

# Test health endpoint
curl http://localhost:8080/health
# Expected: {"status": "ok"}

# Test player endpoint
curl http://localhost:8080/api/v1/players?contest=1
# Expected: JSON array of players

# Test optimization endpoint
curl -X POST http://localhost:8080/api/v1/optimize \
  -H "Content-Type: application/json" \
  -d '{
    "contest_id": 1,
    "num_lineups": 5,
    "correlation_weight": 0.5
  }'
# Expected: JSON array of optimized lineups

# Test WebSocket connection
wscat -c ws://localhost:8080/api/v1/ws
# Expected: Connected successfully

# Check frontend is running
curl http://localhost:5173
# Expected: HTML response
```

## Final Validation Checklist
- [ ] Backend tests pass: `cd backend && go test ./... -v`
- [ ] Frontend tests pass: `cd frontend && npm test`
- [ ] No linting errors: `golangci-lint run` and `npm run lint`
- [ ] No type errors: `npm run type-check`
- [ ] Docker services start: `docker-compose up`
- [ ] API endpoints respond correctly
- [ ] WebSocket connections work
- [ ] Can generate optimized lineups via UI
- [ ] Can export lineups as CSV
- [ ] Lineup validation works (position/salary constraints)
- [ ] Monte Carlo simulations complete in <5 seconds

---

## Anti-Patterns to Avoid
- ❌ Don't use blocking operations in WebSocket handlers
- ❌ Don't store sensitive data in frontend state
- ❌ Don't skip position validation in optimizer
- ❌ Don't use floats for monetary values (use cents as int)
- ❌ Don't hardcode platform-specific rules
- ❌ Don't ignore correlation in simulations
- ❌ Don't create lineups that violate platform rules

## Production Considerations
- Use connection pooling for PostgreSQL
- Implement rate limiting on optimization endpoint
- Cache player data with appropriate TTL
- Use CDN for frontend assets
- Implement proper error tracking (Sentry)
- Add monitoring for optimization performance
- Set up alerts for failed exports

---

**Confidence Score: 9/10**

This PRP provides comprehensive context for implementing a DFS lineup optimizer with all necessary algorithms, architectural patterns, and validation steps. The one point deduction is because some DFS platform APIs are unofficial and may require adjustments during implementation.