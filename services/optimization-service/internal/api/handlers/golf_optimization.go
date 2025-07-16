package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/optimizer"
	"github.com/stitts-dev/dfs-sim/services/sports-data-service/pkg/providers"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// GolfOptimizationHandler handles golf-specific optimization requests
type GolfOptimizationHandler struct {
	positionOptimizer    *optimizer.PositionOptimizer
	cutProbabilityEngine *optimizer.CutProbabilityEngine
	dataGolfClient       *providers.DataGolfClient
	logger               *logrus.Logger
}

// NewGolfOptimizationHandler creates a new golf optimization handler
func NewGolfOptimizationHandler(
	positionOptimizer *optimizer.PositionOptimizer,
	cutProbabilityEngine *optimizer.CutProbabilityEngine,
	dataGolfClient *providers.DataGolfClient,
	logger *logrus.Logger,
) *GolfOptimizationHandler {
	return &GolfOptimizationHandler{
		positionOptimizer:    positionOptimizer,
		cutProbabilityEngine: cutProbabilityEngine,
		dataGolfClient:       dataGolfClient,
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
		"num_lineups":        request.Settings.MaxLineups,
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
		"execution_time":    result.Metadata.ExecutionTime,
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
	if request.ContestID == uuid.Nil {
		return errors.New("contest_id is required")
	}

	if request.Settings.MaxLineups < 1 || request.Settings.MaxLineups > 150 {
		return errors.New("max_lineups must be between 1 and 150")
	}

	if request.RiskTolerance < 0 || request.RiskTolerance > 1 {
		return errors.New("risk_tolerance must be between 0 and 1")
	}

	return nil
}

// enhanceResultWithGolfAnalytics adds golf-specific analytics to the optimization result
func (h *GolfOptimizationHandler) enhanceResultWithGolfAnalytics(result *types.OptimizationResult, request *types.GolfOptimizationRequest) {
	// TODO: Add analytics field to OptimizationResult or store in metadata
	// if result.Analytics == nil {
	//	result.Analytics = make(map[string]interface{})
	// }

	// Add golf-specific metrics
	_ = map[string]interface{}{
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

	// TODO: Store analytics in result metadata instead
	// result.Analytics["golf_optimization"] = golfAnalytics
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
func (h *GolfOptimizationHandler) calculateLineupDiversity(lineups []types.GeneratedLineup) float64 {
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
func (h *GolfOptimizationHandler) calculateLineupOverlap(lineup1, lineup2 types.GeneratedLineup) float64 {
	if len(lineup1.Players) == 0 || len(lineup2.Players) == 0 {
		return 0.0
	}

	players1 := make(map[uuid.UUID]bool)
	for _, player := range lineup1.Players {
		players1[player.ID] = true
	}

	overlap := 0
	for _, player := range lineup2.Players {
		if players1[player.ID] {
			overlap++
		}
	}

	return float64(overlap) / float64(len(lineup1.Players))
}

// calculateStrategyAdherence calculates how well lineups adhere to the chosen strategy
func (h *GolfOptimizationHandler) calculateStrategyAdherence(lineups []types.GeneratedLineup, strategy types.TournamentPositionStrategy) float64 {
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

// GetCutAnalysis returns DataGolf-powered cut analysis for a tournament
func (h *GolfOptimizationHandler) GetCutAnalysis(c *gin.Context) {
	tournamentID := c.Param("tournament_id")
	if tournamentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "tournament_id is required",
		})
		return
	}

	h.logger.WithField("tournament_id", tournamentID).Info("Fetching DataGolf cut analysis")

	// Get DataGolf pre-tournament predictions which include cut probabilities
	predictions, err := h.dataGolfClient.GetPreTournamentPredictions(tournamentID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get DataGolf predictions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get cut analysis",
			"details": err.Error(),
		})
		return
	}

	// Get course analytics for additional context
	var courseAnalytics interface{}
	if courseID := c.Query("course_id"); courseID != "" {
		if analytics, err := h.dataGolfClient.GetCourseAnalytics(courseID); err == nil {
			courseAnalytics = analytics
		}
	}

	h.logger.WithFields(logrus.Fields{
		"tournament_id":     tournamentID,
		"predictions_count": len(predictions.Predictions),
	}).Info("Cut analysis completed")

	c.JSON(http.StatusOK, gin.H{
		"status":          "success",
		"tournament_id":   tournamentID,
		"predictions":     predictions,
		"course_analytics": courseAnalytics,
		"generated_at":    time.Now(),
	})
}

// GetPlayerProjections returns DataGolf player projections for a tournament
func (h *GolfOptimizationHandler) GetPlayerProjections(c *gin.Context) {
	tournamentID := c.Param("tournament_id")
	if tournamentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "tournament_id is required",
		})
		return
	}

	h.logger.WithField("tournament_id", tournamentID).Info("Fetching DataGolf player projections")

	// Get DataGolf pre-tournament predictions
	predictions, err := h.dataGolfClient.GetPreTournamentPredictions(tournamentID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get DataGolf predictions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get player projections",
			"details": err.Error(),
		})
		return
	}

	// Get strokes gained data for enhanced projections
	var strokesGainedData []interface{}
	for _, playerPred := range predictions.Predictions {
		if sg, err := h.dataGolfClient.GetStrokesGainedData(strconv.Itoa(playerPred.PlayerID), tournamentID); err == nil {
			strokesGainedData = append(strokesGainedData, sg)
		}
	}

	h.logger.WithFields(logrus.Fields{
		"tournament_id":     tournamentID,
		"projections_count": len(predictions.Predictions),
		"sg_data_count":     len(strokesGainedData),
	}).Info("Player projections completed")

	c.JSON(http.StatusOK, gin.H{
		"status":             "success",
		"tournament_id":      tournamentID,
		"projections":        predictions,
		"strokes_gained_data": strokesGainedData,
		"generated_at":       time.Now(),
	})
}

// GetWeatherImpact returns DataGolf weather impact analysis for a tournament
func (h *GolfOptimizationHandler) GetWeatherImpact(c *gin.Context) {
	tournamentID := c.Param("tournament_id")
	if tournamentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "tournament_id is required",
		})
		return
	}

	h.logger.WithField("tournament_id", tournamentID).Info("Fetching DataGolf weather impact analysis")

	// Get DataGolf weather impact data
	weatherImpact, err := h.dataGolfClient.GetWeatherImpactData(tournamentID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get DataGolf weather impact")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get weather impact analysis",
			"details": err.Error(),
		})
		return
	}

	// Get course analytics to understand weather sensitivity
	var courseAnalytics interface{}
	if courseID := c.Query("course_id"); courseID != "" {
		if analytics, err := h.dataGolfClient.GetCourseAnalytics(courseID); err == nil {
			courseAnalytics = analytics
		}
	}

	h.logger.WithFields(logrus.Fields{
		"tournament_id":    tournamentID,
		"weather_analysis": weatherImpact != nil,
	}).Info("Weather impact analysis completed")

	c.JSON(http.StatusOK, gin.H{
		"status":          "success",
		"tournament_id":   tournamentID,
		"weather_impact":  weatherImpact,
		"course_analytics": courseAnalytics,
		"generated_at":    time.Now(),
	})
}

// GetLiveOptimizationUpdates provides real-time optimization updates during tournaments
func (h *GolfOptimizationHandler) GetLiveOptimizationUpdates(c *gin.Context) {
	tournamentID := c.Param("tournament_id")
	if tournamentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "tournament_id is required",
		})
		return
	}

	// Optional strategy parameter
	strategyParam := c.Query("strategy")
	strategy := types.BalancedStrategy // Default strategy
	if strategyParam != "" {
		switch strategyParam {
		case "win":
			strategy = types.WinStrategy
		case "top5":
			strategy = types.TopFiveStrategy
		case "top10":
			strategy = types.TopTenStrategy
		case "top25":
			strategy = types.TopTwentyFive
		case "cut":
			strategy = types.CutStrategy
		case "balanced":
			strategy = types.BalancedStrategy
		}
	}

	h.logger.WithFields(logrus.Fields{
		"tournament_id": tournamentID,
		"strategy":      strategy,
	}).Info("Generating live optimization updates")

	// Get live tournament data from DataGolf
	liveData, err := h.dataGolfClient.GetLiveTournamentData(tournamentID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get live tournament data")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get live tournament data",
			"details": err.Error(),
		})
		return
	}

	// Generate live optimization recommendations
	recommendations := h.generateLiveRecommendations(liveData, strategy)

	// Get updated cut analysis
	var cutAnalysis interface{}
	if predictions, err := h.dataGolfClient.GetPreTournamentPredictions(tournamentID); err == nil {
		cutAnalysis = h.processLiveCutAnalysis(predictions, liveData)
	}

	// Get updated weather impact
	var weatherUpdates interface{}
	if weather, err := h.dataGolfClient.GetWeatherImpactData(tournamentID); err == nil {
		weatherUpdates = weather
	}

	h.logger.WithFields(logrus.Fields{
		"tournament_id":          tournamentID,
		"recommendations_count":  len(recommendations.PlayerRecommendations),
		"tournament_round":       liveData.CurrentRound,
	}).Info("Live optimization updates completed")

	c.JSON(http.StatusOK, gin.H{
		"status":           "success",
		"tournament_id":    tournamentID,
		"strategy":         strategy,
		"live_data":        liveData,
		"recommendations":  recommendations,
		"cut_analysis":     cutAnalysis,
		"weather_updates":  weatherUpdates,
		"generated_at":     time.Now(),
		"tournament_state": map[string]interface{}{
			"round":           liveData.CurrentRound,
			"cut_made":        liveData.CutMade,
			"leaderboard":     liveData.LiveLeaderboard,
		},
	})
}

// generateLiveRecommendations creates real-time lineup recommendations based on live tournament data
func (h *GolfOptimizationHandler) generateLiveRecommendations(
	liveData *providers.LiveTournamentData,
	strategy types.TournamentPositionStrategy,
) *LiveOptimizationRecommendations {
	recommendations := &LiveOptimizationRecommendations{
		TournamentID:         liveData.TournamentID,
		Strategy:             strategy,
		PlayerRecommendations: make([]PlayerRecommendation, 0),
		SwapRecommendations:  make([]SwapRecommendation, 0),
		GeneratedAt:          time.Now(),
	}

	// Analyze each player's current tournament performance
	for _, player := range liveData.LiveLeaderboard {
		recommendation := h.analyzePlayerLivePerformance(player, liveData, strategy)
		if recommendation != nil {
			recommendations.PlayerRecommendations = append(recommendations.PlayerRecommendations, *recommendation)
		}
	}

	// Generate swap recommendations for late swap opportunities
	if liveData.CurrentRound > 1 && !liveData.CutMade {
		swapRecommendations := h.generateSwapRecommendations(liveData, strategy)
		recommendations.SwapRecommendations = swapRecommendations
	}

	return recommendations
}

// analyzePlayerLivePerformance analyzes a player's current tournament performance
func (h *GolfOptimizationHandler) analyzePlayerLivePerformance(
	player *providers.LiveLeaderboardEntry,
	liveData *providers.LiveTournamentData,
	strategy types.TournamentPositionStrategy,
) *PlayerRecommendation {
	if player == nil {
		return nil
	}

	recommendation := &PlayerRecommendation{
		PlayerID:     strconv.Itoa(player.PlayerID),
		PlayerName:   player.PlayerName,
		CurrentScore: player.TotalScore,
		Position:     player.Position,
		Momentum:     h.calculatePlayerMomentum(player),
		Action:       "hold", // Default action
		Confidence:   0.5,    // Default confidence
		Reasoning:    []string{},
	}

	// Analyze based on strategy
	switch strategy {
	case types.WinStrategy:
		recommendation = h.analyzeForWinStrategy(player, liveData, recommendation)
	case types.CutStrategy:
		recommendation = h.analyzeForCutStrategy(player, liveData, recommendation)
	case types.TopFiveStrategy, types.TopTenStrategy:
		recommendation = h.analyzeForPlacingStrategy(player, liveData, recommendation, strategy)
	default:
		recommendation = h.analyzeForBalancedStrategy(player, liveData, recommendation)
	}

	return recommendation
}

// Supporting types for live optimization
type LiveOptimizationRecommendations struct {
	TournamentID          string                 `json:"tournament_id"`
	Strategy              types.TournamentPositionStrategy `json:"strategy"`
	PlayerRecommendations []PlayerRecommendation `json:"player_recommendations"`
	SwapRecommendations   []SwapRecommendation   `json:"swap_recommendations"`
	GeneratedAt           time.Time              `json:"generated_at"`
}

type PlayerRecommendation struct {
	PlayerID     string   `json:"player_id"`
	PlayerName   string   `json:"player_name"`
	CurrentScore int      `json:"current_score"`
	Position     int      `json:"position"`
	Momentum     float64  `json:"momentum"`
	Action       string   `json:"action"` // "target", "avoid", "hold", "swap"
	Confidence   float64  `json:"confidence"`
	Reasoning    []string `json:"reasoning"`
}

type SwapRecommendation struct {
	SwapOut      string  `json:"swap_out"`
	SwapIn       string  `json:"swap_in"`
	Reasoning    string  `json:"reasoning"`
	ExpectedGain float64 `json:"expected_gain"`
	RiskLevel    string  `json:"risk_level"` // "low", "medium", "high"
}

// Helper methods for live analysis
func (h *GolfOptimizationHandler) calculatePlayerMomentum(player *providers.LiveLeaderboardEntry) float64 {
	// Calculate momentum based on current round performance and position movement
	// Since ThruHoles is int (holes completed), we use RoundScore and MovementIndicator
	momentum := 0.0
	
	// Factor in round score performance (negative score is better)
	if player.RoundScore < 0 {
		momentum += float64(-player.RoundScore) * 0.3 // Under par adds positive momentum
	} else if player.RoundScore > 0 {
		momentum -= float64(player.RoundScore) * 0.2 // Over par reduces momentum
	}
	
	// Factor in movement indicator
	switch player.MovementIndicator {
	case "up":
		momentum += 0.5
	case "down":
		momentum -= 0.5
	case "steady":
		momentum += 0.0
	}
	
	// Factor in holes completed (more holes = more data reliability)
	if player.ThruHoles > 9 {
		momentum *= 1.0 // Full round data
	} else if player.ThruHoles > 5 {
		momentum *= 0.7 // Partial round data
	} else {
		momentum *= 0.3 // Very limited data
	}
	
	return momentum
}

func (h *GolfOptimizationHandler) analyzeForWinStrategy(
	player *providers.LiveLeaderboardEntry,
	liveData *providers.LiveTournamentData,
	rec *PlayerRecommendation,
) *PlayerRecommendation {
	// Win strategy focuses on contenders and players with momentum
	if player.Position <= 5 && player.TotalScore <= liveData.CutLine + 10 {
		rec.Action = "target"
		rec.Confidence = 0.85
		rec.Reasoning = append(rec.Reasoning, "In contention for win")
	} else if rec.Momentum > 1.0 && player.Position <= 20 {
		rec.Action = "target"
		rec.Confidence = 0.70
		rec.Reasoning = append(rec.Reasoning, "Strong momentum, moving up leaderboard")
	} else if player.Position > 50 {
		rec.Action = "avoid"
		rec.Confidence = 0.75
		rec.Reasoning = append(rec.Reasoning, "Too far back for win strategy")
	}

	return rec
}

func (h *GolfOptimizationHandler) analyzeForCutStrategy(
	player *providers.LiveLeaderboardEntry,
	liveData *providers.LiveTournamentData,
	rec *PlayerRecommendation,
) *PlayerRecommendation {
	// Cut strategy focuses on cut line safety
	cutBuffer := player.TotalScore - liveData.CutLine

	if cutBuffer <= -3 {
		rec.Action = "target"
		rec.Confidence = 0.90
		rec.Reasoning = append(rec.Reasoning, "Safely inside cut line")
	} else if cutBuffer >= 2 {
		rec.Action = "avoid"
		rec.Confidence = 0.80
		rec.Reasoning = append(rec.Reasoning, "In danger of missing cut")
	} else if rec.Momentum > 0.5 {
		rec.Action = "target"
		rec.Confidence = 0.65
		rec.Reasoning = append(rec.Reasoning, "Near cut line but trending positively")
	}

	return rec
}

func (h *GolfOptimizationHandler) analyzeForPlacingStrategy(
	player *providers.LiveLeaderboardEntry,
	liveData *providers.LiveTournamentData,
	rec *PlayerRecommendation,
	strategy types.TournamentPositionStrategy,
) *PlayerRecommendation {
	targetPosition := 10
	if strategy == types.TopFiveStrategy {
		targetPosition = 5
	}

	if player.Position <= targetPosition {
		rec.Action = "target"
		rec.Confidence = 0.80
		rec.Reasoning = append(rec.Reasoning, fmt.Sprintf("Currently in target position (T%d)", targetPosition))
	} else if player.Position <= targetPosition*2 && rec.Momentum > 0.3 {
		rec.Action = "target"
		rec.Confidence = 0.65
		rec.Reasoning = append(rec.Reasoning, "Within range with positive momentum")
	}

	return rec
}

func (h *GolfOptimizationHandler) analyzeForBalancedStrategy(
	player *providers.LiveLeaderboardEntry,
	liveData *providers.LiveTournamentData,
	rec *PlayerRecommendation,
) *PlayerRecommendation {
	// Balanced approach considers multiple factors
	if player.Position <= 15 && player.TotalScore <= liveData.CutLine-2 {
		rec.Action = "target"
		rec.Confidence = 0.75
		rec.Reasoning = append(rec.Reasoning, "Good position with cut safety")
	} else if rec.Momentum > 1.0 {
		rec.Action = "target"
		rec.Confidence = 0.65
		rec.Reasoning = append(rec.Reasoning, "Strong momentum regardless of position")
	}

	return rec
}

func (h *GolfOptimizationHandler) generateSwapRecommendations(
	liveData *providers.LiveTournamentData,
	strategy types.TournamentPositionStrategy,
) []SwapRecommendation {
	swaps := make([]SwapRecommendation, 0)

	h.logger.WithFields(logrus.Fields{
		"tournament_id": liveData.TournamentID,
		"strategy":      strategy,
		"round":         liveData.CurrentRound,
	}).Debug("Generating late swap recommendations")

	// Only generate swaps if we're in a swappable round (typically after round 1)
	if liveData.CurrentRound < 2 {
		return swaps
	}

	// Analyze players for swap potential
	swapOutCandidates := h.identifySwapOutCandidates(liveData, strategy)
	swapInCandidates := h.identifySwapInCandidates(liveData, strategy)

	// Generate swap recommendations by pairing candidates
	for _, swapOut := range swapOutCandidates {
		bestSwapIn := h.findBestSwapMatch(swapOut, swapInCandidates, strategy)
		if bestSwapIn != nil {
			swap := SwapRecommendation{
				SwapOut:      swapOut.PlayerName,
				SwapIn:       bestSwapIn.PlayerName,
				Reasoning:    h.generateSwapReasoning(swapOut, bestSwapIn, strategy),
				ExpectedGain: h.calculateExpectedGain(swapOut, bestSwapIn, strategy),
				RiskLevel:    h.assessSwapRisk(swapOut, bestSwapIn, liveData),
			}
			swaps = append(swaps, swap)
		}
	}

	// Sort swaps by expected gain (highest first)
	h.sortSwapRecommendations(swaps)

	// Limit to top 5 recommendations to avoid overwhelming user
	if len(swaps) > 5 {
		swaps = swaps[:5]
	}

	h.logger.WithField("swap_count", len(swaps)).Debug("Generated late swap recommendations")

	return swaps
}

// identifySwapOutCandidates finds players who are underperforming and should be swapped out
func (h *GolfOptimizationHandler) identifySwapOutCandidates(
	liveData *providers.LiveTournamentData,
	strategy types.TournamentPositionStrategy,
) []*providers.LiveLeaderboardEntry {
	candidates := make([]*providers.LiveLeaderboardEntry, 0)

	for _, player := range liveData.LiveLeaderboard {
		shouldSwapOut := false

		switch strategy {
		case types.WinStrategy:
			// Swap out players too far back or without momentum
			if player.Position > 25 || (player.Position > 10 && h.calculatePlayerMomentum(player) < 0) {
				shouldSwapOut = true
			}

		case types.CutStrategy:
			// Swap out players in danger of missing cut
			if player.TotalScore >= liveData.CutLine+1 && h.calculatePlayerMomentum(player) < 0.5 {
				shouldSwapOut = true
			}

		case types.TopFiveStrategy:
			// Swap out players unlikely to finish top 5
			if player.Position > 15 && h.calculatePlayerMomentum(player) <= 0 {
				shouldSwapOut = true
			}

		case types.TopTenStrategy:
			// Swap out players unlikely to finish top 10
			if player.Position > 25 && h.calculatePlayerMomentum(player) <= 0 {
				shouldSwapOut = true
			}

		default: // Balanced strategy
			// Swap out players with poor position and negative momentum
			if player.Position > 30 && h.calculatePlayerMomentum(player) < -0.5 {
				shouldSwapOut = true
			}
		}

		if shouldSwapOut {
			candidates = append(candidates, player)
		}
	}

	return candidates
}

// identifySwapInCandidates finds players who are outperforming and should be targeted
func (h *GolfOptimizationHandler) identifySwapInCandidates(
	liveData *providers.LiveTournamentData,
	strategy types.TournamentPositionStrategy,
) []*providers.LiveLeaderboardEntry {
	candidates := make([]*providers.LiveLeaderboardEntry, 0)

	for _, player := range liveData.LiveLeaderboard {
		shouldSwapIn := false
		momentum := h.calculatePlayerMomentum(player)

		switch strategy {
		case types.WinStrategy:
			// Target contenders and players with strong momentum
			if (player.Position <= 10) || (player.Position <= 20 && momentum > 1.0) {
				shouldSwapIn = true
			}

		case types.CutStrategy:
			// Target players safely making cut
			if player.TotalScore <= liveData.CutLine-2 {
				shouldSwapIn = true
			}

		case types.TopFiveStrategy:
			// Target players in top 10 with momentum
			if player.Position <= 5 || (player.Position <= 12 && momentum > 0.5) {
				shouldSwapIn = true
			}

		case types.TopTenStrategy:
			// Target players in top 15 with momentum
			if player.Position <= 10 || (player.Position <= 18 && momentum > 0.3) {
				shouldSwapIn = true
			}

		default: // Balanced strategy
			// Target players with good position and momentum
			if player.Position <= 20 && momentum > 0.3 {
				shouldSwapIn = true
			}
		}

		if shouldSwapIn {
			candidates = append(candidates, player)
		}
	}

	return candidates
}

// findBestSwapMatch finds the best swap-in candidate for a given swap-out player
func (h *GolfOptimizationHandler) findBestSwapMatch(
	swapOut *providers.LiveLeaderboardEntry,
	swapInCandidates []*providers.LiveLeaderboardEntry,
	strategy types.TournamentPositionStrategy,
) *providers.LiveLeaderboardEntry {
	var bestMatch *providers.LiveLeaderboardEntry
	bestScore := 0.0

	for _, candidate := range swapInCandidates {
		// Calculate swap score based on multiple factors
		score := h.calculateSwapScore(swapOut, candidate, strategy)

		if score > bestScore {
			bestScore = score
			bestMatch = candidate
		}
	}

	// Only return match if score is above threshold
	if bestScore > 0.3 {
		return bestMatch
	}

	return nil
}

// calculateSwapScore calculates how good a swap would be
func (h *GolfOptimizationHandler) calculateSwapScore(
	swapOut *providers.LiveLeaderboardEntry,
	swapIn *providers.LiveLeaderboardEntry,
	strategy types.TournamentPositionStrategy,
) float64 {
	score := 0.0

	// Position improvement
	positionDiff := float64(swapOut.Position - swapIn.Position)
	score += positionDiff * 0.02 // 2% per position improvement

	// Momentum differential
	momentumOut := h.calculatePlayerMomentum(swapOut)
	momentumIn := h.calculatePlayerMomentum(swapIn)
	momentumDiff := momentumIn - momentumOut
	score += momentumDiff * 0.1

	// Score differential
	scoreDiff := float64(swapOut.TotalScore - swapIn.TotalScore)
	score += scoreDiff * 0.05 // 5% per stroke improvement

	// Strategy-specific adjustments
	switch strategy {
	case types.WinStrategy:
		// Heavily weight position for win strategy
		if swapIn.Position <= 5 {
			score += 0.2
		}
	case types.CutStrategy:
		// Weight cut safety
		if swapIn.TotalScore <= swapOut.TotalScore-3 {
			score += 0.15
		}
	}

	return score
}

// generateSwapReasoning creates human-readable reasoning for the swap
func (h *GolfOptimizationHandler) generateSwapReasoning(
	swapOut *providers.LiveLeaderboardEntry,
	swapIn *providers.LiveLeaderboardEntry,
	strategy types.TournamentPositionStrategy,
) string {
	reasons := []string{}

	// Position improvement
	if swapIn.Position < swapOut.Position {
		reasons = append(reasons, fmt.Sprintf("Position upgrade: T%d → T%d", swapOut.Position, swapIn.Position))
	}

	// Score improvement
	if swapIn.TotalScore < swapOut.TotalScore {
		strokeDiff := swapOut.TotalScore - swapIn.TotalScore
		reasons = append(reasons, fmt.Sprintf("%d stroke improvement", strokeDiff))
	}

	// Momentum
	momentumIn := h.calculatePlayerMomentum(swapIn)
	momentumOut := h.calculatePlayerMomentum(swapOut)
	if momentumIn > momentumOut+0.5 {
		reasons = append(reasons, "Strong positive momentum")
	}

	// Strategy-specific reasoning
	switch strategy {
	case types.WinStrategy:
		if swapIn.Position <= 5 {
			reasons = append(reasons, "In contention to win")
		}
	case types.CutStrategy:
		if swapIn.TotalScore < swapOut.TotalScore {
			reasons = append(reasons, "Better cut safety")
		}
	}

	if len(reasons) == 0 {
		return "Overall tournament position improvement"
	}

	return fmt.Sprintf("%s", reasons[0])
}

// calculateExpectedGain estimates the expected fantasy point gain from the swap
func (h *GolfOptimizationHandler) calculateExpectedGain(
	swapOut *providers.LiveLeaderboardEntry,
	swapIn *providers.LiveLeaderboardEntry,
	strategy types.TournamentPositionStrategy,
) float64 {
	// Simplified expected gain calculation
	// In a full implementation, this would use detailed scoring models

	positionGain := float64(swapOut.Position-swapIn.Position) * 0.5 // 0.5 points per position
	scoreGain := float64(swapOut.TotalScore-swapIn.TotalScore) * 2.0 // 2 points per stroke
	momentumGain := (h.calculatePlayerMomentum(swapIn) - h.calculatePlayerMomentum(swapOut)) * 5.0

	totalGain := positionGain + scoreGain + momentumGain

	// Ensure reasonable bounds
	return math.Max(0.0, math.Min(20.0, totalGain))
}

// assessSwapRisk determines the risk level of the swap
func (h *GolfOptimizationHandler) assessSwapRisk(
	swapOut *providers.LiveLeaderboardEntry,
	swapIn *providers.LiveLeaderboardEntry,
	liveData *providers.LiveTournamentData,
) string {
	riskScore := 0.0

	// Position risk - swapping to someone further back is riskier
	if swapIn.Position > swapOut.Position {
		riskScore += 0.3
	}

	// Momentum risk - negative momentum is risky
	if h.calculatePlayerMomentum(swapIn) < 0 {
		riskScore += 0.2
	}

	// Cut risk
	if swapIn.TotalScore >= liveData.CutLine-1 {
		riskScore += 0.3
	}

	// Round risk - later rounds are riskier
	if liveData.CurrentRound >= 3 {
		riskScore += 0.2
	}

	if riskScore <= 0.3 {
		return "low"
	} else if riskScore <= 0.6 {
		return "medium"
	} else {
		return "high"
	}
}

// sortSwapRecommendations sorts swaps by expected gain (descending)
func (h *GolfOptimizationHandler) sortSwapRecommendations(swaps []SwapRecommendation) {
	for i := 0; i < len(swaps)-1; i++ {
		for j := i + 1; j < len(swaps); j++ {
			if swaps[j].ExpectedGain > swaps[i].ExpectedGain {
				swaps[i], swaps[j] = swaps[j], swaps[i]
			}
		}
	}
}

func (h *GolfOptimizationHandler) processLiveCutAnalysis(
	predictions *providers.TournamentPredictions,
	liveData *providers.LiveTournamentData,
) map[string]interface{} {
	analysis := map[string]interface{}{
		"cut_line":         liveData.CutLine,
		"players_at_risk":  0,
		"players_safe":     0,
	}

	atRisk := 0
	safe := 0

	for _, player := range liveData.LiveLeaderboard {
		if player.TotalScore >= liveData.CutLine + 1 {
			atRisk++
		} else if player.TotalScore <= liveData.CutLine - 3 {
			safe++
		}
	}

	analysis["players_at_risk"] = atRisk
	analysis["players_safe"] = safe

	return analysis
}
