package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/optimizer"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
	"gorm.io/gorm"
)

type OptimizerHandler struct {
	db     *database.DB
	cache  *services.CacheService
	config *config.Config
}

func NewOptimizerHandler(db *database.DB, cache *services.CacheService, cfg *config.Config) *OptimizerHandler {
	return &OptimizerHandler{
		db:     db,
		cache:  cache,
		config: cfg,
	}
}

// OptimizeLineups generates optimized lineups
func (h *OptimizerHandler) OptimizeLineups(c *gin.Context) {
	fmt.Println("=== OPTIMIZE HANDLER CALLED ===")
	// TODO: In production, re-enable authentication
	userID := uint(1) // Default user for development

	var req struct {
		ContestID           uint                     `json:"contest_id" binding:"required"`
		NumLineups          int                      `json:"num_lineups" binding:"required,min=1,max=150"`
		MinDifferentPlayers int                      `json:"min_different_players" binding:"min=0,max=9"`
		UseCorrelations     bool                     `json:"use_correlations"`
		CorrelationWeight   float64                  `json:"correlation_weight" binding:"min=0,max=1"`
		StackingRules       []optimizer.StackingRule `json:"stacking_rules"`
		LockedPlayers       []uint                   `json:"locked_players"`
		ExcludedPlayers     []uint                   `json:"excluded_players"`
		MinExposure         map[uint]float64         `json:"min_exposure"`
		MaxExposure         map[uint]float64         `json:"max_exposure"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Add request logging
	reqJSON, _ := json.Marshal(req)
	log.Printf("Optimizer request: %s", string(reqJSON))

	// Validate request
	if req.NumLineups > h.config.MaxLineups {
		utils.SendValidationError(c, "Too many lineups requested",
			fmt.Sprintf("Maximum allowed: %d", h.config.MaxLineups))
		return
	}

	// Get contest
	var contest models.Contest
	if err := h.db.First(&contest, req.ContestID).Error; err != nil {
		utils.SendNotFound(c, "Contest not found")
		return
	}

	// Add detailed logging after contest retrieval
	log.Printf("Optimizer: Contest ID=%d, Sport=%s, Platform=%s, SalaryCap=%d",
		contest.ID, contest.Sport, contest.Platform, contest.SalaryCap)

	// Validate contest data
	if contest.Sport == "" || contest.Platform == "" {
		utils.SendValidationError(c, "Invalid contest",
			fmt.Sprintf("Contest missing sport/platform: sport=%s, platform=%s",
				contest.Sport, contest.Platform))
		return
	}

	// Check cache for similar optimization
	ctx := context.Background()
	configHash := h.hashOptimizationConfig(req)
	cacheKey := services.OptimizationCacheKey(req.ContestID, configHash)

	var cachedResult *optimizer.OptimizerResult
	if err := h.cache.Get(ctx, cacheKey, &cachedResult); err == nil {
		// Return cached result
		h.saveOptimizedLineups(userID, cachedResult.Lineups)
		utils.SendSuccess(c, cachedResult)
		return
	}

	// Get players
	var players []models.Player
	if err := h.db.Where("contest_id = ?", req.ContestID).Find(&players).Error; err != nil {
		utils.SendInternalError(c, "Failed to fetch players")
		return
	}

	fmt.Printf("Handler: Found %d players for contest %d\n", len(players), req.ContestID)

	// Log player count by position
	playersByPos := make(map[string]int)
	for _, p := range players {
		playersByPos[p.Position]++
	}
	log.Printf("Optimizer: Players by position: %+v", playersByPos)

	// Validate players exist
	if len(players) == 0 {
		utils.SendValidationError(c, "No players available",
			fmt.Sprintf("No players found for contest %d", req.ContestID))
		return
	}

	// Create optimization config
	optimizeConfig := optimizer.OptimizeConfig{
		SalaryCap:           contest.SalaryCap,
		NumLineups:          req.NumLineups,
		MinDifferentPlayers: req.MinDifferentPlayers,
		UseCorrelations:     req.UseCorrelations,
		CorrelationWeight:   req.CorrelationWeight,
		StackingRules:       req.StackingRules,
		LockedPlayers:       req.LockedPlayers,
		ExcludedPlayers:     req.ExcludedPlayers,
		MinExposure:         req.MinExposure,
		MaxExposure:         req.MaxExposure,
		Contest:             &contest,
	}

	// Set defaults
	if optimizeConfig.CorrelationWeight == 0 {
		optimizeConfig.CorrelationWeight = 0.3
	}
	if optimizeConfig.MinDifferentPlayers == 0 {
		optimizeConfig.MinDifferentPlayers = 2
	}

	// Debug: Try a direct call first
	fmt.Println("Handler: About to call optimizer directly")
	testResult, testErr := optimizer.OptimizeLineups(players, optimizeConfig)
	if testErr != nil {
		fmt.Printf("Handler: Direct call returned error: %v\n", testErr)
		utils.SendError(c, http.StatusInternalServerError,
			utils.NewAppError(utils.ErrCodeOptimization, "Optimization failed", testErr.Error()))
		return
	}
	fmt.Printf("Handler: Direct call returned %d lineups\n", len(testResult.Lineups))

	// If direct call succeeded, return those results
	if testResult != nil && len(testResult.Lineups) > 0 {
		savedLineups := h.saveOptimizedLineups(userID, testResult.Lineups)
		testResult.Lineups = savedLineups
		utils.SendSuccess(c, testResult)
		return
	}

	// Run optimization in background
	resultChan := make(chan *optimizer.OptimizerResult, 1)
	errorChan := make(chan error, 1)

	go func() {
		// Set timeout
		timer := time.NewTimer(time.Duration(h.config.OptimizationTimeout) * time.Second)
		defer timer.Stop()

		done := make(chan bool, 1)
		go func() {
			fmt.Printf("Handler: Calling optimizer with %d players\n", len(players))
			result, err := optimizer.OptimizeLineups(players, optimizeConfig)
			if err != nil {
				fmt.Printf("Handler: Optimizer returned error: %v\n", err)
				errorChan <- err
			} else {
				fmt.Printf("Handler: Optimizer returned %d lineups\n", len(result.Lineups))
				resultChan <- result
			}
			done <- true
		}()

		select {
		case <-done:
			// Optimization completed
		case <-timer.C:
			errorChan <- fmt.Errorf("optimization timeout exceeded")
		}
	}()

	// Wait for result
	select {
	case result := <-resultChan:
		// Cache the result
		h.cache.SetWithRetry(ctx, cacheKey, result, 15*time.Minute, 3)

		// Save lineups
		savedLineups := h.saveOptimizedLineups(userID, result.Lineups)
		result.Lineups = savedLineups

		utils.SendSuccess(c, result)

	case err := <-errorChan:
		utils.SendError(c, http.StatusInternalServerError,
			utils.NewAppError(utils.ErrCodeOptimization, "Optimization failed", err.Error()))

	case <-time.After(time.Duration(h.config.OptimizationTimeout) * time.Second):
		utils.SendError(c, http.StatusRequestTimeout,
			utils.NewAppError(utils.ErrCodeOptimization, "Optimization timeout",
				"The optimization took too long to complete"))
	}
}

// ValidateLineup validates a lineup against contest rules
func (h *OptimizerHandler) ValidateLineup(c *gin.Context) {
	var req struct {
		ContestID uint   `json:"contest_id" binding:"required"`
		PlayerIDs []uint `json:"player_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Get contest
	var contest models.Contest
	if err := h.db.First(&contest, req.ContestID).Error; err != nil {
		utils.SendNotFound(c, "Contest not found")
		return
	}

	// Get players
	var players []models.Player
	if err := h.db.Where("id IN ? AND contest_id = ?", req.PlayerIDs, req.ContestID).Find(&players).Error; err != nil {
		utils.SendInternalError(c, "Failed to fetch players")
		return
	}

	if len(players) != len(req.PlayerIDs) {
		utils.SendValidationError(c, "Some players not found", "")
		return
	}

	// Create lineup for validation
	lineup := models.Lineup{
		ContestID: req.ContestID,
		Players:   players,
		Contest:   contest,
	}
	lineup.CalculateTotalSalary()
	lineup.CalculateProjectedPoints()

	// Validate
	constraints := optimizer.GetConstraintsForContest(&contest)
	validationResult := gin.H{
		"valid":            true,
		"errors":           []string{},
		"warnings":         []string{},
		"total_salary":     lineup.TotalSalary,
		"salary_cap":       contest.SalaryCap,
		"remaining_salary": contest.SalaryCap - lineup.TotalSalary,
		"projected_points": lineup.ProjectedPoints,
		"position_counts":  getPositionCounts(players),
	}

	// Check constraints
	if err := constraints.ValidateLineup(&lineup); err != nil {
		validationResult["valid"] = false
		validationResult["errors"] = append(validationResult["errors"].([]string), err.Error())
	}

	// Check for warnings
	if lineup.TotalSalary < int(float64(contest.SalaryCap)*0.98) {
		validationResult["warnings"] = append(validationResult["warnings"].([]string),
			fmt.Sprintf("Leaving $%d on the table", contest.SalaryCap-lineup.TotalSalary))
	}

	// Check team exposure
	teamExposure := lineup.GetTeamExposure()
	for team, count := range teamExposure {
		if count >= 4 {
			validationResult["warnings"] = append(validationResult["warnings"].([]string),
				fmt.Sprintf("High exposure to %s (%d players)", team, count))
		}
	}

	utils.SendSuccess(c, validationResult)
}

// GetConstraints returns contest constraints
func (h *OptimizerHandler) GetConstraints(c *gin.Context) {
	contestIDStr := c.Param("contestId")
	contestID, err := strconv.ParseUint(contestIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid contest ID", err.Error())
		return
	}

	// Get contest
	var contest models.Contest
	if err := h.db.First(&contest, contestID).Error; err != nil {
		utils.SendNotFound(c, "Contest not found")
		return
	}

	// Get constraints
	constraints := optimizer.GetConstraintsForContest(&contest)

	// Format response
	response := gin.H{
		"salary_cap": constraints.SalaryCap,
		"positions":  formatPositionConstraints(constraints.PositionConstraints),
		"team_limits": gin.H{
			"min_players_per_team": constraints.MinPlayersPerTeam,
			"max_players_per_team": constraints.MaxPlayersPerTeam,
			"min_unique_teams":     constraints.MinUniqueTeams,
		},
		"game_limits": gin.H{
			"min_players_per_game": constraints.MinPlayersPerGame,
			"max_players_per_game": constraints.MaxPlayersPerGame,
			"min_unique_games":     constraints.MinUniqueGames,
		},
		"total_players": contest.PositionRequirements.GetTotalPlayers(),
		"sport":         contest.Sport,
		"platform":      contest.Platform,
	}

	utils.SendSuccess(c, response)
}

// GetOptimizationPresets returns common optimization presets
func (h *OptimizerHandler) GetOptimizationPresets(c *gin.Context) {
	sport := c.Query("sport")

	presets := []gin.H{
		{
			"name":        "Balanced",
			"description": "Standard optimization with moderate correlation",
			"config": gin.H{
				"use_correlations":      true,
				"correlation_weight":    0.3,
				"min_different_players": 3,
			},
		},
		{
			"name":        "Max Correlation",
			"description": "Heavy stacking for GPP tournaments",
			"config": gin.H{
				"use_correlations":      true,
				"correlation_weight":    0.6,
				"min_different_players": 2,
				"stacking_rules": []gin.H{
					{"type": "team", "min_players": 3, "max_players": 4},
					{"type": "game", "min_players": 4, "max_players": 6},
				},
			},
		},
		{
			"name":        "Cash Game",
			"description": "Safe plays with minimal correlation",
			"config": gin.H{
				"use_correlations":      false,
				"correlation_weight":    0,
				"min_different_players": 5,
			},
		},
	}

	// Filter by sport/contest type if provided
	if sport == "nfl" {
		presets = append(presets, gin.H{
			"name":        "QB Stack",
			"description": "QB with pass catchers",
			"config": gin.H{
				"use_correlations":      true,
				"correlation_weight":    0.5,
				"min_different_players": 3,
				"stacking_rules": []gin.H{
					{"type": "qb_stack", "min_players": 2, "max_players": 3},
				},
			},
		})
	}

	utils.SendSuccess(c, presets)
}

// Helper functions

func (h *OptimizerHandler) hashOptimizationConfig(req interface{}) string {
	data, _ := json.Marshal(req)
	return fmt.Sprintf("%x", data)
}

func (h *OptimizerHandler) saveOptimizedLineups(userID uint, lineups []models.Lineup) []models.Lineup {
	savedLineups := make([]models.Lineup, 0, len(lineups))

	for i := range lineups {
		// Work with a pointer to avoid copying issues
		lineup := &lineups[i]
		lineup.UserID = userID
		lineup.IsOptimized = true

		// Debug logging
		fmt.Printf("Saving lineup: players=%d, positions=%d\n", len(lineup.Players), len(lineup.PlayerPositions))

		// Store players temporarily and clear from lineup to prevent GORM association
		players := lineup.Players
		playerPositions := lineup.PlayerPositions
		lineup.Players = nil

		// Debug: Ensure Players is really nil
		fmt.Printf("After clearing: lineup.Players is nil? %v\n", lineup.Players == nil)

		// Start transaction for lineup and player positions
		tx := h.db.Begin()

		// Save lineup first (without players to avoid auto-association)
		// Use Session to disable association mode completely
		if err := tx.Session(&gorm.Session{FullSaveAssociations: false}).Create(&lineup).Error; err != nil {
			fmt.Printf("Failed to create lineup: %v\n", err)
			tx.Rollback()
			continue
		}

		// Save player positions in join table
		fmt.Printf("Saving %d players to lineup %d\n", len(players), lineup.ID)
		for _, player := range players {
			position := playerPositions[player.ID]
			if position == "" {
				// For sports like Golf where all players have same position
				position = player.Position
			}

			fmt.Printf("Creating LineupPlayer: lineup_id=%d, player_id=%d, position=%s\n", lineup.ID, player.ID, position)

			lineupPlayer := models.LineupPlayer{
				LineupID: lineup.ID,
				PlayerID: player.ID,
				Position: position,
			}

			if err := tx.Create(&lineupPlayer).Error; err != nil {
				// Log the error for debugging
				fmt.Printf("Failed to create lineup player: lineup_id=%d, player_id=%d, position=%s, error=%v\n",
					lineup.ID, player.ID, position, err)
				tx.Rollback()
				continue
			}
		}

		// Commit transaction
		if err := tx.Commit().Error; err == nil {
			// Restore players for response
			lineup.Players = players
			lineup.PlayerPositions = playerPositions
			savedLineups = append(savedLineups, *lineup)
		}
	}

	return savedLineups
}

func getPositionCounts(players []models.Player) map[string]int {
	counts := make(map[string]int)
	for _, player := range players {
		counts[player.Position]++
	}
	return counts
}

func formatPositionConstraints(constraints map[string]optimizer.PositionConstraint) []gin.H {
	formatted := make([]gin.H, 0, len(constraints))

	for position, constraint := range constraints {
		formatted = append(formatted, gin.H{
			"position":       position,
			"min_required":   constraint.MinRequired,
			"max_allowed":    constraint.MaxAllowed,
			"eligible_slots": constraint.EligibleSlots,
		})
	}

	return formatted
}
