package optimizer

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"

	"github.com/stitts-dev/dfs-sim/services/sports-data-service/pkg/providers"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// PositionOptimizer handles position-specific optimization strategies for golf tournaments
type PositionOptimizer struct {
	baseOptimizer      BaseOptimizerInterface
	cutProbEngine      *CutProbabilityEngine
	dataGolfClient     *providers.DataGolfClient
	strategyConfigs    map[types.TournamentPositionStrategy]*StrategyConfig
}

// BaseOptimizerInterface defines the interface for the base optimization engine
type BaseOptimizerInterface interface {
	Optimize(ctx context.Context, request *types.OptimizationRequest) (*types.OptimizationResult, error)
}

// StrategyConfig represents configuration for a specific tournament strategy
type StrategyConfig struct {
	Strategy              types.TournamentPositionStrategy
	MinCutProbability     float64
	WeightCutProbability  float64
	PreferHighVariance    bool
	MaxExposure           float64
	MinPlayerDifference   int
	RiskTolerance         float64
	PositionWeights       map[string]float64 // win, top5, top10, top25 weights
	VarianceMultiplier    float64
	ConsistencyWeight     float64
}

// PositionObjective represents the optimization objective for position-based strategies
type PositionObjective struct {
	WinWeight    float64
	Top5Weight   float64
	Top10Weight  float64
	Top25Weight  float64
	CutWeight    float64
	PointsWeight float64
}

// PlayerStrategyScore represents a player's score for a specific strategy
type PlayerStrategyScore struct {
	PlayerID          string
	BaseScore         float64
	CutAdjustedScore  float64
	PositionScore     float64
	VarianceScore     float64
	FinalScore        float64
	StrategyFit       float64
}

// NewPositionOptimizer creates a new position strategy optimizer
func NewPositionOptimizer(baseOptimizer BaseOptimizerInterface, cutProbEngine *CutProbabilityEngine, dataGolfClient *providers.DataGolfClient) *PositionOptimizer {
	optimizer := &PositionOptimizer{
		baseOptimizer:   baseOptimizer,
		cutProbEngine:   cutProbEngine,
		dataGolfClient:  dataGolfClient,
		strategyConfigs: make(map[types.TournamentPositionStrategy]*StrategyConfig),
	}

	// Initialize strategy configurations
	optimizer.initializeStrategyConfigs()
	
	return optimizer
}

// OptimizeForStrategy optimizes lineups for a specific tournament strategy
func (p *PositionOptimizer) OptimizeForStrategy(ctx context.Context, request *types.GolfOptimizationRequest) (*types.OptimizationResult, error) {
	// Get strategy configuration
	config, exists := p.strategyConfigs[request.TournamentStrategy]
	if !exists {
		return nil, fmt.Errorf("unsupported strategy: %s", request.TournamentStrategy)
	}

	log.Printf("Optimizing for strategy: %s", request.TournamentStrategy)

	// Apply strategy-specific modifications to the request
	modifiedRequest := p.applyStrategyModifications(request, config)

	// Calculate strategy-specific player scores
	if err := p.calculateStrategyScores(ctx, modifiedRequest, config); err != nil {
		return nil, fmt.Errorf("failed to calculate strategy scores: %w", err)
	}

	// Run the base optimization with modified parameters
	result, err := p.baseOptimizer.Optimize(ctx, &modifiedRequest.OptimizationRequest)
	if err != nil {
		return nil, fmt.Errorf("base optimization failed: %w", err)
	}

	// Post-process results with strategy-specific analytics
	p.addStrategyAnalytics(result, request.TournamentStrategy, config)

	log.Printf("Successfully optimized %d lineups for %s strategy", len(result.Lineups), request.TournamentStrategy)
	
	return result, nil
}

// applyStrategyModifications modifies the optimization request based on strategy
func (p *PositionOptimizer) applyStrategyModifications(request *types.GolfOptimizationRequest, config *StrategyConfig) *types.GolfOptimizationRequest {
	modified := *request // Copy the request

	switch config.Strategy {
	case types.WinStrategy:
		// Maximize ceiling with high variance
		modified.Settings.MinDifferentPlayers = 4  // More overlap allowed
		modified.PreferHighVariance = true
		log.Printf("Applied Win Strategy modifications: min_different_players=%d, prefer_high_variance=%t", modified.Settings.MinDifferentPlayers, modified.PreferHighVariance)

	case types.TopTenStrategy:
		// Balance consistency with upside
		modified.Settings.MinDifferentPlayers = 3
		modified.WeightCutProbability = 0.7
		log.Printf("Applied Top10 Strategy modifications: min_different_players=%d, cut_weight=%.2f", modified.Settings.MinDifferentPlayers, modified.WeightCutProbability)

	case types.TopFiveStrategy:
		// High upside with moderate consistency
		modified.Settings.MinDifferentPlayers = 3
		modified.WeightCutProbability = 0.6

	case types.TopTwentyFive:
		// Balanced approach with slight consistency bias
		modified.Settings.MinDifferentPlayers = 2
		modified.WeightCutProbability = 0.8

	case types.CutStrategy:
		// Prioritize cut makers for cash games
		modified.MinCutProbability = 0.65
		modified.WeightCutProbability = 0.9
		log.Printf("Applied Cut Strategy modifications: min_cut=%.2f, cut_weight=%.2f", modified.MinCutProbability, modified.WeightCutProbability)

	case types.BalancedStrategy:
		// Default balanced approach
		modified.Settings.MinDifferentPlayers = 2
		modified.WeightCutProbability = 0.7
	}

	// Apply risk tolerance adjustments
	if request.RiskTolerance > 0 {
		riskMultiplier := 1.0 + (request.RiskTolerance - 0.5) * 0.5 // Scale risk tolerance
		if modified.PreferHighVariance {
			// Increase preference for high variance players
			modified.Settings.RandomnessLevel = riskMultiplier * 0.1
		}
	}

	return &modified
}

// calculateStrategyScores calculates strategy-specific scores for all players
func (p *PositionOptimizer) calculateStrategyScores(ctx context.Context, request *types.GolfOptimizationRequest, config *StrategyConfig) error {
	log.Printf("Calculating DataGolf strategy scores for %s with cut optimization: %v", config.Strategy, request.CutOptimization)
	
	// Skip DataGolf integration if client is not available
	if p.dataGolfClient == nil {
		log.Printf("DataGolf client not available, using default scoring")
		return nil
	}

	// Get DataGolf predictions for the tournament
	predictions, err := p.dataGolfClient.GetPreTournamentPredictions(request.TournamentID)
	if err != nil {
		log.Printf("Failed to get DataGolf predictions: %v, falling back to default scoring", err)
		return nil // Don't fail the optimization, just use default scoring
	}

	// Get course analytics for course-specific adjustments
	var courseAnalytics *providers.CourseAnalytics
	if request.CourseID != "" {
		if analytics, err := p.dataGolfClient.GetCourseAnalytics(request.CourseID); err == nil {
			courseAnalytics = analytics
		}
	}

	// Get weather impact data if available
	var weatherImpact *providers.WeatherImpactAnalysis
	if request.WeatherConsideration {
		if weather, err := p.dataGolfClient.GetWeatherImpactData(request.TournamentID); err == nil {
			weatherImpact = weather
		}
	}

	log.Printf("DataGolf data retrieved: predictions=%d, course_analytics=%v, weather=%v", 
		len(predictions.Predictions), courseAnalytics != nil, weatherImpact != nil)

	// Calculate strategy-specific scores for each player
	for i, player := range request.PlayerPool {
		// Find DataGolf prediction for this player
		var playerPrediction *providers.PlayerPrediction
		for _, pred := range predictions.Predictions {
			if fmt.Sprintf("%d", pred.PlayerID) == player.ExternalID {
				playerPrediction = &pred
				break
			}
		}

		if playerPrediction == nil {
			continue // Skip players without DataGolf data
		}

		// Calculate base strategy score using DataGolf probabilities
		strategyScore := p.calculatePlayerStrategyScore(playerPrediction, courseAnalytics, weatherImpact, config)
		
		// Update player's projected score with strategy-adjusted value
		originalScore := player.ProjectedPoints
		adjustedScore := originalScore * strategyScore
		
		// Apply cut probability adjustment if enabled
		if request.CutOptimization {
			cutProbAdj := math.Pow(playerPrediction.MakeCutProbability, config.WeightCutProbability)
			adjustedScore *= cutProbAdj
		}

		// Update the player in the pool
		request.PlayerPool[i].ProjectedPoints = adjustedScore
		
		log.Printf("Player %s: original=%.2f, strategy_score=%.3f, cut_prob=%.3f, final=%.2f", 
			player.Name, originalScore, strategyScore, playerPrediction.MakeCutProbability, adjustedScore)
	}

	log.Printf("Strategy score calculation completed for %d players", len(request.PlayerPool))
	return nil
}

// calculatePlayerStrategyScore calculates strategy-specific score for a player
func (p *PositionOptimizer) calculatePlayerStrategyScore(
	prediction *providers.PlayerPrediction,
	courseAnalytics *providers.CourseAnalytics,
	weatherImpact *providers.WeatherImpactAnalysis,
	config *StrategyConfig,
) float64 {
	// Get position weights for this strategy
	weights := config.PositionWeights
	
	// Calculate weighted position score
	positionScore := 0.0
	positionScore += weights["win"] * prediction.WinProbability
	positionScore += weights["top5"] * prediction.Top5Probability
	positionScore += weights["top10"] * prediction.Top10Probability
	positionScore += weights["top25"] * prediction.Top20Probability
	positionScore += weights["cut"] * prediction.MakeCutProbability

	// Apply comprehensive course fit adjustment if available
	courseFitMultiplier := 1.0
	if courseAnalytics != nil {
		courseFitMultiplier = p.calculateCourseFitMultiplier(prediction, courseAnalytics, config)
	} else {
		// Use basic course fit from prediction
		courseFitMultiplier = 1.0 + (prediction.CourseFit * 0.1)
	}

	// Apply comprehensive weather adjustment if available
	weatherMultiplier := 1.0
	if weatherImpact != nil {
		weatherMultiplier = p.calculateComprehensiveWeatherMultiplier(prediction, weatherImpact, config)
	}

	// Apply variance preference using VolatilityRating from DataGolf
	varianceMultiplier := 1.0
	if config.PreferHighVariance && prediction.VolatilityRating > 0.6 {
		varianceMultiplier = 1.0 + config.VarianceMultiplier * 0.1
	} else if !config.PreferHighVariance && prediction.VolatilityRating < 0.4 {
		varianceMultiplier = 1.0 + config.ConsistencyWeight * 0.1
	}

	// Apply strokes gained optimization if available
	sgMultiplier := 1.0
	// TODO: Implement strokes gained optimization when SG fields are available in DataGolf response
	// if prediction has SG data {
	//   sgMultiplier = p.calculateStrokesGainedMultiplier(prediction, courseAnalytics, config)
	// }
	
	// Combine all factors including strokes gained optimization
	finalScore := positionScore * courseFitMultiplier * weatherMultiplier * varianceMultiplier * sgMultiplier
	
	// Ensure score is within reasonable bounds
	return math.Max(0.1, math.Min(2.0, finalScore))
}

// calculateCourseFitMultiplier calculates comprehensive course fit multiplier using DataGolf course analytics
func (p *PositionOptimizer) calculateCourseFitMultiplier(
	prediction *providers.PlayerPrediction,
	courseAnalytics *providers.CourseAnalytics,
	config *StrategyConfig,
) float64 {
	baseFit := 1.0
	
	// Course difficulty adjustment - favor players who perform well on difficult courses
	if courseAnalytics.DifficultyRating > 7.5 && prediction.VolatilityRating < 0.4 {
		baseFit *= 1.08 // Boost consistent players on difficult courses
	} else if courseAnalytics.DifficultyRating < 6.0 && prediction.VolatilityRating > 0.6 {
		baseFit *= 1.05 // Slight boost for volatile players on easier courses
	}
	
	// Course length adjustment - favor distance players on long courses
	if courseAnalytics.Length > 7200 {
		baseFit *= 1.03 // Slight boost on long courses (detailed metrics not available)
	} else if courseAnalytics.Length < 6800 {
		baseFit *= 1.02 // Slight boost on shorter courses
	}
	
	// Use available course analytics for general difficulty
	if courseAnalytics.DifficultyRating > 8.0 {
		baseFit *= 1.04 // Boost on difficult courses
	}
	
	// Course-specific skill premium analysis
	skillFitMultiplier := p.calculateSkillFitForCourse(prediction, courseAnalytics)
	baseFit *= skillFitMultiplier
	
	// Use course fit score from prediction
	baseFit *= (1.0 + prediction.CourseFit * 0.1)
	
	// Course strategy fit - adjust based on whether course rewards the chosen strategy
	strategyFitMultiplier := p.calculateCourseStrategyFit(courseAnalytics, config)
	baseFit *= strategyFitMultiplier
	
	// Bound the multiplier to reasonable limits
	return math.Max(0.85, math.Min(1.25, baseFit))
}

// calculateSkillFitForCourse calculates how well a player's skills match course requirements
func (p *PositionOptimizer) calculateSkillFitForCourse(
	prediction *providers.PlayerPrediction,
	courseAnalytics *providers.CourseAnalytics,
) float64 {
	skillFit := 1.0
	
	// Use available skill premiums from course analytics
	if courseAnalytics.SkillPremiums.DrivingDistance > 0.7 {
		skillFit *= 1.02 // Reward when driving distance is important
	}
	
	if courseAnalytics.SkillPremiums.DrivingAccuracy > 0.7 {
		skillFit *= 1.02 // Reward when driving accuracy is important
	}
	
	if courseAnalytics.SkillPremiums.ApproachPrecision > 0.7 {
		skillFit *= 1.03 // Reward when approach precision is key
	}
	
	if courseAnalytics.SkillPremiums.ShortGameSkill > 0.7 {
		skillFit *= 1.02 // Reward when short game is important
	}
	
	if courseAnalytics.SkillPremiums.PuttingConsistency > 0.7 {
		skillFit *= 1.03 // Reward when putting is crucial
	}
	
	return skillFit
}

// calculateHistoricalCoursePerformance analyzes player's historical performance at the course
func (p *PositionOptimizer) calculateHistoricalCoursePerformance(
	courseHistory *providers.PlayerCourseHistory,
	config *StrategyConfig,
) float64 {
	historyFit := 1.0
	
	// Use course fit score as base multiplier
	historyFit *= (1.0 + courseHistory.CourseFitScore * 0.1)
	
	// Boost based on total appearances (experience factor)
	if courseHistory.TotalAppearances >= 5 {
		historyFit *= 1.02 // Experience bonus
	}
	
	// Boost based on best finish
	if courseHistory.BestFinish <= 5 {
		historyFit *= 1.05 // Has contended at this course
	} else if courseHistory.BestFinish <= 10 {
		historyFit *= 1.03 // Has finished well at this course
	}
	
	// Analyze recent form entries if available
	if len(courseHistory.RecentForm) > 0 {
		recentFinish := courseHistory.RecentForm[0].Position
		if recentFinish <= 10 {
			historyFit *= 1.04 // Recent good finish
		}
	}
	
	// Sample size confidence adjustment
	if courseHistory.TotalAppearances < 3 {
		historyFit = 1.0 + (historyFit - 1.0) * 0.5 // Reduce impact with small sample
	}
	
	return historyFit
}

// calculateCourseStrategyFit determines how well the course rewards the chosen strategy
func (p *PositionOptimizer) calculateCourseStrategyFit(
	courseAnalytics *providers.CourseAnalytics,
	config *StrategyConfig,
) float64 {
	strategyFit := 1.0
	
	switch config.Strategy {
	case types.WinStrategy:
		// Win strategy on difficult courses
		if courseAnalytics.DifficultyRating > 7.0 {
			strategyFit *= 1.03 // Difficult courses create more separation
		}
		
	case types.CutStrategy:
		// Cut strategy benefits from course predictability
		if courseAnalytics.DifficultyRating < 6.0 {
			strategyFit *= 1.04 // Easier courses = more predictable cuts
		}
		
	case types.TopTenStrategy, types.TopTwentyFive:
		// Mid-tier strategies favor moderate difficulty
		if courseAnalytics.DifficultyRating > 6.0 && courseAnalytics.DifficultyRating < 8.0 {
			strategyFit *= 1.02 // Moderate difficulty ideal
		}
	}
	
	return strategyFit
}

// calculateComprehensiveWeatherMultiplier provides detailed weather impact analysis using DataGolf data
func (p *PositionOptimizer) calculateComprehensiveWeatherMultiplier(
	prediction *providers.PlayerPrediction,
	weatherImpact *providers.WeatherImpactAnalysis,
	config *StrategyConfig,
) float64 {
	baseMultiplier := 1.0
	
	// Wind impact analysis
	windMultiplier := p.calculateWindImpact(prediction, weatherImpact)
	baseMultiplier *= windMultiplier
	
	// Rain/moisture impact analysis
	rainMultiplier := p.calculateRainImpact(prediction, weatherImpact)
	baseMultiplier *= rainMultiplier
	
	// Temperature impact analysis
	tempMultiplier := p.calculateTemperatureImpact(prediction, weatherImpact)
	baseMultiplier *= tempMultiplier
	
	// Strategy-specific weather adjustments
	strategyMultiplier := p.calculateWeatherStrategyImpact(weatherImpact, config)
	baseMultiplier *= strategyMultiplier
	
	// Ensure reasonable bounds
	return math.Max(0.80, math.Min(1.25, baseMultiplier))
}

// calculateWindImpact analyzes wind conditions and player adaptability
func (p *PositionOptimizer) calculateWindImpact(
	prediction *providers.PlayerPrediction,
	weatherImpact *providers.WeatherImpactAnalysis,
) float64 {
	windMultiplier := 1.0
	
	// Use overall impact score and weather advantage
	if weatherImpact.OverallImpact > 0.5 {
		// Significant weather impact expected
		if prediction.WeatherAdvantage > 0.1 {
			windMultiplier *= 1.05 // Boost players with weather advantage
		} else if prediction.WeatherAdvantage < -0.1 {
			windMultiplier *= 0.95 // Penalty for weather-disadvantaged players
		}
	}
	
	// High volatility players may benefit from chaotic conditions
	if weatherImpact.OverallImpact > 0.3 && prediction.VolatilityRating > 0.6 {
		windMultiplier *= 1.02 // Slight boost for volatile players in tough conditions
	}
	
	return windMultiplier
}

// calculateRainImpact analyzes weather conditions impact
func (p *PositionOptimizer) calculateRainImpact(
	prediction *providers.PlayerPrediction,
	weatherImpact *providers.WeatherImpactAnalysis,
) float64 {
	rainMultiplier := 1.0
	
	// Use weather advantage for general conditions
	if weatherImpact.OverallImpact > 0.3 {
		// Moderate to high weather impact
		if prediction.WeatherAdvantage > 0.05 {
			rainMultiplier *= 1.03 // Boost for weather-adapted players
		} else if prediction.WeatherAdvantage < -0.05 {
			rainMultiplier *= 0.97 // Penalty for weather-sensitive players
		}
	}
	
	return rainMultiplier
}

// calculateTemperatureImpact analyzes general weather impact
func (p *PositionOptimizer) calculateTemperatureImpact(
	prediction *providers.PlayerPrediction,
	weatherImpact *providers.WeatherImpactAnalysis,
) float64 {
	tempMultiplier := 1.0
	
	// Use overall weather impact and player weather advantage
	if weatherImpact.OverallImpact > 0.2 {
		if prediction.WeatherAdvantage > 0.0 {
			tempMultiplier *= 1.02 // Small boost for weather-adapted players
		} else if prediction.WeatherAdvantage < 0.0 {
			tempMultiplier *= 0.98 // Small penalty for weather-sensitive players
		}
	}
	
	return tempMultiplier
}

// calculateWeatherStrategyImpact adjusts weather impact based on tournament strategy
func (p *PositionOptimizer) calculateWeatherStrategyImpact(
	weatherImpact *providers.WeatherImpactAnalysis,
	config *StrategyConfig,
) float64 {
	strategyMultiplier := 1.0
	
	// Use overall impact score for strategy adjustments
	if weatherImpact.OverallImpact > 0.5 {
		// High weather impact conditions
		switch config.Strategy {
		case types.CutStrategy:
			// Difficult weather makes cuts more valuable
			strategyMultiplier *= 1.04
			
		case types.WinStrategy:
			// Difficult weather can create more separation
			strategyMultiplier *= 1.03
			
		case types.TopTenStrategy, types.TopFiveStrategy:
			// Mid-tier strategies may be safer
			strategyMultiplier *= 1.02
		}
	} else if weatherImpact.OverallImpact < 0.2 {
		// Mild conditions
		switch config.Strategy {
		case types.WinStrategy:
			// Good scoring conditions may compress the field
			strategyMultiplier *= 0.99
			
		case types.CutStrategy:
			// Higher cut rates in good conditions
			strategyMultiplier *= 0.98
		}
	}
	
	return strategyMultiplier
}

// calculateStrokesGainedMultiplier provides comprehensive strokes gained optimization analysis
func (p *PositionOptimizer) calculateStrokesGainedMultiplier(
	prediction *providers.PlayerPrediction,
	courseAnalytics *providers.CourseAnalytics,
	config *StrategyConfig,
) float64 {
	sgMultiplier := 1.0
	
	// Base strokes gained analysis
	sgScores := map[string]float64{
		"off_tee":        prediction.SGOffTee,
		"approach":       prediction.SGApproach,
		"around_green":   prediction.SGAroundGreen,
		"putting":        prediction.SGPutting,
	}
	
	// Calculate weighted SG score based on course analytics
	if courseAnalytics != nil {
		weightedSG := p.calculateWeightedStrokesGained(sgScores, courseAnalytics)
		sgMultiplier *= (1.0 + weightedSG * 0.1) // Convert SG to multiplier
	} else {
		// Fallback to simple SG average if no course analytics
		avgSG := (prediction.SGOffTee + prediction.SGApproach + prediction.SGAroundGreen + prediction.SGPutting) / 4.0
		sgMultiplier *= (1.0 + avgSG * 0.08)
	}
	
	// Strategy-specific SG weighting
	strategySGMultiplier := p.calculateStrategySGWeighting(sgScores, config)
	sgMultiplier *= strategySGMultiplier
	
	// SG consistency analysis
	sgConsistencyMultiplier := p.calculateSGConsistency(prediction, config)
	sgMultiplier *= sgConsistencyMultiplier
	
	// SG correlation analysis for tournament strategy
	sgCorrelationMultiplier := p.calculateSGCorrelationFactor(sgScores, config)
	sgMultiplier *= sgCorrelationMultiplier
	
	// Bound the SG multiplier
	return math.Max(0.85, math.Min(1.20, sgMultiplier))
}

// calculateWeightedStrokesGained weights SG categories based on course importance
func (p *PositionOptimizer) calculateWeightedStrokesGained(
	sgScores map[string]float64,
	courseAnalytics *providers.CourseAnalytics,
) float64 {
	// Course-specific SG weights based on what the course rewards
	weights := map[string]float64{
		"off_tee":      0.25, // Default weights
		"approach":     0.30,
		"around_green": 0.20,
		"putting":      0.25,
	}
	
	// Adjust weights based on course analytics
	if courseAnalytics.DrivingPremium > 0.7 {
		weights["off_tee"] += 0.10
		weights["approach"] -= 0.05
		weights["around_green"] -= 0.03
		weights["putting"] -= 0.02
	}
	
	if courseAnalytics.ApproachPremium > 0.7 {
		weights["approach"] += 0.12
		weights["off_tee"] -= 0.04
		weights["around_green"] -= 0.04
		weights["putting"] -= 0.04
	}
	
	if courseAnalytics.ShortGamePremium > 0.7 {
		weights["around_green"] += 0.10
		weights["putting"] += 0.05
		weights["off_tee"] -= 0.08
		weights["approach"] -= 0.07
	}
	
	if courseAnalytics.PuttingPremium > 0.7 {
		weights["putting"] += 0.12
		weights["around_green"] += 0.03
		weights["approach"] -= 0.08
		weights["off_tee"] -= 0.07
	}
	
	// Calculate weighted SG score
	weightedSG := 0.0
	weightedSG += sgScores["off_tee"] * weights["off_tee"]
	weightedSG += sgScores["approach"] * weights["approach"]
	weightedSG += sgScores["around_green"] * weights["around_green"]
	weightedSG += sgScores["putting"] * weights["putting"]
	
	return weightedSG
}

// calculateStrategySGWeighting adjusts SG importance based on tournament strategy
func (p *PositionOptimizer) calculateStrategySGWeighting(
	sgScores map[string]float64,
	config *StrategyConfig,
) float64 {
	strategyMultiplier := 1.0
	
	switch config.Strategy {
	case types.WinStrategy:
		// Win strategy emphasizes approach and putting for birdie opportunities
		if sgScores["approach"] > 0.3 {
			strategyMultiplier *= 1.08
		}
		if sgScores["putting"] > 0.3 {
			strategyMultiplier *= 1.06
		}
		
	case types.CutStrategy:
		// Cut strategy emphasizes consistency in all categories
		sgVariance := p.calculateSGVariance(sgScores)
		if sgVariance < 0.2 { // Low variance = consistent across categories
			strategyMultiplier *= 1.10
		}
		// Favor off-the-tee consistency for cut making
		if sgScores["off_tee"] > 0.0 {
			strategyMultiplier *= 1.05
		}
		
	case types.TopFiveStrategy, types.TopTenStrategy:
		// Placing strategies favor balanced excellence
		excellentCategories := 0
		for _, sg := range sgScores {
			if sg > 0.25 {
				excellentCategories++
			}
		}
		if excellentCategories >= 2 {
			strategyMultiplier *= 1.07
		}
		
	case types.BalancedStrategy:
		// Balanced strategy rewards well-rounded SG performance
		minSG := math.Min(math.Min(sgScores["off_tee"], sgScores["approach"]),
			math.Min(sgScores["around_green"], sgScores["putting"]))
		if minSG > -0.1 { // No major weaknesses
			strategyMultiplier *= 1.06
		}
	}
	
	return strategyMultiplier
}

// calculateSGConsistency analyzes strokes gained consistency across categories
func (p *PositionOptimizer) calculateSGConsistency(
	prediction *providers.PlayerPrediction,
	config *StrategyConfig,
) float64 {
	consistencyMultiplier := 1.0
	
	// Calculate SG category variance
	sgScoresMap := map[string]float64{
		"off_tee":      prediction.SGOffTee,
		"approach":     prediction.SGApproach,
		"around_green": prediction.SGAroundGreen,
		"putting":      prediction.SGPutting,
	}
	
	variance := p.calculateSGVariance(sgScoresMap)
	
	// Consistency preferences based on strategy
	if config.Strategy == types.CutStrategy || config.Strategy == types.BalancedStrategy {
		// These strategies prefer consistency
		if variance < 0.15 {
			consistencyMultiplier *= 1.05 // Reward low variance
		} else if variance > 0.4 {
			consistencyMultiplier *= 0.98 // Slight penalty for high variance
		}
	} else if config.Strategy == types.WinStrategy {
		// Win strategy can benefit from specialization
		maxSG := math.Max(math.Max(prediction.SGOffTee, prediction.SGApproach),
			math.Max(prediction.SGAroundGreen, prediction.SGPutting))
		if maxSG > 0.5 {
			consistencyMultiplier *= 1.04 // Reward elite performance in any category
		}
	}
	
	return consistencyMultiplier
}

// calculateSGCorrelationFactor analyzes how SG performance correlates with strategy success
func (p *PositionOptimizer) calculateSGCorrelationFactor(
	sgScores map[string]float64,
	config *StrategyConfig,
) float64 {
	correlationMultiplier := 1.0
	
	// Total strokes gained - baseline player quality
	totalSG := sgScores["off_tee"] + sgScores["approach"] + sgScores["around_green"] + sgScores["putting"]
	
	// High-level players get strategy-specific bonuses
	if totalSG > 1.0 { // Elite overall SG performance
		switch config.Strategy {
		case types.WinStrategy:
			correlationMultiplier *= 1.08 // Elite players have better win chances
		case types.TopFiveStrategy:
			correlationMultiplier *= 1.06 // Elite players finish in top 5 more often
		case types.TopTenStrategy:
			correlationMultiplier *= 1.04 // Moderate bonus for top 10
		}
	}
	
	// Poor SG performance penalties
	if totalSG < -0.5 {
		correlationMultiplier *= 0.95 // General penalty for poor SG
		if config.Strategy == types.WinStrategy {
			correlationMultiplier *= 0.92 // Larger penalty for win strategy
		}
	}
	
	// Category-specific correlations
	if config.Strategy == types.WinStrategy {
		// Putting and approach matter most for wins
		if sgScores["putting"] > 0.4 && sgScores["approach"] > 0.3 {
			correlationMultiplier *= 1.06
		}
	}
	
	return correlationMultiplier
}

// calculateSGVariance calculates variance across SG categories
func (p *PositionOptimizer) calculateSGVariance(sgScores map[string]float64) float64 {
	values := []float64{
		sgScores["off_tee"],
		sgScores["approach"],
		sgScores["around_green"],
		sgScores["putting"],
	}
	
	// Calculate mean
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	
	// Calculate variance
	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values))
	
	return math.Sqrt(variance) // Return standard deviation
}

// addStrategyAnalytics adds strategy-specific analytics to the optimization result
func (p *PositionOptimizer) addStrategyAnalytics(result *types.OptimizationResult, strategy types.TournamentPositionStrategy, config *StrategyConfig) {
	// Store analytics in the metadata (simplified implementation)
	// In a full implementation, this would extend the metadata structure
	
	avgCutProb := p.calculateAvgCutProbability(result)
	diversityScore := p.calculateLineupDiversity(result)
	
	log.Printf("Added strategy analytics for %s: avg_cut_prob=%.3f, diversity=%.3f", 
		strategy, avgCutProb, diversityScore)
}

// calculateAvgCutProbability calculates average cut probability across lineups
func (p *PositionOptimizer) calculateAvgCutProbability(result *types.OptimizationResult) float64 {
	if len(result.Lineups) == 0 {
		return 0.0
	}

	// Simplified implementation - would calculate actual cut probabilities in production
	return 0.75 // Placeholder average cut probability
}

// calculateAvgExpectedFinish calculates average expected finish position
func (p *PositionOptimizer) calculateAvgExpectedFinish(result *types.OptimizationResult) float64 {
	// Simplified implementation
	return 45.0 // Average expected finish
}

// calculateRiskRewardRatio calculates the risk/reward ratio for the lineups
func (p *PositionOptimizer) calculateRiskRewardRatio(result *types.OptimizationResult) float64 {
	// Simplified implementation
	return 2.5 // Placeholder risk/reward ratio
}

// calculateLineupDiversity calculates diversity score across lineups
func (p *PositionOptimizer) calculateLineupDiversity(result *types.OptimizationResult) float64 {
	// Simplified implementation
	return 0.75 // Placeholder diversity score
}

// calculateStrategyFitScore calculates how well lineups fit the chosen strategy
func (p *PositionOptimizer) calculateStrategyFitScore(result *types.OptimizationResult, config *StrategyConfig) float64 {
	// Simplified implementation
	avgCutProb := p.calculateAvgCutProbability(result)
	
	fitScore := 0.0
	switch config.Strategy {
	case types.WinStrategy:
		fitScore = 0.7 + avgCutProb * 0.3
	case types.CutStrategy:
		fitScore = avgCutProb * 0.9 + 0.1
	default:
		fitScore = avgCutProb * 0.6 + 0.4
	}

	return math.Max(0.0, math.Min(1.0, fitScore))
}

// initializeStrategyConfigs initializes the configuration for each strategy
func (p *PositionOptimizer) initializeStrategyConfigs() {
	p.strategyConfigs[types.WinStrategy] = &StrategyConfig{
		Strategy:              types.WinStrategy,
		MinCutProbability:     0.60,
		WeightCutProbability:  0.4,
		PreferHighVariance:    true,
		MaxExposure:           0.40,
		MinPlayerDifference:   4,
		RiskTolerance:         0.8,
		VarianceMultiplier:    1.5,
		ConsistencyWeight:     0.2,
		PositionWeights: map[string]float64{
			"win": 1.0, "top5": 0.3, "top10": 0.1, "top25": 0.05, "cut": 0.4,
		},
	}

	p.strategyConfigs[types.TopFiveStrategy] = &StrategyConfig{
		Strategy:              types.TopFiveStrategy,
		MinCutProbability:     0.65,
		WeightCutProbability:  0.6,
		PreferHighVariance:    true,
		MaxExposure:           0.35,
		MinPlayerDifference:   3,
		RiskTolerance:         0.7,
		VarianceMultiplier:    1.2,
		ConsistencyWeight:     0.4,
		PositionWeights: map[string]float64{
			"win": 0.7, "top5": 1.0, "top10": 0.4, "top25": 0.1, "cut": 0.6,
		},
	}

	p.strategyConfigs[types.TopTenStrategy] = &StrategyConfig{
		Strategy:              types.TopTenStrategy,
		MinCutProbability:     0.70,
		WeightCutProbability:  0.7,
		PreferHighVariance:    false,
		MaxExposure:           0.30,
		MinPlayerDifference:   3,
		RiskTolerance:         0.6,
		VarianceMultiplier:    1.0,
		ConsistencyWeight:     0.6,
		PositionWeights: map[string]float64{
			"win": 0.4, "top5": 0.8, "top10": 1.0, "top25": 0.3, "cut": 0.7,
		},
	}

	p.strategyConfigs[types.TopTwentyFive] = &StrategyConfig{
		Strategy:              types.TopTwentyFive,
		MinCutProbability:     0.75,
		WeightCutProbability:  0.8,
		PreferHighVariance:    false,
		MaxExposure:           0.25,
		MinPlayerDifference:   2,
		RiskTolerance:         0.5,
		VarianceMultiplier:    0.8,
		ConsistencyWeight:     0.7,
		PositionWeights: map[string]float64{
			"win": 0.2, "top5": 0.4, "top10": 0.7, "top25": 1.0, "cut": 0.8,
		},
	}

	p.strategyConfigs[types.CutStrategy] = &StrategyConfig{
		Strategy:              types.CutStrategy,
		MinCutProbability:     0.80,
		WeightCutProbability:  0.9,
		PreferHighVariance:    false,
		MaxExposure:           0.20,
		MinPlayerDifference:   1,
		RiskTolerance:         0.3,
		VarianceMultiplier:    0.6,
		ConsistencyWeight:     0.9,
		PositionWeights: map[string]float64{
			"win": 0.1, "top5": 0.2, "top10": 0.3, "top25": 0.5, "cut": 1.0,
		},
	}

	p.strategyConfigs[types.BalancedStrategy] = &StrategyConfig{
		Strategy:              types.BalancedStrategy,
		MinCutProbability:     0.70,
		WeightCutProbability:  0.7,
		PreferHighVariance:    false,
		MaxExposure:           0.25,
		MinPlayerDifference:   2,
		RiskTolerance:         0.5,
		VarianceMultiplier:    1.0,
		ConsistencyWeight:     0.5,
		PositionWeights: map[string]float64{
			"win": 0.5, "top5": 0.6, "top10": 0.7, "top25": 0.6, "cut": 0.7,
		},
	}
}

// GetStrategyConfig returns the configuration for a given strategy
func (p *PositionOptimizer) GetStrategyConfig(strategy types.TournamentPositionStrategy) (*StrategyConfig, error) {
	config, exists := p.strategyConfigs[strategy]
	if !exists {
		return nil, fmt.Errorf("strategy %s not found", strategy)
	}
	return config, nil
}

// GetAvailableStrategies returns all available tournament strategies
func (p *PositionOptimizer) GetAvailableStrategies() []types.TournamentPositionStrategy {
	strategies := make([]types.TournamentPositionStrategy, 0, len(p.strategyConfigs))
	for strategy := range p.strategyConfigs {
		strategies = append(strategies, strategy)
	}
	
	// Sort for consistent ordering
	sort.Slice(strategies, func(i, j int) bool {
		return string(strategies[i]) < string(strategies[j])
	})
	
	return strategies
}