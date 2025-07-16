package handlers

import (
	"crypto/md5"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/websocket"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/optimizer"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/cache"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// OptimizationHandler handles optimization-related endpoints
type OptimizationHandler struct {
	db              *database.DB
	cache           *cache.AnalyticsCache
	wsHub           *websocket.Hub
	config          *config.Config
	logger          *logrus.Logger
	dpOptimizer     *optimizer.DPOptimizer
	analyticsEngine *optimizer.AnalyticsEngine
}

// OptimizationRequestV2 represents enhanced optimization request with strategy options
type OptimizationRequestV2 struct {
	UserID                 uuid.UUID                        `json:"user_id"`
	ContestID              uuid.UUID                        `json:"contest_id"`
	PlayerPool             []types.Player                   `json:"player_pool"`
	Strategy               optimizer.OptimizationObjective `json:"strategy"`
	NumLineups             int                              `json:"num_lineups"`
	SalaryCap              int                              `json:"salary_cap"`
	MinDifferentPlayers    int                              `json:"min_different_players"`
	UseAnalytics           bool                             `json:"use_analytics"`
	UseCorrelations        bool                             `json:"use_correlations"`
	CorrelationWeight      float64                          `json:"correlation_weight"`
	ExposureConfig         optimizer.ExposureConfig         `json:"exposure_config"`
	PerformanceMode        string                           `json:"performance_mode"`
	StackingRules          []types.StackingRule             `json:"stacking_rules"`
	LockedPlayers          []uuid.UUID                      `json:"locked_players"`
	ExcludedPlayers        []uuid.UUID                      `json:"excluded_players"`
	MinExposure            map[uuid.UUID]float64            `json:"min_exposure"`
	MaxExposure            map[uuid.UUID]float64            `json:"max_exposure"`
	OwnershipStrategy      string                           `json:"ownership_strategy"`
}

// OptimizationResponseV2 represents enhanced optimization response with analytics
type OptimizationResponseV2 struct {
	Lineups           []types.GeneratedLineup        `json:"lineups"`
	Analytics         *OptimizationAnalytics         `json:"analytics"`
	ExposureReport    *optimizer.ExposureReport      `json:"exposure_report"`
	PerformanceMetrics *PerformanceMetrics           `json:"performance_metrics"`
	Strategy          optimizer.OptimizationObjective `json:"strategy"`
	CorrelationMatrix map[string]float64             `json:"correlation_matrix"`
}

// OptimizationAnalytics provides detailed analytics about the optimization
type OptimizationAnalytics struct {
	TotalLineups        int                                         `json:"total_lineups"`
	AverageScore        float64                                     `json:"average_score"`
	TopScore            float64                                     `json:"top_score"`
	ScoreVariance       float64                                     `json:"score_variance"`
	DiversityScore      float64                                     `json:"diversity_score"`
	StackAnalysis       map[string]int                              `json:"stack_analysis"`
	PositionBreakdown   map[string]int                              `json:"position_breakdown"`
	SalaryDistribution  map[string]int                              `json:"salary_distribution"`
	PlayerAnalytics     map[uuid.UUID]*optimizer.PlayerAnalytics   `json:"player_analytics"`
}

// PerformanceMetrics tracks optimization performance
type PerformanceMetrics struct {
	OptimizationTime    time.Duration               `json:"optimization_time"`
	StatesExplored      int64                       `json:"states_explored"`
	CacheHitRate        float64                     `json:"cache_hit_rate"`
	MemoryUsage         int64                       `json:"memory_usage_bytes"`
	Algorithm           string                      `json:"algorithm"`
	PerformanceMode     string                      `json:"performance_mode"`
}

// NewOptimizationHandler creates a new optimization handler
func NewOptimizationHandler(
	db *database.DB,
	cache *cache.AnalyticsCache,
	wsHub *websocket.Hub,
	config *config.Config,
	logger *logrus.Logger,
) *OptimizationHandler {
	return &OptimizationHandler{
		db:              db,
		cache:           cache,
		wsHub:           wsHub,
		config:          config,
		logger:          logger,
		dpOptimizer:     optimizer.NewDPOptimizer(),
		analyticsEngine: optimizer.NewAnalyticsEngine(),
	}
}

// OptimizeLineups handles lineup optimization requests
func (h *OptimizationHandler) OptimizeLineups(c *gin.Context) {
	var req types.OptimizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "Invalid request format",
			Code:  "INVALID_REQUEST",
			Details: map[string]string{
				"validation_error": err.Error(),
			},
		})
		return
	}

	// Generate cache key for the request
	cacheKey := h.generateCacheKey(req)
	
	// Check cache first
	if cached, err := h.cache.GetOptimizationResult(c.Request.Context(), cacheKey); err == nil && cached != nil {
		h.logger.WithField("cache_key", cacheKey).Info("Returning cached optimization result")
		// Convert cached DPResult back to OptimizationResult
		response := h.convertFromDPResult(cached)
		c.JSON(http.StatusOK, response)
		return
	}

	// Create progress channel for WebSocket updates
	progressChan := make(chan types.ProgressUpdate, 100)
	defer close(progressChan)

	// Start progress forwarding to WebSocket if user ID provided
	if req.UserID != uuid.Nil {
		go h.forwardProgressToWebSocket(req.UserID, progressChan)
	}

	// Build correlation matrix for golf if applicable
	correlationMatrix := make(map[string]float64)
	if len(req.PlayerPool) > 0 && req.PlayerPool[0].TeeTime != "" {
		// Golf-specific correlation matrix
		correlationMatrix = h.buildGolfCorrelationMatrix(req.PlayerPool)
	} else {
		// General correlation matrix
		correlationMatrix = h.buildGeneralCorrelationMatrix(req.PlayerPool)
	}

	// Send initial progress update
	progressChan <- types.ProgressUpdate{
		Type:        "optimization",
		Progress:    0.0,
		Message:     "Starting optimization...",
		CurrentStep: "initialization",
		TotalSteps:  req.Settings.MaxLineups,
		Timestamp:   time.Now(),
	}

	// Run optimization with progress tracking
	startTime := time.Now()
	
	// Convert request settings to optimizer settings
	settings := optimizer.OptimizeConfig{
		SalaryCap:           req.Constraints.SalaryCap,
		NumLineups:          req.Settings.MaxLineups,
		MinDifferentPlayers: req.Settings.MinDifferentPlayers,
		UseCorrelations:     req.Settings.UseCorrelations,
		CorrelationWeight:   req.Settings.CorrelationWeight,
		StackingRules:       req.Settings.StackingRules,
		LockedPlayers:       req.Settings.LockedPlayers,
		ExcludedPlayers:     req.Settings.ExcludedPlayers,
		MinExposure:         req.Settings.MinExposure,
		MaxExposure:         req.Settings.MaxExposure,
	}
	
	// Convert OptimizationPlayer to Player for optimization
	players := make([]types.Player, len(req.PlayerPool))
	for i, op := range req.PlayerPool {
		players[i] = convertOptimizationPlayerToPlayer(op)
	}
	
	result, err := optimizer.OptimizeLineups(players, settings)
	if err != nil {
		h.logger.WithError(err).Error("Optimization failed")
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: "Optimization failed",
			Code:  "OPTIMIZATION_ERROR",
			Details: map[string]string{
				"error": err.Error(),
			},
		})
		return
	}

	// Convert lineups to GeneratedLineup format
	generatedLineups := make([]types.GeneratedLineup, len(result.Lineups))
	for i, lineup := range result.Lineups {
		generatedLineups[i] = types.GeneratedLineup{
			ID:              fmt.Sprintf("%d", lineup.ID),
			Players:         lineup.Players,
			TotalSalary:     lineup.TotalSalary,
			ProjectedPoints: lineup.ProjectedPoints,
			Exposure:        1.0 / float64(len(result.Lineups)), // Equal exposure
		}
	}

	// Convert metadata
	metadata := types.OptimizationMetadata{
		TotalLineups:     len(result.Lineups),
		ExecutionTime:    result.Metadata.ExecutionTime,
		AverageUniqueess: 0.8, // Placeholder
		TopProjection:    0.0, // Placeholder
		AverageProjection: 0.0, // Placeholder
		StacksGenerated:  0, // Placeholder
	}

	// Calculate average projection
	totalProjection := 0.0
	for _, lineup := range result.Lineups {
		totalProjection += lineup.ProjectedPoints
		if lineup.ProjectedPoints > metadata.TopProjection {
			metadata.TopProjection = lineup.ProjectedPoints
		}
	}
	if len(result.Lineups) > 0 {
		metadata.AverageProjection = totalProjection / float64(len(result.Lineups))
	}

	// Create response
	response := types.OptimizationResult{
		Lineups:           generatedLineups,
		Metadata:          metadata,
		CorrelationMatrix: correlationMatrix,
	}

	// Convert result to DPResult for caching
	dpResult := h.convertToDPResult(result, time.Since(startTime))
	
	// Cache the result
	if err := h.cache.SetOptimizationResult(c.Request.Context(), cacheKey, dpResult, 24*time.Hour); err != nil {
		h.logger.WithError(err).Warn("Failed to cache optimization result")
	}

	// Send final progress update
	progressChan <- types.ProgressUpdate{
		Type:        "optimization",
		Progress:    1.0,
		Message:     fmt.Sprintf("Optimization completed! Generated %d lineups in %v", len(result.Lineups), time.Since(startTime)),
		CurrentStep: "completed",
		TotalSteps:  req.Settings.MaxLineups,
		Timestamp:   time.Now(),
	}

	h.logger.WithFields(logrus.Fields{
		"lineups_generated": len(result.Lineups),
		"execution_time":    time.Since(startTime),
		"user_id":          req.UserID,
		"contest_id":       req.ContestID,
	}).Info("Optimization completed successfully")

	c.JSON(http.StatusOK, response)
}

// OptimizeLineupsV2 handles enhanced lineup optimization requests with strategy support
func (h *OptimizationHandler) OptimizeLineupsV2(c *gin.Context) {
	var req OptimizationRequestV2
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "Invalid request format",
			Code:  "INVALID_REQUEST",
			Details: map[string]string{
				"validation_error": err.Error(),
			},
		})
		return
	}

	// Validate request
	if err := h.validateRequestV2(req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "Invalid request parameters",
			Code:  "INVALID_PARAMETERS",
			Details: map[string]string{
				"validation_error": err.Error(),
			},
		})
		return
	}

	// Generate cache key for the enhanced request
	cacheKey := h.generateCacheKeyV2(req)
	
	// Check cache first if analytics is not explicitly requested
	if !req.UseAnalytics {
		if cached, err := h.cache.GetOptimizationResult(c.Request.Context(), cacheKey); err == nil && cached != nil {
			h.logger.WithField("cache_key", cacheKey).Info("Returning cached enhanced optimization result")
			// Convert cached result to V2 format
			response := h.convertCachedToV2Response(cached, req)
			c.JSON(http.StatusOK, response)
			return
		}
	}

	// Create progress channel for WebSocket updates
	progressChan := make(chan types.ProgressUpdate, 100)
	defer close(progressChan)

	// Start progress forwarding to WebSocket if user ID provided
	if req.UserID != uuid.Nil {
		go h.forwardProgressToWebSocket(req.UserID, progressChan)
	}

	// Send initial progress update
	progressChan <- types.ProgressUpdate{
		Type:        "enhanced_optimization",
		Progress:    0.0,
		Message:     fmt.Sprintf("Starting %s strategy optimization...", req.Strategy),
		CurrentStep: "initialization",
		TotalSteps:  req.NumLineups,
		Timestamp:   time.Now(),
	}

	// Run enhanced optimization
	startTime := time.Now()
	
	// Convert request to enhanced config
	config := h.convertToOptimizeConfigV2(req)
	
	// Determine contest information
	if len(req.PlayerPool) > 0 {
		// Create a minimal contest from the request
		config.Contest = &types.Contest{
			ID:       req.ContestID,
			SportID:  uuid.New(), // This would be determined from players
			Platform: "draftkings", // Default or from request
			SalaryCap: req.SalaryCap,
		}
	}

	// Calculate player analytics if enabled
	var playerAnalytics map[uuid.UUID]*optimizer.PlayerAnalytics
	if req.UseAnalytics {
		progressChan <- types.ProgressUpdate{
			Type:        "enhanced_optimization",
			Progress:    0.1,
			Message:     "Calculating player analytics...",
			CurrentStep: "analytics",
			TotalSteps:  req.NumLineups,
			Timestamp:   time.Now(),
		}

		analytics, err := h.analyticsEngine.CalculateBulkAnalytics(req.PlayerPool, make(map[uuid.UUID][]optimizer.PerformanceData))
		if err != nil {
			h.logger.WithError(err).Warn("Failed to calculate analytics, proceeding without")
		} else {
			playerAnalytics = analytics
		}
	}

	progressChan <- types.ProgressUpdate{
		Type:        "enhanced_optimization",
		Progress:    0.3,
		Message:     "Running dynamic programming optimization...",
		CurrentStep: "optimization",
		TotalSteps:  req.NumLineups,
		Timestamp:   time.Now(),
	}

	// Run enhanced optimization
	lineups, err := h.dpOptimizer.OptimizeWithDPV2(req.PlayerPool, config)
	if err != nil {
		h.logger.WithError(err).Error("Enhanced optimization failed")
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: "Enhanced optimization failed",
			Code:  "OPTIMIZATION_ERROR",
			Details: map[string]string{
				"error": err.Error(),
			},
		})
		return
	}

	if len(lineups) == 0 {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "No valid lineups could be generated",
			Code:  "NO_LINEUPS_GENERATED",
			Details: map[string]string{
				"strategy": string(req.Strategy),
				"players":  fmt.Sprintf("%d", len(req.PlayerPool)),
			},
		})
		return
	}

	progressChan <- types.ProgressUpdate{
		Type:        "enhanced_optimization",
		Progress:    0.8,
		Message:     "Generating analytics and exposure reports...",
		CurrentStep: "analysis",
		TotalSteps:  req.NumLineups,
		Timestamp:   time.Now(),
	}

	// Generate analytics
	analytics := h.generateOptimizationAnalytics(lineups, playerAnalytics, req)
	
	// Generate exposure report
	exposureManager := optimizer.NewExposureManager(req.ExposureConfig)
	for i, lineup := range lineups {
		for _, player := range lineup.Players {
			exposureManager.AddPlayerToLineup(player.ID, player.Team, i)
		}
		exposureManager.CompleteLineup()
	}
	exposureReport := exposureManager.GenerateExposureReport(req.PlayerPool)

	// Get performance metrics
	stats := h.dpOptimizer.GetStats()
	performanceMetrics := &PerformanceMetrics{
		OptimizationTime: time.Since(startTime),
		StatesExplored:   stats.StatesCached,
		CacheHitRate:     float64(stats.CacheHits) / float64(stats.CacheHits + stats.CacheMisses),
		MemoryUsage:      stats.MemoryUsage,
		Algorithm:        "enhanced_dp",
		PerformanceMode:  req.PerformanceMode,
	}

	// Create enhanced response
	response := OptimizationResponseV2{
		Lineups:           lineups,
		Analytics:         analytics,
		ExposureReport:    exposureReport,
		PerformanceMetrics: performanceMetrics,
		Strategy:          req.Strategy,
		CorrelationMatrix: make(map[string]float64), // TODO: Add correlation matrix
	}

	// Cache the result (using DPResult conversion)
	dpResult := &optimizer.DPResult{
		OptimalScore:     0.0, // Would need to be properly calculated
		OptimalPlayers:   []uuid.UUID{}, // Would need to be properly set
		StatesExplored:   int(stats.StatesCached),
		CacheHitRate:     float64(stats.CacheHits) / float64(stats.CacheHits + stats.CacheMisses),
		OptimizationTime: time.Since(startTime),
	}
	if err := h.cache.SetOptimizationResult(c.Request.Context(), cacheKey, dpResult, 12*time.Hour); err != nil {
		h.logger.WithError(err).Warn("Failed to cache enhanced optimization result")
	}

	// Send final progress update
	progressChan <- types.ProgressUpdate{
		Type:        "enhanced_optimization",
		Progress:    1.0,
		Message:     fmt.Sprintf("Enhanced optimization completed! Generated %d lineups in %v using %s strategy", len(lineups), time.Since(startTime), req.Strategy),
		CurrentStep: "completed",
		TotalSteps:  req.NumLineups,
		Timestamp:   time.Now(),
	}

	h.logger.WithFields(logrus.Fields{
		"lineups_generated": len(lineups),
		"execution_time":    time.Since(startTime),
		"strategy":          req.Strategy,
		"analytics_enabled": req.UseAnalytics,
		"cache_hit_rate":    performanceMetrics.CacheHitRate,
		"user_id":          req.UserID,
		"contest_id":       req.ContestID,
	}).Info("Enhanced optimization completed successfully")

	c.JSON(http.StatusOK, response)
}

// GetOptimizationStrategies returns available optimization strategies
func (h *OptimizationHandler) GetOptimizationStrategies(c *gin.Context) {
	strategies := map[string]interface{}{
		"strategies": []map[string]interface{}{
			{
				"id":          "ceiling",
				"name":        "Max Ceiling",
				"description": "Maximize upside potential for GPP tournaments",
				"best_for":    "Large field tournaments, high variance contests",
				"risk_level":  "high",
			},
			{
				"id":          "floor",
				"name":        "Max Floor",
				"description": "Maximize safety and consistency for cash games",
				"best_for":    "Cash games, head-to-head contests, 50/50s",
				"risk_level":  "low",
			},
			{
				"id":          "balanced",
				"name":        "Balanced",
				"description": "Balance between ceiling and floor",
				"best_for":    "Multi-entry tournaments, mixed contests",
				"risk_level":  "medium",
			},
			{
				"id":          "contrarian",
				"name":        "Contrarian",
				"description": "Target low ownership players for tournaments",
				"best_for":    "Large GPP tournaments, contrarian strategies",
				"risk_level":  "high",
			},
			{
				"id":          "correlation",
				"name":        "Correlation",
				"description": "Maximize correlation and stacking opportunities",
				"best_for":    "Game stacks, team stacks, correlated builds",
				"risk_level":  "medium",
			},
			{
				"id":          "value",
				"name":        "Value",
				"description": "Optimize for points per dollar efficiency",
				"best_for":    "Value-focused builds, salary cap challenges",
				"risk_level":  "low",
			},
		},
		"default_strategy": "balanced",
		"recommended_settings": map[string]interface{}{
			"gpp": map[string]interface{}{
				"strategy":            "ceiling",
				"use_analytics":       true,
				"performance_mode":    "quality",
				"ownership_strategy":  "contrarian",
			},
			"cash": map[string]interface{}{
				"strategy":            "floor",
				"use_analytics":       true,
				"performance_mode":    "balanced",
				"ownership_strategy":  "balanced",
			},
		},
	}

	c.JSON(http.StatusOK, strategies)
}

// ValidateOptimizationRequest validates an optimization request without running it
func (h *OptimizationHandler) ValidateOptimizationRequest(c *gin.Context) {
	var req types.OptimizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "Invalid request format",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	// Validate constraints
	if err := h.validateConstraints(req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "Invalid constraints",
			Code:  "INVALID_CONSTRAINTS",
			Details: map[string]string{
				"validation_error": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse{
		Message: "Optimization request is valid",
		Data: map[string]interface{}{
			"player_count":    len(req.PlayerPool),
			"max_lineups":     req.Settings.MaxLineups,
			"estimated_time":  h.estimateOptimizationTime(req),
		},
	})
}

// GetCacheStatus returns cache statistics
func (h *OptimizationHandler) GetCacheStatus(c *gin.Context) {
	status := h.cache.GetStatus(c.Request.Context())
	c.JSON(http.StatusOK, status)
}

// Helper methods

func (h *OptimizationHandler) generateCacheKey(req types.OptimizationRequest) string {
	// Create hash of the request for cache key
	hash := md5.New()
	hash.Write([]byte(fmt.Sprintf("%+v", req)))
	return fmt.Sprintf("optimization:%d:%x", req.ContestID, hash.Sum(nil))
}

func (h *OptimizationHandler) forwardProgressToWebSocket(userID uuid.UUID, progressChan <-chan types.ProgressUpdate) {
	for progress := range progressChan {
		h.wsHub.BroadcastToUser(userID, progress)
	}
}

func (h *OptimizationHandler) buildGolfCorrelationMatrix(players []types.OptimizationPlayer) map[string]float64 {
	matrix := make(map[string]float64)
	
	// Golf-specific correlation logic based on tee times
	teeTimeGroups := make(map[string][]types.OptimizationPlayer)
	for _, player := range players {
		teeTimeGroups[player.TeeTime] = append(teeTimeGroups[player.TeeTime], player)
	}

	// Higher correlation for players with same tee times
	for _, group := range teeTimeGroups {
		if len(group) > 1 {
			for i, p1 := range group {
				for j, p2 := range group {
					if i != j {
						key := fmt.Sprintf("%d-%d", p1.ID, p2.ID)
						matrix[key] = 0.3 // Moderate correlation for same tee time
					}
				}
			}
		}
	}

	return matrix
}

func (h *OptimizationHandler) buildGeneralCorrelationMatrix(players []types.OptimizationPlayer) map[string]float64 {
	matrix := make(map[string]float64)
	
	// Team-based correlations
	teamGroups := make(map[string][]types.OptimizationPlayer)
	for _, player := range players {
		teamGroups[player.Team] = append(teamGroups[player.Team], player)
	}

	// Higher correlation for players from same team
	for _, group := range teamGroups {
		if len(group) > 1 {
			for i, p1 := range group {
				for j, p2 := range group {
					if i != j {
						key := fmt.Sprintf("%d-%d", p1.ID, p2.ID)
						matrix[key] = 0.2 // Moderate correlation for same team
					}
				}
			}
		}
	}

	return matrix
}

func (h *OptimizationHandler) validateConstraints(req types.OptimizationRequest) error {
	// Check salary cap
	if req.Constraints.SalaryCap <= 0 {
		return fmt.Errorf("salary cap must be positive")
	}

	// Check position requirements
	if len(req.Constraints.PositionRequirements) == 0 {
		return fmt.Errorf("position requirements are required")
	}

	// Check max lineups
	if req.Settings.MaxLineups <= 0 {
		return fmt.Errorf("max lineups must be positive")
	}

	if req.Settings.MaxLineups > h.config.MaxLineups {
		return fmt.Errorf("max lineups exceeds limit of %d", h.config.MaxLineups)
	}

	return nil
}

func (h *OptimizationHandler) estimateOptimizationTime(req types.OptimizationRequest) string {
	// Simple estimation based on player count and lineups
	playerCount := len(req.PlayerPool)
	lineupCount := req.Settings.MaxLineups
	
	// Rough estimate: 1ms per player per lineup
	estimatedMs := playerCount * lineupCount
	duration := time.Duration(estimatedMs) * time.Millisecond
	
	return duration.String()
}

// Helper methods for enhanced API

// validateRequestV2 validates the enhanced optimization request
func (h *OptimizationHandler) validateRequestV2(req OptimizationRequestV2) error {
	if len(req.PlayerPool) == 0 {
		return fmt.Errorf("player pool cannot be empty")
	}
	
	if req.NumLineups <= 0 || req.NumLineups > 500 {
		return fmt.Errorf("num_lineups must be between 1 and 500, got %d", req.NumLineups)
	}
	
	if req.SalaryCap <= 0 {
		return fmt.Errorf("salary_cap must be positive, got %d", req.SalaryCap)
	}
	
	// Validate strategy
	validStrategies := map[optimizer.OptimizationObjective]bool{
		optimizer.MaxCeiling:  true,
		optimizer.MaxFloor:    true,
		optimizer.Balanced:    true,
		optimizer.Contrarian:  true,
		optimizer.Correlation: true,
		optimizer.Value:       true,
	}
	
	if !validStrategies[req.Strategy] {
		return fmt.Errorf("invalid strategy: %s", req.Strategy)
	}
	
	// Validate performance mode
	validModes := map[string]bool{
		"speed":    true,
		"balanced": true,
		"quality":  true,
	}
	
	if req.PerformanceMode != "" && !validModes[req.PerformanceMode] {
		return fmt.Errorf("invalid performance_mode: %s", req.PerformanceMode)
	}
	
	return nil
}

// generateCacheKeyV2 creates a cache key for enhanced optimization requests
func (h *OptimizationHandler) generateCacheKeyV2(req OptimizationRequestV2) string {
	hash := md5.New()
	
	// Include key parameters that affect optimization
	keyData := fmt.Sprintf("v2:strategy:%s:lineups:%d:salary:%d:analytics:%v:correlations:%v:mode:%s",
		req.Strategy, req.NumLineups, req.SalaryCap, req.UseAnalytics, req.UseCorrelations, req.PerformanceMode)
	
	// Add player IDs to ensure uniqueness
	for _, player := range req.PlayerPool {
		keyData += fmt.Sprintf(":%s", player.ID.String())
	}
	
	hash.Write([]byte(keyData))
	return fmt.Sprintf("optimization_v2:%s:%x", req.Strategy, hash.Sum(nil))
}

// convertToOptimizeConfigV2 converts API request to internal config
func (h *OptimizationHandler) convertToOptimizeConfigV2(req OptimizationRequestV2) optimizer.OptimizeConfigV2 {
	return optimizer.OptimizeConfigV2{
		SalaryCap:           req.SalaryCap,
		NumLineups:          req.NumLineups,
		MinDifferentPlayers: req.MinDifferentPlayers,
		UseCorrelations:     req.UseCorrelations,
		CorrelationWeight:   req.CorrelationWeight,
		StackingRules:       req.StackingRules,
		LockedPlayers:       req.LockedPlayers,
		ExcludedPlayers:     req.ExcludedPlayers,
		MinExposure:         req.MinExposure,
		MaxExposure:         req.MaxExposure,
		Strategy:            req.Strategy,
		PlayerAnalytics:     req.UseAnalytics,
		ExposureManagement:  req.ExposureConfig,
		PerformanceMode:     req.PerformanceMode,
		OwnershipStrategy:   req.OwnershipStrategy,
	}
}

// generateOptimizationAnalytics creates analytics for the optimization result
func (h *OptimizationHandler) generateOptimizationAnalytics(
	lineups []types.GeneratedLineup,
	playerAnalytics map[uuid.UUID]*optimizer.PlayerAnalytics,
	req OptimizationRequestV2,
) *OptimizationAnalytics {
	analytics := &OptimizationAnalytics{
		TotalLineups:       len(lineups),
		StackAnalysis:      make(map[string]int),
		PositionBreakdown:  make(map[string]int),
		SalaryDistribution: make(map[string]int),
		PlayerAnalytics:    playerAnalytics,
	}
	
	if len(lineups) == 0 {
		return analytics
	}
	
	// Calculate score statistics
	totalScore := 0.0
	scores := make([]float64, len(lineups))
	
	for i, lineup := range lineups {
		scores[i] = lineup.ProjectedPoints
		totalScore += lineup.ProjectedPoints
		
		if lineup.ProjectedPoints > analytics.TopScore {
			analytics.TopScore = lineup.ProjectedPoints
		}
		
		// Count positions
		for _, player := range lineup.Players {
			analytics.PositionBreakdown[player.Position]++
			
			// Salary distribution
			salaryBucket := h.getSalaryBucket(player.Salary)
			analytics.SalaryDistribution[salaryBucket]++
		}
		
		// Analyze stacks
		stacks := h.analyzeLineupStacks(lineup)
		for stack, count := range stacks {
			analytics.StackAnalysis[stack] += count
		}
	}
	
	analytics.AverageScore = totalScore / float64(len(lineups))
	
	// Calculate variance
	sumSquaredDiff := 0.0
	for _, score := range scores {
		diff := score - analytics.AverageScore
		sumSquaredDiff += diff * diff
	}
	analytics.ScoreVariance = sumSquaredDiff / float64(len(lineups))
	
	// Calculate diversity score
	analytics.DiversityScore = h.calculatePortfolioDiversity(lineups)
	
	return analytics
}

// convertCachedToV2Response converts cached result to V2 response format
func (h *OptimizationHandler) convertCachedToV2Response(cached *optimizer.DPResult, req OptimizationRequestV2) OptimizationResponseV2 {
	// This is a simplified conversion - in practice, you'd cache the full response
	return OptimizationResponseV2{
		Lineups:    []types.GeneratedLineup{}, // Would need to be cached
		Analytics:  &OptimizationAnalytics{},  // Would need to be cached
		Strategy:   req.Strategy,
		PerformanceMetrics: &PerformanceMetrics{
			Algorithm:       "enhanced_dp",
			PerformanceMode: req.PerformanceMode,
		},
	}
}

// getSalaryBucket returns salary bucket for analytics
func (h *OptimizationHandler) getSalaryBucket(salary int) string {
	switch {
	case salary >= 10000:
		return "premium_10k+"
	case salary >= 8000:
		return "high_8k-10k"
	case salary >= 6000:
		return "medium_6k-8k"
	case salary >= 4000:
		return "value_4k-6k"
	default:
		return "minimum_<4k"
	}
}

// analyzeLineupStacks analyzes stacking in a lineup
func (h *OptimizationHandler) analyzeLineupStacks(lineup types.GeneratedLineup) map[string]int {
	stacks := make(map[string]int)
	teamCounts := make(map[string]int)
	gameCounts := make(map[string]int)
	
	// Count team and game occurrences
	for _, player := range lineup.Players {
		teamCounts[player.Team]++
		
		// Create game key (simplified)
		gameKey := fmt.Sprintf("%s-game", player.Team)
		gameCounts[gameKey]++
	}
	
	// Identify stacks
	for team, count := range teamCounts {
		if count >= 2 {
			stackKey := fmt.Sprintf("%s_team_stack_%d", team, count)
			stacks[stackKey] = 1
		}
	}
	
	for game, count := range gameCounts {
		if count >= 3 {
			stackKey := fmt.Sprintf("%s_game_stack_%d", game, count)
			stacks[stackKey] = 1
		}
	}
	
	return stacks
}

// calculatePortfolioDiversity calculates diversity across all lineups
func (h *OptimizationHandler) calculatePortfolioDiversity(lineups []types.GeneratedLineup) float64 {
	if len(lineups) <= 1 {
		return 1.0
	}
	
	// Count unique players across all lineups
	playerCounts := make(map[uuid.UUID]int)
	totalPlayerSlots := 0
	
	for _, lineup := range lineups {
		for _, player := range lineup.Players {
			playerCounts[player.ID]++
			totalPlayerSlots++
		}
	}
	
	// Calculate diversity using normalized entropy
	if totalPlayerSlots == 0 {
		return 0.0
	}
	
	entropy := 0.0
	for _, count := range playerCounts {
		if count > 0 {
			p := float64(count) / float64(totalPlayerSlots)
			entropy -= p * math.Log2(p)
		}
	}
	
	// Normalize by maximum possible entropy
	maxEntropy := math.Log2(float64(len(playerCounts)))
	if maxEntropy > 0 {
		return entropy / maxEntropy
	}
	
	return 0.0
}

// convertToDPResult converts OptimizerResult to DPResult for caching
func (h *OptimizationHandler) convertToDPResult(result *optimizer.OptimizerResult, optimizationTime time.Duration) *optimizer.DPResult {
	// Calculate optimal score from the best lineup
	optimalScore := 0.0
	var optimalPlayers []uuid.UUID
	
	if len(result.Lineups) > 0 {
		bestLineup := result.Lineups[0]
		optimalScore = bestLineup.ProjectedPoints
		optimalPlayers = make([]uuid.UUID, len(bestLineup.Players))
		
		for i, player := range bestLineup.Players {
			optimalPlayers[i] = player.ID
		}
	}
	
	// Calculate cache hit rate (placeholder - would need actual metrics)
	cacheHitRate := 0.0
	if result.TotalCombinations > 0 {
		cacheHitRate = float64(result.ValidCombinations) / float64(result.TotalCombinations)
	}
	
	return &optimizer.DPResult{
		OptimalScore:     optimalScore,
		OptimalPlayers:   optimalPlayers,
		StatesExplored:   int(result.TotalCombinations),
		CacheHitRate:     cacheHitRate,
		OptimizationTime: optimizationTime,
	}
}

// convertFromDPResult converts DPResult back to OptimizationResult for API response
func (h *OptimizationHandler) convertFromDPResult(dpResult *optimizer.DPResult) types.OptimizationResult {
	// Create a single lineup from the optimal players (basic reconstruction)
	lineups := make([]types.GeneratedLineup, 0)
	
	if len(dpResult.OptimalPlayers) > 0 {
		// TODO: This is a simplified reconstruction - in a full implementation,
		// we'd need to store more lineup data in the cache or reconstruct properly
		lineup := types.GeneratedLineup{
			ID:               uuid.New().String(),
			Players:          make([]types.LineupPlayer, 0), // Would need to reconstruct from player IDs
			TotalSalary:      0,  // Would need to be reconstructed
			ProjectedPoints:  dpResult.OptimalScore,
			Exposure:         1.0,
		}
		lineups = append(lineups, lineup)
	}
	
	// Create basic metadata
	metadata := types.OptimizationMetadata{
		TotalLineups:      len(lineups),
		ExecutionTime:     dpResult.OptimizationTime,
		AverageUniqueess:  0.8, // Placeholder
		TopProjection:     dpResult.OptimalScore,
		AverageProjection: dpResult.OptimalScore,
		StacksGenerated:   0, // Placeholder
	}
	
	return types.OptimizationResult{
		Lineups:           lineups,
		Metadata:          metadata,
		CorrelationMatrix: make(map[string]float64), // Empty for cached results
	}
}

// convertOptimizationPlayerToPlayer converts from types.OptimizationPlayer to types.Player
func convertOptimizationPlayerToPlayer(op types.OptimizationPlayer) types.Player {
	return types.Player{
		ID:              op.ID,
		ExternalID:      op.ExternalID,
		Name:            op.Name,
		Team:            &op.Team,
		Position:        &op.Position,
		SalaryDK:        &op.Salary,
		ProjectedPoints: &op.ProjectedPoints,
		FloorPoints:     &op.FloorPoints,
		CeilingPoints:   &op.CeilingPoints,
		OwnershipDK:     &op.Ownership,
		// Set other fields to reasonable defaults
		SportID:         uuid.New(), // TODO: Should be passed in request
		IsActive:        func() *bool { b := true; return &b }(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}