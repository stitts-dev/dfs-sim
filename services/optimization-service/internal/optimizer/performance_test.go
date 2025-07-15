package optimizer

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stitts-dev/dfs-sim/shared/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BenchmarkDPOptimization_150Lineups validates the PRP requirement:
// Optimization of 150 lineups should complete in <3 seconds
func BenchmarkDPOptimization_150Lineups(b *testing.B) {
	// Create realistic test data
	players := createTestPlayers(100) // 100 players for realistic pool
	config := OptimizeConfigV2{
		SalaryCap:           50000,
		NumLineups:          150, // PRP requirement: 150 lineups
		MinDifferentPlayers: 4,
		UseCorrelations:     true,
		CorrelationWeight:   0.1,
		Contest:             createTestContest(),
		Strategy:            Balanced,
		PlayerAnalytics:     true,
		PerformanceMode:     "speed",
	}

	// Initialize DP optimizer
	optimizer := NewDPOptimizer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		result, err := optimizer.OptimizeWithDPV2(players, config)
		duration := time.Since(start)

		require.NoError(b, err)
		require.NotNil(b, result)
		require.Len(b, result, 150, "Should generate exactly 150 lineups")

		// PRP Requirement: Must complete in <3 seconds
		assert.Less(b, duration, 3*time.Second, "Optimization should complete in <3 seconds")

		b.Logf("Optimization completed in %v for %d lineups", duration, len(result))
	}
}

// TestDPOptimization_PerformanceValidation tests the specific PRP requirements
func TestDPOptimization_PerformanceValidation(t *testing.T) {
	players := createTestPlayers(80)
	config := OptimizeConfigV2{
		SalaryCap:           50000,
		NumLineups:          150,
		MinDifferentPlayers: 4,
		UseCorrelations:     true,
		CorrelationWeight:   0.1,
		Contest:             createTestContest(),
		Strategy:            Balanced,
		PlayerAnalytics:     true,
		PerformanceMode:     "speed",
	}

	optimizer := NewDPOptimizer()

	// Measure optimization time
	start := time.Now()
	result, err := optimizer.OptimizeWithDPV2(players, config)
	duration := time.Since(start)

	// Validate results
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 150, len(result), "Should generate exactly 150 lineups")

	// PRP Requirement 1: <3 second optimization time
	assert.Less(t, duration, 3*time.Second, "Optimization must complete in <3 seconds")

	// PRP Requirement 2: Valid lineups
	for i, lineup := range result {
		assert.Len(t, lineup.Players, 8, "Lineup %d should have 8 players", i)
		assert.LessOrEqual(t, lineup.TotalSalary, config.SalaryCap, "Lineup %d should respect salary cap", i)
		assert.Greater(t, lineup.ProjectedPoints, 0.0, "Lineup %d should have positive projected points", i)
	}

	// Log performance metrics
	t.Logf("Performance Validation Results:")
	t.Logf("- Optimization time: %v", duration)
	t.Logf("- Lineups generated: %d", len(result))
	if len(result) > 0 {
		t.Logf("- Average time per lineup: %v", duration/time.Duration(len(result)))
	} else {
		t.Logf("- Average time per lineup: N/A (no lineups generated)")
	}
	t.Logf("- PRP requirement (<3s): %t", duration < 3*time.Second)
}

// TestBackwardCompatibility validates that the original API still works
func TestBackwardCompatibility(t *testing.T) {
	players := createTestPlayers(50)
	config := OptimizeConfig{
		SalaryCap:           50000,
		NumLineups:          5,
		MinDifferentPlayers: 3,
		UseCorrelations:     true,
		CorrelationWeight:   0.1,
		Contest:             createTestContest(),
	}

	optimizer := NewDPOptimizer()

	start := time.Now()
	result, err := optimizer.OptimizeWithDP(players, config, "draftkings")
	duration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Greater(t, result.OptimalScore, 0.0)
	assert.NotEmpty(t, result.OptimalPlayers)

	t.Logf("Backward compatibility test completed in %v", duration)
}

// Helper functions

func createTestPlayers(count int) []types.Player {
	players := make([]types.Player, count)
	positions := []string{"PG", "SG", "SF", "PF", "C"}
	teams := []string{"LAL", "GSW", "BOS", "MIA", "DEN"}

	for i := 0; i < count; i++ {
		players[i] = types.Player{
			ID:              uuid.New(),
			Name:            fmt.Sprintf("Player_%d", i+1),
			Position:        positions[i%len(positions)],
			Team:            teams[i%len(teams)],
			SalaryDK:        4000 + (i%12)*500, // Range: 4000-9500
			ProjectedPoints: 20.0 + float64(i%30), // Range: 20-50
			CeilingPoints:   25.0 + float64(i%35), // Range: 25-60
			FloorPoints:     15.0 + float64(i%20), // Range: 15-35
		}
	}

	return players
}

func createTestContest() *types.Contest {
	return &types.Contest{
		ID:        uuid.New(),
		SportID:   uuid.New(),
		Platform:  "draftkings",
		SalaryCap: 50000,
		Name:      "Test Contest",
	}
}