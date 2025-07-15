# PRD: Golf Tournament Strategy Specialization

## Executive Summary

Transform the golf DFS optimization from basic player selection into sophisticated tournament strategy modeling with cut probability optimization, position-based strategies (T5, T10, T25), and dynamic tournament adjustments. This specialization will provide golf DFS users with professional-grade tournament analysis that adapts to tournament progression and optimal finishing position strategies.

## Problem Statement

### Current State Analysis
**Location**: `services/optimization-service/internal/optimizer/golf_correlation.go` & AI recommendations

**Critical Issues Identified:**
1. **Missing Cut Optimization**: No integration of cut probability into lineup building decisions
2. **Static Tournament Strategy**: No differentiation between T5, T10, T25 optimization goals  
3. **Basic Correlation Usage**: Lines 72-101 have tee time correlations but not utilized in optimization
4. **No Position Modeling**: Missing tournament progression and leaderboard position strategy
5. **Weather Integration Gaps**: Weather data exists but not integrated into optimization algorithms
6. **Incomplete Course History**: Lines 170-181 have course history but limited strategic application

**Strategic Impact:**
- Missing ~15-20% edge from cut probability optimization in tournament play
- No adaptation to different tournament payout structures (majors vs regular events)
- Suboptimal late-round strategy when chasing specific finishing positions
- Missing weather-based correlation opportunities (wind days, etc.)

## Success Metrics

### Golf-Specific Performance
- **Cut Rate Improvement**: Increase lineup cut percentage by 12-15% through cut modeling
- **Finishing Position Accuracy**: Achieve T10 finish 25%+ more often with position-based optimization
- **Weather Edge**: Gain 8-10% ROI improvement on weather-impacted tournaments
- **Course Correlation**: Improve player selection accuracy by 20% using course history

### Tournament Strategy Effectiveness
- **Major Championship Performance**: Specialized optimization for major tournament structures
- **Late Round Adaptation**: Dynamic strategy changes based on tournament position
- **Cut Line Optimization**: Minimize missed cuts while maximizing upside potential
- **Position-Specific ROI**: Different ROI targets for T5 vs T10 vs T25 strategies

## Technical Specifications

### 1. Cut Probability Optimization Engine

**Cut Modeling Framework:**
```go
type CutProbabilityEngine struct {
    HistoricalCutData    map[uint]*PlayerCutHistory    `json:"historical_cuts"`
    CourseCutModel      *CourseCutPredictor           `json:"course_cut_model"`
    WeatherCutImpact    *WeatherCutAdjustment         `json:"weather_cut_impact"`
    FieldStrengthModel  *FieldStrengthAnalyzer        `json:"field_strength"`
    LiveCutTracking     *LiveCutTracker               `json:"live_cut_tracking"`
}

type PlayerCutHistory struct {
    PlayerID            uint                          `json:"player_id"`
    TotalTournaments    int                           `json:"total_tournaments"`
    CutsMade           int                            `json:"cuts_made"`
    CutPercentage      float64                        `json:"cut_percentage"`
    CourseSpecificCuts map[string]CutRecord          `json:"course_specific"`
    RecentForm         []bool                         `json:"recent_form"`       // Last 10 tournaments
    WeatherPerformance map[string]float64             `json:"weather_performance"`
}

type CutProbability struct {
    PlayerID           uint                           `json:"player_id"`
    BaseCutProb        float64                        `json:"base_cut_probability"`
    CourseCutProb      float64                        `json:"course_cut_probability"`
    WeatherAdjusted    float64                        `json:"weather_adjusted"`
    FieldAdjusted      float64                        `json:"field_adjusted"`
    FinalCutProb       float64                        `json:"final_cut_probability"`
    Confidence         float64                        `json:"confidence"`        // 0-1 prediction confidence
}
```

**Cut Optimization Integration:**
- **Lineup Building**: Weight cut probability in player value calculations
- **Risk Management**: Balance high-upside players with safe cut-makers
- **Strategy Selection**: Different cut thresholds for cash vs GPP optimization
- **Dynamic Adjustment**: Update cut probabilities based on round 1 results

### 2. Tournament Position Strategy Engine

**Position-Based Optimization:**
```go
type TournamentPositionStrategy string

const (
    WinStrategy       TournamentPositionStrategy = "win"        // Target 1st place
    TopFiveStrategy   TournamentPositionStrategy = "top_5"      // Target T5 finish  
    TopTenStrategy    TournamentPositionStrategy = "top_10"     // Target T10 finish
    TopTwentyFive     TournamentPositionStrategy = "top_25"     // Target T25 finish
    CutStrategy       TournamentPositionStrategy = "make_cut"   // Just make cut
    BalancedStrategy  TournamentPositionStrategy = "balanced"   // Optimal risk/reward
)

type PositionOptimizer struct {
    Strategy            TournamentPositionStrategy    `json:"strategy"`
    PayoutStructure     TournamentPayouts            `json:"payout_structure"`
    PositionProbabilities map[uint]PositionProbs     `json:"position_probabilities"`
    RiskTolerance       float64                      `json:"risk_tolerance"`
    ContestType         string                       `json:"contest_type"`
}

type PositionProbs struct {
    PlayerID           uint                          `json:"player_id"`
    WinProbability     float64                       `json:"win_probability"`
    Top5Probability    float64                       `json:"top5_probability"`
    Top10Probability   float64                       `json:"top10_probability"`
    Top25Probability   float64                       `json:"top25_probability"`
    CutProbability     float64                       `json:"cut_probability"`
    ExpectedPosition   float64                       `json:"expected_position"`
}
```

**Strategy-Specific Lineup Building:**
- **Win Strategy**: Maximize ceiling with high-volatility players  
- **Top 5 Strategy**: Balance upside with consistency for consistent top finishes
- **Top 10 Strategy**: Focus on safe players with steady performance
- **Cut Strategy**: Emphasize cut-makers for cash games and satellite qualifying

### 3. Advanced Golf Correlation Engine

**Golf-Specific Correlations:**
```go
type GolfCorrelationEngine struct {
    TeeTimeCorrelations  *TeeTimeCorrelationMatrix    `json:"tee_time_correlations"`
    CourseCorrelations   *CourseCorrelationMatrix     `json:"course_correlations"`
    WeatherCorrelations  *WeatherCorrelationMatrix    `json:"weather_correlations"`
    CountryCorrelations  *CountryCorrelationMatrix    `json:"country_correlations"`
    SkillCorrelations    *SkillCorrelationMatrix      `json:"skill_correlations"`
}

type TeeTimeCorrelationMatrix struct {
    SameGroupCorr       float64                       `json:"same_group_correlation"`
    AdjacentGroupCorr   float64                       `json:"adjacent_group_correlation"`
    SameWaveCorr        float64                       `json:"same_wave_correlation"`
    WindCorrelationBonus float64                      `json:"wind_correlation_bonus"`
    TimeDecayFactor     float64                       `json:"time_decay_factor"`
}

type WeatherCorrelationImpact struct {
    WeatherType         string                        `json:"weather_type"`        // "wind", "rain", "calm"
    CorrelationBonus    float64                       `json:"correlation_bonus"`   // Additional correlation
    VarianceImpact      float64                       `json:"variance_impact"`     // Effect on scoring variance  
    SkillBias          float64                        `json:"skill_bias"`          // Advantage to better players
    Course lengthImpact float64                       `json:"length_impact"`       // Course length interaction
}
```

### 4. Dynamic Tournament Adjustment Engine

**Live Tournament Adaptation:**
```go
type TournamentProgressTracker struct {
    CurrentRound        int                           `json:"current_round"`
    CutLine            float64                        `json:"current_cut_line"`
    LeaderboardPositions map[uint]int                 `json:"leaderboard_positions"`
    WeatherForecast     []WeatherCondition            `json:"weather_forecast"`
    TeeTimesSchedule    map[int][]TeeTime             `json:"tee_times_schedule"`
    StrategyAdjustments *StrategyAdjustments          `json:"strategy_adjustments"`
}

type StrategyAdjustments struct {
    CutLinePressure     float64                       `json:"cut_line_pressure"`    // How tight is cut
    LeaderChasing       bool                          `json:"leader_chasing"`       // Need aggressive plays
    WeatherAdvantage    map[string]float64            `json:"weather_advantage"`    // Tee time advantages
    LateSwapTargets     []uint                        `json:"late_swap_targets"`    // Optimal swap candidates
}

type LateSwapRecommendation struct {
    SwapOut            uint                           `json:"swap_out_player"`
    SwapIn             uint                           `json:"swap_in_player"}
    Reasoning          string                         `json:"reasoning"`
    ExpectedGain       float64                        `json:"expected_gain"`
    RiskFactor         float64                        `json:"risk_factor"`
    TimeWindow         time.Duration                  `json:"time_window"`          // How long recommendation valid
}
```

## Implementation Plan

### Phase 1: Cut Probability Framework (Week 1-2)
1. **Historical Analysis**: Build cut probability models from tournament history
2. **Course Integration**: Develop course-specific cut prediction models
3. **Weather Impact**: Model weather effects on cut probability
4. **Optimization Integration**: Incorporate cut probability into lineup building

### Phase 2: Position Strategy Engine (Week 3-4)  
1. **Position Modeling**: Build T5/T10/T25 probability models for all players
2. **Strategy Framework**: Implement position-specific optimization objectives
3. **Payout Integration**: Connect tournament payout structures to optimization
4. **Risk Calibration**: Balance risk/reward for different position targets

### Phase 3: Advanced Correlation System (Week 5-6)
1. **Tee Time Optimization**: Utilize tee time correlation data in lineup building
2. **Weather Correlations**: Build weather-specific correlation adjustments  
3. **Course History**: Integrate course-specific performance correlations
4. **Country/Regional**: Implement nationality-based correlation bonuses

### Phase 4: Live Tournament Adaptation (Week 7-8)
1. **Progress Tracking**: Build real-time tournament status monitoring
2. **Dynamic Adjustments**: Implement live strategy modification algorithms
3. **Late Swap Engine**: Create optimal late swap recommendation system
4. **Performance Validation**: Backtest against historical tournament results

## API Enhancements

### Golf-Specific Optimization Request
```go
type GolfOptimizationRequest struct {
    // Base optimization fields...
    TournamentStrategy    TournamentPositionStrategy  `json:"tournament_strategy"`
    CutOptimization      bool                        `json:"enable_cut_optimization"`
    WeatherConsideration bool                        `json:"include_weather"`
    CourseHistory        bool                        `json:"use_course_history"`
    TeeTimeCorrelations  bool                        `json:"tee_time_correlations"`
    TournamentType       string                      `json:"tournament_type"`      // "major", "regular", "wgc"
    RiskTolerance        float64                     `json:"risk_tolerance"`       // 0-1 scale
    CurrentRound         int                         `json:"current_round"`        // For live tournaments
}

type GolfContextData struct {
    WeatherForecast      []WeatherCondition          `json:"weather_forecast"`
    CourseConditions     CourseConditions            `json:"course_conditions"`
    FieldStrength        float64                     `json:"field_strength"`       // Tournament field quality
    CutProjection        float64                     `json:"projected_cut_line"`
    TeeTimesSchedule     []TeeTimeGroup              `json:"tee_times"`
}
```

### Golf-Specific Response Format
```go
type GolfOptimizationResponse struct {
    // Base response fields...
    GolfAnalytics        GolfLineupAnalytics         `json:"golf_analytics"`
    CutAnalysis         CutAnalysis                  `json:"cut_analysis"`
    PositionProjections PositionProjections          `json:"position_projections"`
    WeatherImpact       WeatherImpactAnalysis        `json:"weather_impact"`
    TeeTimeStrategy     TeeTimeStrategy              `json:"tee_time_strategy"`
    LateSwapSuggestions []LateSwapRecommendation     `json:"late_swap_suggestions"`
}

type GolfLineupAnalytics struct {
    AverageCutProbability float64                    `json:"avg_cut_probability"`
    ExpectedFinishPosition float64                   `json:"expected_finish"`
    Top5Probability       float64                    `json:"top5_probability"`
    Top10Probability      float64                    `json:"top10_probability"`
    WinProbability        float64                    `json:"win_probability"`
    RiskScore            float64                     `json:"risk_score"`           // 0-100 lineup risk
    WeatherAdvantage     float64                     `json:"weather_advantage"`    // Expected weather edge
}

type CutAnalysis struct {
    PlayersLikelyToCut   int                         `json:"players_likely_cut"`
    CutSafetyRating     float64                     `json:"cut_safety_rating"`
    RiskiestPlayer      uint                        `json:"riskiest_player"`
    SafestPlayer        uint                        `json:"safest_player"`
    CutLineProjection   float64                     `json:"cut_line_projection"`
}
```

## Integration Points

### With Core Optimization (PRD-1)
- **Player Analytics**: Cut probability and position modeling integrated into player value calculations
- **Multi-Objective**: Position strategies become optimization objectives
- **Risk Management**: Cut probability constraints in exposure management

### With AI Recommendations (PRD-2)
- **Golf Context**: Tournament-specific prompting with cut and position context
- **Strategy Communication**: AI explains position strategy and cut considerations  
- **Late Swap Intelligence**: AI recommendations for optimal tournament adjustments

### With Monte Carlo (PRD-3)
- **Golf Distributions**: Custom distributions incorporating cut probability
- **Position Simulation**: Simulate finish position probabilities accurately
- **Weather Simulation**: Golf-specific weather impact modeling

## Advanced Features

### 1. Machine Learning Cut Prediction
```go
type MLCutPredictor struct {
    FeatureExtractor     *GolfFeatureExtractor       `json:"feature_extractor"`
    CutModel            *CutPredictionModel          `json:"cut_model"`
    CourseModels        map[string]*CourseModel      `json:"course_models"`
    EnsemblePrediction  *EnsembleCutPredictor        `json:"ensemble_predictor"`
}
```

### 2. Real-time Course Conditions
```go
type LiveCourseMonitor struct {
    WindMonitoring      *WindConditionTracker        `json:"wind_monitoring"`
    GreensConditions    *GreensSpeedTracker          `json:"greens_conditions"`
    PinPositions        *PinPositionTracker          `json:"pin_positions"`
    CourseSetup         *CourseSetupAnalyzer         `json:"course_setup"`
}
```

### 3. Tournament Progression Modeling
```go
type TournamentProgressModel struct {
    Round1Prediction    *Round1Model                 `json:"round1_model"`
    CutLineEvolution    *CutLinePredictor            `json:"cutline_evolution"`
    WeekendMovement     *WeekendMovementModel        `json:"weekend_movement"`
    FinalRoundStrategy  *FinalRoundOptimizer         `json:"final_round_strategy"`
}
```

## Golf-Specific Strategies

### Tournament Type Specialization
- **Majors**: Emphasize consistency and experience over pure upside
- **WGC Events**: Balance international players and field strength
- **Regular PGA**: Standard optimization with course history emphasis
- **Playoffs**: Strategy changes based on FedEx Cup implications

### Weather Strategy Integration
- **Wind Days**: Increase tee time correlation, favor experienced players
- **Rain/Soft Conditions**: Favor long hitters, adjust course correlation
- **Perfect Conditions**: Standard optimization with scoring variance reduction

### Course Type Optimization
- **Links Golf**: Favor wind players, creativity, course management
- **Target Golf**: Emphasize approach play and putting
- **Long Courses**: Favor distance players in optimization
- **Short Courses**: Emphasize short game and putting statistics

## Risk Mitigation

### Data Quality Risks
- **Cut Line Accuracy**: Multiple source validation for cut projections
- **Weather Reliability**: Integration with multiple weather services
- **Course Condition**: Real-time validation of course setup data

### Strategy Risks  
- **Over-Optimization**: Maintain balance between cut safety and upside
- **Weather Dependency**: Fallback strategies for unexpected weather changes
- **Tournament Evolution**: Adaptive strategies for changing tournament conditions

### Performance Risks
- **Computational Complexity**: Efficient algorithms for real-time optimization
- **Data Pipeline**: Robust handling of live tournament data feeds
- **Model Accuracy**: Continuous validation against actual tournament results

## Success Validation

### Historical Backtesting
- **Cut Rate Validation**: Test cut probability models against 3+ years of data
- **Position Accuracy**: Validate finish position predictions
- **Weather Impact**: Confirm weather correlation improvements

### Live Tournament Testing
- **Real-time Performance**: Monitor optimization accuracy during live tournaments
- **Late Swap Effectiveness**: Track success rate of late swap recommendations
- **Strategy ROI**: Measure ROI improvement by tournament strategy type

## Future Enhancements

### Advanced Analytics
- **Shot-by-Shot Modeling**: Granular shot-level performance prediction
- **Strokes Gained Integration**: Comprehensive strokes gained optimization
- **Course Fit Modeling**: Advanced course/player fit analysis

### International Tour Integration
- **European Tour**: Expand to DP World Tour events
- **Asian Tours**: Integration with Asian tour data and strategies
- **LIV Golf**: Adaptation to team-based tournament formats

---

**Estimated Timeline**: 8 weeks
**Resource Requirements**: 2 golf data specialists, 1 algorithm engineer, 1 domain expert
**Dependencies**: Comprehensive golf data pipeline, weather integration, real-time tournament feeds
**Risk Level**: Medium (golf domain complexity, data availability challenges)