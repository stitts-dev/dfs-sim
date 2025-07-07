package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/dfs"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
)

type PlayerHandler struct {
	db          *database.DB
	cache       *services.CacheService
	aggregator  *services.DataAggregator
	dataFetcher *services.DataFetcherService
}

func NewPlayerHandler(db *database.DB, cache *services.CacheService, aggregator *services.DataAggregator, dataFetcher *services.DataFetcherService) *PlayerHandler {
	return &PlayerHandler{
		db:          db,
		cache:       cache,
		aggregator:  aggregator,
		dataFetcher: dataFetcher,
	}
}

// GetPlayers returns all players for a contest
func (h *PlayerHandler) GetPlayers(c *gin.Context) {
	contestIDStr := c.Param("id")
	contestID, err := strconv.ParseUint(contestIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid contest ID", err.Error())
		return
	}

	// Check cache first
	cacheKey := services.PlayersCacheKey(uint(contestID))
	var players []models.Player

	ctx := context.Background()
	if err := h.cache.Get(ctx, cacheKey, &players); err == nil {
		utils.SendSuccess(c, players)
		return
	}

	// Check if we need to fetch fresh data
	var contest models.Contest
	if err := h.db.First(&contest, contestID).Error; err != nil {
		utils.SendNotFound(c, "Contest not found")
		return
	}

	// If data is stale (older than 2 hours), trigger a background update
	if time.Since(contest.LastDataUpdate) > 2*time.Hour {
		go h.dataFetcher.FetchOnDemand(uint(contestID))
	}

	// Query parameters
	position := c.Query("position")
	team := c.Query("team")
	minSalary := c.DefaultQuery("minSalary", "0")
	maxSalary := c.DefaultQuery("maxSalary", "999999")
	sortBy := c.DefaultQuery("sortBy", "projected_points")
	sortOrder := c.DefaultQuery("sortOrder", "desc")
	search := c.Query("search")

	// Build query
	query := h.db.Model(&models.Player{}).Where("contest_id = ?", contestID)

	// Apply filters
	if position != "" {
		query = query.Where("position = ?", position)
	}
	if team != "" {
		query = query.Where("team = ?", team)
	}

	minSal, _ := strconv.Atoi(minSalary)
	maxSal, _ := strconv.Atoi(maxSalary)
	query = query.Where("salary >= ? AND salary <= ?", minSal, maxSal)

	if search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
	}

	// Apply sorting
	orderClause := sortBy + " " + sortOrder
	query = query.Order(orderClause)

	// Execute query
	if err := query.Find(&players).Error; err != nil {
		utils.SendInternalError(c, "Failed to fetch players")
		return
	}

	// Cache the results if no filters applied
	if position == "" && team == "" && search == "" {
		h.cache.SetWithRetry(ctx, cacheKey, players, 5*time.Minute, 3)
	}

	utils.SendSuccess(c, players)
}

// GetPlayer returns a single player by ID
func (h *PlayerHandler) GetPlayer(c *gin.Context) {
	playerIDStr := c.Param("id")
	playerID, err := strconv.ParseUint(playerIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid player ID", err.Error())
		return
	}

	var player models.Player
	if err := h.db.First(&player, playerID).Error; err != nil {
		utils.SendNotFound(c, "Player not found")
		return
	}

	utils.SendSuccess(c, player)
}

// GetPlayerStats returns historical stats for a player
func (h *PlayerHandler) GetPlayerStats(c *gin.Context) {
	playerIDStr := c.Param("id")
	playerID, err := strconv.ParseUint(playerIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid player ID", err.Error())
		return
	}

	// Get player to determine sport
	var player models.Player
	if err := h.db.First(&player, playerID).Error; err != nil {
		utils.SendNotFound(c, "Player not found")
		return
	}

	// Get aggregated player data for more detailed stats
	cacheKey := "aggregated:player:" + strconv.FormatUint(playerID, 10)
	var aggPlayer dfs.AggregatedPlayer
	err = h.cache.GetSimple(cacheKey, &aggPlayer)
	
	stats := map[string]interface{}{
		"player_id": playerID,
		"stats":     player.ProjectedPoints,
	}
	
	// If we have aggregated data, include more stats
	if err == nil && aggPlayer.Stats != nil {
		stats["detailed_stats"] = aggPlayer.Stats
		stats["confidence"] = aggPlayer.Confidence
		stats["data_sources"] = []string{}
		if aggPlayer.ESPNData != nil {
			stats["data_sources"] = append(stats["data_sources"].([]string), "espn")
		}
		if aggPlayer.BallDontLieData != nil {
			stats["data_sources"] = append(stats["data_sources"].([]string), "balldontlie")
		}
	}

	utils.SendSuccess(c, stats)
}

// UpdatePlayerProjections updates player projections (admin only)
func (h *PlayerHandler) UpdatePlayerProjections(c *gin.Context) {
	var updates []struct {
		PlayerID        uint    `json:"player_id" binding:"required"`
		ProjectedPoints float64 `json:"projected_points" binding:"required,min=0"`
		FloorPoints     float64 `json:"floor_points" binding:"required,min=0"`
		CeilingPoints   float64 `json:"ceiling_points" binding:"required,min=0"`
		Ownership       float64 `json:"ownership" binding:"min=0,max=100"`
	}

	if err := c.ShouldBindJSON(&updates); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Update each player
	tx := h.db.Begin()
	for _, update := range updates {
		if err := tx.Model(&models.Player{}).Where("id = ?", update.PlayerID).Updates(map[string]interface{}{
			"projected_points": update.ProjectedPoints,
			"floor_points":     update.FloorPoints,
			"ceiling_points":   update.CeilingPoints,
			"ownership":        update.Ownership,
		}).Error; err != nil {
			tx.Rollback()
			utils.SendInternalError(c, "Failed to update player projections")
			return
		}
	}
	tx.Commit()

	// Clear cache
	// Would need to get contest ID to clear specific cache
	// For now, just return success

	utils.SendSuccess(c, gin.H{
		"updated": len(updates),
		"message": "Player projections updated successfully",
	})
}

// GetPlayerNews returns recent news for a player
func (h *PlayerHandler) GetPlayerNews(c *gin.Context) {
	playerIDStr := c.Param("id")
	playerID, err := strconv.ParseUint(playerIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid player ID", err.Error())
		return
	}

	// Get player
	var player models.Player
	if err := h.db.First(&player, playerID).Error; err != nil {
		utils.SendNotFound(c, "Player not found")
		return
	}

	// Check for injury status
	news := []map[string]interface{}{}
	
	if player.IsInjured {
		news = append(news, map[string]interface{}{
			"id":        1,
			"player_id": playerID,
			"title":     "Injury Update",
			"content":   player.InjuryStatus,
			"source":    "Team Report",
			"published": player.UpdatedAt,
			"impact":    "negative",
		})
	}
	
	// In production, you would fetch from ESPN news API or similar
	// For now, return injury status if available

	utils.SendSuccess(c, news)
}

// GetPositionStats returns aggregate stats for a position
func (h *PlayerHandler) GetPositionStats(c *gin.Context) {
	contestIDStr := c.Param("contestId")
	contestID, err := strconv.ParseUint(contestIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid contest ID", err.Error())
		return
	}

	position := c.Query("position")
	if position == "" {
		utils.SendValidationError(c, "Position parameter required", "")
		return
	}

	// Get aggregate stats for position
	var result struct {
		AvgProjected float64
		AvgSalary    float64
		MinSalary    int
		MaxSalary    int
		Count        int
	}

	h.db.Model(&models.Player{}).
		Select("AVG(projected_points) as avg_projected, AVG(salary) as avg_salary, MIN(salary) as min_salary, MAX(salary) as max_salary, COUNT(*) as count").
		Where("contest_id = ? AND position = ?", contestID, position).
		Scan(&result)

	stats := gin.H{
		"position":      position,
		"avg_projected": result.AvgProjected,
		"avg_salary":    result.AvgSalary,
		"min_salary":    result.MinSalary,
		"max_salary":    result.MaxSalary,
		"player_count":  result.Count,
		"avg_value":     result.AvgProjected / (result.AvgSalary / 1000), // Points per thousand dollars
	}

	utils.SendSuccess(c, stats)
}
