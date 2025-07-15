package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/models"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/services"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
)

// OwnershipHandler handles ownership intelligence endpoints
type OwnershipHandler struct {
	ownershipAnalyzer *services.OwnershipAnalyzer
	config            *config.Config
	logger            *logrus.Logger
}

// NewOwnershipHandler creates a new ownership handler
func NewOwnershipHandler(
	ownershipAnalyzer *services.OwnershipAnalyzer,
	config *config.Config,
	logger *logrus.Logger,
) *OwnershipHandler {
	return &OwnershipHandler{
		ownershipAnalyzer: ownershipAnalyzer,
		config:            config,
		logger:            logger,
	}
}

// GetOwnershipData returns current ownership data for a contest
func (h *OwnershipHandler) GetOwnershipData(c *gin.Context) {
	contestIDParam := c.Param("contestId")
	contestID, err := strconv.ParseUint(contestIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contest ID"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"contest_id": contestID,
		"user_id":    c.GetString("user_id"),
	}).Info("Processing ownership data request")

	// Get ownership insights
	ownershipAnalysis, err := h.ownershipAnalyzer.GetOwnershipInsights(uint(contestID))
	if err != nil {
		h.logger.WithError(err).Error("Failed to get ownership insights")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get ownership data", "details": err.Error()})
		return
	}

	// Get additional ownership metadata
	metadata := h.getOwnershipMetadata(uint(contestID))

	response := gin.H{
		"contest_id": contestID,
		"ownership_analysis": ownershipAnalysis,
		"metadata": metadata,
		"generated_at": "now", // Would use actual timestamp
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
	})
}

// GetLeverageOpportunities returns leverage opportunities for a contest
func (h *OwnershipHandler) GetLeverageOpportunities(c *gin.Context) {
	contestIDParam := c.Param("contestId")
	contestID, err := strconv.ParseUint(contestIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contest ID"})
		return
	}

	// Parse query parameters
	contestType := c.DefaultQuery("contest_type", "gpp")
	sport := c.DefaultQuery("sport", "")
	minLeverageScore := c.DefaultQuery("min_leverage_score", "0.3")
	maxResults := c.DefaultQuery("max_results", "20")

	minScore, _ := strconv.ParseFloat(minLeverageScore, 64)
	maxRes, _ := strconv.Atoi(maxResults)

	h.logger.WithFields(logrus.Fields{
		"contest_id":         contestID,
		"contest_type":       contestType,
		"min_leverage_score": minScore,
		"max_results":        maxRes,
	}).Info("Processing leverage opportunities request")

	// Get contest players
	players, err := h.getContestPlayersForLeverage(uint(contestID), sport)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get contest players")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get contest players"})
		return
	}

	// Calculate leverage opportunities
	leverageOpportunities, err := h.ownershipAnalyzer.CalculateLeverageOpportunities(
		uint(contestID),
		players,
		contestType,
		[]models.LineupReference{}, // No existing lineups for this endpoint
	)
	if err != nil {
		h.logger.WithError(err).Error("Failed to calculate leverage opportunities")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate leverage opportunities"})
		return
	}

	// Filter by minimum leverage score
	filteredOpportunities := []services.LeveragePlay{}
	for _, opportunity := range leverageOpportunities {
		if opportunity.LeverageScore >= minScore {
			filteredOpportunities = append(filteredOpportunities, opportunity)
		}
	}

	// Limit results
	if len(filteredOpportunities) > maxRes {
		filteredOpportunities = filteredOpportunities[:maxRes]
	}

	response := gin.H{
		"contest_id": contestID,
		"leverage_opportunities": filteredOpportunities,
		"filters": gin.H{
			"contest_type":       contestType,
			"min_leverage_score": minScore,
			"max_results":        maxRes,
		},
		"total_found": len(leverageOpportunities),
		"returned":    len(filteredOpportunities),
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
	})
}

// GetOwnershipTrends returns ownership trend analysis for a contest
func (h *OwnershipHandler) GetOwnershipTrends(c *gin.Context) {
	contestIDParam := c.Param("contestId")
	contestID, err := strconv.ParseUint(contestIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contest ID"})
		return
	}

	// Parse query parameters
	playerIDsParam := c.Query("player_ids")
	timeframe := c.DefaultQuery("timeframe", "24h") // 1h, 6h, 24h
	includeProjections := c.DefaultQuery("include_projections", "false") == "true"

	h.logger.WithFields(logrus.Fields{
		"contest_id":          contestID,
		"timeframe":           timeframe,
		"include_projections": includeProjections,
	}).Info("Processing ownership trends request")

	// Parse player IDs
	var playerIDs []uint
	if playerIDsParam != "" {
		playerIDStrs := strings.Split(playerIDsParam, ",")
		for _, idStr := range playerIDStrs {
			if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32); err == nil {
				playerIDs = append(playerIDs, uint(id))
			}
		}
	}

	// If no specific players requested, get trends for all players
	if len(playerIDs) == 0 {
		playerIDs = h.getActivePlayerIDs(uint(contestID))
	}

	// Get ownership trends
	trends, err := h.ownershipAnalyzer.GetOwnershipTrends(uint(contestID), playerIDs)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get ownership trends")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get ownership trends"})
		return
	}

	// Filter trends based on timeframe
	filteredTrends := h.filterTrendsByTimeframe(trends, timeframe)

	// Add projections if requested
	if includeProjections {
		h.enhanceTrendsWithProjections(filteredTrends, uint(contestID))
	}

	response := gin.H{
		"contest_id": contestID,
		"trends":     filteredTrends,
		"timeframe":  timeframe,
		"player_count": len(filteredTrends),
		"generated_at": "now", // Would use actual timestamp
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
	})
}

// GetPlayerOwnershipHistory returns historical ownership data for specific players
func (h *OwnershipHandler) GetPlayerOwnershipHistory(c *gin.Context) {
	contestIDParam := c.Param("contestId")
	contestID, err := strconv.ParseUint(contestIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contest ID"})
		return
	}

	playerIDParam := c.Query("player_id")
	if playerIDParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Player ID is required"})
		return
	}

	playerID, err := strconv.ParseUint(playerIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid player ID"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"contest_id": contestID,
		"player_id":  playerID,
	}).Info("Processing player ownership history request")

	// Get ownership history
	history, err := h.getPlayerOwnershipHistory(uint(contestID), uint(playerID))
	if err != nil {
		h.logger.WithError(err).Error("Failed to get player ownership history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get ownership history"})
		return
	}

	// Calculate summary statistics
	summary := h.calculateOwnershipSummary(history)

	response := gin.H{
		"contest_id": contestID,
		"player_id":  playerID,
		"history":    history,
		"summary":    summary,
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
	})
}

// GetOwnershipAlerts returns real-time ownership alerts
func (h *OwnershipHandler) GetOwnershipAlerts(c *gin.Context) {
	contestIDParam := c.Param("contestId")
	contestID, err := strconv.ParseUint(contestIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contest ID"})
		return
	}

	// Parse query parameters
	alertTypes := c.DefaultQuery("types", "all") // "spike", "drop", "threshold", "all"
	severity := c.DefaultQuery("severity", "medium") // "low", "medium", "high", "critical"
	since := c.DefaultQuery("since", "1h") // Time period to look back

	h.logger.WithFields(logrus.Fields{
		"contest_id":  contestID,
		"alert_types": alertTypes,
		"severity":    severity,
		"since":       since,
	}).Info("Processing ownership alerts request")

	// Get ownership alerts
	alerts := h.getOwnershipAlerts(uint(contestID), alertTypes, severity, since)

	response := gin.H{
		"contest_id": contestID,
		"alerts":     alerts,
		"filters": gin.H{
			"types":    alertTypes,
			"severity": severity,
			"since":    since,
		},
		"total_alerts": len(alerts),
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
	})
}

// Helper methods

func (h *OwnershipHandler) getOwnershipMetadata(contestID uint) map[string]interface{} {
	// This would get metadata about ownership data quality, update frequency, etc.
	return map[string]interface{}{
		"last_updated":     "2024-01-01T12:00:00Z", // Would be actual timestamp
		"update_frequency": "5 minutes",
		"data_source":      "live_tracking",
		"confidence":       95.0,
		"total_entries":    8500,
		"sample_size":      7500,
	}
}

func (h *OwnershipHandler) getContestPlayersForLeverage(contestID uint, sport string) ([]models.PlayerRecommendation, error) {
	// This would query the database for players in the contest
	// For now, return empty slice - would be populated from actual player data
	var players []models.PlayerRecommendation
	
	// TODO: Implement actual database query
	// Example implementation:
	// query := h.db.Where("contest_id = ?", contestID)
	// if sport != "" {
	//     query = query.Where("sport = ?", sport)
	// }
	// err := query.Find(&players).Error
	
	return players, nil
}

func (h *OwnershipHandler) getActivePlayerIDs(contestID uint) []uint {
	// This would get all active player IDs for the contest
	// Placeholder implementation
	return []uint{1, 2, 3, 4, 5} // Would come from database
}

func (h *OwnershipHandler) filterTrendsByTimeframe(trends []services.OwnershipTrend, timeframe string) []services.OwnershipTrend {
	// Filter trends based on timeframe
	// This is a simplified implementation - would use actual time filtering
	return trends
}

func (h *OwnershipHandler) enhanceTrendsWithProjections(trends []services.OwnershipTrend, contestID uint) {
	// Add ownership projections to trend data
	// This would implement prediction algorithms
	for i := range trends {
		// Add projected ownership fields
		// trends[i].ProjectedOwnership = calculateProjection(trends[i])
	}
}

func (h *OwnershipHandler) getPlayerOwnershipHistory(contestID, playerID uint) ([]map[string]interface{}, error) {
	// Get historical ownership snapshots for a player
	// Placeholder implementation
	history := []map[string]interface{}{
		{
			"timestamp":  "2024-01-01T10:00:00Z",
			"ownership":  15.5,
			"change":     2.3,
			"trend":      "rising",
		},
		{
			"timestamp":  "2024-01-01T11:00:00Z",
			"ownership":  17.8,
			"change":     2.3,
			"trend":      "rising",
		},
	}
	
	return history, nil
}

func (h *OwnershipHandler) calculateOwnershipSummary(history []map[string]interface{}) map[string]interface{} {
	if len(history) == 0 {
		return map[string]interface{}{}
	}

	// Calculate summary statistics
	return map[string]interface{}{
		"current_ownership": 17.8,
		"peak_ownership":   18.2,
		"min_ownership":    12.1,
		"avg_ownership":    15.8,
		"total_change":     5.7,
		"trend_direction":  "rising",
		"volatility":       "medium",
	}
}

func (h *OwnershipHandler) getOwnershipAlerts(contestID uint, alertTypes, severity, since string) []map[string]interface{} {
	// Get real-time ownership alerts
	// Placeholder implementation
	alerts := []map[string]interface{}{
		{
			"id":          "alert_001",
			"type":        "spike",
			"player_id":   123,
			"player_name": "Sample Player",
			"severity":    "high",
			"message":     "Ownership increased by 15% in last hour",
			"current_ownership": 25.8,
			"previous_ownership": 10.8,
			"change":      15.0,
			"timestamp":   "2024-01-01T12:30:00Z",
		},
		{
			"id":          "alert_002",
			"type":        "threshold",
			"player_id":   456,
			"player_name": "Another Player",
			"severity":    "medium",
			"message":     "Ownership crossed 20% threshold",
			"current_ownership": 20.1,
			"threshold":   20.0,
			"timestamp":   "2024-01-01T12:25:00Z",
		},
	}

	// Filter alerts based on criteria
	// This would implement actual filtering logic
	
	return alerts
}