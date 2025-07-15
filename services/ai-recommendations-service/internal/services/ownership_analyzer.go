package services

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/models"
	"gorm.io/gorm"
)

// OwnershipAnalyzer provides ownership intelligence and leverage calculations
type OwnershipAnalyzer struct {
	db             *gorm.DB
	cache          *CacheService
	logger         *logrus.Logger
	leverageCalc   *LeverageCalculator
	patternEngine  *PatternEngine
	liveTracker    *LiveOwnershipTracker
}

// LeverageCalculator computes contrarian opportunities
type LeverageCalculator struct {
	logger *logrus.Logger
}

// PatternEngine analyzes historical ownership patterns
type PatternEngine struct {
	db     *gorm.DB
	logger *logrus.Logger
}

// LiveOwnershipTracker monitors real-time ownership changes
type LiveOwnershipTracker struct {
	cache  *CacheService
	logger *logrus.Logger
}

// LeveragePlay represents a contrarian opportunity
type LeveragePlay struct {
	PlayerID            uint    `json:"player_id"`
	PlayerName          string  `json:"player_name"`
	Position            string  `json:"position"`
	Salary              float64 `json:"salary"`
	Projection          float64 `json:"projection"`
	CurrentOwnership    float64 `json:"current_ownership"`
	ProjectedOwnership  float64 `json:"projected_ownership"`
	LeverageScore       float64 `json:"leverage_score"`
	LeverageType        string  `json:"leverage_type"` // "contrarian", "pivot", "fade"
	OpportunityRating   float64 `json:"opportunity_rating"`
	RiskRating          float64 `json:"risk_rating"`
	ValueRating         float64 `json:"value_rating"`
	ConfidenceLevel     float64 `json:"confidence_level"`
	ReasoningFactors    []string `json:"reasoning_factors"`
	ExpectedROI         float64 `json:"expected_roi"`
	OptimalExposure     float64 `json:"optimal_exposure"`
}

// OwnershipPattern represents historical ownership behavior
type OwnershipPattern struct {
	PlayerType      string    `json:"player_type"`
	SalaryRange     string    `json:"salary_range"`
	ContestType     string    `json:"contest_type"`
	AvgOwnership    float64   `json:"avg_ownership"`
	OwnershipStdDev float64   `json:"ownership_stddev"`
	SuccessRate     float64   `json:"success_rate"`
	AvgROI          float64   `json:"avg_roi"`
	SampleSize      int       `json:"sample_size"`
	LastUpdated     time.Time `json:"last_updated"`
}

// OwnershipTrend represents ownership movement over time
type OwnershipTrend struct {
	PlayerID        uint      `json:"player_id"`
	Timestamps      []time.Time `json:"timestamps"`
	OwnershipValues []float64 `json:"ownership_values"`
	TrendDirection  string    `json:"trend_direction"` // "rising", "falling", "stable", "volatile"
	TrendStrength   float64   `json:"trend_strength"`  // 0-1
	Velocity        float64   `json:"velocity"`        // % change per hour
	Acceleration    float64   `json:"acceleration"`    // change in velocity
}

// StackOwnership represents ownership correlation between players
type StackOwnership struct {
	StackType       string             `json:"stack_type"` // "game", "team", "positional"
	PlayerIDs       []uint             `json:"player_ids"`
	StackOwnership  float64            `json:"stack_ownership"`
	IndividualOwnership map[uint]float64 `json:"individual_ownership"`
	CorrelationScore float64           `json:"correlation_score"`
	LeverageOpportunity float64        `json:"leverage_opportunity"`
}

// NewOwnershipAnalyzer creates a new ownership analyzer
func NewOwnershipAnalyzer(db *gorm.DB, cache *CacheService, logger *logrus.Logger) *OwnershipAnalyzer {
	return &OwnershipAnalyzer{
		db:            db,
		cache:         cache,
		logger:        logger,
		leverageCalc:  NewLeverageCalculator(logger),
		patternEngine: NewPatternEngine(db, logger),
		liveTracker:   NewLiveOwnershipTracker(cache, logger),
	}
}

// CalculateLeverageOpportunities identifies contrarian plays
func (oa *OwnershipAnalyzer) CalculateLeverageOpportunities(
	contestID uint,
	players []models.PlayerRecommendation,
	contestType string,
	existingLineups []models.LineupReference,
) ([]LeveragePlay, error) {
	
	oa.logger.WithField("contest_id", contestID).Debug("Calculating leverage opportunities")

	var leveragePlays []LeveragePlay

	// Get current ownership data
	ownershipData, err := oa.getCurrentOwnership(contestID)
	if err != nil {
		oa.logger.WithError(err).Warn("Failed to get current ownership data")
		// Continue with projected ownership
	}

	// Calculate leverage for each player
	for _, player := range players {
		leveragePlay, err := oa.calculatePlayerLeverage(
			player,
			ownershipData,
			contestType,
			existingLineups,
		)
		if err != nil {
			oa.logger.WithError(err).WithField("player_id", player.PlayerID).Warn("Failed to calculate player leverage")
			continue
		}

		if leveragePlay.LeverageScore > 0.3 { // Only include meaningful leverage plays
			leveragePlays = append(leveragePlays, *leveragePlay)
		}
	}

	// Sort by leverage score
	sort.Slice(leveragePlays, func(i, j int) bool {
		return leveragePlays[i].LeverageScore > leveragePlays[j].LeverageScore
	})

	// Limit to top opportunities
	if len(leveragePlays) > 20 {
		leveragePlays = leveragePlays[:20]
	}

	oa.logger.WithFields(logrus.Fields{
		"contest_id": contestID,
		"total_opportunities": len(leveragePlays),
	}).Info("Calculated leverage opportunities")

	return leveragePlays, nil
}

// calculatePlayerLeverage computes leverage metrics for a single player
func (oa *OwnershipAnalyzer) calculatePlayerLeverage(
	player models.PlayerRecommendation,
	ownershipData map[uint]float64,
	contestType string,
	existingLineups []models.LineupReference,
) (*LeveragePlay, error) {

	currentOwnership := ownershipData[player.PlayerID]
	if currentOwnership == 0 {
		currentOwnership = player.Ownership // Use projected if current not available
	}

	// Calculate value rating
	valueRating := oa.calculateValueRating(player.Projection, player.Salary, currentOwnership)
	
	// Calculate leverage score
	leverageScore := oa.leverageCalc.CalculateLeverageScore(
		player.Projection,
		player.Salary,
		currentOwnership,
		contestType,
	)

	// Determine leverage type
	leverageType := oa.determineLeverageType(currentOwnership, player.Projection, player.Salary)

	// Calculate risk rating
	riskRating := oa.calculateRiskRating(player, currentOwnership, contestType)

	// Calculate expected ROI
	expectedROI := oa.calculateExpectedROI(player, currentOwnership, contestType)

	// Calculate optimal exposure
	optimalExposure := oa.calculateOptimalExposure(leverageScore, riskRating, len(existingLineups))

	// Generate reasoning factors
	reasoningFactors := oa.generateReasoningFactors(player, currentOwnership, leverageType, valueRating)

	// Calculate confidence level
	confidenceLevel := oa.calculateConfidenceLevel(player, ownershipData, contestType)

	leveragePlay := &LeveragePlay{
		PlayerID:           player.PlayerID,
		PlayerName:         player.PlayerName,
		Position:           player.Position,
		Salary:             player.Salary,
		Projection:         player.Projection,
		CurrentOwnership:   currentOwnership,
		ProjectedOwnership: player.Ownership,
		LeverageScore:      leverageScore,
		LeverageType:       leverageType,
		OpportunityRating:  oa.calculateOpportunityRating(leverageScore, valueRating, riskRating),
		RiskRating:         riskRating,
		ValueRating:        valueRating,
		ConfidenceLevel:    confidenceLevel,
		ReasoningFactors:   reasoningFactors,
		ExpectedROI:        expectedROI,
		OptimalExposure:    optimalExposure,
	}

	return leveragePlay, nil
}

// GetOwnershipInsights provides comprehensive ownership analysis
func (oa *OwnershipAnalyzer) GetOwnershipInsights(contestID uint) (*models.OwnershipAnalysis, error) {
	// Get current ownership data
	ownershipData, err := oa.getCurrentOwnership(contestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ownership data: %w", err)
	}

	// Get ownership trends
	trends, err := oa.getOwnershipTrends(contestID)
	if err != nil {
		oa.logger.WithError(err).Warn("Failed to get ownership trends")
		trends = []models.OwnershipTrend{}
	}

	// Categorize players by ownership levels
	highOwnership := oa.categorizeHighOwnership(ownershipData)
	lowOwnership := oa.categorizeLowOwnership(ownershipData)
	chalkPlays := oa.identifyChalkPlays(ownershipData)
	contrianPlays := oa.identifyContrianPlays(ownershipData)

	// Calculate stack ownership
	stackOwnership := oa.calculateStackOwnership(contestID, ownershipData)

	analysis := &models.OwnershipAnalysis{
		HighOwnership:   highOwnership,
		LowOwnership:    lowOwnership,
		OwnershipTrends: trends,
		ChalkPlays:      chalkPlays,
		ContrianPlays:   contrianPlays,
		StackOwnership:  stackOwnership,
	}

	return analysis, nil
}

// GetOwnershipTrends analyzes ownership movement over time
func (oa *OwnershipAnalyzer) GetOwnershipTrends(contestID uint, playerIDs []uint) ([]OwnershipTrend, error) {
	var trends []OwnershipTrend

	for _, playerID := range playerIDs {
		trend, err := oa.analyzePlayerOwnershipTrend(contestID, playerID)
		if err != nil {
			oa.logger.WithError(err).WithField("player_id", playerID).Warn("Failed to analyze ownership trend")
			continue
		}
		trends = append(trends, *trend)
	}

	return trends, nil
}

// analyzePlayerOwnershipTrend analyzes ownership movement for a specific player
func (oa *OwnershipAnalyzer) analyzePlayerOwnershipTrend(contestID, playerID uint) (*OwnershipTrend, error) {
	// Get historical ownership snapshots
	snapshots, err := oa.getOwnershipSnapshots(contestID, playerID)
	if err != nil {
		return nil, err
	}

	if len(snapshots) < 3 {
		return &OwnershipTrend{
			PlayerID:       playerID,
			TrendDirection: "insufficient_data",
			TrendStrength:  0,
		}, nil
	}

	// Extract timestamps and values
	var timestamps []time.Time
	var values []float64
	for _, snapshot := range snapshots {
		timestamps = append(timestamps, snapshot.SnapshotTime)
		values = append(values, snapshot.OwnershipPercentage)
	}

	// Calculate trend metrics
	trendDirection := oa.calculateTrendDirection(values)
	trendStrength := oa.calculateTrendStrength(values)
	velocity := oa.calculateVelocity(timestamps, values)
	acceleration := oa.calculateAcceleration(timestamps, values)

	trend := &OwnershipTrend{
		PlayerID:        playerID,
		Timestamps:      timestamps,
		OwnershipValues: values,
		TrendDirection:  trendDirection,
		TrendStrength:   trendStrength,
		Velocity:        velocity,
		Acceleration:    acceleration,
	}

	return trend, nil
}

// Helper methods for calculations

func (oa *OwnershipAnalyzer) calculateValueRating(projection, salary, ownership float64) float64 {
	// Value = (Projection / (Salary/1000)) * (1 + OwnershipPenalty)
	basePPD := projection / (salary / 1000) // Points per dollar

	// Apply ownership penalty (higher ownership = lower value rating)
	ownershipPenalty := ownership / 100 * 0.5 // Max 50% penalty at 100% ownership
	adjustedValue := basePPD * (1 - ownershipPenalty)

	// Normalize to 0-10 scale
	return math.Min(adjustedValue/5*10, 10)
}

func (oa *OwnershipAnalyzer) determineLeverageType(ownership, projection, salary float64) string {
	if ownership < 5 {
		return "contrarian"
	} else if ownership < 15 && projection/(salary/1000) > 3.0 {
		return "pivot"
	} else if ownership > 30 {
		return "fade"
	}
	return "neutral"
}

func (oa *OwnershipAnalyzer) calculateRiskRating(player models.PlayerRecommendation, ownership float64, contestType string) float64 {
	baseRisk := 5.0 // Start with medium risk

	// Adjust based on player attributes
	if len(player.Tags) > 0 {
		for _, tag := range player.Tags {
			switch tag {
			case "injury_risk":
				baseRisk += 2.0
			case "weather_dependent":
				baseRisk += 1.0
			case "consistent":
				baseRisk -= 1.0
			case "volatile":
				baseRisk += 1.5
			}
		}
	}

	// Adjust based on ownership
	if ownership < 5 {
		baseRisk += 1.0 // Low ownership = higher risk
	} else if ownership > 30 {
		baseRisk -= 0.5 // High ownership = crowd validation
	}

	// Adjust based on contest type
	if contestType == "gpp" {
		baseRisk -= 0.5 // GPP allows more risk
	} else if contestType == "cash" {
		baseRisk += 1.0 // Cash games penalize risk
	}

	// Clamp to 0-10 scale
	return math.Max(0, math.Min(10, baseRisk))
}

func (oa *OwnershipAnalyzer) calculateExpectedROI(player models.PlayerRecommendation, ownership float64, contestType string) float64 {
	// Simplified ROI calculation based on value and ownership
	value := player.Projection / (player.Salary / 1000)
	
	baseROI := (value - 3.0) * 0.2 // Assume 3.0 PPD is break-even
	
	// Ownership adjustment
	if contestType == "gpp" {
		// In GPP, lower ownership can mean higher ROI potential
		ownershipBonus := (20 - ownership) / 100 * 0.1
		baseROI += ownershipBonus
	}

	return baseROI
}

func (oa *OwnershipAnalyzer) calculateOptimalExposure(leverageScore, riskRating float64, existingLineups int) float64 {
	// Base exposure on leverage score
	baseExposure := leverageScore * 0.3 // Max 30% exposure for perfect leverage

	// Adjust based on risk
	riskAdjustment := (10 - riskRating) / 10 * 0.1 // Lower risk = higher exposure
	adjustedExposure := baseExposure + riskAdjustment

	// Account for existing lineups
	if existingLineups > 0 {
		diversityFactor := math.Min(1.0, float64(existingLineups)/10.0)
		adjustedExposure *= (1 - diversityFactor*0.3) // Reduce exposure for diversity
	}

	return math.Max(0, math.Min(0.4, adjustedExposure)) // Cap at 40%
}

func (oa *OwnershipAnalyzer) generateReasoningFactors(player models.PlayerRecommendation, ownership float64, leverageType string, valueRating float64) []string {
	var factors []string

	if ownership < 10 {
		factors = append(factors, fmt.Sprintf("Low ownership (%.1f%%) creates leverage opportunity", ownership))
	}

	if valueRating > 7 {
		factors = append(factors, "Excellent value based on projection and salary")
	}

	if leverageType == "contrarian" {
		factors = append(factors, "Strong contrarian play - public likely undervaluing")
	}

	factors = append(factors, fmt.Sprintf("Projected %.1f points at $%.0f salary", player.Projection, player.Salary))

	// Add player-specific factors
	for _, factor := range player.RealTimeFactors {
		factors = append(factors, factor)
	}

	return factors
}

func (oa *OwnershipAnalyzer) calculateConfidenceLevel(player models.PlayerRecommendation, ownershipData map[uint]float64, contestType string) float64 {
	confidence := player.Confidence // Start with player confidence

	// Adjust based on ownership data quality
	if _, hasData := ownershipData[player.PlayerID]; hasData {
		confidence += 0.1 // Boost confidence with real ownership data
	}

	// Adjust based on contest type
	if contestType == "cash" && player.RiskLevel == "low" {
		confidence += 0.1
	} else if contestType == "gpp" && player.RiskLevel == "high" {
		confidence += 0.05
	}

	return math.Min(1.0, confidence)
}

func (oa *OwnershipAnalyzer) calculateOpportunityRating(leverageScore, valueRating, riskRating float64) float64 {
	// Weighted combination of factors
	opportunity := leverageScore*0.4 + valueRating*0.4 + (10-riskRating)*0.2
	return math.Min(10, opportunity)
}

// Data retrieval methods
func (oa *OwnershipAnalyzer) getCurrentOwnership(contestID uint) (map[uint]float64, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("ownership:current:%d", contestID)
	var ownershipData map[uint]float64
	if err := oa.cache.Get(cacheKey, &ownershipData); err == nil {
		return ownershipData, nil
	}

	// Get from database
	var snapshots []models.OwnershipSnapshot
	err := oa.db.Where("contest_id = ?", contestID).
		Order("snapshot_time DESC").
		Limit(1).
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}

	ownershipData = make(map[uint]float64)
	if len(snapshots) > 0 {
		// Parse ownership data from latest snapshot
		// This would depend on how ownership data is stored
	}

	// Cache for short duration
	oa.cache.Set(cacheKey, ownershipData, 2*time.Minute)

	return ownershipData, nil
}

func (oa *OwnershipAnalyzer) getOwnershipSnapshots(contestID, playerID uint) ([]models.OwnershipSnapshot, error) {
	var snapshots []models.OwnershipSnapshot
	err := oa.db.Where("contest_id = ? AND player_id = ?", contestID, playerID).
		Order("snapshot_time ASC").
		Find(&snapshots).Error
	return snapshots, err
}

func (oa *OwnershipAnalyzer) getOwnershipTrends(contestID uint) ([]models.OwnershipTrend, error) {
	// Implementation would retrieve trend data
	return []models.OwnershipTrend{}, nil
}

// Categorization methods
func (oa *OwnershipAnalyzer) categorizeHighOwnership(ownershipData map[uint]float64) []models.PlayerOwnership {
	var highOwnership []models.PlayerOwnership
	for playerID, ownership := range ownershipData {
		if ownership > 20 { // Consider 20%+ as high ownership
			highOwnership = append(highOwnership, models.PlayerOwnership{
				PlayerID:  playerID,
				Ownership: ownership,
			})
		}
	}
	return highOwnership
}

func (oa *OwnershipAnalyzer) categorizeLowOwnership(ownershipData map[uint]float64) []models.PlayerOwnership {
	var lowOwnership []models.PlayerOwnership
	for playerID, ownership := range ownershipData {
		if ownership < 5 { // Consider <5% as low ownership
			lowOwnership = append(lowOwnership, models.PlayerOwnership{
				PlayerID:  playerID,
				Ownership: ownership,
			})
		}
	}
	return lowOwnership
}

func (oa *OwnershipAnalyzer) identifyChalkPlays(ownershipData map[uint]float64) []models.PlayerOwnership {
	var chalkPlays []models.PlayerOwnership
	for playerID, ownership := range ownershipData {
		if ownership > 30 { // Consider 30%+ as chalk
			chalkPlays = append(chalkPlays, models.PlayerOwnership{
				PlayerID:  playerID,
				Ownership: ownership,
			})
		}
	}
	return chalkPlays
}

func (oa *OwnershipAnalyzer) identifyContrianPlays(ownershipData map[uint]float64) []models.PlayerOwnership {
	var contrianPlays []models.PlayerOwnership
	for playerID, ownership := range ownershipData {
		if ownership < 8 { // Consider <8% as contrarian
			contrianPlays = append(contrianPlays, models.PlayerOwnership{
				PlayerID:  playerID,
				Ownership: ownership,
			})
		}
	}
	return contrianPlays
}

func (oa *OwnershipAnalyzer) calculateStackOwnership(contestID uint, ownershipData map[uint]float64) map[string]float64 {
	// Implementation would calculate stack correlations
	return make(map[string]float64)
}

// Trend calculation methods
func (oa *OwnershipAnalyzer) calculateTrendDirection(values []float64) string {
	if len(values) < 2 {
		return "insufficient_data"
	}

	recent := values[len(values)-3:] // Look at last 3 points
	if len(recent) < 3 {
		recent = values
	}

	increasing := 0
	decreasing := 0
	
	for i := 1; i < len(recent); i++ {
		if recent[i] > recent[i-1] {
			increasing++
		} else if recent[i] < recent[i-1] {
			decreasing++
		}
	}

	if increasing > decreasing {
		return "rising"
	} else if decreasing > increasing {
		return "falling"
	}
	return "stable"
}

func (oa *OwnershipAnalyzer) calculateTrendStrength(values []float64) float64 {
	if len(values) < 3 {
		return 0
	}

	// Calculate coefficient of variation as strength measure
	mean := oa.calculateMean(values)
	variance := oa.calculateVariance(values, mean)
	stdDev := math.Sqrt(variance)
	
	if mean == 0 {
		return 0
	}
	
	return math.Min(1.0, stdDev/mean)
}

func (oa *OwnershipAnalyzer) calculateVelocity(timestamps []time.Time, values []float64) float64 {
	if len(timestamps) < 2 {
		return 0
	}

	// Calculate average rate of change per hour
	totalChange := values[len(values)-1] - values[0]
	totalTime := timestamps[len(timestamps)-1].Sub(timestamps[0]).Hours()
	
	if totalTime == 0 {
		return 0
	}
	
	return totalChange / totalTime
}

func (oa *OwnershipAnalyzer) calculateAcceleration(timestamps []time.Time, values []float64) float64 {
	if len(timestamps) < 3 {
		return 0
	}

	// Calculate change in velocity
	velocities := make([]float64, len(values)-1)
	for i := 1; i < len(values); i++ {
		timeChange := timestamps[i].Sub(timestamps[i-1]).Hours()
		if timeChange > 0 {
			velocities[i-1] = (values[i] - values[i-1]) / timeChange
		}
	}

	if len(velocities) < 2 {
		return 0
	}

	return velocities[len(velocities)-1] - velocities[0]
}

func (oa *OwnershipAnalyzer) calculateMean(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (oa *OwnershipAnalyzer) calculateVariance(values []float64, mean float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += math.Pow(v-mean, 2)
	}
	return sum / float64(len(values))
}

// Constructor functions for sub-components
func NewLeverageCalculator(logger *logrus.Logger) *LeverageCalculator {
	return &LeverageCalculator{logger: logger}
}

func (lc *LeverageCalculator) CalculateLeverageScore(projection, salary, ownership float64, contestType string) float64 {
	// Base leverage calculation
	value := projection / (salary / 1000)
	ownershipFactor := math.Max(0.1, (30-ownership)/30) // Higher leverage for lower ownership
	
	leverageScore := value * ownershipFactor * 0.1
	
	// Adjust for contest type
	if contestType == "gpp" {
		leverageScore *= 1.2 // GPP rewards leverage more
	}
	
	return math.Min(1.0, leverageScore)
}

func NewPatternEngine(db *gorm.DB, logger *logrus.Logger) *PatternEngine {
	return &PatternEngine{db: db, logger: logger}
}

func NewLiveOwnershipTracker(cache *CacheService, logger *logrus.Logger) *LiveOwnershipTracker {
	return &LiveOwnershipTracker{cache: cache, logger: logger}
}