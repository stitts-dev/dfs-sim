package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
)

type PreferencesHandler struct {
	db    *database.DB
	cache *services.CacheService
}

func NewPreferencesHandler(db *database.DB, cache *services.CacheService) *PreferencesHandler {
	return &PreferencesHandler{
		db:    db,
		cache: cache,
	}
}

// GetPreferences returns the current user's preferences
// GET /api/user/preferences
func (h *PreferencesHandler) GetPreferences(c *gin.Context) {
	// Get user ID from auth context or use default for development
	userIDValue, exists := c.Get("user_id")
	if !exists {
		// Use default user ID 1 for development
		userIDValue = 1
	}

	// Convert user ID to int
	var userID int
	switch v := userIDValue.(type) {
	case uint:
		userID = int(v)
	case int:
		userID = v
	case string:
		var err error
		userID, err = strconv.Atoi(v)
		if err != nil {
			utils.SendInternalError(c, "Invalid user ID format")
			return
		}
	default:
		utils.SendInternalError(c, "Invalid user ID type")
		return
	}

	// Get or create user preferences
	prefs, err := models.GetUserPreferences(h.db, userID)
	if err != nil {
		utils.SendInternalError(c, "Failed to retrieve preferences")
		return
	}

	utils.SendSuccess(c, prefs)
}

// UpdatePreferences updates the current user's preferences
// PUT /api/user/preferences
func (h *PreferencesHandler) UpdatePreferences(c *gin.Context) {
	// Get user ID from auth context or use default for development
	userIDValue, exists := c.Get("user_id")
	if !exists {
		// Use default user ID 1 for development
		userIDValue = 1
	}

	// Convert user ID to int
	var userID int
	switch v := userIDValue.(type) {
	case uint:
		userID = int(v)
	case int:
		userID = v
	case string:
		var err error
		userID, err = strconv.Atoi(v)
		if err != nil {
			utils.SendInternalError(c, "Invalid user ID format")
			return
		}
	default:
		utils.SendInternalError(c, "Invalid user ID type")
		return
	}

	// Define the update request structure
	var updateReq struct {
		BeginnerMode         *bool    `json:"beginner_mode"`
		ShowTooltips         *bool    `json:"show_tooltips"`
		TooltipDelay         *int     `json:"tooltip_delay"`
		PreferredSports      []string `json:"preferred_sports"`
		AISuggestionsEnabled *bool    `json:"ai_suggestions_enabled"`
	}

	// Bind JSON request
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Build updates map - only include non-nil fields
	updates := make(map[string]interface{})

	if updateReq.BeginnerMode != nil {
		updates["beginner_mode"] = *updateReq.BeginnerMode
	}
	if updateReq.ShowTooltips != nil {
		updates["show_tooltips"] = *updateReq.ShowTooltips
	}
	if updateReq.TooltipDelay != nil {
		// Validate tooltip delay
		if *updateReq.TooltipDelay < 0 || *updateReq.TooltipDelay > 5000 {
			utils.SendValidationError(c, "Invalid tooltip delay", "Tooltip delay must be between 0 and 5000 milliseconds")
			return
		}
		updates["tooltip_delay"] = *updateReq.TooltipDelay
	}
	if updateReq.PreferredSports != nil {
		// Validate sports
		validSports := []string{"nfl", "nba", "mlb", "nhl", "pga", "nascar", "mma", "soccer"}
		for _, sport := range updateReq.PreferredSports {
			isValid := false
			for _, valid := range validSports {
				if sport == valid {
					isValid = true
					break
				}
			}
			if !isValid {
				utils.SendValidationError(c, "Invalid sport", "Sport '"+sport+"' is not supported")
				return
			}
		}
		updates["preferred_sports"] = updateReq.PreferredSports
	}
	if updateReq.AISuggestionsEnabled != nil {
		updates["ai_suggestions_enabled"] = *updateReq.AISuggestionsEnabled
	}

	// Ensure user preferences exist first
	_, err := models.GetUserPreferences(h.db, userID)
	if err != nil {
		utils.SendInternalError(c, "Failed to retrieve preferences")
		return
	}

	// Update preferences
	if len(updates) > 0 {
		if err := models.UpdateUserPreferences(h.db, userID, updates); err != nil {
			utils.SendInternalError(c, "Failed to update preferences")
			return
		}
	}

	// Return updated preferences
	prefs, err := models.GetUserPreferences(h.db, userID)
	if err != nil {
		utils.SendInternalError(c, "Failed to retrieve updated preferences")
		return
	}

	utils.SendSuccess(c, prefs)
}

// ResetPreferences resets the current user's preferences to defaults
// POST /api/user/preferences/reset
func (h *PreferencesHandler) ResetPreferences(c *gin.Context) {
	// Get user ID from auth context or use default for development
	userIDValue, exists := c.Get("user_id")
	if !exists {
		// Use default user ID 1 for development
		userIDValue = 1
	}

	// Convert user ID to int
	var userID int
	switch v := userIDValue.(type) {
	case uint:
		userID = int(v)
	case int:
		userID = v
	case string:
		var err error
		userID, err = strconv.Atoi(v)
		if err != nil {
			utils.SendInternalError(c, "Invalid user ID format")
			return
		}
	default:
		utils.SendInternalError(c, "Invalid user ID type")
		return
	}

	// Reset to default values
	defaultPrefs := map[string]interface{}{
		"beginner_mode":          false,
		"show_tooltips":          true,
		"tooltip_delay":          500,
		"preferred_sports":       []string{},
		"ai_suggestions_enabled": true,
	}

	// Ensure user preferences exist first
	_, err := models.GetUserPreferences(h.db, userID)
	if err != nil {
		utils.SendInternalError(c, "Failed to retrieve preferences")
		return
	}

	// Update to defaults
	if err := models.UpdateUserPreferences(h.db, userID, defaultPrefs); err != nil {
		utils.SendInternalError(c, "Failed to reset preferences")
		return
	}

	// Return updated preferences
	prefs, err := models.GetUserPreferences(h.db, userID)
	if err != nil {
		utils.SendInternalError(c, "Failed to retrieve updated preferences")
		return
	}

	utils.SendSuccess(c, prefs)
}