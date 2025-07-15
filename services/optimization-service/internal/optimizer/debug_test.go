package optimizer

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// TestSimpleOptimization tests basic optimization functionality
func TestSimpleOptimization(t *testing.T) {
	// Create a minimal set of test players that should easily create valid lineups
	players := []types.Player{
		{ID: uuid.New(), Name: "PG1", Position: "PG", SalaryDK: 5000, ProjectedPoints: 30.0, Team: "LAL"},
		{ID: uuid.New(), Name: "SG1", Position: "SG", SalaryDK: 5000, ProjectedPoints: 30.0, Team: "GSW"},
		{ID: uuid.New(), Name: "SF1", Position: "SF", SalaryDK: 5000, ProjectedPoints: 30.0, Team: "BOS"},
		{ID: uuid.New(), Name: "PF1", Position: "PF", SalaryDK: 5000, ProjectedPoints: 30.0, Team: "MIA"},
		{ID: uuid.New(), Name: "C1", Position: "C", SalaryDK: 5000, ProjectedPoints: 30.0, Team: "DEN"},
		{ID: uuid.New(), Name: "PG2", Position: "PG", SalaryDK: 5000, ProjectedPoints: 25.0, Team: "PHX"}, // For G slot
		{ID: uuid.New(), Name: "SF2", Position: "SF", SalaryDK: 5000, ProjectedPoints: 25.0, Team: "NYK"}, // For F slot
		{ID: uuid.New(), Name: "PG3", Position: "PG", SalaryDK: 5000, ProjectedPoints: 20.0, Team: "CHA"}, // For UTIL slot
	}

	config := OptimizeConfigV2{
		SalaryCap:           50000, // 8 * 5000 = 40000, well within budget
		NumLineups:          1,     // Just try to generate 1 lineup
		MinDifferentPlayers: 0,
		UseCorrelations:     false, // Disable correlations for simplicity
		Contest:             createTestContest(),
		Strategy:            Balanced,
		PlayerAnalytics:     false, // Disable analytics for simplicity
		PerformanceMode:     "speed",
	}

	optimizer := NewDPOptimizer()
	result, err := optimizer.OptimizeWithDPV2(players, config)

	t.Logf("Simple optimization test:")
	t.Logf("- Players provided: %d", len(players))
	t.Logf("- Total salary if all selected: %d", len(players)*5000)
	t.Logf("- Salary cap: %d", config.SalaryCap)
	t.Logf("- Error: %v", err)
	t.Logf("- Lineups generated: %d", len(result))

	if err != nil {
		t.Logf("Error details: %v", err)
	}

	if len(result) > 0 {
		lineup := result[0]
		t.Logf("First lineup:")
		t.Logf("- Players: %d", len(lineup.Players))
		t.Logf("- Total salary: %d", lineup.TotalSalary)
		t.Logf("- Projected points: %.2f", lineup.ProjectedPoints)
		
		for i, player := range lineup.Players {
			t.Logf("  %d. %s (%s) - %d salary, %.1f points", 
				i+1, player.Name, player.Position, player.Salary, player.ProjectedPoints)
		}
	} else {
		t.Logf("No lineups generated!")
		
		// Debug: Let's check position slots
		slots := GetPositionSlots("nba", "draftkings")
		t.Logf("Position slots available:")
		for _, slot := range slots {
			t.Logf("- %s (allowed: %v, priority: %d)", slot.SlotName, slot.AllowedPositions, slot.Priority)
		}
		
		// Debug: Check if players can fill slots
		t.Logf("Player eligibility check:")
		for _, player := range players {
			eligibleSlots := []string{}
			for _, slot := range slots {
				if CanPlayerFillSlot(player, slot) {
					eligibleSlots = append(eligibleSlots, slot.SlotName)
				}
			}
			t.Logf("- %s (%s): can fill %v", player.Name, player.Position, eligibleSlots)
		}
	}
}

// TestBackwardCompatibilitySimple tests the original API with simple data
func TestBackwardCompatibilitySimple(t *testing.T) {
	players := []types.Player{
		{ID: uuid.New(), Name: "PG1", Position: "PG", SalaryDK: 5000, ProjectedPoints: 30.0, Team: "LAL"},
		{ID: uuid.New(), Name: "SG1", Position: "SG", SalaryDK: 5000, ProjectedPoints: 30.0, Team: "GSW"},
		{ID: uuid.New(), Name: "SF1", Position: "SF", SalaryDK: 5000, ProjectedPoints: 30.0, Team: "BOS"},
		{ID: uuid.New(), Name: "PF1", Position: "PF", SalaryDK: 5000, ProjectedPoints: 30.0, Team: "MIA"},
		{ID: uuid.New(), Name: "C1", Position: "C", SalaryDK: 5000, ProjectedPoints: 30.0, Team: "DEN"},
	}

	config := OptimizeConfig{
		SalaryCap:           50000,
		NumLineups:          1,
		MinDifferentPlayers: 0,
		UseCorrelations:     false,
		Contest:             createTestContest(),
	}

	optimizer := NewDPOptimizer()
	result, err := optimizer.OptimizeWithDP(players, config, "draftkings")

	t.Logf("Backward compatibility test:")
	t.Logf("- Error: %v", err)
	if result != nil {
		t.Logf("- Optimal score: %.2f", result.OptimalScore)
		t.Logf("- Players in lineup: %d", len(result.OptimalPlayers))
		t.Logf("- States explored: %d", result.StatesExplored)
		t.Logf("- Cache hit rate: %.2f", result.CacheHitRate)
	} else {
		t.Logf("- Result: nil")
	}
}