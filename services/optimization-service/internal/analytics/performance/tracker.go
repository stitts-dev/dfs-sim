package performance

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/shared/pkg/logger"
)

// Tracker handles performance aggregation and attribution analysis
type Tracker struct {
	logger      *logrus.Logger
	workerCount int
}

// TrackerConfig defines configuration for performance tracking
type TrackerConfig struct {
	UserID        int       `json:"user_id"`
	TimeFrame     string    `json:"time_frame"` // "7d", "30d", "90d", "1y"
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	Sports        []string  `json:"sports"`
	ContestTypes  []string  `json:"contest_types"`
	WorkerCount   int       `json:"worker_count"`
	EnableAttribution bool  `json:"enable_attribution"`
}

// PerformanceReport contains comprehensive performance analysis
type PerformanceReport struct {
	UserID           int                        `json:"user_id"`
	TimeFrame        string                     `json:"time_frame"`
	Period           DateRange                  `json:"period"`
	Summary          PerformanceSummary         `json:"summary"`
	Attribution      AttributionAnalysis        `json:"attribution"`
	SportBreakdown   map[string]SportMetrics    `json:"sport_breakdown"`
	ContestBreakdown map[string]ContestMetrics  `json:"contest_breakdown"`
	TrendAnalysis    TrendAnalysis              `json:"trend_analysis"`
	Recommendations  []PerformanceRecommendation `json:"recommendations"`
	GeneratedAt      time.Time                  `json:"generated_at"`
}

// PerformanceSummary contains overall performance metrics
type PerformanceSummary struct {
	TotalLineups     int     `json:"total_lineups"`
	TotalSpent       float64 `json:"total_spent"`
	TotalWon         float64 `json:"total_won"`
	NetProfit        float64 `json:"net_profit"`
	ROI              float64 `json:"roi"`
	WinRate          float64 `json:"win_rate"`
	AvgScore         float64 `json:"avg_score"`
	AvgEntryFee      float64 `json:"avg_entry_fee"`
	SharpeRatio      float64 `json:"sharpe_ratio"`
	MaxDrawdown      float64 `json:"max_drawdown"`
	ConsistencyScore float64 `json:"consistency_score"`
}

// AttributionAnalysis breaks down performance by various factors
type AttributionAnalysis struct {
	SportContribution    map[string]float64 `json:"sport_contribution"`
	ContestContribution  map[string]float64 `json:"contest_contribution"`
	StackingContribution map[string]float64 `json:"stacking_contribution"`
	PlayerContribution   []PlayerAttribution `json:"player_contribution"`
	TimeContribution     []TimeAttribution   `json:"time_contribution"`
}

// PlayerAttribution shows individual player impact
type PlayerAttribution struct {
	PlayerID        string  `json:"player_id"`
	PlayerName      string  `json:"player_name"`
	TimesUsed       int     `json:"times_used"`
	TotalROI        float64 `json:"total_roi"`
	AvgROI          float64 `json:"avg_roi"`
	WinRate         float64 `json:"win_rate"`
	ImpactScore     float64 `json:"impact_score"`
}

// TimeAttribution shows performance over time periods
type TimeAttribution struct {
	Date     time.Time `json:"date"`
	Period   string    `json:"period"` // "daily", "weekly", "monthly"
	ROI      float64   `json:"roi"`
	Lineups  int       `json:"lineups"`
	WinRate  float64   `json:"win_rate"`
}

// SportMetrics contains sport-specific performance
type SportMetrics struct {
	Sport        string  `json:"sport"`
	Lineups      int     `json:"lineups"`
	ROI          float64 `json:"roi"`
	WinRate      float64 `json:"win_rate"`
	AvgScore     float64 `json:"avg_score"`
	Contribution float64 `json:"contribution"`
}

// ContestMetrics contains contest-specific performance
type ContestMetrics struct {
	ContestType  string  `json:"contest_type"`
	Lineups      int     `json:"lineups"`
	ROI          float64 `json:"roi"`
	WinRate      float64 `json:"win_rate"`
	AvgEntryFee  float64 `json:"avg_entry_fee"`
	Contribution float64 `json:"contribution"`
}

// TrendAnalysis shows performance trends over time
type TrendAnalysis struct {
	ROITrend         float64                `json:"roi_trend"`
	WinRateTrend     float64                `json:"win_rate_trend"`
	ScoreTrend       float64                `json:"score_trend"`
	SeasonalPatterns map[string]float64     `json:"seasonal_patterns"`
	RecentVsHistoric PerformanceComparison  `json:"recent_vs_historic"`
}

// PerformanceComparison compares two time periods
type PerformanceComparison struct {
	RecentPeriod    PerformanceSummary `json:"recent_period"`
	HistoricPeriod  PerformanceSummary `json:"historic_period"`
	Improvement     float64            `json:"improvement"`
	SignificantChange bool             `json:"significant_change"`
}

// PerformanceRecommendation provides actionable insights
type PerformanceRecommendation struct {
	Type        string  `json:"type"`
	Priority    string  `json:"priority"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Impact      float64 `json:"impact"`
	ActionItems []string `json:"action_items"`
}

// DateRange represents a time period
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// LineupPerformanceData represents historical lineup performance
type LineupPerformanceData struct {
	LineupID        string    `json:"lineup_id"`
	UserID          int       `json:"user_id"`
	Sport           string    `json:"sport"`
	ContestType     string    `json:"contest_type"`
	EntryFee        float64   `json:"entry_fee"`
	Winnings        float64   `json:"winnings"`
	ActualScore     float64   `json:"actual_score"`
	ProjectedScore  float64   `json:"projected_score"`
	Rank            int       `json:"rank"`
	TotalEntries    int       `json:"total_entries"`
	Players         []PlayerData `json:"players"`
	Date            time.Time `json:"date"`
	HasStacking     bool      `json:"has_stacking"`
	StackingType    string    `json:"stacking_type"`
}

// PlayerData represents player information in a lineup
type PlayerData struct {
	PlayerID        string  `json:"player_id"`
	Name            string  `json:"name"`
	Position        string  `json:"position"`
	Team            string  `json:"team"`
	Salary          int     `json:"salary"`
	ProjectedPoints float64 `json:"projected_points"`
	ActualPoints    float64 `json:"actual_points"`
	Ownership       float64 `json:"ownership"`
}

// MetricResult represents a calculated metric from worker
type MetricResult struct {
	Type    string      `json:"type"`
	Value   float64     `json:"value"`
	Data    interface{} `json:"data"`
	Error   error       `json:"error"`
}

// NewTracker creates a new performance tracker
func NewTracker() *Tracker {
	return &Tracker{
		logger:      logger.GetLogger(),
		workerCount: runtime.NumCPU(),
	}
}

// AggregatePerformance performs comprehensive performance analysis
func (t *Tracker) AggregatePerformance(ctx context.Context, userID int, config TrackerConfig) (*PerformanceReport, error) {
	startTime := time.Now()
	
	t.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"time_frame": config.TimeFrame,
		"start_date": config.StartDate,
		"end_date":   config.EndDate,
	}).Info("Starting performance aggregation")

	// Fetch user data (this would typically query database)
	lineups, contests, results, err := t.fetchUserData(userID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user data: %w", err)
	}

	if len(lineups) == 0 {
		return nil, fmt.Errorf("no lineup data found for user %d", userID)
	}

	// Initialize report
	report := &PerformanceReport{
		UserID:           userID,
		TimeFrame:        config.TimeFrame,
		Period:           DateRange{Start: config.StartDate, End: config.EndDate},
		SportBreakdown:   make(map[string]SportMetrics),
		ContestBreakdown: make(map[string]ContestMetrics),
		GeneratedAt:      time.Now(),
	}

	// Calculate metrics using worker pool
	metricsChan := make(chan MetricResult, 20)
	var wg sync.WaitGroup

	// Determine worker count
	workerCount := config.WorkerCount
	if workerCount <= 0 {
		workerCount = t.workerCount
	}

	// Launch workers for different metrics
	wg.Add(6)
	go t.calculateBasicMetrics(lineups, metricsChan, &wg)
	go t.calculateSharpeRatio(results, metricsChan, &wg)
	go t.calculateDrawdown(results, metricsChan, &wg)
	go t.calculateWinRate(contests, results, metricsChan, &wg)
	go t.calculateSportBreakdown(lineups, metricsChan, &wg)
	go t.calculateContestBreakdown(lineups, metricsChan, &wg)

	// Attribution analysis (if enabled)
	if config.EnableAttribution {
		wg.Add(1)
		go t.performAttributionAnalysis(lineups, metricsChan, &wg)
	}

	// Collect results
	go func() {
		wg.Wait()
		close(metricsChan)
	}()

	// Process metric results
	summary := PerformanceSummary{}
	attribution := AttributionAnalysis{}
	var errors []error

	for result := range metricsChan {
		if result.Error != nil {
			errors = append(errors, result.Error)
			continue
		}

		switch result.Type {
		case "basic_metrics":
			if data, ok := result.Data.(PerformanceSummary); ok {
				summary = data
			}
		case "sharpe_ratio":
			summary.SharpeRatio = result.Value
		case "max_drawdown":
			summary.MaxDrawdown = result.Value
		case "win_rate":
			summary.WinRate = result.Value
		case "sport_breakdown":
			if data, ok := result.Data.(map[string]SportMetrics); ok {
				report.SportBreakdown = data
			}
		case "contest_breakdown":
			if data, ok := result.Data.(map[string]ContestMetrics); ok {
				report.ContestBreakdown = data
			}
		case "attribution":
			if data, ok := result.Data.(AttributionAnalysis); ok {
				attribution = data
			}
		}
	}

	// Handle any errors
	if len(errors) > 0 {
		t.logger.WithField("errors", errors).Warn("Some metrics calculations failed")
	}

	// Finalize report
	report.Summary = summary
	report.Attribution = attribution

	// Calculate trend analysis
	report.TrendAnalysis = t.calculateTrendAnalysis(lineups)

	// Generate recommendations
	report.Recommendations = t.generateRecommendations(report)

	processingTime := time.Since(startTime)
	t.logger.WithFields(logrus.Fields{
		"user_id":         userID,
		"total_lineups":   len(lineups),
		"processing_time": processingTime,
	}).Info("Performance aggregation completed")

	return report, nil
}

// Worker functions for parallel metric calculation

func (t *Tracker) calculateBasicMetrics(lineups []LineupPerformanceData, results chan<- MetricResult, wg *sync.WaitGroup) {
	defer wg.Done()

	totalSpent := 0.0
	totalWon := 0.0
	totalScore := 0.0
	scores := make([]float64, len(lineups))
	entryFees := make([]float64, len(lineups))

	for i, lineup := range lineups {
		totalSpent += lineup.EntryFee
		totalWon += lineup.Winnings
		totalScore += lineup.ActualScore
		scores[i] = lineup.ActualScore
		entryFees[i] = lineup.EntryFee
	}

	summary := PerformanceSummary{
		TotalLineups: len(lineups),
		TotalSpent:   totalSpent,
		TotalWon:     totalWon,
		NetProfit:    totalWon - totalSpent,
		AvgScore:     totalScore / float64(len(lineups)),
		AvgEntryFee:  t.calculateMean(entryFees),
	}

	if totalSpent > 0 {
		summary.ROI = (totalWon - totalSpent) / totalSpent
	}

	summary.ConsistencyScore = t.calculateConsistency(scores)

	results <- MetricResult{
		Type: "basic_metrics",
		Data: summary,
	}
}

func (t *Tracker) calculateSharpeRatio(results []LineupPerformanceData, metricsChan chan<- MetricResult, wg *sync.WaitGroup) {
	defer wg.Done()

	rois := make([]float64, len(results))
	for i, result := range results {
		if result.EntryFee > 0 {
			roi := (result.Winnings - result.EntryFee) / result.EntryFee
			rois[i] = roi
		}
	}

	avgROI := t.calculateMean(rois)
	stdROI := math.Sqrt(t.calculateVariance(rois))

	sharpeRatio := 0.0
	if stdROI > 0 {
		sharpeRatio = avgROI / stdROI
	}

	metricsChan <- MetricResult{
		Type:  "sharpe_ratio",
		Value: sharpeRatio,
	}
}

func (t *Tracker) calculateDrawdown(results []LineupPerformanceData, metricsChan chan<- MetricResult, wg *sync.WaitGroup) {
	defer wg.Done()

	runningBalance := 0.0
	peak := 0.0
	maxDrawdown := 0.0

	for _, result := range results {
		runningBalance += result.Winnings - result.EntryFee
		
		if runningBalance > peak {
			peak = runningBalance
		}
		
		drawdown := (peak - runningBalance) / math.Max(peak, 1.0)
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	metricsChan <- MetricResult{
		Type:  "max_drawdown",
		Value: maxDrawdown,
	}
}

func (t *Tracker) calculateWinRate(contests []LineupPerformanceData, results []LineupPerformanceData, metricsChan chan<- MetricResult, wg *sync.WaitGroup) {
	defer wg.Done()

	wins := 0
	for _, result := range results {
		if result.Winnings > result.EntryFee {
			wins++
		}
	}

	winRate := 0.0
	if len(results) > 0 {
		winRate = float64(wins) / float64(len(results))
	}

	metricsChan <- MetricResult{
		Type:  "win_rate",
		Value: winRate,
	}
}

func (t *Tracker) calculateSportBreakdown(lineups []LineupPerformanceData, metricsChan chan<- MetricResult, wg *sync.WaitGroup) {
	defer wg.Done()

	sportData := make(map[string][]LineupPerformanceData)
	for _, lineup := range lineups {
		sportData[lineup.Sport] = append(sportData[lineup.Sport], lineup)
	}

	breakdown := make(map[string]SportMetrics)
	totalROI := 0.0

	for sport, sportLineups := range sportData {
		spent := 0.0
		won := 0.0
		totalScore := 0.0
		wins := 0

		for _, lineup := range sportLineups {
			spent += lineup.EntryFee
			won += lineup.Winnings
			totalScore += lineup.ActualScore
			if lineup.Winnings > lineup.EntryFee {
				wins++
			}
		}

		roi := 0.0
		if spent > 0 {
			roi = (won - spent) / spent
			totalROI += roi
		}

		breakdown[sport] = SportMetrics{
			Sport:    sport,
			Lineups:  len(sportLineups),
			ROI:      roi,
			WinRate:  float64(wins) / float64(len(sportLineups)),
			AvgScore: totalScore / float64(len(sportLineups)),
		}
	}

	// Calculate contribution percentages
	for sport, metrics := range breakdown {
		if totalROI != 0 {
			metrics.Contribution = metrics.ROI / totalROI
			breakdown[sport] = metrics
		}
	}

	metricsChan <- MetricResult{
		Type: "sport_breakdown",
		Data: breakdown,
	}
}

func (t *Tracker) calculateContestBreakdown(lineups []LineupPerformanceData, metricsChan chan<- MetricResult, wg *sync.WaitGroup) {
	defer wg.Done()

	contestData := make(map[string][]LineupPerformanceData)
	for _, lineup := range lineups {
		contestData[lineup.ContestType] = append(contestData[lineup.ContestType], lineup)
	}

	breakdown := make(map[string]ContestMetrics)

	for contestType, contestLineups := range contestData {
		spent := 0.0
		won := 0.0
		wins := 0

		for _, lineup := range contestLineups {
			spent += lineup.EntryFee
			won += lineup.Winnings
			if lineup.Winnings > lineup.EntryFee {
				wins++
			}
		}

		roi := 0.0
		if spent > 0 {
			roi = (won - spent) / spent
		}

		breakdown[contestType] = ContestMetrics{
			ContestType: contestType,
			Lineups:     len(contestLineups),
			ROI:         roi,
			WinRate:     float64(wins) / float64(len(contestLineups)),
			AvgEntryFee: spent / float64(len(contestLineups)),
		}
	}

	metricsChan <- MetricResult{
		Type: "contest_breakdown",
		Data: breakdown,
	}
}

func (t *Tracker) performAttributionAnalysis(lineups []LineupPerformanceData, metricsChan chan<- MetricResult, wg *sync.WaitGroup) {
	defer wg.Done()

	attribution := AttributionAnalysis{
		SportContribution:   make(map[string]float64),
		ContestContribution: make(map[string]float64),
		StackingContribution: make(map[string]float64),
	}

	// Sport contribution
	sportROI := make(map[string]float64)
	sportCount := make(map[string]int)
	
	for _, lineup := range lineups {
		roi := 0.0
		if lineup.EntryFee > 0 {
			roi = (lineup.Winnings - lineup.EntryFee) / lineup.EntryFee
		}
		sportROI[lineup.Sport] += roi
		sportCount[lineup.Sport]++
	}

	totalSportROI := 0.0
	for sport, roi := range sportROI {
		avgROI := roi / float64(sportCount[sport])
		attribution.SportContribution[sport] = avgROI
		totalSportROI += avgROI
	}

	// Normalize sport contributions
	if totalSportROI != 0 {
		for sport := range attribution.SportContribution {
			attribution.SportContribution[sport] /= totalSportROI
		}
	}

	// Contest type contribution
	contestROI := make(map[string]float64)
	contestCount := make(map[string]int)
	
	for _, lineup := range lineups {
		roi := 0.0
		if lineup.EntryFee > 0 {
			roi = (lineup.Winnings - lineup.EntryFee) / lineup.EntryFee
		}
		contestROI[lineup.ContestType] += roi
		contestCount[lineup.ContestType]++
	}

	for contestType, roi := range contestROI {
		avgROI := roi / float64(contestCount[contestType])
		attribution.ContestContribution[contestType] = avgROI
	}

	// Stacking contribution
	stackedROI := 0.0
	unstackedROI := 0.0
	stackedCount := 0
	unstackedCount := 0

	for _, lineup := range lineups {
		roi := 0.0
		if lineup.EntryFee > 0 {
			roi = (lineup.Winnings - lineup.EntryFee) / lineup.EntryFee
		}

		if lineup.HasStacking {
			stackedROI += roi
			stackedCount++
		} else {
			unstackedROI += roi
			unstackedCount++
		}
	}

	if stackedCount > 0 {
		attribution.StackingContribution["stacked"] = stackedROI / float64(stackedCount)
	}
	if unstackedCount > 0 {
		attribution.StackingContribution["unstacked"] = unstackedROI / float64(unstackedCount)
	}

	// Player contribution analysis
	playerStats := make(map[string]*PlayerAttribution)
	
	for _, lineup := range lineups {
		roi := 0.0
		if lineup.EntryFee > 0 {
			roi = (lineup.Winnings - lineup.EntryFee) / lineup.EntryFee
		}
		
		isWin := lineup.Winnings > lineup.EntryFee

		for _, player := range lineup.Players {
			if _, exists := playerStats[player.PlayerID]; !exists {
				playerStats[player.PlayerID] = &PlayerAttribution{
					PlayerID:   player.PlayerID,
					PlayerName: player.Name,
				}
			}
			
			playerStats[player.PlayerID].TimesUsed++
			playerStats[player.PlayerID].TotalROI += roi
			if isWin {
				playerStats[player.PlayerID].WinRate += 1
			}
		}
	}

	// Finalize player stats
	for _, stats := range playerStats {
		if stats.TimesUsed > 0 {
			stats.AvgROI = stats.TotalROI / float64(stats.TimesUsed)
			stats.WinRate = stats.WinRate / float64(stats.TimesUsed)
			stats.ImpactScore = stats.AvgROI * math.Log(float64(stats.TimesUsed)+1)
		}
		attribution.PlayerContribution = append(attribution.PlayerContribution, *stats)
	}

	// Sort players by impact score
	sort.Slice(attribution.PlayerContribution, func(i, j int) bool {
		return attribution.PlayerContribution[i].ImpactScore > attribution.PlayerContribution[j].ImpactScore
	})

	metricsChan <- MetricResult{
		Type: "attribution",
		Data: attribution,
	}
}

// Helper functions

func (t *Tracker) fetchUserData(userID int, config TrackerConfig) ([]LineupPerformanceData, []LineupPerformanceData, []LineupPerformanceData, error) {
	// This would typically query the database
	// For now, return empty slices as placeholder
	return []LineupPerformanceData{}, []LineupPerformanceData{}, []LineupPerformanceData{}, nil
}

func (t *Tracker) calculateTrendAnalysis(lineups []LineupPerformanceData) TrendAnalysis {
	// Sort lineups by date
	sort.Slice(lineups, func(i, j int) bool {
		return lineups[i].Date.Before(lineups[j].Date)
	})

	// Calculate trends
	rois := make([]float64, len(lineups))
	winRates := make([]float64, len(lineups))
	scores := make([]float64, len(lineups))

	for i, lineup := range lineups {
		if lineup.EntryFee > 0 {
			rois[i] = (lineup.Winnings - lineup.EntryFee) / lineup.EntryFee
		}
		winRates[i] = 0
		if lineup.Winnings > lineup.EntryFee {
			winRates[i] = 1
		}
		scores[i] = lineup.ActualScore
	}

	return TrendAnalysis{
		ROITrend:     t.calculateTrend(rois),
		WinRateTrend: t.calculateTrend(winRates),
		ScoreTrend:   t.calculateTrend(scores),
		SeasonalPatterns: make(map[string]float64),
		RecentVsHistoric: PerformanceComparison{}, // Would calculate recent vs historic comparison
	}
}

func (t *Tracker) generateRecommendations(report *PerformanceReport) []PerformanceRecommendation {
	var recommendations []PerformanceRecommendation

	// ROI-based recommendations
	if report.Summary.ROI < 0 {
		recommendations = append(recommendations, PerformanceRecommendation{
			Type:        "performance",
			Priority:    "high",
			Title:       "Negative ROI Detected",
			Description: "Your overall ROI is negative. Consider reviewing your player selection and contest strategy.",
			Impact:      math.Abs(report.Summary.ROI),
			ActionItems: []string{
				"Review top performing lineups for patterns",
				"Analyze player selection criteria",
				"Consider lower entry fee contests initially",
			},
		})
	}

	// Sport diversification recommendations
	if len(report.SportBreakdown) == 1 {
		recommendations = append(recommendations, PerformanceRecommendation{
			Type:        "diversification",
			Priority:    "medium",
			Title:       "Sport Diversification",
			Description: "Consider diversifying across multiple sports to reduce risk.",
			Impact:      0.1,
			ActionItems: []string{
				"Try contests in 2-3 different sports",
				"Start with small entry fees in new sports",
				"Study optimal strategies for each sport",
			},
		})
	}

	// Consistency recommendations
	if report.Summary.ConsistencyScore < 0.5 {
		recommendations = append(recommendations, PerformanceRecommendation{
			Type:        "consistency",
			Priority:    "medium",
			Title:       "Improve Consistency",
			Description: "Your performance shows high variance. Focus on more consistent strategies.",
			Impact:      0.2,
			ActionItems: []string{
				"Use more chalk players for consistent floor",
				"Reduce tournament entry variance",
				"Implement bankroll management rules",
			},
		})
	}

	return recommendations
}

// Utility functions

func (t *Tracker) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (t *Tracker) calculateVariance(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	mean := t.calculateMean(values)
	sumSquares := 0.0
	for _, v := range values {
		sumSquares += (v - mean) * (v - mean)
	}
	return sumSquares / float64(len(values)-1)
}

func (t *Tracker) calculateConsistency(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	variance := t.calculateVariance(values)
	mean := t.calculateMean(values)
	if mean == 0 {
		return 0
	}
	return 1.0 / (1.0 + variance/math.Abs(mean))
}

func (t *Tracker) calculateTrend(values []float64) float64 {
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