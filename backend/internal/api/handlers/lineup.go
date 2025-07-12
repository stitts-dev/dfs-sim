package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/optimizer"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
	"gorm.io/gorm"
)

type LineupHandler struct {
	db    *database.DB
	cache *services.CacheService
}

func NewLineupHandler(db *database.DB, cache *services.CacheService) *LineupHandler {
	return &LineupHandler{
		db:    db,
		cache: cache,
	}
}

// GetLineups returns user's lineups
func (h *LineupHandler) GetLineups(c *gin.Context) {
	// TODO: In production, re-enable authentication
	// For development, use a default user ID
	userID := uint(1) // Default user for development

	// Query parameters
	contestID := c.Query("contest_id")
	isSubmitted := c.Query("submitted")
	sortBy := c.DefaultQuery("sortBy", "created_at")
	sortOrder := c.DefaultQuery("sortOrder", "desc")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "20"))

	// Build query
	query := h.db.Model(&models.Lineup{}).Where("user_id = ?", userID)

	if contestID != "" {
		query = query.Where("contest_id = ?", contestID)
	}
	if isSubmitted != "" {
		submitted, _ := strconv.ParseBool(isSubmitted)
		query = query.Where("is_submitted = ?", submitted)
	}

	// Count total
	var total int64
	query.Count(&total)

	// Apply pagination and sorting
	offset := (page - 1) * perPage
	query = query.Offset(offset).Limit(perPage).Order(sortBy + " " + sortOrder)

	// Preload associations
	query = query.Preload("Contest")

	var lineups []models.Lineup
	if err := query.Find(&lineups).Error; err != nil {
		utils.SendInternalError(c, "Failed to fetch lineups")
		return
	}

	// Load players for each lineup
	for i := range lineups {
		if err := lineups[i].LoadPlayers(h.db.DB); err != nil {
			utils.SendInternalError(c, "Failed to load lineup players")
			return
		}
	}

	// Calculate metadata
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	meta := &utils.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}

	utils.SendSuccessWithMeta(c, lineups, meta)
}

// GetLineup returns a single lineup
func (h *LineupHandler) GetLineup(c *gin.Context) {
	// TODO: In production, re-enable authentication
	userID := uint(1) // Default user for development
	lineupIDStr := c.Param("id")
	lineupID, err := strconv.ParseUint(lineupIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid lineup ID", err.Error())
		return
	}

	var lineup models.Lineup
	err = h.db.Preload("Contest").
		Where("id = ? AND user_id = ?", lineupID, userID).
		First(&lineup).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.SendNotFound(c, "Lineup not found")
		} else {
			utils.SendInternalError(c, "Failed to fetch lineup")
		}
		return
	}

	// Load players for the lineup
	if err := lineup.LoadPlayers(h.db.DB); err != nil {
		utils.SendInternalError(c, "Failed to load lineup players")
		return
	}

	utils.SendSuccess(c, lineup)
}

// CreateLineup creates a new lineup
func (h *LineupHandler) CreateLineup(c *gin.Context) {
	// TODO: In production, re-enable authentication
	userID := uint(1) // Default user for development

	var req struct {
		ContestID uint   `json:"contest_id" binding:"required"`
		Name      string `json:"name"`
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

	// Create lineup
	lineup := models.Lineup{
		UserID:    userID,
		ContestID: req.ContestID,
		Name:      req.Name,
		Players:   players,
		Contest:   contest,
	}

	// Calculate totals
	lineup.CalculateTotalSalary()
	lineup.CalculateProjectedPoints()
	lineup.CalculateOwnership()

	// Validate lineup
	constraints := optimizer.GetConstraintsForContest(&contest)
	if err := constraints.ValidateLineup(&lineup); err != nil {
		utils.SendValidationError(c, "Invalid lineup", err.Error())
		return
	}

	// Initialize player positions for manual lineup (each player fills their natural position)
	lineup.PlayerPositions = make(map[uint]string)
	for _, player := range lineup.Players {
		lineup.PlayerPositions[player.ID] = player.Position
	}

	// Store players temporarily and clear from lineup to prevent GORM association
	savedPlayers := lineup.Players
	lineup.Players = nil

	// Save lineup with custom transaction to handle positions
	tx := h.db.Begin()
	if err := tx.Create(&lineup).Error; err != nil {
		tx.Rollback()
		utils.SendInternalError(c, "Failed to create lineup")
		return
	}

	// Save player positions in join table
	for _, player := range savedPlayers {
		lineupPlayer := models.LineupPlayer{
			LineupID: lineup.ID,
			PlayerID: player.ID,
			Position: player.Position,
		}
		if err := tx.Create(&lineupPlayer).Error; err != nil {
			tx.Rollback()
			utils.SendInternalError(c, "Failed to save lineup players")
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		utils.SendInternalError(c, "Failed to save lineup")
		return
	}

	// Restore players for response
	lineup.Players = savedPlayers

	// Clear cache
	ctx := context.Background()
	cacheKey := services.LineupCacheKey(userID, req.ContestID)
	h.cache.Delete(ctx, cacheKey)

	utils.SendSuccess(c, lineup)
}

// UpdateLineup updates an existing lineup
func (h *LineupHandler) UpdateLineup(c *gin.Context) {
	// TODO: In production, re-enable authentication
	userID := uint(1) // Default user for development
	lineupIDStr := c.Param("id")
	lineupID, err := strconv.ParseUint(lineupIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid lineup ID", err.Error())
		return
	}

	var req struct {
		Name      string `json:"name"`
		PlayerIDs []uint `json:"player_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Get existing lineup
	var lineup models.Lineup
	err = h.db.Preload("Contest").Where("id = ? AND user_id = ?", lineupID, userID).First(&lineup).Error
	if err != nil {
		utils.SendNotFound(c, "Lineup not found")
		return
	}

	// Check if already submitted
	if lineup.IsSubmitted {
		utils.SendValidationError(c, "Cannot update submitted lineup", "")
		return
	}

	// Update name if provided
	if req.Name != "" {
		lineup.Name = req.Name
	}

	// Update players if provided
	if len(req.PlayerIDs) > 0 {
		var players []models.Player
		if err := h.db.Where("id IN ? AND contest_id = ?", req.PlayerIDs, lineup.ContestID).Find(&players).Error; err != nil {
			utils.SendInternalError(c, "Failed to fetch players")
			return
		}

		if len(players) != len(req.PlayerIDs) {
			utils.SendValidationError(c, "Some players not found", "")
			return
		}

		lineup.Players = players
		lineup.CalculateTotalSalary()
		lineup.CalculateProjectedPoints()
		lineup.CalculateOwnership()

		// Validate updated lineup
		constraints := optimizer.GetConstraintsForContest(&lineup.Contest)
		if err := constraints.ValidateLineup(&lineup); err != nil {
			utils.SendValidationError(c, "Invalid lineup", err.Error())
			return
		}
	}

	// Save updates with transaction
	tx := h.db.Begin()
	if err := tx.Save(&lineup).Error; err != nil {
		tx.Rollback()
		utils.SendInternalError(c, "Failed to update lineup")
		return
	}

	// Update player associations if provided
	if len(req.PlayerIDs) > 0 {
		// Delete existing lineup players
		if err := tx.Where("lineup_id = ?", lineup.ID).Delete(&models.LineupPlayer{}).Error; err != nil {
			tx.Rollback()
			utils.SendInternalError(c, "Failed to update lineup players")
			return
		}

		// Add new lineup players
		for _, player := range lineup.Players {
			lineupPlayer := models.LineupPlayer{
				LineupID: lineup.ID,
				PlayerID: player.ID,
				Position: player.Position,
			}
			if err := tx.Create(&lineupPlayer).Error; err != nil {
				tx.Rollback()
				utils.SendInternalError(c, "Failed to update lineup players")
				return
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		utils.SendInternalError(c, "Failed to update lineup")
		return
	}

	// Clear cache
	ctx := context.Background()
	cacheKey := services.LineupCacheKey(userID, lineup.ContestID)
	h.cache.Delete(ctx, cacheKey)

	// Reload lineup with players
	h.db.First(&lineup, lineup.ID)
	if err := lineup.LoadPlayers(h.db.DB); err != nil {
		utils.SendInternalError(c, "Failed to load lineup players")
		return
	}

	utils.SendSuccess(c, lineup)
}

// DeleteLineup deletes a lineup
func (h *LineupHandler) DeleteLineup(c *gin.Context) {
	// TODO: In production, re-enable authentication
	userID := uint(1) // Default user for development
	lineupIDStr := c.Param("id")
	lineupID, err := strconv.ParseUint(lineupIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid lineup ID", err.Error())
		return
	}

	// Get lineup
	var lineup models.Lineup
	err = h.db.Where("id = ? AND user_id = ?", lineupID, userID).First(&lineup).Error
	if err != nil {
		utils.SendNotFound(c, "Lineup not found")
		return
	}

	// Check if already submitted
	if lineup.IsSubmitted {
		utils.SendValidationError(c, "Cannot delete submitted lineup", "")
		return
	}

	// Delete lineup (cascade will handle associations)
	if err := h.db.Delete(&lineup).Error; err != nil {
		utils.SendInternalError(c, "Failed to delete lineup")
		return
	}

	// Clear cache
	ctx := context.Background()
	cacheKey := services.LineupCacheKey(userID, lineup.ContestID)
	h.cache.Delete(ctx, cacheKey)

	utils.SendSuccess(c, gin.H{"message": "Lineup deleted successfully"})
}

// SubmitLineup submits a lineup to a contest
func (h *LineupHandler) SubmitLineup(c *gin.Context) {
	// TODO: In production, re-enable authentication
	userID := uint(1) // Default user for development
	lineupIDStr := c.Param("id")
	lineupID, err := strconv.ParseUint(lineupIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid lineup ID", err.Error())
		return
	}

	// Get lineup with contest
	var lineup models.Lineup
	err = h.db.Preload("Contest").Preload("Players").
		Where("id = ? AND user_id = ?", lineupID, userID).
		First(&lineup).Error
	if err != nil {
		utils.SendNotFound(c, "Lineup not found")
		return
	}

	// Check if already submitted
	if lineup.IsSubmitted {
		utils.SendValidationError(c, "Lineup already submitted", "")
		return
	}

	// Check contest is still open
	if time.Now().After(lineup.Contest.StartTime) {
		utils.SendValidationError(c, "Contest has already started", "")
		return
	}

	// Validate lineup one more time
	constraints := optimizer.GetConstraintsForContest(&lineup.Contest)
	if err := constraints.ValidateLineup(&lineup); err != nil {
		utils.SendValidationError(c, "Invalid lineup", err.Error())
		return
	}

	// Check entry limits
	var userEntries int64
	h.db.Model(&models.Lineup{}).
		Where("user_id = ? AND contest_id = ? AND is_submitted = ?", userID, lineup.ContestID, true).
		Count(&userEntries)

	if int(userEntries) >= lineup.Contest.MaxLineupsPerUser {
		utils.SendValidationError(c, "Maximum entries reached for this contest", "")
		return
	}

	// Submit lineup
	lineup.IsSubmitted = true
	if err := h.db.Save(&lineup).Error; err != nil {
		utils.SendInternalError(c, "Failed to submit lineup")
		return
	}

	// Update contest entry count
	h.db.Model(&lineup.Contest).UpdateColumn("total_entries", gorm.Expr("total_entries + ?", 1))

	utils.SendSuccess(c, gin.H{
		"message": "Lineup submitted successfully",
		"lineup":  lineup,
	})
}

// CloneLineup creates a copy of an existing lineup
func (h *LineupHandler) CloneLineup(c *gin.Context) {
	// TODO: In production, re-enable authentication
	userID := uint(1) // Default user for development
	lineupIDStr := c.Param("id")
	lineupID, err := strconv.ParseUint(lineupIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid lineup ID", err.Error())
		return
	}

	// Get original lineup
	var original models.Lineup
	err = h.db.Preload("Players").Where("id = ? AND user_id = ?", lineupID, userID).First(&original).Error
	if err != nil {
		utils.SendNotFound(c, "Lineup not found")
		return
	}

	// Create clone
	clone := models.Lineup{
		UserID:          userID,
		ContestID:       original.ContestID,
		Name:            original.Name + " (Copy)",
		TotalSalary:     original.TotalSalary,
		ProjectedPoints: original.ProjectedPoints,
		Ownership:       original.Ownership,
		Players:         original.Players,
		IsOptimized:     original.IsOptimized,
	}

	// Save clone
	if err := h.db.Create(&clone).Error; err != nil {
		utils.SendInternalError(c, "Failed to clone lineup")
		return
	}

	// Clear cache
	ctx := context.Background()
	cacheKey := services.LineupCacheKey(userID, original.ContestID)
	h.cache.Delete(ctx, cacheKey)

	utils.SendSuccess(c, clone)
}

// GetLineupComparison compares multiple lineups
func (h *LineupHandler) GetLineupComparison(c *gin.Context) {
	// TODO: In production, re-enable authentication
	userID := uint(1) // Default user for development
	lineupIDsStr := c.Query("ids")
	if lineupIDsStr == "" {
		utils.SendValidationError(c, "Lineup IDs required", "")
		return
	}

	// Parse lineup IDs
	var lineupIDs []uint
	// Would parse comma-separated IDs here

	var lineups []models.Lineup
	err := h.db.Preload("Players").
		Where("id IN ? AND user_id = ?", lineupIDs, userID).
		Find(&lineups).Error
	if err != nil {
		utils.SendInternalError(c, "Failed to fetch lineups")
		return
	}

	// Calculate comparison metrics
	comparison := gin.H{
		"lineups":            lineups,
		"unique_players":     calculateUniquePlayers(lineups),
		"overlap_percentage": calculateOverlapPercentage(lineups),
		"salary_range":       calculateSalaryRange(lineups),
		"projection_range":   calculateProjectionRange(lineups),
	}

	utils.SendSuccess(c, comparison)
}

// Helper functions

func calculateUniquePlayers(lineups []models.Lineup) int {
	uniquePlayers := make(map[uint]bool)
	for _, lineup := range lineups {
		for _, player := range lineup.Players {
			uniquePlayers[player.ID] = true
		}
	}
	return len(uniquePlayers)
}

func calculateOverlapPercentage(lineups []models.Lineup) float64 {
	if len(lineups) < 2 {
		return 0
	}

	// Count player occurrences
	playerCount := make(map[uint]int)
	for _, lineup := range lineups {
		for _, player := range lineup.Players {
			playerCount[player.ID]++
		}
	}

	// Calculate overlap
	overlapping := 0
	for _, count := range playerCount {
		if count > 1 {
			overlapping++
		}
	}

	totalUnique := len(playerCount)
	if totalUnique == 0 {
		return 0
	}

	return float64(overlapping) / float64(totalUnique) * 100
}

func calculateSalaryRange(lineups []models.Lineup) gin.H {
	if len(lineups) == 0 {
		return gin.H{"min": 0, "max": 0, "avg": 0}
	}

	min := lineups[0].TotalSalary
	max := lineups[0].TotalSalary
	sum := 0

	for _, lineup := range lineups {
		if lineup.TotalSalary < min {
			min = lineup.TotalSalary
		}
		if lineup.TotalSalary > max {
			max = lineup.TotalSalary
		}
		sum += lineup.TotalSalary
	}

	return gin.H{
		"min": min,
		"max": max,
		"avg": sum / len(lineups),
	}
}

func calculateProjectionRange(lineups []models.Lineup) gin.H {
	if len(lineups) == 0 {
		return gin.H{"min": 0, "max": 0, "avg": 0}
	}

	min := lineups[0].ProjectedPoints
	max := lineups[0].ProjectedPoints
	sum := 0.0

	for _, lineup := range lineups {
		if lineup.ProjectedPoints < min {
			min = lineup.ProjectedPoints
		}
		if lineup.ProjectedPoints > max {
			max = lineup.ProjectedPoints
		}
		sum += lineup.ProjectedPoints
	}

	return gin.H{
		"min": min,
		"max": max,
		"avg": sum / float64(len(lineups)),
	}
}
