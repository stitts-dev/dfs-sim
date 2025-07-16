# DataGolf API Integration & Optimization Algorithm Enhancement PRD

**Project:** SaberSim Clone DFS Platform  
**Focus:** Architecture & Algorithm Enhancement  
**Priority:** High  
**Date:** 2025-07-15

## 🎯 Executive Summary

This PRD outlines a comprehensive architecture for integrating DataGolf API ($270/year) into our existing DFS golf optimization platform, with particular emphasis on enhancing optimization algorithms and creating a scalable foundation for advanced golf analytics. The integration leverages our robust microservices architecture while introducing sophisticated strokes gained analytics, course fit modeling, and AI-powered optimization strategies.

## 🏗️ Current Architecture Analysis

### Microservices Foundation
Our current system demonstrates excellent architectural patterns:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Frontend      │    │   API Gateway    │    │ User Service    │
│   (React)       │◄───┤   (Go + Nginx)   │◄───┤   (Go + Auth)   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ Sports Data     │    │ Optimization     │    │ Realtime        │
│ Service (Golf)  │◄───┤ Service (Core)   │◄───┤ Service (Live)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │ AI Recommendations│
                       │ Service (Claude)  │
                       └──────────────────┘
```

### Current Golf Optimization Capabilities

**Existing Algorithm Strengths:**
- 6 tournament strategies (Win, Top5, Top10, Top25, Cut, Balanced)
- Advanced cut probability modeling with weather/course factors
- Dynamic programming optimization with strategy-aware configurations
- Golf-specific correlation algorithms (tee times, course history)
- Monte Carlo simulation with correlation-aware outcome generation
- Real-time late swap optimization

**Current Limitations:**
- Basic performance metrics (position, score) vs. advanced analytics
- Limited course fit modeling (boolean flags vs. quantitative models)
- Weather consideration as simple flags vs. sophisticated impact modeling
- Manual correlation weights vs. data-driven relationship modeling

## 🚀 DataGolf Integration Architecture

### Phase 1: Enhanced Data Infrastructure

#### 1.1 Data Provider Architecture
```go
// Enhanced provider interface supporting advanced metrics
type GolfDataProvider interface {
    // Standard methods
    GetPlayers(sport types.Sport, date string) ([]types.PlayerData, error)
    GetCurrentTournament() (*GolfTournamentData, error)
    GetTournamentSchedule() ([]GolfTournamentData, error)
    
    // DataGolf-specific advanced methods
    GetStrokesGainedData(playerID string, tournamentID string) (*StrokesGainedMetrics, error)
    GetCourseAnalytics(courseID string) (*CourseAnalytics, error)
    GetPreTournamentPredictions(tournamentID string) (*TournamentPredictions, error)
    GetLiveTournamentData(tournamentID string) (*LiveTournamentData, error)
    GetPlayerCourseHistory(playerID, courseID string) (*PlayerCourseHistory, error)
    GetWeatherImpactData(tournamentID string) (*WeatherImpactAnalysis, error)
}
```

#### 1.2 Enhanced Data Models
```go
type StrokesGainedMetrics struct {
    PlayerID           int64     `json:"player_id"`
    TournamentID       string    `json:"tournament_id"`
    SGOffTheTee       float64   `json:"sg_off_the_tee"`
    SGApproach        float64   `json:"sg_approach"`
    SGAroundTheGreen  float64   `json:"sg_around_the_green"`
    SGPutting         float64   `json:"sg_putting"`
    SGTotal           float64   `json:"sg_total"`
    Consistency       float64   `json:"consistency_rating"`
    VolatilityIndex   float64   `json:"volatility_index"`
    UpdatedAt         time.Time `json:"updated_at"`
}

type CourseAnalytics struct {
    CourseID              string                 `json:"course_id"`
    DifficultyRating      float64               `json:"difficulty_rating"`
    Length                int                   `json:"length"`
    Par                   int                   `json:"par"`
    PlayerTypeAdvantages  map[string]float64    `json:"player_type_advantages"`
    WeatherSensitivity    map[string]float64    `json:"weather_sensitivity"`
    HistoricalScoring     ScoreDistribution     `json:"historical_scoring"`
    KeyHoles              []int                 `json:"key_holes"`
    SkillPremiums         SkillPremiumWeights   `json:"skill_premiums"`
}

type SkillPremiumWeights struct {
    DrivingDistance    float64 `json:"driving_distance"`
    DrivingAccuracy    float64 `json:"driving_accuracy"`
    ApproachPrecision  float64 `json:"approach_precision"`
    ShortGameSkill     float64 `json:"short_game_skill"`
    PuttingConsistency float64 `json:"putting_consistency"`
}
```

### Phase 2: Algorithm Enhancement Architecture

#### 2.1 Strokes Gained Optimization Engine
```go
type StrokesGainedOptimizer struct {
    // Core components
    dataProvider      GolfDataProvider
    courseModelEngine *CourseModelEngine
    weatherEngine     *WeatherImpactEngine
    correlationEngine *AdvancedCorrelationEngine
    
    // Configuration
    optimizationWeights map[string]float64
    strategyProfiles    map[string]*StrategyProfile
    volatilityTargets   map[string]float64
}

type StrategyProfile struct {
    Name                string             `json:"name"`
    SGWeights           SGCategoryWeights  `json:"sg_weights"`
    VolatilityTolerance float64            `json:"volatility_tolerance"`
    CorrelationTargets  CorrelationTargets `json:"correlation_targets"`
    CutProbabilityMin   float64            `json:"cut_probability_min"`
    UpstideTargets      UpsideTargets      `json:"upside_targets"`
}

type SGCategoryWeights struct {
    OffTheTee      float64 `json:"off_the_tee"`       // e.g., 1.2 for tournaments favoring bombers
    Approach       float64 `json:"approach"`          // e.g., 1.0 baseline importance
    AroundTheGreen float64 `json:"around_the_green"`  // e.g., 0.8 for easier short game courses
    Putting        float64 `json:"putting"`           // e.g., 0.9 for consistent greens
}
```

#### 2.2 Course Fit Modeling System
```go
type CourseModelEngine struct {
    historicalPerformance map[string]*PlayerCourseHistory
    courseFeatureAnalysis map[string]*CourseFeatures
    playerProfiles        map[string]*PlayerProfile
    fitCalculator         *CourseFitCalculator
}

type CourseFitCalculator struct {
    // 5-attribute model from DataGolf research
    attributeWeights map[string]float64
    
    // Advanced modeling components
    nonLinearAdjustments map[string]func(float64) float64
    interactionEffects   map[string]map[string]float64
    weatherAdjustments   map[string]*WeatherCoefficients
}

func (cfc *CourseFitCalculator) CalculateCourseFit(
    playerProfile *PlayerProfile,
    courseFeatures *CourseFeatures,
    weatherConditions *WeatherConditions,
) (*CourseFitResult, error) {
    // Implement sophisticated course fit calculation
    baseScore := cfc.calculateBaseAttributeScore(playerProfile, courseFeatures)
    weatherAdjustment := cfc.calculateWeatherImpact(playerProfile, weatherConditions)
    interactionBonus := cfc.calculateInteractionEffects(playerProfile, courseFeatures)
    
    return &CourseFitResult{
        FitScore:          baseScore + weatherAdjustment + interactionBonus,
        ConfidenceLevel:   cfc.calculateConfidence(playerProfile, courseFeatures),
        KeyAdvantages:     cfc.identifyKeyAdvantages(playerProfile, courseFeatures),
        RiskFactors:       cfc.identifyRiskFactors(playerProfile, courseFeatures),
    }, nil
}
```

#### 2.3 Advanced Correlation Engine
```go
type AdvancedCorrelationEngine struct {
    // Multi-dimensional correlation modeling
    teeTimeCorrelations    *TeeTimeCorrelationModel
    weatherCorrelations    *WeatherCorrelationModel
    skillCorrelations      *SkillBasedCorrelationModel
    tournamentCorrelations *TournamentStateCorrelationModel
    
    // Dynamic correlation adjustment
    realTimeAdjuster      *RealTimeCorrelationAdjuster
    historicalValidator   *CorrelationHistoricalValidator
}

type TeeTimeCorrelationModel struct {
    // Wave-based correlations (AM/PM)
    waveAdvantageCorr     map[string]float64
    groupPlayCorr         map[string]float64
    weatherExposureCorr   map[string]float64
    
    // Dynamic tee time advantage calculation
    windPatternAnalysis   *WindPatternAnalyzer
    temperatureEffects    *TemperatureEffectAnalyzer
}

type SkillBasedCorrelationModel struct {
    // Players with similar skill profiles correlation
    sgCategoryCorrelations map[string]map[string]float64
    courseTypeCorrelations map[string]map[string]float64
    playStyleCorrelations  map[string]map[string]float64
    
    // Contrarian correlation identification
    ownershipAntiCorrelations map[string]float64
}
```

#### 2.4 Enhanced Monte Carlo Engine
```go
type EnhancedMonteCarloEngine struct {
    // Core simulation components
    distributionEngine    *SGBasedDistributionEngine
    correlationMatrix     *DynamicCorrelationMatrix
    scenarioGenerator     *TournamentScenarioGenerator
    
    // Advanced simulation features
    volatilityModeling    *VolatilityModelingEngine
    cutLineSimulation     *CutLineSimulationEngine
    weatherImpactSim      *WeatherImpactSimulator
    
    // Performance optimization
    parallelWorkers       int
    batchOptimization     *BatchOptimizationEngine
}

func (emce *EnhancedMonteCarloEngine) RunAdvancedSimulation(
    lineup *models.Lineup,
    tournament *models.GolfTournament,
    scenarios *TournamentScenarios,
) (*AdvancedSimulationResult, error) {
    // Generate strokes gained-based score distributions
    playerDistributions := emce.distributionEngine.GenerateDistributions(lineup, tournament)
    
    // Apply course-specific and weather-specific adjustments
    adjustedDistributions := emce.applyContextualAdjustments(playerDistributions, scenarios)
    
    // Run correlation-aware simulation
    results := emce.simulateWithAdvancedCorrelations(adjustedDistributions, scenarios)
    
    return &AdvancedSimulationResult{
        ROIProjection:         results.ROI,
        VolatilityMetrics:     results.Volatility,
        ScenarioBreakdown:     results.ScenarioPerformance,
        CutLineAnalysis:       results.CutLineResults,
        WeatherSensitivity:    results.WeatherImpact,
        OptimalityScore:       results.OptimalityRating,
        ConfidenceIntervals:   results.ConfidenceRanges,
    }, nil
}
```

### Phase 3: AI Enhancement Architecture

#### 3.1 Enhanced AI Prompt Builder
```go
type DataGolfAIPromptBuilder struct {
    contextBuilder        *ContextualDataBuilder
    metricAnalyzer        *SGMetricAnalyzer
    courseFitAnalyzer     *CourseFitAnalyzer
    correlationAnalyzer   *CorrelationInsightAnalyzer
    strategyRecommender   *StrategyRecommendationEngine
}

func (dgpb *DataGolfAIPromptBuilder) BuildAdvancedGolfPrompt(
    tournament *models.GolfTournament,
    players []*models.GolfPlayer,
    constraints *models.OptimizationConstraints,
    dataGolfInsights *DataGolfInsights,
) (*AIPromptContext, error) {
    
    return &AIPromptContext{
        TournamentContext: dgpb.buildTournamentContext(tournament, dataGolfInsights),
        PlayerAnalysis:    dgpb.buildPlayerAnalysis(players, dataGolfInsights),
        CourseFitMatrix:   dgpb.buildCourseFitMatrix(players, tournament, dataGolfInsights),
        WeatherAnalysis:   dgpb.buildWeatherAnalysis(tournament, dataGolfInsights),
        CorrelationInsights: dgpb.buildCorrelationInsights(players, dataGolfInsights),
        StrategyRecommendations: dgpb.buildStrategyRecommendations(constraints, dataGolfInsights),
        MetricPrioritization: dgpb.buildMetricPrioritization(tournament, dataGolfInsights),
    }, nil
}
```

#### 3.2 Real-Time Optimization Adjustment
```go
type RealTimeOptimizationEngine struct {
    // Live data processing
    liveDataProcessor     *LiveTournamentProcessor
    cutLinePredictor      *DynamicCutLinePredictor
    weatherMonitor        *WeatherImpactMonitor
    
    // Dynamic recommendation engine
    lateSwapOptimizer     *LateSwapOptimizer
    riskAdjuster          *DynamicRiskAdjuster
    correlationUpdater    *LiveCorrelationUpdater
}

func (rtoe *RealTimeOptimizationEngine) ProcessLiveUpdate(
    liveData *LiveTournamentData,
    currentLineups []*models.Lineup,
) (*LiveOptimizationRecommendations, error) {
    
    // Analyze current tournament state
    tournamentState := rtoe.liveDataProcessor.AnalyzeTournamentState(liveData)
    
    // Update cut line predictions
    updatedCutLine := rtoe.cutLinePredictor.UpdateCutLinePrediction(liveData)
    
    // Generate swap recommendations
    swapRecommendations := rtoe.lateSwapOptimizer.GenerateSwapRecommendations(
        currentLineups, tournamentState, updatedCutLine)
    
    return &LiveOptimizationRecommendations{
        CutLineUpdate:        updatedCutLine,
        SwapRecommendations:  swapRecommendations,
        RiskAdjustments:      rtoe.calculateRiskAdjustments(tournamentState),
        CorrelationUpdates:   rtoe.updateCorrelations(liveData),
        ConfidenceMetrics:    rtoe.calculateConfidenceMetrics(liveData),
    }, nil
}
```

## 🔧 Database Schema Enhancements

### Advanced Analytics Tables
```sql
-- Strokes gained historical data
CREATE TABLE strokes_gained_history (
    id SERIAL PRIMARY KEY,
    player_id INTEGER REFERENCES golf_players(id),
    tournament_id INTEGER REFERENCES golf_tournaments(id),
    sg_off_the_tee DECIMAL(6,3),
    sg_approach DECIMAL(6,3), 
    sg_around_the_green DECIMAL(6,3),
    sg_putting DECIMAL(6,3),
    sg_total DECIMAL(6,3),
    consistency_rating DECIMAL(4,3),
    volatility_index DECIMAL(4,3),
    round_number INTEGER,
    course_conditions JSONB,
    weather_conditions JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Course analytics and modeling
CREATE TABLE course_analytics (
    id SERIAL PRIMARY KEY,
    course_id INTEGER REFERENCES golf_courses(id),
    difficulty_rating DECIMAL(4,2),
    skill_premiums JSONB, -- driving_distance, accuracy, approach, short_game, putting
    weather_sensitivity JSONB,
    historical_scoring JSONB,
    key_holes INTEGER[],
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Player course fit modeling
CREATE TABLE player_course_fits (
    player_id INTEGER REFERENCES golf_players(id),
    course_id INTEGER REFERENCES golf_courses(id),
    fit_score DECIMAL(4,3),
    confidence_level DECIMAL(3,2),
    key_advantages TEXT[],
    risk_factors TEXT[],
    historical_performance JSONB,
    weather_adjustments JSONB,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (player_id, course_id)
);

-- Advanced correlation tracking
CREATE TABLE correlation_matrices (
    id SERIAL PRIMARY KEY,
    tournament_id INTEGER REFERENCES golf_tournaments(id),
    correlation_type VARCHAR(50), -- 'tee_time', 'weather', 'skill_based', 'cut_line'
    correlation_data JSONB,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Optimization algorithm performance tracking
CREATE TABLE algorithm_performance (
    id SERIAL PRIMARY KEY,
    algorithm_version VARCHAR(50),
    tournament_id INTEGER REFERENCES golf_tournaments(id),
    strategy_type VARCHAR(50),
    performance_metrics JSONB, -- ROI, accuracy, volatility, etc.
    lineup_analysis JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 📊 Algorithm Enhancement Roadmap

### Phase 1: Foundation Enhancement (Weeks 1-4)
- **DataGolf API Integration**: Replace RapidAPI with professional-grade data
- **Strokes Gained Implementation**: Core SG metrics in optimization algorithms
- **Course Fit Modeling**: Basic 5-attribute course fit calculation
- **Enhanced Data Models**: Support for advanced metrics and analytics

### Phase 2: Advanced Optimization (Weeks 5-8)
- **Multi-Strategy Optimization**: Enhanced strategy profiles with SG weighting
- **Advanced Correlation Engine**: Skill-based and weather-based correlations
- **Volatility Modeling**: Player consistency and volatility integration
- **Cut Line Sophistication**: Dynamic cut probability with real-time updates

### Phase 3: AI Integration (Weeks 9-12)
- **Enhanced AI Prompts**: DataGolf insights in AI recommendations
- **Real-Time Optimization**: Live tournament state optimization
- **Scenario Modeling**: Weather and condition-based scenario analysis
- **Performance Learning**: Algorithm performance tracking and improvement

### Phase 4: Advanced Analytics (Weeks 13-16)
- **Proprietary Metrics**: Custom performance indicators based on outcomes
- **Ensemble Modeling**: Multiple algorithm approaches with weighted outcomes
- **Market Inefficiency Detection**: Identify pricing/ownership advantages
- **Predictive Course Modeling**: Future tournament performance prediction

## 🎯 Optimization Algorithm Expansion Opportunities

### 1. Multi-Objective Optimization
```go
type MultiObjectiveOptimizer struct {
    objectives []OptimizationObjective
    weights    map[string]float64
    constraints []OptimizationConstraint
}

type OptimizationObjective struct {
    Name       string
    Calculator func(*models.Lineup, *TournamentContext) float64
    Weight     float64
    Target     float64
}

// Objectives could include:
// - Expected value maximization
// - Variance minimization  
// - Cut probability optimization
// - Correlation target achievement
// - Ownership leverage maximization
// - Weather risk minimization
```

### 2. Machine Learning Integration
```go
type MLEnhancedOptimizer struct {
    playerPerformanceML  *PlayerPerformancePredictor
    courseFitML          *CourseFitPredictor
    correlationML        *CorrelationPredictor
    ownershipML          *OwnershipPredictor
    
    // Ensemble modeling
    ensembleWeights      map[string]float64
    modelValidator       *CrossValidationEngine
}

// ML Model Integration Points:
// - Player performance prediction based on recent form
// - Course fit prediction using historical and contextual data
// - Dynamic correlation prediction based on tournament conditions
// - Ownership prediction for leverage optimization
// - Cut line prediction using real-time data
```

### 3. Advanced Simulation Techniques
```go
type AdvancedSimulationEngine struct {
    // Monte Carlo variants
    standardMonteCarlo    *MonteCarloEngine
    antitheticMonteCarlo  *AntitheticVariateEngine
    stratifiedMonteCarlo  *StratifiedSamplingEngine
    
    // Alternative simulation methods
    bootstrapSimulation   *BootstrapSimulationEngine
    bayesianSimulation    *BayesianSimulationEngine
    neuralSimulation      *NeuralNetworkSimulationEngine
}
```

## 🔄 Implementation Strategy

### Microservices Enhancement
- **Sports Data Service**: Enhanced with DataGolf provider and advanced analytics
- **Optimization Service**: Expanded with sophisticated algorithm variants
- **AI Recommendations Service**: Enhanced with DataGolf insights and advanced prompting
- **Realtime Service**: Enhanced with live DataGolf data and dynamic optimization
- **Analytics Service**: New service for performance tracking and ML model management

### Configuration-Driven Architecture
```go
type OptimizationConfig struct {
    Provider          string                 `json:"provider"`           // "datagolf", "rapidapi"
    AlgorithmVersion  string                 `json:"algorithm_version"`  // "v1_basic", "v2_sg", "v3_ml"
    StrategyProfiles  map[string]Strategy    `json:"strategy_profiles"`
    CorrelationConfig CorrelationConfig      `json:"correlation_config"`
    CachingStrategy   CachingStrategy        `json:"caching_strategy"`
    MLModelConfig     MLModelConfig          `json:"ml_model_config"`
}
```

### Scalability Considerations
- **Horizontal Scaling**: Stateless services with Redis clustering
- **Algorithm Versioning**: Support multiple optimization algorithms simultaneously
- **A/B Testing**: Framework for testing algorithm improvements
- **Performance Monitoring**: Comprehensive metrics for optimization quality
- **Graceful Degradation**: Fallback mechanisms for service failures

## 📈 Success Metrics & KPIs

### Technical Metrics
- **API Reliability**: 99.9% uptime with DataGolf integration
- **Response Time**: <200ms for optimization requests
- **Cache Hit Rate**: >85% for frequently accessed data
- **Algorithm Accuracy**: 15-25% improvement in ROI vs. baseline

### Business Metrics
- **User Engagement**: 40% increase in advanced optimization usage
- **Retention**: 25% improvement in monthly active users
- **Revenue**: $50-100/user/month from premium features
- **Competitive Advantage**: Top 10% performance in major tournaments

### Algorithm Performance Metrics
- **Prediction Accuracy**: Track actual vs. predicted player performance
- **Optimization Quality**: Measure lineup ROI across different strategies
- **Correlation Validation**: Validate correlation predictions with actual outcomes
- **Cut Line Accuracy**: Track cut line prediction accuracy

## 🎯 Future Expansion Opportunities

### Advanced Features
- **Multi-Tournament Optimization**: Optimize across multiple tournaments simultaneously
- **Bankroll Management**: Integrate bankroll and variance management
- **Social Features**: Community-driven insights and strategy sharing
- **Mobile Optimization**: Native mobile app with real-time notifications
- **White-Label Solutions**: B2B platform for other DFS companies

### Data Science Initiatives
- **Proprietary Metrics**: Develop custom performance indicators
- **Market Research**: Analyze DFS market inefficiencies
- **Weather Modeling**: Advanced weather impact prediction
- **Injury Prediction**: Early injury risk detection using performance data
- **Course Design Analysis**: How course setup affects player performance

## 📋 DataGolf API Implementation Details

### API Key & Authentication
- **API Key**: `ec1ac262bd9c6286beafa521b01f`
- **Cost**: $270/year professional subscription
- **Authentication**: Simple query parameter authentication
- **Rate Limits**: Professional tier limits (monitor usage)

### Core API Endpoints Integration

#### 1. General Use Endpoints
```go
type DataGolfClient struct {
    APIKey     string
    BaseURL    string
    HTTPClient *http.Client
}

// Player List & IDs - Foundation for player matching
func (dg *DataGolfClient) GetPlayerList() (*PlayerListResponse, error) {
    url := fmt.Sprintf("%s/get-player-list?file_format=json&key=%s", dg.BaseURL, dg.APIKey)
    // Returns players with DataGolf IDs, country, amateur status
}

// Tour Schedules - Tournament discovery and planning
func (dg *DataGolfClient) GetTourSchedule(tour string) (*ScheduleResponse, error) {
    url := fmt.Sprintf("%s/get-schedule?tour=%s&file_format=json&key=%s", dg.BaseURL, tour, dg.APIKey)
    // Tours: pga, euro, kft, alt (LIV)
}

// Field Updates - Real-time field changes, WDs, tee times, salaries
func (dg *DataGolfClient) GetFieldUpdates(tour string) (*FieldUpdatesResponse, error) {
    url := fmt.Sprintf("%s/field-updates?tour=%s&file_format=json&key=%s", dg.BaseURL, tour, dg.APIKey)
    // Critical for DFS - WDs, Monday qualifiers, salary updates
}
```

#### 2. Model Predictions - Core Optimization Data
```go
// Pre-Tournament Predictions - Primary optimization input
func (dg *DataGolfClient) GetPreTournamentPredictions(tour string) (*PreTournamentResponse, error) {
    url := fmt.Sprintf("%s/preds/pre-tournament?tour=%s&file_format=json&key=%s", dg.BaseURL, tour, dg.APIKey)
    // Returns: win, top 5, top 10, top 20, make cut probabilities
    // Baseline + course history models available
}

// Player Skill Decompositions - Advanced optimization foundation
func (dg *DataGolfClient) GetPlayerSkillDecompositions(tour string) (*SkillDecompositionResponse, error) {
    url := fmt.Sprintf("%s/preds/player-decompositions?tour=%s&file_format=json&key=%s", dg.BaseURL, tour, dg.APIKey)
    // Detailed strokes-gained breakdown for course fit modeling
}

// Skill Ratings - Historical performance metrics
func (dg *DataGolfClient) GetSkillRatings() (*SkillRatingsResponse, error) {
    url := fmt.Sprintf("%s/preds/skill-ratings?display=value&file_format=json&key=%s", dg.BaseURL, dg.APIKey)
    // SG categories, ranks, minimum 30 rounds requirement
}

// Fantasy Projections - Direct DFS integration
func (dg *DataGolfClient) GetFantasyProjections(tour, site, slate string) (*FantasyProjectionsResponse, error) {
    url := fmt.Sprintf("%s/preds/fantasy-projection-defaults?tour=%s&site=%s&slate=%s&file_format=json&key=%s", 
        dg.BaseURL, tour, site, slate, dg.APIKey)
    // Sites: draftkings, fanduel, yahoo
    // Slates: main, showdown, showdown_late, weekend, captain
}
```

#### 3. Live Model Endpoints - Real-time Optimization
```go
// Live Model Predictions - Tournament state optimization
func (dg *DataGolfClient) GetLiveModelPredictions(tour string) (*LiveModelResponse, error) {
    url := fmt.Sprintf("%s/preds/in-play?tour=%s&file_format=json&key=%s", dg.BaseURL, tour, dg.APIKey)
    // Updates every 5 minutes during tournaments
}

// Live Tournament Stats - Performance tracking
func (dg *DataGolfClient) GetLiveTournamentStats(stats, round string) (*LiveStatsResponse, error) {
    url := fmt.Sprintf("%s/preds/live-tournament-stats?stats=%s&round=%s&file_format=json&key=%s", 
        dg.BaseURL, stats, round, dg.APIKey)
    // SG categories, traditional stats, real-time updates
}

// Live Hole Scoring - Detailed course analysis
func (dg *DataGolfClient) GetLiveHoleStats(tour string) (*LiveHoleStatsResponse, error) {
    url := fmt.Sprintf("%s/preds/live-hole-stats?tour=%s&file_format=json&key=%s", dg.BaseURL, tour, dg.APIKey)
    // Hole-by-hole scoring distributions by tee time wave
}
```

#### 4. Historical Data - Algorithm Training & Validation
```go
// Historical Raw Data - Algorithm backtesting
func (dg *DataGolfClient) GetHistoricalRounds(tour, eventID, year string) (*HistoricalRoundsResponse, error) {
    url := fmt.Sprintf("%s/historical-raw-data/rounds?tour=%s&event_id=%s&year=%s&file_format=json&key=%s", 
        dg.BaseURL, tour, eventID, year, dg.APIKey)
    // 22 global tours, comprehensive historical data
}

// Historical DFS Data - Performance validation
func (dg *DataGolfClient) GetHistoricalDFSData(tour, site, eventID, year string) (*HistoricalDFSResponse, error) {
    url := fmt.Sprintf("%s/historical-dfs-data/points?tour=%s&site=%s&event_id=%s&year=%s&file_format=json&key=%s", 
        dg.BaseURL, tour, site, eventID, year, dg.APIKey)
    // Salaries, ownership, fantasy points for algorithm validation
}
```

### Data Models Enhancement

#### Enhanced Player Models
```go
type DataGolfPlayer struct {
    DataGolfID        int64   `json:"dg_id"`
    FirstName         string  `json:"first_name"`
    LastName          string  `json:"last_name"`
    Country           string  `json:"country"`
    AmateurStatus     string  `json:"am"`
    
    // Skill ratings (when available)
    SGOffTheTee       *float64 `json:"sg_ott,omitempty"`
    SGApproach        *float64 `json:"sg_app,omitempty"`
    SGAroundGreen     *float64 `json:"sg_arg,omitempty"`
    SGPutting         *float64 `json:"sg_putt,omitempty"`
    SGTotal           *float64 `json:"sg_total,omitempty"`
    
    // Tournament predictions
    WinProbability    *float64 `json:"win_prob,omitempty"`
    Top5Probability   *float64 `json:"top5_prob,omitempty"`
    Top10Probability  *float64 `json:"top10_prob,omitempty"`
    Top20Probability  *float64 `json:"top20_prob,omitempty"`
    CutProbability    *float64 `json:"mc_prob,omitempty"`
    
    // Fantasy projections
    FantasyProjection *float64 `json:"fantasy_proj,omitempty"`
    Salary           *int64   `json:"salary,omitempty"`
    Ownership        *float64 `json:"ownership,omitempty"`
}
```

#### Tournament Context Models
```go
type DataGolfTournament struct {
    EventID           string              `json:"event_id"`
    EventName         string              `json:"event_name"`
    CourseName        string              `json:"course"`
    CourseID          string              `json:"course_id"`
    Tour              string              `json:"tour"`
    StartDate         time.Time           `json:"date"`
    Location          TournamentLocation  `json:"location"`
    
    // Course characteristics
    Difficulty        *float64            `json:"difficulty,omitempty"`
    Length           *int                `json:"length,omitempty"`
    Par              *int                `json:"par,omitempty"`
    
    // Weather conditions
    WeatherConditions *WeatherData        `json:"weather,omitempty"`
    
    // Field information
    FieldSize        int                 `json:"field_size"`
    CutLine          *int                `json:"cut_line,omitempty"`
}

type TournamentLocation struct {
    City        string  `json:"city"`
    Country     string  `json:"country"`
    Latitude    float64 `json:"latitude"`
    Longitude   float64 `json:"longitude"`
}
```

### Implementation Priority Matrix

#### Phase 1: Core Integration (Weeks 1-2)
**High Priority**
- [ ] Player List & IDs integration for player matching
- [ ] Pre-Tournament Predictions for optimization core
- [ ] Field Updates for real-time field changes
- [ ] Fantasy Projections for DFS-specific data

**Medium Priority**
- [ ] Tour Schedules for tournament planning
- [ ] Skill Decompositions for advanced modeling
- [ ] Live Model Predictions for real-time updates

#### Phase 2: Advanced Features (Weeks 3-4)
**High Priority**
- [ ] Live Tournament Stats integration
- [ ] Skill Ratings for historical analysis
- [ ] Historical DFS Data for backtesting

**Medium Priority**
- [ ] Live Hole Stats for detailed analysis
- [ ] Historical Raw Data for comprehensive modeling
- [ ] Betting Tools integration (future feature)

### Data Synchronization Strategy

#### Real-time Data Flow
```go
type DataSyncManager struct {
    dataGolfClient    *DataGolfClient
    cacheManager      *redis.Client
    updateScheduler   *cron.Cron
    conflictResolver  *DataConflictResolver
}

// Sync intervals based on data type
var SyncIntervals = map[string]time.Duration{
    "field_updates":        5 * time.Minute,   // Critical for WDs
    "live_predictions":     5 * time.Minute,   // During tournaments
    "live_stats":          10 * time.Minute,   // Real-time performance
    "pre_tournament":      30 * time.Minute,   // Pre-tournament updates
    "player_list":         24 * time.Hour,     // Daily refresh
    "schedules":           24 * time.Hour,     // Daily refresh
}
```

#### Caching Strategy
```go
type CacheConfig struct {
    // Short-term cache for live data
    LiveDataTTL       time.Duration // 5 minutes
    
    // Medium-term cache for predictions
    PredictionsTTL    time.Duration // 30 minutes
    
    // Long-term cache for historical data
    HistoricalTTL     time.Duration // 24 hours
    
    // Player/course reference data
    ReferenceTTL      time.Duration // 7 days
}
```

### Error Handling & Fallback Strategy

#### API Reliability Management
```go
type APIReliabilityManager struct {
    primaryProvider   DataProvider   // DataGolf
    fallbackProvider  DataProvider   // RapidAPI (existing)
    circuitBreaker    *CircuitBreaker
    retryPolicy       *RetryPolicy
}

func (arm *APIReliabilityManager) GetPlayerData(req *PlayerDataRequest) (*PlayerDataResponse, error) {
    // Try DataGolf first
    if arm.circuitBreaker.IsHealthy("datagolf") {
        if resp, err := arm.primaryProvider.GetPlayerData(req); err == nil {
            return resp, nil
        }
        arm.circuitBreaker.RecordFailure("datagolf")
    }
    
    // Fallback to RapidAPI with data mapping
    return arm.fallbackProvider.GetPlayerData(req)
}
```

## 🚀 Testing & Validation Strategy

### API Integration Testing
```go
type DataGolfTestSuite struct {
    client        *DataGolfClient
    mockServer    *httptest.Server
    testData      *TestDataProvider
}

// Test coverage for all critical endpoints
func (suite *DataGolfTestSuite) TestCriticalEndpoints() {
    tests := []struct {
        name         string
        endpoint     string
        method       func() (interface{}, error)
        validation   func(interface{}) error
    }{
        {"PlayerList", "/get-player-list", suite.client.GetPlayerList, suite.validatePlayerList},
        {"PreTournament", "/preds/pre-tournament", suite.client.GetPreTournamentPredictions, suite.validatePredictions},
        {"FieldUpdates", "/field-updates", suite.client.GetFieldUpdates, suite.validateFieldUpdates},
        {"FantasyProjections", "/preds/fantasy-projection-defaults", suite.client.GetFantasyProjections, suite.validateFantasy},
    }
    
    for _, test := range tests {
        suite.Run(test.name, func() {
            result, err := test.method()
            suite.NoError(err)
            suite.NoError(test.validation(result))
        })
    }
}
```

### Algorithm Performance Validation
```go
type AlgorithmValidator struct {
    historicalData    *HistoricalDataProvider
    backtestEngine    *BacktestEngine
    metricsCollector  *PerformanceMetrics
}

// Validate algorithm improvements with historical data
func (av *AlgorithmValidator) ValidateAlgorithmEnhancement(
    baselineAlgorithm Algorithm,
    enhancedAlgorithm Algorithm,
    testPeriod DateRange,
) (*ValidationReport, error) {
    
    // Run both algorithms on historical tournaments
    baselineResults := av.backtestEngine.RunBacktest(baselineAlgorithm, testPeriod)
    enhancedResults := av.backtestEngine.RunBacktest(enhancedAlgorithm, testPeriod)
    
    return &ValidationReport{
        ROIImprovement:        enhancedResults.ROI - baselineResults.ROI,
        VolatilityChange:      enhancedResults.Volatility - baselineResults.Volatility,
        WinRateImprovement:    enhancedResults.WinRate - baselineResults.WinRate,
        SharpRatioChange:      enhancedResults.SharpeRatio - baselineResults.SharpeRatio,
        StatisticalSignificance: av.calculateSignificance(baselineResults, enhancedResults),
    }, nil
}
```

This comprehensive architecture provides a robust foundation for integrating DataGolf API while creating an extensible platform for advanced golf optimization algorithms. The modular design ensures that enhancements can be made incrementally while maintaining system reliability and performance.