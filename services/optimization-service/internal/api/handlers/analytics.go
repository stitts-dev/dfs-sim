package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/analytics/ml"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/analytics/performance"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/analytics/portfolio"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/websocket"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/pkg/analytics"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
)

// AnalyticsHandler handles analytics-related endpoints
type AnalyticsHandler struct {
	db                 *database.DB
	wsHub              *websocket.Hub
	config             *config.Config
	logger             *logrus.Logger
	featureExtractor   *ml.FeatureExtractor
	predictor          *ml.Predictor
	performanceTracker *performance.Tracker
	metricsCalculator  *analytics.MetricsCalculator
}

// PortfolioAnalyticsRequest represents portfolio analytics request
type PortfolioAnalyticsRequest struct {
	UserID      int                        `json:"user_id" binding:"required"`
	TimeFrame   string                     `json:"time_frame" binding:"required"` // "7d", "30d", "90d", "1y"
	StartDate   *time.Time                 `json:"start_date"`
	EndDate     *time.Time                 `json:"end_date"`
	Config      portfolio.PortfolioConfig  `json:"config"`
}

// MLPredictionRequest represents ML prediction request
type MLPredictionRequest struct {
	UserID      int                 `json:"user_id" binding:"required"`
	Features    map[string]float64  `json:"features" binding:"required"`
	ModelConfig ml.ModelConfig      `json:"model_config"`
}

// PerformanceAnalysisRequest represents performance analysis request
type PerformanceAnalysisRequest struct {
	UserID                int       `json:"user_id" binding:"required"`
	TimeFrame            string    `json:"time_frame" binding:"required"`
	StartDate            time.Time `json:"start_date"`
	EndDate              time.Time `json:"end_date"`
	Sports               []string  `json:"sports"`
	ContestTypes         []string  `json:"contest_types"`
	EnableAttribution    bool      `json:"enable_attribution"`
}

// FeatureExtractionRequest represents feature extraction request
type FeatureExtractionRequest struct {
	UserID     int `json:"user_id" binding:"required"`
	TimeWindow int `json:"time_window" binding:"required"`
}

// AnalyticsResponse represents a standard analytics response
type AnalyticsResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(
	db *database.DB,
	wsHub *websocket.Hub,
	config *config.Config,
	logger *logrus.Logger,
) *AnalyticsHandler {
	return &AnalyticsHandler{
		db:                 db,
		wsHub:              wsHub,
		config:             config,
		logger:             logger,
		featureExtractor:   ml.NewFeatureExtractor(),
		predictor:          ml.NewPredictor(ml.ModelConfig{}),
		performanceTracker: performance.NewTracker(),
		metricsCalculator:  analytics.NewMetricsCalculator(),
	}
}

// GetPortfolioAnalytics handles portfolio analytics calculation
func (h *AnalyticsHandler) GetPortfolioAnalytics(c *gin.Context) {
	var req PortfolioAnalyticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid portfolio analytics request")
		c.JSON(http.StatusBadRequest, AnalyticsResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	requestID := uuid.New().String()
	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"user_id":    req.UserID,
		"time_frame": req.TimeFrame,
	}).Info("Processing portfolio analytics request")

	// Set default date range if not provided
	if req.StartDate == nil || req.EndDate == nil {
		endDate := time.Now()
		startDate := h.getStartDateFromTimeFrame(req.TimeFrame, endDate)
		req.StartDate = &startDate
		req.EndDate = &endDate
	}

	// Fetch lineup data for portfolio analysis
	lineupData, err := h.fetchPortfolioData(c.Request.Context(), req.UserID, *req.StartDate, *req.EndDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch portfolio data")
		c.JSON(http.StatusInternalServerError, AnalyticsResponse{
			Success: false,
			Error:   "Failed to fetch portfolio data",
		})
		return
	}

	// Perform portfolio optimization
	result, err := portfolio.OptimizePortfolio(c.Request.Context(), lineupData, req.Config)
	if err != nil {
		h.logger.WithError(err).Error("Portfolio optimization failed")
		c.JSON(http.StatusInternalServerError, AnalyticsResponse{
			Success: false,
			Error:   fmt.Sprintf("Portfolio optimization failed: %v", err),
		})
		return
	}

	// Send real-time update via WebSocket
	h.sendAnalyticsUpdate(requestID, "portfolio_complete", result)

	c.JSON(http.StatusOK, AnalyticsResponse{
		Success: true,
		Data:    result,
		Meta: map[string]interface{}{
			"request_id": requestID,
			"user_id":    req.UserID,
			"time_frame": req.TimeFrame,
		},
	})
}

// GetMLPredictions handles ML prediction generation
func (h *AnalyticsHandler) GetMLPredictions(c *gin.Context) {
	var req MLPredictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid ML prediction request")
		c.JSON(http.StatusBadRequest, AnalyticsResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	requestID := uuid.New().String()
	h.logger.WithFields(logrus.Fields{
		"request_id":  requestID,
		"user_id":     req.UserID,
		"model_type":  req.ModelConfig.ModelType,
		"features":    len(req.Features),
	}).Info("Processing ML prediction request")

	// Generate prediction
	result, err := h.predictor.Predict(c.Request.Context(), req.Features, req.ModelConfig)
	if err != nil {
		h.logger.WithError(err).Error("ML prediction failed")
		c.JSON(http.StatusInternalServerError, AnalyticsResponse{
			Success: false,
			Error:   fmt.Sprintf("ML prediction failed: %v", err),
		})
		return
	}

	// Set user ID in result
	result.UserID = req.UserID

	// Send real-time update via WebSocket
	h.sendAnalyticsUpdate(requestID, "prediction_complete", result)

	c.JSON(http.StatusOK, AnalyticsResponse{
		Success: true,
		Data:    result,
		Meta: map[string]interface{}{
			"request_id": requestID,
			"user_id":    req.UserID,
			"model_type": req.ModelConfig.ModelType,
		},
	})
}

// GetPerformanceAnalysis handles performance analysis calculation
func (h *AnalyticsHandler) GetPerformanceAnalysis(c *gin.Context) {
	var req PerformanceAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid performance analysis request")
		c.JSON(http.StatusBadRequest, AnalyticsResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	requestID := uuid.New().String()
	h.logger.WithFields(logrus.Fields{
		"request_id":         requestID,
		"user_id":            req.UserID,
		"time_frame":         req.TimeFrame,
		"enable_attribution": req.EnableAttribution,
	}).Info("Processing performance analysis request")

	// Create tracker config
	config := performance.TrackerConfig{
		UserID:            req.UserID,
		TimeFrame:         req.TimeFrame,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		Sports:            req.Sports,
		ContestTypes:      req.ContestTypes,
		EnableAttribution: req.EnableAttribution,
	}

	// Perform performance analysis
	result, err := h.performanceTracker.AggregatePerformance(c.Request.Context(), req.UserID, config)
	if err != nil {
		h.logger.WithError(err).Error("Performance analysis failed")
		c.JSON(http.StatusInternalServerError, AnalyticsResponse{
			Success: false,
			Error:   fmt.Sprintf("Performance analysis failed: %v", err),
		})
		return
	}

	// Send real-time update via WebSocket
	h.sendAnalyticsUpdate(requestID, "performance_complete", result)

	c.JSON(http.StatusOK, AnalyticsResponse{
		Success: true,
		Data:    result,
		Meta: map[string]interface{}{
			"request_id": requestID,
			"user_id":    req.UserID,
			"time_frame": req.TimeFrame,
		},
	})
}

// ExtractFeatures handles feature extraction for ML models
func (h *AnalyticsHandler) ExtractFeatures(c *gin.Context) {
	var req FeatureExtractionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid feature extraction request")
		c.JSON(http.StatusBadRequest, AnalyticsResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	requestID := uuid.New().String()
	h.logger.WithFields(logrus.Fields{
		"request_id":  requestID,
		"user_id":     req.UserID,
		"time_window": req.TimeWindow,
	}).Info("Processing feature extraction request")

	// Fetch user history for feature extraction
	history, err := h.fetchUserHistory(c.Request.Context(), req.UserID, req.TimeWindow)
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch user history")
		c.JSON(http.StatusInternalServerError, AnalyticsResponse{
			Success: false,
			Error:   "Failed to fetch user history",
		})
		return
	}

	// Extract features
	features, err := h.featureExtractor.ExtractUserFeatures(c.Request.Context(), req.UserID, history, req.TimeWindow)
	if err != nil {
		h.logger.WithError(err).Error("Feature extraction failed")
		c.JSON(http.StatusInternalServerError, AnalyticsResponse{
			Success: false,
			Error:   fmt.Sprintf("Feature extraction failed: %v", err),
		})
		return
	}

	// Send real-time update via WebSocket
	h.sendAnalyticsUpdate(requestID, "features_complete", features)

	c.JSON(http.StatusOK, AnalyticsResponse{
		Success: true,
		Data:    features,
		Meta: map[string]interface{}{
			"request_id":  requestID,
			"user_id":     req.UserID,
			"time_window": req.TimeWindow,
		},
	})
}

// GetUserMetrics handles user-specific performance metrics
func (h *AnalyticsHandler) GetUserMetrics(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, AnalyticsResponse{
			Success: false,
			Error:   "Invalid user ID",
		})
		return
	}

	timeFrame := c.DefaultQuery("time_frame", "30d")
	
	h.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"time_frame": timeFrame,
	}).Info("Fetching user metrics")

	// Fetch user returns data
	returns, err := h.fetchUserReturns(c.Request.Context(), userID, timeFrame)
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch user returns")
		c.JSON(http.StatusInternalServerError, AnalyticsResponse{
			Success: false,
			Error:   "Failed to fetch user returns",
		})
		return
	}

	// Calculate comprehensive metrics
	benchmarkReturns := []float64{} // TODO: Implement benchmark data
	riskFreeRate := 0.02 / 252      // Daily risk-free rate (2% annual)
	
	metrics := h.metricsCalculator.CalculatePerformanceMetrics(returns, benchmarkReturns, riskFreeRate)
	riskMetrics := h.metricsCalculator.CalculateRiskMetrics(returns)

	response := map[string]interface{}{
		"performance_metrics": metrics,
		"risk_metrics":        riskMetrics,
		"data_points":         len(returns),
		"time_frame":          timeFrame,
	}

	c.JSON(http.StatusOK, AnalyticsResponse{
		Success: true,
		Data:    response,
		Meta: map[string]interface{}{
			"user_id":    userID,
			"time_frame": timeFrame,
		},
	})
}

// GetCorrelationMatrix handles correlation matrix calculation
func (h *AnalyticsHandler) GetCorrelationMatrix(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, AnalyticsResponse{
			Success: false,
			Error:   "Invalid user ID",
		})
		return
	}

	timeFrame := c.DefaultQuery("time_frame", "30d")
	
	h.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"time_frame": timeFrame,
	}).Info("Calculating correlation matrix")

	// Fetch correlation data (returns by sport/strategy)
	returnSeries, assetNames, err := h.fetchCorrelationData(c.Request.Context(), userID, timeFrame)
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch correlation data")
		c.JSON(http.StatusInternalServerError, AnalyticsResponse{
			Success: false,
			Error:   "Failed to fetch correlation data",
		})
		return
	}

	// Calculate correlation matrix
	corrMatrix := h.metricsCalculator.CalculateCorrelationMatrix(returnSeries, assetNames)

	c.JSON(http.StatusOK, AnalyticsResponse{
		Success: true,
		Data:    corrMatrix,
		Meta: map[string]interface{}{
			"user_id":    userID,
			"time_frame": timeFrame,
			"assets":     len(assetNames),
		},
	})
}

// GetAnalyticsHealth provides health check for analytics services
func (h *AnalyticsHandler) GetAnalyticsHealth(c *gin.Context) {
	health := map[string]interface{}{
		"status":              "healthy",
		"feature_extractor":   h.featureExtractor != nil,
		"predictor":           h.predictor != nil,
		"performance_tracker": h.performanceTracker != nil,
		"metrics_calculator":  h.metricsCalculator != nil,
		"timestamp":           time.Now(),
	}

	c.JSON(http.StatusOK, AnalyticsResponse{
		Success: true,
		Data:    health,
	})
}

// Helper functions

func (h *AnalyticsHandler) sendAnalyticsUpdate(requestID, eventType string, data interface{}) {
	if h.wsHub != nil {
		_ = map[string]interface{}{
			"request_id": requestID,
			"event_type": eventType,
			"data":       data,
			"timestamp":  time.Now(),
		}
		// TODO: Fix websocket broadcasting with proper user ID
		// h.wsHub.BroadcastToUser(userID, update)
	}
}

func (h *AnalyticsHandler) getStartDateFromTimeFrame(timeFrame string, endDate time.Time) time.Time {
	switch timeFrame {
	case "7d":
		return endDate.AddDate(0, 0, -7)
	case "30d":
		return endDate.AddDate(0, 0, -30)
	case "90d":
		return endDate.AddDate(0, 0, -90)
	case "1y":
		return endDate.AddDate(-1, 0, 0)
	default:
		return endDate.AddDate(0, 0, -30) // Default to 30 days
	}
}

func (h *AnalyticsHandler) fetchPortfolioData(ctx context.Context, userID int, startDate, endDate time.Time) ([]portfolio.LineupData, error) {
	// TODO: Implement database query to fetch user's lineup performance data
	// This would query the user_performance_history and related tables
	return []portfolio.LineupData{}, nil
}

func (h *AnalyticsHandler) fetchUserHistory(ctx context.Context, userID int, timeWindow int) ([]ml.UserLineupHistory, error) {
	// TODO: Implement database query to fetch user's lineup history
	// This would query lineups, players, contests, and performance data
	return []ml.UserLineupHistory{}, nil
}

func (h *AnalyticsHandler) fetchUserReturns(ctx context.Context, userID int, timeFrame string) ([]float64, error) {
	// TODO: Implement database query to fetch user's daily/weekly returns
	// This would aggregate ROI data from user_performance_history
	return []float64{}, nil
}

func (h *AnalyticsHandler) fetchCorrelationData(ctx context.Context, userID int, timeFrame string) ([][]float64, []string, error) {
	// TODO: Implement database query to fetch return series by sport/strategy
	// This would group returns by different categories for correlation analysis
	return [][]float64{}, []string{}, nil
}

// RegisterAnalyticsRoutes registers all analytics routes
func RegisterAnalyticsRoutes(router *gin.RouterGroup, handler *AnalyticsHandler) {
	analytics := router.Group("/analytics")
	{
		// Portfolio analytics
		analytics.POST("/portfolio", handler.GetPortfolioAnalytics)
		
		// ML predictions
		analytics.POST("/predictions", handler.GetMLPredictions)
		
		// Performance analysis
		analytics.POST("/performance", handler.GetPerformanceAnalysis)
		
		// Feature extraction
		analytics.POST("/features", handler.ExtractFeatures)
		
		// User-specific metrics
		analytics.GET("/users/:user_id/metrics", handler.GetUserMetrics)
		analytics.GET("/users/:user_id/correlation", handler.GetCorrelationMatrix)
		
		// Health check
		analytics.GET("/health", handler.GetAnalyticsHealth)
	}
}