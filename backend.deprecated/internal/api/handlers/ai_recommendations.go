package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/utils"
)

// AIRecommendationHandler handles AI recommendation endpoints
type AIRecommendationHandler struct {
	aiService *services.AIRecommendationService
}

// NewAIRecommendationHandler creates a new AI recommendation handler
func NewAIRecommendationHandler(aiService *services.AIRecommendationService) *AIRecommendationHandler {
	return &AIRecommendationHandler{
		aiService: aiService,
	}
}

// RecommendPlayers handles POST /api/ai/recommend-players
func (h *AIRecommendationHandler) RecommendPlayers(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	// TODO: Re-enable authentication in production
	userID := 1 // Default user ID for development
	userIDStr, exists := c.Get("userID")
	if exists {
		var ok bool
		userID, ok = userIDStr.(int)
		if !ok {
			// Try converting from string
			if userIDString, isString := userIDStr.(string); isString {
				var err error
				userID, err = strconv.Atoi(userIDString)
				if err != nil {
					userID = 1 // Default to 1 if conversion fails
				}
			}
		}
	}

	var req services.PlayerRecommendationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Validate required fields
	if req.ContestID == 0 {
		utils.SendValidationError(c, "Contest ID is required", "")
		return
	}

	if req.Sport == "" {
		utils.SendValidationError(c, "Sport is required", "")
		return
	}

	if req.ContestType == "" {
		utils.SendValidationError(c, "Contest type is required", "")
		return
	}

	// Set default optimization goal if not provided
	if req.OptimizeFor == "" {
		if req.ContestType == "GPP" {
			req.OptimizeFor = "ceiling"
		} else {
			req.OptimizeFor = "floor"
		}
	}

	// Get recommendations
	recommendations, err := h.aiService.GetPlayerRecommendations(c.Request.Context(), userID, req)
	if err != nil {
		// Check if it's a rate limit error
		if err.Error() == "AI rate limit exceeded, please try again later" {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}

		utils.SendInternalError(c, "Failed to get recommendations: "+err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"recommendations": recommendations,
		"request": gin.H{
			"contest_id":       req.ContestID,
			"contest_type":     req.ContestType,
			"sport":            req.Sport,
			"remaining_budget": req.RemainingBudget,
			"optimize_for":     req.OptimizeFor,
			"positions_needed": req.PositionsNeeded,
		},
	})
}

// AnalyzeLineup handles POST /api/ai/analyze-lineup
func (h *AIRecommendationHandler) AnalyzeLineup(c *gin.Context) {
	// Get user ID from context
	// TODO: Re-enable authentication in production
	userID := 1 // Default user ID for development
	userIDStr, exists := c.Get("userID")
	if exists {
		var ok bool
		userID, ok = userIDStr.(int)
		if !ok {
			// Try converting from string
			if userIDString, isString := userIDStr.(string); isString {
				var err error
				userID, err = strconv.Atoi(userIDString)
				if err != nil {
					userID = 1 // Default to 1 if conversion fails
				}
			}
		}
	}

	var req services.LineupAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request body", err.Error())
		return
	}

	// Validate required fields
	if req.LineupID == 0 {
		utils.SendValidationError(c, "Lineup ID is required", "")
		return
	}

	if req.ContestType == "" {
		utils.SendValidationError(c, "Contest type is required", "")
		return
	}

	if req.Sport == "" {
		utils.SendValidationError(c, "Sport is required", "")
		return
	}

	// Get analysis
	analysis, err := h.aiService.AnalyzeLineup(c.Request.Context(), userID, req)
	if err != nil {
		// Check if it's a rate limit error
		if err.Error() == "AI rate limit exceeded, please try again later" {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}

		utils.SendInternalError(c, "Failed to analyze lineup: "+err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"analysis":  analysis,
		"lineup_id": req.LineupID,
	})
}

// GetRecommendationHistory handles GET /api/ai/recommendations/history
func (h *AIRecommendationHandler) GetRecommendationHistory(c *gin.Context) {
	// Get user ID from context
	// TODO: Re-enable authentication in production
	userID := 1 // Default user ID for development
	userIDStr, exists := c.Get("userID")
	if exists {
		var ok bool
		userID, ok = userIDStr.(int)
		if !ok {
			// Try converting from string
			if userIDString, isString := userIDStr.(string); isString {
				var err error
				userID, err = strconv.Atoi(userIDString)
				if err != nil {
					userID = 1 // Default to 1 if conversion fails
				}
			}
		}
	}

	// Get optional limit parameter
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Get recommendation history
	history, err := h.aiService.GetRecommendationHistory(userID, limit)
	if err != nil {
		utils.SendInternalError(c, "Failed to fetch recommendation history: "+err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"recommendations": history,
		"count":           len(history),
		"limit":           limit,
	})
}

// RegisterRoutes registers all AI recommendation routes
func (h *AIRecommendationHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Log that we're registering AI routes
	fmt.Println("DEBUG: Registering AI recommendation routes")

	ai := router.Group("/ai")
	{
		ai.POST("/recommend-players", h.RecommendPlayers)
		ai.POST("/analyze-lineup", h.AnalyzeLineup)
		ai.GET("/recommendations/history", h.GetRecommendationHistory)
	}

	fmt.Println("DEBUG: AI routes registered successfully")
}
