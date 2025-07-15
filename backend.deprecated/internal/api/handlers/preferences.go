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
	// Check if user is authenticated
	authenticated, _ := c.Get("authenticated")
	isAuthenticated := authenticated == true

	// For anonymous users, return default preferences without database storage
	if !isAuthenticated {
		defaultPrefs := &models.UserPreferences{
			UserID:               0, // 0 indicates anonymous user
			BeginnerMode:         true,
			ShowTooltips:         true,
			TooltipDelay:         500,
			PreferredSports:      []string{},
			AISuggestionsEnabled: true,
		}
		utils.SendSuccess(c, defaultPrefs)
		return
	}

	// For authenticated users, get user ID from auth context
	userIDValue, exists := c.Get("user_id")
	if !exists {
		utils.SendInternalError(c, "User ID not found in auth context")
		return
	}

	// Convert user ID to uint
	var userID uint
	switch v := userIDValue.(type) {
	case uint:
		userID = v
	case int:
		userID = uint(v)
	case string:
		var err error
		parsed, err := strconv.Atoi(v)
		if err != nil {
			utils.SendInternalError(c, "Invalid user ID format")
			return
		}
		userID = uint(parsed)
	default:
		utils.SendInternalError(c, "Invalid user ID type")
		return
	}

	// Get or create user preferences from database
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
	// Check if user is authenticated
	authenticated, _ := c.Get("authenticated")
	isAuthenticated := authenticated == true

	// For anonymous users, just return success (preferences stored locally)
	if !isAuthenticated {
		// Parse the request to validate it but don't save to database
		var updateReq struct {
			BeginnerMode         *bool    `json:"beginner_mode"`
			ShowTooltips         *bool    `json:"show_tooltips"`
			TooltipDelay         *int     `json:"tooltip_delay"`
			PreferredSports      []string `json:"preferred_sports"`
			AISuggestionsEnabled *bool    `json:"ai_suggestions_enabled"`
		}

		// Bind JSON request for validation
		if err := c.ShouldBindJSON(&updateReq); err != nil {
			utils.SendValidationError(c, "Invalid request body", err.Error())
			return
		}

		// Validate tooltip delay if provided
		if updateReq.TooltipDelay != nil {
			if *updateReq.TooltipDelay < 0 || *updateReq.TooltipDelay > 5000 {
				utils.SendValidationError(c, "Invalid tooltip delay", "Tooltip delay must be between 0 and 5000 milliseconds")
				return
			}
		}

		// Validate sports if provided
		if updateReq.PreferredSports != nil {
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
		}

		// Return success for anonymous users (preferences handled by frontend)
		utils.SendSuccess(c, map[string]interface{}{
			"message":   "Preferences updated locally",
			"anonymous": true,
		})
		return
	}

	// For authenticated users, get user ID from auth context
	userIDValue, exists := c.Get("user_id")
	if !exists {
		utils.SendInternalError(c, "User ID not found in auth context")
		return
	}

	// Convert user ID to uint
	var userID uint
	switch v := userIDValue.(type) {
	case uint:
		userID = v
	case int:
		userID = uint(v)
	case string:
		var err error
		parsed, err := strconv.Atoi(v)
		if err != nil {
			utils.SendInternalError(c, "Invalid user ID format")
			return
		}
		userID = uint(parsed)
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
	// Check if user is authenticated
	authenticated, _ := c.Get("authenticated")
	isAuthenticated := authenticated == true

	// For anonymous users, return default preferences
	if !isAuthenticated {
		defaultPrefs := &models.UserPreferences{
			UserID:               0, // 0 indicates anonymous user
			BeginnerMode:         true,
			ShowTooltips:         true,
			TooltipDelay:         500,
			PreferredSports:      []string{},
			AISuggestionsEnabled: true,
		}
		utils.SendSuccess(c, defaultPrefs)
		return
	}

	// For authenticated users, get user ID from auth context
	userIDValue, exists := c.Get("user_id")
	if !exists {
		utils.SendInternalError(c, "User ID not found in auth context")
		return
	}

	// Convert user ID to uint
	var userID uint
	switch v := userIDValue.(type) {
	case uint:
		userID = v
	case int:
		userID = uint(v)
	case string:
		var err error
		parsed, err := strconv.Atoi(v)
		if err != nil {
			utils.SendInternalError(c, "Invalid user ID format")
			return
		}
		userID = uint(parsed)
	default:
		utils.SendInternalError(c, "Invalid user ID type")
		return
	}

	// Reset to default values
	defaultPrefs := map[string]interface{}{
		"beginner_mode":          true,
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

// MigratePreferences migrates anonymous preferences to authenticated user during signup
// POST /api/user/preferences/migrate
func (h *PreferencesHandler) MigratePreferences(c *gin.Context) {
	// This endpoint requires authentication
	userIDValue, exists := c.Get("user_id")
	if !exists {
		utils.SendUnauthorized(c, "Authentication required for preference migration")
		return
	}

	// Convert user ID to uint
	var userID uint
	switch v := userIDValue.(type) {
	case uint:
		userID = v
	case int:
		userID = uint(v)
	case string:
		var err error
		parsed, err := strconv.Atoi(v)
		if err != nil {
			utils.SendInternalError(c, "Invalid user ID format")
			return
		}
		userID = uint(parsed)
	default:
		utils.SendInternalError(c, "Invalid user ID type")
		return
	}

	// Define the migration request structure
	var migrationReq struct {
		BeginnerMode         bool     `json:"beginner_mode"`
		ShowTooltips         bool     `json:"show_tooltips"`
		TooltipDelay         int      `json:"tooltip_delay"`
		PreferredSports      []string `json:"preferred_sports"`
		AISuggestionsEnabled bool     `json:"ai_suggestions_enabled"`
	}

	// Bind JSON request
	if err := c.ShouldBindJSON(&migrationReq); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Validate tooltip delay
	if migrationReq.TooltipDelay < 0 || migrationReq.TooltipDelay > 5000 {
		utils.SendValidationError(c, "Invalid tooltip delay", "Tooltip delay must be between 0 and 5000 milliseconds")
		return
	}

	// Validate sports
	if migrationReq.PreferredSports != nil {
		validSports := []string{"nfl", "nba", "mlb", "nhl", "pga", "nascar", "mma", "soccer"}
		for _, sport := range migrationReq.PreferredSports {
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
	}

	// Create or update user preferences with migrated values
	updates := map[string]interface{}{
		"beginner_mode":          migrationReq.BeginnerMode,
		"show_tooltips":          migrationReq.ShowTooltips,
		"tooltip_delay":          migrationReq.TooltipDelay,
		"preferred_sports":       migrationReq.PreferredSports,
		"ai_suggestions_enabled": migrationReq.AISuggestionsEnabled,
	}

	// Update preferences with migrated values
	if err := models.UpdateUserPreferences(h.db, userID, updates); err != nil {
		utils.SendInternalError(c, "Failed to migrate preferences")
		return
	}

	// Return updated preferences
	prefs, err := models.GetUserPreferences(h.db, userID)
	if err != nil {
		utils.SendInternalError(c, "Failed to retrieve migrated preferences")
		return
	}

	utils.SendSuccess(c, prefs)
}
