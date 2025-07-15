package ml

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/shared/pkg/logger"
)

// FeatureExtractor handles feature extraction from user history and lineups
type FeatureExtractor struct {
	logger *logrus.Logger
}

// FeatureSet represents extracted features for ML models
type FeatureSet struct {
	UserID           int                    `json:"user_id"`
	Features         map[string]float64     `json:"features"`
	CategoricalFeats map[string]string      `json:"categorical_features"`
	TimeSeriesFeats  []TimeSeriesFeature    `json:"time_series_features"`
	ExtractedAt      time.Time              `json:"extracted_at"`
	FeatureVersion   string                 `json:"feature_version"`
}

// TimeSeriesFeature represents time-based features
type TimeSeriesFeature struct {
	Date     time.Time `json:"date"`
	Value    float64   `json:"value"`
	Feature  string    `json:"feature"`
	Window   string    `json:"window"` // "daily", "weekly", "monthly"
}

// UserLineupHistory represents historical lineup data for feature extraction
type UserLineupHistory struct {
	UserID          int                    `json:"user_id"`
	LineupID        string                 `json:"lineup_id"`
	Sport           string                 `json:"sport"`
	ContestType     string                 `json:"contest_type"`
	EntryFee        float64                `json:"entry_fee"`
	Winnings        float64                `json:"winnings"`
	ActualScore     float64                `json:"actual_score"`
	ProjectedScore  float64                `json:"projected_score"`
	Players         []PlayerFeature        `json:"players"`
	Date            time.Time              `json:"date"`
	RankPercentile  float64                `json:"rank_percentile"`
	ContestSize     int                    `json:"contest_size"`
}

// PlayerFeature represents player-specific features in a lineup
type PlayerFeature struct {
	PlayerID        string  `json:"player_id"`
	Name            string  `json:"name"`
	Position        string  `json:"position"`
	Team            string  `json:"team"`
	Salary          int     `json:"salary"`
	ProjectedPoints float64 `json:"projected_points"`
	ActualPoints    float64 `json:"actual_points"`
	Ownership       float64 `json:"ownership"`
	IsStacked       bool    `json:"is_stacked"`
}

// NewFeatureExtractor creates a new feature extractor instance
func NewFeatureExtractor() *FeatureExtractor {
	return &FeatureExtractor{
		logger: logger.GetLogger(),
	}
}

// ExtractUserFeatures extracts comprehensive features from user history
func (fe *FeatureExtractor) ExtractUserFeatures(ctx context.Context, userID int, history []UserLineupHistory, timeWindow int) (*FeatureSet, error) {
	fe.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"history_len": len(history),
		"time_window": timeWindow,
	}).Info("Starting feature extraction")

	if len(history) == 0 {
		return nil, fmt.Errorf("no history available for user %d", userID)
	}

	featureSet := &FeatureSet{
		UserID:           userID,
		Features:         make(map[string]float64),
		CategoricalFeats: make(map[string]string),
		TimeSeriesFeats:  []TimeSeriesFeature{},
		ExtractedAt:      time.Now(),
		FeatureVersion:   "1.0",
	}

	// Extract basic performance features
	fe.extractPerformanceFeatures(history, featureSet)

	// Extract behavioral features
	fe.extractBehavioralFeatures(history, featureSet)

	// Extract sport-specific features
	fe.extractSportFeatures(history, featureSet)

	// Extract player selection patterns
	fe.extractPlayerSelectionFeatures(history, featureSet)

	// Extract time-series features
	fe.extractTimeSeriesFeatures(history, featureSet, timeWindow)

	// Extract risk and diversification features
	fe.extractRiskFeatures(history, featureSet)

	// Extract contest type preferences
	fe.extractContestTypeFeatures(history, featureSet)

	// Generate encoded categorical features
	fe.encodeCategoricalFeatures(featureSet)

	fe.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"feature_count":  len(featureSet.Features),
		"categorical":    len(featureSet.CategoricalFeats),
		"time_series":    len(featureSet.TimeSeriesFeats),
	}).Info("Feature extraction completed")

	return featureSet, nil
}

// extractPerformanceFeatures extracts ROI, win rate, and scoring metrics
func (fe *FeatureExtractor) extractPerformanceFeatures(history []UserLineupHistory, features *FeatureSet) {
	if len(history) == 0 {
		return
	}

	totalSpent := 0.0
	totalWon := 0.0
	totalScore := 0.0
	wins := 0
	scores := make([]float64, len(history))
	rois := make([]float64, len(history))

	for i, lineup := range history {
		totalSpent += lineup.EntryFee
		totalWon += lineup.Winnings
		totalScore += lineup.ActualScore
		scores[i] = lineup.ActualScore
		
		if lineup.Winnings > lineup.EntryFee {
			wins++
		}
		
		if lineup.EntryFee > 0 {
			roi := (lineup.Winnings - lineup.EntryFee) / lineup.EntryFee
			rois[i] = roi
		}
	}

	// Basic performance metrics
	features.Features["total_lineups"] = float64(len(history))
	features.Features["avg_score"] = totalScore / float64(len(history))
	features.Features["win_rate"] = float64(wins) / float64(len(history))
	
	if totalSpent > 0 {
		features.Features["total_roi"] = (totalWon - totalSpent) / totalSpent
	}

	// Statistical measures
	features.Features["score_variance"] = calculateVariance(scores)
	features.Features["score_stddev"] = math.Sqrt(features.Features["score_variance"])
	features.Features["roi_variance"] = calculateVariance(rois)
	features.Features["roi_stddev"] = math.Sqrt(features.Features["roi_variance"])

	// Percentile-based features
	sortedScores := make([]float64, len(scores))
	copy(sortedScores, scores)
	sort.Float64s(sortedScores)
	
	features.Features["score_p25"] = percentile(sortedScores, 0.25)
	features.Features["score_p50"] = percentile(sortedScores, 0.50)
	features.Features["score_p75"] = percentile(sortedScores, 0.75)
	features.Features["score_p90"] = percentile(sortedScores, 0.90)

	// Consistency metrics
	features.Features["score_consistency"] = calculateConsistency(scores)
	features.Features["recent_performance_trend"] = calculateTrend(scores[int(math.Max(0, float64(len(scores)-10))):])
}

// extractBehavioralFeatures extracts user behavior patterns
func (fe *FeatureExtractor) extractBehavioralFeatures(history []UserLineupHistory, features *FeatureSet) {
	if len(history) == 0 {
		return
	}

	entryFees := make([]float64, len(history))
	projectedErrors := make([]float64, len(history))
	stackingUsage := 0
	
	dailyActivity := make(map[string]int)
	weeklyActivity := make(map[int]int)

	for i, lineup := range history {
		entryFees[i] = lineup.EntryFee
		
		if lineup.ProjectedScore > 0 {
			error := math.Abs(lineup.ActualScore - lineup.ProjectedScore) / lineup.ProjectedScore
			projectedErrors[i] = error
		}

		// Check for stacking usage
		hasStack := false
		teamCounts := make(map[string]int)
		for _, player := range lineup.Players {
			teamCounts[player.Team]++
			if player.IsStacked {
				hasStack = true
			}
		}
		if hasStack {
			stackingUsage++
		}

		// Activity patterns
		dateKey := lineup.Date.Format("2006-01-02")
		dailyActivity[dateKey]++
		weekday := int(lineup.Date.Weekday())
		weeklyActivity[weekday]++
	}

	// Entry fee patterns
	features.Features["avg_entry_fee"] = calculateMean(entryFees)
	features.Features["entry_fee_variance"] = calculateVariance(entryFees)
	features.Features["max_entry_fee"] = calculateMax(entryFees)
	features.Features["min_entry_fee"] = calculateMin(entryFees)

	// Projection accuracy
	features.Features["avg_projection_error"] = calculateMean(projectedErrors)
	features.Features["projection_error_variance"] = calculateVariance(projectedErrors)

	// Strategy usage
	features.Features["stacking_usage_rate"] = float64(stackingUsage) / float64(len(history))

	// Activity patterns
	maxDaily := 0
	for _, count := range dailyActivity {
		if count > maxDaily {
			maxDaily = count
		}
	}
	features.Features["max_daily_lineups"] = float64(maxDaily)
	features.Features["avg_lineups_per_active_day"] = float64(len(history)) / float64(len(dailyActivity))

	// Weekday preferences
	for day, count := range weeklyActivity {
		features.Features[fmt.Sprintf("weekday_%d_activity", day)] = float64(count) / float64(len(history))
	}
}

// extractSportFeatures extracts sport-specific patterns
func (fe *FeatureExtractor) extractSportFeatures(history []UserLineupHistory, features *FeatureSet) {
	sportCounts := make(map[string]int)
	sportPerformance := make(map[string][]float64)
	
	for _, lineup := range history {
		sportCounts[lineup.Sport]++
		
		roi := 0.0
		if lineup.EntryFee > 0 {
			roi = (lineup.Winnings - lineup.EntryFee) / lineup.EntryFee
		}
		sportPerformance[lineup.Sport] = append(sportPerformance[lineup.Sport], roi)
	}

	// Sport preferences
	totalLineups := float64(len(history))
	for sport, count := range sportCounts {
		features.Features[fmt.Sprintf("sport_%s_frequency", sport)] = float64(count) / totalLineups
		
		if rois, exists := sportPerformance[sport]; exists && len(rois) > 0 {
			features.Features[fmt.Sprintf("sport_%s_avg_roi", sport)] = calculateMean(rois)
			features.Features[fmt.Sprintf("sport_%s_roi_variance", sport)] = calculateVariance(rois)
		}
	}

	// Dominant sport
	maxCount := 0
	dominantSport := ""
	for sport, count := range sportCounts {
		if count > maxCount {
			maxCount = count
			dominantSport = sport
		}
	}
	features.CategoricalFeats["dominant_sport"] = dominantSport
	features.Features["sport_diversity"] = float64(len(sportCounts))
}

// extractPlayerSelectionFeatures extracts player selection patterns
func (fe *FeatureExtractor) extractPlayerSelectionFeatures(history []UserLineupHistory, features *FeatureSet) {
	playerUsage := make(map[string]int)
	salaryUsage := make([]float64, len(history))
	ownershipPatterns := make([]float64, 0)
	
	for i, lineup := range history {
		totalSalary := 0
		avgOwnership := 0.0
		
		for _, player := range lineup.Players {
			playerUsage[player.PlayerID]++
			totalSalary += player.Salary
			ownershipPatterns = append(ownershipPatterns, player.Ownership)
			avgOwnership += player.Ownership
		}
		
		salaryUsage[i] = float64(totalSalary)
		if len(lineup.Players) > 0 {
			features.Features[fmt.Sprintf("lineup_%d_avg_ownership", i)] = avgOwnership / float64(len(lineup.Players))
		}
	}

	// Salary usage patterns
	features.Features["avg_salary_usage"] = calculateMean(salaryUsage)
	features.Features["salary_usage_variance"] = calculateVariance(salaryUsage)

	// Player loyalty
	playerCounts := make([]int, 0, len(playerUsage))
	for _, count := range playerUsage {
		playerCounts = append(playerCounts, count)
	}
	
	if len(playerCounts) > 0 {
		floatCounts := make([]float64, len(playerCounts))
		for i, count := range playerCounts {
			floatCounts[i] = float64(count)
		}
		features.Features["avg_player_usage"] = calculateMean(floatCounts)
		features.Features["player_loyalty_variance"] = calculateVariance(floatCounts)
	}

	// Ownership patterns
	if len(ownershipPatterns) > 0 {
		features.Features["avg_ownership_selection"] = calculateMean(ownershipPatterns)
		features.Features["ownership_variance"] = calculateVariance(ownershipPatterns)
		
		// Contrarian vs chalk tendencies
		chalkyLineups := 0
		contrarianLineups := 0
		for _, ownership := range ownershipPatterns {
			if ownership > 0.15 { // High ownership threshold
				chalkyLineups++
			} else if ownership < 0.05 { // Low ownership threshold
				contrarianLineups++
			}
		}
		
		features.Features["chalky_tendency"] = float64(chalkyLineups) / float64(len(ownershipPatterns))
		features.Features["contrarian_tendency"] = float64(contrarianLineups) / float64(len(ownershipPatterns))
	}
}

// extractTimeSeriesFeatures extracts rolling window features
func (fe *FeatureExtractor) extractTimeSeriesFeatures(history []UserLineupHistory, features *FeatureSet, window int) {
	if len(history) < window {
		return
	}

	// Sort by date
	sortedHistory := make([]UserLineupHistory, len(history))
	copy(sortedHistory, history)
	sort.Slice(sortedHistory, func(i, j int) bool {
		return sortedHistory[i].Date.Before(sortedHistory[j].Date)
	})

	// Calculate rolling windows
	windows := []int{7, 14, 30}
	
	for _, w := range windows {
		if len(sortedHistory) >= w {
			recentHistory := sortedHistory[len(sortedHistory)-w:]
			
			// Rolling ROI
			totalSpent := 0.0
			totalWon := 0.0
			scores := make([]float64, len(recentHistory))
			
			for i, lineup := range recentHistory {
				totalSpent += lineup.EntryFee
				totalWon += lineup.Winnings
				scores[i] = lineup.ActualScore
				
				// Add time series feature
				features.TimeSeriesFeats = append(features.TimeSeriesFeats, TimeSeriesFeature{
					Date:    lineup.Date,
					Value:   lineup.ActualScore,
					Feature: "score",
					Window:  fmt.Sprintf("%d_day", w),
				})
			}
			
			if totalSpent > 0 {
				features.Features[fmt.Sprintf("rolling_%d_roi", w)] = (totalWon - totalSpent) / totalSpent
			}
			
			features.Features[fmt.Sprintf("rolling_%d_avg_score", w)] = calculateMean(scores)
			features.Features[fmt.Sprintf("rolling_%d_score_trend", w)] = calculateTrend(scores)
		}
	}
}

// extractRiskFeatures extracts risk and diversification metrics
func (fe *FeatureExtractor) extractRiskFeatures(history []UserLineupHistory, features *FeatureSet) {
	if len(history) == 0 {
		return
	}

	rois := make([]float64, len(history))
	
	for i, lineup := range history {
		if lineup.EntryFee > 0 {
			roi := (lineup.Winnings - lineup.EntryFee) / lineup.EntryFee
			rois[i] = roi
		}
	}

	// Risk metrics
	roiStdDev := math.Sqrt(calculateVariance(rois))
	avgROI := calculateMean(rois)
	
	features.Features["roi_volatility"] = roiStdDev
	
	// Sharpe ratio (using ROI standard deviation)
	if roiStdDev > 0 {
		features.Features["sharpe_ratio"] = avgROI / roiStdDev
	}

	// Maximum drawdown
	runningMax := rois[0]
	maxDrawdown := 0.0
	
	for _, roi := range rois {
		if roi > runningMax {
			runningMax = roi
		}
		drawdown := (runningMax - roi) / math.Max(runningMax, 1e-8)
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}
	
	features.Features["max_drawdown"] = maxDrawdown

	// Kelly criterion estimate
	winCount := 0
	winSum := 0.0
	lossSum := 0.0
	
	for _, roi := range rois {
		if roi > 0 {
			winCount++
			winSum += roi
		} else {
			lossSum += math.Abs(roi)
		}
	}
	
	if len(rois) > 0 && lossSum > 0 {
		winRate := float64(winCount) / float64(len(rois))
		avgWin := winSum / math.Max(float64(winCount), 1)
		avgLoss := lossSum / math.Max(float64(len(rois)-winCount), 1)
		
		if avgLoss > 0 {
			kellyFraction := winRate - ((1-winRate)*avgWin)/avgLoss
			features.Features["kelly_fraction"] = kellyFraction
		}
	}
}

// extractContestTypeFeatures extracts contest type preferences
func (fe *FeatureExtractor) extractContestTypeFeatures(history []UserLineupHistory, features *FeatureSet) {
	contestTypeCounts := make(map[string]int)
	contestTypePerformance := make(map[string][]float64)
	
	for _, lineup := range history {
		contestTypeCounts[lineup.ContestType]++
		
		roi := 0.0
		if lineup.EntryFee > 0 {
			roi = (lineup.Winnings - lineup.EntryFee) / lineup.EntryFee
		}
		contestTypePerformance[lineup.ContestType] = append(contestTypePerformance[lineup.ContestType], roi)
	}

	totalLineups := float64(len(history))
	for contestType, count := range contestTypeCounts {
		features.Features[fmt.Sprintf("contest_%s_frequency", contestType)] = float64(count) / totalLineups
		
		if rois, exists := contestTypePerformance[contestType]; exists && len(rois) > 0 {
			features.Features[fmt.Sprintf("contest_%s_avg_roi", contestType)] = calculateMean(rois)
		}
	}
}

// encodeCategoricalFeatures converts categorical features to numerical encodings
func (fe *FeatureExtractor) encodeCategoricalFeatures(features *FeatureSet) {
	// One-hot encode categorical features
	for key, value := range features.CategoricalFeats {
		// Simple label encoding for now - could be expanded to one-hot
		features.Features[fmt.Sprintf("%s_encoded", key)] = simpleHashEncoding(value)
	}
}

// Utility functions

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateVariance(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	mean := calculateMean(values)
	sumSquares := 0.0
	for _, v := range values {
		sumSquares += (v - mean) * (v - mean)
	}
	return sumSquares / float64(len(values)-1)
}

func calculateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func calculateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func percentile(sortedValues []float64, p float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	index := p * float64(len(sortedValues)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	
	if lower == upper {
		return sortedValues[lower]
	}
	
	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

func calculateConsistency(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	variance := calculateVariance(values)
	mean := calculateMean(values)
	if mean == 0 {
		return 0
	}
	return 1.0 / (1.0 + variance/math.Abs(mean))
}

func calculateTrend(values []float64) float64 {
	n := len(values)
	if n <= 1 {
		return 0
	}
	
	// Simple linear regression slope
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0
	
	for i, y := range values {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}
	
	denominator := float64(n)*sumXX - sumX*sumX
	if math.Abs(denominator) < 1e-8 {
		return 0
	}
	
	slope := (float64(n)*sumXY - sumX*sumY) / denominator
	return slope
}

func simpleHashEncoding(value string) float64 {
	// Simple hash-based encoding for categorical values
	hash := 0
	for _, char := range value {
		hash = (hash*31 + int(char)) % 1000000
	}
	return float64(hash) / 1000000.0
}