package ownership

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/models"
)

// TrendAnalyzer analyzes ownership trends and predicts future movements
type TrendAnalyzer struct {
	redisClient *redis.Client
	logger      *logrus.Logger
	
	// Configuration
	maxDataPoints    int           // Maximum data points to keep for trend analysis
	minDataPoints    int           // Minimum data points required for reliable trends
	trendWindow      time.Duration // Time window for trend calculations
	cacheTTL         time.Duration // Cache TTL for trend data
}

// TrendData represents historical ownership data for trend analysis
type TrendData struct {
	ContestID   string                 `json:"contest_id"`
	PlayerID    uint                   `json:"player_id"`
	DataPoints  []OwnershipDataPoint   `json:"data_points"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// OwnershipDataPoint represents a single ownership measurement
type OwnershipDataPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	Ownership  float64   `json:"ownership"`
	TotalEntries int     `json:"total_entries"`
}

// TrendMetrics contains calculated trend metrics
type TrendMetrics struct {
	PlayerID         uint      `json:"player_id"`
	CurrentOwnership float64   `json:"current_ownership"`
	PrevOwnership    float64   `json:"prev_ownership"`
	OwnershipChange  float64   `json:"ownership_change"`
	Velocity         float64   `json:"velocity"`          // Rate of change per hour
	Acceleration     float64   `json:"acceleration"`      // Change in velocity
	TrendDirection   string    `json:"trend_direction"`   // "up", "down", "stable"
	TrendStrength    float64   `json:"trend_strength"`    // 0-1, strength of the trend
	Prediction       float64   `json:"prediction"`        // Predicted ownership at lock
	Confidence       float64   `json:"confidence"`        // 0-1, confidence in prediction
	LastCalculated   time.Time `json:"last_calculated"`
}

// PredictionModel contains parameters for ownership prediction
type PredictionModel struct {
	LinearWeight     float64 `json:"linear_weight"`      // Weight for linear trend
	AccelerationWeight float64 `json:"acceleration_weight"` // Weight for acceleration component
	SeasonalWeight   float64 `json:"seasonal_weight"`    // Weight for seasonal patterns
	NoiseReduction   float64 `json:"noise_reduction"`    // Factor to reduce noise in predictions
}

// NewTrendAnalyzer creates a new trend analyzer
func NewTrendAnalyzer(redisClient *redis.Client, logger *logrus.Logger) *TrendAnalyzer {
	return &TrendAnalyzer{
		redisClient:   redisClient,
		logger:        logger,
		maxDataPoints: 100,         // Keep last 100 data points per player
		minDataPoints: 3,           // Need at least 3 points for trends
		trendWindow:   6 * time.Hour, // 6-hour window for trend analysis
		cacheTTL:      5 * time.Minute, // Cache trends for 5 minutes
	}
}

// UpdateTrends updates trend data for all players in a contest
func (ta *TrendAnalyzer) UpdateTrends(contestID string, ownership map[uint]float64, timestamp time.Time) error {
	ctx := context.Background()
	
	for playerID, ownershipPct := range ownership {
		if err := ta.updatePlayerTrend(ctx, contestID, playerID, ownershipPct, timestamp); err != nil {
			ta.logger.WithError(err).WithFields(logrus.Fields{
				"contest_id": contestID,
				"player_id":  playerID,
			}).Error("Failed to update player trend")
		}
	}
	
	return nil
}

// updatePlayerTrend updates trend data for a specific player
func (ta *TrendAnalyzer) updatePlayerTrend(ctx context.Context, contestID string, playerID uint, ownership float64, timestamp time.Time) error {
	cacheKey := fmt.Sprintf("trend:%s:%d", contestID, playerID)
	
	// Get existing trend data
	var trendData TrendData
	existingData, err := ta.redisClient.Get(ctx, cacheKey).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get existing trend data: %w", err)
	}
	
	if err == nil {
		if err := json.Unmarshal([]byte(existingData), &trendData); err != nil {
			ta.logger.WithError(err).Warn("Failed to unmarshal trend data, starting fresh")
			trendData = TrendData{
				ContestID:  contestID,
				PlayerID:   playerID,
				DataPoints: make([]OwnershipDataPoint, 0),
			}
		}
	} else {
		// No existing data, create new
		trendData = TrendData{
			ContestID:  contestID,
			PlayerID:   playerID,
			DataPoints: make([]OwnershipDataPoint, 0),
		}
	}
	
	// Add new data point
	newDataPoint := OwnershipDataPoint{
		Timestamp: timestamp,
		Ownership: ownership,
		// TotalEntries would be provided in real implementation
		TotalEntries: 1000, // Mock value
	}
	
	trendData.DataPoints = append(trendData.DataPoints, newDataPoint)
	trendData.UpdatedAt = timestamp
	
	// Remove old data points to keep within limits
	if len(trendData.DataPoints) > ta.maxDataPoints {
		trendData.DataPoints = trendData.DataPoints[len(trendData.DataPoints)-ta.maxDataPoints:]
	}
	
	// Remove data points outside the trend window
	cutoffTime := timestamp.Add(-ta.trendWindow)
	validDataPoints := make([]OwnershipDataPoint, 0)
	for _, point := range trendData.DataPoints {
		if point.Timestamp.After(cutoffTime) {
			validDataPoints = append(validDataPoints, point)
		}
	}
	trendData.DataPoints = validDataPoints
	
	// Save updated trend data
	trendDataBytes, _ := json.Marshal(trendData)
	if err := ta.redisClient.Set(ctx, cacheKey, trendDataBytes, ta.cacheTTL).Err(); err != nil {
		return fmt.Errorf("failed to save trend data: %w", err)
	}
	
	return nil
}

// CalculateTrends calculates ownership trends for all players in a contest
func (ta *TrendAnalyzer) CalculateTrends(contestID string, timeRange time.Duration) ([]models.OwnershipTrend, error) {
	ctx := context.Background()
	
	// Get all trend keys for this contest
	pattern := fmt.Sprintf("trend:%s:*", contestID)
	keys, err := ta.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get trend keys: %w", err)
	}
	
	trends := make([]models.OwnershipTrend, 0, len(keys))
	
	for _, key := range keys {
		// Extract player ID from key
		var playerID uint
		if _, err := fmt.Sscanf(key, "trend:%s:%d", contestID, &playerID); err != nil {
			continue
		}
		
		// Calculate trend for this player
		trend, err := ta.calculatePlayerTrend(ctx, contestID, playerID, timeRange)
		if err != nil {
			ta.logger.WithError(err).WithField("player_id", playerID).Warn("Failed to calculate player trend")
			continue
		}
		
		if trend != nil {
			trends = append(trends, *trend)
		}
	}
	
	// Sort trends by ownership change magnitude
	sort.Slice(trends, func(i, j int) bool {
		return math.Abs(trends[i].OwnershipChange) > math.Abs(trends[j].OwnershipChange)
	})
	
	return trends, nil
}

// calculatePlayerTrend calculates trend metrics for a specific player
func (ta *TrendAnalyzer) calculatePlayerTrend(ctx context.Context, contestID string, playerID uint, timeRange time.Duration) (*models.OwnershipTrend, error) {
	cacheKey := fmt.Sprintf("trend:%s:%d", contestID, playerID)
	
	// Get trend data
	trendDataBytes, err := ta.redisClient.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get trend data: %w", err)
	}
	
	var trendData TrendData
	if err := json.Unmarshal([]byte(trendDataBytes), &trendData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trend data: %w", err)
	}
	
	// Need at least minimum data points
	if len(trendData.DataPoints) < ta.minDataPoints {
		return nil, nil
	}
	
	// Filter data points within time range
	cutoffTime := time.Now().Add(-timeRange)
	validPoints := make([]OwnershipDataPoint, 0)
	for _, point := range trendData.DataPoints {
		if point.Timestamp.After(cutoffTime) {
			validPoints = append(validPoints, point)
		}
	}
	
	if len(validPoints) < 2 {
		return nil, nil
	}
	
	// Sort by timestamp
	sort.Slice(validPoints, func(i, j int) bool {
		return validPoints[i].Timestamp.Before(validPoints[j].Timestamp)
	})
	
	// Calculate trend metrics
	currentOwnership := validPoints[len(validPoints)-1].Ownership
	prevOwnership := validPoints[0].Ownership
	ownershipChange := currentOwnership - prevOwnership
	
	// Calculate velocity (change per hour)
	timeDiff := validPoints[len(validPoints)-1].Timestamp.Sub(validPoints[0].Timestamp)
	velocity := 0.0
	if timeDiff.Hours() > 0 {
		velocity = ownershipChange / timeDiff.Hours()
	}
	
	// Determine trend direction
	trendDirection := "stable"
	if math.Abs(ownershipChange) > 0.5 { // 0.5% threshold
		if ownershipChange > 0 {
			trendDirection = "up"
		} else {
			trendDirection = "down"
		}
	}
	
	trend := &models.OwnershipTrend{
		PlayerID:         playerID,
		ContestID:        contestID,
		CurrentOwnership: currentOwnership,
		PrevOwnership:    prevOwnership,
		OwnershipChange:  ownershipChange,
		Velocity:         velocity,
		TrendDirection:   trendDirection,
	}
	
	return trend, nil
}

// CalculateVelocity calculates ownership change velocity between two snapshots
func (ta *TrendAnalyzer) CalculateVelocity(prev, current map[uint]float64, timeDiff time.Duration) map[uint]float64 {
	velocity := make(map[uint]float64)
	
	if timeDiff.Hours() == 0 {
		return velocity
	}
	
	for playerID, currentOwnership := range current {
		if prevOwnership, exists := prev[playerID]; exists {
			change := currentOwnership - prevOwnership
			velocity[playerID] = change / timeDiff.Hours()
		}
	}
	
	return velocity
}

// PredictOwnership predicts ownership at contest lock based on current trends
func (ta *TrendAnalyzer) PredictOwnership(contestID string, playerID uint, lockTime time.Time) (*TrendMetrics, error) {
	ctx := context.Background()
	
	// Check cache first
	predictionKey := fmt.Sprintf("prediction:%s:%d", contestID, playerID)
	cachedPrediction, err := ta.redisClient.Get(ctx, predictionKey).Result()
	if err == nil {
		var metrics TrendMetrics
		if err := json.Unmarshal([]byte(cachedPrediction), &metrics); err == nil {
			// Return cached prediction if recent enough
			if time.Since(metrics.LastCalculated) < 2*time.Minute {
				return &metrics, nil
			}
		}
	}
	
	// Get trend data
	trendKey := fmt.Sprintf("trend:%s:%d", contestID, playerID)
	trendDataBytes, err := ta.redisClient.Get(ctx, trendKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get trend data: %w", err)
	}
	
	var trendData TrendData
	if err := json.Unmarshal([]byte(trendDataBytes), &trendData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trend data: %w", err)
	}
	
	if len(trendData.DataPoints) < ta.minDataPoints {
		return nil, fmt.Errorf("insufficient data points for prediction")
	}
	
	// Calculate prediction using multiple models
	prediction := ta.calculatePrediction(trendData.DataPoints, lockTime)
	
	// Cache the prediction
	predictionBytes, _ := json.Marshal(prediction)
	ta.redisClient.Set(ctx, predictionKey, predictionBytes, 2*time.Minute)
	
	return prediction, nil
}

// calculatePrediction calculates ownership prediction using trend analysis
func (ta *TrendAnalyzer) calculatePrediction(dataPoints []OwnershipDataPoint, lockTime time.Time) *TrendMetrics {
	if len(dataPoints) < 2 {
		return nil
	}
	
	// Sort by timestamp
	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Timestamp.Before(dataPoints[j].Timestamp)
	})
	
	currentPoint := dataPoints[len(dataPoints)-1]
	prevPoint := dataPoints[len(dataPoints)-2]
	
	// Calculate basic metrics
	ownershipChange := currentPoint.Ownership - prevPoint.Ownership
	timeDiff := currentPoint.Timestamp.Sub(prevPoint.Timestamp)
	velocity := 0.0
	if timeDiff.Hours() > 0 {
		velocity = ownershipChange / timeDiff.Hours()
	}
	
	// Calculate acceleration if we have enough points
	acceleration := 0.0
	if len(dataPoints) >= 3 {
		prevPrevPoint := dataPoints[len(dataPoints)-3]
		prevVelocity := (prevPoint.Ownership - prevPrevPoint.Ownership) / 
			prevPoint.Timestamp.Sub(prevPrevPoint.Timestamp).Hours()
		acceleration = velocity - prevVelocity
	}
	
	// Linear prediction
	timeToLock := lockTime.Sub(currentPoint.Timestamp).Hours()
	linearPrediction := currentPoint.Ownership + (velocity * timeToLock)
	
	// Acceleration-adjusted prediction
	accelerationAdjustment := 0.5 * acceleration * timeToLock * timeToLock
	adjustedPrediction := linearPrediction + accelerationAdjustment
	
	// Apply bounds (ownership can't be negative or > 100%)
	if adjustedPrediction < 0 {
		adjustedPrediction = 0
	}
	if adjustedPrediction > 100 {
		adjustedPrediction = 100
	}
	
	// Calculate confidence based on trend consistency
	confidence := ta.calculateConfidence(dataPoints)
	
	// Determine trend direction and strength
	trendDirection := "stable"
	trendStrength := math.Abs(velocity) / 10.0 // Normalize velocity to 0-1 scale
	if trendStrength > 1.0 {
		trendStrength = 1.0
	}
	
	if math.Abs(ownershipChange) > 0.5 {
		if ownershipChange > 0 {
			trendDirection = "up"
		} else {
			trendDirection = "down"
		}
	}
	
	return &TrendMetrics{
		CurrentOwnership: currentPoint.Ownership,
		PrevOwnership:    prevPoint.Ownership,
		OwnershipChange:  ownershipChange,
		Velocity:         velocity,
		Acceleration:     acceleration,
		TrendDirection:   trendDirection,
		TrendStrength:    trendStrength,
		Prediction:       adjustedPrediction,
		Confidence:       confidence,
		LastCalculated:   time.Now(),
	}
}

// calculateConfidence calculates prediction confidence based on trend consistency
func (ta *TrendAnalyzer) calculateConfidence(dataPoints []OwnershipDataPoint) float64 {
	if len(dataPoints) < 3 {
		return 0.5 // Low confidence with insufficient data
	}
	
	// Calculate variance in velocity to determine consistency
	velocities := make([]float64, 0, len(dataPoints)-1)
	for i := 1; i < len(dataPoints); i++ {
		timeDiff := dataPoints[i].Timestamp.Sub(dataPoints[i-1].Timestamp).Hours()
		if timeDiff > 0 {
			velocity := (dataPoints[i].Ownership - dataPoints[i-1].Ownership) / timeDiff
			velocities = append(velocities, velocity)
		}
	}
	
	if len(velocities) < 2 {
		return 0.5
	}
	
	// Calculate mean and variance
	mean := 0.0
	for _, v := range velocities {
		mean += v
	}
	mean /= float64(len(velocities))
	
	variance := 0.0
	for _, v := range velocities {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(velocities))
	
	// Convert variance to confidence (lower variance = higher confidence)
	// Use exponential decay to map variance to 0-1 scale
	confidence := math.Exp(-variance / 10.0)
	
	// Factor in data quantity (more data points = higher confidence)
	dataFactor := float64(len(dataPoints)) / 10.0
	if dataFactor > 1.0 {
		dataFactor = 1.0
	}
	
	confidence *= dataFactor
	
	if confidence > 1.0 {
		confidence = 1.0
	}
	
	return confidence
}

// GetTrendSummary returns a summary of all trends for a contest
func (ta *TrendAnalyzer) GetTrendSummary(contestID string) (map[string]interface{}, error) {
	trends, err := ta.CalculateTrends(contestID, 2*time.Hour)
	if err != nil {
		return nil, err
	}
	
	summary := map[string]interface{}{
		"total_players": len(trends),
		"trends_by_direction": map[string]int{
			"up":     0,
			"down":   0,
			"stable": 0,
		},
		"top_gainers":  make([]models.OwnershipTrend, 0),
		"top_fallers":  make([]models.OwnershipTrend, 0),
		"most_volatile": make([]models.OwnershipTrend, 0),
	}
	
	gainers := make([]models.OwnershipTrend, 0)
	fallers := make([]models.OwnershipTrend, 0)
	
	directionCounts := summary["trends_by_direction"].(map[string]int)
	
	for _, trend := range trends {
		directionCounts[trend.TrendDirection]++
		
		if trend.OwnershipChange > 1.0 { // > 1% increase
			gainers = append(gainers, trend)
		} else if trend.OwnershipChange < -1.0 { // > 1% decrease
			fallers = append(fallers, trend)
		}
	}
	
	// Sort and limit top gainers/fallers
	sort.Slice(gainers, func(i, j int) bool {
		return gainers[i].OwnershipChange > gainers[j].OwnershipChange
	})
	
	sort.Slice(fallers, func(i, j int) bool {
		return fallers[i].OwnershipChange < fallers[j].OwnershipChange
	})
	
	// Take top 5 of each
	if len(gainers) > 5 {
		gainers = gainers[:5]
	}
	if len(fallers) > 5 {
		fallers = fallers[:5]
	}
	
	summary["top_gainers"] = gainers
	summary["top_fallers"] = fallers
	
	return summary, nil
}