# PRP: Core Optimization Algorithm Overhaul

## Goal

Transform the DFS optimization engine from exponential backtracking to sophisticated dynamic programming with advanced player analytics. Achieve 10x speed improvement (sub-3 second optimization for 150 lineups) while enabling professional-grade optimization strategies that compete with industry leaders like SaberSim.

## Why

- **Performance Crisis**: Current algorithm takes 15-30+ seconds for complex contests, unacceptable for production DFS platform
- **Quality Gap**: Missing advanced analytics (ceiling/floor, volatility, ownership projections) that professional players require
- **Scalability Limits**: Hard-coded generation limits and O(n!) complexity prevent scaling to 500+ lineups
- **Competitive Disadvantage**: Suboptimal player selection missing contrarian value plays and sophisticated stacking strategies

## What

Replace the exponential backtracking optimization algorithm with a multi-dimensional dynamic programming knapsack solver enhanced with professional-grade player analytics and multi-objective optimization strategies.

### Success Criteria

- [ ] Optimization time reduced from 15-30s to <3s for 150 lineups
- [ ] Support 500+ lineup generation without performance degradation
- [ ] Increase average lineup score by 8-12% through better player analytics
- [ ] Implement 6+ optimization objectives (ceiling, floor, balanced, contrarian, correlation, value)
- [ ] Reduce memory usage by 60% through efficient data structures
- [ ] Maintain 100% backward compatibility with existing API

## All Needed Context

### Documentation & References

```yaml
# MUST READ - Critical context for implementation

- url: https://medium.com/@fabianterh/how-to-solve-the-knapsack-problem-with-dynamic-programming-eb88c706d3cf
  why: Dynamic programming approach with O(n*budget*positions) complexity
  critical: Memoization performs 40x better than recursive approaches

- url: https://medium.com/@kangeugine/fantasy-football-as-a-data-scientist-part-2-knapsack-problem-6b7083955e93
  why: Multi-dimensional knapsack for fantasy sports with position constraints
  critical: Shows how to handle position eligibility in DP solution

- url: https://medium.com/sports-analytics-and-data-science/fantasy-premier-league-lineup-optimization-using-mixed-linear-programming-the-knapsack-problem-3c19b3b007a2
  why: Advanced constraint handling for multiple position requirements
  
- url: https://www.fantasylabs.com/articles/understanding-ceiling-projections-gpps/
  why: Ceiling/floor projection methodology for tournament optimization
  critical: 85th percentile floor, 15th percentile ceiling for risk modeling

- url: https://www.4for4.com/nfl-dfs-floor-ceiling-projections  
  why: Implementation patterns for volatility and consistency metrics
  critical: Coefficient of variation (CV) = std_dev / mean for volatility

- file: services/optimization-service/internal/optimizer/algorithm.go
  why: Current implementation with O(n!) complexity issues (lines 255-446)
  critical: Hard limit of 10 players per position (line 407), max 10k lineups (lines 249-252)
  
- file: services/optimization-service/internal/optimizer/correlation.go
  why: Existing correlation matrix implementation to preserve
  critical: Sport-specific correlation calculations already working well

- file: services/optimization-service/internal/optimizer/stacking.go  
  why: Sophisticated stacking rules already implemented
  critical: QB stacks, game stacks, ownership leverage patterns

- file: services/optimization-service/internal/optimizer/slots.go
  why: Position slot system for handling flex positions
  critical: Priority-based slot filling, platform-specific requirements

- file: services/optimization-service/internal/optimizer/algorithm_test.go
  why: Performance benchmarks and test patterns to maintain
  critical: Benchmark targets: BenchmarkOptimizeLineups_NBA_Large for 300 players/150 lineups
```

### Current Codebase Tree (Core Optimization Service)

```bash
services/optimization-service/
├── internal/
│   ├── optimizer/
│   │   ├── algorithm.go          # REPLACE: O(n!) backtracking (lines 255-446)
│   │   ├── correlation.go        # ENHANCE: Add advanced correlation metrics  
│   │   ├── golf_correlation.go   # KEEP: Golf-specific correlation logic
│   │   ├── stacking.go          # ENHANCE: Add multi-objective stacking
│   │   ├── slots.go             # KEEP: Position slot handling
│   │   ├── constraints.go       # ENHANCE: Advanced constraint validation
│   │   ├── algorithm_test.go    # ENHANCE: Add performance benchmarks
│   │   └── integration_test.go  # ENHANCE: End-to-end validation
│   ├── api/handlers/
│   │   └── optimization.go     # ENHANCE: Add strategy parameter support
│   └── simulator/              # INTEGRATE: Use analytics for distributions
└── shared/types/
    └── common.go               # ENHANCE: Add analytics types
```

### Desired Codebase Tree with New Files

```bash
services/optimization-service/
├── internal/
│   ├── optimizer/
│   │   ├── algorithm.go          # REPLACED: Dynamic programming knapsack
│   │   ├── analytics.go          # NEW: Player ceiling/floor/volatility engine
│   │   ├── dp_optimizer.go       # NEW: Multi-dimensional DP implementation
│   │   ├── objectives.go         # NEW: Multi-objective optimization strategies
│   │   ├── exposure.go           # NEW: Portfolio-level exposure management
│   │   ├── correlation.go        # ENHANCED: Advanced correlation metrics
│   │   ├── stacking.go          # ENHANCED: Strategy-aware stacking
│   │   ├── constraints.go       # ENHANCED: Performance constraint validation
│   │   └── analytics_test.go    # NEW: Analytics engine test suite
│   └── cache/
│       └── analytics_cache.go   # NEW: Redis caching for computed analytics
```

### Known Gotchas of Our Codebase & Library Quirks

```go
// CRITICAL: Current algorithm limits player evaluation
// Line 407 in algorithm.go: if playersTried >= 10 && len(validLineups) > 0
// This causes us to miss high-value contrarian plays

// CRITICAL: Hard generation limits regardless of quality  
// Lines 249-252: maxLineups := config.NumLineups * 100; if maxLineups > 10000 { maxLineups = 10000 }
// Need to remove artificial caps and use quality-based pruning

// CRITICAL: Position slot system is complex but working
// slots.go handles flex positions correctly - DO NOT break this logic
// Priority-based filling ensures required positions filled first

// CRITICAL: Correlation matrix is sport-specific and working well
// correlation.go has detailed sport correlations - preserve this logic
// Golf correlation uses tee times and course history - keep intact

// CRITICAL: Shared types must maintain backward compatibility
// API contracts in shared/types/common.go cannot break existing consumers
// All new fields must be optional with sensible defaults

// PERFORMANCE: Redis caching patterns from other services
// Use existing cache patterns from shared/pkg/cache/
// Analytics calculations are expensive - cache aggressively
```

## Implementation Blueprint

### Data Models and Structure

Create advanced analytics and optimization state models for type safety and consistency.

```go
// Player analytics for ceiling/floor/volatility calculations
type PlayerAnalytics struct {
    PlayerID           uint    `json:"player_id"`
    BaseProjection     float64 `json:"base_projection"`
    Ceiling           float64 `json:"ceiling"`           // 90th percentile
    Floor             float64 `json:"floor"`             // 10th percentile  
    Volatility        float64 `json:"volatility"`        // Coefficient of variation
    ValueRating       float64 `json:"value_rating"`      // Points per dollar
    OwnershipProjection float64 `json:"ownership_projection"`
    CeilingProbability float64 `json:"ceiling_probability"`
    FloorProbability   float64 `json:"floor_probability"`
}

// Multi-dimensional DP optimization state
type OptimizationState struct {
    Budget           int                 `json:"budget"`
    Positions        map[string]int      `json:"positions"`      // Position -> remaining slots
    UsedPlayers      []uint             `json:"used_players"`
    CurrentScore     float64            `json:"current_score"`
    CorrelationBonus float64            `json:"correlation_bonus"`
    StateHash        string             `json:"state_hash"`     // For memoization
}

// Enhanced optimization configuration
type OptimizeConfigV2 struct {
    // Existing fields preserved for backward compatibility
    SalaryCap           int              `json:"salary_cap"`
    NumLineups          int              `json:"num_lineups"`
    MinDifferentPlayers int              `json:"min_different_players"`
    
    // New strategy options
    Strategy           OptimizationObjective `json:"strategy"`
    PlayerAnalytics    bool                  `json:"enable_analytics"`
    ExposureManagement ExposureConfig        `json:"exposure_config"`
    PerformanceMode    string               `json:"performance_mode"`  // "speed", "quality", "balanced"
    
    // Advanced constraints
    MaxCorrelation     float64              `json:"max_correlation"`
    MinDiversity       int                  `json:"min_diversity"`
    OwnershipStrategy  string               `json:"ownership_strategy"` // "contrarian", "chalk", "balanced"
}
```

### List of Tasks to Complete the PRP (In Order)

```yaml
Task 1 - Create Player Analytics Engine:
  CREATE internal/optimizer/analytics.go:
    - IMPLEMENT PlayerAnalytics struct with ceiling/floor calculations
    - MIRROR error handling patterns from existing optimizer files
    - ADD volatility calculations using coefficient of variation
    - INTEGRATE with existing player data structures

Task 2 - Implement Dynamic Programming Optimizer:
  CREATE internal/optimizer/dp_optimizer.go:
    - IMPLEMENT multi-dimensional knapsack with memoization
    - PATTERN: State management similar to lineupCandidate but optimized
    - REPLACE exponential backtracking with O(n*budget*positions) complexity
    - PRESERVE position slot compatibility from slots.go

Task 3 - Create Multi-Objective Framework:
  CREATE internal/optimizer/objectives.go:
    - IMPLEMENT 6 optimization strategies (ceiling, floor, balanced, contrarian, correlation, value)
    - PATTERN: Strategy pattern similar to stacking.go sport-specific methods
    - INTEGRATE with PlayerAnalytics for advanced scoring

Task 4 - Enhance Exposure Management:
  CREATE internal/optimizer/exposure.go:
    - IMPLEMENT portfolio-level exposure constraints
    - PATTERN: Similar to existing diversity constraints in algorithm.go
    - ADD real-time exposure balancing across lineups

Task 5 - Replace Core Algorithm:
  MODIFY internal/optimizer/algorithm.go:
    - REPLACE recursive backtracking (lines 255-446) with DP calls
    - PRESERVE API compatibility with OptimizeLineups function signature
    - INTEGRATE analytics engine and multi-objective optimization
    - MAINTAIN existing correlation and stacking integration

Task 6 - Add Performance Caching:
  CREATE internal/cache/analytics_cache.go:
    - IMPLEMENT Redis caching for computed analytics
    - PATTERN: Follow existing cache patterns from other services
    - ADD cache warming for tournament data

Task 7 - Enhance API Layer:
  MODIFY internal/api/handlers/optimization.go:
    - ADD strategy parameter support to optimization endpoints
    - PRESERVE backward compatibility with existing requests
    - ADD performance mode selection

Task 8 - Comprehensive Testing:
  CREATE internal/optimizer/analytics_test.go:
    - ADD unit tests for analytics calculations
    - BENCHMARK ceiling/floor/volatility performance
    - VALIDATE against known data patterns

  ENHANCE internal/optimizer/algorithm_test.go:
    - ADD performance benchmarks for DP algorithm
    - VALIDATE 3-second target for 150 lineups
    - TEST backward compatibility with existing API
```

### Per Task Pseudocode

```go
// Task 1 - Analytics Engine Core Logic
func (a *AnalyticsEngine) CalculatePlayerAnalytics(player types.Player, historicalData []PerformanceData) *PlayerAnalytics {
    // PATTERN: Input validation first (existing pattern in optimizer)
    if player.ProjectedPoints <= 0 || player.Salary <= 0 {
        return nil  // Invalid player data
    }
    
    // Calculate base metrics
    baseProjection := player.ProjectedPoints
    
    // CRITICAL: Use coefficient of variation for volatility
    // CV = standard_deviation / mean (lower = more consistent)
    variance := calculateVariance(historicalData)
    volatility := math.Sqrt(variance) / baseProjection
    
    // RESEARCH: Floor = 85th percentile, Ceiling = 15th percentile
    floor := calculatePercentile(historicalData, 0.15)      // 85% chance of scoring at least this
    ceiling := calculatePercentile(historicalData, 0.85)    // 15% chance of scoring at least this
    
    // Value rating: points per dollar (existing pattern)
    valueRating := baseProjection / float64(player.Salary) * 1000
    
    return &PlayerAnalytics{
        PlayerID:           player.ID,
        BaseProjection:     baseProjection,
        Ceiling:           ceiling,
        Floor:             floor,
        Volatility:        volatility,
        ValueRating:       valueRating,
        // Additional fields calculated...
    }
}

// Task 2 - Dynamic Programming Core
func (dp *DPOptimizer) OptimizeWithDP(players []types.Player, config OptimizeConfigV2) []types.GeneratedLineup {
    // PATTERN: Memoization table similar to existing correlation caching
    memoTable := make(map[string]*OptimizationState)
    
    // CRITICAL: Sort players by value rating first (research shows this improves pruning)
    sort.Slice(players, func(i, j int) bool {
        valueI := players[i].ProjectedPoints / float64(players[i].Salary)
        valueJ := players[j].ProjectedPoints / float64(players[j].Salary)
        return valueI > valueJ
    })
    
    // PATTERN: Position slots from existing slots.go system
    slots := GetPositionSlots(config.Contest.Sport, config.Contest.Platform)
    
    // Multi-dimensional DP state space
    for budget := 0; budget <= config.SalaryCap; budget++ {
        for slotIndex := 0; slotIndex < len(slots); slotIndex++ {
            // ALGORITHM: dp[budget][slotIndex] = max(include_player, exclude_player)
            // CRITICAL: Memoize expensive correlation calculations
            stateKey := generateStateKey(budget, slotIndex, usedPlayers)
            
            if cachedState, exists := memoTable[stateKey]; exists {
                // Return cached result - massive performance gain
                continue
            }
            
            // Calculate optimal state for this configuration
            optimalState := calculateOptimalState(budget, slotIndex, players, config)
            memoTable[stateKey] = optimalState
        }
    }
    
    // PATTERN: Convert to existing GeneratedLineup format
    return convertStatesToLineups(memoTable, config.NumLineups)
}

// Task 3 - Multi-Objective Scoring
func (obj *ObjectiveManager) CalculateObjectiveScore(player types.Player, analytics *PlayerAnalytics, objective OptimizationObjective) float64 {
    switch objective {
    case MaxCeiling:
        // GPP tournaments - maximize upside potential
        return analytics.Ceiling + (analytics.CeilingProbability * 20)
        
    case MaxFloor:
        // Cash games - maximize safety
        return analytics.Floor + (analytics.FloorProbability * 15)
        
    case Contrarian:
        // Low ownership tournaments
        ownershipPenalty := math.Pow(player.Ownership/100, 2) * 10
        return analytics.BaseProjection - ownershipPenalty
        
    case Correlation:
        // Stack-heavy optimization
        // PATTERN: Use existing correlation matrix from correlation.go
        correlationBonus := obj.correlations.CalculateLineupCorrelation(currentLineup)
        return analytics.BaseProjection + (correlationBonus * 15)
        
    // Additional objectives...
    }
}
```

### Integration Points

```yaml
EXISTING_CORRELATION_SYSTEM:
  - preserve: correlation.go sport-specific calculations
  - enhance: Add analytics-based correlation weights
  - pattern: "correlationScore += analyticsBonus(player1Analytics, player2Analytics)"

POSITION_SLOT_SYSTEM:
  - preserve: slots.go priority-based filling logic
  - integrate: DP state must respect slot constraints
  - critical: "DO NOT break flex position handling"

API_BACKWARD_COMPATIBILITY:
  - preserve: OptimizeLineups function signature
  - enhance: Add optional V2 config parameters
  - pattern: "if config.Strategy != nil { useNewObjective() } else { useExisting() }"

REDIS_CACHING:
  - pattern: shared/pkg/cache/ existing patterns
  - add: Analytics caching with TTL
  - key: "analytics:player:{id}:date:{yyyymmdd}"

PERFORMANCE_MONITORING:
  - add: Optimization time metrics
  - pattern: existing WebSocket progress reporting
  - critical: "Must hit 3-second target consistently"
```

## Validation Loop

### Level 1: Syntax & Style

```bash
# Run these FIRST - fix any errors before proceeding
cd services/optimization-service
go mod tidy
gofmt -w internal/optimizer/
golangci-lint run internal/optimizer/

# Expected: No linting errors. If errors, READ the error and fix.
```

### Level 2: Unit Tests

```go
// CREATE internal/optimizer/analytics_test.go
func TestCalculatePlayerAnalytics(t *testing.T) {
    player := types.Player{
        ID: 1, Name: "Test Player", Salary: 8000, ProjectedPoints: 40.0,
    }
    historicalData := []PerformanceData{
        {Points: 35.0}, {Points: 42.0}, {Points: 38.0}, {Points: 45.0}, {Points: 30.0},
    }
    
    analytics := engine.CalculatePlayerAnalytics(player, historicalData)
    
    assert.NotNil(t, analytics)
    assert.Equal(t, 40.0, analytics.BaseProjection)
    assert.Greater(t, analytics.Ceiling, analytics.BaseProjection) // Ceiling > base
    assert.Less(t, analytics.Floor, analytics.BaseProjection)     // Floor < base
    assert.Greater(t, analytics.Volatility, 0.0)                 // Volatility calculated
}

func TestDPOptimizerPerformance(t *testing.T) {
    contest, players := setupBenchmarkData("nba", "draftkings", 300)
    config := OptimizeConfigV2{
        SalaryCap: 50000,
        NumLineups: 150,
        Strategy: MaxCeiling,
    }
    
    start := time.Now()
    result := optimizer.OptimizeWithDP(players, config)
    duration := time.Since(start)
    
    assert.Less(t, duration, 3*time.Second, "Must complete in under 3 seconds")
    assert.Len(t, result, 150, "Must generate requested number of lineups")
}
```

```bash
# Run and iterate until passing:
cd services/optimization-service
go test ./internal/optimizer/ -v
go test ./internal/optimizer/ -bench=BenchmarkOptimizeLineups_NBA_Large

# Expected: All tests pass, benchmark under 3 seconds
```

### Level 3: Integration Test

```bash
# Start optimization service
cd services/optimization-service
go run cmd/server/main.go

# Test enhanced API endpoint
curl -X POST http://localhost:8082/optimize \
  -H "Content-Type: application/json" \
  -d '{
    "salary_cap": 50000,
    "num_lineups": 20,
    "strategy": "ceiling",
    "enable_analytics": true,
    "performance_mode": "quality"
  }'

# Expected: Sub-3 second response with enhanced lineup analytics
# Check optimization_time_ms < 3000
```

## Final Validation Checklist

- [ ] All tests pass: `go test ./internal/optimizer/ -v`
- [ ] No linting errors: `golangci-lint run internal/optimizer/`
- [ ] Performance target met: 150 lineups in <3 seconds
- [ ] Backward compatibility: Existing API still works
- [ ] Integration test successful: Enhanced API responds correctly
- [ ] Memory usage reduced: Profile shows 60% reduction
- [ ] Analytics accuracy: Ceiling/floor calculations validated against historical data
- [ ] Multi-objective strategies working: All 6 objectives produce different results

---

## Anti-Patterns to Avoid

- ❌ Don't break existing correlation.go sport-specific logic
- ❌ Don't modify position slot system without full testing
- ❌ Don't ignore performance benchmarks - fix them immediately
- ❌ Don't skip analytics caching - calculations are expensive
- ❌ Don't hardcode strategy weights - make them configurable
- ❌ Don't sacrifice backward compatibility for new features
- ❌ Don't use floating-point keys in memoization tables
- ❌ Don't ignore memory pressure - use efficient data structures

---

**Confidence Score: 9/10**

This PRP provides comprehensive context including exact file locations, performance issues, external research findings, and detailed implementation guidance. The validation loop ensures iterative refinement capability. The main risk is the complexity of the multi-dimensional DP algorithm, but the extensive research and existing codebase patterns significantly reduce implementation risk.