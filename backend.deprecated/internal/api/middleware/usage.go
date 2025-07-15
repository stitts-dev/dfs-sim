package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
)

// UsageCheckType represents the type of usage to check
type UsageCheckType string

const (
	UsageOptimization UsageCheckType = "optimization"
	UsageSimulation   UsageCheckType = "simulation"
)

// CheckUsageLimit middleware checks if the user can perform the action based on their subscription tier
func CheckUsageLimit(db *database.DB, usageType UsageCheckType) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if user is authenticated
		authenticated, exists := c.Get("authenticated")
		if !exists || authenticated != true {
			// For unauthenticated users, use a default user (development mode)
			// In production, this would require authentication
			userID := uint(1) // Default admin user
			
			user, err := models.GetUserByID(db, userID)
			if err != nil {
				utils.SendInternalError(c, "Failed to get user")
				c.Abort()
				return
			}
			
			// Set user in context for handlers to use
			c.Set("user", user)
			c.Set("user_id", user.ID)
		} else {
			// Get authenticated user
			userIDValue, exists := c.Get("user_id")
			if !exists {
				utils.SendUnauthorized(c, "User ID not found")
				c.Abort()
				return
			}
			
			var userID uint
			switch v := userIDValue.(type) {
			case uint:
				userID = v
			case int:
				userID = uint(v)
			default:
				utils.SendInternalError(c, "Invalid user ID type")
				c.Abort()
				return
			}
			
			user, err := models.GetUserByID(db, userID)
			if err != nil {
				utils.SendInternalError(c, "Failed to get user")
				c.Abort()
				return
			}
			
			// Set user in context
			c.Set("user", user)
		}
		
		// Get user from context
		userValue, _ := c.Get("user")
		user := userValue.(*models.User)
		
		// Check if user can perform the action
		var canPerform bool
		var err error
		
		switch usageType {
		case UsageOptimization:
			canPerform, err = user.CanOptimize(db)
		case UsageSimulation:
			canPerform, err = user.CanSimulate(db)
		default:
			utils.SendInternalError(c, "Invalid usage type")
			c.Abort()
			return
		}
		
		if err != nil {
			utils.SendInternalError(c, "Failed to check usage limits")
			c.Abort()
			return
		}
		
		if !canPerform {
			// Get tier info for error message
			tier, err := user.GetTier(db)
			if err != nil {
				utils.SendInternalError(c, "Failed to get subscription tier")
				c.Abort()
				return
			}
			
			var limitMessage string
			switch usageType {
			case UsageOptimization:
				if tier.MonthlyOptimizations == -1 {
					limitMessage = "unlimited optimizations"
				} else {
					limitMessage = fmt.Sprintf("%d optimizations per month", tier.MonthlyOptimizations)
				}
			case UsageSimulation:
				if tier.MonthlySimulations == -1 {
					limitMessage = "unlimited simulations"
				} else {
					limitMessage = fmt.Sprintf("%d simulations per month", tier.MonthlySimulations)
				}
			}
			
			utils.SendValidationError(c, "Usage limit exceeded",
				fmt.Sprintf("Your %s tier allows %s. Upgrade to continue.", 
					tier.Name, limitMessage))
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// IncrementUsage middleware increments the usage counter after successful action
func IncrementUsage(db *database.DB, usageType UsageCheckType) gin.HandlerFunc {
	return func(c *gin.Context) {
		// This should be called after the handler completes successfully
		c.Next()
		
		// Only increment if the response was successful (2xx status codes)
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			userValue, exists := c.Get("user")
			if !exists {
				return // No user to increment for
			}
			
			user := userValue.(*models.User)
			
			switch usageType {
			case UsageOptimization:
				user.IncrementOptimizationUsage(db)
			case UsageSimulation:
				user.IncrementSimulationUsage(db)
			}
		}
	}
}

// OptionalUsageCheck is a lighter version that doesn't block but still tracks usage
func OptionalUsageCheck(db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get authenticated user
		userIDValue, exists := c.Get("user_id")
		if exists {
			var userID uint
			switch v := userIDValue.(type) {
			case uint:
				userID = v
			case int:
				userID = uint(v)
			default:
				c.Next()
				return
			}
			
			user, err := models.GetUserByID(db, userID)
			if err == nil {
				c.Set("user", user)
			}
		} else {
			// Use default user for development
			user, err := models.GetUserByID(db, 1)
			if err == nil {
				c.Set("user", user)
				c.Set("user_id", user.ID)
			}
		}
		
		c.Next()
	}
}