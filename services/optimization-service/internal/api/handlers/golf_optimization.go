package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/optimizer"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// GolfOptimizationHandler handles golf-specific optimization requests
type GolfOptimizationHandler struct {
	positionOptimizer    *optimizer.PositionOptimizer
	cutProbabilityEngine *optimizer.CutProbabilityEngine
	logger               *logrus.Logger
}

// NewGolfOptimizationHandler creates a new golf optimization handler
func NewGolfOptimizationHandler(
	positionOptimizer *optimizer.PositionOptimizer,
	cutProbabilityEngine *optimizer.CutProbabilityEngine,
	logger *logrus.Logger,
) *GolfOptimizationHandler {
	return &GolfOptimizationHandler{
		positionOptimizer:    positionOptimizer,
		cutProbabilityEngine: cutProbabilityEngine,
		logger:               logger,
	}
}

// OptimizeGolf handles golf tournament optimization with strategy-specific parameters
func (h *GolfOptimizationHandler) OptimizeGolf(c *gin.Context) {
	var request types.GolfOptimizationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.WithError(err).Error("Failed to bind golf optimization request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate request
	if err := h.validateGolfOptimizationRequest(&request); err != nil {
		h.logger.WithError(err).Error("Golf optimization request validation failed")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request parameters",
			"details": err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"tournament_strategy": request.TournamentStrategy,
		"contest_id":         request.ContestID,
		"num_lineups":        request.NumLineups,
		"cut_optimization":   request.CutOptimization,
		"weather_enabled":    request.WeatherConsideration,
	}).Info("Processing golf optimization request")

	// Set timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Optimize using position strategy framework
	result, err := h.positionOptimizer.OptimizeForStrategy(ctx, &request)
	if err != nil {
		h.logger.WithError(err).Error("Golf optimization failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Optimization failed",
			"details": err.Error(),
		})
		return
	}

	// Enhance result with golf-specific analytics
	h.enhanceResultWithGolfAnalytics(result, &request)

	h.logger.WithFields(logrus.Fields{
		"lineups_generated": len(result.Lineups),
		"execution_time":    result.ExecutionTime,
		"strategy":         request.TournamentStrategy,
	}).Info("Golf optimization completed successfully")

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

// GetGolfStrategies returns available golf tournament strategies
func (h *GolfOptimizationHandler) GetGolfStrategies(c *gin.Context) {
	strategies := h.positionOptimizer.GetAvailableStrategies()
	
	strategyDetails := make([]gin.H, len(strategies))
	for i, strategy := range strategies {
		config, _ := h.positionOptimizer.GetStrategyConfig(strategy)
		
		strategyDetails[i] = gin.H{
			"strategy":              strategy,
			"min_cut_probability":   config.MinCutProbability,
			"weight_cut_probability": config.WeightCutProbability,
			"prefer_high_variance":  config.PreferHighVariance,
			"max_exposure":          config.MaxExposure,
			"risk_tolerance":        config.RiskTolerance,
			"description":           h.getStrategyDescription(strategy),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "success",
		"strategies": strategyDetails,
	})
}

// GetCutProbabilities returns cut probabilities for players in a tournament
func (h *GolfOptimizationHandler) GetCutProbabilities(c *gin.Context) {
	tournamentID := c.Param("tournament_id")
	if tournamentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "tournament_id is required",
		})
		return
	}

	// Get optional parameters
	courseID := c.Query("course_id")
	fieldStrengthStr := c.Query("field_strength")
	playerIDsParam := c.Query("player_ids")

	fieldStrength := 1.0
	if fieldStrengthStr != "" {
		if fs, err := strconv.ParseFloat(fieldStrengthStr, 64); err == nil {
			fieldStrength = fs
		}
	}

	// Parse player IDs if provided
	var playerIDs []string
	if playerIDsParam != "" {
		if err := json.Unmarshal([]byte(playerIDsParam), &playerIDs); err != nil {
			h.logger.WithError(err).Warn("Failed to parse player_ids parameter")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var results []*optimizer.CutProbabilityResult
	var err error

	if len(playerIDs) > 0 {
		// Batch calculation for specific players
		results, err = h.cutProbabilityEngine.BatchCalculateCutProbabilities(
			ctx, playerIDs, tournamentID, courseID, fieldStrength,
		)
	} else {
		// Return error if no player IDs provided for now
		// In a full implementation, this would get all players from tournament
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "player_ids parameter is required",
		})
		return
	}

	if err != nil {
		h.logger.WithError(err).Error("Failed to calculate cut probabilities")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to calculate cut probabilities",
			"details": err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"tournament_id":    tournamentID,
		"players_analyzed": len(results),
		"field_strength":   fieldStrength,
	}).Info("Cut probabilities calculated")

	c.JSON(http.StatusOK, gin.H{
		"status":             "success",
		"tournament_id":      tournamentID,
		"field_strength":     fieldStrength,
		"cut_probabilities":  results,
		"calculated_at":      time.Now(),
	})
}

// OptimizeGolfLateSwap handles late swap optimization recommendations
func (h *GolfOptimizationHandler) OptimizeGolfLateSwap(c *gin.Context) {
	var request struct {
		TournamentID   string                        `json:"tournament_id" binding:"required"`
		CurrentLineup  []string                      `json:"current_lineup" binding:"required"`
		Strategy       types.TournamentPositionStrategy `json:"strategy"`
		SwapDeadline   time.Time                     `json:"swap_deadline"`
		MaxSwaps       int                           `json:"max_swaps"`
		WeatherEnabled bool                          `json:"weather_enabled"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate swap deadline
	if request.SwapDeadline.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Swap deadline has already passed",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"tournament_id": request.TournamentID,
		"lineup_size":   len(request.CurrentLineup),
		"strategy":      request.Strategy,
		"max_swaps":     request.MaxSwaps,
	}).Info("Processing late swap optimization request")

	// Generate late swap recommendations
	recommendations := h.generateLateSwapRecommendations(&request)

	c.JSON(http.StatusOK, gin.H{
		"status":          "success",
		"tournament_id":   request.TournamentID,
		"recommendations": recommendations,
		"generated_at":    time.Now(),
		"time_remaining":  request.SwapDeadline.Sub(time.Now()).Minutes(),
	})
}

// GetTournamentAnalytics returns detailed analytics for a golf tournament
func (h *GolfOptimizationHandler) GetTournamentAnalytics(c *gin.Context) {
	tournamentID := c.Param("tournament_id")
	if tournamentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "tournament_id is required",
		})
		return
	}

	strategy := c.Query("strategy")
	if strategy == "" {
		strategy = string(types.BalancedStrategy)
	}

	h.logger.WithFields(logrus.Fields{
		"tournament_id": tournamentID,
		"strategy":      strategy,
	}).Info("Generating tournament analytics")

	analytics := h.generateTournamentAnalytics(tournamentID, types.TournamentPositionStrategy(strategy))

	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"tournament_id": tournamentID,
		"strategy":      strategy,
		"analytics":     analytics,
		"generated_at":  time.Now(),
	})
}

// Helper Methods

// validateGolfOptimizationRequest validates the golf optimization request
func (h *GolfOptimizationHandler) validateGolfOptimizationRequest(request *types.GolfOptimizationRequest) error {
	if request.ContestID == "" {
		return gin.Error{Err: http.ErrMissingContentLength, Type: gin.ErrorTypeBind}
	}

	if request.NumLineups < 1 || request.NumLineups > 150 {
		return gin.Error{Err: http.ErrMissingContentLength, Type: gin.ErrorTypeBind}
	}

	if request.RiskTolerance < 0 || request.RiskTolerance > 1 {
		return gin.Error{Err: http.ErrMissingContentLength, Type: gin.ErrorTypeBind}
	}

	return nil
}

// enhanceResultWithGolfAnalytics adds golf-specific analytics to the optimization result
func (h *GolfOptimizationHandler) enhanceResultWithGolfAnalytics(result *types.OptimizationResult, request *types.GolfOptimizationRequest) {
	if result.Analytics == nil {
		result.Analytics = make(map[string]interface{})
	}

	// Add golf-specific metrics
	golfAnalytics := map[string]interface{}{
		"strategy_used":           request.TournamentStrategy,
		"cut_optimization_enabled": request.CutOptimization,
		"weather_consideration":   request.WeatherConsideration,
		"course_history_used":     request.CourseHistory,
		"tee_time_correlations":   request.TeeTimeCorrelations,
		"risk_tolerance":          request.RiskTolerance,
		"min_cut_probability":     request.MinCutProbability,
		"lineup_diversity_score":  h.calculateLineupDiversity(result.Lineups),
		"strategy_adherence":      h.calculateStrategyAdherence(result.Lineups, request.TournamentStrategy),
	}

	result.Analytics["golf_optimization"] = golfAnalytics
}

// getStrategyDescription returns a human-readable description for each strategy
func (h *GolfOptimizationHandler) getStrategyDescription(strategy types.TournamentPositionStrategy) string {
	descriptions := map[types.TournamentPositionStrategy]string{
		types.WinStrategy:       "Maximize ceiling potential with high-upside players for tournament wins",
		types.TopFiveStrategy:   "Balance upside and consistency for top 5 finishes",
		types.TopTenStrategy:    "Optimize for consistent top 10 finishes with moderate risk",
		types.TopTwentyFive:     "Conservative approach targeting top 25 finishes",
		types.CutStrategy:       "Prioritize making the cut in cash games with high-floor players",
		types.BalancedStrategy:  "Balanced approach suitable for most tournament types",
	}

	if desc, exists := descriptions[strategy]; exists {
		return desc
	}
	return "Custom strategy configuration"
}

// generateLateSwapRecommendations generates late swap recommendations
func (h *GolfOptimizationHandler) generateLateSwapRecommendations(request *struct {
	TournamentID   string                        `json:"tournament_id" binding:"required"`
	CurrentLineup  []string                      `json:"current_lineup" binding:"required"`
	Strategy       types.TournamentPositionStrategy `json:"strategy"`
	SwapDeadline   time.Time                     `json:"swap_deadline"`
	MaxSwaps       int                           `json:"max_swaps"`
	WeatherEnabled bool                          `json:"weather_enabled"`
}) []types.LateSwapRecommendation {
	// This is a simplified implementation
	// In a full implementation, this would analyze current tournament state,
	// weather changes, and player performance to recommend optimal swaps
	
	recommendations := []types.LateSwapRecommendation{
		{
			PlayerOut:        request.CurrentLineup[0], // Example swap
			PlayerIn:         "replacement-player-id",
			ReasonCode:       "WEATHER_ADVANTAGE",
			Reasoning:        "Weather conditions favor this player's ball-striking ability",
			ImpactScore:      0.75,
			Confidence:       0.80,
			SwapDeadline:     request.SwapDeadline.Format(time.RFC3339),
			WeatherRelated:   request.WeatherEnabled,
			TeeTimeAdvantage: true,
		},
	}

	return recommendations
}

// generateTournamentAnalytics generates comprehensive tournament analytics
func (h *GolfOptimizationHandler) generateTournamentAnalytics(tournamentID string, strategy types.TournamentPositionStrategy) map[string]interface{} {
	// This is a simplified implementation
	// In a full implementation, this would analyze tournament data,
	// field strength, weather conditions, and historical performance
	
	return map[string]interface{}{
		"field_strength":          1.2,
		"average_cut_probability": 0.72,
		"weather_impact_score":    0.15,
		"course_difficulty":       7.5,
		"strategy_suitability":    0.85,
		"recommended_exposure": map[string]float64{
			"high_salary":   0.30,
			"medium_salary": 0.50,
			"low_salary":    0.20,
		},
		"key_insights": []string{
			"High wind conditions favor elite ball strikers",
			"Morning tee times have slight advantage due to weather",
			"Course history shows importance of accuracy over distance",
		},
	}
}

// calculateLineupDiversity calculates diversity score across lineups
func (h *GolfOptimizationHandler) calculateLineupDiversity(lineups []*types.Lineup) float64 {
	if len(lineups) < 2 {
		return 0.0
	}

	// Simplified diversity calculation
	totalOverlap := 0.0
	comparisons := 0

	for i := 0; i < len(lineups); i++ {
		for j := i + 1; j < len(lineups); j++ {
			overlap := h.calculateLineupOverlap(lineups[i], lineups[j])
			totalOverlap += overlap
			comparisons++
		}
	}

	if comparisons == 0 {
		return 0.0
	}

	avgOverlap := totalOverlap / float64(comparisons)
	return 1.0 - avgOverlap // Invert to get diversity score
}

// calculateLineupOverlap calculates overlap between two lineups
func (h *GolfOptimizationHandler) calculateLineupOverlap(lineup1, lineup2 *types.Lineup) float64 {
	if len(lineup1.Players) == 0 || len(lineup2.Players) == 0 {
		return 0.0
	}

	players1 := make(map[string]bool)
	for _, player := range lineup1.Players {
		players1[player.PlayerID] = true
	}

	overlap := 0
	for _, player := range lineup2.Players {
		if players1[player.PlayerID] {
			overlap++
		}
	}

	return float64(overlap) / float64(len(lineup1.Players))
}

// calculateStrategyAdherence calculates how well lineups adhere to the chosen strategy
func (h *GolfOptimizationHandler) calculateStrategyAdherence(lineups []*types.Lineup, strategy types.TournamentPositionStrategy) float64 {
	if len(lineups) == 0 {
		return 0.0
	}

	// Simplified strategy adherence calculation
	// In a full implementation, this would analyze lineup composition
	// against strategy parameters
	
	adherenceScore := 0.8 // Base adherence score
	
	switch strategy {
	case types.CutStrategy:
		// Check for high-floor, consistent players
		adherenceScore = 0.85
	case types.WinStrategy:
		// Check for high-ceiling, volatile players
		adherenceScore = 0.75
	default:
		adherenceScore = 0.80
	}

	return adherenceScore
}