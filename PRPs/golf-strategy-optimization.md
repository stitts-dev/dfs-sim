# PRP: Golf Tournament Strategy Optimization

name: "Golf Tournament Strategy Specialization with Cut & Position Modeling"
description: |

## Purpose
Transform the golf DFS optimization from basic player selection into sophisticated tournament strategy modeling with cut probability optimization, position-based strategies (T5, T10, T25), and dynamic tournament adjustments. This specialization will provide golf DFS users with professional-grade tournament analysis that adapts to tournament progression and optimal finishing position strategies.

## Core Principles
1. **Context is King**: Include ALL necessary documentation, examples, and caveats
2. **Validation Loops**: Provide executable tests/lints the AI can run and fix
3. **Information Dense**: Use keywords and patterns from the codebase
4. **Progressive Success**: Start simple, validate, then enhance
5. **Global rules**: Be sure to follow all rules in CLAUDE.md

---

## Goal
Implement a comprehensive golf tournament strategy optimization system that:
- Integrates cut probability modeling into lineup building with 12-15% improvement in cut rate
- Provides position-specific optimization strategies (Win, T5, T10, T25) for different contest types
- Utilizes advanced weather and tee time correlations for 8-10% ROI improvement
- Enables dynamic tournament adjustments and late swap recommendations
- Builds upon existing golf correlation framework in `services/optimization-service/internal/optimizer/golf_correlation.go`

## Why
- **Business Value**: Professional-grade golf DFS optimization gives competitive edge worth 15-20% ROI improvement
- **User Impact**: Transform casual golf DFS players into sophisticated tournament strategists
- **Integration**: Extends existing golf implementation with minimal breaking changes
- **Problems Solved**: Addresses missing cut optimization, static strategies, and underutilized correlation data

## What
User-visible features:
- **Cut Probability Display**: Show cut probability for each player with confidence levels
- **Strategy Selector**: Choose optimization strategy (Win, T5, T10, T25, Make Cut, Balanced)
- **Weather Impact Analysis**: Visual indicators of weather advantages/disadvantages
- **Live Tournament Tracking**: Real-time cut line updates and position tracking
- **Late Swap Recommendations**: AI-powered suggestions for optimal lineup changes

Technical requirements:
- Extend existing golf models with cut probability and position projection fields
- Implement weather service integration for real-time conditions
- Create position-based optimization objectives in the algorithm
- Build WebSocket updates for live tournament tracking

### Success Criteria
- [ ] Cut probability predictions achieve 75%+ accuracy on historical data
- [ ] Position-based strategies show distinct lineup differences
- [ ] Weather correlations properly influence optimization
- [ ] Late swap recommendations improve ROI by 5%+ in backtesting
- [ ] All existing golf functionality remains functional

## All Needed Context

### Documentation & References (list all context needed to implement the feature)
```yaml
# MUST READ - Include these in your context window
- url: https://datagolf.com/predictive-model-methodology/
  why: Comprehensive golf probability modeling methodology including cut predictions
  
- url: https://link.springer.com/article/10.1007/s00484-023-02549-6
  why: Research showing weather impacts 44% of scoring variance, wind being most significant
  
- file: services/optimization-service/internal/optimizer/golf_correlation.go
  why: Current correlation implementation to extend (lines 72-101 have tee time correlations)
  
- file: services/sports-data-service/internal/models/golf.go
  why: Existing golf data models that need extending with new fields
  
- file: services/sports-data-service/internal/services/golf_projections.go
  why: Current projection service with placeholder weather integration

- doc: https://www.fantasylabs.com/articles/analyzing-mlb-weather-factors/
  section: Weather correlation impacts on DFS
  critical: Wind is all negative impact, no helpful side - critical for strategy

- docfile: INITs/PRD-golf-strategy-optimization.md
  why: Complete feature specification with data structures and algorithms

- url: https://github.com/Azalea-Sports-Analytics/masters-prediction-ML
  why: Example ML approach to golf tournament prediction for reference patterns
```

### Current Codebase tree (run `tree` in the root of the project) to get an overview of the codebase
```bash
services/
├── optimization-service/
│   ├── internal/
│   │   ├── optimizer/
│   │   │   ├── algorithm.go          # Core optimization algorithm
│   │   │   ├── golf_correlation.go   # Golf-specific correlations (EXTEND THIS)
│   │   │   ├── correlation.go        # Base correlation logic
│   │   │   ├── constraints.go        # Position constraints
│   │   │   └── stacking.go          # Stacking strategies
│   │   └── api/
│   │       └── handlers/
│   │           └── optimization.go   # Optimization endpoints
├── sports-data-service/
│   ├── internal/
│   │   ├── models/
│   │   │   └── golf.go              # Golf data models (EXTEND THIS)
│   │   ├── services/
│   │   │   ├── golf_projections.go  # Projection calculations (EXTEND THIS)
│   │   │   └── weather.go           # Weather service (IMPLEMENT THIS)
│   │   └── providers/
│   │       ├── rapidapi_golf.go     # Golf data provider
│   │       └── weather_api.go       # Weather provider (CREATE THIS)
```

### Desired Codebase tree with files to be added and responsibility of file
```bash
services/
├── optimization-service/
│   ├── internal/
│   │   ├── optimizer/
│   │   │   ├── golf_correlation.go      # Extended with weather correlations
│   │   │   ├── golf_cut_probability.go  # NEW: Cut probability engine
│   │   │   ├── golf_position_strategy.go # NEW: Position-based optimization
│   │   │   └── golf_tournament_state.go  # NEW: Live tournament tracking
│   │   └── api/
│   │       └── handlers/
│   │           └── golf_optimization.go   # NEW: Golf-specific endpoints
├── sports-data-service/
│   ├── internal/
│   │   ├── models/
│   │   │   └── golf.go                  # Extended with probability fields
│   │   ├── services/
│   │   │   ├── golf_projections.go      # Enhanced with cut/position modeling
│   │   │   ├── weather.go               # Real weather API integration
│   │   │   └── golf_ml_predictor.go     # NEW: ML-based predictions
│   │   └── providers/
│   │       └── openweather_provider.go   # NEW: OpenWeather API client
└── shared/
    └── types/
        └── golf_strategy.go              # NEW: Shared strategy types
```

### Known Gotchas of our codebase & Library Quirks
```go
// CRITICAL: UUID type issues between models and optimizer
// Example: Cannot convert uuid.UUID to interface{} for correlation matrix
// Solution: Use string IDs in correlation calculations

// CRITICAL: RapidAPI rate limiting is VERY aggressive
// Only 20 requests/day on basic plan - must cache aggressively
// Weather API calls count against this limit

// CRITICAL: GORM preloading with UUID foreign keys requires pointer receivers
// Example: db.Preload("Players").Find(&tournament) won't work without proper model setup

// CRITICAL: WebSocket hub in optimization service expects specific message format
// Must follow existing patterns in services/optimization-service/internal/websocket/hub.go

// CRITICAL: Redis caching uses service-specific DB numbers
// Golf Service: DB 0, Optimization Service: DB 1
// Don't mix cache keys between services
```

## Implementation Blueprint

### Data models and structure

Create the core data models for cut probability and position strategy.
```go
// File: services/shared/types/golf_strategy.go
package types

type TournamentPositionStrategy string

const (
    WinStrategy       TournamentPositionStrategy = "win"
    TopFiveStrategy   TournamentPositionStrategy = "top_5"
    TopTenStrategy    TournamentPositionStrategy = "top_10"
    TopTwentyFive     TournamentPositionStrategy = "top_25"
    CutStrategy       TournamentPositionStrategy = "make_cut"
    BalancedStrategy  TournamentPositionStrategy = "balanced"
)

// File: services/sports-data-service/internal/models/golf.go (EXTEND)
// Add these fields to GolfProjection model:
type GolfProjection struct {
    // ... existing fields ...
    
    // Cut probability modeling
    BaseCutProbability      float64 `json:"base_cut_probability"`
    CourseCutProbability    float64 `json:"course_cut_probability"`
    WeatherAdjustedCut      float64 `json:"weather_adjusted_cut"`
    FinalCutProbability     float64 `json:"final_cut_probability"`
    CutConfidence          float64 `json:"cut_confidence"`
    
    // Position probabilities
    Top5Probability        float64 `json:"top5_probability"`
    Top10Probability       float64 `json:"top10_probability"`
    Top25Probability       float64 `json:"top25_probability"`
    ExpectedFinishPosition float64 `json:"expected_finish_position"`
    
    // Weather impact
    WeatherAdvantage       float64 `json:"weather_advantage"`
    TeeTimeAdvantage      float64 `json:"tee_time_advantage"`
}

// Add to optimization request:
type GolfOptimizationRequest struct {
    OptimizationRequest
    TournamentStrategy    TournamentPositionStrategy `json:"tournament_strategy"`
    CutOptimization      bool                       `json:"enable_cut_optimization"`
    WeatherConsideration bool                       `json:"include_weather"`
    CourseHistory        bool                       `json:"use_course_history"`
    TeeTimeCorrelations  bool                       `json:"tee_time_correlations"`
    RiskTolerance        float64                    `json:"risk_tolerance"`
}
```

### list of tasks to be completed to fullfill the PRP in the order they should be completed

```yaml
Task 1: Extend Golf Models with Probability Fields
MODIFY services/sports-data-service/internal/models/golf.go:
  - FIND pattern: "type GolfProjection struct"
  - ADD cut probability fields after existing projection fields
  - ADD position probability fields
  - ADD weather impact fields
  - PRESERVE existing field names and types

Task 2: Implement Weather Service Integration
CREATE services/sports-data-service/internal/providers/openweather_provider.go:
  - MIRROR pattern from: rapidapi_golf.go
  - IMPLEMENT OpenWeatherMap API client with rate limiting
  - FOCUS on wind speed, temperature, precipitation
  - CACHE responses for 1 hour minimum

CREATE services/sports-data-service/internal/services/weather.go:
  - REPLACE placeholder implementation
  - INTEGRATE with OpenWeather provider
  - MAP weather conditions to golf impact scores
  - HANDLE API failures gracefully

Task 3: Build Cut Probability Engine
CREATE services/optimization-service/internal/optimizer/golf_cut_probability.go:
  - IMPLEMENT historical cut analysis from player data
  - BUILD course-specific cut models
  - INTEGRATE weather impact on cut probability
  - CALCULATE confidence scores based on data quality

Task 4: Implement Position Strategy Framework
CREATE services/optimization-service/internal/optimizer/golf_position_strategy.go:
  - DEFINE position-specific optimization objectives
  - IMPLEMENT different risk/reward profiles per strategy
  - INTEGRATE with existing optimization algorithm
  - ENSURE backward compatibility with default optimization

Task 5: Enhance Golf Projections Service
MODIFY services/sports-data-service/internal/services/golf_projections.go:
  - FIND pattern: "func (s *GolfProjectionService) GenerateProjections"
  - INTEGRATE cut probability calculations
  - ADD position probability modeling
  - CONNECT weather service for real-time adjustments
  - PRESERVE existing projection logic

Task 6: Extend Golf Correlation Engine
MODIFY services/optimization-service/internal/optimizer/golf_correlation.go:
  - FIND pattern: "func (g *GolfCorrelationEngine) CalculateCorrelation"
  - ADD weather correlation adjustments
  - ENHANCE tee time correlations with weather data
  - IMPLEMENT dynamic correlation based on tournament conditions
  - KEEP existing correlation ranges

Task 7: Create Golf-Specific Optimization Endpoints
CREATE services/optimization-service/internal/api/handlers/golf_optimization.go:
  - MIRROR pattern from: optimization.go
  - ADD golf-specific request/response handling
  - IMPLEMENT strategy-based optimization routing
  - INCLUDE detailed analytics in response

Task 8: Build Tournament State Tracker
CREATE services/optimization-service/internal/optimizer/golf_tournament_state.go:
  - TRACK live tournament progress and cut line
  - MONITOR weather changes during tournament
  - CALCULATE late swap recommendations
  - INTEGRATE with WebSocket for real-time updates

Task 9: Add Database Migrations
CREATE services/sports-data-service/migrations/005_add_golf_strategy_fields.sql:
  - ADD cut probability columns to golf_projections
  - ADD position probability columns
  - ADD weather tracking columns
  - CREATE indexes for performance

Task 10: Implement Integration Tests
CREATE services/optimization-service/internal/optimizer/golf_strategy_test.go:
  - TEST cut probability accuracy
  - TEST position strategy differences
  - TEST weather correlation impacts
  - VALIDATE against historical data
```

### Per task pseudocode as needed added to each task
```go
// Task 3: Cut Probability Engine
// Pseudocode for golf_cut_probability.go
type CutProbabilityEngine struct {
    historicalData    *HistoricalCutData
    courseModels      map[string]*CourseCutModel
    weatherService    *WeatherService
}

func (c *CutProbabilityEngine) CalculateCutProbability(player *Player, tournament *Tournament) (*CutProbability, error) {
    // PATTERN: Always validate input first
    if err := c.validateInput(player, tournament); err != nil {
        return nil, err
    }
    
    // Base probability from historical performance
    baseProb := c.calculateHistoricalCutRate(player)
    
    // Course-specific adjustment
    courseProb := baseProb
    if courseModel, exists := c.courseModels[tournament.CourseID]; exists {
        courseProb = courseModel.AdjustProbability(player, baseProb)
    }
    
    // Weather impact (CRITICAL: Check rate limits)
    weatherProb := courseProb
    if weather, err := c.weatherService.GetConditions(tournament.Location); err == nil {
        // Wind > 15mph reduces cut probability by 10-15%
        windImpact := c.calculateWindImpact(weather.WindSpeed)
        weatherProb = courseProb * (1.0 - windImpact)
    }
    
    // Field strength adjustment
    fieldAdjusted := c.adjustForFieldStrength(weatherProb, tournament.FieldStrength)
    
    // Confidence based on data quality
    confidence := c.calculateConfidence(player, tournament)
    
    return &CutProbability{
        PlayerID:           player.ID,
        BaseCutProb:        baseProb,
        CourseCutProb:      courseProb,
        WeatherAdjusted:    weatherProb,
        FinalCutProb:       fieldAdjusted,
        Confidence:         confidence,
    }, nil
}

// Task 4: Position Strategy Implementation
// Key optimization difference per strategy
func (p *PositionOptimizer) OptimizeForStrategy(request *GolfOptimizationRequest) (*OptimizationResult, error) {
    switch request.TournamentStrategy {
    case WinStrategy:
        // PATTERN: Maximize ceiling with high variance
        request.MinPlayerDifference = 4  // More overlap allowed
        request.MaxExposure = 0.40       // Higher exposure to top players
        request.OptimizationSettings.PreferHighVariance = true
        
    case TopTenStrategy:
        // PATTERN: Balance consistency with upside
        request.MinPlayerDifference = 3
        request.MaxExposure = 0.30
        request.OptimizationSettings.WeightCutProbability = 0.7
        
    case CutStrategy:
        // PATTERN: Prioritize cut makers for cash games
        request.OptimizationSettings.MinCutProbability = 0.65
        request.OptimizationSettings.WeightCutProbability = 0.9
    }
    
    // CRITICAL: Pass modified request to core optimizer
    return p.baseOptimizer.Optimize(request)
}

// Task 5: Weather Integration
// Weather impact calculation following research
func (w *WeatherService) CalculateGolfImpact(conditions *WeatherConditions) *WeatherImpact {
    impact := &WeatherImpact{}
    
    // Wind is most significant (19-27% variance impact)
    if conditions.WindSpeed > 20 {
        impact.ScoreImpact = 2.5  // strokes
        impact.VarianceMultiplier = 1.4
    } else if conditions.WindSpeed > 15 {
        impact.ScoreImpact = 1.5
        impact.VarianceMultiplier = 1.25
    } else if conditions.WindSpeed > 10 {
        impact.ScoreImpact = 0.75
        impact.VarianceMultiplier = 1.1
    }
    
    // Temperature (wet-bulb better predictor than air temp)
    wetBulb := w.calculateWetBulb(conditions.Temperature, conditions.Humidity)
    if wetBulb < 50 {
        impact.ScoreImpact += 0.5
        impact.VarianceMultiplier *= 1.1
    }
    
    // Rain softens course (helps and hurts equally)
    if conditions.Precipitation > 0 {
        impact.SoftConditions = true
        impact.DistanceReduction = 0.05  // 5% distance loss
    }
    
    return impact
}
```

### Integration Points
```yaml
DATABASE:
  - migration: "Add cut_probability, position_probabilities to golf_projections table"
  - index: "CREATE INDEX idx_golf_proj_cut_prob ON golf_projections(final_cut_probability)"
  - index: "CREATE INDEX idx_golf_proj_position ON golf_projections(expected_finish_position)"
  
CONFIG:
  - add to: services/sports-data-service/config/config.go
  - pattern: "OPENWEATHER_API_KEY = os.Getenv('OPENWEATHER_API_KEY')"
  - pattern: "WEATHER_CACHE_TTL = 3600  // 1 hour"
  
ROUTES:
  - add to: services/optimization-service/internal/api/router.go
  - pattern: "v1.POST('/optimize/golf', handlers.OptimizeGolf)"
  - pattern: "v1.GET('/optimize/golf/strategies', handlers.GetGolfStrategies)"
  
CACHE:
  - add to: services/sports-data-service/internal/services/cache.go
  - pattern: "WEATHER_PREFIX = 'weather:'"
  - pattern: "CUT_PROB_PREFIX = 'cutprob:'"
```

## Validation Loop

### Level 1: Syntax & Style
```bash
# Run these FIRST - fix any errors before proceeding
cd services/optimization-service
golangci-lint run internal/optimizer/golf_*.go

cd services/sports-data-service  
golangci-lint run internal/services/golf_*.go
golangci-lint run internal/providers/*weather*.go

# Expected: No errors. If errors, READ the error and fix.
```

### Level 2: Unit Tests each new feature/file/function use existing test patterns
```go
// CREATE services/optimization-service/internal/optimizer/golf_cut_probability_test.go
func TestCutProbabilityCalculation(t *testing.T) {
    engine := NewCutProbabilityEngine()
    
    // Test historical cut rate
    player := &Player{
        ID: "test-player",
        GolfStats: &GolfStats{
            CutsMade: 15,
            TournamentsPlayed: 20,
        },
    }
    
    prob, err := engine.CalculateCutProbability(player, mockTournament())
    assert.NoError(t, err)
    assert.InDelta(t, 0.75, prob.BaseCutProb, 0.01)
}

func TestWeatherImpactOnCutProbability(t *testing.T) {
    // Test wind reduces cut probability
    engine := NewCutProbabilityEngine()
    engine.weatherService = &MockWeatherService{
        WindSpeed: 25, // High wind
    }
    
    prob, err := engine.CalculateCutProbability(mockPlayer(), mockTournament())
    assert.NoError(t, err)
    assert.Less(t, prob.WeatherAdjusted, prob.CourseCutProb)
}

func TestPositionStrategyOptimization(t *testing.T) {
    optimizer := NewPositionOptimizer()
    
    // Test different strategies produce different lineups
    winLineup := optimizer.OptimizeForStrategy(&GolfOptimizationRequest{
        TournamentStrategy: WinStrategy,
    })
    
    cutLineup := optimizer.OptimizeForStrategy(&GolfOptimizationRequest{
        TournamentStrategy: CutStrategy,
    })
    
    // Win strategy should have higher variance players
    assert.Greater(t, calculateLineupVariance(winLineup), calculateLineupVariance(cutLineup))
}
```

```bash
# Run and iterate until passing:
cd services/optimization-service
go test ./internal/optimizer/golf_* -v

cd services/sports-data-service
go test ./internal/services/golf_* -v
go test ./internal/providers/*weather* -v

# If failing: Read error, understand root cause, fix code, re-run
```

### Level 3: Integration Test
```bash
# Start services
docker-compose up -d sports-data-service optimization-service

# Test golf optimization with strategy
curl -X POST http://localhost:8082/api/v1/optimize/golf \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "sport": "GOLF",
    "contest_id": "test-tournament-id",
    "tournament_strategy": "top_10",
    "enable_cut_optimization": true,
    "include_weather": true,
    "num_lineups": 20
  }'

# Expected: Response with lineups optimized for T10 finish
# Check: avg_cut_probability > 0.70
# Check: weather_impact included in response

# Test cut probability endpoint
curl -X GET http://localhost:8081/api/v1/golf/tournaments/{id}/projections \
  -H "Authorization: Bearer $TOKEN"

# Expected: Projections include cut_probability fields
# If error: Check logs at docker-compose logs sports-data-service
```

### Level 4: Historical Validation
```bash
# Run backtesting script
cd services/optimization-service
go run cmd/backtest/main.go \
  --sport=golf \
  --strategy=cut_optimization \
  --start-date=2024-01-01 \
  --end-date=2024-12-31

# Expected metrics:
# - Cut rate improvement: 12-15%
# - Position accuracy: Within 5% of target
# - Weather correlation: R² > 0.3
```

## Final validation Checklist
- [ ] All tests pass: `go test ./... -v`
- [ ] No linting errors: `golangci-lint run`
- [ ] Cut probability predictions validated against historical data
- [ ] Position strategies produce distinct lineups
- [ ] Weather integration reduces API calls via caching
- [ ] WebSocket updates work for live tournament tracking
- [ ] Backward compatibility maintained for existing endpoints
- [ ] Performance: Optimization completes in < 5 seconds for 150 lineups
- [ ] Documentation updated with new strategy options

---

## Anti-Patterns to Avoid
- ❌ Don't make weather API calls without checking cache first
- ❌ Don't ignore RapidAPI rate limits (20/day!)
- ❌ Don't hardcode cut probability thresholds
- ❌ Don't break existing golf optimization behavior
- ❌ Don't store weather data without TTL
- ❌ Don't calculate correlations with UUID type (use strings)

## Additional Research Context

### Cut Line Modeling Best Practices
Based on Data Golf methodology:
- Use 2-year weighted averages for baseline cut probability
- Recent form (last 7 events) is highly predictive
- Course history matters but sample sizes are small
- Field strength significantly impacts cut difficulty

### Weather Impact Research
From peer-reviewed studies:
- Wind explains 19-27% of scoring variance
- Combined weather effects explain 44% of variance
- Wet-bulb temperature better predictor than air temperature
- Morning vs afternoon waves can have 2-3 stroke differences

### Position Strategy Insights
From DFS analysis:
- Only 9% of lineups have all 6 golfers make cut
- High ownership on recent good performers creates contrarian opportunities
- 64% of rounds show significant variance from average
- Course fit exists but is difficult to model systematically

### Correlation Considerations
From research findings:
- Playing partners have minimal direct correlation
- Weather-based tee time correlation is primary factor
- Same-group stacking not recommended unlike team sports
- Focus on weather windows rather than playing partnerships

## Performance Considerations
- Cache weather data aggressively (1-hour minimum)
- Pre-calculate cut probabilities during data sync
- Use database indexes on probability fields
- Implement circuit breaker for weather API
- Batch probability calculations for efficiency

## Future Enhancement Opportunities
1. Machine learning cut prediction models
2. Strokes gained category integration  
3. Shot-by-shot simulation modeling
4. International tour support (European, Asian)
5. Team event optimization (Ryder Cup format)

---

**Confidence Score**: 8.5/10
- Strong foundation with existing golf implementation
- Clear research backing for algorithms
- Some complexity in weather integration and ML aspects
- Risk in achieving exact percentage improvements