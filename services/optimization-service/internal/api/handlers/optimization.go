package handlers

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/optimizer"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/websocket"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/pkg/cache"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// OptimizationHandler handles optimization-related endpoints
type OptimizationHandler struct {
	db     *database.DB
	cache  *cache.OptimizationCacheService
	wsHub  *websocket.Hub
	config *config.Config
	logger *logrus.Logger
}

// NewOptimizationHandler creates a new optimization handler
func NewOptimizationHandler(
	db *database.DB,
	cache *cache.OptimizationCacheService,
	wsHub *websocket.Hub,
	config *config.Config,
	logger *logrus.Logger,
) *OptimizationHandler {
	return &OptimizationHandler{
		db:     db,
		cache:  cache,
		wsHub:  wsHub,
		config: config,
		logger: logger,
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
		c.JSON(http.StatusOK, cached)
		return
	}

	// Create progress channel for WebSocket updates
	progressChan := make(chan types.ProgressUpdate, 100)
	defer close(progressChan)

	// Start progress forwarding to WebSocket if user ID provided
	if req.UserID > 0 {
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

	// Initialize optimizer
	opt := optimizer.NewOptimizer(
		req.PlayerPool,
		req.Constraints,
		correlationMatrix,
		h.logger,
	)

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
	result, err := opt.OptimizeWithProgress(req.Settings, progressChan)
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

	// Create response
	response := types.OptimizationResult{
		Lineups:           result.Lineups,
		Metadata:          result.Metadata,
		CorrelationMatrix: correlationMatrix,
	}

	// Cache the result
	if err := h.cache.SetOptimizationResult(c.Request.Context(), cacheKey, &response, 24*time.Hour); err != nil {
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

func (h *OptimizationHandler) forwardProgressToWebSocket(userID uint, progressChan <-chan types.ProgressUpdate) {
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