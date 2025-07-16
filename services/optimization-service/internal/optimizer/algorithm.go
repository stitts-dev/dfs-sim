package optimizer

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stitts-dev/dfs-sim/shared/types"
	"github.com/stitts-dev/dfs-sim/shared/pkg/logger"
	"github.com/sirupsen/logrus"
)

type OptimizeConfig struct {
	SalaryCap           int                     `json:"salary_cap"`
	NumLineups          int                     `json:"num_lineups"`
	MinDifferentPlayers int                     `json:"min_different_players"`
	UseCorrelations     bool                    `json:"use_correlations"`
	CorrelationWeight   float64                 `json:"correlation_weight"`
	StackingRules       []types.StackingRule    `json:"stacking_rules"`
	LockedPlayers       []uuid.UUID             `json:"locked_players"`
	ExcludedPlayers     []uuid.UUID             `json:"excluded_players"`
	MinExposure         map[uuid.UUID]float64   `json:"min_exposure"`
	MaxExposure         map[uuid.UUID]float64   `json:"max_exposure"`
	Contest             *types.Contest          `json:"-"`
	
	// Portfolio-level constraints (optional)
	UsePortfolioConstraints bool                 `json:"use_portfolio_constraints"`
	PortfolioConfig         *PortfolioConstraint `json:"portfolio_config,omitempty"`
}


// PortfolioConstraint defines portfolio-level optimization constraints
type PortfolioConstraint struct {
	RiskAversion        float64                   `json:"risk_aversion"`
	MaxPositionSize     float64                   `json:"max_position_size"`
	MinDiversification  float64                   `json:"min_diversification"`
	UseRiskParity       bool                      `json:"use_risk_parity"`
	SportConstraints    map[string]PortfolioLimit `json:"sport_constraints"`
	TeamConstraints     map[string]PortfolioLimit `json:"team_constraints"`
	PlayerConstraints   map[string]PortfolioLimit `json:"player_constraints"`
	RegularizationParam float64                   `json:"regularization_param"`
}

// PortfolioLimit defines min/max allocation limits
type PortfolioLimit struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type OptimizerResult struct {
	Lineups           []types.GeneratedLineup `json:"lineups"`
	OptimizationTime  int64                   `json:"optimization_time_ms"`
	TotalCombinations int64                   `json:"total_combinations"`
	ValidCombinations int64                   `json:"valid_combinations"`
	Metadata          OptimizerMetadata       `json:"metadata"`
}

type OptimizerMetadata struct {
	ExecutionTime   time.Duration `json:"execution_time"`
	Algorithm       string        `json:"algorithm"`
	PerformanceMode string        `json:"performance_mode"`
}

type lineupCandidate struct {
	players         []types.Player
	totalSalary     int
	projectedPoints float64
	positions       map[string][]types.Player
	// Track which position slot each player fills (for flex positions)
	playerPositions map[uuid.UUID]string // playerID -> position slot filled
}

func OptimizeLineups(players []types.Player, config OptimizeConfig) (*OptimizerResult, error) {
	// Generate unique optimization ID for request tracing
	optimizationID := uuid.New().String()
	startTime := getCurrentTimeMs()
	result := &OptimizerResult{}

	// Initialize logger with optimization context
	sportID := "unknown"
	if config.Contest != nil {
		sportID = config.Contest.SportID.String()
	}
	logger := logger.WithOptimizationContext(optimizationID, sportID, config.Contest.Platform)
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

	// Apply portfolio-level optimization if enabled
	var finalLineups []lineupCandidate
	if config.UsePortfolioConstraints && config.PortfolioConfig != nil {
		logger.Info("Applying portfolio-level optimization")
		finalLineups = applyPortfolioOptimization(validLineups, config, logger)
	} else {
		// Apply diversity constraints and exposure limits (default behavior)
		finalLineups = applyDiversityConstraints(validLineups, config)
	}

	// Convert to model lineups
	result.Lineups = make([]types.GeneratedLineup, 0, len(finalLineups))
	for i, candidate := range finalLineups {
		// Convert Player slice to LineupPlayer slice
		lineupPlayers := make([]types.LineupPlayer, len(candidate.players))
		for j, player := range candidate.players {
			// Use appropriate salary based on platform
			salary := 0
			if player.SalaryDK != nil {
				salary = *player.SalaryDK
			}
			if strings.ToLower(config.Contest.Platform) == "fanduel" && player.SalaryFD != nil && *player.SalaryFD > 0 {
				salary = *player.SalaryFD
			}
			
			team := ""
			if player.Team != nil {
				team = *player.Team
			}
			
			position := ""
			if player.Position != nil {
				position = *player.Position
			}
			
			projectedPoints := 0.0
			if player.ProjectedPoints != nil {
				projectedPoints = *player.ProjectedPoints
			}
			
			lineupPlayers[j] = types.LineupPlayer{
				ID:              player.ID,
				Name:            player.Name,
				Team:            team,
				Position:        position,
				Salary:          salary,
				ProjectedPoints: projectedPoints,
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
	
	// Set metadata
	result.Metadata = OptimizerMetadata{
		ExecutionTime:   time.Duration(result.OptimizationTime) * time.Millisecond,
		Algorithm:       "standard",
		PerformanceMode: "balanced",
	}

	return result, nil
}

func filterPlayers(players []types.Player, config OptimizeConfig, logger *logrus.Entry) []types.Player {
	excludeMap := make(map[uuid.UUID]bool)
	for _, id := range config.ExcludedPlayers {
		excludeMap[id] = true
	}

	filtered := make([]types.Player, 0, len(players))
	excludedCount := 0
	injuredCount := 0

	for _, player := range players {
		if excludeMap[player.ID] {
			excludedCount++
		} else if player.IsInjured != nil && *player.IsInjured {
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
		position := ""
		if player.Position != nil {
			position = *player.Position
		}
		byPosition[position] = append(byPosition[position], player)
	}

	// Sort each position by value (projected points per dollar)
	for position := range byPosition {
		sort.Slice(byPosition[position], func(i, j int) bool {
			playerI := byPosition[position][i]
			playerJ := byPosition[position][j]
			
			// Handle nil values safely
			projI := 0.0
			if playerI.ProjectedPoints != nil {
				projI = *playerI.ProjectedPoints
			}
			projJ := 0.0
			if playerJ.ProjectedPoints != nil {
				projJ = *playerJ.ProjectedPoints
			}
			
			salaryI := 1 // Avoid division by zero
			if playerI.SalaryDK != nil && *playerI.SalaryDK > 0 {
				salaryI = *playerI.SalaryDK
			}
			salaryJ := 1 // Avoid division by zero
			if playerJ.SalaryDK != nil && *playerJ.SalaryDK > 0 {
				salaryJ = *playerJ.SalaryDK
			}
			
			valueI := projI / float64(salaryI)
			valueJ := projJ / float64(salaryJ)
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
				topPlayerNames = append(topPlayerNames, fmt.Sprintf("%s($%d)", p.Name, p.SalaryDK))
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

	// Early validation
	if config.Contest == nil {
		logger.Error("No contest provided to optimizer")
		return []lineupCandidate{}
	}

	// Get position slots for this sport/platform  
	sportName := getSportNameFromID(config.Contest.SportID)
	slots := GetPositionSlots(sportName, config.Contest.Platform)
	if len(slots) == 0 {
		logger.WithFields(logrus.Fields{
			"sport_id": config.Contest.SportID,
			"sport":    sportName,
			"platform": config.Contest.Platform,
		}).Error("No position slots found for contest")
		return []lineupCandidate{}
	}

	// Check if we should use the new DP optimizer based on config
	if shouldUseDP(config) {
		return generateLineupsWithDP(playersByPosition, config, logger)
	}

	// Fallback to original backtracking for backward compatibility
	return generateLineupsWithBacktracking(playersByPosition, config, logger, slots)
}

// shouldUseDP determines whether to use the new DP optimizer
func shouldUseDP(config OptimizeConfig) bool {
	// Use DP for larger optimizations or when explicitly enabled
	if config.NumLineups > 20 {
		return true
	}
	
	// Use DP for complex constraints
	if len(config.StackingRules) > 2 || len(config.LockedPlayers) > 3 {
		return true
	}
	
	return false
}

// generateLineupsWithDP uses the new dynamic programming optimizer with enhanced analytics
func generateLineupsWithDP(playersByPosition map[string][]types.Player, config OptimizeConfig, logger *logrus.Entry) []lineupCandidate {
	logger.Info("Using enhanced DP optimizer with analytics and multi-objective framework")
	
	// Convert playersByPosition back to a flat list
	allPlayers := make([]types.Player, 0)
	for _, players := range playersByPosition {
		allPlayers = append(allPlayers, players...)
	}
	
	// Initialize DP optimizer
	dpOptimizer := NewDPOptimizer()
	
	// Create enhanced configuration with strategy support
	configV2 := OptimizeConfigV2{
		// Preserve original config fields
		SalaryCap:           config.SalaryCap,
		NumLineups:          config.NumLineups,
		MinDifferentPlayers: config.MinDifferentPlayers,
		UseCorrelations:     config.UseCorrelations,
		CorrelationWeight:   config.CorrelationWeight,
		StackingRules:       config.StackingRules,
		LockedPlayers:       config.LockedPlayers,
		ExcludedPlayers:     config.ExcludedPlayers,
		MinExposure:         config.MinExposure,
		MaxExposure:         config.MaxExposure,
		Contest:             config.Contest,
		
		// Enhanced strategy options
		Strategy:            determineOptimizationStrategy(config),
		PlayerAnalytics:     true, // Enable analytics by default for DP
		PerformanceMode:     "balanced",
		
		// Exposure management configuration
		ExposureManagement: ExposureConfig{
			MaxPlayerExposure:   30.0, // Default 30% max exposure
			MaxTeamExposure:     40.0, // Default 40% max team exposure
			MaxGameExposure:     35.0, // Default 35% max game exposure
			MinUniquePlayers:    config.MinDifferentPlayers,
		},
		
		// Advanced constraints
		MaxCorrelation:    0.8,
		MinDiversity:      config.MinDifferentPlayers,
		OwnershipStrategy: "balanced",
	}
	
	// Use enhanced V2 optimizer for multiple lineups
	generatedLineups, err := dpOptimizer.OptimizeWithDPV2(allPlayers, configV2)
	if err != nil {
		logger.WithError(err).Error("Enhanced DP optimization failed")
		// Fallback to simpler single-lineup DP optimization
		return generateLineupsWithSimpleDP(allPlayers, config, logger)
	}
	
	// Convert GeneratedLineup back to lineupCandidate for compatibility
	validLineups := make([]lineupCandidate, 0, len(generatedLineups))
	for _, genLineup := range generatedLineups {
		// Convert LineupPlayer back to Player
		players := make([]types.Player, 0, len(genLineup.Players))
		
		for _, lineupPlayer := range genLineup.Players {
			// Find the original player data
			for _, player := range allPlayers {
				if player.ID == lineupPlayer.ID {
					players = append(players, player)
					break
				}
			}
		}
		
		if len(players) == len(genLineup.Players) {
			candidate := lineupCandidate{
				players:         players,
				totalSalary:     genLineup.TotalSalary,
				projectedPoints: genLineup.ProjectedPoints,
				positions:       make(map[string][]types.Player),
				playerPositions: make(map[uuid.UUID]string),
			}
			
			// Organize by position for compatibility
			for _, player := range players {
				position := ""
				if player.Position != nil {
					position = *player.Position
				}
				candidate.positions[position] = append(candidate.positions[position], player)
				candidate.playerPositions[player.ID] = position
			}
			
			validLineups = append(validLineups, candidate)
		}
	}
	
	logger.WithFields(logrus.Fields{
		"lineups_generated": len(validLineups),
		"strategy_used":     configV2.Strategy,
		"analytics_enabled": configV2.PlayerAnalytics,
	}).Info("Enhanced DP optimization completed successfully")
	
	return validLineups
}

// OptimizationObjective and constants are defined in objectives.go

// determineOptimizationStrategy determines the best strategy based on config
func determineOptimizationStrategy(config OptimizeConfig) OptimizationObjective {
	// Analyze config to determine best strategy
	if config.UseCorrelations && len(config.StackingRules) > 0 {
		return Correlation // Focus on stacking when correlation rules are present
	}
	
	// Check contest type for strategy hints
	if config.Contest != nil {
		contestType := strings.ToLower(config.Contest.ContestType)
		if strings.Contains(contestType, "gpp") || strings.Contains(contestType, "tournament") {
			return MaxCeiling // Tournament strategy for GPP contests
		}
		if strings.Contains(contestType, "cash") || strings.Contains(contestType, "50/50") {
			return MaxFloor // Cash game strategy for safer contests
		}
	}
	
	// Default to balanced strategy
	return Balanced
}

// generateLineupsWithSimpleDP provides fallback single-lineup DP optimization
func generateLineupsWithSimpleDP(allPlayers []types.Player, config OptimizeConfig, logger *logrus.Entry) []lineupCandidate {
	logger.Info("Using simple DP fallback optimization")
	
	dpOptimizer := NewDPOptimizer()
	validLineups := make([]lineupCandidate, 0, config.NumLineups)
	
	// Create exposure manager for diversity tracking
	exposureConfig := ExposureConfig{
		MaxPlayerExposure:   30.0,
		MaxTeamExposure:     40.0,
		MinUniquePlayers:    config.MinDifferentPlayers,
	}
	exposureManager := NewExposureManager(exposureConfig)
	
	for i := 0; i < config.NumLineups; i++ {
		platform := strings.ToLower(config.Contest.Platform)
		result, err := dpOptimizer.OptimizeWithDP(allPlayers, config, platform)
		
		if err != nil {
			logger.WithError(err).Warnf("Simple DP optimization failed for lineup %d", i+1)
			continue
		}
		
		// Convert DP result to lineupCandidate
		lineup := convertDPResultToLineup(result, allPlayers, config, logger)
		if lineup != nil && isValidLineup(lineup, config) {
			// Check exposure constraints
			canAdd := true
			for _, player := range lineup.players {
				team := ""
				if player.Team != nil {
					team = *player.Team
				}
				if !exposureManager.CanAddPlayer(player.ID, team, i) {
					canAdd = false
					break
				}
			}
			
			if canAdd {
				validLineups = append(validLineups, *lineup)
				
				// Track exposure
				for _, player := range lineup.players {
					team := ""
					if player.Team != nil {
						team = *player.Team
					}
					exposureManager.AddPlayerToLineup(player.ID, team, i)
				}
				exposureManager.CompleteLineup()
			}
		}
		
		// Remove some optimal players for diversity
		if result != nil && len(result.OptimalPlayers) > 0 {
			allPlayers = filterOutTopPlayers(allPlayers, result.OptimalPlayers, 2) // Remove top 2 players
		}
	}
	
	return validLineups
}

// generateLineupsWithBacktracking uses the original backtracking algorithm
func generateLineupsWithBacktracking(playersByPosition map[string][]types.Player, config OptimizeConfig, logger *logrus.Entry, slots []PositionSlot) []lineupCandidate {
	logger.Debug("Using backtracking optimizer for lineup generation")
	
	var validLineups []lineupCandidate

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
	var backtrack func(current *lineupCandidate, slotIndex int, usedPlayers map[uuid.UUID]bool)

	backtrack = func(current *lineupCandidate, slotIndex int, usedPlayers map[uuid.UUID]bool) {
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
				lineupCopy.playerPositions = make(map[uuid.UUID]string)
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
				playerSalary := getSalaryForPlatform(player, config.Contest.Platform)
				if current.totalSalary+playerSalary > config.SalaryCap {
					continue
				}

				playersTried++

				// Add player to lineup
				current.players = append(current.players, player)
				current.totalSalary += playerSalary
				if player.ProjectedPoints != nil {
					current.projectedPoints += *player.ProjectedPoints
				}
				current.playerPositions[player.ID] = slot.SlotName
				usedPlayers[player.ID] = true

				// Recurse to fill next slot
				backtrack(current, slotIndex+1, usedPlayers)

				// Backtrack
				current.players = current.players[:len(current.players)-1]
				current.totalSalary -= playerSalary
				if player.ProjectedPoints != nil {
					current.projectedPoints -= *player.ProjectedPoints
				}
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
		playerPositions: make(map[uuid.UUID]string),
	}

	usedPlayers := make(map[uuid.UUID]bool)

	// For locked players, we'll track them but let the backtracking algorithm handle placement
	lockedPlayerSet := make(map[uuid.UUID]bool)
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
			additionalSalary += getSalaryForPlatform(player, config.Contest.Platform)
			if player.ProjectedPoints != nil {
				additionalPoints += *player.ProjectedPoints
			}
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
			additionalSalary += getSalaryForPlatform(player, config.Contest.Platform)
			if player.ProjectedPoints != nil {
				additionalPoints += *player.ProjectedPoints
			}
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
		salaryI := getSalaryForPlatform(eligible[i], "draftkings")
		salaryJ := getSalaryForPlatform(eligible[j], "draftkings")
		
		projI := 0.0
		if eligible[i].ProjectedPoints != nil {
			projI = *eligible[i].ProjectedPoints
		}
		projJ := 0.0
		if eligible[j].ProjectedPoints != nil {
			projJ = *eligible[j].ProjectedPoints
		}
		
		valueI := projI / float64(salaryI)
		valueJ := projJ / float64(salaryJ)
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

func validateStackingRules(lineup *lineupCandidate, rules []types.StackingRule) bool {
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

func validateTeamStacking(lineup *lineupCandidate, rule types.StackingRule) bool {
	teamCounts := make(map[string]int)
	for _, player := range lineup.players {
		team := ""
		if player.Team != nil {
			team = *player.Team
		}
		teamCounts[team]++
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

func validateGameStacking(lineup *lineupCandidate, rule types.StackingRule) bool {
	gameCounts := make(map[string]int)

	for _, player := range lineup.players {
		team := ""
		if player.Team != nil {
			team = *player.Team
		}
		opponent := ""
		if player.Opponent != nil {
			opponent = *player.Opponent
		}
		gameKey := getGameKey(team, opponent)
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
	playerExposure := make(map[uuid.UUID]int)

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

func hasPlayer(players []types.Player, playerID uuid.UUID) bool {
	for _, p := range players {
		if p.ID == playerID {
			return true
		}
	}
	return false
}

func isPlayerLocked(playerID uuid.UUID, lockedPlayers []uuid.UUID) bool {
	for _, id := range lockedPlayers {
		if id == playerID {
			return true
		}
	}
	return false
}

func mapToSlice(playerMap map[uuid.UUID]types.Player) []types.Player {
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
	playerMap := make(map[uuid.UUID]bool)
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
		team := ""
		if player.Team != nil {
			team = *player.Team
		}
		teamCounts[team]++
		
		opponent := ""
		if player.Opponent != nil {
			opponent = *player.Opponent
		}
		gameKey := getGameKey(team, opponent)
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

// convertDPResultToLineup converts DP optimization result to lineupCandidate
func convertDPResultToLineup(result *DPResult, allPlayers []types.Player, config OptimizeConfig, logger *logrus.Entry) *lineupCandidate {
	if result == nil || len(result.OptimalPlayers) == 0 {
		return nil
	}
	
	// Create player map for quick lookup
	playerMap := make(map[uuid.UUID]types.Player)
	for _, player := range allPlayers {
		playerMap[player.ID] = player
	}
	
	// Build lineup from optimal player IDs
	lineup := &lineupCandidate{
		players:         make([]types.Player, 0, len(result.OptimalPlayers)),
		positions:       make(map[string][]types.Player),
		playerPositions: make(map[uuid.UUID]string),
		totalSalary:     0,
		projectedPoints: 0,
	}
	
	for _, playerID := range result.OptimalPlayers {
		if player, exists := playerMap[playerID]; exists && playerID != uuid.Nil {
			lineup.players = append(lineup.players, player)
			// Use DK salary by default
			if player.SalaryDK != nil {
				lineup.totalSalary += *player.SalaryDK
			}
			if player.ProjectedPoints != nil {
				lineup.projectedPoints += *player.ProjectedPoints
			}
			
			// Track by position
			position := ""
			if player.Position != nil {
				position = *player.Position
			}
			if lineup.positions[position] == nil {
				lineup.positions[position] = make([]types.Player, 0)
			}
			lineup.positions[position] = append(lineup.positions[position], player)
		} else {
			logger.WithField("player_id", playerID).Warn("Optimal player not found in player map")
		}
	}
	
	return lineup
}

// filterOutPlayers removes specified players from the player list to ensure diversity
func filterOutPlayers(players []types.Player, playerIDsToRemove []uuid.UUID) []types.Player {
	if len(playerIDsToRemove) == 0 {
		return players
	}
	
	// Create set of IDs to remove
	removeSet := make(map[uuid.UUID]bool)
	for _, id := range playerIDsToRemove {
		removeSet[id] = true
	}
	
	// Filter out players
	filtered := make([]types.Player, 0, len(players))
	for _, player := range players {
		if !removeSet[player.ID] {
			filtered = append(filtered, player)
		}
	}
	
	return filtered
}

// applyPortfolioOptimization applies portfolio-level constraints and optimization
func applyPortfolioOptimization(lineups []lineupCandidate, config OptimizeConfig, logger *logrus.Entry) []lineupCandidate {
	if config.PortfolioConfig == nil || len(lineups) == 0 {
		return lineups
	}

	portfolioConfig := config.PortfolioConfig
	logger.WithFields(logrus.Fields{
		"lineups_count":     len(lineups),
		"risk_aversion":     portfolioConfig.RiskAversion,
		"max_position_size": portfolioConfig.MaxPositionSize,
		"use_risk_parity":   portfolioConfig.UseRiskParity,
	}).Info("Starting portfolio optimization")

	// Convert lineups to portfolio format
	portfolioLineups := convertLineupsToPortfolioFormat(lineups)

	// Apply portfolio constraints
	constrainedLineups := applyPortfolioConstraints(portfolioLineups, portfolioConfig, logger)

	// Calculate portfolio weights using Modern Portfolio Theory
	weights := calculateOptimalPortfolioWeights(constrainedLineups, portfolioConfig, logger)

	// Select lineups based on optimal weights
	selectedLineups := selectLineupsFromWeights(constrainedLineups, weights, config.NumLineups, logger)

	logger.WithFields(logrus.Fields{
		"original_lineups": len(lineups),
		"final_lineups":    len(selectedLineups),
		"optimization":     "portfolio_complete",
	}).Info("Portfolio optimization completed")

	return selectedLineups
}

// convertLineupsToPortfolioFormat converts lineupCandidate to portfolio analysis format
func convertLineupsToPortfolioFormat(lineups []lineupCandidate) []lineupCandidate {
	// Add lineup metadata for portfolio analysis
	for i := range lineups {
		// Calculate lineup risk score based on player variance
		riskScore := calculateLineupRisk(lineups[i])
		
		// Store risk score in projected points for portfolio optimization
		// This is a simplified approach - in practice, you'd extend the struct
		lineups[i].projectedPoints = lineups[i].projectedPoints * (1.0 - riskScore*0.1)
	}
	return lineups
}

// calculateLineupRisk calculates risk score for a lineup
func calculateLineupRisk(lineup lineupCandidate) float64 {
	if len(lineup.players) == 0 {
		return 0.0
	}

	// Calculate risk based on:
	// 1. Player concentration (same team)
	// 2. Salary concentration (high-priced players)
	// 3. Position concentration

	teamCounts := make(map[string]int)
	highSalaryCount := 0

	for _, player := range lineup.players {
		team := ""
		if player.Team != nil {
			team = *player.Team
		}
		teamCounts[team]++
		
		// Count high-salary players (top 25% salary range)
		if player.SalaryDK != nil && *player.SalaryDK > 8000 { // Simplified threshold using DK salary
			highSalaryCount++
		}
	}

	// Team concentration risk
	teamRisk := 0.0
	for _, count := range teamCounts {
		concentration := float64(count) / float64(len(lineup.players))
		if concentration > 0.4 { // More than 40% from one team
			teamRisk += concentration - 0.4
		}
	}

	// Salary concentration risk
	salaryRisk := 0.0
	if highSalaryCount > len(lineup.players)/2 {
		salaryRisk = 0.2 // High salary concentration
	}

	// Combined risk score (0-1 scale)
	return math.Min(1.0, teamRisk + salaryRisk)
}

// applyPortfolioConstraints applies portfolio-level constraints to lineups
func applyPortfolioConstraints(lineups []lineupCandidate, config *PortfolioConstraint, logger *logrus.Entry) []lineupCandidate {
	constrainedLineups := make([]lineupCandidate, 0, len(lineups))

	for _, lineup := range lineups {
		if validatePortfolioConstraints(lineup, config) {
			constrainedLineups = append(constrainedLineups, lineup)
		}
	}

	logger.WithFields(logrus.Fields{
		"original_count":    len(lineups),
		"constrained_count": len(constrainedLineups),
		"filtered_out":      len(lineups) - len(constrainedLineups),
	}).Debug("Portfolio constraints applied")

	return constrainedLineups
}

// validatePortfolioConstraints checks if lineup meets portfolio constraints
func validatePortfolioConstraints(lineup lineupCandidate, config *PortfolioConstraint) bool {
	// Check team concentration limits
	if len(config.TeamConstraints) > 0 {
		teamCounts := make(map[string]int)
		for _, player := range lineup.players {
			team := ""
		if player.Team != nil {
			team = *player.Team
		}
		teamCounts[team]++
		}

		for team, limit := range config.TeamConstraints {
			count := teamCounts[team]
			concentration := float64(count) / float64(len(lineup.players))
			if concentration < limit.Min || concentration > limit.Max {
				return false
			}
		}
	}

	// Check player concentration limits
	if len(config.PlayerConstraints) > 0 {
		for _, player := range lineup.players {
			if limit, exists := config.PlayerConstraints[fmt.Sprintf("%d", player.ID)]; exists {
				// For individual players, constraint is binary (0 or 1)
				if limit.Min > 0 && limit.Max < 1 {
					return false // Player not allowed
				}
			}
		}
	}

	// Check diversification requirement
	if config.MinDiversification > 0 {
		diversificationScore := calculateLineupDiversification(lineup)
		if diversificationScore < config.MinDiversification {
			return false
		}
	}

	return true
}

// calculateLineupDiversification calculates diversification score for a lineup
func calculateLineupDiversification(lineup lineupCandidate) float64 {
	if len(lineup.players) <= 1 {
		return 0.0
	}

	// Calculate Herfindahl-Hirschman Index for teams
	teamCounts := make(map[string]int)
	for _, player := range lineup.players {
		team := ""
		if player.Team != nil {
			team = *player.Team
		}
		teamCounts[team]++
	}

	hhi := 0.0
	totalPlayers := float64(len(lineup.players))
	for _, count := range teamCounts {
		share := float64(count) / totalPlayers
		hhi += share * share
	}

	// Diversification score = 1 - HHI (higher is more diversified)
	return 1.0 - hhi
}

// calculateOptimalPortfolioWeights calculates optimal weights using portfolio theory
func calculateOptimalPortfolioWeights(lineups []lineupCandidate, config *PortfolioConstraint, logger *logrus.Entry) map[int]float64 {
	n := len(lineups)
	if n == 0 {
		return make(map[int]float64)
	}

	weights := make(map[int]float64)

	if config.UseRiskParity {
		// Risk parity: equal risk contribution
		for i := 0; i < n; i++ {
			weights[i] = 1.0 / float64(n)
		}
	} else {
		// Mean-variance optimization
		returns := make([]float64, n)
		risks := make([]float64, n)

		// Calculate expected returns and risks
		for i, lineup := range lineups {
			returns[i] = lineup.projectedPoints
			risks[i] = calculateLineupRisk(lineup)
		}

		// Simple mean-variance allocation
		totalReturn := 0.0
		totalRisk := 0.0
		for i := 0; i < n; i++ {
			totalReturn += returns[i]
			totalRisk += risks[i]
		}

		// Weight by risk-adjusted return
		for i := 0; i < n; i++ {
			if totalRisk > 0 {
				riskAdjustedReturn := returns[i] / (1.0 + risks[i]*config.RiskAversion)
				weights[i] = riskAdjustedReturn / totalReturn
			} else {
				weights[i] = 1.0 / float64(n)
			}
		}

		// Normalize weights
		totalWeight := 0.0
		for _, w := range weights {
			totalWeight += w
		}
		if totalWeight > 0 {
			for i := range weights {
				weights[i] /= totalWeight
			}
		}
	}

	// Apply position size limits
	maxWeight := config.MaxPositionSize
	if maxWeight <= 0 {
		maxWeight = 1.0 / float64(n) * 2.0 // Default: 2x equal weight
	}

	for i := range weights {
		if weights[i] > maxWeight {
			weights[i] = maxWeight
		}
	}

	// Re-normalize after applying limits
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}
	if totalWeight > 0 {
		for i := range weights {
			weights[i] /= totalWeight
		}
	}

	logger.WithFields(logrus.Fields{
		"weights_calculated": len(weights),
		"max_weight":        getMaxWeight(weights),
		"min_weight":        getMinWeight(weights),
	}).Debug("Portfolio weights calculated")

	return weights
}

// selectLineupsFromWeights selects lineups based on portfolio weights
func selectLineupsFromWeights(lineups []lineupCandidate, weights map[int]float64, numLineups int, logger *logrus.Entry) []lineupCandidate {
	if numLineups <= 0 || len(lineups) == 0 {
		return []lineupCandidate{}
	}

	// Create weighted selection
	weightedLineups := make([]WeightedLineup, 0, len(lineups))
	for i, lineup := range lineups {
		if weight, exists := weights[i]; exists && weight > 0 {
			weightedLineups = append(weightedLineups, WeightedLineup{
				lineup: lineup,
				weight: weight,
				index:  i,
			})
		}
	}

	// Sort by weight (descending)
	sort.Slice(weightedLineups, func(i, j int) bool {
		return weightedLineups[i].weight > weightedLineups[j].weight
	})

	// Select top lineups up to numLineups
	selectedCount := numLineups
	if selectedCount > len(weightedLineups) {
		selectedCount = len(weightedLineups)
	}

	selectedLineups := make([]lineupCandidate, selectedCount)
	for i := 0; i < selectedCount; i++ {
		selectedLineups[i] = weightedLineups[i].lineup
	}

	logger.WithFields(logrus.Fields{
		"available_lineups": len(weightedLineups),
		"selected_lineups":  selectedCount,
		"top_weight":       getTopWeight(weightedLineups),
	}).Debug("Lineups selected from portfolio weights")

	return selectedLineups
}

// getSportNameFromID converts SportID to sport name string
// TODO: This should query the database or use a proper mapping service
func getSportNameFromID(sportID uuid.UUID) string {
	// For now, return a default sport - in production this would query the database
	// or use a mapping service to convert SportID to sport name
	return "nba" // Default to NBA for now
}

// filterOutTopPlayers removes top N players from the list for diversity
func filterOutTopPlayers(players []types.Player, optimalPlayerIDs []uuid.UUID, countToRemove int) []types.Player {
	if countToRemove <= 0 || len(optimalPlayerIDs) == 0 {
		return players
	}
	
	// Only remove up to the specified count
	removeCount := countToRemove
	if removeCount > len(optimalPlayerIDs) {
		removeCount = len(optimalPlayerIDs)
	}
	
	removeMap := make(map[uuid.UUID]bool)
	for i := 0; i < removeCount; i++ {
		removeMap[optimalPlayerIDs[i]] = true
	}
	
	filtered := make([]types.Player, 0, len(players))
	for _, player := range players {
		if !removeMap[player.ID] {
			filtered = append(filtered, player)
		}
	}
	
	return filtered
}

// Helper functions for portfolio optimization

func getMaxWeight(weights map[int]float64) float64 {
	max := 0.0
	for _, w := range weights {
		if w > max {
			max = w
		}
	}
	return max
}

func getMinWeight(weights map[int]float64) float64 {
	if len(weights) == 0 {
		return 0.0
	}
	min := 1.0
	for _, w := range weights {
		if w < min {
			min = w
		}
	}
	return min
}

type WeightedLineup struct {
	lineup lineupCandidate
	weight float64
	index  int
}

func getTopWeight(weightedLineups []WeightedLineup) float64 {
	if len(weightedLineups) == 0 {
		return 0.0
	}
	return weightedLineups[0].weight
}

// getSalaryForPlatform gets the appropriate salary for the platform
func getSalaryForPlatform(player types.Player, platform string) int {
	if strings.ToLower(platform) == "fanduel" && player.SalaryFD != nil && *player.SalaryFD > 0 {
		return *player.SalaryFD
	}
	if player.SalaryDK != nil && *player.SalaryDK > 0 {
		return *player.SalaryDK
	}
	// Fallback to FD if DK is 0 or nil
	if player.SalaryFD != nil {
		return *player.SalaryFD
	}
	return 0
}
