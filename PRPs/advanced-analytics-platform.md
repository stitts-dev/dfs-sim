# PRP: Advanced Analytics & Performance Tracking Platform

name: advanced-analytics-platform
description: Build a comprehensive analytics platform applying portfolio theory to DFS lineup management, integrating machine learning for pattern recognition, and creating advanced performance tracking that learns from user decisions.

## Core Principles
1. **Context is King**: All portfolio theory algorithms, ML model implementations, and data pipeline details included
2. **Validation Loops**: Statistical validation, backtesting framework, and performance metrics verification
3. **Information Dense**: Complete implementation blueprints with pseudocode for complex algorithms
4. **Progressive Success**: Start with core analytics infrastructure, then add ML, then portfolio optimization
5. **Global rules**: Follow Go conventions, use existing patterns, integrate with current architecture

## Goal
Implement institutional-quality analytics for DFS players including portfolio-level risk management, machine learning pattern recognition, comprehensive performance tracking, and continuous learning from user decisions to improve strategies by 20-30% ROI.

## Why
- **Business Value**: Users currently make suboptimal portfolio decisions costing 15-25% ROI
- **User Impact**: Provides data-driven insights to improve long-term profitability
- **Integration**: Enhances existing optimization with portfolio-level constraints and ML insights
- **Problem Solved**: Transforms isolated lineup decisions into systematic portfolio management

## What
### User-Visible Behavior
1. Portfolio Analytics Dashboard showing risk-adjusted returns, Sharpe ratios, diversification scores
2. ML-powered recommendations based on historical performance patterns
3. Real-time performance tracking with attribution analysis
4. Strategy evolution reports showing improvement over time
5. A/B testing framework for strategy validation

### Success Criteria
- [ ] Portfolio optimization reduces correlation by 40% while maintaining returns
- [ ] ML models achieve 75%+ accuracy on user preference prediction
- [ ] Performance attribution identifies specific decision impacts
- [ ] Real-time analytics updates via WebSocket
- [ ] Historical data aggregation with configurable time windows
- [ ] Integration with existing optimization engine

## All Needed Context

### Documentation & References
```yaml
- file: services/optimization-service/internal/optimizer/algorithm.go
  why: Current optimization patterns to extend with portfolio constraints

- file: backend.deprecated/internal/services/ai_recommendations.go  
  why: Existing AI integration patterns and Claude API usage

- file: services/optimization-service/internal/simulator/monte_carlo.go
  why: Statistical computation patterns and worker pool architecture

- file: services/optimization-service/internal/websocket/hub.go
  why: WebSocket patterns for real-time analytics updates

- url: https://github.com/gorgonia/gorgonia
  section: Basic usage and tensor operations
  why: Neural network implementation for pattern recognition

- url: https://github.com/sjwhitworth/golearn
  section: Classification and regression examples
  why: Traditional ML algorithms for performance prediction

- url: https://github.com/gonum/gonum
  section: Matrix operations and optimization
  why: Numerical algorithms for portfolio optimization

- url: https://github.com/cimomo/portfolio-go
  section: Portfolio tracking patterns
  why: Reference implementation for portfolio analytics

CRITICAL: Go ML ecosystem is limited compared to Python. We'll use:
- Gorgonia for neural networks (pattern recognition)
- GoLearn for traditional ML (random forests, regression)
- Gonum for numerical optimization (portfolio theory)
- Custom implementations where libraries are lacking
```

### Current Codebase
```
services/
├── optimization-service/
│   ├── internal/
│   │   ├── optimizer/         # Extend with portfolio constraints
│   │   ├── simulator/         # Add portfolio simulation
│   │   └── websocket/         # Real-time analytics updates
│   └── pkg/
│       └── cache/            # Analytics result caching
├── user-service/
│   └── internal/models/      # User performance profiles
└── api-gateway/
    └── internal/api/handlers/ # Analytics API endpoints

shared/
├── types/                    # Common analytics types
└── pkg/
    └── database/            # Analytics table migrations
```

### Desired Codebase
```
services/
├── optimization-service/
│   ├── internal/
│   │   ├── analytics/
│   │   │   ├── portfolio/          # Portfolio optimization
│   │   │   ├── ml/                 # Machine learning models
│   │   │   ├── performance/        # Performance tracking
│   │   │   └── learning/           # Adaptation engine
│   │   ├── optimizer/              # Extended with portfolio
│   │   └── websocket/              # Analytics streaming
│   └── pkg/
│       └── analytics/              # Shared analytics utils
└── analytics-service/              # New dedicated service
    ├── cmd/server/main.go
    ├── internal/
    │   ├── aggregator/             # Data aggregation
    │   ├── models/                 # Analytics models
    │   └── api/handlers/           # Analytics endpoints
    └── migrations/                 # Analytics tables

backend.deprecated/
└── internal/services/              # Reference implementations
```

### Known Gotchas & Library Quirks
```yaml
GOTCHA 1: Gorgonia tensor operations
  - Tensors are not garbage collected automatically
  - Must call g.Close() on graphs to prevent memory leaks
  - Use WithName() for debugging tensor operations

GOTCHA 2: GoLearn data format
  - Requires CSV/LibSVM format for training data
  - Feature names must not contain spaces
  - Missing values must be explicitly handled

GOTCHA 3: Portfolio optimization convergence
  - Quadratic programming can fail with ill-conditioned matrices
  - Add regularization term to covariance matrix
  - Use iterative refinement for large portfolios

GOTCHA 4: Real-time analytics performance
  - Aggregate data in background workers
  - Use materialized views for complex queries
  - Implement sampling for high-frequency updates

GOTCHA 5: ML model versioning
  - Store model artifacts with version tags
  - Maintain feature schema compatibility
  - Implement gradual rollout for new models
```

## Implementation Blueprint

### Data Models
```go
// Analytics Service Models
type PortfolioAnalytics struct {
    UserID              int                    `json:"user_id"`
    TimeFrame           string                 `json:"time_frame"`
    TotalROI            float64                `json:"total_roi"`
    SharpeRatio         float64                `json:"sharpe_ratio"`
    MaxDrawdown         float64                `json:"max_drawdown"`
    DiversificationScore float64                `json:"diversification_score"`
    RiskContribution    map[string]float64     `json:"risk_contribution"`
    UpdatedAt           time.Time              `json:"updated_at"`
}

type MLPrediction struct {
    PredictionID        string                 `json:"prediction_id"`
    UserID              int                    `json:"user_id"`
    ModelID             string                 `json:"model_id"`
    PredictionType      string                 `json:"prediction_type"`
    Features            map[string]float64     `json:"features"`
    Prediction          interface{}            `json:"prediction"`
    Confidence          float64                `json:"confidence"`
    Timestamp           time.Time              `json:"timestamp"`
}

type UserPerformanceHistory struct {
    UserID              int                    `json:"user_id"`
    Date                time.Time              `json:"date"`
    ContestsEntered     int                    `json:"contests_entered"`
    TotalSpent          float64                `json:"total_spent"`
    TotalWon            float64                `json:"total_won"`
    WinRate             float64                `json:"win_rate"`
    AvgROI              float64                `json:"avg_roi"`
    SportBreakdown      map[string]Performance `json:"sport_breakdown"`
}
```

### Task List
```yaml
Task 1:
CREATE services/optimization-service/internal/analytics/portfolio/optimizer.go:
  - PATTERN: Reference optimization-service/internal/optimizer/algorithm.go
  - Implement Modern Portfolio Theory calculations
  - Add covariance matrix computation
  - Include risk parity optimization
  - CRITICAL: Use gonum/mat for matrix operations

Task 2:
CREATE services/optimization-service/internal/analytics/ml/feature_extractor.go:
  - Extract features from user history, lineups, contest data
  - Implement time-series features (rolling averages, trends)
  - Add categorical encoding for sports/positions
  - PATTERN: Similar to ai_recommendations.go data preparation

Task 3:
CREATE services/optimization-service/internal/analytics/ml/predictor.go:
  - Implement GoLearn random forest for performance prediction
  - Add Gorgonia neural network for pattern recognition
  - Include model training and evaluation pipelines
  - GOTCHA: Handle GoLearn's CSV format requirements

Task 4:
CREATE services/optimization-service/internal/analytics/performance/tracker.go:
  - Build performance aggregation engine
  - Implement attribution analysis
  - Add benchmark comparison logic
  - PATTERN: Use simulator/monte_carlo.go worker pools

Task 5:
MODIFY services/optimization-service/internal/optimizer/algorithm.go:
  - Add portfolio-level constraints
  - Integrate ML predictions into optimization
  - Include risk budgeting logic
  - CRITICAL: Maintain backward compatibility

Task 6:
CREATE migrations/012_add_analytics_tables.sql:
  - portfolio_analytics table
  - ml_predictions table  
  - user_performance_history table
  - feature_cache table
  - model_artifacts table

Task 7:
CREATE services/optimization-service/internal/api/handlers/analytics.go:
  - Portfolio analytics endpoints
  - ML prediction endpoints
  - Performance report endpoints
  - PATTERN: Follow existing handler patterns

Task 8:
MODIFY services/optimization-service/internal/websocket/hub.go:
  - Add analytics event types
  - Implement real-time analytics streaming
  - Include performance update broadcasts
  - GOTCHA: Rate limit high-frequency updates

Task 9:
CREATE services/optimization-service/pkg/analytics/metrics.go:
  - Sharpe ratio calculation
  - Maximum drawdown computation
  - Risk-adjusted returns
  - Portfolio correlation analysis
  - CRITICAL: Handle edge cases (zero variance)

Task 10:
CREATE cmd/analytics-worker/main.go:
  - Background worker for aggregations
  - Model retraining scheduler
  - Performance report generator
  - PATTERN: Similar to cmd/server/main.go
```

### Pseudocode (Per Task)

#### Task 1: Portfolio Optimizer
```go
// CRITICAL: Modern Portfolio Theory implementation
func OptimizePortfolio(lineups []Lineup, config PortfolioConfig) (*PortfolioResult, error) {
    // Step 1: Calculate returns matrix
    returns := calculateReturns(lineups)
    
    // Step 2: Compute covariance matrix
    // GOTCHA: Add regularization to avoid singular matrix
    covMatrix := gonum.Covariance(returns)
    regularized := addRegularization(covMatrix, 1e-8)
    
    // Step 3: Solve quadratic programming problem
    // min: 0.5 * w^T * Σ * w - λ * w^T * μ
    // s.t.: w^T * 1 = 1, w >= 0
    weights := solveQP(regularized, expectedReturns, config.RiskAversion)
    
    // Step 4: Apply portfolio constraints
    // PATTERN: Similar to optimizer/constraints.go
    constrainedWeights := applyConstraints(weights, config.Constraints)
    
    return &PortfolioResult{
        Weights: constrainedWeights,
        ExpectedReturn: dotProduct(constrainedWeights, expectedReturns),
        Risk: math.Sqrt(portfolioVariance(constrainedWeights, covMatrix)),
    }, nil
}
```

#### Task 3: ML Predictor
```go
// CRITICAL: Pattern recognition with Gorgonia
func (p *Predictor) TrainNeuralNetwork(features, labels *tensor.Dense) error {
    g := gorgonia.NewGraph()
    
    // Input layer
    x := gorgonia.NewMatrix(g, tensor.Float64, 
        gorgonia.WithShape(batchSize, featureCount),
        gorgonia.WithName("x"))
    
    // Hidden layers
    // GOTCHA: Initialize weights properly to avoid vanishing gradients
    w1 := gorgonia.NewMatrix(g, tensor.Float64,
        gorgonia.WithShape(featureCount, hiddenSize),
        gorgonia.WithInit(gorgonia.GlorotU(1.0)),
        gorgonia.WithName("w1"))
    
    // Forward pass
    hidden := gorgonia.Must(gorgonia.Mul(x, w1))
    activated := gorgonia.Must(gorgonia.Rectify(hidden))
    
    // Output layer with softmax for classification
    output := gorgonia.Must(gorgonia.SoftMax(finalLayer))
    
    // Loss function
    losses := gorgonia.Must(gorgonia.Neg(
        gorgonia.Must(gorgonia.Mean(
            gorgonia.Must(gorgonia.Mul(y, 
                gorgonia.Must(gorgonia.Log(output))))))))
    
    // CRITICAL: Must close graph to prevent memory leaks
    defer g.Close()
    
    // Training loop with gradient descent
    solver := gorgonia.NewAdamSolver(gorgonia.WithLearnRate(0.001))
    
    return nil
}
```

#### Task 4: Performance Tracker
```go
// PATTERN: Worker pool from monte_carlo.go
func (t *Tracker) AggregatePerformance(userID int, timeFrame string) (*PerformanceReport, error) {
    // Step 1: Fetch raw data
    lineups, contests, results := t.fetchUserData(userID, timeFrame)
    
    // Step 2: Calculate base metrics
    // CRITICAL: Handle concurrent calculations
    metricsChan := make(chan MetricResult, 10)
    var wg sync.WaitGroup
    
    // Launch workers for different metrics
    wg.Add(4)
    go t.calculateROI(lineups, results, metricsChan, &wg)
    go t.calculateSharpe(results, metricsChan, &wg)
    go t.calculateDrawdown(results, metricsChan, &wg)
    go t.calculateWinRate(contests, results, metricsChan, &wg)
    
    // Step 3: Attribution analysis
    // GOTCHA: Ensure time periods align
    attribution := t.performAttribution(lineups, results)
    
    // Step 4: Generate report
    return &PerformanceReport{
        Metrics: collectMetrics(metricsChan),
        Attribution: attribution,
        Recommendations: t.generateRecommendations(metrics, attribution),
    }, nil
}
```

### Integration Points
1. **Optimization Service**: Extend optimizer with portfolio constraints from analytics
2. **WebSocket Hub**: Stream real-time analytics updates to frontend
3. **API Gateway**: Route analytics endpoints to optimization service
4. **Frontend**: New analytics dashboard components consuming WebSocket updates
5. **Database**: Shared Supabase instance with new analytics tables

## Validation Loop

### Level 1: Syntax & Style
```bash
# Navigate to optimization service
cd services/optimization-service

# Run linter
golangci-lint run ./internal/analytics/...

# Run go fmt
go fmt ./internal/analytics/...

# Check for compilation errors
go build ./...
```

### Level 2: Unit Tests
```bash
# Test portfolio optimization algorithms
go test -v ./internal/analytics/portfolio/... -run TestPortfolioOptimization

# Test ML feature extraction
go test -v ./internal/analytics/ml/... -run TestFeatureExtraction

# Test performance calculations
go test -v ./internal/analytics/performance/... -run TestMetrics

# Run with coverage
go test -cover ./internal/analytics/...
```

### Level 3: Integration Tests
```bash
# Test full analytics pipeline
go test -v ./tests/analytics_integration_test.go

# Test WebSocket analytics streaming
go test -v ./tests/analytics_websocket_test.go

# Load test analytics aggregation
go test -v -bench=. ./internal/analytics/performance/...

# Validate ML model accuracy
go test -v ./tests/ml_validation_test.go -run TestModelAccuracy
```

## Final Validation Checklist
- [ ] Portfolio optimization produces valid weight allocations (sum to 1)
- [ ] ML models achieve >75% accuracy on test set
- [ ] Performance metrics match manual calculations
- [ ] Real-time updates arrive within 100ms
- [ ] Analytics queries complete in <500ms for 1 year of data
- [ ] Memory usage stable during extended model training
- [ ] All existing optimization tests still pass

## Anti-Patterns to Avoid
1. **DON'T** load entire user history into memory - use streaming/batching
2. **DON'T** retrain ML models on every request - use scheduled retraining
3. **DON'T** calculate analytics synchronously - use background workers
4. **DON'T** store raw features in database - aggregate and compress
5. **DON'T** ignore numerical stability in portfolio optimization
6. **DON'T** broadcast every analytics update - implement intelligent throttling

## Confidence Score: 8/10

The implementation is comprehensive with clear patterns from existing code. The main challenges are:
- Limited Go ML ecosystem requires careful library selection
- Portfolio optimization numerical stability needs attention
- Real-time analytics performance requires careful optimization

The high confidence comes from:
- Existing patterns for optimization and WebSocket streaming
- Clear integration points with current architecture
- Well-defined validation criteria
- Proven algorithms with Go implementations available