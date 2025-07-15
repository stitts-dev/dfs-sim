package optimizer

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/stitts-dev/dfs-sim/shared/types"
)

// CutProbabilityEngine calculates cut probabilities for golf tournaments
type CutProbabilityEngine struct {
	historicalData    *HistoricalCutData
	courseModels      map[string]*CourseCutModel
	weatherService    WeatherServiceInterface
	redisClient       *redis.Client
	cacheTTL          time.Duration
}


// HistoricalCutData represents historical cut line data
type HistoricalCutData struct {
	cutHistory      map[string][]CutEvent
	playerCutRates  map[string]*PlayerCutStats
	lastUpdated     time.Time
}

// CutEvent represents a single cut event in tournament history
type CutEvent struct {
	TournamentID string
	CourseID     string
	CutLine      int
	FieldSize    int
	Weather      *types.WeatherConditions
	Date         time.Time
}

// PlayerCutStats represents a player's historical cut performance
type PlayerCutStats struct {
	PlayerID          string
	TournamentsPlayed int
	CutsMade          int
	CutRate           float64
	RecentForm        []bool    // Last 7 events
	RecentFormRate    float64
	StrokesGainedAvg  float64
	LastUpdated       time.Time
}

// CourseCutModel represents cut prediction model for a specific course
type CourseCutModel struct {
	CourseID        string
	HistoricalCuts  []int
	AverageCutLine  float64
	CutVariance     float64
	WeatherFactor   float64
	FieldStrengthFactor float64
	LastUpdated     time.Time
}

// CutProbabilityResult represents the result of cut probability calculation
type CutProbabilityResult struct {
	PlayerID           string
	TournamentID       string
	BaseCutProb        float64
	CourseCutProb      float64
	WeatherAdjusted    float64
	FinalCutProb       float64
	Confidence         float64
	FieldStrengthAdj   float64
	RecentFormAdj      float64
	CalculatedAt       time.Time
}

// NewCutProbabilityEngine creates a new cut probability engine
func NewCutProbabilityEngine(weatherService WeatherServiceInterface, redisClient *redis.Client) *CutProbabilityEngine {
	engine := &CutProbabilityEngine{
		historicalData: &HistoricalCutData{
			cutHistory:     make(map[string][]CutEvent),
			playerCutRates: make(map[string]*PlayerCutStats),
			lastUpdated:    time.Now(),
		},
		courseModels:   make(map[string]*CourseCutModel),
		weatherService: weatherService,
		redisClient:    redisClient,
		cacheTTL:       4 * time.Hour, // Cache for 4 hours
	}

	// Initialize with some example data - would be loaded from database in production
	engine.initializeHistoricalData()
	
	return engine
}

// CalculateCutProbability calculates the cut probability for a player in a tournament
func (c *CutProbabilityEngine) CalculateCutProbability(ctx context.Context, playerID, tournamentID, courseID string, fieldStrength float64) (*CutProbabilityResult, error) {
	// Validate input
	if err := c.validateInput(playerID, tournamentID, courseID); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Check cache first
	cacheKey := fmt.Sprintf("cutprob:%s:%s", playerID, tournamentID)
	if cached, err := c.getCachedResult(ctx, cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	// Get player historical cut rate
	baseProb, err := c.calculateHistoricalCutRate(playerID)
	if err != nil {
		log.Printf("Warning: Could not calculate historical cut rate for player %s: %v", playerID, err)
		baseProb = 0.70 // Default baseline
	}

	// Course-specific adjustment
	courseProb := baseProb
	if courseModel, exists := c.courseModels[courseID]; exists {
		courseProb = c.adjustForCourse(baseProb, courseModel, fieldStrength)
	}

	// Weather impact (check rate limits)
	weatherProb := courseProb
	weatherImpact := 0.0
	if weather, err := c.weatherService.GetWeatherConditions(ctx, courseID); err == nil {
		impact := c.weatherService.CalculateGolfImpact(weather)
		weatherImpact = c.calculateWeatherCutImpact(impact)
		weatherProb = courseProb * (1.0 + weatherImpact)
	} else {
		log.Printf("Warning: Could not get weather data for course %s: %v", courseID, err)
	}

	// Field strength adjustment
	fieldAdjusted := c.adjustForFieldStrength(weatherProb, fieldStrength)

	// Recent form adjustment
	formAdjusted, formAdj := c.adjustForRecentForm(fieldAdjusted, playerID)

	// Final probability with bounds checking
	finalProb := math.Max(0.05, math.Min(0.98, formAdjusted))

	// Confidence based on data quality
	confidence := c.calculateConfidence(playerID, courseID)

	result := &CutProbabilityResult{
		PlayerID:         playerID,
		TournamentID:     tournamentID,
		BaseCutProb:      baseProb,
		CourseCutProb:    courseProb,
		WeatherAdjusted:  weatherProb,
		FinalCutProb:     finalProb,
		Confidence:       confidence,
		FieldStrengthAdj: fieldStrength,
		RecentFormAdj:    formAdj,
		CalculatedAt:     time.Now(),
	}

	// Cache the result
	if err := c.cacheResult(ctx, cacheKey, result); err != nil {
		log.Printf("Warning: Failed to cache cut probability result: %v", err)
	}

	return result, nil
}

// BatchCalculateCutProbabilities calculates cut probabilities for multiple players
func (c *CutProbabilityEngine) BatchCalculateCutProbabilities(ctx context.Context, playerIDs []string, tournamentID, courseID string, fieldStrength float64) ([]*CutProbabilityResult, error) {
	results := make([]*CutProbabilityResult, 0, len(playerIDs))
	
	for _, playerID := range playerIDs {
		result, err := c.CalculateCutProbability(ctx, playerID, tournamentID, courseID, fieldStrength)
		if err != nil {
			log.Printf("Warning: Failed to calculate cut probability for player %s: %v", playerID, err)
			// Continue with other players
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// validateInput validates the input parameters
func (c *CutProbabilityEngine) validateInput(playerID, tournamentID, courseID string) error {
	if playerID == "" {
		return fmt.Errorf("player ID is required")
	}
	if tournamentID == "" {
		return fmt.Errorf("tournament ID is required")
	}
	if courseID == "" {
		return fmt.Errorf("course ID is required")
	}
	return nil
}

// calculateHistoricalCutRate calculates a player's historical cut rate
func (c *CutProbabilityEngine) calculateHistoricalCutRate(playerID string) (float64, error) {
	playerStats, exists := c.historicalData.playerCutRates[playerID]
	if !exists {
		return 0.70, fmt.Errorf("no historical data found for player %s", playerID)
	}

	if playerStats.TournamentsPlayed == 0 {
		return 0.70, fmt.Errorf("player %s has no tournament history", playerID)
	}

	// Base cut rate from historical data
	baseCutRate := float64(playerStats.CutsMade) / float64(playerStats.TournamentsPlayed)

	// Weight recent form more heavily (last 7 events = 40% weight)
	weightedCutRate := (0.6 * baseCutRate) + (0.4 * playerStats.RecentFormRate)

	return math.Max(0.05, math.Min(0.95, weightedCutRate)), nil
}

// adjustForCourse adjusts cut probability based on course-specific factors
func (c *CutProbabilityEngine) adjustForCourse(baseProb float64, courseModel *CourseCutModel, fieldStrength float64) float64 {
	// Course difficulty adjustment
	courseDifficultyFactor := 1.0
	if courseModel.CutVariance > 2.0 {
		// High variance course (easier cuts)
		courseDifficultyFactor = 1.1
	} else if courseModel.CutVariance < 1.0 {
		// Low variance course (harder cuts)
		courseDifficultyFactor = 0.9
	}

	// Field strength impact
	fieldFactor := 1.0 + (fieldStrength - 1.0) * courseModel.FieldStrengthFactor
	
	return baseProb * courseDifficultyFactor * fieldFactor
}

// calculateWeatherCutImpact calculates weather impact on cut probability
func (c *CutProbabilityEngine) calculateWeatherCutImpact(impact *types.WeatherImpact) float64 {
	if impact == nil {
		return 0.0
	}

	// Wind impact on cut probability (higher wind = lower cut probability)
	windImpact := 0.0
	if impact.WindAdvantage < -0.1 {
		// High wind reduces cut probability by 10-15%
		windImpact = -0.15
	} else if impact.WindAdvantage < -0.05 {
		// Moderate wind reduces cut probability by 5-8%
		windImpact = -0.08
	}

	// Temperature and precipitation impacts are generally smaller
	tempImpact := impact.ScoreImpact * -0.02 // Each stroke impact reduces cut prob by 2%
	
	return windImpact + tempImpact
}

// adjustForFieldStrength adjusts cut probability based on field strength
func (c *CutProbabilityEngine) adjustForFieldStrength(baseProb float64, fieldStrength float64) float64 {
	// Field strength > 1.0 means stronger field, which makes cuts harder
	// Field strength < 1.0 means weaker field, which makes cuts easier
	strengthFactor := 2.0 - fieldStrength // Invert and scale
	adjustment := (strengthFactor - 1.0) * 0.1 // Max 10% adjustment
	
	return baseProb * (1.0 + adjustment)
}

// adjustForRecentForm adjusts cut probability based on recent form
func (c *CutProbabilityEngine) adjustForRecentForm(baseProb float64, playerID string) (float64, float64) {
	playerStats, exists := c.historicalData.playerCutRates[playerID]
	if !exists || len(playerStats.RecentForm) == 0 {
		return baseProb, 0.0
	}

	// Calculate recent form impact
	recentFormImpact := playerStats.RecentFormRate - playerStats.CutRate
	formAdjustment := recentFormImpact * 0.3 // Weight recent form 30%
	
	adjustedProb := baseProb * (1.0 + formAdjustment)
	return adjustedProb, formAdjustment
}

// calculateConfidence calculates confidence score based on data quality
func (c *CutProbabilityEngine) calculateConfidence(playerID, courseID string) float64 {
	confidence := 0.5 // Base confidence
	
	// Player data quality
	if playerStats, exists := c.historicalData.playerCutRates[playerID]; exists {
		if playerStats.TournamentsPlayed >= 20 {
			confidence += 0.3
		} else if playerStats.TournamentsPlayed >= 10 {
			confidence += 0.2
		} else if playerStats.TournamentsPlayed >= 5 {
			confidence += 0.1
		}
	}

	// Course data quality
	if courseModel, exists := c.courseModels[courseID]; exists {
		if len(courseModel.HistoricalCuts) >= 10 {
			confidence += 0.2
		} else if len(courseModel.HistoricalCuts) >= 5 {
			confidence += 0.1
		}
	}

	return math.Min(1.0, confidence)
}

// getCachedResult retrieves cached cut probability result
func (c *CutProbabilityEngine) getCachedResult(ctx context.Context, key string) (*CutProbabilityResult, error) {
	if c.redisClient == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	_, err := c.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	// Parse cached result (simplified - would use proper JSON unmarshaling)
	// For now, return nil to force recalculation
	return nil, fmt.Errorf("cache parsing not implemented")
}

// cacheResult stores cut probability result in cache
func (c *CutProbabilityEngine) cacheResult(ctx context.Context, key string, result *CutProbabilityResult) error {
	if c.redisClient == nil {
		return nil // No caching if Redis not available
	}

	// Store result in cache (simplified - would use proper JSON marshaling)
	// For now, just set a simple key
	return c.redisClient.Set(ctx, key, "cached", c.cacheTTL).Err()
}

// initializeHistoricalData initializes the engine with example historical data
func (c *CutProbabilityEngine) initializeHistoricalData() {
	// Example player data - would be loaded from database
	c.historicalData.playerCutRates["player1"] = &PlayerCutStats{
		PlayerID:          "player1",
		TournamentsPlayed: 25,
		CutsMade:          18,
		CutRate:           0.72,
		RecentForm:        []bool{true, true, false, true, true, true, false},
		RecentFormRate:    0.71, // 5/7
		StrokesGainedAvg:  0.3,
		LastUpdated:       time.Now(),
	}

	// Example course model
	c.courseModels["augusta_national"] = &CourseCutModel{
		CourseID:            "augusta_national",
		HistoricalCuts:      []int{1, 2, 1, 0, 2, 1, 3, 2, 1, 2},
		AverageCutLine:      1.5,
		CutVariance:         1.2,
		WeatherFactor:       0.1,
		FieldStrengthFactor: 0.15,
		LastUpdated:         time.Now(),
	}
}