package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

type LineupHandler struct {
	db     *database.DB
	logger *logrus.Logger
}

func NewLineupHandler(db *database.DB, logger *logrus.Logger) *LineupHandler {
	return &LineupHandler{
		db:     db,
		logger: logger,
	}
}

func (h *LineupHandler) GetUserLineups(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	userIDStr, ok := userIDInterface.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	sport := c.Query("sport")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter"})
		return
	}

	query := h.db.Where("user_id = ?", userID)

	if sport != "" {
		query = query.Where("sport = ?", sport)
	}

	var lineups []types.Lineup
	if err := query.Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&lineups).Error; err != nil {
		h.logger.WithError(err).Error("Failed to fetch user lineups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Get total count for pagination
	var total int64
	countQuery := h.db.Model(&types.Lineup{}).Where("user_id = ?", userID)
	if sport != "" {
		countQuery = countQuery.Where("sport = ?", sport)
	}
	countQuery.Count(&total)

	c.JSON(http.StatusOK, gin.H{
		"lineups": lineups,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func (h *LineupHandler) CreateLineup(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	userIDStr, ok := userIDInterface.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	var req types.Lineup
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set user ID and validate
	req.UserID = userID
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lineup name is required"})
		return
	}

	if req.Sport == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sport is required"})
		return
	}

	// Create lineup
	if err := h.db.Create(&req).Error; err != nil {
		h.logger.WithError(err).Error("Failed to create lineup")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, req)
}

func (h *LineupHandler) GetLineup(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	userIDStr, ok := userIDInterface.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	lineupIDStr := c.Param("id")
	lineupID, err := uuid.Parse(lineupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lineup ID"})
		return
	}

	var lineup types.Lineup
	if err := h.db.Where("id = ? AND user_id = ?", lineupID, userID).First(&lineup).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Lineup not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to fetch lineup")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, lineup)
}

func (h *LineupHandler) UpdateLineup(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	userIDStr, ok := userIDInterface.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	lineupIDStr := c.Param("id")
	lineupID, err := uuid.Parse(lineupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lineup ID"})
		return
	}

	// Find existing lineup
	var lineup types.Lineup
	if err := h.db.Where("id = ? AND user_id = ?", lineupID, userID).First(&lineup).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Lineup not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to fetch lineup")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	var req types.Lineup
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	lineup.Name = req.Name
	lineup.Sport = req.Sport
	lineup.Players = req.Players
	lineup.TotalSalary = req.TotalSalary
	lineup.ProjectedPoints = req.ProjectedPoints
	lineup.ActualPoints = req.ActualPoints
	lineup.IsLocked = req.IsLocked

	if err := h.db.Save(&lineup).Error; err != nil {
		h.logger.WithError(err).Error("Failed to update lineup")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, lineup)
}

func (h *LineupHandler) DeleteLineup(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	userIDStr, ok := userIDInterface.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	lineupIDStr := c.Param("id")
	lineupID, err := uuid.Parse(lineupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lineup ID"})
		return
	}

	// Check if lineup exists and belongs to user
	var lineup types.Lineup
	if err := h.db.Where("id = ? AND user_id = ?", lineupID, userID).First(&lineup).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Lineup not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to fetch lineup")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Delete lineup
	if err := h.db.Delete(&lineup).Error; err != nil {
		h.logger.WithError(err).Error("Failed to delete lineup")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Lineup deleted successfully"})
}

func (h *LineupHandler) ExportLineup(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	userIDStr, ok := userIDInterface.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	lineupIDStr := c.Param("id")
	lineupID, err := uuid.Parse(lineupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lineup ID"})
		return
	}

	platform := c.Query("platform")
	if platform == "" {
		platform = "draftkings" // Default to DraftKings
	}

	// Find lineup
	var lineup types.Lineup
	if err := h.db.Where("id = ? AND user_id = ?", lineupID, userID).First(&lineup).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Lineup not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to fetch lineup")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// For now, return a simple export format
	// In a real implementation, this would generate platform-specific CSV
	exportData := gin.H{
		"lineup_id": lineup.ID,
		"name":      lineup.Name,
		"platform":  platform,
		"players":   lineup.Players,
		"format":    "csv", // Could be extended to support other formats
	}

	c.JSON(http.StatusOK, exportData)
}