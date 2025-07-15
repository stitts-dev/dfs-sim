# PRD: Monte Carlo Simulation Enhancement

## Executive Summary

Upgrade the Monte Carlo simulation engine from basic variance modeling to sophisticated, sport-specific outcome generation with dynamic correlation matrices and realistic distribution modeling. This enhancement will provide users with accurate contest simulation and risk assessment capabilities that mirror professional DFS analysis tools.

## Problem Statement

### Current State Analysis
**Location**: `services/optimization-service/internal/simulator/monte_carlo.go`

**Critical Issues Identified:**
1. **Oversimplified Distributions**: Lines 210-230 use basic 25% variance assumption for all players
2. **Poor Correlation Implementation**: Lines 188-195 use crude correlation approximation
3. **Static Injury Modeling**: Line 221 uses flat 2% DNP probability regardless of player/sport
4. **Missing Sport-Specific Logic**: No differentiation between NBA, NFL, golf variance patterns
5. **Basic Contest Modeling**: Lines 425-433 assume simple top 20% cash line without GPP nuance
6. **No Weather Integration**: Missing weather impact on player variance and correlations

**Performance Impact:**
- Simulation results don't reflect real contest variance patterns
- Poor prediction of lineup success probability in actual contests
- Missing edge cases (weather games, injury-prone players, etc.)
- Inaccurate ROI calculations leading to poor bankroll decisions

## Success Metrics

### Simulation Accuracy
- **ROI Prediction**: Improve ROI prediction accuracy by 25-30% vs current system
- **Variance Modeling**: Align simulated variance with historical contest results (±5%)
- **Correlation Accuracy**: Match real player correlation patterns from contest data
- **Contest Modeling**: Accurate payout simulation for different contest structures

### Performance Targets
- **Simulation Speed**: Maintain <3 second simulation for 10K iterations
- **Memory Efficiency**: Reduce memory usage by 40% through optimized data structures
- **Scalability**: Support 100K+ simulations for comprehensive analysis
- **Real-time**: Enable live simulation updates during contests

## Technical Specifications

### 1. Sport-Specific Distribution Engine

**Distribution Models:**
```go
type DistributionType string

const (
    Normal         DistributionType = "normal"
    LogNormal      DistributionType = "lognormal" 
    Beta           DistributionType = "beta"
    Gamma          DistributionType = "gamma"
    Exponential    DistributionType = "exponential"
    Custom         DistributionType = "custom"
)

type PlayerDistribution struct {
    DistType        DistributionType `json:"distribution_type"`
    Parameters      []float64        `json:"parameters"`        // Distribution-specific params
    Floor           float64          `json:"floor"`             // Hard minimum
    Ceiling         float64          `json:"ceiling"`           // Hard maximum  
    SkewnessFactor  float64          `json:"skewness"`          // Distribution skew
    KurtosisFactor  float64          `json:"kurtosis"`          // Tail behavior
    InjuryProb      float64          `json:"injury_probability"` // DNP probability
    WeatherImpact   *WeatherEffect   `json:"weather_impact"`    // Weather variance modifier
}

type SportDistributionConfig struct {
    Sport              string                           `json:"sport"`
    DefaultDistribution DistributionType                `json:"default_distribution"`
    PositionOverrides  map[string]DistributionType      `json:"position_overrides"`
    VarianceFactors    map[string]float64               `json:"variance_factors"`
    CorrelationModel   string                           `json:"correlation_model"`
}
```

**Sport-Specific Configurations:**
- **NBA**: Beta distributions for high-usage players, normal for role players
- **NFL**: Log-normal for skill positions, gamma for RBs, exponential for kickers
- **Golf**: Custom distributions with cut probability and round-by-round variance
- **MLB**: Normal distributions with park factor and weather adjustments

### 2. Advanced Correlation Modeling

**Dynamic Correlation Engine:**
```go
type AdvancedCorrelationMatrix struct {
    BaseCorrelations    map[uint]map[uint]float64      `json:"base_correlations"`
    GameStateCorrelations map[string]float64           `json:"gamestate_correlations"`
    WeatherCorrelations map[string]float64             `json:"weather_correlations"`
    InjuryCorrelations  map[uint]float64               `json:"injury_correlations"`
    StackingBonuses     map[string]float64             `json:"stacking_bonuses"`
    TimeDecayFactors    map[string]float64             `json:"time_decay"`
}

type CorrelationContext struct {
    GameState          string      `json:"game_state"`       // "blowout", "close", "overtime"
    WeatherConditions  string      `json:"weather"`          // "wind", "rain", "dome"
    InjuryReports      []uint      `json:"injury_concerns"`  // Players with injury risk
    ContestType        string      `json:"contest_type"`     // "cash", "gpp", "satellite"
    TimeToKickoff      int         `json:"time_to_kickoff"`  // Minutes until game start
}
```

**Context-Aware Correlations:**
- **Game Script**: Correlations change based on projected game flow
- **Weather Impact**: Wind/rain increases correlation between certain position groups
- **Injury Risk**: Increased negative correlation with backup players
- **Contest Type**: Different correlation patterns for cash vs GPP optimization

### 3. Realistic Outcome Generation

**Multi-Layered Simulation:**
```go
type OutcomeGenerator struct {
    BaseProjections    map[uint]float64              `json:"base_projections"`
    DistributionEngine *DistributionEngine           `json:"distribution_engine"`
    CorrelationEngine  *AdvancedCorrelationMatrix    `json:"correlation_engine"`
    EventSimulator     *GameEventSimulator           `json:"event_simulator"`
    ContextFactors     *ContextualFactors            `json:"context_factors"`
}

type GameEventSimulator struct {
    InjuryEvents       *InjurySimulator              `json:"injury_events"`
    WeatherEvents      *WeatherSimulator             `json:"weather_events"`  
    GameScriptEvents   *GameScriptSimulator          `json:"gamescript_events"`
    VolatilityEvents   *VolatilitySimulator          `json:"volatility_events"`
}

type SimulationResult struct {
    PlayerScores       map[uint]float64              `json:"player_scores"`
    EventsOccurred     []GameEvent                   `json:"events_occurred"`
    CorrelationScore   float64                       `json:"correlation_score"`
    VarianceExplained  float64                       `json:"variance_explained"`
    Outliers           []uint                        `json:"outlier_players"`
}
```

### 4. Enhanced Contest Modeling

**Contest Structure Engine:**
```go
type ContestModelConfig struct {
    ContestType        string                        `json:"contest_type"`
    EntryFee           float64                       `json:"entry_fee"`
    TotalEntries       int                           `json:"total_entries"`
    PayoutStructure    []PayoutTier                  `json:"payout_structure"`
    FieldComposition   *FieldComposition             `json:"field_composition"`
    OwnershipModel     *OwnershipModel               `json:"ownership_model"`
}

type FieldComposition struct {
    SharksPercentage   float64                       `json:"sharks_percentage"`    // Top players
    RecsPercentage     float64                       `json:"recs_percentage"`      // Recreational players  
    BeginnersPercentage float64                      `json:"beginners_percentage"` // New players
    SkillAdjustments   map[string]float64            `json:"skill_adjustments"`    // Scoring adjustments by skill
}

type OwnershipModel struct {
    ChalkThreshold     float64                       `json:"chalk_threshold"`      // High ownership level
    LeverageOpportunity float64                      `json:"leverage_opportunity"` // Contrarian edge
    StackOwnership     map[string]float64            `json:"stack_ownership"`      // Team/game stack rates
    PositionBias       map[string]float64            `json:"position_bias"`        // Position popularity bias
}
```

## Implementation Plan

### Phase 1: Distribution Engine Overhaul (Week 1-2)
1. **Distribution Library**: Implement sport-specific distribution models
2. **Parameter Estimation**: Build historical data analysis for distribution fitting
3. **Validation Framework**: Backtesting against known contest results  
4. **Performance Optimization**: Efficient random number generation

### Phase 2: Advanced Correlation Modeling (Week 3-4)
1. **Context Engine**: Build dynamic correlation adjustment system
2. **Historical Analysis**: Mine correlation patterns from contest data
3. **Real-time Factors**: Integrate weather and injury impact on correlations
4. **Validation**: Compare simulated correlations with actual contest patterns

### Phase 3: Event Simulation Framework (Week 5-6)
1. **Event Generators**: Build injury, weather, and volatility simulators
2. **Game Script Modeling**: Implement blowout/close game correlation changes
3. **Integration**: Combine event simulation with outcome generation
4. **Testing**: Validate event probabilities against historical data

### Phase 4: Contest Modeling Enhancement (Week 7-8)
1. **Payout Structures**: Implement accurate GPP and cash game modeling
2. **Field Composition**: Model different skill level distributions in contests
3. **Ownership Integration**: Advanced ownership impact on contest outcomes
4. **ROI Calculation**: Precise expected value calculations

## API Enhancements

### Enhanced Simulation Request
```go
type SimulationConfigV2 struct {
    // Existing fields...
    SportConfig            SportDistributionConfig   `json:"sport_config"`
    ContestModel          ContestModelConfig        `json:"contest_model"`
    CorrelationSettings   CorrelationSettings       `json:"correlation_settings"`
    EventSimulation       EventSimulationConfig     `json:"event_simulation"`
    AdvancedAnalytics     bool                      `json:"enable_advanced_analytics"`
    CustomDistributions   map[uint]PlayerDistribution `json:"custom_distributions"`
}

type CorrelationSettings struct {
    UseAdvancedCorrelations bool                    `json:"use_advanced_correlations"`
    WeatherImpact          bool                     `json:"include_weather_impact"`
    GameScriptAdjustments  bool                     `json:"gamescript_adjustments"`
    InjuryCorrelations     bool                     `json:"injury_correlations"`
    TimeDecayEnabled       bool                     `json:"time_decay_enabled"`
}
```

### Enhanced Simulation Response
```go
type SimulationResultV2 struct {
    // Existing fields...
    AdvancedMetrics       AdvancedSimulationMetrics  `json:"advanced_metrics"`
    DistributionAnalysis  DistributionAnalysis       `json:"distribution_analysis"`
    CorrelationBreakdown  CorrelationBreakdown       `json:"correlation_breakdown"`
    EventAnalysis         EventAnalysis              `json:"event_analysis"`
    ContestProjections    ContestProjections         `json:"contest_projections"`
    RiskAssessment        RiskAssessment             `json:"risk_assessment"`
}

type AdvancedSimulationMetrics struct {
    DistributionFit       float64                    `json:"distribution_fit_score"`
    CorrelationAccuracy   float64                    `json:"correlation_accuracy"`
    OutlierRate          float64                     `json:"outlier_rate"`
    VarianceExplained    float64                     `json:"variance_explained"`
    ModelConfidence      float64                     `json:"model_confidence"`
}

type ContestProjections struct {
    ExpectedFinish       float64                     `json:"expected_finish"`
    FinishDistribution   []float64                   `json:"finish_distribution"`
    CashProbability      float64                     `json:"cash_probability"`
    ROIProjection        float64                     `json:"roi_projection"`
    RiskAdjustedROI      float64                     `json:"risk_adjusted_roi"`
    BincoopProbability   map[string]float64          `json:"binCoop_probability"`   // "1st", "top5", "top10", etc.
}
```

## Integration Points

### With Core Optimization (PRD-1)
- **Distribution Parameters**: Use player analytics variance in simulation distributions
- **Optimization Feedback**: Simulation results inform optimization strategy selection
- **Risk Metrics**: Simulation variance data used in optimization constraints

### With AI Recommendations (PRD-2)  
- **Confidence Scoring**: Simulation variance informs recommendation confidence levels
- **Risk Communication**: Simulation results help explain recommendation risk/reward
- **Portfolio Analysis**: Simulation across multiple lineups for portfolio recommendations

### With Golf Strategy (PRD-4)
- **Cut Modeling**: Special simulation logic for golf cut probability
- **Round-by-Round**: Simulate progressive tournament results and adjustments
- **Weather Integration**: Golf-specific weather impact on scoring distributions

## Advanced Features

### 1. Machine Learning Distribution Fitting
```go
type MLDistributionFitter struct {
    HistoricalData       []PlayerPerformance        `json:"historical_data"`
    FeatureExtractor     *FeatureExtractor          `json:"feature_extractor"`
    ModelEnsemble        []DistributionModel        `json:"model_ensemble"`
    ValidationFramework  *CrossValidation           `json:"validation"`
}
```

### 2. Real-time Adaptation Engine  
```go
type AdaptiveSimulator struct {
    LiveDataStream       *RealTimeDataStream        `json:"live_data"`
    ParameterUpdater     *ParameterUpdater          `json:"parameter_updater"`
    ModelRecalibration   *ModelRecalibrator         `json:"recalibration"`
    PerformanceMonitor   *PerformanceMonitor        `json:"performance_monitor"`
}
```

### 3. Portfolio Simulation Engine
```go
type PortfolioSimulator struct {
    MultiLineupSimulation *MultiLineupEngine        `json:"multi_lineup"`
    CorrelationMatrixPortfolio map[string]float64   `json:"portfolio_correlations"`
    RiskMetrics           *PortfolioRiskMetrics     `json:"risk_metrics"`
    OptimalAllocation     *AllocationOptimizer      `json:"allocation_optimizer"`
}
```

## Risk Mitigation

### Computational Risks
- **Performance Monitoring**: Automatic fallback to simpler models if performance degrades
- **Memory Management**: Efficient data structures for large-scale simulations  
- **Parallel Processing**: Multi-threaded simulation with proper resource management

### Model Accuracy Risks
- **Continuous Validation**: Daily backtesting against previous day's contest results
- **Parameter Drift Detection**: Monitor and alert on significant parameter changes
- **Fallback Models**: Multiple model approaches with automatic selection

### Data Quality Risks
- **Input Validation**: Robust checking of projection and player data
- **Outlier Detection**: Automatic flagging of unusual simulation results
- **Historical Verification**: Cross-reference patterns with known historical data

## Success Validation

### Automated Testing
- **Distribution Accuracy**: Daily validation of simulated vs actual score distributions
- **Correlation Testing**: Weekly analysis of simulated vs actual player correlations
- **ROI Accuracy**: Monthly backtesting of ROI predictions vs actual results

### Performance Metrics
- **Simulation Speed**: Continuous monitoring of simulation performance
- **Memory Usage**: Resource utilization tracking and optimization
- **Model Confidence**: Automated scoring of model prediction accuracy

## Future Enhancements

### Advanced Statistical Methods
- **Bayesian Inference**: Dynamic parameter updating based on new data
- **Copula Models**: Advanced correlation structure modeling
- **Extreme Value Theory**: Better modeling of outlier performances

### Real-time Integration
- **Live Simulation**: Real-time simulation updates during contests
- **Dynamic Rebalancing**: Automatic lineup adjustment recommendations
- **In-game Adaptation**: Simulation parameter updates based on live scoring

---

**Estimated Timeline**: 8 weeks
**Resource Requirements**: 2 data scientists, 1 backend engineer, 1 performance engineer
**Dependencies**: Historical contest data, real-time data pipeline, enhanced compute infrastructure  
**Risk Level**: Medium (computational complexity, model validation requirements)