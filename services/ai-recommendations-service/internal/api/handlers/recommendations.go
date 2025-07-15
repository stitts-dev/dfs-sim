package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/models"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/services"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/websocket"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"gorm.io/gorm"
)

// RecommendationHandler handles AI recommendation endpoints
type RecommendationHandler struct {
	db        *gorm.DB
	aiEngine  *services.AIEngine
	wsHub     *websocket.RecommendationHub
	config    *config.Config
	logger    *logrus.Logger
}

// NewRecommendationHandler creates a new recommendation handler
func NewRecommendationHandler(
	db *gorm.DB,
	aiEngine *services.AIEngine,
	wsHub *websocket.RecommendationHub,
	config *config.Config,
	logger *logrus.Logger,
) *RecommendationHandler {
	return &RecommendationHandler{
		db:       db,
		aiEngine: aiEngine,
		wsHub:    wsHub,
		config:   config,
		logger:   logger,
	}
}

// GetPlayerRecommendations generates AI-powered player recommendations
func (h *RecommendationHandler) GetPlayerRecommendations(c *gin.Context) {
	var request models.SmartRecommendationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.WithError(err).Warn("Invalid recommendation request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Validate request
	if err := h.validateRecommendationRequest(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"contest_id":   request.ContestID,
		"sport":        request.Sport,
		"contest_type": request.ContestType,
		"user_id":      c.GetString("user_id"), // Assume middleware sets this
	}).Info("Processing player recommendation request")

	// Build AI engine request
	aiRequest := &services.RecommendationRequest{
		ContestID:           uint(request.ContestID),
		UserID:              h.getUserID(c),
		Players:             h.convertToPlayerRecommendations(request),
		Context:             h.buildPromptContext(request),
		RequestType:         "player_recommendations",
		IncludeRealTimeData: request.IncludeRealTimeData,
		IncludeLeverageAnalysis: true,
		MaxRecommendations:  request.MaxRecommendations,
		CacheResults:        true,
	}

	// Generate recommendations
	response, err := h.aiEngine.GenerateRecommendations(c.Request.Context(), aiRequest)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate AI recommendations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate recommendations", "details": err.Error()})
		return
	}

	// Save recommendation to database
	h.saveRecommendationToDatabase(aiRequest, response)

	// Broadcast to WebSocket clients if user is connected
	if userID := h.getUserIDString(c); userID != "" {
		update := &models.RecommendationUpdate{
			Type:      "recommendation",
			UserID:    userID,
			Data:      response,
			Timestamp: time.Now(),
		}
		h.wsHub.BroadcastInsight(userID, update)
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
		"meta": gin.H{
			"request_id":         response.RequestID,
			"processing_time_ms": response.ProcessingTimeMs,
			"cache_hit":          response.CacheHit,
		},
	})
}

// GetLineupRecommendations provides lineup-level AI recommendations
func (h *RecommendationHandler) GetLineupRecommendations(c *gin.Context) {
	var request struct {
		ContestID       int                              `json:"contest_id" binding:"required"`
		Sport           string                           `json:"sport" binding:"required"`
		ContestType     string                           `json:"contest_type" binding:"required"`
		CurrentLineup   []models.PlayerRecommendation    `json:"current_lineup"`
		PositionsNeeded []string                         `json:"positions_needed"`
		Budget          float64                          `json:"remaining_budget"`
		Strategy        string                           `json:"strategy"`
		Context         models.PromptContext             `json:"context"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"contest_id":        request.ContestID,
		"current_lineup":    len(request.CurrentLineup),
		"positions_needed":  len(request.PositionsNeeded),
		"remaining_budget":  request.Budget,
	}).Info("Processing lineup recommendation request")

	// Build AI request for lineup optimization
	aiRequest := &services.RecommendationRequest{
		ContestID:           uint(request.ContestID),
		UserID:              h.getUserID(c),
		Players:             request.CurrentLineup,
		Context:             request.Context,
		RequestType:         "lineup_optimization",
		IncludeRealTimeData: true,
		IncludeLeverageAnalysis: true,
		MaxRecommendations:  10,
		CacheResults:        true,
	}

	// Enhance context with lineup-specific data
	aiRequest.Context.ContestType = request.ContestType
	aiRequest.Context.Sport = request.Sport

	response, err := h.aiEngine.GenerateRecommendations(c.Request.Context(), aiRequest)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate lineup recommendations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate lineup recommendations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
		"meta": gin.H{
			"request_id":         response.RequestID,
			"processing_time_ms": response.ProcessingTimeMs,
		},
	})
}

// GetSwapRecommendations provides late-swap AI recommendations
func (h *RecommendationHandler) GetSwapRecommendations(c *gin.Context) {
	var request struct {
		ContestID      int                           `json:"contest_id" binding:"required"`
		CurrentLineup  []models.PlayerRecommendation `json:"current_lineup" binding:"required"`
		TimeToLock     string                        `json:"time_to_lock"`
		SwapBudget     float64                       `json:"swap_budget"`
		MaxSwaps       int                           `json:"max_swaps"`
		Priority       string                        `json:"priority"` // "safety", "upside", "leverage"
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"contest_id":   request.ContestID,
		"lineup_size":  len(request.CurrentLineup),
		"time_to_lock": request.TimeToLock,
		"priority":     request.Priority,
	}).Info("Processing late swap recommendation request")

	// Parse time to lock
	timeToLock, err := time.ParseDuration(request.TimeToLock)
	if err != nil {
		timeToLock = 30 * time.Minute // Default
	}

	// Build context for late swap
	context := models.PromptContext{
		Sport:             h.inferSportFromLineup(request.CurrentLineup),
		ContestType:       "gpp", // Assume GPP for late swaps
		OptimizationGoal:  request.Priority,
		TimeToLock:        timeToLock,
		OwnershipStrategy: "balanced",
		RiskTolerance:     "medium",
	}

	aiRequest := &services.RecommendationRequest{
		ContestID:           uint(request.ContestID),
		UserID:              h.getUserID(c),
		Players:             request.CurrentLineup,
		Context:             context,
		RequestType:         "late_swap",
		IncludeRealTimeData: true,
		IncludeLeverageAnalysis: true,
		MaxRecommendations:  request.MaxSwaps,
		CacheResults:        false, // Don't cache late swap recommendations
	}

	response, err := h.aiEngine.GenerateRecommendations(c.Request.Context(), aiRequest)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate swap recommendations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate swap recommendations"})
		return
	}

	// Broadcast urgent late swap alert to user
	if userID := h.getUserIDString(c); userID != "" {
		h.wsHub.BroadcastLateSwapAlert(uint(request.ContestID), response)
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
		"meta": gin.H{
			"urgency":            "high",
			"time_to_lock":       request.TimeToLock,
			"processing_time_ms": response.ProcessingTimeMs,
		},
	})
}

// Helper methods

func (h *RecommendationHandler) validateRecommendationRequest(request *models.SmartRecommendationRequest) error {
	if request.ContestID <= 0 {
		return fmt.Errorf("invalid contest ID")
	}

	if request.Sport == "" {
		return fmt.Errorf("sport is required")
	}

	if request.ContestType == "" {
		return fmt.Errorf("contest type is required")
	}

	// Set defaults
	if request.MaxRecommendations == 0 {
		request.MaxRecommendations = 10
	}

	if request.RiskTolerance == "" {
		request.RiskTolerance = "medium"
	}

	if request.OptimizeFor == "" {
		request.OptimizeFor = "roi"
	}

	return nil
}

func (h *RecommendationHandler) getUserID(c *gin.Context) uint {
	// Extract user ID from JWT token or session
	if userIDStr := c.GetString("user_id"); userIDStr != "" {
		if userID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			return uint(userID)
		}
	}
	return 0 // Anonymous user
}

func (h *RecommendationHandler) getUserIDString(c *gin.Context) string {
	return c.GetString("user_id")
}

func (h *RecommendationHandler) convertToPlayerRecommendations(request models.SmartRecommendationRequest) []models.PlayerRecommendation {
	// This would typically fetch player data from the database based on the request
	// For now, return empty slice - would be populated from actual player data
	var players []models.PlayerRecommendation
	
	// TODO: Query database for players in this contest
	// Example implementation:
	// var dbPlayers []models.Player
	// h.db.Where("contest_id = ?", request.ContestID).Find(&dbPlayers)
	// for _, player := range dbPlayers {
	//     players = append(players, models.PlayerRecommendation{
	//         PlayerID: player.ID,
	//         PlayerName: player.Name,
	//         Position: player.Position,
	//         Salary: player.Salary,
	//         Projection: player.ProjectedPoints,
	//     })
	// }
	
	return players
}

func (h *RecommendationHandler) buildPromptContext(request models.SmartRecommendationRequest) models.PromptContext {
	timeToLock := 2 * time.Hour // Default
	if request.TimeToLock != "" {
		if duration, err := time.ParseDuration(request.TimeToLock); err == nil {
			timeToLock = duration
		}
	}

	return models.PromptContext{
		Sport:             request.Sport,
		ContestType:       request.ContestType,
		OptimizationGoal:  request.OptimizeFor,
		TimeToLock:        timeToLock,
		OwnershipStrategy: request.OwnershipStrategy,
		RiskTolerance:     request.RiskTolerance,
		// Would populate other fields from database/user profile
	}
}

func (h *RecommendationHandler) inferSportFromLineup(lineup []models.PlayerRecommendation) string {
	if len(lineup) > 0 {
		// Simple sport inference based on position
		for _, player := range lineup {
			switch player.Position {
			case "QB", "RB", "WR", "TE", "K", "DST":
				return "nfl"
			case "PG", "SG", "SF", "PF", "C":
				return "nba"
			case "G":
				return "golf"
			}
		}
	}
	return "unknown"
}

func (h *RecommendationHandler) saveRecommendationToDatabase(request *services.RecommendationRequest, response *services.RecommendationResponse) {
	// Save to ai_recommendations table
	recommendation := models.AIRecommendation{
		UserID:         request.UserID,
		ContestID:      request.ContestID,
		Request:        h.marshalToJSON(request),
		Response:       h.marshalToJSON(response),
		ModelUsed:      response.ModelUsed,
		Confidence:     response.Confidence,
		TokensUsed:     &response.TokensUsed,
		ResponseTimeMs: &int(response.ProcessingTimeMs),
	}

	if err := h.db.Create(&recommendation).Error; err != nil {
		h.logger.WithError(err).Error("Failed to save recommendation to database")
	}
}

func (h *RecommendationHandler) marshalToJSON(data interface{}) json.RawMessage {
	bytes, err := json.Marshal(data)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal data to JSON")
		return json.RawMessage("{}")
	}
	return json.RawMessage(bytes)
}