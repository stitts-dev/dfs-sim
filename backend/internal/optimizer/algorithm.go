package optimizer

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/models"
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
	Contest             *models.Contest  `json:"-"`
}

type StackingRule struct {
	Type       string   `json:"type"` // "team", "game", "mini"
	MinPlayers int      `json:"min_players"`
	MaxPlayers int      `json:"max_players"`
	Teams      []string `json:"teams,omitempty"`
}

type OptimizerResult struct {
	Lineups           []models.Lineup `json:"lineups"`
	OptimizationTime  int64           `json:"optimization_time_ms"`
	TotalCombinations int64           `json:"total_combinations"`
	ValidCombinations int64           `json:"valid_combinations"`
}

type lineupCandidate struct {
	players         []models.Player
	totalSalary     int
	projectedPoints float64
	positions       map[string][]models.Player
	// Track which position slot each player fills (for flex positions)
	playerPositions map[uint]string // playerID -> position slot filled
}

func OptimizeLineups(players []models.Player, config OptimizeConfig) (*OptimizerResult, error) {
	startTime := getCurrentTimeMs()
	result := &OptimizerResult{}

	// Filter out excluded players
	filteredPlayers := filterPlayers(players, config)

	// Organize players by position
	playersByPosition := organizeByPosition(filteredPlayers)

	// Generate all valid lineup combinations
	validLineups := generateValidLineups(playersByPosition, config)

	// Sort by projected points (with correlation bonus if enabled)
	sort.Slice(validLineups, func(i, j int) bool {
		return validLineups[i].projectedPoints > validLineups[j].projectedPoints
	})

	// Apply diversity constraints and exposure limits
	finalLineups := applyDiversityConstraints(validLineups, config)

	// Convert to model lineups
	result.Lineups = make([]models.Lineup, 0, len(finalLineups))
	for i, candidate := range finalLineups {
		lineup := models.Lineup{
			ContestID:        config.Contest.ID,
			TotalSalary:      candidate.totalSalary,
			ProjectedPoints:  candidate.projectedPoints,
			IsOptimized:      true,
			OptimizationRank: i + 1,
			Players:          candidate.players,
			PlayerPositions:  candidate.playerPositions,
		}
		lineup.CalculateOwnership()
		result.Lineups = append(result.Lineups, lineup)
	}

	result.OptimizationTime = getCurrentTimeMs() - startTime
	result.ValidCombinations = int64(len(validLineups))

	return result, nil
}

func filterPlayers(players []models.Player, config OptimizeConfig) []models.Player {
	excludeMap := make(map[uint]bool)
	for _, id := range config.ExcludedPlayers {
		excludeMap[id] = true
	}

	filtered := make([]models.Player, 0, len(players))
	for _, player := range players {
		if !excludeMap[player.ID] && !player.IsInjured {
			filtered = append(filtered, player)
		}
	}

	return filtered
}

func organizeByPosition(players []models.Player) map[string][]models.Player {
	byPosition := make(map[string][]models.Player)
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

	return byPosition
}

func generateValidLineups(playersByPosition map[string][]models.Player, config OptimizeConfig) []lineupCandidate {
	var validLineups []lineupCandidate
	requirements := config.Contest.PositionRequirements

	// Use recursive backtracking to generate lineups
	var backtrack func(current *lineupCandidate, positionOrder []string, posIndex int)

	backtrack = func(current *lineupCandidate, positionOrder []string, posIndex int) {
		// Base case: all positions filled
		if posIndex >= len(positionOrder) {
			if isValidLineup(current, config) {
				// Create a copy and add to results
				lineupCopy := *current
				lineupCopy.players = make([]models.Player, len(current.players))
				copy(lineupCopy.players, current.players)
				// Deep copy playerPositions map
				lineupCopy.playerPositions = make(map[uint]string)
				for k, v := range current.playerPositions {
					lineupCopy.playerPositions[k] = v
				}
				validLineups = append(validLineups, lineupCopy)
			}
			return
		}

		position := positionOrder[posIndex]
		requiredCount := requirements[position]

		// Handle flex positions
		if position == "UTIL" || position == "FLEX" {
			eligiblePlayers := getFlexEligiblePlayers(playersByPosition, position, current)
			generateFlexCombinations(current, eligiblePlayers, requiredCount, positionOrder, posIndex, config, &backtrack)
			return
		}

		// Regular positions
		players := playersByPosition[position]
		generatePositionCombinations(current, players, position, requiredCount, positionOrder, posIndex, config, &backtrack)
	}

	// Initialize and start backtracking
	positionOrder := getPositionOrder(requirements)
	initial := &lineupCandidate{
		players:         make([]models.Player, 0, 9),
		positions:       make(map[string][]models.Player),
		playerPositions: make(map[uint]string),
	}

	// Add locked players first
	for _, playerID := range config.LockedPlayers {
		for _, playerList := range playersByPosition {
			for _, player := range playerList {
				if player.ID == playerID {
					initial.players = append(initial.players, player)
					initial.totalSalary += player.Salary
					initial.projectedPoints += player.ProjectedPoints
					initial.positions[player.Position] = append(initial.positions[player.Position], player)
					initial.playerPositions[player.ID] = player.Position
					break
				}
			}
		}
	}

	backtrack(initial, positionOrder, 0)

	return validLineups
}

func generatePositionCombinations(current *lineupCandidate, players []models.Player, position string, required int, positionOrder []string, posIndex int, config OptimizeConfig, backtrack *func(*lineupCandidate, []string, int)) {
	// Calculate how many we already have for this position
	existing := len(current.positions[position])
	needed := required - existing

	if needed <= 0 {
		// Position already filled, move to next
		(*backtrack)(current, positionOrder, posIndex+1)
		return
	}

	// Try different combinations of players
	generateCombinations(players, needed, func(combo []models.Player) {
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
		current.players = current.players[:len(current.players)-len(combo)]
		current.positions[position] = current.positions[position][:len(current.positions[position])-len(combo)]
		current.totalSalary -= additionalSalary
		current.projectedPoints -= additionalPoints + calculateCorrelationBonus(current.players, config.CorrelationWeight)
	})
}

func generateFlexCombinations(current *lineupCandidate, eligiblePlayers []models.Player, required int, positionOrder []string, posIndex int, config OptimizeConfig, backtrack *func(*lineupCandidate, []string, int)) {
	if required <= 0 {
		(*backtrack)(current, positionOrder, posIndex+1)
		return
	}

	// For flex positions, we can use players from multiple eligible positions
	generateCombinations(eligiblePlayers, required, func(combo []models.Player) {
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
		current.players = current.players[:len(current.players)-len(combo)]
		current.totalSalary -= additionalSalary
		current.projectedPoints -= additionalPoints + calculateCorrelationBonus(current.players, config.CorrelationWeight)
	})
}

func getFlexEligiblePlayers(playersByPosition map[string][]models.Player, flexType string, current *lineupCandidate) []models.Player {
	var eligible []models.Player

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
	// Check total players
	totalRequired := 0
	for _, count := range config.Contest.PositionRequirements {
		totalRequired += count
	}

	if len(lineup.players) != totalRequired {
		return false
	}

	// Check salary cap
	if lineup.totalSalary > config.SalaryCap {
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

func hasPlayer(players []models.Player, playerID uint) bool {
	for _, p := range players {
		if p.ID == playerID {
			return true
		}
	}
	return false
}

func getPositionOrder(requirements models.PositionRequirements) []string {
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

func generateCombinations(players []models.Player, k int, callback func([]models.Player)) {
	if k > len(players) {
		return
	}

	// Use iterative approach with bit manipulation for better performance
	n := len(players)
	for i := 0; i < (1 << n); i++ {
		if countBits(i) == k {
			combo := make([]models.Player, 0, k)
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

func calculateCorrelationBonus(players []models.Player, weight float64) float64 {
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
