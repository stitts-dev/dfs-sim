package optimizer

import (
	"fmt"
	"log"
	"sort"

	"github.com/stitts-dev/dfs-sim/shared/types"
)

// PositionSlot represents a position slot in a lineup
type PositionSlot struct {
	SlotName         string   // e.g., "PG", "G", "UTIL"
	AllowedPositions []string // e.g., ["PG"] or ["PG", "SG"]
	Priority         int      // Fill order (1 = first)
	IsRequired       bool     // Must be filled
}

// SlotAssignment represents a player assigned to a specific slot
type SlotAssignment struct {
	PlayerID uint
	SlotName string
}

// GetPositionSlots returns the position slots for a given sport and platform
func GetPositionSlots(sport, platform string) []PositionSlot {
	log.Printf("GetPositionSlots: sport=%s, platform=%s", sport, platform)

	slots := []PositionSlot{}
	switch sport {
	case "nba":
		slots = getNBASlots(platform)
	case "nfl":
		slots = getNFLSlots(platform)
	case "mlb":
		slots = getMLBSlots(platform)
	case "nhl":
		slots = getNHLSlots(platform)
	case "golf":
		slots = getGolfSlots(platform)
	default:
		log.Printf("WARNING: Unknown sport '%s' for slot resolution", sport)
	}

	log.Printf("GetPositionSlots: Returning %d slots for %s/%s", len(slots), sport, platform)
	return slots
}

func getNBASlots(platform string) []PositionSlot {
	if platform == "draftkings" {
		return []PositionSlot{
			// Concrete positions first (priority 1-5)
			{SlotName: "PG", AllowedPositions: []string{"PG"}, Priority: 1, IsRequired: true},
			{SlotName: "SG", AllowedPositions: []string{"SG"}, Priority: 2, IsRequired: true},
			{SlotName: "SF", AllowedPositions: []string{"SF"}, Priority: 3, IsRequired: true},
			{SlotName: "PF", AllowedPositions: []string{"PF"}, Priority: 4, IsRequired: true},
			{SlotName: "C", AllowedPositions: []string{"C"}, Priority: 5, IsRequired: true},
			// Flex positions last (priority 6-8)
			{SlotName: "G", AllowedPositions: []string{"PG", "SG"}, Priority: 6, IsRequired: true},
			{SlotName: "F", AllowedPositions: []string{"SF", "PF"}, Priority: 7, IsRequired: true},
			{SlotName: "UTIL", AllowedPositions: []string{"PG", "SG", "SF", "PF", "C"}, Priority: 8, IsRequired: true},
		}
	} else if platform == "fanduel" {
		return []PositionSlot{
			{SlotName: "PG", AllowedPositions: []string{"PG"}, Priority: 1, IsRequired: true},
			{SlotName: "PG", AllowedPositions: []string{"PG"}, Priority: 2, IsRequired: true},
			{SlotName: "SG", AllowedPositions: []string{"SG"}, Priority: 3, IsRequired: true},
			{SlotName: "SG", AllowedPositions: []string{"SG"}, Priority: 4, IsRequired: true},
			{SlotName: "SF", AllowedPositions: []string{"SF"}, Priority: 5, IsRequired: true},
			{SlotName: "SF", AllowedPositions: []string{"SF"}, Priority: 6, IsRequired: true},
			{SlotName: "PF", AllowedPositions: []string{"PF"}, Priority: 7, IsRequired: true},
			{SlotName: "PF", AllowedPositions: []string{"PF"}, Priority: 8, IsRequired: true},
			{SlotName: "C", AllowedPositions: []string{"C"}, Priority: 9, IsRequired: true},
		}
	}
	return []PositionSlot{}
}

func getNFLSlots(platform string) []PositionSlot {
	if platform == "draftkings" {
		return []PositionSlot{
			{SlotName: "QB", AllowedPositions: []string{"QB"}, Priority: 1, IsRequired: true},
			{SlotName: "RB", AllowedPositions: []string{"RB"}, Priority: 2, IsRequired: true},
			{SlotName: "RB", AllowedPositions: []string{"RB"}, Priority: 3, IsRequired: true},
			{SlotName: "WR", AllowedPositions: []string{"WR"}, Priority: 4, IsRequired: true},
			{SlotName: "WR", AllowedPositions: []string{"WR"}, Priority: 5, IsRequired: true},
			{SlotName: "WR", AllowedPositions: []string{"WR"}, Priority: 6, IsRequired: true},
			{SlotName: "TE", AllowedPositions: []string{"TE"}, Priority: 7, IsRequired: true},
			{SlotName: "FLEX", AllowedPositions: []string{"RB", "WR", "TE"}, Priority: 8, IsRequired: true},
			{SlotName: "DST", AllowedPositions: []string{"DST"}, Priority: 9, IsRequired: true},
		}
	} else if platform == "fanduel" {
		return []PositionSlot{
			{SlotName: "QB", AllowedPositions: []string{"QB"}, Priority: 1, IsRequired: true},
			{SlotName: "RB", AllowedPositions: []string{"RB"}, Priority: 2, IsRequired: true},
			{SlotName: "RB", AllowedPositions: []string{"RB"}, Priority: 3, IsRequired: true},
			{SlotName: "WR", AllowedPositions: []string{"WR"}, Priority: 4, IsRequired: true},
			{SlotName: "WR", AllowedPositions: []string{"WR"}, Priority: 5, IsRequired: true},
			{SlotName: "WR", AllowedPositions: []string{"WR"}, Priority: 6, IsRequired: true},
			{SlotName: "TE", AllowedPositions: []string{"TE"}, Priority: 7, IsRequired: true},
			{SlotName: "FLEX", AllowedPositions: []string{"RB", "WR", "TE"}, Priority: 8, IsRequired: true},
			{SlotName: "D/ST", AllowedPositions: []string{"D/ST"}, Priority: 9, IsRequired: true},
		}
	}
	return []PositionSlot{}
}

func getMLBSlots(platform string) []PositionSlot {
	if platform == "draftkings" {
		return []PositionSlot{
			{SlotName: "P", AllowedPositions: []string{"P", "SP", "RP"}, Priority: 1, IsRequired: true},
			{SlotName: "P", AllowedPositions: []string{"P", "SP", "RP"}, Priority: 2, IsRequired: true},
			{SlotName: "C", AllowedPositions: []string{"C"}, Priority: 3, IsRequired: true},
			{SlotName: "1B", AllowedPositions: []string{"1B"}, Priority: 4, IsRequired: true},
			{SlotName: "2B", AllowedPositions: []string{"2B"}, Priority: 5, IsRequired: true},
			{SlotName: "3B", AllowedPositions: []string{"3B"}, Priority: 6, IsRequired: true},
			{SlotName: "SS", AllowedPositions: []string{"SS"}, Priority: 7, IsRequired: true},
			{SlotName: "OF", AllowedPositions: []string{"OF", "LF", "CF", "RF"}, Priority: 8, IsRequired: true},
			{SlotName: "OF", AllowedPositions: []string{"OF", "LF", "CF", "RF"}, Priority: 9, IsRequired: true},
			{SlotName: "OF", AllowedPositions: []string{"OF", "LF", "CF", "RF"}, Priority: 10, IsRequired: true},
		}
	} else if platform == "fanduel" {
		return []PositionSlot{
			{SlotName: "P", AllowedPositions: []string{"P", "SP", "RP"}, Priority: 1, IsRequired: true},
			{SlotName: "C/1B", AllowedPositions: []string{"C", "1B"}, Priority: 2, IsRequired: true},
			{SlotName: "2B", AllowedPositions: []string{"2B"}, Priority: 3, IsRequired: true},
			{SlotName: "3B", AllowedPositions: []string{"3B"}, Priority: 4, IsRequired: true},
			{SlotName: "SS", AllowedPositions: []string{"SS"}, Priority: 5, IsRequired: true},
			{SlotName: "OF", AllowedPositions: []string{"OF", "LF", "CF", "RF"}, Priority: 6, IsRequired: true},
			{SlotName: "OF", AllowedPositions: []string{"OF", "LF", "CF", "RF"}, Priority: 7, IsRequired: true},
			{SlotName: "OF", AllowedPositions: []string{"OF", "LF", "CF", "RF"}, Priority: 8, IsRequired: true},
			{SlotName: "UTIL", AllowedPositions: []string{"C", "1B", "2B", "3B", "SS", "OF", "LF", "CF", "RF"}, Priority: 9, IsRequired: true},
		}
	}
	return []PositionSlot{}
}

func getNHLSlots(platform string) []PositionSlot {
	if platform == "draftkings" {
		return []PositionSlot{
			{SlotName: "C", AllowedPositions: []string{"C"}, Priority: 1, IsRequired: true},
			{SlotName: "C", AllowedPositions: []string{"C"}, Priority: 2, IsRequired: true},
			{SlotName: "W", AllowedPositions: []string{"W", "LW", "RW"}, Priority: 3, IsRequired: true},
			{SlotName: "W", AllowedPositions: []string{"W", "LW", "RW"}, Priority: 4, IsRequired: true},
			{SlotName: "W", AllowedPositions: []string{"W", "LW", "RW"}, Priority: 5, IsRequired: true},
			{SlotName: "D", AllowedPositions: []string{"D"}, Priority: 6, IsRequired: true},
			{SlotName: "D", AllowedPositions: []string{"D"}, Priority: 7, IsRequired: true},
			{SlotName: "G", AllowedPositions: []string{"G"}, Priority: 8, IsRequired: true},
			{SlotName: "UTIL", AllowedPositions: []string{"C", "W", "LW", "RW", "D"}, Priority: 9, IsRequired: true},
		}
	} else if platform == "fanduel" {
		return []PositionSlot{
			{SlotName: "C", AllowedPositions: []string{"C"}, Priority: 1, IsRequired: true},
			{SlotName: "C", AllowedPositions: []string{"C"}, Priority: 2, IsRequired: true},
			{SlotName: "W", AllowedPositions: []string{"W", "LW", "RW"}, Priority: 3, IsRequired: true},
			{SlotName: "W", AllowedPositions: []string{"W", "LW", "RW"}, Priority: 4, IsRequired: true},
			{SlotName: "W", AllowedPositions: []string{"W", "LW", "RW"}, Priority: 5, IsRequired: true},
			{SlotName: "W", AllowedPositions: []string{"W", "LW", "RW"}, Priority: 6, IsRequired: true},
			{SlotName: "D", AllowedPositions: []string{"D"}, Priority: 7, IsRequired: true},
			{SlotName: "D", AllowedPositions: []string{"D"}, Priority: 8, IsRequired: true},
			{SlotName: "G", AllowedPositions: []string{"G"}, Priority: 9, IsRequired: true},
		}
	}
	return []PositionSlot{}
}

func getGolfSlots(platform string) []PositionSlot {
	// Golf is simple - just 6 golfers
	slots := make([]PositionSlot, 6)
	for i := 0; i < 6; i++ {
		slots[i] = PositionSlot{
			SlotName:         "G",
			AllowedPositions: []string{"G"},
			Priority:         i + 1,
			IsRequired:       true,
		}
	}
	return slots
}

// CanPlayerFillSlot checks if a player can fill a specific slot
func CanPlayerFillSlot(player types.Player, slot PositionSlot) bool {
	for _, allowedPos := range slot.AllowedPositions {
		if player.Position == allowedPos {
			return true
		}
	}
	return false
}

// AssignPlayersToSlots assigns players to lineup slots based on position compatibility
func AssignPlayersToSlots(players []types.Player, slots []PositionSlot) ([]SlotAssignment, error) {
	if len(players) < len(slots) {
		return nil, fmt.Errorf("not enough players (%d) to fill all slots (%d)", len(players), len(slots))
	}

	// Sort slots by priority
	sortedSlots := make([]PositionSlot, len(slots))
	copy(sortedSlots, slots)
	sort.Slice(sortedSlots, func(i, j int) bool {
		return sortedSlots[i].Priority < sortedSlots[j].Priority
	})

	// Track which players are already assigned
	assignedPlayers := make(map[uint]bool)
	assignments := make([]SlotAssignment, 0, len(slots))

	// Try to fill each slot in priority order
	for _, slot := range sortedSlots {
		assigned := false

		// Try to find an unassigned player that can fill this slot
		for _, player := range players {
			if assignedPlayers[player.ID] {
				continue
			}

			if CanPlayerFillSlot(player, slot) {
				assignments = append(assignments, SlotAssignment{
					PlayerID: player.ID,
					SlotName: slot.SlotName,
				})
				assignedPlayers[player.ID] = true
				assigned = true
				break
			}
		}

		if !assigned && slot.IsRequired {
			return nil, fmt.Errorf("cannot fill position %s - no eligible players available", slot.SlotName)
		}
	}

	return assignments, nil
}

// GetSlotAssignmentsMap converts slice of assignments to a map for easy lookup
func GetSlotAssignmentsMap(assignments []SlotAssignment) map[uint]string {
	result := make(map[uint]string)
	for _, assignment := range assignments {
		result[assignment.PlayerID] = assignment.SlotName
	}
	return result
}
