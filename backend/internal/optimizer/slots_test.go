package optimizer

import (
	"testing"

	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestGetPositionSlots_NBA_DraftKings(t *testing.T) {
	slots := GetPositionSlots("nba", "draftkings")

	assert.Len(t, slots, 8, "NBA DraftKings should have 8 position slots")

	// Verify concrete positions
	assert.Equal(t, "PG", slots[0].SlotName)
	assert.Equal(t, []string{"PG"}, slots[0].AllowedPositions)
	assert.Equal(t, 1, slots[0].Priority)
	assert.True(t, slots[0].IsRequired)

	assert.Equal(t, "SG", slots[1].SlotName)
	assert.Equal(t, []string{"SG"}, slots[1].AllowedPositions)

	assert.Equal(t, "SF", slots[2].SlotName)
	assert.Equal(t, []string{"SF"}, slots[2].AllowedPositions)

	assert.Equal(t, "PF", slots[3].SlotName)
	assert.Equal(t, []string{"PF"}, slots[3].AllowedPositions)

	assert.Equal(t, "C", slots[4].SlotName)
	assert.Equal(t, []string{"C"}, slots[4].AllowedPositions)

	// Verify flex positions
	assert.Equal(t, "G", slots[5].SlotName)
	assert.Equal(t, []string{"PG", "SG"}, slots[5].AllowedPositions)
	assert.Equal(t, 6, slots[5].Priority)

	assert.Equal(t, "F", slots[6].SlotName)
	assert.Equal(t, []string{"SF", "PF"}, slots[6].AllowedPositions)
	assert.Equal(t, 7, slots[6].Priority)

	assert.Equal(t, "UTIL", slots[7].SlotName)
	assert.Equal(t, []string{"PG", "SG", "SF", "PF", "C"}, slots[7].AllowedPositions)
	assert.Equal(t, 8, slots[7].Priority)
}

func TestGetPositionSlots_Golf(t *testing.T) {
	slots := GetPositionSlots("golf", "draftkings")

	assert.Len(t, slots, 6, "Golf should have 6 position slots")

	// All slots should be "G" for golfer
	for i, slot := range slots {
		assert.Equal(t, "G", slot.SlotName)
		assert.Equal(t, []string{"G"}, slot.AllowedPositions)
		assert.Equal(t, i+1, slot.Priority)
		assert.True(t, slot.IsRequired)
	}
}

func TestCanPlayerFillSlot(t *testing.T) {
	tests := []struct {
		name     string
		player   models.Player
		slot     PositionSlot
		expected bool
	}{
		{
			name:     "PG can fill PG slot",
			player:   models.Player{Position: "PG"},
			slot:     PositionSlot{SlotName: "PG", AllowedPositions: []string{"PG"}},
			expected: true,
		},
		{
			name:     "PG can fill G flex slot",
			player:   models.Player{Position: "PG"},
			slot:     PositionSlot{SlotName: "G", AllowedPositions: []string{"PG", "SG"}},
			expected: true,
		},
		{
			name:     "PG can fill UTIL slot",
			player:   models.Player{Position: "PG"},
			slot:     PositionSlot{SlotName: "UTIL", AllowedPositions: []string{"PG", "SG", "SF", "PF", "C"}},
			expected: true,
		},
		{
			name:     "PG cannot fill F slot",
			player:   models.Player{Position: "PG"},
			slot:     PositionSlot{SlotName: "F", AllowedPositions: []string{"SF", "PF"}},
			expected: false,
		},
		{
			name:     "C cannot fill G slot",
			player:   models.Player{Position: "C"},
			slot:     PositionSlot{SlotName: "G", AllowedPositions: []string{"PG", "SG"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanPlayerFillSlot(tt.player, tt.slot)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAssignPlayersToSlots(t *testing.T) {
	// Test NBA lineup with flex positions
	players := []models.Player{
		{ID: 1, Name: "Curry", Position: "PG"},
		{ID: 2, Name: "Harden", Position: "SG"},
		{ID: 3, Name: "LeBron", Position: "SF"},
		{ID: 4, Name: "Davis", Position: "PF"},
		{ID: 5, Name: "Jokic", Position: "C"},
		{ID: 6, Name: "Morant", Position: "PG"}, // For G slot
		{ID: 7, Name: "Butler", Position: "SF"}, // For F slot
		{ID: 8, Name: "Embiid", Position: "C"},  // For UTIL slot
	}

	slots := GetPositionSlots("nba", "draftkings")

	assignments, err := AssignPlayersToSlots(players, slots)
	assert.NoError(t, err)
	assert.Len(t, assignments, 8)

	// Verify concrete positions filled first
	expectedAssignments := map[uint]string{
		1: "PG",   // Curry
		2: "SG",   // Harden
		3: "SF",   // LeBron
		4: "PF",   // Davis
		5: "C",    // Jokic
		6: "G",    // Morant (PG in G flex)
		7: "F",    // Butler (SF in F flex)
		8: "UTIL", // Embiid (C in UTIL)
	}

	for _, assignment := range assignments {
		expected, exists := expectedAssignments[assignment.PlayerID]
		assert.True(t, exists, "Player %d should have assignment", assignment.PlayerID)
		assert.Equal(t, expected, assignment.SlotName, "Player %d should be in slot %s", assignment.PlayerID, expected)
	}
}

func TestAssignPlayersToSlots_InsufficientPlayers(t *testing.T) {
	// Not enough players to fill all slots
	players := []models.Player{
		{ID: 1, Name: "Curry", Position: "PG"},
		{ID: 2, Name: "Harden", Position: "SG"},
		{ID: 3, Name: "LeBron", Position: "SF"},
	}

	slots := GetPositionSlots("nba", "draftkings")

	_, err := AssignPlayersToSlots(players, slots)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enough players")
}

func TestAssignPlayersToSlots_NoValidAssignment(t *testing.T) {
	// Players that can't fill required positions
	players := []models.Player{
		{ID: 1, Name: "Player1", Position: "PG"},
		{ID: 2, Name: "Player2", Position: "PG"},
		{ID: 3, Name: "Player3", Position: "PG"},
		{ID: 4, Name: "Player4", Position: "PG"},
		{ID: 5, Name: "Player5", Position: "PG"},
		{ID: 6, Name: "Player6", Position: "PG"},
		{ID: 7, Name: "Player7", Position: "PG"},
		{ID: 8, Name: "Player8", Position: "PG"},
	}

	slots := GetPositionSlots("nba", "draftkings")

	_, err := AssignPlayersToSlots(players, slots)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot fill position")
}

func TestFlexPositionPriority(t *testing.T) {
	// Test that concrete positions are filled before flex
	players := []models.Player{
		{ID: 1, Name: "Curry", Position: "PG"},
		{ID: 2, Name: "Morant", Position: "PG"}, // Should go to G flex
		{ID: 3, Name: "Harden", Position: "SG"},
		{ID: 4, Name: "LeBron", Position: "SF"},
		{ID: 5, Name: "Davis", Position: "PF"},
		{ID: 6, Name: "Jokic", Position: "C"},
		{ID: 7, Name: "Butler", Position: "SF"}, // Should go to F flex
		{ID: 8, Name: "Tatum", Position: "SF"},  // Should go to UTIL
	}

	slots := GetPositionSlots("nba", "draftkings")

	assignments, err := AssignPlayersToSlots(players, slots)
	assert.NoError(t, err)

	// Create assignment map for easier checking
	assignmentMap := make(map[uint]string)
	for _, a := range assignments {
		assignmentMap[a.PlayerID] = a.SlotName
	}

	// Verify concrete positions filled with first available players
	assert.Equal(t, "PG", assignmentMap[1])   // Curry gets PG
	assert.Equal(t, "G", assignmentMap[2])    // Morant gets G flex
	assert.Equal(t, "SG", assignmentMap[3])   // Harden gets SG
	assert.Equal(t, "SF", assignmentMap[4])   // LeBron gets SF
	assert.Equal(t, "PF", assignmentMap[5])   // Davis gets PF
	assert.Equal(t, "C", assignmentMap[6])    // Jokic gets C
	assert.Equal(t, "F", assignmentMap[7])    // Butler gets F flex
	assert.Equal(t, "UTIL", assignmentMap[8]) // Tatum gets UTIL
}

func TestGetPositionSlots_AllSports(t *testing.T) {
	sports := []struct {
		sport    string
		platform string
		expected int
	}{
		{"nba", "draftkings", 8},
		{"nba", "fanduel", 9},
		{"nfl", "draftkings", 9},
		{"nfl", "fanduel", 9},
		{"mlb", "draftkings", 10},
		{"mlb", "fanduel", 9},
		{"nhl", "draftkings", 9},
		{"nhl", "fanduel", 9},
		{"golf", "draftkings", 6},
		{"golf", "fanduel", 6},
	}

	for _, s := range sports {
		t.Run(s.sport+"_"+s.platform, func(t *testing.T) {
			slots := GetPositionSlots(s.sport, s.platform)
			assert.Len(t, slots, s.expected, "%s %s should have %d slots", s.sport, s.platform, s.expected)

			// Verify all slots have required fields
			for i, slot := range slots {
				assert.NotEmpty(t, slot.SlotName, "Slot %d should have name", i)
				assert.NotEmpty(t, slot.AllowedPositions, "Slot %d should have allowed positions", i)
				assert.Greater(t, slot.Priority, 0, "Slot %d should have positive priority", i)
			}
		})
	}
}
