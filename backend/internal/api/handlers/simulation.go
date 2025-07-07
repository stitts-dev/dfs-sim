package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/internal/simulator"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
	"gorm.io/gorm"
)

type SimulationHandler struct {
	db    *database.DB
	cache *services.CacheService
	wsHub *services.WebSocketHub
}

func NewSimulationHandler(db *database.DB, cache *services.CacheService, wsHub *services.WebSocketHub) *SimulationHandler {
	return &SimulationHandler{
		db:    db,
		cache: cache,
		wsHub: wsHub,
	}
}

// RunSimulation runs Monte Carlo simulation for a lineup
func (h *SimulationHandler) RunSimulation(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		LineupID        uint                `json:"lineup_id" binding:"required"`
		NumSimulations  int                 `json:"num_simulations" binding:"required,min=100,max=100000"`
		UseCorrelations bool                `json:"use_correlations"`
		ContestSize     int                 `json:"contest_size" binding:"required,min=2"`
		PayoutStructure []models.PayoutTier `json:"payout_structure"`
		EntryFee        float64             `json:"entry_fee" binding:"required,min=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Get lineup with players
	var lineup models.Lineup
	err := h.db.Preload("Contest").Preload("Players").
		Where("id = ? AND user_id = ?", req.LineupID, userID).
		First(&lineup).Error
	if err != nil {
		utils.SendNotFound(c, "Lineup not found")
		return
	}

	// Check cache for existing simulation
	ctx := context.Background()
	cacheKey := services.SimulationCacheKey(req.LineupID)

	var cachedResult models.SimulationResult
	if err := h.cache.Get(ctx, cacheKey, &cachedResult); err == nil {
		utils.SendSuccess(c, cachedResult)
		return
	}

	// Create simulation config
	config := &models.SimulationConfig{
		NumSimulations:  req.NumSimulations,
		UseCorrelations: req.UseCorrelations,
		ContestSize:     req.ContestSize,
		PayoutStructure: req.PayoutStructure,
		EntryFee:        req.EntryFee,
	}

	// If no payout structure provided, use defaults
	if len(config.PayoutStructure) == 0 {
		config.PayoutStructure = simulator.GetPayoutStructure(&lineup.Contest)
	}

	// Get all players for correlation calculations
	var allPlayers []models.Player
	if err := h.db.Where("contest_id = ?", lineup.ContestID).Find(&allPlayers).Error; err != nil {
		utils.SendInternalError(c, "Failed to fetch players")
		return
	}

	// Create simulator
	sim := simulator.NewSimulator(config, allPlayers)

	// Create progress channel
	progressChan := make(chan models.SimulationProgress, 100)
	defer close(progressChan)

	// Start progress reporter
	go h.reportProgress(userID.(uint), req.LineupID, progressChan)

	// Run simulation
	lineups := []models.Lineup{lineup}
	result, err := sim.SimulateContest(lineups, progressChan)
	if err != nil {
		utils.SendInternalError(c, "Simulation failed: "+err.Error())
		return
	}

	// Save result
	if err := h.db.Create(result).Error; err != nil {
		utils.SendInternalError(c, "Failed to save simulation results")
		return
	}

	// Cache result
	h.cache.SetWithRetry(ctx, cacheKey, *result, 30*time.Minute, 3)

	// Broadcast completion
	h.wsHub.BroadcastToUser(userID.(uint), "simulation_complete", gin.H{
		"lineup_id": req.LineupID,
		"result":    result,
	})

	utils.SendSuccess(c, result)
}

// GetSimulationResult retrieves simulation results
func (h *SimulationHandler) GetSimulationResult(c *gin.Context) {
	userID, _ := c.Get("user_id")
	lineupIDStr := c.Param("lineupId")
	lineupID, err := strconv.ParseUint(lineupIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid lineup ID", err.Error())
		return
	}

	// Verify lineup ownership
	var lineup models.Lineup
	err = h.db.Where("id = ? AND user_id = ?", lineupID, userID).First(&lineup).Error
	if err != nil {
		utils.SendNotFound(c, "Lineup not found")
		return
	}

	// Check cache first
	ctx := context.Background()
	cacheKey := services.SimulationCacheKey(uint(lineupID))

	var result models.SimulationResult
	if err := h.cache.Get(ctx, cacheKey, &result); err == nil {
		utils.SendSuccess(c, result)
		return
	}

	// Get from database
	err = h.db.Where("lineup_id = ?", lineupID).Order("created_at DESC").First(&result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.SendNotFound(c, "No simulation results found")
		} else {
			utils.SendInternalError(c, "Failed to fetch simulation results")
		}
		return
	}

	// Cache for future requests
	h.cache.Set(ctx, cacheKey, result, 30*time.Minute)

	utils.SendSuccess(c, result)
}

// BatchSimulate runs simulations for multiple lineups
func (h *SimulationHandler) BatchSimulate(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		LineupIDs       []uint              `json:"lineup_ids" binding:"required,min=1,max=20"`
		NumSimulations  int                 `json:"num_simulations" binding:"required,min=100,max=10000"`
		UseCorrelations bool                `json:"use_correlations"`
		ContestSize     int                 `json:"contest_size" binding:"required,min=2"`
		PayoutStructure []models.PayoutTier `json:"payout_structure"`
		EntryFee        float64             `json:"entry_fee" binding:"required,min=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Get lineups
	var lineups []models.Lineup
	err := h.db.Preload("Contest").Preload("Players").
		Where("id IN ? AND user_id = ?", req.LineupIDs, userID).
		Find(&lineups).Error
	if err != nil {
		utils.SendInternalError(c, "Failed to fetch lineups")
		return
	}

	if len(lineups) != len(req.LineupIDs) {
		utils.SendValidationError(c, "Some lineups not found", "")
		return
	}

	// Verify all lineups are for the same contest
	contestID := lineups[0].ContestID
	for _, lineup := range lineups {
		if lineup.ContestID != contestID {
			utils.SendValidationError(c, "All lineups must be for the same contest", "")
			return
		}
	}

	// Create simulation config
	config := &models.SimulationConfig{
		NumSimulations:  req.NumSimulations,
		UseCorrelations: req.UseCorrelations,
		ContestSize:     req.ContestSize,
		PayoutStructure: req.PayoutStructure,
		EntryFee:        req.EntryFee,
	}

	if len(config.PayoutStructure) == 0 {
		config.PayoutStructure = simulator.GetPayoutStructure(&lineups[0].Contest)
	}

	// Get all players
	var allPlayers []models.Player
	if err := h.db.Where("contest_id = ?", contestID).Find(&allPlayers).Error; err != nil {
		utils.SendInternalError(c, "Failed to fetch players")
		return
	}

	// Run batch simulation
	results := make([]models.SimulationResult, 0, len(lineups))
	errors := make([]string, 0)

	for _, lineup := range lineups {
		sim := simulator.NewSimulator(config, allPlayers)

		// Run simulation for this lineup
		result, err := sim.SimulateContest([]models.Lineup{lineup}, nil)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Lineup %d: %s", lineup.ID, err.Error()))
			continue
		}

		// Save result
		if err := h.db.Create(result).Error; err == nil {
			results = append(results, *result)

			// Cache result
			ctx := context.Background()
			cacheKey := services.SimulationCacheKey(lineup.ID)
			h.cache.Set(ctx, cacheKey, *result, 30*time.Minute)
		}
	}

	response := gin.H{
		"results":    results,
		"successful": len(results),
		"failed":     len(errors),
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	utils.SendSuccess(c, response)
}

// CompareSimulations compares simulation results for multiple lineups
func (h *SimulationHandler) CompareSimulations(c *gin.Context) {
	userID, _ := c.Get("user_id")
	lineupIDsStr := c.Query("lineup_ids")
	if lineupIDsStr == "" {
		utils.SendValidationError(c, "Lineup IDs required", "")
		return
	}

	// Parse lineup IDs (would parse comma-separated IDs)
	var lineupIDs []uint
	// ... parsing logic ...

	// Verify lineup ownership
	var count int64
	h.db.Model(&models.Lineup{}).Where("id IN ? AND user_id = ?", lineupIDs, userID).Count(&count)
	if int(count) != len(lineupIDs) {
		utils.SendUnauthorized(c, "Access denied to some lineups")
		return
	}

	// Get simulation results
	var results []models.SimulationResult
	err := h.db.Where("lineup_id IN ?", lineupIDs).Find(&results).Error
	if err != nil {
		utils.SendInternalError(c, "Failed to fetch simulation results")
		return
	}

	// Create comparison
	comparison := gin.H{
		"lineup_count": len(lineupIDs),
		"results":      results,
		"summary": gin.H{
			"best_roi":        getBestROI(results),
			"best_win_prob":   getBestWinProbability(results),
			"best_cash_prob":  getBestCashProbability(results),
			"highest_ceiling": getHighestCeiling(results),
			"most_consistent": getMostConsistent(results),
		},
	}

	utils.SendSuccess(c, comparison)
}

// Helper functions

func (h *SimulationHandler) reportProgress(userID uint, lineupID uint, progressChan <-chan models.SimulationProgress) {
	for progress := range progressChan {
		// Broadcast progress via WebSocket
		h.wsHub.BroadcastToUser(userID, "simulation_progress", gin.H{
			"lineup_id":   lineupID,
			"completed":   progress.Completed,
			"total":       progress.TotalSimulations,
			"percentage":  float64(progress.Completed) / float64(progress.TotalSimulations) * 100,
			"eta_seconds": progress.EstimatedTimeRemaining.Seconds(),
		})
	}
}

func getBestROI(results []models.SimulationResult) gin.H {
	if len(results) == 0 {
		return nil
	}

	best := results[0]
	for _, result := range results {
		if result.ROI > best.ROI {
			best = result
		}
	}

	return gin.H{
		"lineup_id": best.LineupID,
		"roi":       best.ROI,
	}
}

func getBestWinProbability(results []models.SimulationResult) gin.H {
	if len(results) == 0 {
		return nil
	}

	best := results[0]
	for _, result := range results {
		if result.WinProbability > best.WinProbability {
			best = result
		}
	}

	return gin.H{
		"lineup_id":       best.LineupID,
		"win_probability": best.WinProbability,
	}
}

func getBestCashProbability(results []models.SimulationResult) gin.H {
	if len(results) == 0 {
		return nil
	}

	best := results[0]
	for _, result := range results {
		if result.CashProbability > best.CashProbability {
			best = result
		}
	}

	return gin.H{
		"lineup_id":        best.LineupID,
		"cash_probability": best.CashProbability,
	}
}

func getHighestCeiling(results []models.SimulationResult) gin.H {
	if len(results) == 0 {
		return nil
	}

	best := results[0]
	for _, result := range results {
		if result.Percentile99 > best.Percentile99 {
			best = result
		}
	}

	return gin.H{
		"lineup_id":     best.LineupID,
		"ceiling_score": best.Percentile99,
	}
}

func getMostConsistent(results []models.SimulationResult) gin.H {
	if len(results) == 0 {
		return nil
	}

	best := results[0]
	lowestStdDev := best.StandardDeviation

	for _, result := range results {
		if result.StandardDeviation < lowestStdDev {
			best = result
			lowestStdDev = result.StandardDeviation
		}
	}

	return gin.H{
		"lineup_id":          best.LineupID,
		"standard_deviation": best.StandardDeviation,
		"mean":               best.Mean,
	}
}
