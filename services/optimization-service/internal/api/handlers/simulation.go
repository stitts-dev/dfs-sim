package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/websocket"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/pkg/cache"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/pkg/simulator"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// SimulationHandler handles simulation-related endpoints
type SimulationHandler struct {
	db     *database.DB
	cache  *cache.OptimizationCacheService
	wsHub  *websocket.Hub
	config *config.Config
	logger *logrus.Logger
}

// NewSimulationHandler creates a new simulation handler
func NewSimulationHandler(
	db *database.DB,
	cache *cache.OptimizationCacheService,
	wsHub *websocket.Hub,
	config *config.Config,
	logger *logrus.Logger,
) *SimulationHandler {
	return &SimulationHandler{
		db:     db,
		cache:  cache,
		wsHub:  wsHub,
		config: config,
		logger: logger,
	}
}

// SimulationRequest represents a request to run Monte Carlo simulation
type SimulationRequest struct {
	Lineups          []types.GeneratedLineup `json:"lineups"`
	ContestType      string                  `json:"contest_type"` // "gpp" or "cash"
	Iterations       int                     `json:"iterations"`
	UserID           uuid.UUID               `json:"user_id,omitempty"`
	CorrelationMatrix map[string]float64     `json:"correlation_matrix,omitempty"`
}


// RunSimulation handles simulation requests
func (h *SimulationHandler) RunSimulation(c *gin.Context) {
	var req SimulationRequest
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

	// Validate simulation parameters
	if err := h.validateSimulationRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "Invalid simulation parameters",
			Code:  "INVALID_SIMULATION",
			Details: map[string]string{
				"validation_error": err.Error(),
			},
		})
		return
	}

	// Generate simulation ID
	simulationID := fmt.Sprintf("sim_%d", time.Now().UnixNano())

	// Create progress channel for WebSocket updates
	progressChan := make(chan types.ProgressUpdate, 100)
	defer close(progressChan)

	// Start progress forwarding to WebSocket if user ID provided
	if req.UserID != uuid.Nil {
		go h.forwardProgressToWebSocket(req.UserID, progressChan)
	}

	// Convert request lineups to shared types.Lineup format
	lineups := make([]types.Lineup, len(req.Lineups))
	for i, reqLineup := range req.Lineups {
		lineup := types.Lineup{
			ID:              uuid.New(),
			TotalSalary:     reqLineup.TotalSalary,
			ProjectedPoints: reqLineup.ProjectedPoints,
			Players:         make([]types.LineupPlayer, len(reqLineup.Players)),
		}
		
		for j, player := range reqLineup.Players {
			lineup.Players[j] = types.LineupPlayer{
				ID:              player.ID,
				Name:            player.Name,
				Position:        player.Position,
				Team:            player.Team,
				Salary:          player.Salary,
				ProjectedPoints: player.ProjectedPoints,
			}
		}
		
		lineups[i] = lineup
	}

	// Create a dummy contest for simulation (should come from request in real implementation)
	contest := types.Contest{
		ContestType: req.ContestType,
	}

	// Initialize simulator
	sim := simulator.NewMonteCarloSimulator(
		lineups,
		contest,
		req.Iterations,
		h.config.SimulationWorkers,
		h.logger,
	)
	
	// Set correlation matrix if provided
	if req.CorrelationMatrix != nil {
		sim.SetCorrelationMatrix(req.CorrelationMatrix)
	}

	// Send initial progress update
	progressChan <- types.ProgressUpdate{
		Type:        "simulation",
		Progress:    0.0,
		Message:     fmt.Sprintf("Starting simulation with %d iterations...", req.Iterations),
		CurrentStep: "initialization",
		TotalSteps:  req.Iterations,
		Timestamp:   time.Now(),
	}

	// Run simulation
	startTime := time.Now()
	results, err := sim.RunSimulation(c.Request.Context(), progressChan)
	if err != nil {
		h.logger.WithError(err).Error("Simulation failed")
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: "Simulation failed",
			Code:  "SIMULATION_ERROR",
			Details: map[string]string{
				"error": err.Error(),
			},
		})
		return
	}

	// Convert results to response format
	simulationResult := types.SimulationResult{
		ID:            simulationID,
		Iterations:    results.SimulationCount,
		ExecutionTime: results.ExecutionTime,
		LineupResults: h.convertLineupResults(results.Results),
		OverallStats:  h.calculateOverallStats(results.Results),
		ContestType:   req.ContestType,
		CreatedAt:     time.Now(),
	}

	// Cache the results
	cacheKey := fmt.Sprintf("simulation:%s", simulationID)
	if err := h.cache.SetSimulationResult(c.Request.Context(), cacheKey, &simulationResult, time.Hour); err != nil {
		h.logger.WithError(err).Warn("Failed to cache simulation result")
	}

	// Send final progress update
	progressChan <- types.ProgressUpdate{
		Type:        "simulation",
		Progress:    1.0,
		Message:     fmt.Sprintf("Simulation completed! Processed %d iterations in %v", req.Iterations, time.Since(startTime)),
		CurrentStep: "completed",
		TotalSteps:  req.Iterations,
		Timestamp:   time.Now(),
	}

	h.logger.WithFields(logrus.Fields{
		"simulation_id":  simulationID,
		"iterations":     req.Iterations,
		"execution_time": time.Since(startTime),
		"user_id":       req.UserID,
	}).Info("Simulation completed successfully")

	c.JSON(http.StatusOK, simulationResult)
}

// GetSimulationStatus returns the status of a running simulation
func (h *SimulationHandler) GetSimulationStatus(c *gin.Context) {
	simulationID := c.Param("id")
	
	// Check cache for results
	cacheKey := fmt.Sprintf("simulation:%s", simulationID)
	result, err := h.cache.GetSimulationResult(c.Request.Context(), cacheKey)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse{
			Error: "Simulation not found",
			Code:  "SIMULATION_NOT_FOUND",
		})
		return
	}

	status := map[string]interface{}{
		"id":         simulationID,
		"status":     "completed",
		"created_at": result.CreatedAt,
		"completed_at": result.CreatedAt.Add(result.ExecutionTime),
	}

	c.JSON(http.StatusOK, status)
}

// GetSimulationResults returns the results of a completed simulation
func (h *SimulationHandler) GetSimulationResults(c *gin.Context) {
	simulationID := c.Param("id")
	
	// Get results from cache
	cacheKey := fmt.Sprintf("simulation:%s", simulationID)
	result, err := h.cache.GetSimulationResult(c.Request.Context(), cacheKey)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse{
			Error: "Simulation results not found",
			Code:  "RESULTS_NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Helper methods

func (h *SimulationHandler) validateSimulationRequest(req SimulationRequest) error {
	if len(req.Lineups) == 0 {
		return fmt.Errorf("at least one lineup is required")
	}

	if req.Iterations <= 0 {
		return fmt.Errorf("iterations must be positive")
	}

	if req.Iterations > h.config.MaxSimulations {
		return fmt.Errorf("iterations exceed limit of %d", h.config.MaxSimulations)
	}

	if req.ContestType != "gpp" && req.ContestType != "cash" {
		return fmt.Errorf("contest type must be 'gpp' or 'cash'")
	}

	return nil
}

func (h *SimulationHandler) forwardProgressToWebSocket(userID uuid.UUID, progressChan <-chan types.ProgressUpdate) {
	for progress := range progressChan {
		h.wsHub.BroadcastToUser(userID, progress)
	}
}

func (h *SimulationHandler) convertLineupResults(results []simulator.LineupResult) []types.LineupSimulationResult {
	converted := make([]types.LineupSimulationResult, len(results))
	for i, result := range results {
		converted[i] = types.LineupSimulationResult{
			LineupID:      fmt.Sprintf("%d", result.LineupID),
			ExpectedScore: result.ExpectedPoints,
			ScoreVariance: result.PointsVariance,
			CashRate:      result.CashRate,
			ROI:           result.ROI,
			Top1Percent:   result.TopPercentFinish["top_1_percent"],
			Top10Percent:  result.TopPercentFinish["top_10_percent"],
			MedianFinish:  int(result.Percentiles["50th"]),
			Ceiling:       result.Percentiles["90th"],
			Floor:         result.Percentiles["10th"],
		}
	}
	return converted
}

func (h *SimulationHandler) calculateOverallStats(results []simulator.LineupResult) types.SimulationStats {
	if len(results) == 0 {
		return types.SimulationStats{}
	}

	var totalROI, totalCashRate float64
	bestROI := results[0].ROI
	worstROI := results[0].ROI

	for _, result := range results {
		totalROI += result.ROI
		totalCashRate += result.CashRate
		
		if result.ROI > bestROI {
			bestROI = result.ROI
		}
		if result.ROI < worstROI {
			worstROI = result.ROI
		}
	}

	avgROI := totalROI / float64(len(results))
	avgCashRate := totalCashRate / float64(len(results))

	// Calculate portfolio ROI (equal weight)
	portfolioROI := avgROI

	// Simple Sharpe ratio calculation (would need risk-free rate in production)
	variance := 0.0
	for _, result := range results {
		variance += (result.ROI - avgROI) * (result.ROI - avgROI)
	}
	variance /= float64(len(results))
	sharpe := 0.0
	if variance > 0 {
		sharpe = avgROI / (variance * variance) // Simplified calculation
	}

	return types.SimulationStats{
		TotalLineups:    len(results),
		AverageROI:      avgROI,
		BestROI:         bestROI,
		WorstROI:        worstROI,
		AverageCashRate: avgCashRate,
		PortfolioROI:    portfolioROI,
		Sharpe:          sharpe,
	}
}