package optimizer

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"

	"github.com/stitts-dev/dfs-sim/shared/types"
)

// PositionOptimizer handles position-specific optimization strategies for golf tournaments
type PositionOptimizer struct {
	baseOptimizer      BaseOptimizerInterface
	cutProbEngine      *CutProbabilityEngine
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
func NewPositionOptimizer(baseOptimizer BaseOptimizerInterface, cutProbEngine *CutProbabilityEngine) *PositionOptimizer {
	optimizer := &PositionOptimizer{
		baseOptimizer:   baseOptimizer,
		cutProbEngine:   cutProbEngine,
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
	// This would integrate with the player data and projections
	// For now, we'll log that strategy scores are being calculated
	log.Printf("Calculating strategy scores for %s with cut optimization: %v", config.Strategy, request.CutOptimization)
	
	// In a full implementation, this would:
	// 1. Get all players for the contest
	// 2. Get their projections (including cut probabilities)
	// 3. Calculate strategy-specific scores
	// 4. Update player scores in the optimization request
	
	return nil
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