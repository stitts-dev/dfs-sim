package optimizer

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/stitts-dev/dfs-sim/shared/types"
	"github.com/stitts-dev/dfs-sim/shared/pkg/logger"
	"github.com/sirupsen/logrus"
)

type OptimizeConfig struct {
	SalaryCap           int              `json:"salary_cap"`
	NumLineups          int              `json:"num_lineups"`
	MinDifferentPlayers int              `json:"min_different_players"`
	UseCorrelations     bool             `json:"use_correlations"`
	CorrelationWeight   float64          `json:"correlation_weight"`
	StackingRules       []StackingRule   `json:"stacking_rules"`
	LockedPlayers       []uint           `json:"locked_players"`
	ExcludedPlayers     []uint           `json:"excluded_players"`
	MinExposure         map[uint]float64 `json:"min_exposure"`
	MaxExposure         map[uint]float64 `json:"max_exposure"`
	Contest             *types.Contest  `json:"-"`
}

type StackingRule struct {
	Type       string   `json:"type"` // "team", "game", "mini"
	MinPlayers int      `json:"min_players"`
	MaxPlayers int      `json:"max_players"`
	Teams      []string `json:"teams,omitempty"`
}

type OptimizerResult struct {
	Lineups           []types.GeneratedLineup `json:"lineups"`
	OptimizationTime  int64           `json:"optimization_time_ms"`
	TotalCombinations int64           `json:"total_combinations"`
	ValidCombinations int64           `json:"valid_combinations"`
}

type lineupCandidate struct {
	players         []types.Player
	totalSalary     int
	projectedPoints float64
	positions       map[string][]types.Player
	// Track which position slot each player fills (for flex positions)
	playerPositions map[uint]string // playerID -> position slot filled
}

func OptimizeLineups(players []types.Player, config OptimizeConfig) (*OptimizerResult, error) {
	// Generate unique optimization ID for request tracing
	optimizationID := uuid.New().String()
	startTime := getCurrentTimeMs()
	result := &OptimizerResult{}

	// Initialize logger with optimization context
	logger := logger.WithOptimizationContext(optimizationID, config.Contest.Sport, config.Contest.Platform)
	logger.WithFields(logrus.Fields{
		"total_players": len(players),
		"salary_cap":    config.SalaryCap,
		"num_lineups":   config.NumLineups,
	}).Info("Starting optimization")

	// Filter out excluded players
	filteredPlayers := filterPlayers(players, config, logger)

	if len(filteredPlayers) == 0 {
		return nil, fmt.Errorf("no players available after filtering")
	}

	// Organize players by position
	playersByPosition := organizeByPosition(filteredPlayers, logger)

	// Generate all valid lineup combinations
	validLineups := generateValidLineups(playersByPosition, config, logger)

	logger.WithFields(logrus.Fields{
		"valid_lineups": len(validLineups),
	}).Debug("Lineup generation completed")

	if len(validLineups) == 0 {
		// Provide detailed error message
		minSalary := int(float64(config.SalaryCap) * 0.95)
		return nil, fmt.Errorf("no valid lineups could be generated - check if there are enough players at each position with salaries that fit within the cap ($%d-$%d)", minSalary, config.SalaryCap)
	}

	// Sort by projected points (with correlation bonus if enabled)
	sort.Slice(validLineups, func(i, j int) bool {
		return validLineups[i].projectedPoints > validLineups[j].projectedPoints
	})

	// Apply diversity constraints and exposure limits
	finalLineups := applyDiversityConstraints(validLineups, config)

	// Convert to model lineups
	result.Lineups = make([]types.GeneratedLineup, 0, len(finalLineups))
	for i, candidate := range finalLineups {
		// Convert Player slice to LineupPlayer slice
		lineupPlayers := make([]types.LineupPlayer, len(candidate.players))
		for j, player := range candidate.players {
			lineupPlayers[j] = types.LineupPlayer{
				ID:              player.ID,
				Name:            player.Name,
				Team:            player.Team,
				Position:        player.Position,
				Salary:          player.Salary,
				ProjectedPoints: player.ProjectedPoints,
			}
		}

		lineup := types.GeneratedLineup{
			ID:               fmt.Sprintf("lineup_%d_%s", i+1, uuid.New().String()[:8]),
			Players:          lineupPlayers,
			TotalSalary:      candidate.totalSalary,
			ProjectedPoints:  candidate.projectedPoints,
			Exposure:         0.0, // Will be calculated later
			StackDescription: "", // TODO: Add stack description logic
		}
		result.Lineups = append(result.Lineups, lineup)
	}

	result.OptimizationTime = getCurrentTimeMs() - startTime
	result.ValidCombinations = int64(len(validLineups))

	return result, nil
}

func filterPlayers(players []types.Player, config OptimizeConfig, logger *logrus.Entry) []types.Player {
	excludeMap := make(map[uint]bool)
	for _, id := range config.ExcludedPlayers {
		excludeMap[id] = true
	}

	filtered := make([]types.Player, 0, len(players))
	excludedCount := 0
	injuredCount := 0

	for _, player := range players {
		if excludeMap[player.ID] {
			excludedCount++
		} else if player.IsInjured {
			injuredCount++
		} else {
			filtered = append(filtered, player)
		}
	}

	logger.WithFields(logrus.Fields{
		"total_players":   len(players),
		"excluded_count":  excludedCount,
		"injured_count":   injuredCount,
		"available_count": len(filtered),
	}).Debug("Player filtering complete")

	return filtered
}

func organizeByPosition(players []types.Player, logger *logrus.Entry) map[string][]types.Player {
	byPosition := make(map[string][]types.Player)
	for _, player := range players {
		byPosition[player.Position] = append(byPosition[player.Position], player)
	}

	// Sort each position by value (projected points per dollar)
	for position := range byPosition {
		sort.Slice(byPosition[position], func(i, j int) bool {
			valueI := byPosition[position][i].ProjectedPoints / float64(byPosition[position][i].Salary)
			valueJ := byPosition[position][j].ProjectedPoints / float64(byPosition[position][j].Salary)
			return valueI > valueJ
		})
	}

	// Log player organization stats
	positionCounts := make(map[string]int)
	topPlayers := make(map[string][]string)
	for pos, players := range byPosition {
		positionCounts[pos] = len(players)
		// Get top 3 player names for debug logging
		topPlayerNames := make([]string, 0, 3)
		for i, p := range players {
			if i < 3 {
				topPlayerNames = append(topPlayerNames, fmt.Sprintf("%s($%d)", p.Name, p.Salary))
			}
		}
		topPlayers[pos] = topPlayerNames
	}
	logger.WithFields(logrus.Fields{
		"position_counts": positionCounts,
		"top_players":     topPlayers,
	}).Debug("Players organized by position")

	return byPosition
}

func generateValidLineups(playersByPosition map[string][]types.Player, config OptimizeConfig, logger *logrus.Entry) []lineupCandidate {
	var validLineups []lineupCandidate

	// Early validation
	if config.Contest == nil {
		logger.Error("No contest provided to optimizer")
		return []lineupCandidate{}
	}

	// Get position slots for this sport/platform
	slots := GetPositionSlots(config.Contest.Sport, config.Contest.Platform)
	if len(slots) == 0 {
		logger.WithFields(logrus.Fields{
			"sport":    config.Contest.Sport,
			"platform": config.Contest.Platform,
		}).Error("No position slots found for contest")
		return []lineupCandidate{}
	}

	// Log position slots for debugging
	slotInfo := make([]map[string]interface{}, len(slots))
	for i, slot := range slots {
		slotInfo[i] = map[string]interface{}{
			"index":             i,
			"slot_name":         slot.SlotName,
			"allowed_positions": slot.AllowedPositions,
		}
	}
	logger.WithFields(logrus.Fields{
		"total_slots":  len(slots),
		"slot_details": slotInfo,
	}).Debug("Position slots loaded")

	// Validate player availability for each slot
	for i, slot := range slots {
		hasPlayers := false
		for _, pos := range slot.AllowedPositions {
			if len(playersByPosition[pos]) > 0 {
				hasPlayers = true
				break
			}
		}
		if !hasPlayers {
			logger.WithFields(logrus.Fields{
				"slot_index":        i,
				"slot_name":         slot.SlotName,
				"allowed_positions": slot.AllowedPositions,
			}).Warn("No players available for slot")
		}
	}

	// Limit generation to avoid memory issues
	maxLineups := config.NumLineups * 100
	if maxLineups > 10000 {
		maxLineups = 10000
	}

	// Use recursive backtracking to generate lineups
	var backtrack func(current *lineupCandidate, slotIndex int, usedPlayers map[uint]bool)

	backtrack = func(current *lineupCandidate, slotIndex int, usedPlayers map[uint]bool) {
		// Log backtrack start for first slot only
		if slotIndex == 0 {
			logger.WithFields(logrus.Fields{
				"initial_players": len(current.players),
				"initial_salary":  current.totalSalary,
				"max_lineups":     maxLineups,
			}).Debug("Starting backtrack algorithm")
		}

		// Skip slots that are already filled (from locked players)
		for slotIndex < len(slots) && len(current.players) > slotIndex {
			// Check if this slot is already filled
			slotFilled := false
			for _, player := range current.players {
				if pos, ok := current.playerPositions[player.ID]; ok && pos == slots[slotIndex].SlotName {
					slotFilled = true
					break
				}
			}
			if slotFilled {
				slotIndex++
			} else {
				break
			}
		}

		// Stop if we've found enough lineups
		if len(validLineups) >= maxLineups {
			return
		}

		// Base case: all slots filled
		if slotIndex >= len(slots) {
			logger.WithFields(logrus.Fields{
				"lineup_players": len(current.players),
				"total_salary":   current.totalSalary,
				"salary_cap":     config.SalaryCap,
			}).Debug("Checking lineup validity")

			// Check if all locked players are included
			if len(config.LockedPlayers) > 0 {
				hasAllLocked := true
				for _, lockedID := range config.LockedPlayers {
					found := false
					for _, player := range current.players {
						if player.ID == lockedID {
							found = true
							break
						}
					}
					if !found {
						hasAllLocked = false
						break
					}
				}
				if !hasAllLocked {
					logger.Debug("Lineup missing locked players")
					return
				}
			}

			if isValidLineup(current, config) {
				// Create a copy and add to results
				lineupCopy := *current
				lineupCopy.players = make([]types.Player, len(current.players))
				copy(lineupCopy.players, current.players)
				// Deep copy playerPositions map
				lineupCopy.playerPositions = make(map[uint]string)
				for k, v := range current.playerPositions {
					lineupCopy.playerPositions[k] = v
				}
				validLineups = append(validLineups, lineupCopy)
				logger.WithFields(logrus.Fields{
					"lineup_number":    len(validLineups),
					"projected_points": current.projectedPoints,
					"salary_used":      current.totalSalary,
				}).Debug("Found valid lineup")
			} else {
				minSalary := int(float64(config.SalaryCap) * 0.95)
				logger.WithFields(logrus.Fields{
					"salary_used": current.totalSalary,
					"salary_cap":  config.SalaryCap,
					"min_salary":  minSalary,
				}).Debug("Lineup validation failed")
			}
			return
		}

		slot := slots[slotIndex]
		logger.WithFields(logrus.Fields{
			"slot_index":        slotIndex,
			"slot_name":         slot.SlotName,
			"allowed_positions": slot.AllowedPositions,
		}).Debug("Processing slot")

		// Try each allowed position for this slot
		playersFound := false
		for _, allowedPos := range slot.AllowedPositions {
			players := playersByPosition[allowedPos]

			if len(players) == 0 {
				logger.WithFields(logrus.Fields{
					"position":   allowedPos,
					"slot_index": slotIndex,
					"slot_name":  slot.SlotName,
				}).Debug("No players available for position")
				continue
			}

			logger.WithFields(logrus.Fields{
				"position":     allowedPos,
				"player_count": len(players),
				"slot_index":   slotIndex,
			}).Debug("Players found for position")
			playersFound = true

			// Try each player that can fill this slot
			playersTried := 0
			for _, player := range players {
				// Skip if player already used
				if usedPlayers[player.ID] {
					continue
				}

				// Check salary cap
				if current.totalSalary+player.Salary > config.SalaryCap {
					continue
				}

				playersTried++

				// Add player to lineup
				current.players = append(current.players, player)
				current.totalSalary += player.Salary
				current.projectedPoints += player.ProjectedPoints
				current.playerPositions[player.ID] = slot.SlotName
				usedPlayers[player.ID] = true

				// Recurse to fill next slot
				backtrack(current, slotIndex+1, usedPlayers)

				// Backtrack
				current.players = current.players[:len(current.players)-1]
				current.totalSalary -= player.Salary
				current.projectedPoints -= player.ProjectedPoints
				delete(current.playerPositions, player.ID)
				usedPlayers[player.ID] = false

				// Limit how many players we try per position to avoid exponential explosion
				if playersTried >= 10 && len(validLineups) > 0 {
					break
				}
			}
		}

		if !playersFound {
			logger.WithFields(logrus.Fields{
				"slot_index": slotIndex,
				"slot_name":  slot.SlotName,
			}).Debug("No players found for any allowed position")
		}
	}

	// Initialize lineup
	initial := &lineupCandidate{
		players:         make([]types.Player, 0, len(slots)),
		positions:       make(map[string][]types.Player),
		playerPositions: make(map[uint]string),
	}

	usedPlayers := make(map[uint]bool)

	// For locked players, we'll track them but let the backtracking algorithm handle placement
	lockedPlayerSet := make(map[uint]bool)
	for _, lockedID := range config.LockedPlayers {
		lockedPlayerSet[lockedID] = true
	}
	if len(config.LockedPlayers) > 0 {
		logger.WithFields(logrus.Fields{
			"locked_player_ids": config.LockedPlayers,
		}).Debug("Locked players configured")
	}

	// Start backtracking
	logger.WithFields(logrus.Fields{
		"locked_players_count": len(config.LockedPlayers),
		"max_lineups":          maxLineups,
	}).Debug("Starting backtracking algorithm")
	backtrack(initial, 0, usedPlayers)

	logger.WithFields(logrus.Fields{
		"valid_lineups": len(validLineups),
		"max_lineups":   maxLineups,
	}).Info("Lineup generation completed")
	return validLineups
}

func generatePositionCombinations(current *lineupCandidate, players []types.Player, position string, required int, positionOrder []string, posIndex int, config OptimizeConfig, backtrack *func(*lineupCandidate, []string, int)) {
	// Calculate how many we already have for this position
	existing := len(current.positions[position])
	needed := required - existing

	if needed <= 0 {
		// Position already filled, move to next
		(*backtrack)(current, positionOrder, posIndex+1)
		return
	}

	// Try different combinations of players
	generateCombinations(players, needed, func(combo []types.Player) {
		// Check if any player is already in lineup
		valid := true
		for _, player := range combo {
			if hasPlayer(current.players, player.ID) {
				valid = false
				break
			}
		}

		if !valid {
			return
		}

		// Check salary cap
		additionalSalary := 0
		additionalPoints := 0.0
		for _, player := range combo {
			additionalSalary += player.Salary
			additionalPoints += player.ProjectedPoints
		}

		if current.totalSalary+additionalSalary > config.SalaryCap {
			return
		}

		// Add players and recurse
		for _, player := range combo {
			current.players = append(current.players, player)
			current.positions[position] = append(current.positions[position], player)
			current.playerPositions[player.ID] = position
		}
		current.totalSalary += additionalSalary
		current.projectedPoints += additionalPoints

		// Apply correlation bonus if enabled
		if config.UseCorrelations {
			current.projectedPoints += calculateCorrelationBonus(current.players, config.CorrelationWeight)
		}

		(*backtrack)(current, positionOrder, posIndex+1)

		// Remove players (backtrack)
		for _, player := range combo {
			delete(current.playerPositions, player.ID)
		}
		current.players = current.players[:len(current.players)-len(combo)]
		current.positions[position] = current.positions[position][:len(current.positions[position])-len(combo)]
		current.totalSalary -= additionalSalary
		current.projectedPoints -= additionalPoints + calculateCorrelationBonus(current.players, config.CorrelationWeight)
	})
}

func generateFlexCombinations(current *lineupCandidate, eligiblePlayers []types.Player, required int, positionOrder []string, posIndex int, config OptimizeConfig, backtrack *func(*lineupCandidate, []string, int)) {
	if required <= 0 {
		(*backtrack)(current, positionOrder, posIndex+1)
		return
	}

	// For flex positions, we can use players from multiple eligible positions
	generateCombinations(eligiblePlayers, required, func(combo []types.Player) {
		// Check validity and add similar to regular positions
		valid := true
		additionalSalary := 0
		additionalPoints := 0.0

		for _, player := range combo {
			if hasPlayer(current.players, player.ID) {
				valid = false
				break
			}
			additionalSalary += player.Salary
			additionalPoints += player.ProjectedPoints
		}

		if !valid || current.totalSalary+additionalSalary > config.SalaryCap {
			return
		}

		// Add players
		flexPosition := positionOrder[posIndex] // UTIL, FLEX, G, F
		for _, player := range combo {
			current.players = append(current.players, player)
			current.playerPositions[player.ID] = flexPosition
		}
		current.totalSalary += additionalSalary
		current.projectedPoints += additionalPoints

		if config.UseCorrelations {
			current.projectedPoints += calculateCorrelationBonus(current.players, config.CorrelationWeight)
		}

		(*backtrack)(current, positionOrder, posIndex+1)

		// Remove players
		for _, player := range combo {
			delete(current.playerPositions, player.ID)
		}
		current.players = current.players[:len(current.players)-len(combo)]
		current.totalSalary -= additionalSalary
		current.projectedPoints -= additionalPoints + calculateCorrelationBonus(current.players, config.CorrelationWeight)
	})
}

func getFlexEligiblePlayers(playersByPosition map[string][]types.Player, flexType string, current *lineupCandidate) []types.Player {
	var eligible []types.Player

	switch flexType {
	case "UTIL": // NBA - any position
		for _, players := range playersByPosition {
			eligible = append(eligible, players...)
		}
	case "FLEX": // NFL - RB/WR/TE
		eligible = append(eligible, playersByPosition["RB"]...)
		eligible = append(eligible, playersByPosition["WR"]...)
		eligible = append(eligible, playersByPosition["TE"]...)
	case "G": // NBA - PG or SG
		eligible = append(eligible, playersByPosition["PG"]...)
		eligible = append(eligible, playersByPosition["SG"]...)
	case "F": // NBA - SF or PF
		eligible = append(eligible, playersByPosition["SF"]...)
		eligible = append(eligible, playersByPosition["PF"]...)
	}

	// Sort by value
	sort.Slice(eligible, func(i, j int) bool {
		valueI := eligible[i].ProjectedPoints / float64(eligible[i].Salary)
		valueJ := eligible[j].ProjectedPoints / float64(eligible[j].Salary)
		return valueI > valueJ
	})

	return eligible
}

func isValidLineup(lineup *lineupCandidate, config OptimizeConfig) bool {
	// Check salary cap
	if lineup.totalSalary > config.SalaryCap {
		return false
	}

	// Minimum salary usage (95% of cap)
	minSalary := int(float64(config.SalaryCap) * 0.95)
	if lineup.totalSalary < minSalary {
		return false
	}

	// Check stacking rules
	if !validateStackingRules(lineup, config.StackingRules) {
		return false
	}

	return true
}

func validateStackingRules(lineup *lineupCandidate, rules []StackingRule) bool {
	for _, rule := range rules {
		switch rule.Type {
		case "team":
			if !validateTeamStacking(lineup, rule) {
				return false
			}
		case "game":
			if !validateGameStacking(lineup, rule) {
				return false
			}
		}
	}
	return true
}

func validateTeamStacking(lineup *lineupCandidate, rule StackingRule) bool {
	teamCounts := make(map[string]int)
	for _, player := range lineup.players {
		teamCounts[player.Team]++
	}

	// Check specific teams if provided
	if len(rule.Teams) > 0 {
		for _, team := range rule.Teams {
			count := teamCounts[team]
			if count < rule.MinPlayers || count > rule.MaxPlayers {
				return false
			}
		}
	} else {
		// Check all teams
		for _, count := range teamCounts {
			if count > rule.MaxPlayers {
				return false
			}
		}
	}

	return true
}

func validateGameStacking(lineup *lineupCandidate, rule StackingRule) bool {
	gameCounts := make(map[string]int)

	for _, player := range lineup.players {
		gameKey := getGameKey(player.Team, player.Opponent)
		gameCounts[gameKey]++
	}

	for _, count := range gameCounts {
		if count < rule.MinPlayers || count > rule.MaxPlayers {
			return false
		}
	}

	return true
}

func applyDiversityConstraints(lineups []lineupCandidate, config OptimizeConfig) []lineupCandidate {
	if len(lineups) == 0 || config.NumLineups == 0 {
		return nil
	}

	finalLineups := make([]lineupCandidate, 0, config.NumLineups)
	playerExposure := make(map[uint]int)

	for _, lineup := range lineups {
		if len(finalLineups) >= config.NumLineups {
			break
		}

		// Check diversity constraint
		if len(finalLineups) > 0 && config.MinDifferentPlayers > 0 {
			differentCount := countDifferentPlayers(lineup, finalLineups[len(finalLineups)-1])
			if differentCount < config.MinDifferentPlayers {
				continue
			}
		}

		// Check exposure limits
		valid := true
		for _, player := range lineup.players {
			currentExposure := float64(playerExposure[player.ID]) / float64(len(finalLineups)+1)

			if maxExp, exists := config.MaxExposure[player.ID]; exists && currentExposure > maxExp {
				valid = false
				break
			}
		}

		if !valid {
			continue
		}

		// Add lineup
		finalLineups = append(finalLineups, lineup)

		// Update exposure counts
		for _, player := range lineup.players {
			playerExposure[player.ID]++
		}
	}

	return finalLineups
}

// Helper functions

func hasPlayer(players []types.Player, playerID uint) bool {
	for _, p := range players {
		if p.ID == playerID {
			return true
		}
	}
	return false
}

func isPlayerLocked(playerID uint, lockedPlayers []uint) bool {
	for _, id := range lockedPlayers {
		if id == playerID {
			return true
		}
	}
	return false
}

func mapToSlice(playerMap map[uint]types.Player) []types.Player {
	players := make([]types.Player, 0, len(playerMap))
	for _, player := range playerMap {
		players = append(players, player)
	}
	return players
}

func getPositionOrder(requirements types.PositionRequirements) []string {
	// Order positions to optimize backtracking
	positions := make([]string, 0, len(requirements))
	for pos := range requirements {
		positions = append(positions, pos)
	}

	// Sort by number of required players (ascending) to fail fast
	sort.Slice(positions, func(i, j int) bool {
		// Put flex positions last
		if isFlexPosition(positions[i]) && !isFlexPosition(positions[j]) {
			return false
		}
		if !isFlexPosition(positions[i]) && isFlexPosition(positions[j]) {
			return true
		}
		return requirements[positions[i]] < requirements[positions[j]]
	})

	return positions
}

func isFlexPosition(position string) bool {
	return position == "UTIL" || position == "FLEX" || position == "G" || position == "F"
}

func getGameKey(team1, team2 string) string {
	if team1 < team2 {
		return fmt.Sprintf("%s@%s", team1, team2)
	}
	return fmt.Sprintf("%s@%s", team2, team1)
}

func countDifferentPlayers(lineup1, lineup2 lineupCandidate) int {
	playerMap := make(map[uint]bool)
	for _, p := range lineup1.players {
		playerMap[p.ID] = true
	}

	different := 0
	for _, p := range lineup2.players {
		if !playerMap[p.ID] {
			different++
		}
	}

	return different
}

func generateCombinations(players []types.Player, k int, callback func([]types.Player)) {
	if k > len(players) {
		return
	}

	// Use iterative approach with bit manipulation for better performance
	n := len(players)
	for i := 0; i < (1 << n); i++ {
		if countBits(i) == k {
			combo := make([]types.Player, 0, k)
			for j := 0; j < n; j++ {
				if (i>>j)&1 == 1 {
					combo = append(combo, players[j])
				}
			}
			callback(combo)
		}
	}
}

func countBits(n int) int {
	count := 0
	for n > 0 {
		count += n & 1
		n >>= 1
	}
	return count
}

func getCurrentTimeMs() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func calculateCorrelationBonus(players []types.Player, weight float64) float64 {
	// Simple correlation bonus based on stacking
	bonus := 0.0
	teamCounts := make(map[string]int)
	gameCounts := make(map[string]int)

	for _, player := range players {
		teamCounts[player.Team]++
		gameKey := getGameKey(player.Team, player.Opponent)
		gameCounts[gameKey]++
	}

	// Team stack bonus
	for _, count := range teamCounts {
		if count >= 2 {
			bonus += math.Log(float64(count)) * weight * 2.0
		}
	}

	// Game stack bonus
	for _, count := range gameCounts {
		if count >= 3 {
			bonus += math.Log(float64(count)) * weight * 3.0
		}
	}

	return bonus
}
