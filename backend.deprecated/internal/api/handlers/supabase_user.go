package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/api/middleware"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
)

// SupabaseUserHandler handles Supabase user profile operations
type SupabaseUserHandler struct {
	db           *database.DB
	supabaseUser *services.SupabaseUserService
}

// UpdateUserRequest represents user profile update request
type UpdateUserRequest struct {
	FirstName        *string `json:"first_name,omitempty"`
	LastName         *string `json:"last_name,omitempty"`
	SubscriptionTier *string `json:"subscription_tier,omitempty"`
}

// UpdatePreferencesRequest represents user preferences update request
type UpdatePreferencesRequest struct {
	SportPreferences       []string `json:"sport_preferences,omitempty"`
	PlatformPreferences    []string `json:"platform_preferences,omitempty"`
	ContestTypePreferences []string `json:"contest_type_preferences,omitempty"`
	Theme                  *string  `json:"theme,omitempty"`
	Language               *string  `json:"language,omitempty"`
	NotificationsEnabled   *bool    `json:"notifications_enabled,omitempty"`
	TutorialCompleted      *bool    `json:"tutorial_completed,omitempty"`
	BeginnerMode           *bool    `json:"beginner_mode,omitempty"`
	TooltipsEnabled        *bool    `json:"tooltips_enabled,omitempty"`
}

// NewSupabaseUserHandler creates a new Supabase user handler
func NewSupabaseUserHandler(db *database.DB, supabaseUser *services.SupabaseUserService) *SupabaseUserHandler {
	return &SupabaseUserHandler{
		db:           db,
		supabaseUser: supabaseUser,
	}
}

// RegisterRoutes registers all Supabase user routes
func (h *SupabaseUserHandler) RegisterRoutes(group *gin.RouterGroup, authMiddleware *middleware.SupabaseAuthMiddleware) {
	users := group.Group("/users")
	users.Use(authMiddleware.SupabaseAuthRequired())
	{
		users.GET("/me", h.GetCurrentUser)
		users.PUT("/me", h.UpdateUser)
		users.GET("/preferences", h.GetPreferences)
		users.PUT("/preferences", h.UpdatePreferences)
		users.POST("/preferences/reset", h.ResetPreferences)
		users.GET("/subscription-tiers", h.GetSubscriptionTiers)
		users.GET("/usage", h.GetUsageStats)
	}
}

// GetCurrentUser returns the current authenticated user with preferences
// GET /api/v1/users/me
func (h *SupabaseUserHandler) GetCurrentUser(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.SendUnauthorized(c, "User ID not found")
		return
	}

	user, err := models.GetSupabaseUserByID(h.db, userID)
	if err != nil {
		utils.SendInternalError(c, "Failed to get user profile")
		return
	}

	utils.SendSuccess(c, user)
}

// UpdateUser updates the current user's profile
// PUT /api/v1/users/me
func (h *SupabaseUserHandler) UpdateUser(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.SendUnauthorized(c, "User ID not found")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.FirstName != nil {
		updates["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updates["last_name"] = *req.LastName
	}
	if req.SubscriptionTier != nil {
		// Validate subscription tier exists
		_, err := models.GetSupabaseSubscriptionTierByName(h.db, *req.SubscriptionTier)
		if err != nil {
			utils.SendValidationError(c, "Invalid subscription tier", *req.SubscriptionTier)
			return
		}
		updates["subscription_tier"] = *req.SubscriptionTier
	}

	// Update user in database
	if len(updates) > 0 {
		if err := h.db.Model(&models.SupabaseUser{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
			utils.SendInternalError(c, "Failed to update user profile")
			return
		}
	}

	// Return updated user
	user, err := models.GetSupabaseUserByID(h.db, userID)
	if err != nil {
		utils.SendInternalError(c, "Failed to get updated user profile")
		return
	}

	utils.SendSuccess(c, user)
}

// GetPreferences returns the current user's preferences
// GET /api/v1/users/preferences
func (h *SupabaseUserHandler) GetPreferences(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.SendUnauthorized(c, "User ID not found")
		return
	}

	user, err := models.GetSupabaseUserByID(h.db, userID)
	if err != nil {
		utils.SendInternalError(c, "Failed to get user profile")
		return
	}

	// If preferences don't exist, create default ones
	if user.Preferences == nil {
		prefs, err := models.CreateSupabaseUserPreferences(h.db, userID)
		if err != nil {
			utils.SendInternalError(c, "Failed to create default preferences")
			return
		}
		utils.SendSuccess(c, prefs)
		return
	}

	utils.SendSuccess(c, user.Preferences)
}

// UpdatePreferences updates the current user's preferences with real-time sync
// PUT /api/v1/users/preferences
func (h *SupabaseUserHandler) UpdatePreferences(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.SendUnauthorized(c, "User ID not found")
		return
	}

	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Validate sport preferences if provided
	if req.SportPreferences != nil {
		validSports := []string{"nfl", "nba", "mlb", "nhl", "golf", "pga", "nascar", "mma", "soccer"}
		for _, sport := range req.SportPreferences {
			if !contains(validSports, sport) {
				utils.SendValidationError(c, "Invalid sport", sport)
				return
			}
		}
	}

	// Validate platform preferences if provided
	if req.PlatformPreferences != nil {
		validPlatforms := []string{"draftkings", "fanduel", "superdraft", "prizepicks"}
		for _, platform := range req.PlatformPreferences {
			if !contains(validPlatforms, platform) {
				utils.SendValidationError(c, "Invalid platform", platform)
				return
			}
		}
	}

	// Validate contest type preferences if provided
	if req.ContestTypePreferences != nil {
		validTypes := []string{"gpp", "cash", "tournament", "head2head", "multiplier"}
		for _, contestType := range req.ContestTypePreferences {
			if !contains(validTypes, contestType) {
				utils.SendValidationError(c, "Invalid contest type", contestType)
				return
			}
		}
	}

	// Validate theme if provided
	if req.Theme != nil {
		validThemes := []string{"light", "dark", "auto"}
		if !contains(validThemes, *req.Theme) {
			utils.SendValidationError(c, "Invalid theme", *req.Theme)
			return
		}
	}

	// Validate language if provided
	if req.Language != nil {
		validLanguages := []string{"en", "es", "fr", "de"}
		if !contains(validLanguages, *req.Language) {
			utils.SendValidationError(c, "Invalid language", *req.Language)
			return
		}
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.SportPreferences != nil {
		updates["sport_preferences"] = req.SportPreferences
	}
	if req.PlatformPreferences != nil {
		updates["platform_preferences"] = req.PlatformPreferences
	}
	if req.ContestTypePreferences != nil {
		updates["contest_type_preferences"] = req.ContestTypePreferences
	}
	if req.Theme != nil {
		updates["theme"] = *req.Theme
	}
	if req.Language != nil {
		updates["language"] = *req.Language
	}
	if req.NotificationsEnabled != nil {
		updates["notifications_enabled"] = *req.NotificationsEnabled
	}
	if req.TutorialCompleted != nil {
		updates["tutorial_completed"] = *req.TutorialCompleted
	}
	if req.BeginnerMode != nil {
		updates["beginner_mode"] = *req.BeginnerMode
	}
	if req.TooltipsEnabled != nil {
		updates["tooltips_enabled"] = *req.TooltipsEnabled
	}

	// Update preferences in database (will trigger real-time updates via Supabase)
	if len(updates) > 0 {
		if err := models.UpdateSupabaseUserPreferences(h.db, userID, updates); err != nil {
			utils.SendInternalError(c, "Failed to update preferences")
			return
		}
	}

	// Return updated preferences
	user, err := models.GetSupabaseUserByID(h.db, userID)
	if err != nil {
		utils.SendInternalError(c, "Failed to get updated preferences")
		return
	}

	utils.SendSuccess(c, user.Preferences)
}

// ResetPreferences resets the current user's preferences to defaults
// POST /api/v1/users/preferences/reset
func (h *SupabaseUserHandler) ResetPreferences(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.SendUnauthorized(c, "User ID not found")
		return
	}

	// Reset to default values
	defaultUpdates := map[string]interface{}{
		"sport_preferences":         []string{"nba", "nfl", "mlb", "golf"},
		"platform_preferences":     []string{"draftkings", "fanduel"},
		"contest_type_preferences": []string{"gpp", "cash"},
		"theme":                    "light",
		"language":                 "en",
		"notifications_enabled":    true,
		"tutorial_completed":       false,
		"beginner_mode":            true,
		"tooltips_enabled":         true,
	}

	// Update preferences
	if err := models.UpdateSupabaseUserPreferences(h.db, userID, defaultUpdates); err != nil {
		utils.SendInternalError(c, "Failed to reset preferences")
		return
	}

	// Return reset preferences
	user, err := models.GetSupabaseUserByID(h.db, userID)
	if err != nil {
		utils.SendInternalError(c, "Failed to get reset preferences")
		return
	}

	utils.SendSuccess(c, user.Preferences)
}

// GetSubscriptionTiers returns all available subscription tiers
// GET /api/v1/users/subscription-tiers
func (h *SupabaseUserHandler) GetSubscriptionTiers(c *gin.Context) {
	tiers, err := models.GetAllSupabaseSubscriptionTiers(h.db)
	if err != nil {
		utils.SendInternalError(c, "Failed to get subscription tiers")
		return
	}

	utils.SendSuccess(c, tiers)
}

// GetUsageStats returns the current user's usage statistics
// GET /api/v1/users/usage
func (h *SupabaseUserHandler) GetUsageStats(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.SendUnauthorized(c, "User ID not found")
		return
	}

	user, err := models.GetSupabaseUserByID(h.db, userID)
	if err != nil {
		utils.SendInternalError(c, "Failed to get user profile")
		return
	}

	// Get subscription tier limits
	tier, err := user.GetTier(h.db)
	if err != nil {
		utils.SendInternalError(c, "Failed to get subscription tier")
		return
	}

	// Reset usage if needed
	if err := user.ResetUsageIfNeeded(h.db); err != nil {
		utils.SendInternalError(c, "Failed to update usage counters")
		return
	}

	// Calculate remaining usage
	remainingOptimizations := tier.MonthlyOptimizations - user.MonthlyOptimizationsUsed
	if tier.MonthlyOptimizations == -1 {
		remainingOptimizations = -1 // Unlimited
	}

	remainingSimulations := tier.MonthlySimulations - user.MonthlySimulationsUsed
	if tier.MonthlySimulations == -1 {
		remainingSimulations = -1 // Unlimited
	}

	usageStats := gin.H{
		"subscription_tier":          user.SubscriptionTier,
		"optimizations_used":         user.MonthlyOptimizationsUsed,
		"optimizations_limit":        tier.MonthlyOptimizations,
		"optimizations_remaining":    remainingOptimizations,
		"simulations_used":           user.MonthlySimulationsUsed,
		"simulations_limit":          tier.MonthlySimulations,
		"simulations_remaining":      remainingSimulations,
		"usage_reset_date":           user.UsageResetDate,
		"can_optimize":               remainingOptimizations != 0,
		"can_simulate":               remainingSimulations != 0,
		"ai_recommendations_enabled": tier.AIRecommendations,
	}

	utils.SendSuccess(c, usageStats)
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}