package optimizer

import (
	"fmt"
	"math"

	"github.com/stitts-dev/dfs-sim/shared/types"
	"github.com/sirupsen/logrus"
)

// OptimizationObjective represents different optimization strategies
type OptimizationObjective string

const (
	MaxCeiling     OptimizationObjective = "ceiling"     // GPP tournaments - maximize upside potential
	MaxFloor       OptimizationObjective = "floor"       // Cash games - maximize safety
	Balanced       OptimizationObjective = "balanced"    // Balanced risk/reward
	Contrarian     OptimizationObjective = "contrarian"  // Low ownership tournaments
	Correlation    OptimizationObjective = "correlation" // Stack-heavy optimization
	Value          OptimizationObjective = "value"       // Points per dollar optimization
)

// ObjectiveConfig contains configuration for multi-objective optimization
type ObjectiveConfig struct {
	Objective          OptimizationObjective `json:"objective"`
	Weight             float64               `json:"weight"`             // Weight for this objective (0-1)
	OwnershipThreshold float64               `json:"ownership_threshold"` // For contrarian strategy
	CorrelationWeight  float64               `json:"correlation_weight"`  // For correlation strategy
	RiskTolerance      float64               `json:"risk_tolerance"`      // Risk tolerance (0-1)
	ValueThreshold     float64               `json:"value_threshold"`     // Minimum value per dollar
}


// ObjectiveManager handles multi-objective optimization scoring
type ObjectiveManager struct {
	config       ObjectiveConfig
	analytics    *AnalyticsEngine
	correlations *CorrelationMatrix
	platform     string
}

// NewObjectiveManager creates a new objective manager
func NewObjectiveManager(config ObjectiveConfig, analytics *AnalyticsEngine, correlations *CorrelationMatrix, platform string) *ObjectiveManager {
	return &ObjectiveManager{
		config:       config,
		analytics:    analytics,
		correlations: correlations,
		platform:     platform,
	}
}

// CalculateObjectiveScore computes player score based on optimization objective
func (om *ObjectiveManager) CalculateObjectiveScore(player types.Player, lineup []types.Player, analytics *PlayerAnalytics) float64 {
	if analytics == nil {
		// Fallback to basic projection if analytics unavailable
		return om.calculateBasicScore(player)
	}

	switch om.config.Objective {
	case MaxCeiling:
		return om.calculateCeilingScore(player, analytics)
	case MaxFloor:
		return om.calculateFloorScore(player, analytics)
	case Balanced:
		return om.calculateBalancedScore(player, analytics)
	case Contrarian:
		return om.calculateContrarianscore(player, analytics)
	case Correlation:
		return om.calculateCorrelationScore(player, lineup, analytics)
	case Value:
		return om.calculateValueScore(player, analytics)
	default:
		logrus.Warnf("Unknown optimization objective: %s, using balanced", om.config.Objective)
		return om.calculateBalancedScore(player, analytics)
	}
}

// calculateCeilingScore optimizes for maximum upside potential (GPP tournaments)
func (om *ObjectiveManager) calculateCeilingScore(player types.Player, analytics *PlayerAnalytics) float64 {
	// Heavily weight ceiling and ceiling probability
	ceilingWeight := 0.6
	projectionWeight := 0.25
	probabilityWeight := 0.15

	ceilingScore := analytics.Ceiling * ceilingWeight
	projectionScore := analytics.BaseProjection * projectionWeight
	probabilityScore := analytics.CeilingProbability * analytics.BaseProjection * probabilityWeight

	// Bonus for high upside potential
	upsideBonus := analytics.UpsideRatio * 0.2

	// Penalty for high ownership (want contrarian in GPPs)
	ownershipPenalty := 0.0
	ownership := om.getPlayerOwnership(player)
	if ownership > 25.0 {
		ownershipPenalty = math.Pow(ownership/100, 1.5) * analytics.BaseProjection * 0.1
	}

	totalScore := ceilingScore + projectionScore + probabilityScore + upsideBonus - ownershipPenalty

	logrus.Debugf("Ceiling score for %s: ceiling=%.2f, projection=%.2f, upside=%.2f, penalty=%.2f, total=%.2f",
		player.Name, ceilingScore, projectionScore, upsideBonus, ownershipPenalty, totalScore)

	return math.Max(0, totalScore)
}

// calculateFloorScore optimizes for safety and consistency (cash games)
func (om *ObjectiveManager) calculateFloorScore(player types.Player, analytics *PlayerAnalytics) float64 {
	// Heavily weight floor and consistency
	floorWeight := 0.5
	consistencyWeight := 0.3
	projectionWeight := 0.2

	floorScore := analytics.Floor * floorWeight
	consistencyScore := analytics.ConsistencyScore * analytics.BaseProjection * consistencyWeight
	projectionScore := analytics.BaseProjection * projectionWeight

	// Bonus for low volatility
	volatilityBonus := (1.0 - analytics.Volatility) * analytics.BaseProjection * 0.1

	// Penalty for injury risk
	injuryPenalty := 0.0
	if player.InjuryStatus == "Q" {
		injuryPenalty = analytics.BaseProjection * 0.1
	} else if player.InjuryStatus == "D" {
		injuryPenalty = analytics.BaseProjection * 0.2
	}

	totalScore := floorScore + consistencyScore + projectionScore + volatilityBonus - injuryPenalty

	logrus.Debugf("Floor score for %s: floor=%.2f, consistency=%.2f, volatility_bonus=%.2f, injury_penalty=%.2f, total=%.2f",
		player.Name, floorScore, consistencyScore, volatilityBonus, injuryPenalty, totalScore)

	return math.Max(0, totalScore)
}

// calculateBalancedScore balances risk and reward
func (om *ObjectiveManager) calculateBalancedScore(player types.Player, analytics *PlayerAnalytics) float64 {
	// Balanced weighting across all factors
	projectionWeight := 0.4
	ceilingWeight := 0.2
	floorWeight := 0.2
	valueWeight := 0.2

	projectionScore := analytics.BaseProjection * projectionWeight
	ceilingScore := analytics.Ceiling * ceilingWeight
	floorScore := analytics.Floor * floorWeight
	valueScore := analytics.ValueRating * analytics.BaseProjection * valueWeight

	// Moderate ownership consideration
	ownership := om.getPlayerOwnership(player)
	ownershipAdjustment := 0.0
	if ownership > 35.0 {
		ownershipAdjustment = -analytics.BaseProjection * 0.05 // Small penalty for chalk
	} else if ownership < 10.0 {
		ownershipAdjustment = analytics.BaseProjection * 0.03 // Small bonus for contrarian
	}

	totalScore := projectionScore + ceilingScore + floorScore + valueScore + ownershipAdjustment

	return math.Max(0, totalScore)
}

// calculateContrarianscore optimizes for low ownership
func (om *ObjectiveManager) calculateContrarianscore(player types.Player, analytics *PlayerAnalytics) float64 {
	baseScore := analytics.BaseProjection * 0.6

	ownership := om.getPlayerOwnership(player)
	
	// Heavy bonus for low ownership
	ownershipBonus := 0.0
	if ownership < om.config.OwnershipThreshold {
		// Exponential bonus for very low ownership
		ownershipMultiplier := math.Pow(2.0, (om.config.OwnershipThreshold-ownership)/5.0)
		ownershipBonus = baseScore * (ownershipMultiplier - 1.0) * 0.5
	}

	// Heavy penalty for high ownership
	ownershipPenalty := 0.0
	if ownership > 25.0 {
		ownershipPenalty = math.Pow(ownership/25.0, 2) * baseScore * 0.4
	}

	// Still consider ceiling for tournament upside
	ceilingBonus := analytics.UpsideRatio * 0.3

	totalScore := baseScore + ownershipBonus + ceilingBonus - ownershipPenalty

	logrus.Debugf("Contrarian score for %s (%.1f%% owned): base=%.2f, bonus=%.2f, penalty=%.2f, total=%.2f",
		player.Name, ownership, baseScore, ownershipBonus, ownershipPenalty, totalScore)

	return math.Max(0, totalScore)
}

// calculateCorrelationScore optimizes for stacking and correlation
func (om *ObjectiveManager) calculateCorrelationScore(player types.Player, lineup []types.Player, analytics *PlayerAnalytics) float64 {
	baseScore := analytics.BaseProjection * 0.5

	// Calculate correlation bonus with existing lineup
	correlationBonus := 0.0
	if om.correlations != nil && len(lineup) > 0 {
		correlationBonus = om.calculateLineupCorrelationBonus(player, lineup)
	}

	// Stack bonus based on team/game stacking
	stackBonus := om.calculateStackBonus(player, lineup)

	// Value consideration
	valueBonus := analytics.ValueRating * analytics.BaseProjection * 0.1

	totalScore := baseScore + correlationBonus + stackBonus + valueBonus

	if correlationBonus > 0 || stackBonus > 0 {
		logrus.Debugf("Correlation score for %s: base=%.2f, correlation=%.2f, stack=%.2f, total=%.2f",
			player.Name, baseScore, correlationBonus, stackBonus, totalScore)
	}

	return math.Max(0, totalScore)
}

// calculateValueScore optimizes for points per dollar
func (om *ObjectiveManager) calculateValueScore(player types.Player, analytics *PlayerAnalytics) float64 {
	// Heavy weight on value rating
	valueScore := analytics.ValueRating * analytics.BaseProjection * 0.8

	// Consider floor for value safety
	floorScore := analytics.Floor * 0.2

	// Bonus for exceeding value threshold
	valueBonus := 0.0
	if analytics.ValueRating > om.config.ValueThreshold {
		valueBonus = (analytics.ValueRating - om.config.ValueThreshold) * analytics.BaseProjection * 0.1
	}

	totalScore := valueScore + floorScore + valueBonus

	logrus.Debugf("Value score for %s (%.2f pts/$K): value=%.2f, floor=%.2f, bonus=%.2f, total=%.2f",
		player.Name, analytics.ValueRating, valueScore, floorScore, valueBonus, totalScore)

	return math.Max(0, totalScore)
}

// calculateBasicScore provides fallback scoring when analytics unavailable
func (om *ObjectiveManager) calculateBasicScore(player types.Player) float64 {
	return player.ProjectedPoints
}

// Helper functions

// getPlayerOwnership returns player ownership percentage
func (om *ObjectiveManager) getPlayerOwnership(player types.Player) float64 {
	if om.platform == "fanduel" {
		return player.OwnershipFD
	}
	return player.OwnershipDK
}

// calculateLineupCorrelationBonus calculates correlation bonus with existing lineup
func (om *ObjectiveManager) calculateLineupCorrelationBonus(player types.Player, lineup []types.Player) float64 {
	if om.correlations == nil {
		return 0.0
	}

	totalBonus := 0.0
	for _, lineupPlayer := range lineup {
		// This would use the actual correlation matrix
		correlation := om.getPlayerCorrelation(player, lineupPlayer)
		bonus := correlation * player.ProjectedPoints * om.config.CorrelationWeight
		totalBonus += bonus
	}

	return totalBonus
}

// calculateStackBonus calculates stacking bonus
func (om *ObjectiveManager) calculateStackBonus(player types.Player, lineup []types.Player) float64 {
	teammateBonusCount := 0
	opponentBonusCount := 0

	for _, lineupPlayer := range lineup {
		// Same team stack
		if player.Team == lineupPlayer.Team && player.Team != "" {
			teammateBonusCount++
		}
		// Game stack (opponent)
		if player.Team == lineupPlayer.Opponent || player.Opponent == lineupPlayer.Team {
			opponentBonusCount++
		}
	}

	// Team stack bonus (increases with more teammates)
	teamStackBonus := 0.0
	if teammateBonusCount > 0 {
		stackMultiplier := math.Pow(1.2, float64(teammateBonusCount)) // Exponential growth
		teamStackBonus = player.ProjectedPoints * 0.05 * stackMultiplier
	}

	// Game stack bonus
	gameStackBonus := 0.0
	if opponentBonusCount > 0 {
		gameStackBonus = player.ProjectedPoints * 0.03 * float64(opponentBonusCount)
	}

	return teamStackBonus + gameStackBonus
}

// getPlayerCorrelation gets correlation between two players
func (om *ObjectiveManager) getPlayerCorrelation(player1, player2 types.Player) float64 {
	// Placeholder correlation calculation
	// In production, this would use the actual correlation matrix
	
	// Same team
	if player1.Team == player2.Team && player1.Team != "" {
		return om.getTeamCorrelation(player1.Position, player2.Position)
	}
	
	// Game stack
	if player1.Team == player2.Opponent || player1.Opponent == player2.Team {
		return 0.15 // Base game correlation
	}
	
	return 0.0
}

// getTeamCorrelation returns position-based team correlation
func (om *ObjectiveManager) getTeamCorrelation(pos1, pos2 string) float64 {
	// Position-based correlations (sport-agnostic for now)
	correlations := map[string]map[string]float64{
		"QB": {"WR": 0.50, "TE": 0.35, "RB": -0.15},
		"RB": {"RB": -0.35, "WR": -0.10},
		"WR": {"WR": 0.25, "TE": 0.15},
		"PG": {"SG": 0.30, "SF": 0.20, "PF": 0.15, "C": 0.10},
		"SG": {"PG": 0.30, "SF": 0.25, "PF": 0.15, "C": 0.10},
	}
	
	if corr, exists := correlations[pos1][pos2]; exists {
		return corr
	}
	if corr, exists := correlations[pos2][pos1]; exists {
		return corr
	}
	
	return 0.05 // Default small positive correlation for teammates
}

// GetObjectiveWeights returns the current objective weights
func (om *ObjectiveManager) GetObjectiveWeights() map[string]float64 {
	switch om.config.Objective {
	case MaxCeiling:
		return map[string]float64{
			"ceiling": 0.6, "projection": 0.25, "probability": 0.15, "upside": 0.2,
		}
	case MaxFloor:
		return map[string]float64{
			"floor": 0.5, "consistency": 0.3, "projection": 0.2, "volatility": 0.1,
		}
	case Balanced:
		return map[string]float64{
			"projection": 0.4, "ceiling": 0.2, "floor": 0.2, "value": 0.2,
		}
	case Contrarian:
		return map[string]float64{
			"projection": 0.6, "ownership": 0.5, "ceiling": 0.3,
		}
	case Correlation:
		return map[string]float64{
			"projection": 0.5, "correlation": om.config.CorrelationWeight, "stack": 0.3,
		}
	case Value:
		return map[string]float64{
			"value": 0.8, "floor": 0.2, "threshold": 0.1,
		}
	default:
		return map[string]float64{"projection": 1.0}
	}
}

// SetObjective changes the optimization objective
func (om *ObjectiveManager) SetObjective(objective OptimizationObjective) {
	om.config.Objective = objective
	logrus.Infof("Optimization objective changed to: %s", objective)
}

// GetCurrentObjective returns the current optimization objective
func (om *ObjectiveManager) GetCurrentObjective() OptimizationObjective {
	return om.config.Objective
}

// ValidateObjectiveConfig validates the objective configuration
func ValidateObjectiveConfig(config ObjectiveConfig) error {
	if config.Weight < 0 || config.Weight > 1 {
		return fmt.Errorf("objective weight must be between 0 and 1, got: %f", config.Weight)
	}
	
	if config.OwnershipThreshold < 0 || config.OwnershipThreshold > 100 {
		return fmt.Errorf("ownership threshold must be between 0 and 100, got: %f", config.OwnershipThreshold)
	}
	
	if config.CorrelationWeight < 0 || config.CorrelationWeight > 1 {
		return fmt.Errorf("correlation weight must be between 0 and 1, got: %f", config.CorrelationWeight)
	}
	
	if config.RiskTolerance < 0 || config.RiskTolerance > 1 {
		return fmt.Errorf("risk tolerance must be between 0 and 1, got: %f", config.RiskTolerance)
	}
	
	return nil
}