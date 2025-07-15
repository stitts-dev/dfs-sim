package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
)

type SubscriptionHandler struct {
	db    *database.DB
	cache *services.CacheService
}

type UsageResponse struct {
	User                     *UserInfo                `json:"user"`
	CurrentTier              *models.SubscriptionTier `json:"current_tier"`
	Usage                    *UsageStats              `json:"usage"`
	AvailableTiers           []models.SubscriptionTier `json:"available_tiers"`
	NextBillingDate          *time.Time               `json:"next_billing_date,omitempty"`
	DaysUntilReset           int                      `json:"days_until_reset"`
}

type UserInfo struct {
	ID              uint    `json:"id"`
	PhoneNumber     string  `json:"phone_number"`
	Email           *string `json:"email,omitempty"`
	FirstName       *string `json:"first_name,omitempty"`
	LastName        *string `json:"last_name,omitempty"`
	SubscriptionTier string  `json:"subscription_tier"`
	IsActive        bool    `json:"is_active"`
}

type UsageStats struct {
	OptimizationsUsed    int  `json:"optimizations_used"`
	OptimizationsLimit   int  `json:"optimizations_limit"` // -1 = unlimited
	SimulationsUsed      int  `json:"simulations_used"`
	SimulationsLimit     int  `json:"simulations_limit"`   // -1 = unlimited
	CanOptimize          bool `json:"can_optimize"`
	CanSimulate          bool `json:"can_simulate"`
	UsageResetDate       time.Time `json:"usage_reset_date"`
}

func NewSubscriptionHandler(db *database.DB, cache *services.CacheService) *SubscriptionHandler {
	return &SubscriptionHandler{
		db:    db,
		cache: cache,
	}
}

// RegisterRoutes registers subscription-related routes
func (h *SubscriptionHandler) RegisterRoutes(group *gin.RouterGroup) {
	sub := group.Group("/subscription")
	{
		sub.GET("/usage", h.GetUsageStatus)
		sub.GET("/tiers", h.GetSubscriptionTiers)
		sub.POST("/upgrade", h.UpgradeSubscription)
		sub.POST("/cancel", h.CancelSubscription)
	}
}

// GetUsageStatus returns the current user's usage statistics and subscription info
// GET /api/v1/subscription/usage
func (h *SubscriptionHandler) GetUsageStatus(c *gin.Context) {
	// Get user from context (set by middleware)
	userValue, exists := c.Get("user")
	if !exists {
		utils.SendUnauthorized(c, "User not found")
		return
	}
	
	user := userValue.(*models.User)
	
	// Get current tier
	tier, err := user.GetTier(h.db)
	if err != nil {
		utils.SendInternalError(c, "Failed to get subscription tier")
		return
	}
	
	// Check current usage capabilities
	canOptimize, err := user.CanOptimize(h.db)
	if err != nil {
		utils.SendInternalError(c, "Failed to check optimization limits")
		return
	}
	
	canSimulate, err := user.CanSimulate(h.db)
	if err != nil {
		utils.SendInternalError(c, "Failed to check simulation limits")
		return
	}
	
	// Get all available tiers
	availableTiers, err := models.GetAllSubscriptionTiers(h.db)
	if err != nil {
		utils.SendInternalError(c, "Failed to get subscription tiers")
		return
	}
	
	// Calculate days until usage reset
	now := time.Now()
	nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	daysUntilReset := int(nextMonth.Sub(now).Hours() / 24)
	
	response := &UsageResponse{
		User: &UserInfo{
			ID:               user.ID,
			PhoneNumber:      user.PhoneNumber,
			Email:            user.Email,
			FirstName:        user.FirstName,
			LastName:         user.LastName,
			SubscriptionTier: user.SubscriptionTier,
			IsActive:         user.IsActive,
		},
		CurrentTier: tier,
		Usage: &UsageStats{
			OptimizationsUsed:  user.MonthlyOptimizationsUsed,
			OptimizationsLimit: tier.MonthlyOptimizations,
			SimulationsUsed:    user.MonthlySimulationsUsed,
			SimulationsLimit:   tier.MonthlySimulations,
			CanOptimize:        canOptimize,
			CanSimulate:        canSimulate,
			UsageResetDate:     user.UsageResetDate,
		},
		AvailableTiers:   availableTiers,
		NextBillingDate:  user.SubscriptionExpiresAt,
		DaysUntilReset:   daysUntilReset,
	}
	
	utils.SendSuccess(c, response)
}

// GetSubscriptionTiers returns all available subscription tiers
// GET /api/v1/subscription/tiers
func (h *SubscriptionHandler) GetSubscriptionTiers(c *gin.Context) {
	tiers, err := models.GetAllSubscriptionTiers(h.db)
	if err != nil {
		utils.SendInternalError(c, "Failed to get subscription tiers")
		return
	}
	
	utils.SendSuccess(c, tiers)
}

// UpgradeSubscription handles subscription upgrades
// POST /api/v1/subscription/upgrade
func (h *SubscriptionHandler) UpgradeSubscription(c *gin.Context) {
	userValue, exists := c.Get("user")
	if !exists {
		utils.SendUnauthorized(c, "User not found")
		return
	}
	
	user := userValue.(*models.User)
	
	var req struct {
		TierName      string `json:"tier_name" binding:"required"`
		PaymentMethod string `json:"payment_method"` // "card", "crypto", etc.
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}
	
	// Validate tier exists
	tier, err := models.GetSubscriptionTierByName(h.db, req.TierName)
	if err != nil {
		utils.SendValidationError(c, "Invalid subscription tier", "")
		return
	}
	
	// Check if it's actually an upgrade
	currentTier, err := user.GetTier(h.db)
	if err != nil {
		utils.SendInternalError(c, "Failed to get current tier")
		return
	}
	
	if tier.PriceCents <= currentTier.PriceCents {
		utils.SendValidationError(c, "Invalid upgrade", "Can only upgrade to a higher tier")
		return
	}
	
	// TODO: Process payment here
	// For now, just simulate successful upgrade
	
	// Update user's subscription
	user.SubscriptionTier = tier.Name
	if tier.PriceCents > 0 {
		// Set expiry to one month from now for paid tiers
		expiry := time.Now().AddDate(0, 1, 0)
		user.SubscriptionExpiresAt = &expiry
	}
	
	if err := h.db.Save(user).Error; err != nil {
		utils.SendInternalError(c, "Failed to update subscription")
		return
	}
	
	utils.SendSuccess(c, gin.H{
		"message":           "Subscription upgraded successfully",
		"new_tier":          tier.Name,
		"monthly_price":     tier.PriceCents,
		"expires_at":        user.SubscriptionExpiresAt,
	})
}

// CancelSubscription handles subscription cancellation
// POST /api/v1/subscription/cancel
func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	userValue, exists := c.Get("user")
	if !exists {
		utils.SendUnauthorized(c, "User not found")
		return
	}
	
	user := userValue.(*models.User)
	
	// Check if user has a paid subscription
	if user.SubscriptionTier == "free" {
		utils.SendValidationError(c, "No active subscription", "User is already on free tier")
		return
	}
	
	var req struct {
		CancelImmediately bool   `json:"cancel_immediately"`
		Reason           string `json:"reason,omitempty"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body for simple cancellation
		req.CancelImmediately = false
	}
	
	if req.CancelImmediately {
		// Cancel immediately - downgrade to free tier
		user.SubscriptionTier = "free"
		user.SubscriptionStatus = "cancelled"
		user.SubscriptionExpiresAt = nil
	} else {
		// Cancel at end of billing period
		user.SubscriptionStatus = "cancelled"
		// Keep current tier until expiry
	}
	
	if err := h.db.Save(user).Error; err != nil {
		utils.SendInternalError(c, "Failed to cancel subscription")
		return
	}
	
	message := "Subscription cancelled successfully"
	if !req.CancelImmediately && user.SubscriptionExpiresAt != nil {
		message = "Subscription will be cancelled at the end of the billing period"
	}
	
	utils.SendSuccess(c, gin.H{
		"message":    message,
		"expires_at": user.SubscriptionExpiresAt,
		"new_tier":   user.SubscriptionTier,
	})
}