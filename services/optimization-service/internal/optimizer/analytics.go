package optimizer

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/stitts-dev/dfs-sim/shared/types"
	"github.com/sirupsen/logrus"
)


// PlayerAnalytics represents enhanced analytics for a player
type PlayerAnalytics struct {
	PlayerID            uint    `json:"player_id"`
	BaseProjection      float64 `json:"base_projection"`
	Ceiling             float64 `json:"ceiling"`             // 85th percentile performance
	Floor               float64 `json:"floor"`               // 15th percentile performance
	Volatility          float64 `json:"volatility"`          // Coefficient of variation
	ValueRating         float64 `json:"value_rating"`        // Points per dollar
	OwnershipProjection float64 `json:"ownership_projection"`
	CeilingProbability  float64 `json:"ceiling_probability"` // Probability of hitting ceiling
	FloorProbability    float64 `json:"floor_probability"`   // Probability of hitting floor
	ConsistencyScore    float64 `json:"consistency_score"`   // 1 / volatility
	UpsideRatio         float64 `json:"upside_ratio"`        // Ceiling / base projection
	DownsideRisk        float64 `json:"downside_risk"`       // Base - floor
	SafetyScore         float64 `json:"safety_score"`        // Floor / base projection
	TournamentScore     float64 `json:"tournament_score"`    // Ceiling adjusted for ownership
	CashGameScore       float64 `json:"cash_game_score"`     // Floor adjusted for consistency
}

// PerformanceData represents historical performance data
type PerformanceData struct {
	Points    float64   `json:"points"`
	Salary    int       `json:"salary"`
	Ownership float64   `json:"ownership"`
	GameDate  time.Time `json:"game_date"`
}

// AnalyticsEngine calculates advanced player analytics
type AnalyticsEngine struct {
	logger *logrus.Entry
}

// NewAnalyticsEngine creates a new analytics engine
func NewAnalyticsEngine() *AnalyticsEngine {
	return &AnalyticsEngine{
		logger: logrus.WithField("component", "analytics_engine"),
	}
}

// CalculatePlayerAnalytics computes comprehensive analytics for a player
func (a *AnalyticsEngine) CalculatePlayerAnalytics(player types.Player, historicalData []PerformanceData) (*PlayerAnalytics, error) {
	// Input validation first (existing pattern in optimizer)
	if player.ProjectedPoints <= 0 {
		a.logger.WithField("player_id", player.ID).Warn("Player has invalid projected points")
		return nil, fmt.Errorf("player %s has invalid projected points: %f", player.Name, player.ProjectedPoints)
	}
	
	// Use platform-specific salary based on existing pattern
	salary := player.SalaryDK
	if salary <= 0 {
		salary = player.SalaryFD
	}
	if salary <= 0 {
		a.logger.WithField("player_id", player.ID).Warn("Player has invalid salary")
		return nil, fmt.Errorf("player %s has invalid salary", player.Name)
	}

	analytics := &PlayerAnalytics{
		PlayerID:       uint(player.ID.ID()), // Convert UUID to uint for compatibility
		BaseProjection: player.ProjectedPoints,
	}

	// If we have floor/ceiling from data source, use them as base
	if player.FloorPoints > 0 && player.CeilingPoints > 0 {
		analytics.Floor = player.FloorPoints
		analytics.Ceiling = player.CeilingPoints
	} else {
		// Calculate from historical data or use defaults
		analytics.Floor, analytics.Ceiling = a.calculateFloorCeiling(player.ProjectedPoints, historicalData)
	}

	// Calculate volatility using coefficient of variation
	analytics.Volatility = a.calculateVolatility(player.ProjectedPoints, historicalData)
	
	// Value rating: points per thousand dollars (industry standard)
	analytics.ValueRating = (player.ProjectedPoints / float64(salary)) * 1000

	// Ownership projection (use existing data or estimate)
	if player.OwnershipDK > 0 {
		analytics.OwnershipProjection = player.OwnershipDK
	} else if player.OwnershipFD > 0 {
		analytics.OwnershipProjection = player.OwnershipFD
	} else {
		analytics.OwnershipProjection = a.estimateOwnership(analytics.ValueRating)
	}

	// Advanced derived metrics
	analytics.CeilingProbability = a.calculateCeilingProbability(analytics.Ceiling, analytics.BaseProjection, analytics.Volatility)
	analytics.FloorProbability = a.calculateFloorProbability(analytics.Floor, analytics.BaseProjection, analytics.Volatility)
	analytics.ConsistencyScore = a.calculateConsistencyScore(analytics.Volatility)
	analytics.UpsideRatio = analytics.Ceiling / analytics.BaseProjection
	analytics.DownsideRisk = analytics.BaseProjection - analytics.Floor
	analytics.SafetyScore = analytics.Floor / analytics.BaseProjection
	
	// Tournament vs Cash Game scoring
	analytics.TournamentScore = a.calculateTournamentScore(analytics)
	analytics.CashGameScore = a.calculateCashGameScore(analytics)

	a.logger.WithFields(logrus.Fields{
		"player_id":       player.ID,
		"player_name":     player.Name,
		"value_rating":    analytics.ValueRating,
		"volatility":      analytics.Volatility,
		"ceiling":         analytics.Ceiling,
		"floor":           analytics.Floor,
		"tournament_score": analytics.TournamentScore,
		"cash_score":      analytics.CashGameScore,
	}).Debug("Calculated player analytics")

	return analytics, nil
}

// CalculateBulkAnalytics processes analytics for multiple players efficiently
func (a *AnalyticsEngine) CalculateBulkAnalytics(players []types.Player, historicalDataMap map[uint][]PerformanceData) (map[uint]*PlayerAnalytics, error) {
	results := make(map[uint]*PlayerAnalytics)
	errors := make([]string, 0)

	start := time.Now()
	a.logger.WithField("player_count", len(players)).Info("Starting bulk analytics calculation")

	for _, player := range players {
		playerID := uint(player.ID.ID())
		historicalData := historicalDataMap[playerID]
		
		analytics, err := a.CalculatePlayerAnalytics(player, historicalData)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Player %s: %v", player.Name, err))
			continue
		}
		
		results[playerID] = analytics
	}

	duration := time.Since(start)
	a.logger.WithFields(logrus.Fields{
		"total_players":     len(players),
		"successful_count":  len(results),
		"error_count":       len(errors),
		"processing_time":   duration,
		"avg_time_per_player": duration / time.Duration(len(players)),
	}).Info("Bulk analytics calculation completed")

	if len(errors) > 0 {
		a.logger.WithField("errors", errors).Warn("Some players had analytics calculation errors")
	}

	return results, nil
}

// calculateFloorCeiling calculates floor and ceiling based on historical data or projections
func (a *AnalyticsEngine) calculateFloorCeiling(baseProjection float64, historicalData []PerformanceData) (floor, ceiling float64) {
	if len(historicalData) == 0 {
		// Default estimation based on position volatility patterns
		return a.estimateFloorCeilingFromProjection(baseProjection)
	}

	// Extract points from historical data
	points := make([]float64, len(historicalData))
	for i, data := range historicalData {
		points[i] = data.Points
	}

	// Sort for percentile calculation
	sort.Float64s(points)

	// Floor = 15th percentile (85% chance of scoring at least this much)
	floor = a.calculatePercentile(points, 0.15)
	
	// Ceiling = 85th percentile (15% chance of scoring at least this much)
	ceiling = a.calculatePercentile(points, 0.85)

	// Ensure reasonable bounds relative to base projection
	if floor > baseProjection {
		floor = baseProjection * 0.75 // Floor shouldn't exceed projection
	}
	if ceiling < baseProjection {
		ceiling = baseProjection * 1.25 // Ceiling should exceed projection
	}

	return floor, ceiling
}

// estimateFloorCeilingFromProjection provides default floor/ceiling when no historical data
func (a *AnalyticsEngine) estimateFloorCeilingFromProjection(baseProjection float64) (floor, ceiling float64) {
	// Default volatility multipliers based on DFS research
	// Higher projected players tend to be more consistent
	volatilityFactor := 0.35
	if baseProjection > 40 {
		volatilityFactor = 0.25 // Stars are more consistent
	} else if baseProjection < 20 {
		volatilityFactor = 0.45 // Value plays are more volatile
	}

	floor = baseProjection * (1.0 - volatilityFactor)
	ceiling = baseProjection * (1.0 + volatilityFactor)

	return floor, ceiling
}

// calculateVolatility computes coefficient of variation from historical data
func (a *AnalyticsEngine) calculateVolatility(baseProjection float64, historicalData []PerformanceData) float64 {
	if len(historicalData) < 2 {
		// Default volatility estimation based on projection level
		if baseProjection > 40 {
			return 0.25 // High-projection players more consistent
		} else if baseProjection < 20 {
			return 0.45 // Low-projection players more volatile
		}
		return 0.35 // Default volatility
	}

	// Calculate mean and variance from historical data
	points := make([]float64, len(historicalData))
	sum := 0.0
	for i, data := range historicalData {
		points[i] = data.Points
		sum += data.Points
	}
	
	mean := sum / float64(len(points))
	
	// Calculate variance
	variance := 0.0
	for _, point := range points {
		variance += math.Pow(point-mean, 2)
	}
	variance = variance / float64(len(points)-1)
	
	stdDev := math.Sqrt(variance)
	
	// Coefficient of Variation = std dev / mean
	if mean <= 0 {
		return 0.35 // Default if calculation fails
	}
	
	cv := stdDev / mean
	
	// Cap extreme values
	if cv > 1.0 {
		cv = 1.0
	} else if cv < 0.1 {
		cv = 0.1
	}
	
	return cv
}

// calculatePercentile calculates the specified percentile from sorted data
func (a *AnalyticsEngine) calculatePercentile(sortedData []float64, percentile float64) float64 {
	if len(sortedData) == 0 {
		return 0
	}
	
	if len(sortedData) == 1 {
		return sortedData[0]
	}
	
	// Calculate index for percentile
	index := percentile * float64(len(sortedData)-1)
	lower := int(index)
	upper := lower + 1
	
	if upper >= len(sortedData) {
		return sortedData[len(sortedData)-1]
	}
	
	if lower < 0 {
		return sortedData[0]
	}
	
	// Linear interpolation between the two closest values
	weight := index - float64(lower)
	return sortedData[lower]*(1-weight) + sortedData[upper]*weight
}

// estimateOwnership estimates ownership based on value rating
func (a *AnalyticsEngine) estimateOwnership(valueRating float64) float64 {
	// Basic ownership estimation model
	// Higher value = higher ownership typically
	if valueRating > 6.0 {
		return 25.0 + math.Min(20.0, (valueRating-6.0)*5) // 25-45% for high value
	} else if valueRating > 4.5 {
		return 15.0 + (valueRating-4.5)*6.67 // 15-25% for medium value
	} else {
		return 5.0 + valueRating*2.22 // 5-15% for low value
	}
}

// calculateCeilingProbability estimates probability of hitting ceiling
func (a *AnalyticsEngine) calculateCeilingProbability(ceiling, base, volatility float64) float64 {
	if volatility <= 0 {
		return 0.15 // Default 15% ceiling probability
	}
	
	// Use normal distribution assumption
	// Z-score for how many standard deviations ceiling is from base
	stdDev := base * volatility
	if stdDev <= 0 {
		return 0.15
	}
	
	zScore := (ceiling - base) / stdDev
	
	// Convert to probability (rough approximation)
	prob := 0.5 * math.Erfc(zScore/math.Sqrt(2))
	
	// Cap between 5% and 35%
	if prob > 0.35 {
		prob = 0.35
	} else if prob < 0.05 {
		prob = 0.05
	}
	
	return prob
}

// calculateFloorProbability estimates probability of hitting floor or worse
func (a *AnalyticsEngine) calculateFloorProbability(floor, base, volatility float64) float64 {
	if volatility <= 0 {
		return 0.15 // Default 15% floor probability
	}
	
	stdDev := base * volatility
	if stdDev <= 0 {
		return 0.15
	}
	
	zScore := (floor - base) / stdDev
	
	// Probability of scoring floor or lower
	prob := 0.5 * math.Erfc(-zScore/math.Sqrt(2))
	
	// Cap between 5% and 35%
	if prob > 0.35 {
		prob = 0.35
	} else if prob < 0.05 {
		prob = 0.05
	}
	
	return prob
}

// calculateConsistencyScore converts volatility to consistency (inverse relationship)
func (a *AnalyticsEngine) calculateConsistencyScore(volatility float64) float64 {
	if volatility <= 0 {
		return 1.0
	}
	
	// Consistency = 1 / (1 + volatility)
	// This gives a score from 0-1 where higher = more consistent
	return 1.0 / (1.0 + volatility)
}

// calculateTournamentScore calculates a score optimized for tournament play
func (a *AnalyticsEngine) calculateTournamentScore(analytics *PlayerAnalytics) float64 {
	// Tournament score emphasizes ceiling and contrarian value
	ceilingScore := analytics.Ceiling * 0.6
	
	// Ownership penalty for chalk plays
	ownershipPenalty := math.Pow(analytics.OwnershipProjection/100, 1.5) * 5
	
	// Upside bonus
	upsideBonus := (analytics.UpsideRatio - 1.0) * 10
	
	return ceilingScore - ownershipPenalty + upsideBonus
}

// calculateCashGameScore calculates a score optimized for cash games
func (a *AnalyticsEngine) calculateCashGameScore(analytics *PlayerAnalytics) float64 {
	// Cash game score emphasizes floor and consistency
	floorScore := analytics.Floor * 0.7
	
	// Consistency bonus
	consistencyBonus := analytics.ConsistencyScore * 8
	
	// Safety bonus
	safetyBonus := analytics.SafetyScore * 5
	
	return floorScore + consistencyBonus + safetyBonus
}

// EnhancePlayersWithAnalytics adds analytics to existing player data
func (a *AnalyticsEngine) EnhancePlayersWithAnalytics(players []types.Player, analytics map[uint]*PlayerAnalytics) []EnhancedPlayer {
	enhanced := make([]EnhancedPlayer, 0, len(players))
	
	for _, player := range players {
		playerID := uint(player.ID.ID())
		
		enhancedPlayer := EnhancedPlayer{
			Player:    player,
			Analytics: analytics[playerID],
		}
		
		enhanced = append(enhanced, enhancedPlayer)
	}
	
	return enhanced
}

// EnhancedPlayer combines player data with analytics
type EnhancedPlayer struct {
	Player    types.Player    `json:"player"`
	Analytics *PlayerAnalytics `json:"analytics,omitempty"`
}

// GetObjectiveScore returns the score for a specific optimization objective
func (a *AnalyticsEngine) GetObjectiveScore(player EnhancedPlayer, objective OptimizationObjective) float64 {
	if player.Analytics == nil {
		// Fallback to base projection if no analytics
		return player.Player.ProjectedPoints
	}
	
	switch objective {
	case MaxCeiling:
		return player.Analytics.TournamentScore
	case MaxFloor:
		return player.Analytics.CashGameScore
	case Balanced:
		return (player.Analytics.TournamentScore + player.Analytics.CashGameScore) / 2.0
	case Contrarian:
		// Emphasize low ownership
		ownershipPenalty := math.Pow(player.Analytics.OwnershipProjection/100, 2) * 15
		return player.Analytics.BaseProjection - ownershipPenalty
	case Correlation:
		// Will be enhanced by correlation matrix in actual optimization
		return player.Analytics.BaseProjection
	case Value:
		return player.Analytics.ValueRating
	default:
		return player.Analytics.BaseProjection
	}
}