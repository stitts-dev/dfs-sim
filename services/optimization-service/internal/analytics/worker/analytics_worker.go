package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/analytics/ml"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/analytics/performance"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/analytics/portfolio"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/websocket"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/pkg/logger"
)

// AnalyticsWorker handles background analytics aggregation and processing
type AnalyticsWorker struct {
	db                 *database.DB
	wsHub              *websocket.Hub
	performanceTracker *performance.Tracker
	featureExtractor   *ml.FeatureExtractor
	predictor          *ml.Predictor
	logger             *logrus.Logger
	
	// Worker control
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	isRunning  bool
	mutex      sync.RWMutex
	
	// Configuration
	config WorkerConfig
	
	// Metrics
	stats WorkerStats
}

// WorkerConfig defines configuration for the analytics worker
type WorkerConfig struct {
	PerformanceAggregationInterval time.Duration `json:"performance_aggregation_interval"`
	PortfolioAnalysisInterval      time.Duration `json:"portfolio_analysis_interval"`
	MLModelRefreshInterval         time.Duration `json:"ml_model_refresh_interval"`
	DataCleanupInterval            time.Duration `json:"data_cleanup_interval"`
	BatchSize                      int           `json:"batch_size"`
	MaxRetries                     int           `json:"max_retries"`
	RetryDelay                     time.Duration `json:"retry_delay"`
	EnableRealTimeUpdates          bool          `json:"enable_real_time_updates"`
	DataRetentionDays              int           `json:"data_retention_days"`
}

// WorkerStats tracks worker performance and activity
type WorkerStats struct {
	StartTime                   time.Time         `json:"start_time"`
	LastPerformanceAggregation  time.Time         `json:"last_performance_aggregation"`
	LastPortfolioAnalysis       time.Time         `json:"last_portfolio_analysis"`
	LastMLModelRefresh          time.Time         `json:"last_ml_model_refresh"`
	LastDataCleanup             time.Time         `json:"last_data_cleanup"`
	UsersProcessed              int64             `json:"users_processed"`
	PerformanceReportsGenerated int64             `json:"performance_reports_generated"`
	PortfolioAnalysesCompleted  int64             `json:"portfolio_analyses_completed"`
	MLPredictionsGenerated      int64             `json:"ml_predictions_generated"`
	Errors                      map[string]int64  `json:"errors"`
	ProcessingTimes             map[string]string `json:"processing_times"`
	mutex                       sync.RWMutex
}

// NewAnalyticsWorker creates a new analytics worker instance
func NewAnalyticsWorker(
	db *database.DB,
	wsHub *websocket.Hub,
	config WorkerConfig,
) *AnalyticsWorker {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &AnalyticsWorker{
		db:                 db,
		wsHub:              wsHub,
		performanceTracker: performance.NewTracker(),
		featureExtractor:   ml.NewFeatureExtractor(),
		predictor:          ml.NewPredictor(ml.ModelConfig{}),
		logger:             logger.GetLogger(),
		ctx:                ctx,
		cancel:             cancel,
		config:             config,
		stats: WorkerStats{
			StartTime: time.Now(),
			Errors:    make(map[string]int64),
			ProcessingTimes: make(map[string]string),
		},
	}
}

// Start begins the background analytics processing
func (aw *AnalyticsWorker) Start() error {
	aw.mutex.Lock()
	defer aw.mutex.Unlock()
	
	if aw.isRunning {
		return fmt.Errorf("analytics worker is already running")
	}
	
	aw.isRunning = true
	aw.stats.StartTime = time.Now()
	
	aw.logger.WithFields(logrus.Fields{
		"performance_interval": aw.config.PerformanceAggregationInterval,
		"portfolio_interval":   aw.config.PortfolioAnalysisInterval,
		"ml_interval":          aw.config.MLModelRefreshInterval,
		"cleanup_interval":     aw.config.DataCleanupInterval,
	}).Info("Starting analytics worker")
	
	// Start background processing goroutines
	aw.wg.Add(4)
	go aw.performanceAggregationWorker()
	go aw.portfolioAnalysisWorker()
	go aw.mlModelRefreshWorker()
	go aw.dataCleanupWorker()
	
	return nil
}

// Stop gracefully stops the analytics worker
func (aw *AnalyticsWorker) Stop() error {
	aw.mutex.Lock()
	defer aw.mutex.Unlock()
	
	if !aw.isRunning {
		return fmt.Errorf("analytics worker is not running")
	}
	
	aw.logger.Info("Stopping analytics worker")
	aw.cancel()
	aw.wg.Wait()
	aw.isRunning = false
	
	aw.logger.Info("Analytics worker stopped successfully")
	return nil
}

// IsRunning returns whether the worker is currently running
func (aw *AnalyticsWorker) IsRunning() bool {
	aw.mutex.RLock()
	defer aw.mutex.RUnlock()
	return aw.isRunning
}

// GetStats returns current worker statistics
func (aw *AnalyticsWorker) GetStats() WorkerStats {
	aw.stats.mutex.RLock()
	defer aw.stats.mutex.RUnlock()
	
	// Create a copy to avoid race conditions
	statsCopy := aw.stats
	statsCopy.Errors = make(map[string]int64)
	statsCopy.ProcessingTimes = make(map[string]string)
	
	for k, v := range aw.stats.Errors {
		statsCopy.Errors[k] = v
	}
	
	for k, v := range aw.stats.ProcessingTimes {
		statsCopy.ProcessingTimes[k] = v
	}
	
	return statsCopy
}

// Performance aggregation worker
func (aw *AnalyticsWorker) performanceAggregationWorker() {
	defer aw.wg.Done()
	
	ticker := time.NewTicker(aw.config.PerformanceAggregationInterval)
	defer ticker.Stop()
	
	aw.logger.Info("Started performance aggregation worker")
	
	for {
		select {
		case <-aw.ctx.Done():
			aw.logger.Info("Performance aggregation worker stopped")
			return
		case <-ticker.C:
			aw.processPerformanceAggregation()
		}
	}
}

// Portfolio analysis worker
func (aw *AnalyticsWorker) portfolioAnalysisWorker() {
	defer aw.wg.Done()
	
	ticker := time.NewTicker(aw.config.PortfolioAnalysisInterval)
	defer ticker.Stop()
	
	aw.logger.Info("Started portfolio analysis worker")
	
	for {
		select {
		case <-aw.ctx.Done():
			aw.logger.Info("Portfolio analysis worker stopped")
			return
		case <-ticker.C:
			aw.processPortfolioAnalysis()
		}
	}
}

// ML model refresh worker
func (aw *AnalyticsWorker) mlModelRefreshWorker() {
	defer aw.wg.Done()
	
	ticker := time.NewTicker(aw.config.MLModelRefreshInterval)
	defer ticker.Stop()
	
	aw.logger.Info("Started ML model refresh worker")
	
	for {
		select {
		case <-aw.ctx.Done():
			aw.logger.Info("ML model refresh worker stopped")
			return
		case <-ticker.C:
			aw.processMLModelRefresh()
		}
	}
}

// Data cleanup worker
func (aw *AnalyticsWorker) dataCleanupWorker() {
	defer aw.wg.Done()
	
	ticker := time.NewTicker(aw.config.DataCleanupInterval)
	defer ticker.Stop()
	
	aw.logger.Info("Started data cleanup worker")
	
	for {
		select {
		case <-aw.ctx.Done():
			aw.logger.Info("Data cleanup worker stopped")
			return
		case <-ticker.C:
			aw.processDataCleanup()
		}
	}
}

// Process performance aggregation for all users
func (aw *AnalyticsWorker) processPerformanceAggregation() {
	startTime := time.Now()
	aw.logger.Info("Starting performance aggregation cycle")
	
	// Get active users with recent activity
	users, err := aw.getActiveUsers()
	if err != nil {
		aw.incrementError("performance_aggregation")
		aw.logger.WithError(err).Error("Failed to get active users for performance aggregation")
		return
	}
	
	processed := 0
	for _, userID := range users {
		if err := aw.processUserPerformance(userID); err != nil {
			aw.incrementError("user_performance")
			aw.logger.WithError(err).WithField("user_id", userID).Error("Failed to process user performance")
			continue
		}
		processed++
	}
	
	// Update stats
	aw.stats.mutex.Lock()
	aw.stats.LastPerformanceAggregation = time.Now()
	aw.stats.UsersProcessed += int64(processed)
	aw.stats.PerformanceReportsGenerated += int64(processed)
	aw.stats.ProcessingTimes["performance_aggregation"] = time.Since(startTime).String()
	aw.stats.mutex.Unlock()
	
	aw.logger.WithFields(logrus.Fields{
		"users_processed": processed,
		"duration":        time.Since(startTime),
	}).Info("Completed performance aggregation cycle")
}

// Process portfolio analysis for eligible users
func (aw *AnalyticsWorker) processPortfolioAnalysis() {
	startTime := time.Now()
	aw.logger.Info("Starting portfolio analysis cycle")
	
	// Get users eligible for portfolio analysis
	users, err := aw.getPortfolioEligibleUsers()
	if err != nil {
		aw.incrementError("portfolio_analysis")
		aw.logger.WithError(err).Error("Failed to get users for portfolio analysis")
		return
	}
	
	processed := 0
	for _, userID := range users {
		if err := aw.processUserPortfolio(userID); err != nil {
			aw.incrementError("user_portfolio")
			aw.logger.WithError(err).WithField("user_id", userID).Error("Failed to process user portfolio")
			continue
		}
		processed++
	}
	
	// Update stats
	aw.stats.mutex.Lock()
	aw.stats.LastPortfolioAnalysis = time.Now()
	aw.stats.PortfolioAnalysesCompleted += int64(processed)
	aw.stats.ProcessingTimes["portfolio_analysis"] = time.Since(startTime).String()
	aw.stats.mutex.Unlock()
	
	aw.logger.WithFields(logrus.Fields{
		"users_processed": processed,
		"duration":        time.Since(startTime),
	}).Info("Completed portfolio analysis cycle")
}

// Process ML model refresh and prediction generation
func (aw *AnalyticsWorker) processMLModelRefresh() {
	startTime := time.Now()
	aw.logger.Info("Starting ML model refresh cycle")
	
	// Refresh ML models with recent data
	if err := aw.refreshMLModels(); err != nil {
		aw.incrementError("ml_model_refresh")
		aw.logger.WithError(err).Error("Failed to refresh ML models")
		return
	}
	
	// Generate new predictions for active users
	users, err := aw.getPredictionEligibleUsers()
	if err != nil {
		aw.incrementError("ml_predictions")
		aw.logger.WithError(err).Error("Failed to get users for ML predictions")
		return
	}
	
	predictions := 0
	for _, userID := range users {
		if err := aw.generateUserPredictions(userID); err != nil {
			aw.incrementError("user_predictions")
			aw.logger.WithError(err).WithField("user_id", userID).Error("Failed to generate user predictions")
			continue
		}
		predictions++
	}
	
	// Update stats
	aw.stats.mutex.Lock()
	aw.stats.LastMLModelRefresh = time.Now()
	aw.stats.MLPredictionsGenerated += int64(predictions)
	aw.stats.ProcessingTimes["ml_model_refresh"] = time.Since(startTime).String()
	aw.stats.mutex.Unlock()
	
	aw.logger.WithFields(logrus.Fields{
		"predictions_generated": predictions,
		"duration":              time.Since(startTime),
	}).Info("Completed ML model refresh cycle")
}

// Process data cleanup and maintenance
func (aw *AnalyticsWorker) processDataCleanup() {
	startTime := time.Now()
	aw.logger.Info("Starting data cleanup cycle")
	
	// Clean up old analytics data
	cutoffDate := time.Now().AddDate(0, 0, -aw.config.DataRetentionDays)
	
	if err := aw.cleanupOldData(cutoffDate); err != nil {
		aw.incrementError("data_cleanup")
		aw.logger.WithError(err).Error("Failed to cleanup old data")
		return
	}
	
	// Update stats
	aw.stats.mutex.Lock()
	aw.stats.LastDataCleanup = time.Now()
	aw.stats.ProcessingTimes["data_cleanup"] = time.Since(startTime).String()
	aw.stats.mutex.Unlock()
	
	aw.logger.WithFields(logrus.Fields{
		"cutoff_date": cutoffDate,
		"duration":    time.Since(startTime),
	}).Info("Completed data cleanup cycle")
}

// Helper methods for database operations

func (aw *AnalyticsWorker) getActiveUsers() ([]int, error) {
	// TODO: Implement database query to get users with recent activity
	// This would query lineups or contests within the last few days
	return []int{}, nil
}

func (aw *AnalyticsWorker) getPortfolioEligibleUsers() ([]int, error) {
	// TODO: Implement database query to get users with sufficient lineup history
	// for meaningful portfolio analysis (e.g., at least 10 lineups)
	return []int{}, nil
}

func (aw *AnalyticsWorker) getPredictionEligibleUsers() ([]int, error) {
	// TODO: Implement database query to get users who would benefit from
	// updated ML predictions based on their activity patterns
	return []int{}, nil
}

func (aw *AnalyticsWorker) processUserPerformance(userID int) error {
	// Create tracker config for this user
	config := performance.TrackerConfig{
		UserID:            userID,
		TimeFrame:         "30d",
		StartDate:         time.Now().AddDate(0, 0, -30),
		EndDate:           time.Now(),
		EnableAttribution: true,
	}
	
	// Generate performance report
	report, err := aw.performanceTracker.AggregatePerformance(aw.ctx, userID, config)
	if err != nil {
		return fmt.Errorf("failed to aggregate performance for user %d: %w", userID, err)
	}
	
	// Store results in database
	if err := aw.storePerformanceReport(userID, report); err != nil {
		return fmt.Errorf("failed to store performance report for user %d: %w", userID, err)
	}
	
	// Send real-time update if enabled
	if aw.config.EnableRealTimeUpdates && aw.wsHub != nil {
		aw.sendPerformanceUpdate(userID, report)
	}
	
	return nil
}

func (aw *AnalyticsWorker) processUserPortfolio(userID int) error {
	// Fetch user's lineup data for portfolio analysis
	lineupData, err := aw.getUserLineupData(userID)
	if err != nil {
		return fmt.Errorf("failed to get lineup data for user %d: %w", userID, err)
	}
	
	if len(lineupData) == 0 {
		return nil // Skip users with no lineup data
	}
	
	// Perform portfolio optimization
	config := portfolio.PortfolioConfig{
		RiskAversion:        0.5,
		UseRiskParity:       true,
		MaxIterations:       1000,
		ConvergenceThreshold: 1e-6,
	}
	
	result, err := portfolio.OptimizePortfolio(aw.ctx, lineupData, config)
	if err != nil {
		return fmt.Errorf("failed to optimize portfolio for user %d: %w", userID, err)
	}
	
	// Store portfolio analysis results
	if err := aw.storePortfolioAnalysis(userID, result); err != nil {
		return fmt.Errorf("failed to store portfolio analysis for user %d: %w", userID, err)
	}
	
	// Send real-time update if enabled
	if aw.config.EnableRealTimeUpdates && aw.wsHub != nil {
		aw.sendPortfolioUpdate(userID, result)
	}
	
	return nil
}

func (aw *AnalyticsWorker) generateUserPredictions(userID int) error {
	// Extract features for the user
	history, err := aw.getUserHistory(userID)
	if err != nil {
		return fmt.Errorf("failed to get user history for user %d: %w", userID, err)
	}
	
	features, err := aw.featureExtractor.ExtractUserFeatures(aw.ctx, userID, history, 30)
	if err != nil {
		return fmt.Errorf("failed to extract features for user %d: %w", userID, err)
	}
	
	// Generate predictions
	modelConfig := ml.ModelConfig{
		ModelType:    "neural_network",
		EpochCount:   100,
		LearningRate: 0.001,
	}
	
	prediction, err := aw.predictor.Predict(aw.ctx, features.Features, modelConfig)
	if err != nil {
		return fmt.Errorf("failed to generate prediction for user %d: %w", userID, err)
	}
	
	// Store prediction results
	if err := aw.storePrediction(userID, prediction); err != nil {
		return fmt.Errorf("failed to store prediction for user %d: %w", userID, err)
	}
	
	// Send real-time update if enabled
	if aw.config.EnableRealTimeUpdates && aw.wsHub != nil {
		aw.sendPredictionUpdate(userID, prediction)
	}
	
	return nil
}

func (aw *AnalyticsWorker) refreshMLModels() error {
	// TODO: Implement ML model retraining with recent data
	// This would fetch recent performance data and retrain models
	aw.logger.Info("ML model refresh completed")
	return nil
}

func (aw *AnalyticsWorker) cleanupOldData(cutoffDate time.Time) error {
	// TODO: Implement database cleanup for old analytics data
	// This would remove or archive old entries from analytics tables
	aw.logger.WithField("cutoff_date", cutoffDate).Info("Data cleanup completed")
	return nil
}

// Database operation helpers (to be implemented)

func (aw *AnalyticsWorker) storePerformanceReport(userID int, report *performance.PerformanceReport) error {
	// TODO: Store performance report in user_performance_history table
	return nil
}

func (aw *AnalyticsWorker) storePortfolioAnalysis(userID int, result *portfolio.PortfolioResult) error {
	// TODO: Store portfolio analysis in portfolio_analytics table
	return nil
}

func (aw *AnalyticsWorker) storePrediction(userID int, prediction *ml.PredictionResult) error {
	// TODO: Store ML prediction in ml_predictions table
	return nil
}

func (aw *AnalyticsWorker) getUserLineupData(userID int) ([]portfolio.LineupData, error) {
	// TODO: Fetch user's lineup performance data for portfolio analysis
	return []portfolio.LineupData{}, nil
}

func (aw *AnalyticsWorker) getUserHistory(userID int) ([]ml.UserLineupHistory, error) {
	// TODO: Fetch user's lineup history for feature extraction
	return []ml.UserLineupHistory{}, nil
}

// Real-time update helpers

func (aw *AnalyticsWorker) sendPerformanceUpdate(userID int, report *performance.PerformanceReport) {
	event := websocket.AnalyticsEvent{
		Type:      "performance_update",
		UserID:    aw.convertUserID(userID),
		EventID:   fmt.Sprintf("perf_%d_%d", userID, time.Now().Unix()),
		Category:  "performance",
		Data:      report,
		Timestamp: time.Now().Unix(),
	}
	
	aw.wsHub.SendAnalyticsEvent(event)
}

func (aw *AnalyticsWorker) sendPortfolioUpdate(userID int, result *portfolio.PortfolioResult) {
	event := websocket.AnalyticsEvent{
		Type:      "portfolio_update",
		UserID:    aw.convertUserID(userID),
		EventID:   fmt.Sprintf("port_%d_%d", userID, time.Now().Unix()),
		Category:  "portfolio",
		Data:      result,
		Timestamp: time.Now().Unix(),
	}
	
	aw.wsHub.SendAnalyticsEvent(event)
}

func (aw *AnalyticsWorker) sendPredictionUpdate(userID int, prediction *ml.PredictionResult) {
	event := websocket.AnalyticsEvent{
		Type:      "prediction_update",
		UserID:    aw.convertUserID(userID),
		EventID:   fmt.Sprintf("pred_%d_%d", userID, time.Now().Unix()),
		Category:  "ml",
		Data:      prediction,
		Timestamp: time.Now().Unix(),
	}
	
	aw.wsHub.SendAnalyticsEvent(event)
}

// Utility methods

func (aw *AnalyticsWorker) convertUserID(userID int) string {
	// TODO: Convert int user ID to UUID if needed
	// For now, return string representation
	return fmt.Sprintf("%d", userID)
}

func (aw *AnalyticsWorker) incrementError(errorType string) {
	aw.stats.mutex.Lock()
	defer aw.stats.mutex.Unlock()
	aw.stats.Errors[errorType]++
}

// GetDefaultConfig returns default configuration for the analytics worker
func GetDefaultConfig() WorkerConfig {
	return WorkerConfig{
		PerformanceAggregationInterval: 1 * time.Hour,
		PortfolioAnalysisInterval:      4 * time.Hour,
		MLModelRefreshInterval:         12 * time.Hour,
		DataCleanupInterval:            24 * time.Hour,
		BatchSize:                      100,
		MaxRetries:                     3,
		RetryDelay:                     5 * time.Minute,
		EnableRealTimeUpdates:          true,
		DataRetentionDays:              90,
	}
}