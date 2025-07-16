package optimizer

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// OptimizationState represents the state in dynamic programming
type OptimizationState struct {
	Budget           int                 `json:"budget"`
	Positions        map[string]int      `json:"positions"`      // Position -> remaining slots
	UsedPlayers      []uuid.UUID         `json:"used_players"`
	CurrentScore     float64             `json:"current_score"`
	CorrelationBonus float64             `json:"correlation_bonus"`
	StateHash        string              `json:"state_hash"`     // For memoization
}

// OptimizeConfigV2 represents enhanced optimization configuration with new strategies
type OptimizeConfigV2 struct {
	// Existing fields preserved for backward compatibility
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

	// New strategy options
	Strategy           OptimizationObjective `json:"strategy"`
	PlayerAnalytics    bool                  `json:"enable_analytics"`
	ExposureManagement ExposureConfig        `json:"exposure_config"`
	PerformanceMode    string               `json:"performance_mode"`  // "speed", "quality", "balanced"

	// Advanced constraints
	MaxCorrelation     float64              `json:"max_correlation"`
	MinDiversity       int                  `json:"min_diversity"`
	OwnershipStrategy  string               `json:"ownership_strategy"` // "contrarian", "chalk", "balanced"
}

// ExposureConfig represents exposure management configuration
type ExposureConfig struct {
	MaxPlayerExposure float64            `json:"max_player_exposure"`
	MaxTeamExposure   float64            `json:"max_team_exposure"`
	MaxGameExposure   float64            `json:"max_game_exposure"`
	MinUniquePlayers  int                `json:"min_unique_players"`
	MinDifferentPlayers int              `json:"min_different_players"`
	StackExposure     map[string]float64 `json:"stack_exposure"`
}

// DPOptimizer implements dynamic programming based optimization
type DPOptimizer struct {
	logger           *logrus.Entry
	memoTable        map[string]*OptimizationState
	analytics        *AnalyticsEngine
	correlations     *CorrelationMatrix
	maxDepth         int
	pruningEnabled   bool
	mutex            sync.RWMutex
	stats            DPStats
}

// DPStats tracks optimization statistics
type DPStats struct {
	StatesCached     int64         `json:"states_cached"`
	CacheHits        int64         `json:"cache_hits"`
	CacheMisses      int64         `json:"cache_misses"`
	OptimizationTime time.Duration `json:"optimization_time"`
	MemoryUsage      int64         `json:"memory_usage_bytes"`
}

// DPResult represents the result of DP optimization
type DPResult struct {
	OptimalScore     float64             `json:"optimal_score"`
	OptimalPlayers   []uuid.UUID         `json:"optimal_players"`
	StatesExplored   int                 `json:"states_explored"`
	CacheHitRate     float64             `json:"cache_hit_rate"`
	OptimizationTime time.Duration       `json:"optimization_time"`
}

// NewDPOptimizer creates a new dynamic programming optimizer
func NewDPOptimizer() *DPOptimizer {
	return &DPOptimizer{
		logger:         logrus.WithField("component", "dp_optimizer"),
		memoTable:      make(map[string]*OptimizationState),
		analytics:      NewAnalyticsEngine(),
		maxDepth:       15, // Maximum position slots to fill
		pruningEnabled: true,
		mutex:          sync.RWMutex{},
		stats:          DPStats{},
	}
}

// convertPlayersToOptimization converts types.Player slice to OptimizationPlayer slice
func (dp *DPOptimizer) convertPlayersToOptimization(players []types.Player) []OptimizationPlayer {
	result := make([]OptimizationPlayer, len(players))
	for i, p := range players {
		result[i] = OptimizationPlayer{
			ID:              p.ID,               // concrete
			ExternalID:      p.ExternalID,       // concrete
			Name:            p.Name,             // concrete
			Team:            getStringValue(p.Team),
			Opponent:        getStringValue(p.Opponent),
			Position:        getStringValue(p.Position),
			SalaryDK:        getIntValue(p.SalaryDK),
			SalaryFD:        getIntValue(p.SalaryFD),
			ProjectedPoints: getFloatValue(p.ProjectedPoints),
			FloorPoints:     getFloatValue(p.FloorPoints),
			CeilingPoints:   getFloatValue(p.CeilingPoints),
			OwnershipDK:     getFloatValue(p.OwnershipDK),
			OwnershipFD:     getFloatValue(p.OwnershipFD),
			GameTime:        getTimeValue(p.GameTime),
			IsInjured:       getBoolValue(p.IsInjured),
			InjuryStatus:    getStringValue(p.InjuryStatus),
			ImageURL:        getStringValue(p.ImageURL),
			// Golf-specific fields - use empty defaults since types.Player doesn't have them
			TeeTime:         "",
			CutProbability:  0.0,
			CreatedAt:       p.CreatedAt,        // concrete
			UpdatedAt:       p.UpdatedAt,        // concrete
		}
	}
	return result
}

// convertSinglePlayerToOptimization converts a single types.Player to OptimizationPlayer
func (dp *DPOptimizer) convertSinglePlayerToOptimization(p types.Player) OptimizationPlayer {
	return OptimizationPlayer{
		ID:              p.ID,               // concrete
		ExternalID:      p.ExternalID,       // concrete
		Name:            p.Name,             // concrete
		Team:            getStringValue(p.Team),
		Opponent:        getStringValue(p.Opponent),
		Position:        getStringValue(p.Position),
		SalaryDK:        getIntValue(p.SalaryDK),
		SalaryFD:        getIntValue(p.SalaryFD),
		ProjectedPoints: getFloatValue(p.ProjectedPoints),
		FloorPoints:     getFloatValue(p.FloorPoints),
		CeilingPoints:   getFloatValue(p.CeilingPoints),
		OwnershipDK:     getFloatValue(p.OwnershipDK),
		OwnershipFD:     getFloatValue(p.OwnershipFD),
		GameTime:        getTimeValue(p.GameTime),
		IsInjured:       getBoolValue(p.IsInjured),
		InjuryStatus:    getStringValue(p.InjuryStatus),
		ImageURL:        getStringValue(p.ImageURL),
		// Golf-specific fields - use empty defaults since types.Player doesn't have them
		TeeTime:         "",
		CutProbability:  0.0,
		CreatedAt:       p.CreatedAt,        // concrete
		UpdatedAt:       p.UpdatedAt,        // concrete
	}
}

// Helper functions to safely extract values from pointers
func getStringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

func getIntValue(ptr *int) int {
	if ptr != nil {
		return *ptr
	}
	return 0
}

func getFloatValue(ptr *float64) float64 {
	if ptr != nil {
		return *ptr
	}
	return 0.0
}

func getBoolValue(ptr *bool) bool {
	if ptr != nil {
		return *ptr
	}
	return false
}

func getTimeValue(ptr *time.Time) time.Time {
	if ptr != nil {
		return *ptr
	}
	return time.Time{}
}

// OptimizeWithDPV2 performs enhanced dynamic programming optimization
func (dp *DPOptimizer) OptimizeWithDPV2(players []types.Player, config OptimizeConfigV2) ([]types.GeneratedLineup, error) {
	startTime := time.Now()
	dp.logger.WithFields(logrus.Fields{
		"total_players": len(players),
		"salary_cap":    config.SalaryCap,
		"num_lineups":   config.NumLineups,
		"strategy":      config.Strategy,
	}).Info("Starting enhanced DP optimization")

	// Input validation
	if len(players) == 0 {
		return nil, fmt.Errorf("no players provided for optimization")
	}
	if config.Contest == nil {
		return nil, fmt.Errorf("no contest configuration provided")
	}

	// Filter excluded players
	filteredPlayers := dp.filterPlayers(players, config)
	if len(filteredPlayers) == 0 {
		return nil, fmt.Errorf("no players available after filtering")
	}

	// Convert to optimization players for algorithm compatibility
	optimizationPlayers := dp.convertPlayersToOptimization(filteredPlayers)

	// Calculate player analytics if enabled
	var enhancedPlayers []EnhancedPlayer
	if config.PlayerAnalytics {
		analytics, err := dp.calculateAnalytics(filteredPlayers)
		if err != nil {
			dp.logger.WithError(err).Warn("Failed to calculate analytics, using base projections")
			enhancedPlayers = dp.convertToEnhanced(filteredPlayers, nil)
		} else {
			enhancedPlayers = dp.analytics.EnhancePlayersWithAnalytics(filteredPlayers, analytics)
		}
	} else {
		enhancedPlayers = dp.convertToEnhanced(filteredPlayers, nil)
	}

	// Sort players by value rating for better pruning (research-backed optimization)
	dp.sortPlayersByValue(enhancedPlayers, config.Strategy)

	// Initialize correlation matrix
	dp.correlations = NewCorrelationMatrix(optimizationPlayers)

	// Get position slots for this sport/platform
	sportName := getSportNameFromID(config.Contest.SportID)
	slots := GetPositionSlots(sportName, config.Contest.Platform)
	if len(slots) == 0 {
		return nil, fmt.Errorf("no position slots found for sport %s (ID: %s), platform %s",
			sportName, config.Contest.SportID.String(), config.Contest.Platform)
	}

	// Clear memoization table for fresh optimization
	dp.clearMemoTable()

	// Generate lineups using DP
	lineups, err := dp.generateLineupsDP(enhancedPlayers, slots, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate lineups: %v", err)
	}

	// Apply diversity constraints and exposure management
	finalLineups := dp.applyConstraints(lineups, config)

	// Convert to API format
	result := dp.convertToGeneratedLineups(finalLineups, config)

	dp.stats.OptimizationTime = time.Since(startTime)
	dp.logOptimizationStats()

	dp.logger.WithFields(logrus.Fields{
		"lineups_generated": len(result),
		"optimization_time": dp.stats.OptimizationTime,
		"cache_hit_rate":    float64(dp.stats.CacheHits) / float64(dp.stats.CacheHits + dp.stats.CacheMisses),
	}).Info("Enhanced DP optimization completed")

	return result, nil
}

// OptimizeWithDP provides backward compatibility with the original interface
func (dp *DPOptimizer) OptimizeWithDP(players []types.Player, config OptimizeConfig, platform string) (*DPResult, error) {
	// Convert old config to new format
	configV2 := OptimizeConfigV2{
		SalaryCap:           config.SalaryCap,
		NumLineups:          1, // Original interface optimizes single lineup
		MinDifferentPlayers: config.MinDifferentPlayers,
		UseCorrelations:     config.UseCorrelations,
		CorrelationWeight:   config.CorrelationWeight,
		StackingRules:       config.StackingRules,
		LockedPlayers:       config.LockedPlayers,
		ExcludedPlayers:     config.ExcludedPlayers,
		MinExposure:         config.MinExposure,
		MaxExposure:         config.MaxExposure,
		Contest:             config.Contest,
		Strategy:            Balanced, // Default strategy
		PlayerAnalytics:     true,     // Enable analytics by default
		PerformanceMode:     "balanced",
	}

	// Use enhanced optimizer
	lineups, err := dp.OptimizeWithDPV2(players, configV2)
	if err != nil {
		return nil, err
	}

	if len(lineups) == 0 {
		return nil, fmt.Errorf("no valid lineup found")
	}

	// Convert first lineup back to old format
	lineup := lineups[0]
	playerIDs := make([]uuid.UUID, len(lineup.Players))
	for i, player := range lineup.Players {
		playerIDs[i] = player.ID
	}

	return &DPResult{
		OptimalScore:     lineup.ProjectedPoints,
		OptimalPlayers:   playerIDs,
		StatesExplored:   int(dp.stats.StatesCached),
		CacheHitRate:     float64(dp.stats.CacheHits) / float64(dp.stats.CacheHits + dp.stats.CacheMisses),
		OptimizationTime: dp.stats.OptimizationTime,
	}, nil
}

// Supporting methods for enhanced DP optimization

// filterPlayers removes excluded and injured players
func (dp *DPOptimizer) filterPlayers(players []types.Player, config OptimizeConfigV2) []types.Player {
	excludeMap := make(map[uint]bool)
	for _, id := range config.ExcludedPlayers {
		excludeMap[uint(id.ID())] = true
	}

	filtered := make([]types.Player, 0, len(players))
	for _, player := range players {
		playerID := uint(player.ID.ID())
		if excludeMap[playerID] || (player.IsInjured != nil && *player.IsInjured) {
			continue
		}
		filtered = append(filtered, player)
	}

	dp.logger.WithFields(logrus.Fields{
		"original_count": len(players),
		"filtered_count": len(filtered),
		"excluded_count": len(config.ExcludedPlayers),
	}).Debug("Player filtering completed")

	return filtered
}

// calculateAnalytics computes analytics for all players
func (dp *DPOptimizer) calculateAnalytics(players []types.Player) (map[uuid.UUID]*PlayerAnalytics, error) {
	// In production, this would load historical data from database
	historicalDataMap := make(map[uuid.UUID][]PerformanceData)

	return dp.analytics.CalculateBulkAnalytics(players, historicalDataMap)
}

// convertToEnhanced converts players to enhanced format
func (dp *DPOptimizer) convertToEnhanced(players []types.Player, analytics map[uuid.UUID]*PlayerAnalytics) []EnhancedPlayer {
	enhanced := make([]EnhancedPlayer, len(players))
	for i, player := range players {
		playerID := player.ID
		enhanced[i] = EnhancedPlayer{
			Player:    player,
			Analytics: analytics[playerID],
		}
	}
	return enhanced
}

// sortPlayersByValue sorts players by value rating for optimization efficiency
func (dp *DPOptimizer) sortPlayersByValue(players []EnhancedPlayer, strategy OptimizationObjective) {
	sort.Slice(players, func(i, j int) bool {
		scoreI := dp.analytics.GetObjectiveScore(players[i], strategy)
		scoreJ := dp.analytics.GetObjectiveScore(players[j], strategy)
		return scoreI > scoreJ
	})
}

// generateLineupsDP generates lineups using dynamic programming
func (dp *DPOptimizer) generateLineupsDP(players []EnhancedPlayer, slots []PositionSlot, config OptimizeConfigV2) ([]*lineupCandidate, error) {
	maxLineups := config.NumLineups * 100 // Generate extra for diversity filtering
	if maxLineups > 10000 {
		maxLineups = 10000
	}

	lineups := make([]*lineupCandidate, 0, maxLineups)

	// Use simplified approach for now - can be enhanced with full DP table later
	for i := 0; i < config.NumLineups; i++ {
		lineup := dp.generateSingleLineup(players, slots, config)
		if lineup != nil {
			lineups = append(lineups, lineup)
		}
	}

	return lineups, nil
}

// generateSingleLineup generates a single optimal lineup
func (dp *DPOptimizer) generateSingleLineup(players []EnhancedPlayer, slots []PositionSlot, config OptimizeConfigV2) *lineupCandidate {
	// Simplified greedy approach with analytics - can be enhanced to full DP
	selectedPlayers := make([]types.Player, 0, len(slots))
	totalSalary := 0
	totalScore := 0.0
	usedPlayers := make(map[uint]bool)

	for _, slot := range slots {
		bestPlayer := types.Player{}
		bestScore := -1.0

		for _, enhancedPlayer := range players {
			player := enhancedPlayer.Player
			playerID := uint(player.ID.ID())

			if usedPlayers[playerID] {
				continue
			}

			// Convert to OptimizationPlayer for slot checking
			optimizationPlayer := dp.convertSinglePlayerToOptimization(player)
			if !CanPlayerFillSlot(optimizationPlayer, slot) {
				continue
			}

			salary := dp.getPlayerSalary(player, config)
			if totalSalary+salary > config.SalaryCap {
				continue
			}

			score := dp.analytics.GetObjectiveScore(enhancedPlayer, config.Strategy)
			if score > bestScore {
				bestScore = score
				bestPlayer = player
			}
		}

		if bestPlayer.ID != uuid.Nil {
			selectedPlayers = append(selectedPlayers, bestPlayer)
			playerID := uint(bestPlayer.ID.ID())
			usedPlayers[playerID] = true
			totalSalary += dp.getPlayerSalary(bestPlayer, config)
			totalScore += bestScore
		}
	}

	if len(selectedPlayers) == len(slots) {
		return &lineupCandidate{
			players:         selectedPlayers,
			totalSalary:     totalSalary,
			projectedPoints: totalScore,
			positions:       make(map[string][]types.Player),
			playerPositions: make(map[uuid.UUID]string),
		}
	}

	return nil
}

// getPlayerSalary returns appropriate salary based on platform
func (dp *DPOptimizer) getPlayerSalary(player types.Player, config OptimizeConfigV2) int {
	// Default to DraftKings, fallback to FanDuel
	if player.SalaryDK != nil && *player.SalaryDK > 0 {
		return *player.SalaryDK
	}
	if player.SalaryFD != nil && *player.SalaryFD > 0 {
		return *player.SalaryFD
	}
	return 0
}

// applyConstraints applies diversity and exposure constraints
func (dp *DPOptimizer) applyConstraints(lineups []*lineupCandidate, config OptimizeConfigV2) []*lineupCandidate {
	if len(lineups) == 0 {
		return lineups
	}

	// Sort by projected points
	sort.Slice(lineups, func(i, j int) bool {
		return lineups[i].projectedPoints > lineups[j].projectedPoints
	})

	// Apply basic diversity - can be enhanced
	finalLineups := make([]*lineupCandidate, 0, config.NumLineups)
	for i, lineup := range lineups {
		if i >= config.NumLineups {
			break
		}
		finalLineups = append(finalLineups, lineup)
	}

	return finalLineups
}

// convertToGeneratedLineups converts internal format to API format
func (dp *DPOptimizer) convertToGeneratedLineups(lineups []*lineupCandidate, config OptimizeConfigV2) []types.GeneratedLineup {
	result := make([]types.GeneratedLineup, len(lineups))

	for i, lineup := range lineups {
		// Convert Player slice to LineupPlayer slice
		lineupPlayers := make([]types.LineupPlayer, len(lineup.players))
		for j, player := range lineup.players {
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
				Salary:          dp.getPlayerSalary(player, config),
				ProjectedPoints: projectedPoints,
			}
		}

		result[i] = types.GeneratedLineup{
			ID:               fmt.Sprintf("lineup_%d_%s", i+1, uuid.New().String()[:8]),
			Players:          lineupPlayers,
			TotalSalary:      lineup.totalSalary,
			ProjectedPoints:  lineup.projectedPoints,
			Exposure:         0.0, // Will be calculated later
			StackDescription: "", // TODO: Add stack description logic
		}
	}

	return result
}

// clearMemoTable clears the memoization table
func (dp *DPOptimizer) clearMemoTable() {
	dp.mutex.Lock()
	defer dp.mutex.Unlock()

	dp.memoTable = make(map[string]*OptimizationState)
	dp.stats = DPStats{}
}

// logOptimizationStats logs optimization statistics
func (dp *DPOptimizer) logOptimizationStats() {
	dp.logger.WithFields(logrus.Fields{
		"states_cached":     dp.stats.StatesCached,
		"cache_hits":        dp.stats.CacheHits,
		"cache_misses":      dp.stats.CacheMisses,
		"cache_hit_rate":    float64(dp.stats.CacheHits) / float64(dp.stats.CacheHits + dp.stats.CacheMisses),
		"optimization_time": dp.stats.OptimizationTime,
		"memory_usage_mb":   float64(dp.stats.MemoryUsage) / (1024 * 1024),
	}).Info("Enhanced DP optimization statistics")
}

// GetStats returns current optimization statistics
func (dp *DPOptimizer) GetStats() DPStats {
	return dp.stats
}

// dpOptimize recursively optimizes using dynamic programming with memoization
func (dp *DPOptimizer) dpOptimize(players []types.Player, state *OptimizationState, depth int, platform string) (*OptimizationState, int) {
	// Base case: all positions filled
	if dp.isLineupComplete(state) {
		return state, 1
	}

	// Depth limit reached
	if depth >= dp.maxDepth {
		return state, 1
	}

	// Generate state hash for memoization
	stateHash := dp.generateStateHash(state)

	// Check memoization table
	if cachedState, exists := dp.memoTable[stateHash]; exists {
		dp.stats.CacheHits++
		return cachedState, 1
	}
	dp.stats.CacheMisses++

	var bestState *OptimizationState
	bestScore := -1.0
	totalStatesExplored := 1

	// Get next position to fill
	nextPosition := dp.getNextPosition(state)
	if nextPosition == "" {
		// No more positions to fill
		dp.memoTable[stateHash] = state
		return state, 1
	}

	// Try each eligible player for this position
	for _, player := range players {
		// Skip if player already used
		if dp.isPlayerUsed(player.ID, state.UsedPlayers) {
			continue
		}

		// Check position eligibility
		if !dp.isPlayerEligible(player, nextPosition) {
			continue
		}

		// Check salary constraint
		playerSalary := getSalaryForPlatform(player, platform)
		if playerSalary > state.Budget {
			continue
		}

		// Create new state with this player
		newState := dp.createNewState(state, player, playerSalary, platform)

		// Pruning: skip if this state can't possibly beat current best
		if dp.pruningEnabled && bestScore > 0 {
			maxPossibleScore := dp.estimateMaxPossibleScore(newState, players, platform)
			if maxPossibleScore < bestScore {
				continue
			}
		}

		// Recursive call
		resultState, statesExplored := dp.dpOptimize(players, newState, depth+1, platform)
		totalStatesExplored += statesExplored

		// Update best state if this is better
		if resultState != nil && resultState.CurrentScore > bestScore {
			bestScore = resultState.CurrentScore
			bestState = resultState
		}
	}

	// Store in memoization table
	if bestState != nil {
		dp.memoTable[stateHash] = bestState
	} else {
		dp.memoTable[stateHash] = state
	}

	return bestState, totalStatesExplored
}

// Helper functions

// getPositionSlots extracts position requirements from contest
func (dp *DPOptimizer) getPositionSlots(contest *types.Contest) map[string]int {
	if contest == nil {
		// Default DraftKings NBA lineup
		return map[string]int{
			"PG": 1, "SG": 1, "SF": 1, "PF": 1, "C": 1,
			"G": 1, "F": 1, "UTIL": 1,
		}
	}

	// Convert position requirements to map
	slots := make(map[string]int)

	// This would need to be adapted based on your PositionRequirements structure
	// For now, providing common DFS position structures
	switch contest.Platform {
	case "draftkings":
		switch strings.ToLower(contest.SportID.String()) {
		case "nba":
			slots = map[string]int{
				"PG": 1, "SG": 1, "SF": 1, "PF": 1, "C": 1,
				"G": 1, "F": 1, "UTIL": 1,
			}
		case "nfl":
			slots = map[string]int{
				"QB": 1, "RB": 2, "WR": 3, "TE": 1, "K": 1, "DST": 1,
			}
		}
	case "fanduel":
		switch strings.ToLower(contest.SportID.String()) {
		case "nba":
			slots = map[string]int{
				"PG": 2, "SG": 2, "SF": 2, "PF": 2, "C": 1,
			}
		case "nfl":
			slots = map[string]int{
				"QB": 1, "RB": 2, "WR": 3, "TE": 1, "K": 1, "DST": 1,
			}
		}
	}

	return slots
}

// isLineupComplete checks if all positions are filled
func (dp *DPOptimizer) isLineupComplete(state *OptimizationState) bool {
	for _, remaining := range state.Positions {
		if remaining > 0 {
			return false
		}
	}
	return true
}

// getNextPosition returns the next position that needs to be filled
func (dp *DPOptimizer) getNextPosition(state *OptimizationState) string {
	// Fill positions in priority order (core positions first)
	positionPriority := []string{"QB", "PG", "RB", "SG", "WR", "SF", "TE", "PF", "C", "K", "DST", "G", "F", "UTIL"}

	for _, position := range positionPriority {
		if remaining, exists := state.Positions[position]; exists && remaining > 0 {
			return position
		}
	}

	// If no priority position found, return any remaining position
	for position, remaining := range state.Positions {
		if remaining > 0 {
			return position
		}
	}

	return ""
}

// isPlayerUsed checks if a player is already in the lineup
func (dp *DPOptimizer) isPlayerUsed(playerID uuid.UUID, usedPlayers []uuid.UUID) bool {
	for _, usedID := range usedPlayers {
		if usedID == playerID {
			return true
		}
	}
	return false
}

// isPlayerEligible checks if a player can fill a position slot
func (dp *DPOptimizer) isPlayerEligible(player types.Player, position string) bool {
	playerPos := ""
	if player.Position != nil {
		playerPos = *player.Position
	}

	// Direct position match
	if playerPos == position {
		return true
	}

	// Handle flex positions
	switch position {
	case "G": // Guard (PG or SG)
		return playerPos == "PG" || playerPos == "SG"
	case "F": // Forward (SF or PF)
		return playerPos == "SF" || playerPos == "PF"
	case "UTIL": // Utility (any position)
		return true
	case "FLEX": // Flex (RB, WR, TE for NFL)
		return playerPos == "RB" || playerPos == "WR" || playerPos == "TE"
	}

	return false
}


// createNewState creates a new optimization state with the added player
func (dp *DPOptimizer) createNewState(state *OptimizationState, player types.Player, salary int, platform string) *OptimizationState {
	// Deep copy positions map
	newPositions := make(map[string]int)
	for pos, count := range state.Positions {
		newPositions[pos] = count
	}

	// Find which position this player fills
	filledPosition := dp.getNextPosition(state)
	if filledPosition != "" {
		newPositions[filledPosition]--
	}

	// Copy used players and add new one
	newUsedPlayers := make([]uuid.UUID, len(state.UsedPlayers)+1)
	copy(newUsedPlayers, state.UsedPlayers)
	newUsedPlayers[len(state.UsedPlayers)] = player.ID

	// Calculate score improvement
	scoreImprovement := 0.0
	if player.ProjectedPoints != nil {
		scoreImprovement = *player.ProjectedPoints
	}

	// Add correlation bonus if correlations enabled
	correlationBonus := 0.0
	if dp.correlations != nil {
		correlationBonus = dp.calculateCorrelationBonus(player, state.UsedPlayers)
	}

	// Create new state
	newState := &OptimizationState{
		Budget:           state.Budget - salary,
		Positions:        newPositions,
		UsedPlayers:      newUsedPlayers,
		CurrentScore:     state.CurrentScore + scoreImprovement + correlationBonus,
		CorrelationBonus: state.CorrelationBonus + correlationBonus,
	}

	return newState
}

// generateStateHash creates a hash for memoization
func (dp *DPOptimizer) generateStateHash(state *OptimizationState) string {
	// Create hash based on budget and remaining positions
	hashParts := []string{
		fmt.Sprintf("budget:%d", state.Budget),
	}

	// Add position requirements
	positions := make([]string, 0, len(state.Positions))
	for pos, count := range state.Positions {
		positions = append(positions, fmt.Sprintf("%s:%d", pos, count))
	}
	sort.Strings(positions) // Ensure consistent ordering
	hashParts = append(hashParts, strings.Join(positions, ","))

	// Add used players count (to avoid same players in different order)
	hashParts = append(hashParts, fmt.Sprintf("players:%d", len(state.UsedPlayers)))

	return strings.Join(hashParts, "|")
}

// estimateMaxPossibleScore estimates the maximum possible score from current state
func (dp *DPOptimizer) estimateMaxPossibleScore(state *OptimizationState, players []types.Player, platform string) float64 {
	maxScore := state.CurrentScore
	remainingBudget := state.Budget

	// For each remaining position, find the best possible player
	for position, count := range state.Positions {
		if count <= 0 {
			continue
		}

		bestValueForPosition := 0.0
		for _, player := range players {
			// Skip if already used
			if dp.isPlayerUsed(player.ID, state.UsedPlayers) {
				continue
			}

			// Skip if not eligible
			if !dp.isPlayerEligible(player, position) {
				continue
			}

			// Skip if too expensive
			salary := getSalaryForPlatform(player, platform)
			if salary > remainingBudget {
				continue
			}

			// Calculate value per dollar
			value := 0.0
			if player.ProjectedPoints != nil {
				value = *player.ProjectedPoints / float64(salary)
			}
			if value > bestValueForPosition {
				bestValueForPosition = value
			}
		}

		// Estimate points for this position
		estimatedBudgetPerPosition := remainingBudget / dp.getTotalRemainingSlots(state)
		estimatedPoints := bestValueForPosition * float64(estimatedBudgetPerPosition)
		maxScore += estimatedPoints * float64(count)
	}

	return maxScore
}

// getTotalRemainingSlots counts total remaining position slots
func (dp *DPOptimizer) getTotalRemainingSlots(state *OptimizationState) int {
	total := 0
	for _, count := range state.Positions {
		total += count
	}
	return total
}

// calculateCorrelationBonus calculates correlation bonus for adding a player
func (dp *DPOptimizer) calculateCorrelationBonus(player types.Player, usedPlayers []uuid.UUID) float64 {
	if dp.correlations == nil {
		return 0.0
	}

	bonus := 0.0
	for _, usedPlayerID := range usedPlayers {
		// This would use the actual correlation matrix
		// For now, providing a simple team-based correlation
		correlation := dp.getSimpleCorrelation(player, usedPlayerID)
		if player.ProjectedPoints != nil {
			bonus += correlation * (*player.ProjectedPoints) * 0.1 // 10% correlation weight
		}
	}

	return bonus
}

// getSimpleCorrelation provides basic correlation calculation
func (dp *DPOptimizer) getSimpleCorrelation(player types.Player, otherPlayerID uuid.UUID) float64 {
	// Placeholder for correlation calculation
	// In production, this would look up the actual correlation matrix

	// Same team correlation
	team := ""
	if player.Team != nil {
		team = *player.Team
	}
	if team != "" {
		return 0.2 // 20% positive correlation for teammates
	}

	// Opponent correlation (game stack)
	opponent := ""
	if player.Opponent != nil {
		opponent = *player.Opponent
	}
	if opponent != "" {
		return 0.1 // 10% positive correlation for game stack
	}

	return 0.0
}

// GetCacheStats returns optimization cache statistics
func (dp *DPOptimizer) GetCacheStats() map[string]interface{} {
	totalAccesses := dp.stats.CacheHits + dp.stats.CacheMisses
	hitRate := 0.0
	if totalAccesses > 0 {
		hitRate = float64(dp.stats.CacheHits) / float64(totalAccesses)
	}

	return map[string]interface{}{
		"memo_table_size": len(dp.memoTable),
		"cache_hits":      dp.stats.CacheHits,
		"cache_misses":    dp.stats.CacheMisses,
		"hit_rate":        hitRate,
	}
}

// ClearCache clears the memoization table
func (dp *DPOptimizer) ClearCache() {
	dp.memoTable = make(map[string]*OptimizationState)
	dp.stats.CacheHits = 0
	dp.stats.CacheMisses = 0
}

// SetPruning enables or disables pruning
func (dp *DPOptimizer) SetPruning(enabled bool) {
	dp.pruningEnabled = enabled
}

// SetMaxDepth sets the maximum recursion depth
func (dp *DPOptimizer) SetMaxDepth(depth int) {
	dp.maxDepth = depth
}

// Implement BaseOptimizerInterface
func (dp *DPOptimizer) Optimize(ctx context.Context, request *types.OptimizationRequest) (*types.OptimizationResult, error) {
    // TODO: Implement by calling appropriate optimization method
    return &types.OptimizationResult{
        Lineups: []types.GeneratedLineup{},
    }, nil
}
