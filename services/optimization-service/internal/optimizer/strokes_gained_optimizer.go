package optimizer

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/stitts-dev/dfs-sim/shared/types"
	"github.com/sirupsen/logrus"
	"github.com/google/uuid"
)

// Placeholder types for future implementation
type WeatherImpactEngine struct {
	// TODO: Implement weather impact analysis engine
}

type AdvancedCorrelationEngine struct {
	// TODO: Implement advanced correlation analysis
}

// StrokesGainedOptimizer implements golf optimization using strokes gained analytics
type StrokesGainedOptimizer struct {
	// Core components
	dataProvider      EnhancedGolfDataProvider
	courseModelEngine *CourseModelEngine
	weatherEngine     *WeatherImpactEngine
	correlationEngine *AdvancedCorrelationEngine
	
	// Configuration
	optimizationWeights map[string]float64
	strategyProfiles    map[string]*StrategyProfile
	volatilityTargets   map[string]float64
	
	// Cache and logging
	cache  types.CacheProvider
	logger *logrus.Entry
}

// StrategyProfile defines golf-specific optimization strategies
type StrategyProfile struct {
	Name                string             `json:"name"`
	Description         string             `json:"description"`
	SGWeights           SGCategoryWeights  `json:"sg_weights"`
	VolatilityTolerance float64            `json:"volatility_tolerance"`
	CorrelationTargets  CorrelationTargets `json:"correlation_targets"`
	CutProbabilityMin   float64            `json:"cut_probability_min"`
	UpsideTargets       UpsideTargets      `json:"upside_targets"`
	RiskProfile         RiskProfile        `json:"risk_profile"`
	WeatherAdaptation   WeatherAdaptation  `json:"weather_adaptation"`
}

// SGCategoryWeights represents weights for different strokes gained categories
type SGCategoryWeights struct {
	OffTheTee      float64 `json:"off_the_tee"`       // e.g., 1.2 for tournaments favoring bombers
	Approach       float64 `json:"approach"`          // e.g., 1.0 baseline importance
	AroundTheGreen float64 `json:"around_the_green"`  // e.g., 0.8 for easier short game courses
	Putting        float64 `json:"putting"`           // e.g., 0.9 for consistent greens
}

// CorrelationTargets defines desired correlation characteristics
type CorrelationTargets struct {
	MaxPlayerCorrelation float64 `json:"max_player_correlation"`
	TeeTimeWeight        float64 `json:"tee_time_weight"`
	WeatherWeight        float64 `json:"weather_weight"`
	SkillWeight          float64 `json:"skill_weight"`
	OwnershipWeight      float64 `json:"ownership_weight"`
}

// UpsideTargets defines upside targeting for different strategies
type UpsideTargets struct {
	WinProbabilityMin    float64 `json:"win_probability_min"`
	Top5ProbabilityMin   float64 `json:"top5_probability_min"`
	Top10ProbabilityMin  float64 `json:"top10_probability_min"`
	VolatilityTarget     float64 `json:"volatility_target"`
	CeilingTarget        float64 `json:"ceiling_target"`
}

// RiskProfile defines risk management parameters
type RiskProfile struct {
	MaxExposurePerPlayer float64 `json:"max_exposure_per_player"`
	MaxStackSize         int     `json:"max_stack_size"`
	DiversificationMin   float64 `json:"diversification_min"`
	VarianceTarget       float64 `json:"variance_target"`
	SafetyFirst          bool    `json:"safety_first"`
}

// WeatherAdaptation defines how strategy adapts to weather
type WeatherAdaptation struct {
	WeatherSensitivity   float64            `json:"weather_sensitivity"`
	AdaptationFactors    map[string]float64 `json:"adaptation_factors"`
	ConditionPreferences map[string]float64 `json:"condition_preferences"`
}

// EnhancedGolfDataProvider interface for DataGolf integration
type EnhancedGolfDataProvider interface {
	GetStrokesGainedData(playerID string, tournamentID string) (*types.StrokesGainedMetrics, error)
	GetCourseAnalytics(courseID string) (*types.CourseAnalytics, error)
	GetPreTournamentPredictions(tournamentID string) (*types.TournamentPredictions, error)
	GetLiveTournamentData(tournamentID string) (*types.LiveTournamentData, error)
	GetPlayerCourseHistory(playerID, courseID string) (*types.PlayerCourseHistory, error)
	GetWeatherImpactData(tournamentID string) (*types.WeatherImpactAnalysis, error)
}

// SGOptimizationConfig represents configuration for strokes gained optimization
type SGOptimizationConfig struct {
	TournamentID        string            `json:"tournament_id"`
	Strategy            string            `json:"strategy"`
	NumLineups          int               `json:"num_lineups"`
	SalaryCap           int               `json:"salary_cap"`
	UseWeatherData      bool              `json:"use_weather_data"`
	UseCourseFit        bool              `json:"use_course_fit"`
	CustomWeights       SGCategoryWeights `json:"custom_weights,omitempty"`
	RiskTolerance       float64           `json:"risk_tolerance"`
	CorrelationStrategy string            `json:"correlation_strategy"`
}

// SGOptimizationResult represents the result of strokes gained optimization
type SGOptimizationResult struct {
	Lineups              []types.GeneratedLineup    `json:"lineups"`
	Strategy             *StrategyProfile           `json:"strategy"`
	CourseAnalysis       *types.CourseAnalytics     `json:"course_analysis"`
	WeatherImpact        *types.WeatherImpactAnalysis `json:"weather_impact"`
	OptimizationMetrics  SGOptimizationMetrics      `json:"optimization_metrics"`
	PlayerAnalysis       []SGPlayerAnalysis         `json:"player_analysis"`
	ExecutionTime        time.Duration              `json:"execution_time"`
}

// SGOptimizationMetrics provides detailed metrics about the optimization
type SGOptimizationMetrics struct {
	TotalPlayersAnalyzed  int     `json:"total_players_analyzed"`
	AvgSGTotal           float64 `json:"avg_sg_total"`
	AvgCourseFit         float64 `json:"avg_course_fit"`
	AvgCutProbability    float64 `json:"avg_cut_probability"`
	CorrelationScore     float64 `json:"correlation_score"`
	DiversificationScore float64 `json:"diversification_score"`
	WeatherAdjustment    float64 `json:"weather_adjustment"`
	OptimalityScore      float64 `json:"optimality_score"`
}

// SGPlayerAnalysis provides analysis for individual players
type SGPlayerAnalysis struct {
	PlayerID            string                  `json:"player_id"`
	PlayerName          string                  `json:"player_name"`
	SGMetrics           types.StrokesGainedMetrics `json:"sg_metrics"`
	CourseFit           float64                 `json:"course_fit"`
	WeatherAdvantage    float64                 `json:"weather_advantage"`
	StrategyFit         float64                 `json:"strategy_fit"`
	OptimalityScore     float64                 `json:"optimality_score"`
	RiskRating          float64                 `json:"risk_rating"`
	ProjectedPerformance ProjectedPerformance   `json:"projected_performance"`
}

// ProjectedPerformance represents projected player performance
type ProjectedPerformance struct {
	ExpectedScore       float64 `json:"expected_score"`
	ExpectedFinish      float64 `json:"expected_finish"`
	CutProbability      float64 `json:"cut_probability"`
	WinProbability      float64 `json:"win_probability"`
	Top10Probability    float64 `json:"top10_probability"`
	VarianceEstimate    float64 `json:"variance_estimate"`
}

// NewStrokesGainedOptimizer creates a new strokes gained optimizer
func NewStrokesGainedOptimizer(
	dataProvider EnhancedGolfDataProvider,
	cache types.CacheProvider,
	logger *logrus.Entry,
) *StrokesGainedOptimizer {
	optimizer := &StrokesGainedOptimizer{
		dataProvider:        dataProvider,
		cache:              cache,
		logger:             logger,
		optimizationWeights: make(map[string]float64),
		strategyProfiles:    make(map[string]*StrategyProfile),
		volatilityTargets:   make(map[string]float64),
	}
	
	// Initialize default strategy profiles
	optimizer.initializeDefaultStrategies()
	
	return optimizer
}

// initializeDefaultStrategies sets up the default golf optimization strategies
func (sgo *StrokesGainedOptimizer) initializeDefaultStrategies() {
	// Win Strategy - High risk, high reward
	sgo.strategyProfiles["Win"] = &StrategyProfile{
		Name:        "Win",
		Description: "High-upside strategy targeting tournament wins",
		SGWeights: SGCategoryWeights{
			OffTheTee:      1.0,
			Approach:       1.3, // Premium on approach play for scoring
			AroundTheGreen: 1.1,
			Putting:        1.0,
		},
		VolatilityTolerance: 0.8, // High volatility tolerance
		CorrelationTargets: CorrelationTargets{
			MaxPlayerCorrelation: 0.3,
			TeeTimeWeight:        0.2,
			WeatherWeight:        0.1,
			SkillWeight:          0.4,
			OwnershipWeight:      0.3,
		},
		CutProbabilityMin: 0.65, // Lower cut probability acceptable for upside
		UpsideTargets: UpsideTargets{
			WinProbabilityMin:   0.02,
			Top5ProbabilityMin:  0.08,
			Top10ProbabilityMin: 0.15,
			VolatilityTarget:    0.8,
			CeilingTarget:       95.0,
		},
		RiskProfile: RiskProfile{
			MaxExposurePerPlayer: 0.4,
			MaxStackSize:         3,
			DiversificationMin:   0.6,
			VarianceTarget:       0.8,
			SafetyFirst:          false,
		},
		WeatherAdaptation: WeatherAdaptation{
			WeatherSensitivity: 0.3,
			AdaptationFactors: map[string]float64{
				"wind_high":    0.2,
				"rain_heavy":   0.1,
				"temperature_extreme": 0.15,
			},
		},
	}

	// Top5 Strategy - Balanced upside with some safety
	sgo.strategyProfiles["Top5"] = &StrategyProfile{
		Name:        "Top5",
		Description: "Balanced strategy targeting top-5 finishes",
		SGWeights: SGCategoryWeights{
			OffTheTee:      1.1,
			Approach:       1.2,
			AroundTheGreen: 1.0,
			Putting:        1.0,
		},
		VolatilityTolerance: 0.6,
		CorrelationTargets: CorrelationTargets{
			MaxPlayerCorrelation: 0.25,
			TeeTimeWeight:        0.25,
			WeatherWeight:        0.15,
			SkillWeight:          0.35,
			OwnershipWeight:      0.25,
		},
		CutProbabilityMin: 0.75,
		UpsideTargets: UpsideTargets{
			WinProbabilityMin:   0.01,
			Top5ProbabilityMin:  0.12,
			Top10ProbabilityMin: 0.20,
			VolatilityTarget:    0.6,
			CeilingTarget:       85.0,
		},
		RiskProfile: RiskProfile{
			MaxExposurePerPlayer: 0.35,
			MaxStackSize:         2,
			DiversificationMin:   0.7,
			VarianceTarget:       0.6,
			SafetyFirst:          false,
		},
		WeatherAdaptation: WeatherAdaptation{
			WeatherSensitivity: 0.25,
			AdaptationFactors: map[string]float64{
				"wind_high":    0.15,
				"rain_heavy":   0.1,
				"temperature_extreme": 0.1,
			},
		},
	}

	// Top10 Strategy - More conservative upside
	sgo.strategyProfiles["Top10"] = &StrategyProfile{
		Name:        "Top10",
		Description: "Conservative upside strategy targeting top-10 finishes",
		SGWeights: SGCategoryWeights{
			OffTheTee:      1.0,
			Approach:       1.1,
			AroundTheGreen: 1.0,
			Putting:        1.1, // Premium on putting for consistency
		},
		VolatilityTolerance: 0.5,
		CorrelationTargets: CorrelationTargets{
			MaxPlayerCorrelation: 0.2,
			TeeTimeWeight:        0.3,
			WeatherWeight:        0.2,
			SkillWeight:          0.3,
			OwnershipWeight:      0.2,
		},
		CutProbabilityMin: 0.80,
		UpsideTargets: UpsideTargets{
			WinProbabilityMin:   0.005,
			Top5ProbabilityMin:  0.08,
			Top10ProbabilityMin: 0.25,
			VolatilityTarget:    0.5,
			CeilingTarget:       75.0,
		},
		RiskProfile: RiskProfile{
			MaxExposurePerPlayer: 0.30,
			MaxStackSize:         2,
			DiversificationMin:   0.75,
			VarianceTarget:       0.5,
			SafetyFirst:          false,
		},
		WeatherAdaptation: WeatherAdaptation{
			WeatherSensitivity: 0.2,
			AdaptationFactors: map[string]float64{
				"wind_high":    0.1,
				"rain_heavy":   0.05,
				"temperature_extreme": 0.05,
			},
		},
	}

	// Top25 Strategy - Safety-first approach
	sgo.strategyProfiles["Top25"] = &StrategyProfile{
		Name:        "Top25",
		Description: "Safety-first strategy targeting top-25 finishes",
		SGWeights: SGCategoryWeights{
			OffTheTee:      0.9,
			Approach:       1.0,
			AroundTheGreen: 1.0,
			Putting:        1.2, // High premium on putting consistency
		},
		VolatilityTolerance: 0.3,
		CorrelationTargets: CorrelationTargets{
			MaxPlayerCorrelation: 0.15,
			TeeTimeWeight:        0.35,
			WeatherWeight:        0.25,
			SkillWeight:          0.25,
			OwnershipWeight:      0.15,
		},
		CutProbabilityMin: 0.85,
		UpsideTargets: UpsideTargets{
			WinProbabilityMin:   0.001,
			Top5ProbabilityMin:  0.05,
			Top10ProbabilityMin: 0.15,
			VolatilityTarget:    0.3,
			CeilingTarget:       65.0,
		},
		RiskProfile: RiskProfile{
			MaxExposurePerPlayer: 0.25,
			MaxStackSize:         1,
			DiversificationMin:   0.85,
			VarianceTarget:       0.3,
			SafetyFirst:          true,
		},
		WeatherAdaptation: WeatherAdaptation{
			WeatherSensitivity: 0.15,
			AdaptationFactors: map[string]float64{
				"wind_high":    0.05,
				"rain_heavy":   0.03,
				"temperature_extreme": 0.03,
			},
		},
	}

	// Cut Strategy - Making the cut focus
	sgo.strategyProfiles["Cut"] = &StrategyProfile{
		Name:        "Cut",
		Description: "Conservative strategy focused on making the cut",
		SGWeights: SGCategoryWeights{
			OffTheTee:      0.8, // Less premium on distance
			Approach:       1.0,
			AroundTheGreen: 1.1, // Premium on scrambling
			Putting:        1.3, // High premium on putting
		},
		VolatilityTolerance: 0.2,
		CorrelationTargets: CorrelationTargets{
			MaxPlayerCorrelation: 0.1,
			TeeTimeWeight:        0.4,
			WeatherWeight:        0.3,
			SkillWeight:          0.2,
			OwnershipWeight:      0.1,
		},
		CutProbabilityMin: 0.90,
		UpsideTargets: UpsideTargets{
			WinProbabilityMin:   0.0,
			Top5ProbabilityMin:  0.02,
			Top10ProbabilityMin: 0.08,
			VolatilityTarget:    0.2,
			CeilingTarget:       55.0,
		},
		RiskProfile: RiskProfile{
			MaxExposurePerPlayer: 0.20,
			MaxStackSize:         1,
			DiversificationMin:   0.9,
			VarianceTarget:       0.2,
			SafetyFirst:          true,
		},
		WeatherAdaptation: WeatherAdaptation{
			WeatherSensitivity: 0.1,
			AdaptationFactors: map[string]float64{
				"wind_high":    0.02,
				"rain_heavy":   0.01,
				"temperature_extreme": 0.01,
			},
		},
	}

	// Balanced Strategy - Well-rounded approach
	sgo.strategyProfiles["Balanced"] = &StrategyProfile{
		Name:        "Balanced",
		Description: "Balanced strategy with moderate risk and upside",
		SGWeights: SGCategoryWeights{
			OffTheTee:      1.0,
			Approach:       1.0,
			AroundTheGreen: 1.0,
			Putting:        1.0,
		},
		VolatilityTolerance: 0.5,
		CorrelationTargets: CorrelationTargets{
			MaxPlayerCorrelation: 0.2,
			TeeTimeWeight:        0.25,
			WeatherWeight:        0.2,
			SkillWeight:          0.3,
			OwnershipWeight:      0.25,
		},
		CutProbabilityMin: 0.75,
		UpsideTargets: UpsideTargets{
			WinProbabilityMin:   0.005,
			Top5ProbabilityMin:  0.06,
			Top10ProbabilityMin: 0.15,
			VolatilityTarget:    0.5,
			CeilingTarget:       70.0,
		},
		RiskProfile: RiskProfile{
			MaxExposurePerPlayer: 0.3,
			MaxStackSize:         2,
			DiversificationMin:   0.75,
			VarianceTarget:       0.5,
			SafetyFirst:          false,
		},
		WeatherAdaptation: WeatherAdaptation{
			WeatherSensitivity: 0.2,
			AdaptationFactors: map[string]float64{
				"wind_high":    0.1,
				"rain_heavy":   0.05,
				"temperature_extreme": 0.05,
			},
		},
	}
}

// OptimizeWithStrokesGained performs optimization using strokes gained analytics
func (sgo *StrokesGainedOptimizer) OptimizeWithStrokesGained(
	ctx context.Context,
	players []types.Player,
	config SGOptimizationConfig,
) (*SGOptimizationResult, error) {
	startTime := time.Now()
	
	sgo.logger.WithFields(logrus.Fields{
		"tournament_id": config.TournamentID,
		"strategy":      config.Strategy,
		"num_lineups":   config.NumLineups,
		"players":       len(players),
	}).Info("Starting strokes gained optimization")

	// Get strategy profile
	strategy, exists := sgo.strategyProfiles[config.Strategy]
	if !exists {
		return nil, fmt.Errorf("unknown strategy: %s", config.Strategy)
	}

	// Get enhanced data
	enhancedData, err := sgo.gatherEnhancedData(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to gather enhanced data: %w", err)
	}

	// Analyze players with strokes gained data
	playerAnalysis, err := sgo.analyzePlayersWithSG(ctx, players, config, enhancedData)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze players: %w", err)
	}

	// Apply strategy-specific scoring
	scoredPlayers := sgo.applyStrategyScoring(playerAnalysis, strategy, enhancedData)

	// Generate optimized lineups
	lineups, err := sgo.generateOptimizedLineups(scoredPlayers, config, strategy, enhancedData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate lineups: %w", err)
	}

	// Calculate optimization metrics
	metrics := sgo.calculateOptimizationMetrics(playerAnalysis, lineups, enhancedData)

	executionTime := time.Since(startTime)

	result := &SGOptimizationResult{
		Lineups:             lineups,
		Strategy:            strategy,
		CourseAnalysis:      enhancedData.CourseAnalysis,
		WeatherImpact:       enhancedData.WeatherImpact,
		OptimizationMetrics: metrics,
		PlayerAnalysis:      playerAnalysis,
		ExecutionTime:       executionTime,
	}

	sgo.logger.WithFields(logrus.Fields{
		"execution_time": executionTime,
		"lineups_generated": len(lineups),
		"avg_sg_total": metrics.AvgSGTotal,
		"optimality_score": metrics.OptimalityScore,
	}).Info("Strokes gained optimization completed")

	return result, nil
}

// EnhancedData holds all the advanced data needed for optimization
type EnhancedData struct {
	CourseAnalysis    *types.CourseAnalytics
	WeatherImpact     *types.WeatherImpactAnalysis
	Predictions       *types.TournamentPredictions
	LiveData          *types.LiveTournamentData
}

// gatherEnhancedData collects all enhanced data needed for optimization
func (sgo *StrokesGainedOptimizer) gatherEnhancedData(ctx context.Context, config SGOptimizationConfig) (*EnhancedData, error) {
	data := &EnhancedData{}
	var err error

	// Get predictions (always needed)
	data.Predictions, err = sgo.dataProvider.GetPreTournamentPredictions(config.TournamentID)
	if err != nil {
		sgo.logger.WithError(err).Warn("Failed to get predictions, continuing without them")
	}

	// Get course analytics if using course fit
	if config.UseCourseFit && data.Predictions != nil && data.Predictions.CourseModel.KeyFactors != nil {
		// Try to extract course ID from predictions or use a default approach
		courseID := sgo.extractCourseID(data.Predictions)
		if courseID != "" {
			data.CourseAnalysis, err = sgo.dataProvider.GetCourseAnalytics(courseID)
			if err != nil {
				sgo.logger.WithError(err).Warn("Failed to get course analytics")
			}
		}
	}

	// Get weather impact if using weather data
	if config.UseWeatherData {
		data.WeatherImpact, err = sgo.dataProvider.GetWeatherImpactData(config.TournamentID)
		if err != nil {
			sgo.logger.WithError(err).Warn("Failed to get weather impact data")
		}
	}

	// Try to get live data (optional)
	data.LiveData, err = sgo.dataProvider.GetLiveTournamentData(config.TournamentID)
	if err != nil {
		sgo.logger.WithError(err).Debug("No live data available (normal for pre-tournament)")
	}

	return data, nil
}

// extractCourseID attempts to extract course ID from predictions or other data
func (sgo *StrokesGainedOptimizer) extractCourseID(predictions *types.TournamentPredictions) string {
	// This would need to be implemented based on how course IDs are stored
	// For now, return empty string to indicate course ID extraction needs implementation
	return ""
}

// analyzePlayersWithSG analyzes players using strokes gained data
func (sgo *StrokesGainedOptimizer) analyzePlayersWithSG(
	ctx context.Context,
	players []types.Player,
	config SGOptimizationConfig,
	enhancedData *EnhancedData,
) ([]SGPlayerAnalysis, error) {
	var analysis []SGPlayerAnalysis

	for _, player := range players {
		playerAnalysis := SGPlayerAnalysis{
			PlayerID:   player.ID.String(),
			PlayerName: player.Name,
		}

		// Get strokes gained data for player
		sgData, err := sgo.dataProvider.GetStrokesGainedData(player.ExternalID, config.TournamentID)
		if err != nil {
			sgo.logger.WithError(err).WithField("player", player.Name).Debug("No SG data available")
			// Use default/estimated SG data
			playerAnalysis.SGMetrics = sgo.estimateStrokesGained(player)
		} else {
			playerAnalysis.SGMetrics = *sgData
		}

		// Calculate course fit if available
		if enhancedData.CourseAnalysis != nil {
			playerAnalysis.CourseFit = sgo.calculateCourseFit(player, enhancedData.CourseAnalysis)
		}

		// Calculate weather advantage if available
		if enhancedData.WeatherImpact != nil {
			playerAnalysis.WeatherAdvantage = sgo.calculateWeatherAdvantage(player, enhancedData.WeatherImpact)
		}

		// Calculate strategy fit
		strategy := sgo.strategyProfiles[config.Strategy]
		playerAnalysis.StrategyFit = sgo.calculateStrategyFit(playerAnalysis.SGMetrics, strategy)

		// Calculate projected performance
		playerAnalysis.ProjectedPerformance = sgo.calculateProjectedPerformance(player, &playerAnalysis, enhancedData)

		// Calculate optimality score
		playerAnalysis.OptimalityScore = sgo.calculateOptimalityScore(&playerAnalysis, strategy, enhancedData)

		// Calculate risk rating
		playerAnalysis.RiskRating = sgo.calculateRiskRating(&playerAnalysis, strategy)

		analysis = append(analysis, playerAnalysis)
	}

	return analysis, nil
}

// estimateStrokesGained provides estimated SG data when actual data is unavailable
func (sgo *StrokesGainedOptimizer) estimateStrokesGained(player types.Player) types.StrokesGainedMetrics {
	// This is a simplified estimation - in production, you'd use more sophisticated methods
	// Based on salary, projected points, and historical averages
	
	baseSG := 0.0
	if player.ProjectedPoints != nil && *player.ProjectedPoints > 0 {
		// Rough estimation based on projected points
		// PGA Tour average is around 0.0 SG Total
		baseSG = (*player.ProjectedPoints - 45.0) / 15.0 // Normalize around average scoring
	}
	
	return types.StrokesGainedMetrics{
		PlayerID:     int64(player.ID.ID()), // Convert UUID to int64 (simplified)
		SGTotal:      baseSG,
		SGOffTheTee:  baseSG * 0.3,
		SGApproach:   baseSG * 0.4,
		SGAroundTheGreen: baseSG * 0.2,
		SGPutting:    baseSG * 0.1,
		Consistency:  0.5, // Default consistency
		VolatilityIndex: 0.3, // Default volatility
		UpdatedAt:    time.Now(),
	}
}

// calculateCourseFit calculates how well a player fits the course
func (sgo *StrokesGainedOptimizer) calculateCourseFit(player types.Player, courseAnalytics *types.CourseAnalytics) float64 {
	// This would implement sophisticated course fit calculation
	// For now, return a placeholder based on player salary (higher salary = better fit)
	fit := 0.5 // Base fit
	
	if player.SalaryDK != nil && *player.SalaryDK > 0 {
		// Normalize salary to 0-1 range and use as fit proxy
		fit = math.Min(1.0, float64(*player.SalaryDK)/12000.0)
	}
	
	return fit
}

// calculateWeatherAdvantage calculates weather advantage for a player
func (sgo *StrokesGainedOptimizer) calculateWeatherAdvantage(player types.Player, weatherImpact *types.WeatherImpactAnalysis) float64 {
	// Look for player-specific weather impact
	for _, impact := range weatherImpact.PlayerImpacts {
		if fmt.Sprintf("%d", impact.PlayerID) == player.ExternalID {
			return impact.WeatherAdvantage
		}
	}
	
	// Default to neutral weather advantage
	return 0.0
}

// calculateStrategyFit calculates how well a player fits the strategy
func (sgo *StrokesGainedOptimizer) calculateStrategyFit(sgMetrics types.StrokesGainedMetrics, strategy *StrategyProfile) float64 {
	fit := 0.0
	
	// Weight each SG category by strategy preferences
	fit += sgMetrics.SGOffTheTee * strategy.SGWeights.OffTheTee
	fit += sgMetrics.SGApproach * strategy.SGWeights.Approach
	fit += sgMetrics.SGAroundTheGreen * strategy.SGWeights.AroundTheGreen
	fit += sgMetrics.SGPutting * strategy.SGWeights.Putting
	
	// Normalize by total weights
	totalWeight := strategy.SGWeights.OffTheTee + strategy.SGWeights.Approach + 
	               strategy.SGWeights.AroundTheGreen + strategy.SGWeights.Putting
	
	if totalWeight > 0 {
		fit = fit / totalWeight
	}
	
	// Apply volatility adjustment
	volatilityAdjustment := 1.0 - math.Abs(sgMetrics.VolatilityIndex - strategy.VolatilityTolerance)
	fit = fit * volatilityAdjustment
	
	return math.Max(0.0, math.Min(1.0, fit))
}

// calculateProjectedPerformance calculates projected performance metrics
func (sgo *StrokesGainedOptimizer) calculateProjectedPerformance(
	player types.Player,
	analysis *SGPlayerAnalysis,
	enhancedData *EnhancedData,
) ProjectedPerformance {
	// Base projected performance on strokes gained and adjustments
	baseSG := analysis.SGMetrics.SGTotal
	
	// Apply course fit adjustment
	courseFitAdjustment := analysis.CourseFit * 0.5 // Up to 0.5 stroke adjustment
	
	// Apply weather adjustment
	weatherAdjustment := analysis.WeatherAdvantage * 0.3 // Up to 0.3 stroke adjustment
	
	// Calculate adjusted SG
	adjustedSG := baseSG + courseFitAdjustment + weatherAdjustment
	
	// Convert to expected performance metrics
	expectedScore := 72.0 - adjustedSG * 4.0 // Rough conversion: 1 SG = 4 strokes over par
	expectedFinish := sgo.sgToExpectedFinish(adjustedSG)
	cutProbability := sgo.sgToCutProbability(adjustedSG)
	winProbability := sgo.sgToWinProbability(adjustedSG)
	top10Probability := sgo.sgToTop10Probability(adjustedSG)
	variance := analysis.SGMetrics.VolatilityIndex * 3.0 // Volatility to score variance
	
	return ProjectedPerformance{
		ExpectedScore:    expectedScore,
		ExpectedFinish:   expectedFinish,
		CutProbability:   cutProbability,
		WinProbability:   winProbability,
		Top10Probability: top10Probability,
		VarianceEstimate: variance,
	}
}

// Helper functions for SG to probability conversions
func (sgo *StrokesGainedOptimizer) sgToExpectedFinish(sg float64) float64 {
	// Rough conversion: better SG = better finish
	// This would be calibrated with historical data
	return math.Max(1.0, 75.0 - sg * 30.0)
}

func (sgo *StrokesGainedOptimizer) sgToCutProbability(sg float64) float64 {
	// Convert SG to cut probability using logistic function
	return 1.0 / (1.0 + math.Exp(-(sg + 1.0) * 3.0))
}

func (sgo *StrokesGainedOptimizer) sgToWinProbability(sg float64) float64 {
	// Convert SG to win probability (much lower probabilities)
	return math.Max(0.001, math.Min(0.15, 0.02 * math.Exp(sg * 2.0)))
}

func (sgo *StrokesGainedOptimizer) sgToTop10Probability(sg float64) float64 {
	// Convert SG to top-10 probability
	return math.Max(0.01, math.Min(0.8, 0.2 * (1.0 + sg * 2.0)))
}

// calculateOptimalityScore calculates overall optimality score for a player
func (sgo *StrokesGainedOptimizer) calculateOptimalityScore(
	analysis *SGPlayerAnalysis,
	strategy *StrategyProfile,
	enhancedData *EnhancedData,
) float64 {
	score := 0.0
	
	// Base score from strategy fit
	score += analysis.StrategyFit * 0.4
	
	// Course fit contribution
	score += analysis.CourseFit * 0.2
	
	// Weather advantage contribution
	score += math.Max(0.0, analysis.WeatherAdvantage) * 0.1
	
	// Cut probability contribution (varies by strategy)
	cutWeight := 1.0 - strategy.UpsideTargets.VolatilityTarget // More conservative = higher cut weight
	score += analysis.ProjectedPerformance.CutProbability * cutWeight * 0.2
	
	// Upside contribution (varies by strategy)
	upsideWeight := strategy.UpsideTargets.VolatilityTarget
	upsideScore := analysis.ProjectedPerformance.WinProbability * 0.5 + 
	               analysis.ProjectedPerformance.Top10Probability * 0.3
	score += upsideScore * upsideWeight * 0.1
	
	return math.Max(0.0, math.Min(1.0, score))
}

// calculateRiskRating calculates risk rating for a player
func (sgo *StrokesGainedOptimizer) calculateRiskRating(analysis *SGPlayerAnalysis, strategy *StrategyProfile) float64 {
	risk := 0.0
	
	// Base risk from volatility
	risk += analysis.SGMetrics.VolatilityIndex * 0.4
	
	// Cut risk
	cutRisk := 1.0 - analysis.ProjectedPerformance.CutProbability
	risk += cutRisk * 0.3
	
	// Performance variance risk
	risk += analysis.ProjectedPerformance.VarianceEstimate / 10.0 * 0.2
	
	// Weather uncertainty risk
	if analysis.WeatherAdvantage < 0 {
		risk += math.Abs(analysis.WeatherAdvantage) * 0.1
	}
	
	return math.Max(0.0, math.Min(1.0, risk))
}

// applyStrategyScoring applies strategy-specific scoring to players
func (sgo *StrokesGainedOptimizer) applyStrategyScoring(
	playerAnalysis []SGPlayerAnalysis,
	strategy *StrategyProfile,
	enhancedData *EnhancedData,
) []SGPlayerAnalysis {
	// Sort players by optimality score
	sort.Slice(playerAnalysis, func(i, j int) bool {
		return playerAnalysis[i].OptimalityScore > playerAnalysis[j].OptimalityScore
	})
	
	// Apply strategy-specific adjustments
	for i := range playerAnalysis {
		// Safety-first strategies penalize high-risk players
		if strategy.RiskProfile.SafetyFirst {
			riskPenalty := playerAnalysis[i].RiskRating * 0.2
			playerAnalysis[i].OptimalityScore = math.Max(0.0, playerAnalysis[i].OptimalityScore - riskPenalty)
		}
		
		// High-upside strategies bonus for win probability
		if strategy.UpsideTargets.VolatilityTarget > 0.6 {
			winBonus := playerAnalysis[i].ProjectedPerformance.WinProbability * 0.1
			playerAnalysis[i].OptimalityScore = math.Min(1.0, playerAnalysis[i].OptimalityScore + winBonus)
		}
		
		// Cut probability minimum enforcement
		if playerAnalysis[i].ProjectedPerformance.CutProbability < strategy.CutProbabilityMin {
			cutPenalty := (strategy.CutProbabilityMin - playerAnalysis[i].ProjectedPerformance.CutProbability) * 0.5
			playerAnalysis[i].OptimalityScore = math.Max(0.0, playerAnalysis[i].OptimalityScore - cutPenalty)
		}
	}
	
	return playerAnalysis
}

// generateOptimizedLineups generates lineups using the scored players
func (sgo *StrokesGainedOptimizer) generateOptimizedLineups(
	scoredPlayers []SGPlayerAnalysis,
	config SGOptimizationConfig,
	strategy *StrategyProfile,
	enhancedData *EnhancedData,
) ([]types.GeneratedLineup, error) {
	// This is a simplified implementation
	// In practice, this would use sophisticated optimization algorithms
	
	var lineups []types.GeneratedLineup
	
	// Generate the requested number of lineups
	for i := 0; i < config.NumLineups; i++ {
		lineup, err := sgo.generateSingleLineup(scoredPlayers, config, strategy, i)
		if err != nil {
			sgo.logger.WithError(err).WithField("lineup_index", i).Warn("Failed to generate lineup")
			continue
		}
		
		lineups = append(lineups, lineup)
	}
	
	return lineups, nil
}

// generateSingleLineup generates a single optimized lineup
func (sgo *StrokesGainedOptimizer) generateSingleLineup(
	scoredPlayers []SGPlayerAnalysis,
	config SGOptimizationConfig,
	strategy *StrategyProfile,
	lineupIndex int,
) (types.GeneratedLineup, error) {
	// Simplified lineup generation - select top 6 players for golf
	// In practice, this would use knapsack optimization with salary constraints
	
	var selectedPlayers []types.LineupPlayer
	totalSalary := 0
	totalProjectedPoints := 0.0
	
	// Select players with some diversity (skip some top players for later lineups)
	skipCount := lineupIndex * 2 // Simple diversity mechanism
	playerIndex := 0
	
	for len(selectedPlayers) < 6 && playerIndex < len(scoredPlayers) {
		if skipCount > 0 {
			skipCount--
			playerIndex++
			continue
		}
		
		player := scoredPlayers[playerIndex]
		
		// Create lineup player (simplified - would need to convert from analysis)
		lineupPlayer := types.LineupPlayer{
			ID:              uuid.MustParse(player.PlayerID),
			Name:            player.PlayerName,
			Position:        "G", // Golf position
			Salary:          8000, // Placeholder - would get from original player data
			ProjectedPoints: player.ProjectedPerformance.ExpectedScore,
		}
		
		// Check salary cap (simplified)
		if totalSalary + lineupPlayer.Salary <= config.SalaryCap {
			selectedPlayers = append(selectedPlayers, lineupPlayer)
			totalSalary += lineupPlayer.Salary
			totalProjectedPoints += lineupPlayer.ProjectedPoints
		}
		
		playerIndex++
	}
	
	if len(selectedPlayers) < 6 {
		return types.GeneratedLineup{}, fmt.Errorf("could not generate valid lineup")
	}
	
	lineup := types.GeneratedLineup{
		ID:              fmt.Sprintf("sg_lineup_%d_%s", lineupIndex+1, uuid.New().String()[:8]),
		Players:         selectedPlayers,
		TotalSalary:     totalSalary,
		ProjectedPoints: totalProjectedPoints,
		Exposure:        0.0, // Would be calculated based on lineup count
		StackDescription: fmt.Sprintf("%s Strategy", strategy.Name),
	}
	
	return lineup, nil
}

// calculateOptimizationMetrics calculates comprehensive metrics about the optimization
func (sgo *StrokesGainedOptimizer) calculateOptimizationMetrics(
	playerAnalysis []SGPlayerAnalysis,
	lineups []types.GeneratedLineup,
	enhancedData *EnhancedData,
) SGOptimizationMetrics {
	metrics := SGOptimizationMetrics{
		TotalPlayersAnalyzed: len(playerAnalysis),
	}
	
	// Calculate averages across all analyzed players
	if len(playerAnalysis) > 0 {
		totalSG := 0.0
		totalCourseFit := 0.0
		totalCutProb := 0.0
		
		for _, analysis := range playerAnalysis {
			totalSG += analysis.SGMetrics.SGTotal
			totalCourseFit += analysis.CourseFit
			totalCutProb += analysis.ProjectedPerformance.CutProbability
		}
		
		count := float64(len(playerAnalysis))
		metrics.AvgSGTotal = totalSG / count
		metrics.AvgCourseFit = totalCourseFit / count
		metrics.AvgCutProbability = totalCutProb / count
	}
	
	// Calculate lineup-specific metrics
	if len(lineups) > 0 {
		// Calculate correlation score (simplified)
		metrics.CorrelationScore = 0.7 // Placeholder
		
		// Calculate diversification score
		metrics.DiversificationScore = sgo.calculateDiversificationScore(lineups)
		
		// Weather adjustment impact
		if enhancedData.WeatherImpact != nil {
			metrics.WeatherAdjustment = enhancedData.WeatherImpact.OverallImpact
		}
		
		// Overall optimality score
		totalOptimality := 0.0
		for _, lineup := range lineups {
			totalOptimality += lineup.ProjectedPoints
		}
		metrics.OptimalityScore = (totalOptimality / float64(len(lineups))) / 100.0 // Normalize
	}
	
	return metrics
}

// calculateDiversificationScore calculates how diversified the lineups are
func (sgo *StrokesGainedOptimizer) calculateDiversificationScore(lineups []types.GeneratedLineup) float64 {
	if len(lineups) <= 1 {
		return 1.0
	}
	
	// Count unique players across all lineups
	uniquePlayers := make(map[uuid.UUID]bool)
	totalPlayers := 0
	
	for _, lineup := range lineups {
		for _, player := range lineup.Players {
			uniquePlayers[player.ID] = true
			totalPlayers++
		}
	}
	
	// Diversification = unique players / total player slots
	if totalPlayers > 0 {
		return float64(len(uniquePlayers)) / float64(totalPlayers)
	}
	
	return 0.0
}

// GetStrategyProfile returns a strategy profile by name
func (sgo *StrokesGainedOptimizer) GetStrategyProfile(name string) (*StrategyProfile, error) {
	strategy, exists := sgo.strategyProfiles[name]
	if !exists {
		return nil, fmt.Errorf("strategy not found: %s", name)
	}
	return strategy, nil
}

// GetAvailableStrategies returns all available strategy names
func (sgo *StrokesGainedOptimizer) GetAvailableStrategies() []string {
	var strategies []string
	for name := range sgo.strategyProfiles {
		strategies = append(strategies, name)
	}
	sort.Strings(strategies)
	return strategies
}