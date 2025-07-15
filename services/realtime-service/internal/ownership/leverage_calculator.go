package ownership

import (
	"math"
	"sort"

	"github.com/sirupsen/logrus"
)

// LeverageCalculator calculates contrarian opportunity scores for DFS players
type LeverageCalculator struct {
	logger *logrus.Logger
	
	// Configuration parameters
	lowOwnershipThreshold  float64 // Ownership % threshold for "low-owned" players
	highOwnershipThreshold float64 // Ownership % threshold for "high-owned" players
	projectionWeight       float64 // Weight given to projection vs ownership in leverage calculation
	volatilityWeight       float64 // Weight given to projection volatility
	stackBonus            float64 // Bonus multiplier for stacking opportunities
}

// LeverageScore represents a player's contrarian opportunity score
type LeverageScore struct {
	PlayerID           uint    `json:"player_id"`
	Ownership          float64 `json:"ownership"`
	LeverageScore      float64 `json:"leverage_score"`     // 0-100 score
	OpportunityType    string  `json:"opportunity_type"`   // "contrarian", "balanced", "chalk"
	RecommendedExposure float64 `json:"recommended_exposure"` // Suggested % exposure in lineups
	RiskLevel          string  `json:"risk_level"`         // "low", "medium", "high"
	StackPotential     float64 `json:"stack_potential"`    // Bonus for stacking opportunities
	ProjectionGap      float64 `json:"projection_gap"`     // Difference between projection and ownership
	VolatilityAdjusted float64 `json:"volatility_adjusted"` // Score adjusted for projection uncertainty
}

// OwnershipTier represents ownership tiers for grouping players
type OwnershipTier string

const (
	TierLowOwned    OwnershipTier = "low_owned"     // < 5% ownership
	TierMediumOwned OwnershipTier = "medium_owned"  // 5-20% ownership
	TierHighOwned   OwnershipTier = "high_owned"    // 20-40% ownership
	TierChalk       OwnershipTier = "chalk"         // > 40% ownership
)

// PlayerProjection represents a player's projection data for leverage calculation
type PlayerProjection struct {
	PlayerID          uint    `json:"player_id"`
	ProjectedPoints   float64 `json:"projected_points"`
	ProjectionStdDev  float64 `json:"projection_std_dev"` // Standard deviation of projection
	Salary            int     `json:"salary"`
	Position          string  `json:"position"`
	Team              string  `json:"team"`
	Opponent          string  `json:"opponent"`
	GameEnvironment   string  `json:"game_environment"`   // "dome", "outdoor", "weather"
	InjuryRisk        float64 `json:"injury_risk"`        // 0-1 injury probability
	IsStackCandidate  bool    `json:"is_stack_candidate"` // Can be stacked with other players
}

// NewLeverageCalculator creates a new leverage calculator
func NewLeverageCalculator(logger *logrus.Logger) *LeverageCalculator {
	return &LeverageCalculator{
		logger:                 logger,
		lowOwnershipThreshold:  5.0,   // 5% ownership threshold
		highOwnershipThreshold: 20.0,  // 20% ownership threshold
		projectionWeight:       0.7,   // 70% weight to projections
		volatilityWeight:       0.3,   // 30% weight to volatility
		stackBonus:            1.2,    // 20% bonus for stack candidates
	}
}

// CalculateLeverageScores calculates leverage scores for all players
func (lc *LeverageCalculator) CalculateLeverageScores(ownership map[uint]float64, totalEntries int) map[uint]float64 {
	leverageScores := make(map[uint]float64)
	
	for playerID, ownershipPct := range ownership {
		// Basic leverage calculation based on ownership inversion
		leverageScore := lc.calculateBasicLeverage(ownershipPct, totalEntries)
		leverageScores[playerID] = leverageScore
	}
	
	return leverageScores
}

// CalculateAdvancedLeverageScores calculates advanced leverage scores with projections
func (lc *LeverageCalculator) CalculateAdvancedLeverageScores(
	ownership map[uint]float64,
	projections map[uint]PlayerProjection,
	totalEntries int,
) []LeverageScore {
	
	scores := make([]LeverageScore, 0, len(ownership))
	
	for playerID, ownershipPct := range ownership {
		projection, hasProjection := projections[playerID]
		
		// Calculate basic leverage
		basicLeverage := lc.calculateBasicLeverage(ownershipPct, totalEntries)
		
		// Calculate advanced leverage if projection is available
		advancedScore := LeverageScore{
			PlayerID:      playerID,
			Ownership:     ownershipPct,
			LeverageScore: basicLeverage,
		}
		
		if hasProjection {
			advancedScore = lc.calculateAdvancedLeverage(ownershipPct, projection, totalEntries)
		}
		
		// Determine opportunity type and risk level
		advancedScore.OpportunityType = lc.getOpportunityType(ownershipPct)
		advancedScore.RiskLevel = lc.getRiskLevel(advancedScore)
		advancedScore.RecommendedExposure = lc.getRecommendedExposure(advancedScore)
		
		scores = append(scores, advancedScore)
	}
	
	// Sort by leverage score (highest first)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].LeverageScore > scores[j].LeverageScore
	})
	
	return scores
}

// calculateBasicLeverage calculates basic leverage score based on ownership inversion
func (lc *LeverageCalculator) calculateBasicLeverage(ownershipPct float64, totalEntries int) float64 {
	// Avoid division by zero
	if ownershipPct <= 0 {
		return 100.0 // Maximum leverage for 0% owned players
	}
	
	// Invert ownership percentage to get basic leverage
	// Lower ownership = higher leverage
	basicLeverage := (1.0 / ownershipPct) * 100.0
	
	// Apply logarithmic scaling to prevent extreme values
	if basicLeverage > 100 {
		basicLeverage = 100 + math.Log10(basicLeverage-100+1)*10
	}
	
	// Cap at maximum value
	if basicLeverage > 200 {
		basicLeverage = 200
	}
	
	// Adjust for contest size (smaller contests have more leverage opportunity)
	sizeAdjustment := 1.0
	if totalEntries > 0 {
		// Higher leverage in smaller contests
		sizeAdjustment = math.Max(0.5, 1.0-(float64(totalEntries)/10000.0)*0.3)
	}
	
	return basicLeverage * sizeAdjustment
}

// calculateAdvancedLeverage calculates leverage with projection and volatility considerations
func (lc *LeverageCalculator) calculateAdvancedLeverage(
	ownershipPct float64,
	projection PlayerProjection,
	totalEntries int,
) LeverageScore {
	
	// Start with basic leverage
	basicLeverage := lc.calculateBasicLeverage(ownershipPct, totalEntries)
	
	// Calculate value-based adjustment
	valuePerDollar := 0.0
	if projection.Salary > 0 {
		valuePerDollar = projection.ProjectedPoints / float64(projection.Salary) * 1000
	}
	
	// Projection vs ownership gap
	// This would typically use actual projections vs implied projections from ownership
	// For now, we'll use a simplified calculation
	projectionGap := lc.calculateProjectionGap(ownershipPct, projection)
	
	// Volatility adjustment (higher volatility = higher leverage potential)
	volatilityMultiplier := 1.0 + (projection.ProjectionStdDev / projection.ProjectedPoints * 0.5)
	
	// Stack potential bonus
	stackMultiplier := 1.0
	if projection.IsStackCandidate {
		stackMultiplier = lc.stackBonus
	}
	
	// Risk adjustment (injury risk reduces leverage)
	riskAdjustment := 1.0 - (projection.InjuryRisk * 0.3)
	
	// Combine all factors
	adjustedLeverage := basicLeverage * volatilityMultiplier * stackMultiplier * riskAdjustment
	
	// Position-specific adjustments
	positionMultiplier := lc.getPositionMultiplier(projection.Position)
	adjustedLeverage *= positionMultiplier
	
	// Game environment adjustment
	environmentMultiplier := lc.getEnvironmentMultiplier(projection.GameEnvironment)
	adjustedLeverage *= environmentMultiplier
	
	// Cap final score
	if adjustedLeverage > 250 {
		adjustedLeverage = 250
	}
	
	return LeverageScore{
		PlayerID:           projection.PlayerID,
		Ownership:          ownershipPct,
		LeverageScore:      adjustedLeverage,
		StackPotential:     (stackMultiplier - 1.0) * 100, // Convert to percentage
		ProjectionGap:      projectionGap,
		VolatilityAdjusted: adjustedLeverage * volatilityMultiplier,
	}
}

// calculateProjectionGap calculates the gap between projections and ownership
func (lc *LeverageCalculator) calculateProjectionGap(ownershipPct float64, projection PlayerProjection) float64 {
	// Simplified calculation - in practice this would compare actual projections
	// to market-implied projections based on ownership
	
	// Estimate market-implied projection based on ownership
	// This is a simplified model - real implementation would be more sophisticated
	impliedProjection := ownershipPct * 0.5 // Rough approximation
	
	// Calculate gap
	gap := projection.ProjectedPoints - impliedProjection
	
	// Normalize gap to percentage
	if projection.ProjectedPoints > 0 {
		return (gap / projection.ProjectedPoints) * 100
	}
	
	return 0
}

// getOpportunityType determines the type of opportunity based on ownership
func (lc *LeverageCalculator) getOpportunityType(ownershipPct float64) string {
	if ownershipPct < lc.lowOwnershipThreshold {
		return "contrarian"
	} else if ownershipPct < lc.highOwnershipThreshold {
		return "balanced"
	} else if ownershipPct < 40 {
		return "popular"
	} else {
		return "chalk"
	}
}

// getRiskLevel determines the risk level of a leverage play
func (lc *LeverageCalculator) getRiskLevel(score LeverageScore) string {
	// Higher leverage generally means higher risk
	if score.LeverageScore < 50 {
		return "low"
	} else if score.LeverageScore < 100 {
		return "medium"
	} else {
		return "high"
	}
}

// getRecommendedExposure calculates recommended exposure percentage for lineups
func (lc *LeverageCalculator) getRecommendedExposure(score LeverageScore) float64 {
	// Base exposure on leverage score and opportunity type
	baseExposure := 0.0
	
	switch score.OpportunityType {
	case "contrarian":
		// High leverage, low ownership - can take higher exposure
		baseExposure = math.Min(score.LeverageScore*0.3, 40.0)
	case "balanced":
		// Medium leverage - moderate exposure
		baseExposure = math.Min(score.LeverageScore*0.2, 25.0)
	case "popular":
		// Lower leverage - conservative exposure
		baseExposure = math.Min(score.LeverageScore*0.1, 15.0)
	case "chalk":
		// Very low leverage - minimal exposure unless projection gap is large
		baseExposure = math.Min(score.LeverageScore*0.05, 8.0)
	}
	
	// Adjust for projection gap
	if score.ProjectionGap > 10 {
		baseExposure *= 1.5 // Increase exposure for strong projection edge
	} else if score.ProjectionGap < -10 {
		baseExposure *= 0.5 // Decrease exposure for negative projection edge
	}
	
	// Cap exposure
	if baseExposure > 50 {
		baseExposure = 50
	}
	
	return baseExposure
}

// getPositionMultiplier returns position-specific leverage multipliers
func (lc *LeverageCalculator) getPositionMultiplier(position string) float64 {
	// Different positions have different leverage characteristics
	switch position {
	case "QB":
		return 1.2 // QBs have high leverage due to correlation effects
	case "RB":
		return 1.0 // RBs are baseline
	case "WR", "TE":
		return 1.1 // WRs/TEs have good leverage potential
	case "K":
		return 0.8 // Kickers have lower leverage potential
	case "DST", "DEF":
		return 1.3 // Defenses can have very high leverage
	default:
		return 1.0
	}
}

// getEnvironmentMultiplier returns environment-specific multipliers
func (lc *LeverageCalculator) getEnvironmentMultiplier(environment string) float64 {
	switch environment {
	case "dome":
		return 1.0 // Stable conditions
	case "outdoor":
		return 1.05 // Slightly more volatile
	case "weather":
		return 1.15 // Weather games have higher volatility and leverage
	case "wind":
		return 1.2 // High wind increases leverage potential
	case "rain":
		return 1.1 // Rain affects passing games
	default:
		return 1.0
	}
}

// GetTopLeveragePlays returns the top leverage opportunities
func (lc *LeverageCalculator) GetTopLeveragePlays(scores []LeverageScore, count int) []LeverageScore {
	// Sort by leverage score
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].LeverageScore > scores[j].LeverageScore
	})
	
	if len(scores) < count {
		return scores
	}
	
	return scores[:count]
}

// GetContrarianPlays returns the best contrarian opportunities
func (lc *LeverageCalculator) GetContrarianPlays(scores []LeverageScore, maxOwnership float64) []LeverageScore {
	contrarian := make([]LeverageScore, 0)
	
	for _, score := range scores {
		if score.Ownership <= maxOwnership && score.LeverageScore > 75 {
			contrarian = append(contrarian, score)
		}
	}
	
	// Sort by leverage score
	sort.Slice(contrarian, func(i, j int) bool {
		return contrarian[i].LeverageScore > contrarian[j].LeverageScore
	})
	
	return contrarian
}

// CalculateStackLeverage calculates leverage for player stacks
func (lc *LeverageCalculator) CalculateStackLeverage(
	stackPlayers []uint,
	ownership map[uint]float64,
	projections map[uint]PlayerProjection,
) float64 {
	
	if len(stackPlayers) == 0 {
		return 0
	}
	
	// Calculate combined ownership for the stack
	combinedOwnership := 1.0
	for _, playerID := range stackPlayers {
		if playerOwnership, exists := ownership[playerID]; exists {
			// Convert percentage to decimal for multiplication
			combinedOwnership *= (playerOwnership / 100.0)
		}
	}
	
	// Convert back to percentage
	combinedOwnership *= 100.0
	
	// Stack leverage is inversely related to combined ownership
	stackLeverage := lc.calculateBasicLeverage(combinedOwnership, 1000) // Use fixed contest size
	
	// Apply stack bonus
	stackLeverage *= lc.stackBonus
	
	// Correlation bonus (stacks have correlation benefits)
	correlationBonus := 1.0 + (float64(len(stackPlayers)) * 0.1)
	stackLeverage *= correlationBonus
	
	return stackLeverage
}

// GetOwnershipTier returns the ownership tier for a given ownership percentage
func (lc *LeverageCalculator) GetOwnershipTier(ownershipPct float64) OwnershipTier {
	if ownershipPct < lc.lowOwnershipThreshold {
		return TierLowOwned
	} else if ownershipPct < lc.highOwnershipThreshold {
		return TierMediumOwned
	} else if ownershipPct < 40 {
		return TierHighOwned
	} else {
		return TierChalk
	}
}

// CalculateLeverageDistribution returns the distribution of leverage across ownership tiers
func (lc *LeverageCalculator) CalculateLeverageDistribution(scores []LeverageScore) map[OwnershipTier]float64 {
	distribution := map[OwnershipTier]float64{
		TierLowOwned:    0,
		TierMediumOwned: 0,
		TierHighOwned:   0,
		TierChalk:       0,
	}
	
	tierCounts := map[OwnershipTier]int{
		TierLowOwned:    0,
		TierMediumOwned: 0,
		TierHighOwned:   0,
		TierChalk:       0,
	}
	
	for _, score := range scores {
		tier := lc.GetOwnershipTier(score.Ownership)
		distribution[tier] += score.LeverageScore
		tierCounts[tier]++
	}
	
	// Calculate averages
	for tier := range distribution {
		if tierCounts[tier] > 0 {
			distribution[tier] /= float64(tierCounts[tier])
		}
	}
	
	return distribution
}