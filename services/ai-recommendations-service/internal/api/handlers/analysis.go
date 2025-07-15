package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/models"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/services"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"gorm.io/gorm"
)

// AnalysisHandler handles lineup and contest analysis endpoints
type AnalysisHandler struct {
	db               *gorm.DB
	aiEngine         *services.AIEngine
	ownershipAnalyzer *services.OwnershipAnalyzer
	config           *config.Config
	logger           *logrus.Logger
}

// NewAnalysisHandler creates a new analysis handler
func NewAnalysisHandler(
	db *gorm.DB,
	aiEngine *services.AIEngine,
	ownershipAnalyzer *services.OwnershipAnalyzer,
	config *config.Config,
	logger *logrus.Logger,
) *AnalysisHandler {
	return &AnalysisHandler{
		db:               db,
		aiEngine:         aiEngine,
		ownershipAnalyzer: ownershipAnalyzer,
		config:           config,
		logger:           logger,
	}
}

// AnalyzeLineup provides comprehensive AI analysis of a lineup
func (h *AnalysisHandler) AnalyzeLineup(c *gin.Context) {
	var request struct {
		ContestID    int                           `json:"contest_id" binding:"required"`
		Lineup       []models.PlayerRecommendation `json:"lineup" binding:"required"`
		AnalysisType string                        `json:"analysis_type"` // "full", "quick", "ownership_focused"
		Context      models.PromptContext          `json:"context"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.WithError(err).Warn("Invalid lineup analysis request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Set default analysis type
	if request.AnalysisType == "" {
		request.AnalysisType = "full"
	}

	h.logger.WithFields(logrus.Fields{
		"contest_id":    request.ContestID,
		"lineup_size":   len(request.Lineup),
		"analysis_type": request.AnalysisType,
		"user_id":       c.GetString("user_id"),
	}).Info("Processing lineup analysis request")

	// Build analysis request
	analysisRequest := &services.LineupAnalysisRequest{
		ContestID:    uint(request.ContestID),
		UserID:       h.getUserID(c),
		Lineup:       request.Lineup,
		Context:      request.Context,
		AnalysisType: request.AnalysisType,
	}

	// Generate analysis
	response, err := h.aiEngine.AnalyzeLineup(c.Request.Context(), analysisRequest)
	if err != nil {
		h.logger.WithError(err).Error("Failed to analyze lineup")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to analyze lineup", "details": err.Error()})
		return
	}

	// Enhance with ownership analysis if requested
	if request.AnalysisType == "ownership_focused" || request.AnalysisType == "full" {
		ownershipInsights, err := h.ownershipAnalyzer.GetOwnershipInsights(uint(request.ContestID))
		if err != nil {
			h.logger.WithError(err).Warn("Failed to get ownership insights for lineup analysis")
		} else {
			// Add ownership context to response
			response.OwnershipProfile = h.analyzeLineupOwnership(request.Lineup, ownershipInsights)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
		"meta": gin.H{
			"analysis_type": request.AnalysisType,
			"lineup_size":   len(request.Lineup),
		},
	})
}

// AnalyzeContest provides AI-powered contest analysis and strategy insights
func (h *AnalysisHandler) AnalyzeContest(c *gin.Context) {
	contestIDParam := c.Param("contest_id")
	contestID, err := strconv.Atoi(contestIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contest ID"})
		return
	}

	var request struct {
		Sport       string `json:"sport"`
		ContestType string `json:"contest_type"`
		Strategy    string `json:"strategy"` // "cash", "gpp", "tournament"
		Focus       string `json:"focus"`    // "ownership", "leverage", "value", "correlation"
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		// Set defaults if no body provided
		request.Strategy = "gpp"
		request.Focus = "leverage"
	}

	h.logger.WithFields(logrus.Fields{
		"contest_id": contestID,
		"strategy":   request.Strategy,
		"focus":      request.Focus,
	}).Info("Processing contest analysis request")

	// Get contest metadata
	contestMeta, err := h.getContestMetadata(uint(contestID))
	if err != nil {
		h.logger.WithError(err).Error("Failed to get contest metadata")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get contest information"})
		return
	}

	// Get ownership analysis
	ownershipAnalysis, err := h.ownershipAnalyzer.GetOwnershipInsights(uint(contestID))
	if err != nil {
		h.logger.WithError(err).Warn("Failed to get ownership analysis")
		ownershipAnalysis = &models.OwnershipAnalysis{} // Empty analysis
	}

	// Get available players for the contest
	players, err := h.getContestPlayers(uint(contestID))
	if err != nil {
		h.logger.WithError(err).Error("Failed to get contest players")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get contest players"})
		return
	}

	// Calculate leverage opportunities
	leverageOpportunities, err := h.ownershipAnalyzer.CalculateLeverageOpportunities(
		uint(contestID),
		players,
		request.Strategy,
		[]models.LineupReference{}, // No existing lineups for contest analysis
	)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to calculate leverage opportunities")
		leverageOpportunities = []services.LeveragePlay{}
	}

	// Build comprehensive contest analysis
	analysis := gin.H{
		"contest_id":      contestID,
		"contest_meta":    contestMeta,
		"ownership_analysis": ownershipAnalysis,
		"leverage_opportunities": leverageOpportunities,
		"strategic_insights": h.generateStrategicInsights(contestMeta, ownershipAnalysis, request.Strategy),
		"key_players": h.identifyKeyPlayers(players, ownershipAnalysis),
		"market_inefficiencies": h.identifyMarketInefficiencies(players, ownershipAnalysis),
		"recommended_stacks": h.generateStackRecommendations(players, request.Sport),
		"timestamp": contestMeta.StartTime,
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   analysis,
		"meta": gin.H{
			"strategy":     request.Strategy,
			"focus":        request.Focus,
			"player_count": len(players),
		},
	})
}

// GetTrends provides trending insights for a specific sport
func (h *AnalysisHandler) GetTrends(c *gin.Context) {
	sport := c.Param("sport")
	if sport == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sport parameter is required"})
		return
	}

	timeframe := c.DefaultQuery("timeframe", "week") // week, month, season
	category := c.DefaultQuery("category", "all")    // ownership, performance, value

	h.logger.WithFields(logrus.Fields{
		"sport":     sport,
		"timeframe": timeframe,
		"category":  category,
	}).Info("Processing trends request")

	// Get trending data
	trends := h.getTrendingData(sport, timeframe, category)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"sport":     sport,
			"timeframe": timeframe,
			"trends":    trends,
			"generated_at": "now", // Would use actual timestamp
		},
		"meta": gin.H{
			"category": category,
			"count":    len(trends),
		},
	})
}

// Helper methods

func (h *AnalysisHandler) getUserID(c *gin.Context) uint {
	if userIDStr := c.GetString("user_id"); userIDStr != "" {
		if userID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			return uint(userID)
		}
	}
	return 0
}

func (h *AnalysisHandler) analyzeLineupOwnership(lineup []models.PlayerRecommendation, ownershipInsights *models.OwnershipAnalysis) string {
	if ownershipInsights == nil {
		return "Unknown - ownership data unavailable"
	}

	totalOwnership := 0.0
	highOwnershipCount := 0
	lowOwnershipCount := 0

	for _, player := range lineup {
		playerOwnership := h.getPlayerOwnership(player.PlayerID, ownershipInsights)
		totalOwnership += playerOwnership

		if playerOwnership > 20 {
			highOwnershipCount++
		} else if playerOwnership < 8 {
			lowOwnershipCount++
		}
	}

	avgOwnership := totalOwnership / float64(len(lineup))

	if avgOwnership > 25 {
		return "Chalk Heavy - high public ownership"
	} else if avgOwnership < 12 {
		return "Contrarian - low public ownership"
	} else if highOwnershipCount > lowOwnershipCount {
		return "Moderately Chalky - slightly above average ownership"
	} else if lowOwnershipCount > highOwnershipCount {
		return "Moderately Contrarian - slightly below average ownership"
	}

	return "Balanced - mixed ownership profile"
}

func (h *AnalysisHandler) getPlayerOwnership(playerID uint, ownershipInsights *models.OwnershipAnalysis) float64 {
	// Search through ownership data for this player
	for _, player := range ownershipInsights.HighOwnership {
		if player.PlayerID == playerID {
			return player.Ownership
		}
	}

	for _, player := range ownershipInsights.LowOwnership {
		if player.PlayerID == playerID {
			return player.Ownership
		}
	}

	return 15.0 // Default average ownership
}

func (h *AnalysisHandler) getContestMetadata(contestID uint) (*models.ContestMetadata, error) {
	// TODO: Implement actual database query to fetch contest metadata
	// Should query contests table with proper joins to get:
	// - Contest details (name, fees, prizes)
	// - Entry counts
	// - Start time and live status
	// - Payout structure information
	// Placeholder implementation
	return &models.ContestMetadata{
		ContestID:   contestID,
		ContestName: "Sample Contest",
		EntryFee:    25.0,
		TotalPrize:  100000.0,
		MaxEntries:  10000,
		CurrentEntries: 8500,
		SalaryCap:   50000,
		StartTime:   time.Now().Add(2 * time.Hour), // Would be actual contest start time
		IsLive:      true,
		PayoutStructure: "top_heavy",
	}, nil
}

func (h *AnalysisHandler) getContestPlayers(contestID uint) ([]models.PlayerRecommendation, error) {
	// This would query the database for players in the contest
	// Placeholder implementation
	var players []models.PlayerRecommendation
	
	// TODO: Implement actual database query
	// err := h.db.Where("contest_id = ?", contestID).Find(&players).Error
	
	return players, nil
}

func (h *AnalysisHandler) generateStrategicInsights(
	contestMeta *models.ContestMetadata,
	ownershipAnalysis *models.OwnershipAnalysis,
	strategy string,
) []string {
	insights := []string{}

	// Generate insights based on contest type and ownership
	if strategy == "gpp" {
		insights = append(insights, "Focus on low-owned players with high upside for tournament leverage")
		
		if len(ownershipAnalysis.ChalkPlays) > 5 {
			insights = append(insights, "High number of chalk plays - consider contrarian approach")
		}
		
		if contestMeta.CurrentEntries > int(float64(contestMeta.MaxEntries)*0.8) {
			insights = append(insights, "Contest is filling up - late entry advantage diminishing")
		}
	} else if strategy == "cash" {
		insights = append(insights, "Prioritize high-floor, consistent players for cash games")
		insights = append(insights, "Avoid highly volatile players - consistency over upside")
	}

	return insights
}

func (h *AnalysisHandler) identifyKeyPlayers(
	players []models.PlayerRecommendation,
	ownershipAnalysis *models.OwnershipAnalysis,
) []models.PlayerRecommendation {
	// Identify the most important players for the contest
	var keyPlayers []models.PlayerRecommendation
	
	// This would implement logic to identify key players based on:
	// - High projections
	// - Ownership levels
	// - Value metrics
	// - Correlation opportunities
	
	return keyPlayers[:min(10, len(keyPlayers))] // Return top 10
}

func (h *AnalysisHandler) identifyMarketInefficiencies(
	players []models.PlayerRecommendation,
	ownershipAnalysis *models.OwnershipAnalysis,
) []map[string]interface{} {
	var inefficiencies []map[string]interface{}
	
	// Identify players who are undervalued or overvalued by the market
	for _, player := range players {
		playerOwnership := h.getPlayerOwnership(player.PlayerID, ownershipAnalysis)
		
		// Simple value vs ownership analysis
		value := player.Projection / (player.Salary / 1000)
		
		if value > 3.5 && playerOwnership < 10 {
			inefficiencies = append(inefficiencies, map[string]interface{}{
				"type":        "undervalued",
				"player_id":   player.PlayerID,
				"player_name": player.PlayerName,
				"value":       value,
				"ownership":   playerOwnership,
				"reason":      "High value with low ownership",
			})
		} else if value < 2.5 && playerOwnership > 25 {
			inefficiencies = append(inefficiencies, map[string]interface{}{
				"type":        "overvalued",
				"player_id":   player.PlayerID,
				"player_name": player.PlayerName,
				"value":       value,
				"ownership":   playerOwnership,
				"reason":      "Low value with high ownership",
			})
		}
	}
	
	return inefficiencies
}

func (h *AnalysisHandler) generateStackRecommendations(
	players []models.PlayerRecommendation,
	sport string,
) []models.StackSuggestion {
	var stacks []models.StackSuggestion
	
	// Generate sport-specific stack recommendations
	switch sport {
	case "nfl":
		stacks = h.generateNFLStacks(players)
	case "golf":
		stacks = h.generateGolfStacks(players)
	default:
		// Generic stacking logic
	}
	
	return stacks
}

func (h *AnalysisHandler) generateNFLStacks(players []models.PlayerRecommendation) []models.StackSuggestion {
	// Implement NFL-specific stacking (QB+WR, game stacks, etc.)
	return []models.StackSuggestion{}
}

func (h *AnalysisHandler) generateGolfStacks(players []models.PlayerRecommendation) []models.StackSuggestion {
	// Implement golf-specific correlations (country, equipment, etc.)
	return []models.StackSuggestion{}
}

func (h *AnalysisHandler) getTrendingData(sport, timeframe, category string) []map[string]interface{} {
	// This would query historical data and identify trends
	// Placeholder implementation
	return []map[string]interface{}{
		{
			"trend_type": "ownership_shift",
			"description": "Low-priced players gaining popularity",
			"impact": "high",
			"timeframe": timeframe,
		},
		{
			"trend_type": "value_opportunity",
			"description": "Mid-tier players providing better value",
			"impact": "medium",
			"timeframe": timeframe,
		},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}