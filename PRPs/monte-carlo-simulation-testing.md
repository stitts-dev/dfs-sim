name: "Monte Carlo Simulation Testing Framework"
description: |

## Purpose
Implement a comprehensive testing framework for the enhanced Monte Carlo simulation engine, covering unit tests for distribution models, integration tests for correlation engines, performance benchmarks, and validation against historical contest data.

## Core Principles
1. **Test Coverage**: Comprehensive testing of all simulation components
2. **Performance Validation**: Ensure simulations meet performance targets
3. **Accuracy Verification**: Validate statistical correctness of distributions
4. **Real-world Validation**: Test against historical contest data

---

## Goal
Create a robust testing framework that ensures the Monte Carlo simulation engine produces accurate, performant, and reliable results across all sports and contest types, with validation against historical data.

## Why
- **Accuracy**: Ensure simulation results match real-world contest variance patterns
- **Performance**: Maintain sub-3 second simulation times for 10K iterations
- **Reliability**: Catch edge cases and distribution anomalies before production
- **Confidence**: Provide measurable validation of simulation accuracy

## What
A comprehensive test suite including:
- Unit tests for distribution generators and correlation models
- Integration tests for end-to-end simulation flows
- Performance benchmarks for scalability validation
- Historical data validation framework
- Test data generators and fixtures

### Success Criteria
- [ ] 90%+ code coverage for simulation components
- [ ] All distribution models pass statistical tests (KS, Chi-square)
- [ ] Performance benchmarks meet targets (<3s for 10K iterations)
- [ ] Historical validation shows <5% variance from actual contest results
- [ ] CI/CD integration with automated test execution

## All Needed Context

### Current Implementation
```yaml
location: services/optimization-service/internal/simulator/
files:
  - monte_carlo.go: Core simulation engine
  - distributions.go: Distribution models
  - contest.go: Contest modeling

testing_gaps:
  - No dedicated test files for simulator package
  - Basic test coverage in optimizer tests only
  - No performance benchmarks
  - No historical validation framework
```

### PRD Reference
From PRD-monte-carlo-simulation-upgrade.md:
- Distribution models: Normal, LogNormal, Beta, Gamma, Exponential
- Sport-specific configurations for NBA, NFL, Golf, MLB
- Advanced correlation modeling with context awareness
- Event simulation (injuries, weather, game script)

### Testing Patterns in Codebase
From internal/optimizer/algorithm_test.go:
- Table-driven tests for multiple scenarios
- Benchmark tests for performance validation
- Helper functions for test data generation
- Assert/require patterns for validation

## Implementation Blueprint

### 1. Distribution Model Unit Tests

```go
// services/optimization-service/internal/simulator/distributions_test.go
package simulator

import (
    "math"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "gonum.org/v1/gonum/stat"
)

func TestNormalDistribution(t *testing.T) {
    tests := []struct {
        name           string
        mean           float64
        stdDev         float64
        samples        int
        toleranceMean  float64
        toleranceStdDev float64
    }{
        {
            name:           "NBA_Player_Standard",
            mean:           45.0,
            stdDev:         11.25, // 25% variance
            samples:        10000,
            toleranceMean:  0.5,
            toleranceStdDev: 0.5,
        },
        {
            name:           "NFL_QB_HighVariance",
            mean:           22.0,
            stdDev:         8.0,
            samples:        10000,
            toleranceMean:  0.3,
            toleranceStdDev: 0.3,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dist := NewNormalDistribution(tt.mean, tt.stdDev)
            samples := make([]float64, tt.samples)
            
            for i := 0; i < tt.samples; i++ {
                samples[i] = dist.Generate()
            }
            
            // Validate statistical properties
            sampleMean := stat.Mean(samples, nil)
            sampleStdDev := stat.StdDev(samples, nil)
            
            assert.InDelta(t, tt.mean, sampleMean, tt.toleranceMean)
            assert.InDelta(t, tt.stdDev, sampleStdDev, tt.toleranceStdDev)
            
            // Kolmogorov-Smirnov test for distribution fit
            ksStatistic := calculateKSStatistic(samples, dist)
            assert.Less(t, ksStatistic, 0.05, "Distribution should pass KS test")
        })
    }
}

func TestBetaDistribution_Golf(t *testing.T) {
    // Golf-specific beta distribution for cut probability
    dist := NewBetaDistribution(2.0, 5.0) // Parameters for typical golf score distribution
    
    samples := make([]float64, 10000)
    for i := range samples {
        samples[i] = dist.Generate()
    }
    
    // Validate properties specific to golf scoring
    mean := stat.Mean(samples, nil)
    assert.InDelta(t, 0.286, mean, 0.01) // Expected mean for Beta(2,5)
    
    // Check bounds
    for _, sample := range samples {
        assert.GreaterOrEqual(t, sample, 0.0)
        assert.LessOrEqual(t, sample, 1.0)
    }
}

func TestLogNormalDistribution_NFL_RB(t *testing.T) {
    // NFL RB scoring follows log-normal due to big play potential
    dist := NewLogNormalDistribution(2.5, 0.8)
    
    samples := make([]float64, 10000)
    for i := range samples {
        samples[i] = dist.Generate()
    }
    
    // Validate long tail for boom performances
    percentile95 := stat.Quantile(0.95, stat.Empirical, samples, nil)
    percentile50 := stat.Quantile(0.50, stat.Empirical, samples, nil)
    
    // 95th percentile should be significantly higher than median
    assert.Greater(t, percentile95/percentile50, 2.5)
}

// Benchmark distribution generation performance
func BenchmarkDistributionGeneration(b *testing.B) {
    distributions := map[string]Distribution{
        "Normal":    NewNormalDistribution(50.0, 12.5),
        "Beta":      NewBetaDistribution(2.0, 5.0),
        "LogNormal": NewLogNormalDistribution(3.0, 0.5),
        "Gamma":     NewGammaDistribution(2.0, 2.0),
    }
    
    for name, dist := range distributions {
        b.Run(name, func(b *testing.B) {
            for i := 0; i < b.N; i++ {
                _ = dist.Generate()
            }
        })
    }
}
```

### 2. Correlation Engine Integration Tests

```go
// services/optimization-service/internal/simulator/correlation_test.go
package simulator

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stitts-dev/dfs-sim/shared/types"
)

func TestAdvancedCorrelationMatrix(t *testing.T) {
    // Test game stack correlations
    players := []types.Player{
        {ID: 1, Name: "Mahomes", Position: "QB", Team: "KC"},
        {ID: 2, Name: "Kelce", Position: "TE", Team: "KC"},
        {ID: 3, Name: "Hill", Position: "WR", Team: "KC"},
        {ID: 4, Name: "Jefferson", Position: "WR", Team: "MIN"},
    }
    
    matrix := NewAdvancedCorrelationMatrix(players)
    
    // QB-pass catcher correlation should be high
    qbTECorr := matrix.GetCorrelation(1, 2)
    assert.Greater(t, qbTECorr, 0.3, "QB-TE correlation should be positive")
    
    // Same team correlation
    teWRCorr := matrix.GetCorrelation(2, 3)
    assert.Greater(t, teWRCorr, 0.1, "Same team correlation should exist")
    
    // Different team, no game stack
    crossTeamCorr := matrix.GetCorrelation(3, 4)
    assert.InDelta(t, 0.0, crossTeamCorr, 0.05, "Different team correlation should be near zero")
}

func TestContextAwareCorrelations(t *testing.T) {
    players := createTestPlayers()
    matrix := NewAdvancedCorrelationMatrix(players)
    
    tests := []struct {
        name     string
        context  CorrelationContext
        player1  uint
        player2  uint
        expected float64
        delta    float64
    }{
        {
            name: "Blowout_Negative_Correlation",
            context: CorrelationContext{
                GameState: "blowout",
            },
            player1:  1, // Starting RB
            player2:  2, // Backup RB same team
            expected: -0.2,
            delta:    0.05,
        },
        {
            name: "Weather_Increased_RB_Correlation",
            context: CorrelationContext{
                WeatherConditions: "heavy_rain",
            },
            player1:  3, // RB1
            player2:  4, // RB2 opposing team
            expected: 0.15,
            delta:    0.05,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            corr := matrix.GetContextualCorrelation(tt.player1, tt.player2, tt.context)
            assert.InDelta(t, tt.expected, corr, tt.delta)
        })
    }
}

func TestCorrelationPerformance(t *testing.T) {
    // Create large player pool
    players := make([]types.Player, 500)
    for i := range players {
        players[i] = types.Player{
            ID:       uint(i + 1),
            Name:     fmt.Sprintf("Player%d", i),
            Position: positions[i%len(positions)],
            Team:     teams[i%len(teams)],
        }
    }
    
    matrix := NewAdvancedCorrelationMatrix(players)
    
    // Measure correlation lookup performance
    start := time.Now()
    iterations := 100000
    
    for i := 0; i < iterations; i++ {
        p1 := uint(rand.Intn(500) + 1)
        p2 := uint(rand.Intn(500) + 1)
        _ = matrix.GetCorrelation(p1, p2)
    }
    
    elapsed := time.Since(start)
    avgLookup := elapsed / time.Duration(iterations)
    
    assert.Less(t, avgLookup, 100*time.Nanosecond, "Correlation lookup should be < 100ns")
}
```

### 3. Monte Carlo Engine Integration Tests

```go
// services/optimization-service/internal/simulator/monte_carlo_integration_test.go
package simulator

import (
    "context"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMonteCarloSimulation_FullFlow(t *testing.T) {
    // Create test lineup with known players
    lineup := createTestLineup()
    contest := types.Contest{
        ID:        1,
        Sport:     "nfl",
        Type:      "gpp",
        EntryFee:  3.0,
        SalaryCap: 50000,
    }
    
    config := &SimulationConfig{
        NumSimulations:    1000,
        SimulationWorkers: 4,
        UseCorrelations:   true,
        ContestSize:       5000,
        EntryFee:         3.0,
        PayoutStructure:  createGPPPayoutStructure(),
    }
    
    simulator := NewSimulator(config, lineup.Players)
    
    ctx := context.Background()
    progressChan := make(chan SimulationProgress, 10)
    
    // Run simulation
    start := time.Now()
    result, err := simulator.SimulateContest([]types.GeneratedLineup{lineup}, progressChan)
    elapsed := time.Since(start)
    
    require.NoError(t, err)
    assert.NotNil(t, result)
    
    // Validate performance
    assert.Less(t, elapsed, 500*time.Millisecond, "1K iterations should complete < 500ms")
    
    // Validate results
    assert.Equal(t, 1000, result.NumSimulations)
    assert.Greater(t, result.Mean, 0.0)
    assert.Greater(t, result.StandardDeviation, 0.0)
    assert.Less(t, result.Min, result.Max)
    
    // Validate percentiles are ordered correctly
    assert.Less(t, result.Percentile25, result.Median)
    assert.Less(t, result.Median, result.Percentile75)
    assert.Less(t, result.Percentile75, result.Percentile90)
    
    // Validate contest-specific metrics
    assert.GreaterOrEqual(t, result.CashProbability, 0.0)
    assert.LessOrEqual(t, result.CashProbability, 100.0)
    assert.GreaterOrEqual(t, result.WinProbability, 0.0)
    assert.LessOrEqual(t, result.WinProbability, 100.0)
}

func TestSimulationProgress(t *testing.T) {
    lineup := createTestLineup()
    config := &SimulationConfig{
        NumSimulations:    100,
        SimulationWorkers: 2,
    }
    
    simulator := NewSimulator(config, lineup.Players)
    progressChan := make(chan SimulationProgress, 100)
    
    go func() {
        _, _ = simulator.SimulateContest([]types.GeneratedLineup{lineup}, progressChan)
    }()
    
    progressUpdates := []SimulationProgress{}
    for progress := range progressChan {
        progressUpdates = append(progressUpdates, progress)
        if progress.Completed >= config.NumSimulations {
            break
        }
    }
    
    // Validate progress reporting
    assert.Greater(t, len(progressUpdates), 0, "Should receive progress updates")
    
    // Progress should increase monotonically
    for i := 1; i < len(progressUpdates); i++ {
        assert.GreaterOrEqual(t, progressUpdates[i].Completed, progressUpdates[i-1].Completed)
    }
    
    // Final progress should be 100%
    lastUpdate := progressUpdates[len(progressUpdates)-1]
    assert.Equal(t, config.NumSimulations, lastUpdate.Completed)
}

func TestParallelSimulations(t *testing.T) {
    // Test multiple lineups simulated in parallel
    lineups := make([]types.GeneratedLineup, 10)
    for i := range lineups {
        lineups[i] = createTestLineupWithVariance(i)
    }
    
    config := &SimulationConfig{
        NumSimulations:    1000,
        SimulationWorkers: 8,
        UseCorrelations:   true,
    }
    
    results := make([]*SimulationResult, len(lineups))
    errors := make([]error, len(lineups))
    
    // Run simulations in parallel
    var wg sync.WaitGroup
    for i, lineup := range lineups {
        wg.Add(1)
        go func(idx int, l types.GeneratedLineup) {
            defer wg.Done()
            simulator := NewSimulator(config, l.Players)
            results[idx], errors[idx] = simulator.SimulateContest([]types.GeneratedLineup{l}, nil)
        }(i, lineup)
    }
    
    wg.Wait()
    
    // Validate all succeeded
    for i, err := range errors {
        require.NoError(t, err, "Simulation %d should succeed", i)
        assert.NotNil(t, results[i])
    }
    
    // Results should vary based on lineup composition
    var means []float64
    for _, result := range results {
        means = append(means, result.Mean)
    }
    
    // Check variance in results
    meanVariance := stat.Variance(means, nil)
    assert.Greater(t, meanVariance, 0.0, "Different lineups should produce different results")
}
```

### 4. Historical Validation Framework

```go
// services/optimization-service/internal/simulator/historical_validation_test.go
package simulator

import (
    "encoding/csv"
    "os"
    "testing"
    "github.com/stretchr/testify/assert"
)

type HistoricalContestResult struct {
    ContestID      string
    LineupID       string
    ActualScore    float64
    ActualFinish   int
    TotalEntries   int
    Payout         float64
}

func TestHistoricalValidation_NFL_GPP(t *testing.T) {
    // Load historical contest results
    results := loadHistoricalResults("testdata/nfl_gpp_results.csv")
    require.NotEmpty(t, results)
    
    // Group by contest
    contestGroups := groupByContest(results)
    
    accuracyMetrics := []float64{}
    
    for contestID, contestResults := range contestGroups {
        // Recreate contest conditions
        lineup := recreateLineupFromHistorical(contestResults[0])
        config := &SimulationConfig{
            NumSimulations:    10000,
            SimulationWorkers: 8,
            ContestSize:       contestResults[0].TotalEntries,
            PayoutStructure:   loadPayoutStructure(contestID),
        }
        
        simulator := NewSimulator(config, lineup.Players)
        simResult, err := simulator.SimulateContest([]types.GeneratedLineup{lineup}, nil)
        require.NoError(t, err)
        
        // Compare simulated vs actual
        actualPercentile := float64(contestResults[0].ActualFinish) / float64(contestResults[0].TotalEntries) * 100
        
        // Find closest simulated percentile
        var closestDiff float64 = 100.0
        percentiles := []float64{
            simResult.Percentile25,
            simResult.Median, 
            simResult.Percentile75,
            simResult.Percentile90,
            simResult.Percentile95,
        }
        
        for _, p := range percentiles {
            diff := math.Abs(p - actualPercentile)
            if diff < closestDiff {
                closestDiff = diff
            }
        }
        
        accuracyMetrics = append(accuracyMetrics, closestDiff)
    }
    
    // Calculate overall accuracy
    meanError := stat.Mean(accuracyMetrics, nil)
    assert.Less(t, meanError, 5.0, "Mean percentile error should be < 5%")
    
    // 90% of predictions within 10 percentile points
    within10 := 0
    for _, err := range accuracyMetrics {
        if err <= 10.0 {
            within10++
        }
    }
    accuracy := float64(within10) / float64(len(accuracyMetrics)) * 100
    assert.Greater(t, accuracy, 90.0, "90% of predictions should be within 10 percentile points")
}

func TestSportSpecificValidation(t *testing.T) {
    sports := []string{"nfl", "nba", "mlb", "golf"}
    
    for _, sport := range sports {
        t.Run(sport, func(t *testing.T) {
            results := loadHistoricalResults(fmt.Sprintf("testdata/%s_results.csv", sport))
            if len(results) == 0 {
                t.Skip("No historical data for", sport)
            }
            
            validateSportDistributions(t, sport, results)
        })
    }
}

func validateSportDistributions(t *testing.T, sport string, results []HistoricalContestResult) {
    // Extract actual score distributions
    actualScores := make([]float64, len(results))
    for i, r := range results {
        actualScores[i] = r.ActualScore
    }
    
    // Get expected distribution parameters for sport
    distConfig := getSportDistributionConfig(sport)
    
    // Fit distribution to actual data
    fittedParams := fitDistribution(actualScores, distConfig.DefaultDistribution)
    
    // Validate fitted parameters match expected ranges
    switch sport {
    case "nfl":
        // NFL should have higher variance
        variance := fittedParams["variance"]
        assert.Greater(t, variance, 0.25, "NFL variance should be > 25%")
        
    case "golf":
        // Golf should have cut probability
        cutRate := calculateCutRate(actualScores)
        assert.Greater(t, cutRate, 0.3, "Golf cut rate should be > 30%")
        assert.Less(t, cutRate, 0.6, "Golf cut rate should be < 60%")
    }
}
```

### 5. Performance Benchmark Suite

```go
// services/optimization-service/internal/simulator/benchmarks_test.go
package simulator

import (
    "testing"
    "time"
)

func BenchmarkMonteCarloSimulation(b *testing.B) {
    sizes := []struct {
        name       string
        iterations int
        workers    int
    }{
        {"Small_1K_4Workers", 1000, 4},
        {"Medium_10K_4Workers", 10000, 4},
        {"Medium_10K_8Workers", 10000, 8},
        {"Large_100K_8Workers", 100000, 8},
        {"Large_100K_16Workers", 100000, 16},
    }
    
    for _, size := range sizes {
        b.Run(size.name, func(b *testing.B) {
            lineup := createLargeTestLineup(150) // 150 players
            config := &SimulationConfig{
                NumSimulations:    size.iterations,
                SimulationWorkers: size.workers,
                UseCorrelations:   true,
            }
            
            simulator := NewSimulator(config, lineup.Players)
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _, err := simulator.SimulateContest([]types.GeneratedLineup{lineup}, nil)
                if err != nil {
                    b.Fatal(err)
                }
            }
            
            // Report useful metrics
            b.ReportMetric(float64(size.iterations)/b.Elapsed().Seconds(), "sims/sec")
        })
    }
}

func BenchmarkCorrelationMatrix(b *testing.B) {
    playerCounts := []int{50, 100, 200, 500, 1000}
    
    for _, count := range playerCounts {
        b.Run(fmt.Sprintf("Players_%d", count), func(b *testing.B) {
            players := generateTestPlayers(count)
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _ = NewAdvancedCorrelationMatrix(players)
            }
        })
    }
}

func BenchmarkDistributionGeneration_Parallel(b *testing.B) {
    workerCounts := []int{1, 2, 4, 8, 16}
    
    for _, workers := range workerCounts {
        b.Run(fmt.Sprintf("Workers_%d", workers), func(b *testing.B) {
            dist := NewNormalDistribution(50.0, 12.5)
            samplesPerWorker := 10000
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                var wg sync.WaitGroup
                for w := 0; w < workers; w++ {
                    wg.Add(1)
                    go func() {
                        defer wg.Done()
                        for s := 0; s < samplesPerWorker; s++ {
                            _ = dist.Generate()
                        }
                    }()
                }
                wg.Wait()
            }
            
            totalSamples := workers * samplesPerWorker
            b.ReportMetric(float64(totalSamples*b.N)/b.Elapsed().Seconds(), "samples/sec")
        })
    }
}

// Memory allocation benchmarks
func BenchmarkMemoryUsage(b *testing.B) {
    configs := []struct {
        name       string
        players    int
        iterations int
    }{
        {"Small", 50, 1000},
        {"Medium", 150, 10000},
        {"Large", 300, 100000},
    }
    
    for _, cfg := range configs {
        b.Run(cfg.name, func(b *testing.B) {
            lineup := createLargeTestLineup(cfg.players)
            config := &SimulationConfig{
                NumSimulations:    cfg.iterations,
                SimulationWorkers: 8,
            }
            
            b.ResetTimer()
            b.ReportAllocs()
            
            for i := 0; i < b.N; i++ {
                simulator := NewSimulator(config, lineup.Players)
                _, _ = simulator.SimulateContest([]types.GeneratedLineup{lineup}, nil)
            }
        })
    }
}
```

### 6. Test Utilities and Fixtures

```go
// services/optimization-service/internal/simulator/test_utils.go
package simulator

import (
    "math/rand"
    "time"
    "github.com/stitts-dev/dfs-sim/shared/types"
)

// Test data generators
func createTestLineup() types.GeneratedLineup {
    return types.GeneratedLineup{
        ID: "test-lineup-1",
        Players: []types.LineupPlayer{
            {ID: 1, Name: "Mahomes", Position: "QB", Team: "KC", Salary: 8000, ProjectedPoints: 25.0},
            {ID: 2, Name: "Cook", Position: "RB", Team: "MIN", Salary: 8500, ProjectedPoints: 20.0},
            {ID: 3, Name: "Henry", Position: "RB", Team: "TEN", Salary: 7500, ProjectedPoints: 18.0},
            {ID: 4, Name: "Adams", Position: "WR", Team: "LV", Salary: 8000, ProjectedPoints: 19.0},
            {ID: 5, Name: "Hill", Position: "WR", Team: "KC", Salary: 8200, ProjectedPoints: 21.0},
            {ID: 6, Name: "Diggs", Position: "WR", Team: "BUF", Salary: 7800, ProjectedPoints: 18.5},
            {ID: 7, Name: "Kelce", Position: "TE", Team: "KC", Salary: 7000, ProjectedPoints: 17.0},
            {ID: 8, Name: "Chase", Position: "WR", Team: "CIN", Salary: 6500, ProjectedPoints: 16.0},
            {ID: 9, Name: "Cowboys", Position: "DST", Team: "DAL", Salary: 3500, ProjectedPoints: 9.0},
        },
        TotalSalary:     50000,
        ProjectedPoints: 163.5,
    }
}

func createTestLineupWithVariance(seed int) types.GeneratedLineup {
    rand.Seed(int64(seed))
    lineup := createTestLineup()
    
    // Add variance to projections
    for i := range lineup.Players {
        variance := rand.Float64() * 10 - 5 // -5 to +5 variance
        lineup.Players[i].ProjectedPoints += variance
    }
    
    return lineup
}

func generateTestPlayers(count int) []types.Player {
    positions := []string{"QB", "RB", "WR", "TE", "DST"}
    teams := []string{"KC", "BUF", "TB", "GB", "DAL", "SF", "LAR", "TEN"}
    
    players := make([]types.Player, count)
    for i := 0; i < count; i++ {
        players[i] = types.Player{
            ID:              uint(i + 1),
            Name:            fmt.Sprintf("Player%d", i+1),
            Position:        positions[i%len(positions)],
            Team:            teams[i%len(teams)],
            Salary:          3000 + rand.Intn(7000),
            ProjectedPoints: 5.0 + rand.Float64()*25.0,
            Floor:           0.0,
            Ceiling:         0.0,
        }
        
        // Calculate floor/ceiling
        players[i].Floor = players[i].ProjectedPoints * 0.7
        players[i].Ceiling = players[i].ProjectedPoints * 1.5
    }
    
    return players
}

// Statistical test helpers
func calculateKSStatistic(samples []float64, dist Distribution) float64 {
    // Simplified KS statistic calculation
    n := len(samples)
    sort.Float64s(samples)
    
    maxDiff := 0.0
    for i, sample := range samples {
        empiricalCDF := float64(i+1) / float64(n)
        theoreticalCDF := dist.CDF(sample)
        diff := math.Abs(empiricalCDF - theoreticalCDF)
        if diff > maxDiff {
            maxDiff = diff
        }
    }
    
    return maxDiff
}

// Payout structure generators
func createGPPPayoutStructure() []PayoutTier {
    return []PayoutTier{
        {MinRank: 1, MaxRank: 1, Payout: 1000000},
        {MinRank: 2, MaxRank: 2, Payout: 500000},
        {MinRank: 3, MaxRank: 3, Payout: 250000},
        {MinRank: 4, MaxRank: 10, Payout: 10000},
        {MinRank: 11, MaxRank: 100, Payout: 1000},
        {MinRank: 101, MaxRank: 1000, Payout: 100},
        {MinRank: 1001, MaxRank: 10000, Payout: 10},
    }
}

func createCashPayoutStructure() []PayoutTier {
    // Top 50% double their money
    return []PayoutTier{
        {MinRank: 1, MaxRank: 2500, Payout: 6.0}, // Double the $3 entry
    }
}
```

### 7. CI/CD Integration

```yaml
# .github/workflows/simulation-tests.yml
name: Simulation Tests

on:
  push:
    paths:
      - 'services/optimization-service/internal/simulator/**'
      - 'services/optimization-service/internal/optimizer/**'
  pull_request:
    paths:
      - 'services/optimization-service/internal/simulator/**'
      - 'services/optimization-service/internal/optimizer/**'

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install dependencies
      run: |
        cd services/optimization-service
        go mod download
    
    - name: Run unit tests with coverage
      run: |
        cd services/optimization-service
        go test -v -race -coverprofile=coverage.out ./internal/simulator/...
        go tool cover -html=coverage.out -o coverage.html
    
    - name: Run benchmarks
      run: |
        cd services/optimization-service
        go test -bench=. -benchmem ./internal/simulator/... | tee benchmark.txt
    
    - name: Check performance regression
      run: |
        # Compare benchmark results with baseline
        # Fail if performance degrades > 10%
        ./scripts/check-performance-regression.sh benchmark.txt
    
    - name: Run historical validation
      run: |
        cd services/optimization-service
        go test -v -tags=historical ./internal/simulator/... -run TestHistorical
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./services/optimization-service/coverage.out
        flags: simulator
    
    - name: Upload benchmark results
      uses: actions/upload-artifact@v3
      with:
        name: benchmark-results
        path: ./services/optimization-service/benchmark.txt
```

## Validation Requirements

### Statistical Validation
1. **Distribution Tests**: KS test, Chi-square test for distribution fit
2. **Correlation Validation**: Compare simulated vs actual player correlations
3. **Variance Analysis**: Ensure sport-specific variance patterns are maintained
4. **Edge Case Testing**: Injury events, weather impacts, extreme performances

### Performance Validation
1. **Speed Benchmarks**: Sub-3 second for 10K iterations
2. **Memory Profiling**: < 100MB for typical simulation runs
3. **Scalability Tests**: Linear scaling with worker count
4. **Concurrent Load**: Handle 100 simultaneous simulations

### Historical Validation
1. **Contest Recreation**: Replay historical contests with known outcomes
2. **Accuracy Metrics**: < 5% mean error on percentile predictions
3. **Sport-Specific**: Validate each sport's unique characteristics
4. **Continuous Updates**: Weekly validation against new contest data

## Risk Mitigation

### Test Data Management
- **Synthetic Data**: Generate representative test data for all scenarios
- **Historical Data**: Sanitized contest results for validation
- **Edge Cases**: Specific test cases for rare events
- **Data Versioning**: Track test data changes with migrations

### Performance Monitoring
- **Benchmark Tracking**: Store benchmark history for regression detection
- **Memory Profiling**: Regular heap analysis to catch leaks
- **CPU Profiling**: Identify hot paths and optimization opportunities
- **Automated Alerts**: Notify on performance degradation

## Success Metrics

### Code Quality
- [ ] 90%+ test coverage for simulation package
- [ ] All critical paths have integration tests
- [ ] Performance benchmarks for all major operations
- [ ] Historical validation suite operational

### Accuracy Metrics
- [ ] Distribution tests pass for all sport configurations
- [ ] Correlation accuracy within 5% of historical data
- [ ] Contest outcome predictions within target accuracy
- [ ] Edge cases properly handled and tested

### Performance Metrics
- [ ] 10K simulations complete in < 3 seconds
- [ ] Memory usage < 100MB for standard runs
- [ ] Linear scalability with worker count
- [ ] No performance regression in CI/CD

## Next Steps

1. **Implement Core Test Suite**: Start with distribution and correlation tests
2. **Historical Data Collection**: Gather and sanitize contest results
3. **Performance Baseline**: Establish current performance benchmarks
4. **CI/CD Integration**: Automate test execution and monitoring
5. **Documentation**: Create testing guidelines and best practices

---

**Priority**: High
**Estimated Effort**: 3-4 weeks
**Dependencies**: 
- Monte Carlo simulation engine implementation
- Historical contest data access
- CI/CD pipeline setup
**Team**: 1 senior engineer, 1 QA engineer