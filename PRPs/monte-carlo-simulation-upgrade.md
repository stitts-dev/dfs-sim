# PRP: Monte Carlo Simulation Enhancement

## Overview
This PRP details the implementation of sophisticated Monte Carlo simulation enhancements for the DFS optimizer, upgrading from basic variance modeling to sport-specific outcome generation with dynamic correlation matrices and realistic distribution modeling.

## Context and Background

### Current Implementation Analysis
**Location**: `services/optimization-service/internal/simulator/`

The current implementation has several limitations:
- **Basic variance**: Uses flat 25% variance for all players
- **Simple correlations**: Crude approximation without proper matrix decomposition  
- **Static modeling**: Fixed injury/DNP probabilities
- **Missing features**: No weather impact, game script modeling, or field composition

### Research Findings

#### Gonum Library Integration
The Gonum library (https://www.gonum.org/) provides optimized linear algebra operations including Cholesky decomposition for correlation matrices. Key packages:
- `gonum.org/v1/gonum/mat` - Matrix operations and decomposition
- `gonum.org/v1/gonum/stat` - Statistical functions and distributions
- `gonum.org/v1/gonum/floats` - Float slice operations

#### Monte Carlo Best Practices
Based on research from DFS simulation implementations:
- Use Cholesky decomposition for proper correlation modeling (https://alexander-pastukhov.github.io/notes-on-statistics/advanced-04-cholesky.html)
- Implement sport-specific distribution models (Beta for high variance, Normal for standard)
- Apply Iman-Conover method for correlation preservation
- Use worker pools for parallel simulation execution

## Implementation Plan

### Task List

1. **Set up Gonum dependencies and create distribution engine**
2. **Implement sport-specific distribution models**
3. **Build advanced correlation matrix with Cholesky decomposition**  
4. **Create context-aware simulation engine**
5. **Implement event simulators (injury, weather, game script)**
6. **Build enhanced contest modeling with field composition**
7. **Add comprehensive testing and validation**
8. **Integrate with optimization API and add performance monitoring**

### Phase 1: Distribution Engine Foundation

#### 1.1 Add Gonum Dependencies
```bash
cd services/optimization-service
go get -u gonum.org/v1/gonum/mat
go get -u gonum.org/v1/gonum/stat
go get -u gonum.org/v1/gonum/stat/distuv
```

#### 1.2 Create Distribution Types
Create `services/optimization-service/internal/simulator/distribution_engine.go`:

```go
package simulator

import (
    "math"
    "gonum.org/v1/gonum/stat/distuv"
)

type DistributionType string

const (
    Normal      DistributionType = "normal"
    LogNormal   DistributionType = "lognormal"
    Beta        DistributionType = "beta"
    Gamma       DistributionType = "gamma"
    Exponential DistributionType = "exponential"
    Custom      DistributionType = "custom"
)

type PlayerDistribution struct {
    DistType       DistributionType `json:"distribution_type"`
    Parameters     []float64        `json:"parameters"`
    Floor          float64          `json:"floor"`
    Ceiling        float64          `json:"ceiling"`
    SkewnessFactor float64          `json:"skewness"`
    KurtosisFactor float64          `json:"kurtosis"`
    InjuryProb     float64          `json:"injury_probability"`
    WeatherImpact  *WeatherEffect   `json:"weather_impact"`
}

type SportDistributionConfig struct {
    Sport               string                      `json:"sport"`
    DefaultDistribution DistributionType           `json:"default_distribution"`
    PositionOverrides   map[string]DistributionType `json:"position_overrides"`
    VarianceFactors     map[string]float64          `json:"variance_factors"`
    CorrelationModel    string                      `json:"correlation_model"`
}

// DistributionEngine handles sport-specific distribution generation
type DistributionEngine struct {
    configs map[string]SportDistributionConfig
    cache   map[string]distuv.Rander
}

func NewDistributionEngine() *DistributionEngine {
    return &DistributionEngine{
        configs: initializeSportConfigs(),
        cache:   make(map[string]distuv.Rander),
    }
}

func initializeSportConfigs() map[string]SportDistributionConfig {
    return map[string]SportDistributionConfig{
        "nba": {
            Sport:               "nba",
            DefaultDistribution: Beta,
            PositionOverrides: map[string]DistributionType{
                "C": Normal, // Centers more predictable
            },
            VarianceFactors: map[string]float64{
                "PG": 1.2, "SG": 1.1, "SF": 1.0, "PF": 0.9, "C": 0.8,
            },
            CorrelationModel: "position_based",
        },
        "nfl": {
            Sport:               "nfl",
            DefaultDistribution: LogNormal,
            PositionOverrides: map[string]DistributionType{
                "K":   Exponential, // Kickers have exponential distribution
                "DST": Gamma,       // Defense special teams
            },
            VarianceFactors: map[string]float64{
                "QB": 1.0, "RB": 1.3, "WR": 1.4, "TE": 1.2, "K": 0.7, "DST": 1.1,
            },
            CorrelationModel: "game_script",
        },
        "golf": {
            Sport:               "golf",
            DefaultDistribution: Custom, // Will use custom cut modeling
            PositionOverrides:   map[string]DistributionType{},
            VarianceFactors: map[string]float64{
                "default": 1.5, // High variance sport
            },
            CorrelationModel: "tee_time",
        },
    }
}

func (de *DistributionEngine) CreateDistribution(
    player *Player, 
    sport string,
) distuv.Rander {
    config := de.configs[sport]
    distType := config.DefaultDistribution
    
    // Check position overrides
    if override, exists := config.PositionOverrides[player.Position]; exists {
        distType = override
    }
    
    // Calculate parameters based on player stats
    mean := player.ProjectedPoints
    variance := player.ProjectedPoints * 0.25 // Base variance
    
    // Apply sport/position variance factors
    if factor, exists := config.VarianceFactors[player.Position]; exists {
        variance *= factor
    }
    
    stdDev := math.Sqrt(variance)
    
    switch distType {
    case Normal:
        return &distuv.Normal{
            Mu:    mean,
            Sigma: stdDev,
        }
    case LogNormal:
        // Convert mean/variance to lognormal parameters
        mu := math.Log(mean) - 0.5*math.Log(1+variance/(mean*mean))
        sigma := math.Sqrt(math.Log(1 + variance/(mean*mean)))
        return &distuv.LogNormal{
            Mu:    mu,
            Sigma: sigma,
        }
    case Beta:
        // Fit beta distribution to player's floor/ceiling
        alpha := mean * mean / variance
        beta := alpha * (player.CeilingPoints/mean - 1)
        return &distuv.Beta{
            Alpha: alpha,
            Beta:  beta,
        }
    case Gamma:
        shape := mean * mean / variance
        rate := mean / variance
        return &distuv.Gamma{
            Alpha: shape,
            Beta:  rate,
        }
    case Exponential:
        return &distuv.Exponential{
            Rate: 1.0 / mean,
        }
    default:
        // Fallback to normal
        return &distuv.Normal{
            Mu:    mean,
            Sigma: stdDev,
        }
    }
}
```

### Phase 2: Advanced Correlation Modeling

#### 2.1 Implement Cholesky-Based Correlation
Create `services/optimization-service/internal/simulator/advanced_correlation.go`:

```go
package simulator

import (
    "math"
    "gonum.org/v1/gonum/mat"
    "gonum.org/v1/gonum/stat"
)

type AdvancedCorrelationMatrix struct {
    BaseCorrelations      map[uint]map[uint]float64    `json:"base_correlations"`
    GameStateCorrelations map[string]float64           `json:"gamestate_correlations"`
    WeatherCorrelations   map[string]float64           `json:"weather_correlations"`
    InjuryCorrelations    map[uint]float64             `json:"injury_correlations"`
    StackingBonuses       map[string]float64           `json:"stacking_bonuses"`
    TimeDecayFactors      map[string]float64           `json:"time_decay"`
    
    // Internal state
    playerIndices map[uint]int
    corrMatrix    *mat.SymDense
    choleskyL     *mat.Cholesky
}

type CorrelationContext struct {
    GameState         string   `json:"game_state"`
    WeatherConditions string   `json:"weather"`
    InjuryReports     []uint   `json:"injury_concerns"`
    ContestType       string   `json:"contest_type"`
    TimeToKickoff     int      `json:"time_to_kickoff"`
}

func NewAdvancedCorrelationMatrix(players []Player) *AdvancedCorrelationMatrix {
    acm := &AdvancedCorrelationMatrix{
        BaseCorrelations:      make(map[uint]map[uint]float64),
        GameStateCorrelations: make(map[string]float64),
        WeatherCorrelations:   make(map[string]float64),
        InjuryCorrelations:    make(map[uint]float64),
        StackingBonuses:       make(map[string]float64),
        TimeDecayFactors:      make(map[string]float64),
        playerIndices:         make(map[uint]int),
    }
    
    // Initialize player indices
    for i, player := range players {
        acm.playerIndices[player.ID] = i
    }
    
    // Build correlation matrix
    acm.buildCorrelationMatrix(players)
    
    return acm
}

func (acm *AdvancedCorrelationMatrix) buildCorrelationMatrix(players []Player) {
    n := len(players)
    data := make([]float64, n*n)
    
    // Initialize with identity matrix
    for i := 0; i < n; i++ {
        data[i*n+i] = 1.0
    }
    
    // Build correlations
    for i := 0; i < n; i++ {
        for j := i + 1; j < n; j++ {
            corr := acm.calculateCorrelation(&players[i], &players[j])
            data[i*n+j] = corr
            data[j*n+i] = corr // Symmetric
        }
    }
    
    acm.corrMatrix = mat.NewSymDense(n, data)
    
    // Perform Cholesky decomposition
    acm.choleskyL = &mat.Cholesky{}
    if ok := acm.choleskyL.Factorize(acm.corrMatrix); !ok {
        // Matrix not positive definite - add small diagonal perturbation
        acm.ensurePositiveDefinite()
    }
}

func (acm *AdvancedCorrelationMatrix) ensurePositiveDefinite() {
    n := acm.corrMatrix.SymmetricDim()
    epsilon := 1e-6
    
    // Add small value to diagonal
    for i := 0; i < n; i++ {
        current := acm.corrMatrix.At(i, i)
        acm.corrMatrix.SetSym(i, i, current+epsilon)
    }
    
    // Retry decomposition
    acm.choleskyL.Factorize(acm.corrMatrix)
}

func (acm *AdvancedCorrelationMatrix) calculateCorrelation(p1, p2 *Player) float64 {
    baseCorr := 0.0
    
    // Same team correlation
    if p1.TeamID == p2.TeamID {
        baseCorr = acm.getPositionCorrelation(p1.Position, p2.Position)
    }
    
    // Opponent correlation (game stack)
    if p1.OpponentID == p2.TeamID || p2.OpponentID == p1.TeamID {
        baseCorr = 0.15 // Base game stack correlation
    }
    
    // Sport-specific adjustments
    switch p1.Sport {
    case "nfl":
        baseCorr = acm.adjustNFLCorrelation(p1, p2, baseCorr)
    case "nba":
        baseCorr = acm.adjustNBACorrelation(p1, p2, baseCorr)
    case "golf":
        baseCorr = acm.adjustGolfCorrelation(p1, p2, baseCorr)
    }
    
    return math.Max(-1.0, math.Min(1.0, baseCorr))
}

func (acm *AdvancedCorrelationMatrix) getPositionCorrelation(pos1, pos2 string) float64 {
    // Position-based correlations
    correlations := map[string]map[string]float64{
        "QB": {"WR": 0.50, "TE": 0.35, "RB": -0.15},
        "RB": {"RB": -0.35, "WR": -0.10},
        "WR": {"WR": 0.25, "TE": 0.15},
        // Add more position correlations
    }
    
    if corr, exists := correlations[pos1][pos2]; exists {
        return corr
    }
    if corr, exists := correlations[pos2][pos1]; exists {
        return corr
    }
    
    return 0.0
}

func (acm *AdvancedCorrelationMatrix) GenerateCorrelatedOutcomes(
    players []Player,
    distributions []distuv.Rander,
    context *CorrelationContext,
) []float64 {
    n := len(players)
    
    // Generate independent standard normals
    independent := make([]float64, n)
    for i := 0; i < n; i++ {
        independent[i] = distuv.Normal{Mu: 0, Sigma: 1}.Rand()
    }
    
    // Apply Cholesky decomposition to create correlated normals
    correlatedNormals := mat.NewVecDense(n, nil)
    independentVec := mat.NewVecDense(n, independent)
    
    // Get lower triangular matrix from Cholesky
    var L mat.TriDense
    acm.choleskyL.LTo(&L)
    
    // Multiply: correlated = L * independent
    correlatedNormals.MulVec(&L, independentVec)
    
    // Transform to target distributions
    outcomes := make([]float64, n)
    for i := 0; i < n; i++ {
        // Convert correlated normal to uniform [0,1]
        uniform := stat.NormalCDF(correlatedNormals.AtVec(i), 0, 1)
        
        // Use inverse CDF of target distribution
        outcomes[i] = acm.inverseTransform(uniform, distributions[i], &players[i])
        
        // Apply context adjustments
        outcomes[i] = acm.applyContextAdjustments(outcomes[i], &players[i], context)
    }
    
    return outcomes
}

func (acm *AdvancedCorrelationMatrix) inverseTransform(
    uniform float64,
    dist distuv.Rander,
    player *Player,
) float64 {
    // Use quantile function for inverse transform
    if quantiler, ok := dist.(distuv.Quantiler); ok {
        value := quantiler.Quantile(uniform)
        
        // Apply floor/ceiling constraints
        value = math.Max(player.FloorPoints, value)
        value = math.Min(player.CeilingPoints, value)
        
        return value
    }
    
    // Fallback to direct sampling
    return dist.Rand()
}
```

### Phase 3: Event Simulation Framework

#### 3.1 Create Event Simulators
Create `services/optimization-service/internal/simulator/event_simulator.go`:

```go
package simulator

import (
    "math/rand"
)

type GameEventSimulator struct {
    InjuryEvents     *InjurySimulator     `json:"injury_events"`
    WeatherEvents    *WeatherSimulator    `json:"weather_events"`
    GameScriptEvents *GameScriptSimulator `json:"gamescript_events"`
    VolatilityEvents *VolatilitySimulator `json:"volatility_events"`
}

type GameEvent struct {
    Type        string  `json:"type"`
    PlayerID    uint    `json:"player_id"`
    Impact      float64 `json:"impact"`
    Description string  `json:"description"`
}

type InjurySimulator struct {
    BaseRates    map[string]float64 // Position-based injury rates
    HistoryRates map[uint]float64   // Player-specific injury history
}

func NewInjurySimulator() *InjurySimulator {
    return &InjurySimulator{
        BaseRates: map[string]float64{
            "RB":  0.025, // 2.5% base injury rate
            "WR":  0.015,
            "QB":  0.010,
            "TE":  0.020,
            "K":   0.005,
            "DST": 0.010,
            // NBA
            "PG": 0.012,
            "SG": 0.012,
            "SF": 0.015,
            "PF": 0.018,
            "C":  0.020,
        },
        HistoryRates: make(map[uint]float64),
    }
}

func (is *InjurySimulator) SimulateInjury(player *Player, rng *rand.Rand) *GameEvent {
    // Base rate from position
    rate := is.BaseRates[player.Position]
    
    // Adjust for injury history
    if historyRate, exists := is.HistoryRates[player.ID]; exists {
        rate *= (1 + historyRate)
    }
    
    // Adjust for existing injury status
    if player.InjuryStatus == "Q" {
        rate *= 3.0 // 3x injury risk if questionable
    } else if player.InjuryStatus == "D" {
        rate *= 5.0 // 5x injury risk if doubtful
    }
    
    if rng.Float64() < rate {
        return &GameEvent{
            Type:        "injury",
            PlayerID:    player.ID,
            Impact:      -1.0, // Complete DNP
            Description: "In-game injury",
        }
    }
    
    // Check for early exit (partial game)
    if rng.Float64() < rate*2 {
        impact := -0.3 - rng.Float64()*0.4 // -30% to -70% points
        return &GameEvent{
            Type:        "early_exit",
            PlayerID:    player.ID,
            Impact:      impact,
            Description: "Left game early",
        }
    }
    
    return nil
}

type WeatherSimulator struct {
    WindImpact map[string]float64
    RainImpact map[string]float64
    ColdImpact map[string]float64
}

func NewWeatherSimulator() *WeatherSimulator {
    return &WeatherSimulator{
        WindImpact: map[string]float64{
            "QB": -0.15, "WR": -0.20, "K": -0.30,
        },
        RainImpact: map[string]float64{
            "QB": -0.10, "WR": -0.15, "RB": 0.05,
        },
        ColdImpact: map[string]float64{
            "QB": -0.05, "K": -0.10,
        },
    }
}

type GameScriptSimulator struct {
    BlowoutThreshold   float64
    GarbageTimeImpact  map[string]float64
    ComebackBonus      map[string]float64
}

func NewGameScriptSimulator() *GameScriptSimulator {
    return &GameScriptSimulator{
        BlowoutThreshold: 21.0, // 21+ point differential
        GarbageTimeImpact: map[string]float64{
            "QB": -0.20, "RB": -0.15, "WR": -0.25,
        },
        ComebackBonus: map[string]float64{
            "QB": 0.15, "WR": 0.20, "TE": 0.10,
        },
    }
}
```

### Phase 4: Enhanced Contest Modeling

#### 4.1 Create Contest Field Simulator
Create `services/optimization-service/internal/simulator/contest_field.go`:

```go
package simulator

type ContestModelConfig struct {
    ContestType      string            `json:"contest_type"`
    EntryFee         float64           `json:"entry_fee"`
    TotalEntries     int               `json:"total_entries"`
    PayoutStructure  []PayoutTier      `json:"payout_structure"`
    FieldComposition *FieldComposition `json:"field_composition"`
    OwnershipModel   *OwnershipModel   `json:"ownership_model"`
}

type FieldComposition struct {
    SharksPercentage    float64            `json:"sharks_percentage"`
    RecsPercentage      float64            `json:"recs_percentage"`
    BeginnersPercentage float64            `json:"beginners_percentage"`
    SkillAdjustments    map[string]float64 `json:"skill_adjustments"`
}

type OwnershipModel struct {
    ChalkThreshold      float64            `json:"chalk_threshold"`
    LeverageOpportunity float64            `json:"leverage_opportunity"`
    StackOwnership      map[string]float64 `json:"stack_ownership"`
    PositionBias        map[string]float64 `json:"position_bias"`
}

type EnhancedContestSimulator struct {
    config         *ContestModelConfig
    fieldGenerator *FieldGenerator
    payoutEngine   *PayoutEngine
}

func NewEnhancedContestSimulator(config *ContestModelConfig) *EnhancedContestSimulator {
    return &EnhancedContestSimulator{
        config:         config,
        fieldGenerator: NewFieldGenerator(config),
        payoutEngine:   NewPayoutEngine(config.PayoutStructure),
    }
}

type FieldGenerator struct {
    config       *ContestModelConfig
    sharkLineups []*GeneratedLineup
    recLineups   []*GeneratedLineup
}

func (fg *FieldGenerator) GenerateContestField(
    players []Player,
    optimizer OptimizerInterface,
    size int,
) []*GeneratedLineup {
    field := make([]*GeneratedLineup, 0, size)
    
    // Calculate skill distribution
    numSharks := int(float64(size) * fg.config.FieldComposition.SharksPercentage)
    numRecs := int(float64(size) * fg.config.FieldComposition.RecsPercentage)
    numBeginners := size - numSharks - numRecs
    
    // Generate shark lineups (optimal with slight variations)
    for i := 0; i < numSharks; i++ {
        lineup := fg.generateSharkLineup(players, optimizer)
        field = append(field, lineup)
    }
    
    // Generate recreational lineups (good but not optimal)
    for i := 0; i < numRecs; i++ {
        lineup := fg.generateRecLineup(players, optimizer)
        field = append(field, lineup)
    }
    
    // Generate beginner lineups (suboptimal choices)
    for i := 0; i < numBeginners; i++ {
        lineup := fg.generateBeginnerLineup(players)
        field = append(field, lineup)
    }
    
    return field
}

func (fg *FieldGenerator) generateSharkLineup(
    players []Player,
    optimizer OptimizerInterface,
) *GeneratedLineup {
    // Use optimizer with slight randomization
    constraints := &OptimizationConstraints{
        MinSalary:        48000,
        MaxExposure:      0.35,
        RequireComplete:  true,
        RandomnessWeight: 0.05, // Small randomness
    }
    
    result := optimizer.Optimize(players, constraints, 1)
    if len(result.Lineups) > 0 {
        return &GeneratedLineup{
            Lineup:     result.Lineups[0],
            SkillLevel: "shark",
        }
    }
    
    return nil
}
```

### Phase 5: Integration and API Updates

#### 5.1 Update Monte Carlo Engine
Update `services/optimization-service/internal/simulator/monte_carlo.go`:

```go
package simulator

import (
    "context"
    "fmt"
    "sync"
    "time"
)

type EnhancedMonteCarloSimulator struct {
    config              *SimulationConfig
    distributionEngine  *DistributionEngine
    correlationMatrix   *AdvancedCorrelationMatrix
    eventSimulator      *GameEventSimulator
    contestSimulator    *EnhancedContestSimulator
    progressChan        chan SimulationProgress
    workers             int
}

func NewEnhancedMonteCarloSimulator(config *SimulationConfig) *EnhancedMonteCarloSimulator {
    return &EnhancedMonteCarloSimulator{
        config:             config,
        distributionEngine: NewDistributionEngine(),
        eventSimulator:     NewGameEventSimulator(),
        progressChan:       make(chan SimulationProgress, 100),
        workers:           config.Workers,
    }
}

func (s *EnhancedMonteCarloSimulator) RunSimulation(
    ctx context.Context,
    lineup *Lineup,
    contest *Contest,
    players []Player,
) (*SimulationResultV2, error) {
    startTime := time.Now()
    
    // Build correlation matrix
    s.correlationMatrix = NewAdvancedCorrelationMatrix(players)
    
    // Initialize contest simulator
    contestConfig := &ContestModelConfig{
        ContestType:  contest.Type,
        EntryFee:     contest.EntryFee,
        TotalEntries: contest.MaxEntries,
        PayoutStructure: contest.PayoutStructure,
        FieldComposition: &FieldComposition{
            SharksPercentage:    0.10,
            RecsPercentage:      0.60,
            BeginnersPercentage: 0.30,
        },
    }
    s.contestSimulator = NewEnhancedContestSimulator(contestConfig)
    
    // Create distributions for all players
    distributions := make([]distuv.Rander, len(players))
    for i, player := range players {
        distributions[i] = s.distributionEngine.CreateDistribution(&player, contest.Sport)
    }
    
    // Run parallel simulations
    results := s.runParallelSimulations(ctx, lineup, players, distributions)
    
    // Analyze results
    analysis := s.analyzeResults(results, lineup, contest)
    
    return &SimulationResultV2{
        SimulationID:          generateID(),
        LineupID:              lineup.ID,
        Iterations:            len(results),
        ExecutionTime:         time.Since(startTime),
        Results:               results,
        AdvancedMetrics:       analysis.Metrics,
        DistributionAnalysis:  analysis.Distribution,
        CorrelationBreakdown:  analysis.Correlation,
        EventAnalysis:         analysis.Events,
        ContestProjections:    analysis.Contest,
        RiskAssessment:        analysis.Risk,
    }, nil
}

func (s *EnhancedMonteCarloSimulator) runParallelSimulations(
    ctx context.Context,
    lineup *Lineup,
    players []Player,
    distributions []distuv.Rander,
) []SimulationIteration {
    numIterations := s.config.NumSimulations
    iterations := make([]SimulationIteration, numIterations)
    
    // Create work channel
    workChan := make(chan int, numIterations)
    for i := 0; i < numIterations; i++ {
        workChan <- i
    }
    close(workChan)
    
    // Worker pool
    var wg sync.WaitGroup
    resultChan := make(chan SimulationIteration, numIterations)
    
    for w := 0; w < s.workers; w++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            
            rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))
            
            for iterNum := range workChan {
                select {
                case <-ctx.Done():
                    return
                default:
                    iteration := s.simulateSingleIteration(
                        lineup, players, distributions, rng, iterNum,
                    )
                    resultChan <- iteration
                    
                    // Report progress
                    if iterNum%100 == 0 {
                        s.progressChan <- SimulationProgress{
                            Current: iterNum,
                            Total:   numIterations,
                            Stage:   "simulating",
                        }
                    }
                }
            }
        }(w)
    }
    
    // Collect results
    go func() {
        wg.Wait()
        close(resultChan)
    }()
    
    i := 0
    for result := range resultChan {
        iterations[i] = result
        i++
    }
    
    return iterations
}

func (s *EnhancedMonteCarloSimulator) simulateSingleIteration(
    lineup *Lineup,
    players []Player,
    distributions []distuv.Rander,
    rng *rand.Rand,
    iterNum int,
) SimulationIteration {
    // Create context for this iteration
    context := &CorrelationContext{
        GameState:     s.determineGameState(rng),
        ContestType:   s.config.ContestType,
        TimeToKickoff: 0, // Assuming contest has started
    }
    
    // Generate correlated outcomes
    outcomes := s.correlationMatrix.GenerateCorrelatedOutcomes(
        players, distributions, context,
    )
    
    // Apply events
    events := make([]GameEvent, 0)
    for i, player := range players {
        // Injury events
        if event := s.eventSimulator.InjuryEvents.SimulateInjury(&player, rng); event != nil {
            outcomes[i] *= (1 + event.Impact)
            events = append(events, *event)
        }
        
        // Weather events (if applicable)
        if s.config.WeatherConditions != "" {
            impact := s.eventSimulator.WeatherEvents.GetWeatherImpact(
                &player, s.config.WeatherConditions,
            )
            outcomes[i] *= (1 + impact)
        }
    }
    
    // Calculate lineup score
    lineupScore := s.calculateLineupScore(lineup, players, outcomes)
    
    // Simulate contest field and determine placement
    field := s.contestSimulator.GenerateQuickField(1000, rng)
    placement := s.calculatePlacement(lineupScore, field)
    payout := s.contestSimulator.CalculatePayout(placement, 1000)
    
    return SimulationIteration{
        IterationNum:    iterNum,
        PlayerOutcomes:  outcomes,
        LineupScore:     lineupScore,
        ContestPlacement: placement,
        Payout:          payout,
        Events:          events,
        CorrelationScore: s.correlationMatrix.GetIterationCorrelation(outcomes),
    }
}
```

### Phase 6: Testing and Validation

#### 6.1 Create Comprehensive Tests
Create `services/optimization-service/internal/simulator/monte_carlo_test.go`:

```go
package simulator

import (
    "context"
    "testing"
    "gonum.org/v1/gonum/stat"
    "github.com/stretchr/testify/assert"
)

func TestDistributionEngine(t *testing.T) {
    engine := NewDistributionEngine()
    
    t.Run("NBA Beta Distribution", func(t *testing.T) {
        player := &Player{
            ID:              1,
            Name:            "Test Player",
            Position:        "PG",
            ProjectedPoints: 40.0,
            FloorPoints:     20.0,
            CeilingPoints:   60.0,
        }
        
        dist := engine.CreateDistribution(player, "nba")
        
        // Generate samples
        samples := make([]float64, 10000)
        for i := range samples {
            samples[i] = dist.Rand()
        }
        
        // Verify distribution properties
        mean := stat.Mean(samples, nil)
        stdDev := stat.StdDev(samples, nil)
        
        assert.InDelta(t, 40.0, mean, 2.0, "Mean should be close to projection")
        assert.Greater(t, stdDev, 5.0, "Should have reasonable variance")
        
        // Check bounds
        for _, sample := range samples {
            assert.GreaterOrEqual(t, sample, 0.0)
            assert.LessOrEqual(t, sample, player.CeilingPoints*1.2)
        }
    })
    
    t.Run("NFL LogNormal Distribution", func(t *testing.T) {
        player := &Player{
            ID:              2,
            Name:            "RB Test",
            Position:        "RB",
            ProjectedPoints: 15.0,
            FloorPoints:     0.0,
            CeilingPoints:   40.0,
        }
        
        dist := engine.CreateDistribution(player, "nfl")
        
        // Test skewness
        samples := make([]float64, 10000)
        for i := range samples {
            samples[i] = dist.Rand()
        }
        
        // Log-normal should have positive skew
        skewness := calculateSkewness(samples)
        assert.Greater(t, skewness, 0.0, "Log-normal should have positive skew")
    })
}

func TestAdvancedCorrelationMatrix(t *testing.T) {
    players := []Player{
        {ID: 1, Name: "QB1", Position: "QB", TeamID: 1, OpponentID: 2},
        {ID: 2, Name: "WR1", Position: "WR", TeamID: 1, OpponentID: 2},
        {ID: 3, Name: "WR2", Position: "WR", TeamID: 1, OpponentID: 2},
        {ID: 4, Name: "RB1", Position: "RB", TeamID: 1, OpponentID: 2},
        {ID: 5, Name: "QB2", Position: "QB", TeamID: 2, OpponentID: 1},
    }
    
    matrix := NewAdvancedCorrelationMatrix(players)
    
    t.Run("Cholesky Decomposition", func(t *testing.T) {
        assert.NotNil(t, matrix.choleskyL, "Cholesky decomposition should succeed")
        
        // Verify positive definite
        assert.True(t, matrix.choleskyL.IsValid(), "Matrix should be valid")
    })
    
    t.Run("Correlation Generation", func(t *testing.T) {
        distributions := make([]distuv.Rander, len(players))
        for i := range distributions {
            distributions[i] = &distuv.Normal{Mu: 0, Sigma: 1}
        }
        
        context := &CorrelationContext{
            GameState:   "normal",
            ContestType: "gpp",
        }
        
        // Generate multiple sets and check correlations
        numSets := 1000
        outcomes := make([][]float64, numSets)
        
        for i := 0; i < numSets; i++ {
            outcomes[i] = matrix.GenerateCorrelatedOutcomes(
                players, distributions, context,
            )
        }
        
        // Check QB-WR correlation
        qbScores := make([]float64, numSets)
        wr1Scores := make([]float64, numSets)
        
        for i := 0; i < numSets; i++ {
            qbScores[i] = outcomes[i][0]
            wr1Scores[i] = outcomes[i][1]
        }
        
        correlation := stat.Correlation(qbScores, wr1Scores, nil)
        assert.Greater(t, correlation, 0.3, "QB-WR should have positive correlation")
        assert.Less(t, correlation, 0.7, "Correlation shouldn't be too strong")
    })
}

func TestEnhancedContestModeling(t *testing.T) {
    config := &ContestModelConfig{
        ContestType:  "gpp",
        EntryFee:     25.0,
        TotalEntries: 5000,
        PayoutStructure: []PayoutTier{
            {Place: 1, Percentage: 0.20},
            {Place: 10, Percentage: 0.05},
            {Place: 50, Percentage: 0.02},
            {Place: 100, Percentage: 0.01},
            {Place: 500, Percentage: 0.005},
            {Place: 1000, Percentage: 0.0025},
        },
        FieldComposition: &FieldComposition{
            SharksPercentage:    0.10,
            RecsPercentage:      0.60,
            BeginnersPercentage: 0.30,
        },
    }
    
    simulator := NewEnhancedContestSimulator(config)
    
    t.Run("Field Generation", func(t *testing.T) {
        // Mock players and optimizer
        players := generateTestPlayers(100)
        optimizer := &MockOptimizer{}
        
        field := simulator.fieldGenerator.GenerateContestField(
            players, optimizer, 1000,
        )
        
        assert.Len(t, field, 1000, "Should generate correct field size")
        
        // Count skill levels
        sharks := 0
        for _, lineup := range field {
            if lineup.SkillLevel == "shark" {
                sharks++
            }
        }
        
        assert.InDelta(t, 100, sharks, 20, "Should have ~10% sharks")
    })
}

// Performance benchmarks
func BenchmarkCorrelatedGeneration(b *testing.B) {
    players := generateTestPlayers(200)
    matrix := NewAdvancedCorrelationMatrix(players)
    distributions := make([]distuv.Rander, len(players))
    
    for i := range distributions {
        distributions[i] = &distuv.Normal{Mu: 50, Sigma: 10}
    }
    
    context := &CorrelationContext{
        GameState:   "normal",
        ContestType: "gpp",
    }
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _ = matrix.GenerateCorrelatedOutcomes(players, distributions, context)
    }
}

func BenchmarkMonteCarloSimulation(b *testing.B) {
    config := &SimulationConfig{
        NumSimulations: 10000,
        Workers:        4,
        ContestType:    "gpp",
    }
    
    simulator := NewEnhancedMonteCarloSimulator(config)
    lineup := generateTestLineup()
    contest := generateTestContest()
    players := generateTestPlayers(200)
    
    ctx := context.Background()
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, _ = simulator.RunSimulation(ctx, lineup, contest, players)
    }
}
```

### Phase 7: API Integration

#### 7.1 Update API Types
Update `services/optimization-service/internal/api/handlers/simulation.go`:

```go
// Add new request/response types
type SimulationRequestV2 struct {
    LineupID              uint                      `json:"lineup_id"`
    ContestID             uint                      `json:"contest_id"`
    NumSimulations        int                       `json:"num_simulations"`
    SportConfig           *SportDistributionConfig  `json:"sport_config,omitempty"`
    ContestModel          *ContestModelConfig       `json:"contest_model,omitempty"`
    CorrelationSettings   *CorrelationSettings      `json:"correlation_settings,omitempty"`
    EventSimulation       *EventSimulationConfig    `json:"event_simulation,omitempty"`
    AdvancedAnalytics     bool                      `json:"enable_advanced_analytics"`
    CustomDistributions   map[uint]PlayerDistribution `json:"custom_distributions,omitempty"`
}

type SimulationResponseV2 struct {
    SimulationID          string                    `json:"simulation_id"`
    LineupID              uint                      `json:"lineup_id"`
    Iterations            int                       `json:"iterations"`
    ExecutionTime         string                    `json:"execution_time"`
    AdvancedMetrics       AdvancedSimulationMetrics `json:"advanced_metrics"`
    DistributionAnalysis  DistributionAnalysis      `json:"distribution_analysis"`
    CorrelationBreakdown  CorrelationBreakdown      `json:"correlation_breakdown"`
    EventAnalysis         EventAnalysis             `json:"event_analysis"`
    ContestProjections    ContestProjections        `json:"contest_projections"`
    RiskAssessment        RiskAssessment            `json:"risk_assessment"`
}

// Update handler
func (h *SimulationHandler) RunSimulationV2(c *gin.Context) {
    var req SimulationRequestV2
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }
    
    // Create enhanced simulator with config
    config := &SimulationConfig{
        NumSimulations:      req.NumSimulations,
        Workers:            4, // Auto-scale based on CPU
        SportConfig:        req.SportConfig,
        ContestModel:       req.ContestModel,
        CorrelationSettings: req.CorrelationSettings,
        EnableAdvanced:     req.AdvancedAnalytics,
    }
    
    simulator := simulator.NewEnhancedMonteCarloSimulator(config)
    
    // Run simulation
    result, err := simulator.RunSimulation(
        c.Request.Context(),
        lineup,
        contest,
        players,
    )
    
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, result)
}
```

## Validation Gates

```bash
# Run unit tests
cd services/optimization-service
go test ./internal/simulator/... -v

# Run integration tests
go test ./internal/simulator/... -tags=integration -v

# Run benchmarks
go test ./internal/simulator/... -bench=. -benchmem

# Check test coverage
go test ./internal/simulator/... -cover

# Lint code
golangci-lint run ./internal/simulator/...

# Statistical validation
go run cmd/validate/main.go --historical-data=./testdata/contest_results.csv
```

## External Documentation References

1. **Gonum Documentation**: https://www.gonum.org/
2. **Cholesky Decomposition for Correlations**: https://alexander-pastukhov.github.io/notes-on-statistics/advanced-04-cholesky.html
3. **Monte Carlo in DFS**: https://medium.com/@tdmiller89/how-the-monte-carlo-simulation-can-be-used-in-fantasy-sports-d53377fc04ff
4. **Statistical Distributions in Go**: https://pkg.go.dev/gonum.org/v1/gonum/stat/distuv

## Key Implementation Patterns

### From Existing Codebase

1. **Worker Pool Pattern** (from current monte_carlo.go)
   - Use buffered channels for work distribution
   - Configurable worker count based on CPU cores
   - Progress reporting via separate channel

2. **Interface-Based Design** (from optimizer package)
   - Define interfaces for extensibility
   - Allow pluggable distribution models
   - Support multiple correlation strategies

3. **Configuration Management** (from services pattern)
   - Use Viper for configuration
   - Environment-based overrides
   - Sensible defaults

4. **Error Handling** (from API handlers)
   - Wrap errors with context
   - Return structured error responses
   - Log errors with correlation IDs

## Performance Optimization Notes

1. **Pre-compute Correlation Matrices**
   - Cache Cholesky decomposition results
   - Use sparse matrices where applicable
   - Implement incremental updates

2. **Parallelize Simulations**
   - Use worker pools effectively
   - Batch operations for cache efficiency
   - Consider GPU acceleration for large-scale simulations

3. **Memory Management**
   - Use object pools for frequently allocated structures
   - Stream results instead of storing all iterations
   - Implement sampling for very large simulations

## Risk Mitigation

1. **Numerical Stability**
   - Add diagonal perturbation for ill-conditioned matrices
   - Validate distribution parameters
   - Handle edge cases gracefully

2. **Performance Degradation**
   - Implement circuit breakers
   - Auto-scale worker count
   - Fallback to simpler models if needed

3. **Data Quality**
   - Validate input data
   - Log anomalies
   - Maintain audit trail

## Success Criteria

- [ ] All distribution models pass Kolmogorov-Smirnov tests
- [ ] Correlation matrices maintain positive definiteness
- [ ] Simulation completes 10K iterations in <3 seconds
- [ ] Historical validation shows <5% variance from actual results
- [ ] API maintains backward compatibility
- [ ] 90%+ test coverage on new code

## Confidence Score: 9/10

The implementation plan is comprehensive with:
- Clear code examples following existing patterns
- Integration with proven libraries (Gonum)
- Detailed testing strategy
- Performance optimization approach
- Risk mitigation strategies

The only uncertainty is around specific sport distribution parameters which will need tuning based on historical data analysis during implementation.