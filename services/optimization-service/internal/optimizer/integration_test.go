package optimizer_test

import (
	"testing"

	"github.com/stitts-dev/dfs-sim/shared/types"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/optimizer"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_NBA_PositionSlotAssignment(t *testing.T) {
	// Create NBA players - need enough for all positions including flex
	players := []types.Player{
		// Point Guards
		{ID: 1, Name: "Curry", Position: "PG", Salary: 8500, ProjectedPoints: 50.5, Team: "GSW"},
		{ID: 2, Name: "Morant", Position: "PG", Salary: 7000, ProjectedPoints: 42.0, Team: "MEM"},
		{ID: 3, Name: "Paul", Position: "PG", Salary: 5500, ProjectedPoints: 35.0, Team: "PHX"},
		// Shooting Guards
		{ID: 4, Name: "Harden", Position: "SG", Salary: 8000, ProjectedPoints: 48.0, Team: "PHI"},
		{ID: 5, Name: "Booker", Position: "SG", Salary: 6500, ProjectedPoints: 40.0, Team: "PHX"},
		{ID: 6, Name: "Beal", Position: "SG", Salary: 5000, ProjectedPoints: 38.0, Team: "WAS"},
		// Small Forwards
		{ID: 7, Name: "LeBron", Position: "SF", Salary: 9000, ProjectedPoints: 52.0, Team: "LAL"},
		{ID: 8, Name: "Butler", Position: "SF", Salary: 6500, ProjectedPoints: 41.0, Team: "MIA"},
		{ID: 9, Name: "Tatum", Position: "SF", Salary: 5500, ProjectedPoints: 45.0, Team: "BOS"},
		// Power Forwards
		{ID: 10, Name: "Davis", Position: "PF", Salary: 8500, ProjectedPoints: 51.0, Team: "LAL"},
		{ID: 11, Name: "Siakam", Position: "PF", Salary: 6000, ProjectedPoints: 38.0, Team: "TOR"},
		{ID: 12, Name: "Collins", Position: "PF", Salary: 4500, ProjectedPoints: 33.0, Team: "ATL"},
		// Centers
		{ID: 13, Name: "Jokic", Position: "C", Salary: 9500, ProjectedPoints: 55.0, Team: "DEN"},
		{ID: 14, Name: "Embiid", Position: "C", Salary: 8000, ProjectedPoints: 53.0, Team: "PHI"},
		{ID: 15, Name: "Towns", Position: "C", Salary: 5500, ProjectedPoints: 43.0, Team: "MIN"},
	}

	contest := &types.Contest{
		ID:        1,
		Sport:     "nba",
		Platform:  "draftkings",
		SalaryCap: 50000,
		PositionRequirements: types.PositionRequirements{
			"PG":   1,
			"SG":   1,
			"SF":   1,
			"PF":   1,
			"C":    1,
			"G":    1,
			"F":    1,
			"UTIL": 1,
		},
	}

	config := optimizer.OptimizeConfig{
		SalaryCap:  contest.SalaryCap,
		NumLineups: 1,
		Contest:    contest,
	}

	result, err := optimizer.OptimizeLineups(players, config)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Lineups, 1)

	lineup := result.Lineups[0]

	// Verify all position slots are filled correctly
	usedSlots := make(map[string]bool)
	for playerID, slot := range lineup.PlayerPositions {
		assert.False(t, usedSlots[slot], "Slot %s should not be used twice", slot)
		usedSlots[slot] = true

		// Find the player
		var player *types.Player
		for _, p := range lineup.Players {
			if p.ID == playerID {
				player = &p
				break
			}
		}
		assert.NotNil(t, player)

		// Verify player can fill the slot
		slots := optimizer.GetPositionSlots("nba", "draftkings")
		validSlot := false
		for _, s := range slots {
			if s.SlotName == slot {
				for _, allowed := range s.AllowedPositions {
					if player.Position == allowed {
						validSlot = true
						break
					}
				}
				break
			}
		}
		assert.True(t, validSlot, "Player %s (%s) should be able to fill slot %s", player.Name, player.Position, slot)
	}

	// Verify all required slots are filled
	assert.Len(t, usedSlots, 8)
	requiredSlots := []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UTIL"}
	for _, slot := range requiredSlots {
		assert.True(t, usedSlots[slot], "Required slot %s should be filled", slot)
	}
}

func TestIntegration_Golf_SimplePositions(t *testing.T) {
	// Create Golf players
	players := []types.Player{
		{ID: 1, Name: "McIlroy", Position: "G", Salary: 9500, ProjectedPoints: 65.0, Team: "NIR"},
		{ID: 2, Name: "Scheffler", Position: "G", Salary: 10000, ProjectedPoints: 68.0, Team: "USA"},
		{ID: 3, Name: "Rahm", Position: "G", Salary: 9200, ProjectedPoints: 64.0, Team: "ESP"},
		{ID: 4, Name: "Cantlay", Position: "G", Salary: 8500, ProjectedPoints: 58.0, Team: "USA"},
		{ID: 5, Name: "Hovland", Position: "G", Salary: 8200, ProjectedPoints: 61.0, Team: "NOR"},
		{ID: 6, Name: "Schauffele", Position: "G", Salary: 7800, ProjectedPoints: 59.0, Team: "USA"},
		{ID: 7, Name: "Spieth", Position: "G", Salary: 7500, ProjectedPoints: 55.0, Team: "USA"},
		{ID: 8, Name: "Finau", Position: "G", Salary: 7000, ProjectedPoints: 53.0, Team: "USA"},
	}

	contest := &types.Contest{
		ID:        2,
		Sport:     "golf",
		Platform:  "draftkings",
		SalaryCap: 50000,
		PositionRequirements: types.PositionRequirements{
			"G": 6,
		},
	}

	config := optimizer.OptimizeConfig{
		SalaryCap:  contest.SalaryCap,
		NumLineups: 1,
		Contest:    contest,
	}

	result, err := optimizer.OptimizeLineups(players, config)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Lineups, 1)

	lineup := result.Lineups[0]
	assert.Len(t, lineup.Players, 6)
	assert.Len(t, lineup.PlayerPositions, 6)

	// All positions should be "G"
	for _, position := range lineup.PlayerPositions {
		assert.Equal(t, "G", position)
	}
}
