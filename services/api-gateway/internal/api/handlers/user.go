package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

type UserHandler struct {
	db     *database.DB
	logger *logrus.Logger
}

func NewUserHandler(db *database.DB, logger *logrus.Logger) *UserHandler {
	return &UserHandler{
		db:     db,
		logger: logger,
	}
}


func (h *UserHandler) GetPreferences(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	userID, ok := userIDInterface.(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	var preferences types.UserPreferences
	if err := h.db.Where("user_id = ?", uint(userID)).First(&preferences).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return default preferences if none exist
			c.JSON(http.StatusOK, gin.H{
				"user_id":                uint(userID),
				"beginner_mode":          true,
				"default_sport":          "nfl",
				"default_contest_type":   "cash",
				"max_exposure":           20.0,
				"min_stack_size":         2,
				"max_stack_size":         4,
				"auto_optimize":          false,
				"notification_settings": gin.H{},
			})
			return
		}
		h.logger.WithError(err).Error("Failed to fetch user preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, preferences)
}

func (h *UserHandler) UpdatePreferences(c *gin.Context) {
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

	var req types.UserPreferences
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set user ID
	req.UserID = userID

	// Try to find existing preferences
	var existingPrefs types.UserPreferences
	if err := h.db.Where("user_id = ?", userID).First(&existingPrefs).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new preferences
			if err := h.db.Create(&req).Error; err != nil {
				h.logger.WithError(err).Error("Failed to create user preferences")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				return
			}
		} else {
			h.logger.WithError(err).Error("Failed to fetch user preferences")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
	} else {
		// Update existing preferences
		req.ID = existingPrefs.ID
		if err := h.db.Save(&req).Error; err != nil {
			h.logger.WithError(err).Error("Failed to update user preferences")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
	}

	c.JSON(http.StatusOK, req)
}

