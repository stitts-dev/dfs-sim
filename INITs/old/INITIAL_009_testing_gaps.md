## FEATURE:

Implement comprehensive testing suite for golf optimization including unit tests, integration tests, performance benchmarks, and API mocks

## EXAMPLES:

Missing test coverage areas:

### 1. Golf-Specific Unit Tests
```go
// backend/internal/optimizer/golf_optimizer_test.go
func TestGolfLineupGeneration(t *testing.T) {
    tests := []struct {
        name     string
        players  []Player
        contest  Contest
        expected LineupCharacteristics
    }{
        {
            name: "Should respect cut line probability",
            players: generateTestGolfers(100),
            contest: Contest{MaxSalary: 50000},
            expected: LineupCharacteristics{
                MinCutProbability: 0.6,
                MaxSalary: 50000,
            },
        },
        {
            name: "Should create diverse lineups",
            players: generateTestGolfers(150),
            contest: Contest{NumLineups: 20},
            expected: LineupCharacteristics{
                MinUniquePlayers: 40, // At least 40 unique players across 20 lineups
            },
        },
    }
}

func TestGolfCorrelationMatrix(t *testing.T) {
    // Test tee time correlations
    // Test weather impact correlations
    // Test course history correlations
}

func TestCutLinePrediction(t *testing.T) {
    // Test various tournament conditions
    // Test confidence scoring
    // Test historical accuracy
}
```

### 2. Integration Tests
```go
// backend/tests/golf_integration_test.go
func TestFullGolfOptimizationFlow(t *testing.T) {
    // 1. Fetch golf data from provider
    // 2. Generate projections
    // 3. Build correlation matrix
    // 4. Run optimization
    // 5. Validate results
    // 6. Export lineups
    
    ctx := context.Background()
    
    // Setup test tournament
    tournament := setupTestTournament()
    
    // Fetch data
    provider := rapidapi.NewGolfProvider(testConfig)
    players, err := provider.GetTournamentField(ctx, tournament.ID)
    require.NoError(t, err)
    
    // Generate projections
    projections := generateProjections(players, tournament)
    
    // Optimize
    optimizer := NewGolfOptimizer(optimizerConfig)
    lineups, err := optimizer.Optimize(projections, constraints)
    require.NoError(t, err)
    
    // Validate
    assert.Len(t, lineups, 20)
    assert.True(t, allLineupsValid(lineups))
    assert.True(t, lineupsAreDiverse(lineups))
}
```

### 3. Performance Benchmarks
```go
// backend/internal/optimizer/benchmark_test.go
func BenchmarkGolfOptimization(b *testing.B) {
    scenarios := []struct {
        name        string
        playerCount int
        lineupCount int
    }{
        {"Small_Field", 50, 10},
        {"Normal_Field", 150, 20},
        {"Large_Field", 200, 50},
        {"Max_Lineups", 150, 150},
    }
    
    for _, scenario := range scenarios {
        b.Run(scenario.name, func(b *testing.B) {
            players := generateBenchmarkPlayers(scenario.playerCount)
            optimizer := NewOptimizer()
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _, _ = optimizer.Optimize(players, scenario.lineupCount)
            }
            
            b.ReportMetric(float64(b.Elapsed())/float64(b.N), "ns/lineup")
        })
    }
}
```

### 4. API Mock Tests
```go
// backend/internal/providers/rapidapi_golf_mock_test.go
type MockGolfProvider struct {
    mock.Mock
}

func (m *MockGolfProvider) GetTournamentLeaderboard(ctx context.Context, id string) (*Leaderboard, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(*Leaderboard), args.Error(1)
}

func TestOptimizationWithMockedData(t *testing.T) {
    mockProvider := new(MockGolfProvider)
    mockProvider.On("GetTournamentLeaderboard", mock.Anything, "test-tournament").
        Return(&Leaderboard{
            Players: generateMockLeaderboard(),
        }, nil)
    
    // Test optimization with mocked data
}
```

### 5. Property-Based Tests
```go
func TestGolfLineupProperties(t *testing.T) {
    quick.Check(func(playerCount, lineupCount int) bool {
        if playerCount < 6 || playerCount > 200 {
            return true // Skip invalid inputs
        }
        
        players := generateRandomPlayers(playerCount)
        lineups := optimize(players, lineupCount)
        
        // Properties that should always hold:
        // 1. All lineups have 6 players
        // 2. All lineups under salary cap
        // 3. No duplicate players in lineup
        // 4. All players from valid player pool
        
        return validateProperties(lineups, players)
    }, nil)
}
```

## DOCUMENTATION:

- Go testing best practices: https://golang.org/doc/tutorial/add-a-test
- Table-driven tests: https://dave.cheney.net/2019/05/07/prefer-table-driven-tests
- Testify framework: https://github.com/stretchr/testify
- Go benchmarking: https://golang.org/pkg/testing/#hdr-Benchmarks

## OTHER CONSIDERATIONS:

- Current test coverage is minimal for golf features
- No integration tests for full optimization flow
- Missing performance benchmarks
- No load testing for concurrent optimizations
- API mocks not implemented (hitting real APIs in tests)
- No property-based testing
- Missing edge case coverage
- No regression test suite
- Frontend component tests missing
- E2E tests not implemented
- No visual regression tests
- Missing API contract tests
- No chaos/failure testing