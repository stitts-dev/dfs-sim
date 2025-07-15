package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/services/user-service/internal/models"
	"github.com/stitts-dev/dfs-sim/services/user-service/internal/services"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"gorm.io/gorm"
)

// SimpleUserHandler handles user management endpoints with integer-based models
type SimpleUserHandler struct {
	userService *services.UserService
	logger      *logrus.Logger
	db          *database.DB
}

// NewSimpleUserHandler creates a new user handler
func NewSimpleUserHandler(userService *services.UserService, logger *logrus.Logger, db *database.DB) *SimpleUserHandler {
	return &SimpleUserHandler{
		userService: userService,
		logger:      logger,
		db:          db,
	}
}

// GetProfile gets the current user's profile
func (h *SimpleUserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	// Parse UUID from context
	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		h.logger.WithError(err).Error("Invalid user ID format")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	user, err := models.GetUserByID(h.db, userUUID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to retrieve user profile")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user profile"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateProfile updates the current user's profile
func (h *SimpleUserHandler) UpdateProfile(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "User profile update not yet implemented",
		"message": "This endpoint will be implemented in the next iteration",
	})
}

// GetPreferences gets user preferences
func (h *SimpleUserHandler) GetPreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		// Return default preferences for anonymous users
		defaultPrefs := &models.UserPreferences{
			BeginnerMode:         true,
			TooltipsEnabled:      true,
			NotificationsEnabled: true,
			TutorialCompleted:    false,
		}
		c.JSON(http.StatusOK, defaultPrefs)
		return
	}

	// Parse UUID from context
	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		h.logger.WithError(err).Error("Invalid user ID format")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	user, err := models.GetUserByID(h.db, userUUID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Create default preferences if none exist
	if user.Preferences == nil {
		prefs, err := models.CreateUserPreferences(h.db, userUUID)
		if err != nil {
			h.logger.WithError(err).Error("Failed to create user preferences")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create preferences"})
			return
		}
		c.JSON(http.StatusOK, prefs)
		return
	}

	c.JSON(http.StatusOK, user.Preferences)
}

// UpdatePreferences updates user preferences
func (h *SimpleUserHandler) UpdatePreferences(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "User preferences update not yet implemented",
		"message": "This endpoint will be implemented in the next iteration",
	})
}

// GetSubscription gets user subscription info
func (h *SimpleUserHandler) GetSubscription(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	// Parse UUID from context
	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		h.logger.WithError(err).Error("Invalid user ID format")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	user, err := models.GetUserByID(h.db, userUUID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	tier, err := user.GetTier(h.db)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get subscription tier")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tier":                    user.SubscriptionTier,
		"status":                  user.SubscriptionStatus,
		"expires_at":              user.SubscriptionExpiresAt,
		"monthly_optimizations_used": user.MonthlyOptimizationsUsed,
		"monthly_simulations_used":   user.MonthlySimulationsUsed,
		"tier_details":            tier,
	})
}

// UpdateSubscription updates user subscription
func (h *SimpleUserHandler) UpdateSubscription(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "User subscription update not yet implemented",
		"message": "This endpoint will be implemented in the next iteration",
	})
}

// Admin endpoints

// ListUsers lists all users (admin only)
func (h *SimpleUserHandler) ListUsers(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "User listing not yet implemented",
		"message": "This endpoint will be implemented in the next iteration",
	})
}

// GetUser gets a specific user by ID (admin only)
func (h *SimpleUserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := models.GetUserByID(h.db, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser updates a specific user (admin only)
func (h *SimpleUserHandler) UpdateUser(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "User update not yet implemented",
		"message": "This endpoint will be implemented in the next iteration",
	})
}

// DeleteUser deletes a specific user (admin only)
func (h *SimpleUserHandler) DeleteUser(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "User deletion not yet implemented",
		"message": "This endpoint will be implemented in the next iteration",
	})
}