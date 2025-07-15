# PRD: Core Optimization Algorithm Overhaul

## Executive Summary

Transform the DFS optimization engine from exponential backtracking to sophisticated dynamic programming with advanced player analytics. This overhaul will improve lineup generation speed by 10x while enabling professional-grade optimization strategies that compete with industry leaders.

## Problem Statement

### Current State Analysis
**Location**: `services/optimization-service/internal/optimizer/algorithm.go`

**Critical Issues Identified:**
1. **Exponential Complexity**: Lines 255-446 use recursive backtracking with O(n!) complexity
2. **Limited Player Evaluation**: Line 407 caps evaluation to top 10 players per position, missing value plays
3. **Hard Generation Limits**: Lines 249-252 artificially cap at 10,000 lineups regardless of quality
4. **Basic Analytics**: Missing ceiling/floor calculations, volatility modeling, advanced correlation
5. **Inefficient Exposure Management**: Simplistic diversity constraints without portfolio optimization

**Performance Impact:**
- Lineup generation taking 15-30+ seconds for complex contests
- Suboptimal player selection missing contrarian value plays
- Poor scaling for tournaments requiring 100+ lineups
- Missing advanced strategies used by professional DFS players

## Success Metrics

### Performance Targets
- **Speed**: Reduce optimization time from 15-30s to <3s for 150 lineups
- **Quality**: Increase average lineup score by 8-12% through better player analytics
- **Scalability**: Support 500+ lineup generation without performance degradation
- **Memory**: Reduce memory usage by 60% through efficient data structures

### User Experience Improvements  
- **Response Time**: Sub-3 second optimization for all contest types
- **Lineup Diversity**: Ensure 95%+ unique lineups with configurable exposure limits
- **Strategy Options**: Support 5+ optimization objectives (ceiling, floor, balanced, contrarian, chalk)

## Technical Specifications

### 1. Dynamic Programming Knapsack Implementation

**Algorithm**: Multi-dimensional knapsack with position constraints
```go
type OptimizationState struct {
    Budget         int
    Positions      map[string]int  // Position -> remaining slots
    UsedPlayers    []uint
    CurrentScore   float64
    CorrelationBonus float64
}

type DPOptimizer struct {
    memoTable      map[string]*OptimizationState
    playersByValue []types.Player
    correlations   *CorrelationMatrix
    config         OptimizeConfig
}
```

**Key Improvements:**
- **O(n * budget * positions)** complexity vs current O(n!)
- **Memoization** of partial solutions
- **Value-based sorting** with advanced metrics
- **Parallel processing** for multiple lineup generation

### 2. Advanced Player Analytics Engine

**Player Value Calculation:**
```go
type PlayerAnalytics struct {
    BaseProjection    float64
    Ceiling          float64  // 90th percentile outcome
    Floor            float64  // 10th percentile outcome  
    Volatility       float64  // Standard deviation
    ValueRating      float64  // Points per dollar
    OwnershipProjec  float64  // Projected contest ownership
    CeilingProbability float64 // Probability of hitting ceiling
    FloorProbability float64   // Probability of hitting floor
}
```

**Analytics Sources:**
- **Historical Performance**: Last 10 games, seasonal trends
- **Matchup Data**: Opponent rankings, pace, implied totals
- **Weather/Conditions**: Golf wind, NFL weather, NBA rest
- **Ownership Projections**: Contest-specific crowd behavior modeling

### 3. Multi-Objective Optimization Framework

**Optimization Strategies:**
```go
type OptimizationObjective string

const (
    MaxCeiling     OptimizationObjective = "ceiling"     // GPP tournaments
    MaxFloor       OptimizationObjective = "floor"       // Cash games  
    Balanced       OptimizationObjective = "balanced"    // Mixed approach
    Contrarian     OptimizationObjective = "contrarian"  // Low ownership
    Correlation    OptimizationObjective = "correlation" // Stack-heavy
    ValuePlay      OptimizationObjective = "value"       // High pts/dollar
)
```

**Objective Functions:**
- **Ceiling**: `score = ceiling_projection + (ceiling_probability * 20)`
- **Floor**: `score = floor_projection + (floor_probability * 15)`  
- **Contrarian**: `score = projection - (ownership_penalty * ownership^2)`

### 4. Sophisticated Exposure Management

**Portfolio-Level Constraints:**
```go
type ExposureManager struct {
    PlayerLimits    map[uint]ExposureLimit
    PositionLimits  map[string]ExposureLimit  
    StackLimits     map[string]ExposureLimit
    CorrelationMax  float64 // Max lineup correlation
}

type ExposureLimit struct {
    MinExposure     float64  // Minimum % across lineups
    MaxExposure     float64  // Maximum % across lineups  
    TargetExposure  float64  // Optimal % target
    Flexibility     float64  // Allowed variance from target
}
```

## Implementation Plan

### Phase 1: Core Algorithm Replacement (Week 1-2)
1. **DP Framework**: Implement multi-dimensional knapsack solver
2. **State Management**: Add memoization and pruning strategies  
3. **Performance Testing**: Validate 10x speed improvement
4. **Backward Compatibility**: Ensure existing API contracts maintained

### Phase 2: Player Analytics Integration (Week 3-4)
1. **Analytics Engine**: Build ceiling/floor/volatility calculations
2. **Data Pipeline**: Integrate matchup and historical data sources
3. **Caching Layer**: Implement Redis caching for computed analytics
4. **Validation**: Compare projections against known benchmarks

### Phase 3: Multi-Objective Framework (Week 5-6)
1. **Objective Functions**: Implement 6 optimization strategies
2. **Configuration API**: Add strategy selection to API endpoints
3. **A/B Testing**: Framework for strategy performance comparison
4. **Documentation**: User guides for strategy selection

### Phase 4: Advanced Exposure Management (Week 7-8)
1. **Portfolio Constraints**: Implement cross-lineup exposure rules
2. **Dynamic Balancing**: Real-time adjustment algorithms
3. **Correlation Limits**: Prevent over-correlated lineup sets
4. **Reporting**: Exposure analytics and optimization reports

## API Changes

### Enhanced Optimization Request
```go
type OptimizeConfigV2 struct {
    // Existing fields...
    Strategy           OptimizationObjective `json:"strategy"`
    PlayerAnalytics    bool                  `json:"enable_analytics"`
    ExposureManagement ExposureConfig        `json:"exposure_config"`
    PerformanceMode    string               `json:"performance_mode"` // "speed", "quality", "balanced"
    
    // Advanced constraints
    MaxCorrelation     float64              `json:"max_correlation"`
    MinDiversity       int                  `json:"min_diversity"`
    OwnershipStrategy  string               `json:"ownership_strategy"` // "contrarian", "chalk", "balanced"
}
```

### Enhanced Response Format
```go
type OptimizerResultV2 struct {
    Lineups            []types.GeneratedLineup `json:"lineups"`
    Analytics          LineupSetAnalytics      `json:"analytics"`
    OptimizationTime   int64                   `json:"optimization_time_ms"`
    Strategy           OptimizationObjective   `json:"strategy_used"`
    ExposureReport     ExposureSummary         `json:"exposure_report"`
    PerformanceMetrics OptimizationMetrics     `json:"performance_metrics"`
}

type LineupSetAnalytics struct {
    AverageProjection  float64            `json:"avg_projection"`
    ProjectionRange    [2]float64         `json:"projection_range"`
    CorrelationMatrix  [][]float64        `json:"correlation_matrix"`
    OwnershipAnalysis  OwnershipSummary   `json:"ownership_analysis"`
    ExposureBreakdown  map[string]float64 `json:"exposure_breakdown"`
}
```

## Integration Points

### With AI Recommendations (PRD-2)
- **Player Analytics**: Share ceiling/floor data for recommendation context
- **Strategy Alignment**: AI recommendations adapt to selected optimization objective
- **Real-time Sync**: Analytics updates trigger recommendation refreshes

### With Monte Carlo (PRD-3)  
- **Distribution Parameters**: Use analytics variance for simulation distributions
- **Correlation Data**: Enhanced correlation matrix for simulation accuracy
- **Outcome Modeling**: Ceiling/floor bounds for realistic outcome generation

### With Golf Strategy (PRD-4)
- **Cut Probability**: Integrate cut modeling into golf player analytics
- **Tournament Position**: Use analytics for T5/T10/T25 optimization strategies
- **Course History**: Factor course-specific data into player analytics

## Risk Mitigation

### Performance Risks
- **Memory Usage**: Implement streaming optimization for large player pools
- **Algorithm Complexity**: Add complexity monitoring and automatic fallbacks
- **Cache Invalidation**: Smart cache eviction based on data freshness

### Quality Risks  
- **Analytics Accuracy**: Validate against historical backtesting data
- **Strategy Effectiveness**: A/B testing framework for strategy validation
- **Edge Cases**: Comprehensive test suite for constraint combinations

## Success Validation

### Automated Testing
- **Performance Benchmarks**: Sub-3 second optimization for 150 lineups
- **Quality Metrics**: Lineup scoring improvement validation
- **Stress Testing**: 500+ lineup generation under load

### User Acceptance  
- **Beta Testing**: Roll out to power users for feedback
- **Strategy Comparison**: Side-by-side with current algorithm
- **Professional Validation**: Compare against industry benchmark tools

## Future Considerations

### Machine Learning Integration
- **Pattern Recognition**: Learn from successful lineup patterns
- **Dynamic Weights**: Adjust correlation weights based on performance data
- **Predictive Analytics**: Forecast optimal strategies by contest type

### Advanced Strategies
- **Game Theory**: Consider opponent lineup strategies in optimization
- **Live Optimization**: Real-time adjustments during contest periods
- **Multi-Contest**: Portfolio optimization across multiple contests simultaneously

---

**Estimated Timeline**: 8 weeks
**Resource Requirements**: 2 senior engineers, 1 data scientist
**Dependencies**: Enhanced data pipeline, caching infrastructure
**Risk Level**: Medium (algorithm complexity, performance requirements)