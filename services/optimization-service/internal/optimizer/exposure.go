package optimizer

import (
	"fmt"
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/stitts-dev/dfs-sim/shared/types"
	"github.com/sirupsen/logrus"
)

// ExposureManager handles portfolio-level exposure constraints
type ExposureManager struct {
	config         ExposureConfig
	playerCount    map[uuid.UUID]int            // Player ID -> count in lineups
	teamCount      map[string]int               // Team -> count across all lineups
	stackCount     map[string]int               // Stack identifier -> count
	totalLineups   int
	maxExposures   map[uuid.UUID]float64        // Player-specific max exposures
	minExposures   map[uuid.UUID]float64        // Player-specific min exposures
	teamExposures  map[string]float64           // Team-specific exposures
	diversityMats  [][]uuid.UUID                // Matrix of player diversity for each lineup
}

// StackExposure represents a stacking combination exposure
type StackExposure struct {
	StackID     string      `json:"stack_id"`
	PlayerIDs   []uuid.UUID `json:"player_ids"`
	Teams       []string    `json:"teams"`
	Count       int         `json:"count"`
	Percentage  float64     `json:"percentage"`
	StackType   string      `json:"stack_type"` // "team", "game", "mini"
}

// ExposureReport provides detailed exposure analysis
type ExposureReport struct {
	PlayerExposures []PlayerExposure `json:"player_exposures"`
	TeamExposures   []TeamExposure   `json:"team_exposures"`
	StackExposures  []StackExposure  `json:"stack_exposures"`
	DiversityScore  float64          `json:"diversity_score"`
	TotalLineups    int              `json:"total_lineups"`
	Violations      []string         `json:"violations"`
}

// PlayerExposure represents exposure for a single player
type PlayerExposure struct {
	PlayerID   uuid.UUID `json:"player_id"`
	PlayerName string    `json:"player_name"`
	Count      int       `json:"count"`
	Percentage float64   `json:"percentage"`
	MaxAllowed float64   `json:"max_allowed"`
	MinRequired float64  `json:"min_required"`
	IsViolation bool     `json:"is_violation"`
}

// TeamExposure represents exposure for a team
type TeamExposure struct {
	Team       string  `json:"team"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
	MaxAllowed float64 `json:"max_allowed"`
	IsViolation bool   `json:"is_violation"`
}

// NewExposureManager creates a new exposure manager
func NewExposureManager(config ExposureConfig) *ExposureManager {
	return &ExposureManager{
		config:        config,
		playerCount:   make(map[uuid.UUID]int),
		teamCount:     make(map[string]int),
		stackCount:    make(map[string]int),
		maxExposures:  make(map[uuid.UUID]float64),
		minExposures:  make(map[uuid.UUID]float64),
		teamExposures: make(map[string]float64),
		diversityMats: make([][]uuid.UUID, 0),
	}
}

// CanAddPlayer checks if a player can be added without violating exposure constraints
func (em *ExposureManager) CanAddPlayer(playerID uuid.UUID, team string, lineupIndex int) bool {
	// Check player exposure limit
	maxExposure := em.getMaxPlayerExposure(playerID)
	currentCount := em.playerCount[playerID]
	
	// If this would be the first lineup, always allow
	if em.totalLineups == 0 {
		return true
	}
	
	// Check if adding would violate exposure
	newExposure := float64(currentCount+1) / float64(em.totalLineups+1) * 100
	if newExposure > maxExposure {
		logrus.Debugf("Player %s would exceed exposure: %.1f%% > %.1f%%", 
			playerID.String(), newExposure, maxExposure)
		return false
	}
	
	// Check team exposure limit
	maxTeamExposure := em.getMaxTeamExposure(team)
	currentTeamCount := em.teamCount[team]
	newTeamExposure := float64(currentTeamCount+1) / float64(em.totalLineups+1) * 100
	
	if newTeamExposure > maxTeamExposure {
		logrus.Debugf("Team %s would exceed exposure: %.1f%% > %.1f%%", 
			team, newTeamExposure, maxTeamExposure)
		return false
	}
	
	// Check diversity constraint
	if !em.checkDiversityConstraint(playerID, lineupIndex) {
		logrus.Debugf("Player %s would violate diversity constraint", playerID.String())
		return false
	}
	
	return true
}

// AddPlayerToLineup adds a player to exposure tracking
func (em *ExposureManager) AddPlayerToLineup(playerID uuid.UUID, team string, lineupIndex int) {
	em.playerCount[playerID]++
	em.teamCount[team]++
	
	// Extend diversity matrix if needed
	for len(em.diversityMats) <= lineupIndex {
		em.diversityMats = append(em.diversityMats, make([]uuid.UUID, 0))
	}
	
	// Add player to lineup diversity tracking
	em.diversityMats[lineupIndex] = append(em.diversityMats[lineupIndex], playerID)
}

// CompleteLineup marks a lineup as complete for exposure calculations
func (em *ExposureManager) CompleteLineup() {
	em.totalLineups++
}

// CheckMinExposures checks if minimum exposure requirements are met
func (em *ExposureManager) CheckMinExposures(players []types.Player) []string {
	violations := make([]string, 0)
	
	for playerID, minExp := range em.minExposures {
		currentCount := em.playerCount[playerID]
		currentExposure := float64(currentCount) / float64(em.totalLineups) * 100
		
		if currentExposure < minExp {
			playerName := em.getPlayerName(playerID, players)
			violations = append(violations, 
				fmt.Sprintf("Player %s has %.1f%% exposure, requires %.1f%%", 
					playerName, currentExposure, minExp))
		}
	}
	
	return violations
}

// GenerateExposureReport creates a comprehensive exposure report
func (em *ExposureManager) GenerateExposureReport(players []types.Player) *ExposureReport {
	report := &ExposureReport{
		PlayerExposures: make([]PlayerExposure, 0),
		TeamExposures:   make([]TeamExposure, 0),
		StackExposures:  make([]StackExposure, 0),
		TotalLineups:    em.totalLineups,
		Violations:      make([]string, 0),
	}
	
	// Generate player exposures
	for playerID, count := range em.playerCount {
		if count == 0 {
			continue
		}
		
		percentage := float64(count) / float64(em.totalLineups) * 100
		maxAllowed := em.getMaxPlayerExposure(playerID)
		minRequired := em.getMinPlayerExposure(playerID)
		
		isViolation := percentage > maxAllowed || percentage < minRequired
		if isViolation {
			violation := fmt.Sprintf("Player %s: %.1f%% (min: %.1f%%, max: %.1f%%)",
				em.getPlayerName(playerID, players), percentage, minRequired, maxAllowed)
			report.Violations = append(report.Violations, violation)
		}
		
		report.PlayerExposures = append(report.PlayerExposures, PlayerExposure{
			PlayerID:    playerID,
			PlayerName:  em.getPlayerName(playerID, players),
			Count:       count,
			Percentage:  percentage,
			MaxAllowed:  maxAllowed,
			MinRequired: minRequired,
			IsViolation: isViolation,
		})
	}
	
	// Generate team exposures
	for team, count := range em.teamCount {
		if count == 0 {
			continue
		}
		
		percentage := float64(count) / float64(em.totalLineups) * 100
		maxAllowed := em.getMaxTeamExposure(team)
		
		isViolation := percentage > maxAllowed
		if isViolation {
			violation := fmt.Sprintf("Team %s: %.1f%% > %.1f%%", team, percentage, maxAllowed)
			report.Violations = append(report.Violations, violation)
		}
		
		report.TeamExposures = append(report.TeamExposures, TeamExposure{
			Team:        team,
			Count:       count,
			Percentage:  percentage,
			MaxAllowed:  maxAllowed,
			IsViolation: isViolation,
		})
	}
	
	// Calculate diversity score
	report.DiversityScore = em.calculateDiversityScore()
	
	// Sort by exposure percentage
	sort.Slice(report.PlayerExposures, func(i, j int) bool {
		return report.PlayerExposures[i].Percentage > report.PlayerExposures[j].Percentage
	})
	
	sort.Slice(report.TeamExposures, func(i, j int) bool {
		return report.TeamExposures[i].Percentage > report.TeamExposures[j].Percentage
	})
	
	logrus.Infof("Generated exposure report: %d players, %d teams, %.2f diversity score, %d violations",
		len(report.PlayerExposures), len(report.TeamExposures), report.DiversityScore, len(report.Violations))
	
	return report
}

// SetPlayerExposureLimit sets custom exposure limit for a specific player
func (em *ExposureManager) SetPlayerExposureLimit(playerID uuid.UUID, maxExposure, minExposure float64) {
	if maxExposure >= 0 && maxExposure <= 100 {
		em.maxExposures[playerID] = maxExposure
	}
	if minExposure >= 0 && minExposure <= 100 {
		em.minExposures[playerID] = minExposure
	}
}

// SetTeamExposureLimit sets custom exposure limit for a specific team
func (em *ExposureManager) SetTeamExposureLimit(team string, maxExposure float64) {
	if maxExposure >= 0 && maxExposure <= 100 {
		em.teamExposures[team] = maxExposure
	}
}

// OptimizeExposures balances exposures across lineups by reordering/replacing players
func (em *ExposureManager) OptimizeExposures(lineups [][]types.Player, targetPlayers []types.Player) [][]types.Player {
	if len(lineups) == 0 {
		return lineups
	}
	
	// Analyze current exposures
	em.analyzeExistingLineups(lineups)
	
	optimizedLineups := make([][]types.Player, len(lineups))
	copy(optimizedLineups, lineups)
	
	// Identify over-exposed and under-exposed players
	overExposed, underExposed := em.identifyExposureImbalances(targetPlayers)
	
	// Attempt to rebalance
	for i := 0; i < len(optimizedLineups) && len(overExposed) > 0 && len(underExposed) > 0; i++ {
		optimizedLineups[i] = em.rebalanceLineup(optimizedLineups[i], overExposed, underExposed)
	}
	
	return optimizedLineups
}

// Helper functions

// getMaxPlayerExposure returns the maximum exposure allowed for a player
func (em *ExposureManager) getMaxPlayerExposure(playerID uuid.UUID) float64 {
	if exposure, exists := em.maxExposures[playerID]; exists {
		return exposure
	}
	return em.config.MaxPlayerExposure
}

// getMinPlayerExposure returns the minimum exposure required for a player
func (em *ExposureManager) getMinPlayerExposure(playerID uuid.UUID) float64 {
	if exposure, exists := em.minExposures[playerID]; exists {
		return exposure
	}
	return 0.0 // Default no minimum
}

// getMaxTeamExposure returns the maximum exposure allowed for a team
func (em *ExposureManager) getMaxTeamExposure(team string) float64 {
	if exposure, exists := em.teamExposures[team]; exists {
		return exposure
	}
	return em.config.MaxTeamExposure
}

// checkDiversityConstraint ensures minimum different players between lineups
func (em *ExposureManager) checkDiversityConstraint(playerID uuid.UUID, lineupIndex int) bool {
	if lineupIndex == 0 || em.config.MinDifferentPlayers <= 0 {
		return true // First lineup or no diversity requirement
	}
	
	// Check against previous lineups
	for i := 0; i < lineupIndex && i < len(em.diversityMats); i++ {
		sharedPlayers := em.countSharedPlayers(em.diversityMats[i], playerID)
		requiredDifferent := em.config.MinDifferentPlayers
		
		// If we're still building the current lineup, estimate overlap
		if lineupIndex < len(em.diversityMats) {
			currentLineupSize := len(em.diversityMats[lineupIndex])
			maxOverlap := 8 - requiredDifferent // Assuming 8-player lineups
			
			if sharedPlayers >= maxOverlap && currentLineupSize < 8 {
				return false
			}
		}
	}
	
	return true
}

// countSharedPlayers counts how many players are shared between lineups
func (em *ExposureManager) countSharedPlayers(lineup []uuid.UUID, additionalPlayer uuid.UUID) int {
	count := 0
	
	// Check if additional player is already in lineup
	for _, playerID := range lineup {
		if playerID == additionalPlayer {
			count++
			break
		}
	}
	
	return count
}

// getPlayerName returns player name for display purposes
func (em *ExposureManager) getPlayerName(playerID uuid.UUID, players []types.Player) string {
	for _, player := range players {
		if player.ID == playerID {
			return player.Name
		}
	}
	return playerID.String()[:8] // Fallback to short UUID
}

// calculateDiversityScore computes overall diversity score for the portfolio
func (em *ExposureManager) calculateDiversityScore() float64 {
	if len(em.diversityMats) < 2 {
		return 1.0 // Perfect diversity with single lineup
	}
	
	totalComparisons := 0
	totalDifferentPlayers := 0
	
	// Compare each pair of lineups
	for i := 0; i < len(em.diversityMats); i++ {
		for j := i + 1; j < len(em.diversityMats); j++ {
			totalComparisons++
			differentPlayers := em.countDifferentPlayers(em.diversityMats[i], em.diversityMats[j])
			totalDifferentPlayers += differentPlayers
		}
	}
	
	if totalComparisons == 0 {
		return 1.0
	}
	
	averageDifferentPlayers := float64(totalDifferentPlayers) / float64(totalComparisons)
	maxPossibleDifferent := 8.0 // Assuming 8-player lineups
	
	return averageDifferentPlayers / maxPossibleDifferent
}

// countDifferentPlayers counts different players between two lineups
func (em *ExposureManager) countDifferentPlayers(lineup1, lineup2 []uuid.UUID) int {
	playerSet := make(map[uuid.UUID]bool)
	
	// Add all players from lineup1
	for _, playerID := range lineup1 {
		playerSet[playerID] = true
	}
	
	different := 0
	// Count players in lineup2 not in lineup1
	for _, playerID := range lineup2 {
		if !playerSet[playerID] {
			different++
		}
	}
	
	// Add players in lineup1 not in lineup2
	playerSet2 := make(map[uuid.UUID]bool)
	for _, playerID := range lineup2 {
		playerSet2[playerID] = true
	}
	
	for _, playerID := range lineup1 {
		if !playerSet2[playerID] {
			different++
		}
	}
	
	return different
}

// analyzeExistingLineups analyzes current lineups to build exposure data
func (em *ExposureManager) analyzeExistingLineups(lineups [][]types.Player) {
	// Reset counters
	em.playerCount = make(map[uuid.UUID]int)
	em.teamCount = make(map[string]int)
	em.totalLineups = len(lineups)
	
	// Count exposures
	for _, lineup := range lineups {
		for _, player := range lineup {
			em.playerCount[player.ID]++
			team := ""
			if player.Team != nil {
				team = *player.Team
			}
			em.teamCount[team]++
		}
	}
}

// identifyExposureImbalances finds over and under-exposed players
func (em *ExposureManager) identifyExposureImbalances(players []types.Player) (overExposed, underExposed []types.Player) {
	for _, player := range players {
		count := em.playerCount[player.ID]
		exposure := float64(count) / float64(em.totalLineups) * 100
		maxExposure := em.getMaxPlayerExposure(player.ID)
		minExposure := em.getMinPlayerExposure(player.ID)
		
		if exposure > maxExposure {
			overExposed = append(overExposed, player)
		} else if exposure < minExposure {
			underExposed = append(underExposed, player)
		}
	}
	
	return overExposed, underExposed
}

// rebalanceLineup attempts to replace over-exposed players with under-exposed ones
func (em *ExposureManager) rebalanceLineup(lineup []types.Player, overExposed, underExposed []types.Player) []types.Player {
	newLineup := make([]types.Player, len(lineup))
	copy(newLineup, lineup)
	
	// Find over-exposed players in this lineup
	for i, player := range newLineup {
		for _, overExp := range overExposed {
			if player.ID == overExp.ID {
				// Try to replace with an under-exposed player
				replacement := em.findReplacementPlayer(player, underExposed)
				if replacement != nil {
					logrus.Debugf("Replacing over-exposed %s with under-exposed %s", 
						player.Name, replacement.Name)
					newLineup[i] = *replacement
					break
				}
			}
		}
	}
	
	return newLineup
}

// findReplacementPlayer finds a suitable under-exposed replacement
func (em *ExposureManager) findReplacementPlayer(original types.Player, underExposed []types.Player) *types.Player {
	// Find under-exposed players in same position with similar salary
	salaryRange := 1000 // Allow $1000 difference
	
	for _, candidate := range underExposed {
		candidatePosition := ""
		if candidate.Position != nil {
			candidatePosition = *candidate.Position
		}
		originalPosition := ""
		if original.Position != nil {
			originalPosition = *original.Position
		}
		if candidatePosition == originalPosition {
			candidateSalary := 0
			if candidate.SalaryDK != nil {
				candidateSalary = *candidate.SalaryDK
			}
			originalSalary := 0
			if original.SalaryDK != nil {
				originalSalary = *original.SalaryDK
			}
			salaryDiff := int(math.Abs(float64(candidateSalary - originalSalary)))
			if salaryDiff <= salaryRange {
				return &candidate
			}
		}
	}
	
	return nil
}

// GetExposureStats returns current exposure statistics
func (em *ExposureManager) GetExposureStats() map[string]interface{} {
	return map[string]interface{}{
		"total_lineups":    em.totalLineups,
		"unique_players":   len(em.playerCount),
		"unique_teams":     len(em.teamCount),
		"diversity_score":  em.calculateDiversityScore(),
		"avg_player_exp":   em.calculateAveragePlayerExposure(),
	}
}

// calculateAveragePlayerExposure calculates average player exposure
func (em *ExposureManager) calculateAveragePlayerExposure() float64 {
	if len(em.playerCount) == 0 || em.totalLineups == 0 {
		return 0.0
	}
	
	totalExposure := 0.0
	for _, count := range em.playerCount {
		exposure := float64(count) / float64(em.totalLineups) * 100
		totalExposure += exposure
	}
	
	return totalExposure / float64(len(em.playerCount))
}

// Reset clears all exposure tracking data
func (em *ExposureManager) Reset() {
	em.playerCount = make(map[uuid.UUID]int)
	em.teamCount = make(map[string]int)
	em.stackCount = make(map[string]int)
	em.totalLineups = 0
	em.diversityMats = make([][]uuid.UUID, 0)
}