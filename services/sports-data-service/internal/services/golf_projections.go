package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/models"
	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/providers"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// CutProbabilityEngineInterface defines the interface for cut probability calculations
type CutProbabilityEngineInterface interface {
	CalculateCutProbability(ctx context.Context, playerID, tournamentID, courseID string, fieldStrength float64) (*CutProbabilityResult, error)
	BatchCalculateCutProbabilities(ctx context.Context, playerIDs []string, tournamentID, courseID string, fieldStrength float64) ([]*CutProbabilityResult, error)
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

// GolfProjectionService handles golf-specific player projections
type GolfProjectionService struct {
	db                   *database.DB
	cache                *CacheService
	logger               *logrus.Logger
	golfProvider         *providers.DataGolfClient
	weatherService       *WeatherService
	cutProbabilityEngine CutProbabilityEngineInterface
}

// NewGolfProjectionService creates a new golf projection service
func NewGolfProjectionService(
	db *database.DB,
	cache *CacheService,
	logger *logrus.Logger,
	apiKey string,
) *GolfProjectionService {
	return &GolfProjectionService{
		db:           db,
		cache:        cache,
		logger:       logger,
		golfProvider: providers.NewDataGolfClient(apiKey, db.DB, cache, logger),
	}
}

// SetWeatherService sets the weather service for weather-adjusted projections
func (gps *GolfProjectionService) SetWeatherService(ws *WeatherService) {
	gps.weatherService = ws
}

// SetCutProbabilityEngine sets the cut probability engine for advanced cut predictions
func (gps *GolfProjectionService) SetCutProbabilityEngine(engine CutProbabilityEngineInterface) {
	gps.cutProbabilityEngine = engine
}

// GenerateProjections generates projections for golf players in a tournament
func (gps *GolfProjectionService) GenerateProjections(
	ctx context.Context,
	players []types.PlayerInterface,
	tournamentID string,
) (map[uuid.UUID]*models.GolfProjection, map[uuid.UUID]map[uuid.UUID]float64, error) {
	projections := make(map[uuid.UUID]*models.GolfProjection)

	// Get tournament details
	tournament, err := gps.getTournament(ctx, tournamentID)
	if err != nil {
		return nil, nil, fmt.Errorf("getting tournament: %w", err)
	}

	// Get player entries and course history
	entries, err := gps.getPlayerEntries(ctx, tournament.ID)
	if err != nil {
		gps.logger.Warn("Failed to get player entries", "error", err)
	}

	// Get course history
	courseHistory, err := gps.getCourseHistory(ctx, tournament.CourseID, players)
	if err != nil {
		gps.logger.Warn("Failed to get course history", "error", err)
	}

	// Generate projections for each player
	for _, player := range players {
		playerID := player.GetID()
		entry := entries[playerID]
		history := courseHistory[playerID]

		projection := gps.generatePlayerProjection(player, tournament, entry, history)
		projections[playerID] = projection
	}

	// Generate correlation matrix
	correlations := gps.generateCorrelations(players, entries)

	return projections, correlations, nil
}

// generatePlayerProjection generates projection for a single player
func (gps *GolfProjectionService) generatePlayerProjection(
	player types.PlayerInterface,
	tournament *models.GolfTournament,
	entry *models.GolfPlayerEntry,
	history *models.GolfCourseHistory,
) *models.GolfProjection {
	projection := &models.GolfProjection{
		PlayerID:     player.GetID().String(),
		TournamentID: tournament.ID.String(),
		Confidence:   0.5, // Base confidence
	}

	// Calculate base expected score
	baseScore := gps.calculateBaseScore(player, tournament)
	projection.ExpectedScore = baseScore

	// Adjust for course history if available
	if history != nil && history.RoundsPlayed > 0 {
		courseAdjustment := gps.calculateCourseAdjustment(history, tournament.CoursePar)
		projection.ExpectedScore += courseAdjustment
		projection.Confidence += 0.1 // More confident with course history
	}

	// Adjust for current form if in-tournament
	if entry != nil && len(entry.RoundsScores) > 0 {
		formAdjustment := gps.calculateFormAdjustment(entry)
		projection.ExpectedScore += formAdjustment
		projection.Confidence += 0.1
	}

	// Enhanced weather adjustment using weather service
	if gps.weatherService != nil {
		// Get current weather conditions for the course
		if weather, err := gps.weatherService.GetWeatherConditions(context.Background(), tournament.CourseID); err == nil {
			weatherImpact := gps.weatherService.CalculateGolfImpact(weather)

			// Apply weather impact to expected score
			projection.ExpectedScore += weatherImpact.ScoreImpact

			// Store weather advantage scores
			projection.WeatherAdvantage = weatherImpact.WindAdvantage
			projection.TeeTimeAdvantage = weatherImpact.TeeTimeAdvantage
			projection.WeatherImpactScore = weatherImpact.ScoreImpact

			gps.logger.Debugf("Applied weather impact for player %s: score_impact=%.2f, wind_advantage=%.3f",
				player.GetID(), weatherImpact.ScoreImpact, weatherImpact.WindAdvantage)
		} else {
			gps.logger.Warnf("Failed to get weather conditions for course %s: %v", tournament.CourseID, err)
		}
	}

	// Enhanced cut probability calculation using cut probability engine
	if gps.cutProbabilityEngine != nil {
		if cutResult, err := gps.cutProbabilityEngine.CalculateCutProbability(
			context.Background(),
			player.GetID().String(),
			tournament.ID.String(),
			tournament.CourseID,
			tournament.FieldStrength,
		); err == nil {
			projection.BaseCutProbability = cutResult.BaseCutProb
			projection.CourseCutProbability = cutResult.CourseCutProb
			projection.WeatherAdjustedCut = cutResult.WeatherAdjusted
			projection.FinalCutProbability = cutResult.FinalCutProb
			projection.CutConfidence = cutResult.Confidence

			gps.logger.Debugf("Calculated cut probability for player %s: final=%.3f, confidence=%.3f",
				player.GetID(), cutResult.FinalCutProb, cutResult.Confidence)
		} else {
			gps.logger.Warnf("Failed to calculate cut probability for player %s: %v", player.GetID(), err)
			// Fallback to simple calculation
			projection.FinalCutProbability = gps.calculateSimpleCutProbability(player, tournament)
		}
	} else {
		// Fallback to simple calculation if engine not available
		projection.FinalCutProbability = gps.calculateSimpleCutProbability(player, tournament)
	}

	// Calculate position probabilities based on cut probability and skill level
	projection.Top5Probability = gps.calculatePositionProbability(player, tournament, projection.FinalCutProbability, 5)
	projection.Top10Probability = gps.calculatePositionProbability(player, tournament, projection.FinalCutProbability, 10)
	projection.Top25Probability = gps.calculatePositionProbability(player, tournament, projection.FinalCutProbability, 25)
	projection.WinProbability = gps.calculatePositionProbability(player, tournament, projection.FinalCutProbability, 1)

	// Calculate expected finish position
	projection.ExpectedFinishPosition = gps.calculateExpectedFinish(player, tournament, projection.FinalCutProbability)

	// Calculate strategy-specific scores
	projection.StrategyFitScore = gps.calculateStrategyFit(projection)
	projection.RiskRewardRatio = gps.calculateRiskReward(projection)
	projection.VarianceScore = gps.calculateVarianceScore(player, projection)

	// Calculate DFS points
	projection.DKPoints = gps.calculateDKPoints(projection, tournament)
	projection.FDPoints = gps.calculateFDPoints(projection, tournament)

	// Cap confidence at 0.9
	projection.Confidence = math.Min(projection.Confidence, 0.9)

	return projection
}

// calculateBaseScore calculates base expected 4-round score
func (gps *GolfProjectionService) calculateBaseScore(player types.PlayerInterface, tournament *models.GolfTournament) float64 {
	// Base score relative to par (4 rounds)
	coursePar := float64(tournament.CoursePar * 4) // 4-round total par

	// Use player's projected points as a proxy for skill level
	// Assuming projected points correlate with expected finish position
	skillFactor := (100 - player.GetProjectedPoints()) / 100 // Higher projected points = lower score

	// Base expectation: field average is typically 1-2 over par per round
	fieldAverage := coursePar + 6.0 // +1.5 per round average

	// Adjust based on skill factor
	expectedScore := fieldAverage - (skillFactor * 8) // Top players shoot 8 under field average

	return expectedScore
}

// calculateCourseAdjustment adjusts score based on course history
func (gps *GolfProjectionService) calculateCourseAdjustment(history *models.GolfCourseHistory, coursePar int) float64 {
	if history.RoundsPlayed == 0 {
		return 0
	}

	// Compare player's historical average to par
	historicalDiffFromPar := history.ScoringAvg - float64(coursePar)

	// Weight recent performance more heavily
	recencyWeight := 0.7
	if history.LastPlayed != nil && time.Since(*history.LastPlayed) > 365*24*time.Hour {
		recencyWeight = 0.4 // Older data is less relevant
	}

	// Adjustment is difference from their usual performance
	fieldAvgDiffFromPar := 1.5                                                      // Typical field averages 1.5 over par
	adjustment := (historicalDiffFromPar - fieldAvgDiffFromPar) * 4 * recencyWeight // 4 rounds

	return adjustment
}

// calculateFormAdjustment adjusts based on current tournament performance
func (gps *GolfProjectionService) calculateFormAdjustment(entry *models.GolfPlayerEntry) float64 {
	if len(entry.RoundsScores) == 0 {
		return 0
	}

	// Calculate trend from round scores
	totalScore := int64(0)
	for _, score := range entry.RoundsScores {
		totalScore += score
	}

	// Compare to field position
	positionFactor := float64(entry.CurrentPosition) / 100.0
	if positionFactor < 0.3 {
		// Top 30% of field, playing well
		return -2.0 // Expect 2 strokes better than baseline
	} else if positionFactor > 0.7 {
		// Bottom 30%, struggling
		return 2.0 // Expect 2 strokes worse
	}

	return 0
}

// calculateWeatherImpact calculates score adjustment for weather
func (gps *GolfProjectionService) calculateWeatherImpact(weather models.WeatherConditions) float64 {
	impact := 1.0

	// Wind impact
	if weather.WindSpeed > 20 {
		impact *= 1.05 // 5% score increase in high wind
	} else if weather.WindSpeed > 15 {
		impact *= 1.03
	}

	// Rain/conditions impact
	if weather.Conditions == "rain" || weather.Conditions == "stormy" {
		impact *= 1.04
	}

	// Temperature impact
	if weather.Temperature < 50 || weather.Temperature > 90 {
		impact *= 1.02
	}

	return impact
}

// Probability calculations

func (gps *GolfProjectionService) calculateCutProbability(
	player types.PlayerInterface,
	tournament *models.GolfTournament,
	entry *models.GolfPlayerEntry,
	history *models.GolfCourseHistory,
) float64 {
	baseProbability := 0.5 // Start at 50%

	// Adjust based on player salary (proxy for ranking/skill)
	// Use DraftKings salary as default, fallback to FanDuel
	salary := player.GetSalaryDK()
	if salary == 0 {
		salary = player.GetSalaryFD()
	}
	if salary > 10000 {
		baseProbability += 0.2
	} else if salary > 8000 {
		baseProbability += 0.1
	} else if salary < 6000 {
		baseProbability -= 0.1
	}

	// Adjust based on course history
	if history != nil && history.TournamentsPlayed > 0 {
		cutRate := float64(history.CutsMade) / float64(history.TournamentsPlayed)
		baseProbability = (baseProbability + cutRate) / 2 // Average with historical rate
	}

	// Adjust based on current position if tournament in progress
	if entry != nil && tournament.CurrentRound > 0 {
		if entry.CurrentPosition <= tournament.CutLine {
			baseProbability += 0.2
		} else {
			strokesBehindCut := entry.TotalScore - tournament.CutLine
			baseProbability -= float64(strokesBehindCut) * 0.05
		}
	}

	// Cap between 0.1 and 0.95
	return math.Max(0.1, math.Min(0.95, baseProbability))
}

func (gps *GolfProjectionService) calculateTop10Probability(
	player types.PlayerInterface,
	tournament *models.GolfTournament,
	cutProbability float64,
) float64 {
	if cutProbability < 0.3 {
		return 0.01 // Very unlikely to top 10 if unlikely to make cut
	}

	// Base on salary/skill level
	baseProbability := 0.1 // 10% base chance

	// Use DraftKings salary as default, fallback to FanDuel
	salary := player.GetSalaryDK()
	if salary == 0 {
		salary = player.GetSalaryFD()
	}
	if salary > 11000 {
		baseProbability = 0.25
	} else if salary > 9000 {
		baseProbability = 0.15
	} else if salary < 7000 {
		baseProbability = 0.05
	}

	// Factor in cut probability
	return baseProbability * (cutProbability / 0.5)
}

func (gps *GolfProjectionService) calculateTop25Probability(
	player types.PlayerInterface,
	tournament *models.GolfTournament,
	cutProbability float64,
) float64 {
	// Top 25 is more achievable than top 10
	top10Prob := gps.calculateTop10Probability(player, tournament, cutProbability)
	return math.Min(cutProbability*0.6, top10Prob*2.5)
}

func (gps *GolfProjectionService) calculateWinProbability(
	player types.PlayerInterface,
	tournament *models.GolfTournament,
	top10Probability float64,
) float64 {
	if top10Probability < 0.1 {
		return 0.001
	}

	// Only elite players have realistic win probability
	// Use DraftKings salary as default, fallback to FanDuel
	salary := player.GetSalaryDK()
	if salary == 0 {
		salary = player.GetSalaryFD()
	}
	if salary > 11500 {
		return top10Probability * 0.15
	} else if salary > 10000 {
		return top10Probability * 0.08
	}

	return top10Probability * 0.04
}

// DFS Points Calculations

func (gps *GolfProjectionService) calculateDKPoints(projection *models.GolfProjection, tournament *models.GolfTournament) float64 {
	points := 0.0

	// DraftKings scoring (simplified)
	// Placement points
	if projection.WinProbability > 0.01 {
		points += 30 * projection.WinProbability // 30 pts for win
	}
	if projection.Top10Probability > 0.1 {
		points += 10 * projection.Top10Probability // 10 pts for top 10
	}
	if projection.Top25Probability > 0.2 {
		points += 6 * projection.Top25Probability // 6 pts for top 25
	}

	// Birdie/eagle points (estimated based on expected score)
	expectedBirdies := (float64(tournament.CoursePar*4) - projection.ExpectedScore) * 0.8
	points += expectedBirdies * 3 // 3 pts per birdie

	// Bogey penalty
	expectedBogeys := math.Max(0, (projection.ExpectedScore-float64(tournament.CoursePar*4))*0.3)
	points -= expectedBogeys * 0.5 // -0.5 per bogey

	// Cut penalty
	if projection.CutProbability < 0.5 {
		points *= projection.CutProbability // Heavily penalize likely missed cuts
	}

	return math.Max(0, points)
}

func (gps *GolfProjectionService) calculateFDPoints(projection *models.GolfProjection, tournament *models.GolfTournament) float64 {
	points := 0.0

	// FanDuel scoring (simplified, similar but different values)
	if projection.WinProbability > 0.01 {
		points += 35 * projection.WinProbability
	}
	if projection.Top10Probability > 0.1 {
		points += 12 * projection.Top10Probability
	}
	if projection.Top25Probability > 0.2 {
		points += 8 * projection.Top25Probability
	}

	// Score-based points
	expectedBirdies := (float64(tournament.CoursePar*4) - projection.ExpectedScore) * 0.8
	points += expectedBirdies * 2.5

	expectedBogeys := math.Max(0, (projection.ExpectedScore-float64(tournament.CoursePar*4))*0.3)
	points -= expectedBogeys * 0.5

	if projection.CutProbability < 0.5 {
		points *= projection.CutProbability
	}

	return math.Max(0, points)
}

// Helper methods

func (gps *GolfProjectionService) getTournament(ctx context.Context, tournamentID string) (*models.GolfTournament, error) {
	var tournament models.GolfTournament

	err := gps.db.WithContext(ctx).
		Where("external_id = ? OR id = ?", tournamentID, tournamentID).
		First(&tournament).Error

	if err != nil {
		return nil, fmt.Errorf("tournament not found: %w", err)
	}

	return &tournament, nil
}

func (gps *GolfProjectionService) getPlayerEntries(ctx context.Context, tournamentID interface{}) (map[uuid.UUID]*models.GolfPlayerEntry, error) {
	var entries []models.GolfPlayerEntry

	err := gps.db.WithContext(ctx).
		Where("tournament_id = ?", tournamentID).
		Find(&entries).Error

	if err != nil {
		return nil, err
	}

	entryMap := make(map[uuid.UUID]*models.GolfPlayerEntry)
	for i := range entries {
		entryMap[entries[i].PlayerID] = &entries[i]
	}

	return entryMap, nil
}

func (gps *GolfProjectionService) getCourseHistory(ctx context.Context, courseID string, players []types.PlayerInterface) (map[uuid.UUID]*models.GolfCourseHistory, error) {
	// Extract player IDs using the interface
	playerIDs := make([]uuid.UUID, len(players))
	for i, p := range players {
		playerIDs[i] = p.GetID()
	}

	var histories []models.GolfCourseHistory

	err := gps.db.WithContext(ctx).
		Where("course_id = ? AND player_id IN ?", courseID, playerIDs).
		Find(&histories).Error

	if err != nil {
		return nil, err
	}

	historyMap := make(map[uuid.UUID]*models.GolfCourseHistory)
	for i := range histories {
		historyMap[histories[i].PlayerID] = &histories[i]
	}

	return historyMap, nil
}

func (gps *GolfProjectionService) generateCorrelations(players []types.PlayerInterface, entries map[uuid.UUID]*models.GolfPlayerEntry) map[uuid.UUID]map[uuid.UUID]float64 {
	// TODO: Integrate with shared correlation package once UUID compatibility is resolved
	// Currently using a simplified correlation implementation based on player positions
	// Generate a simple correlation matrix based on tee times and groupings
	correlations := make(map[uuid.UUID]map[uuid.UUID]float64)

	// Initialize correlation matrix
	for _, p1 := range players {
		correlations[p1.GetID()] = make(map[uuid.UUID]float64)
		for _, p2 := range players {
			if p1.GetID() == p2.GetID() {
				correlations[p1.GetID()][p2.GetID()] = 1.0
			} else {
				correlations[p1.GetID()][p2.GetID()] = 0.0
			}
		}
	}

	// Add correlations for players in same groupings (if available in entries)
	// This is a simplified implementation - in production you'd want more sophisticated correlation modeling
	for _, p1 := range players {
		e1, exists1 := entries[p1.GetID()]
		if !exists1 {
			continue
		}

		for _, p2 := range players {
			if p1.GetID() == p2.GetID() {
				continue
			}

			e2, exists2 := entries[p2.GetID()]
			if !exists2 {
				continue
			}

			// TODO: Implement proper tee time correlation using TeeTimes array
			// Currently using position proximity as a proxy for correlation
			// Players in same tournament round/grouping have some correlation
			// For now, we'll use a simplified correlation based on position proximity
			// In a full implementation, you'd parse TeeTimes array and compare
			if e1.CurrentPosition > 0 && e2.CurrentPosition > 0 {
				positionDiff := math.Abs(float64(e1.CurrentPosition - e2.CurrentPosition))
				if positionDiff < 5 { // Close in leaderboard position
					correlations[p1.GetID()][p2.GetID()] = 0.2
				} else if positionDiff < 20 { // Somewhat close
					correlations[p1.GetID()][p2.GetID()] = 0.05
				}
			}
		}
	}

	return correlations
}

// CalculateCutProbability calculates cut probability for external use
func (gps *GolfProjectionService) CalculateCutProbability(ctx context.Context, entry models.GolfPlayerEntry) float64 {
	// This is a simplified version for external use
	// In a real implementation, you'd want to consider more factors

	if entry.Status == models.EntryStatusCut {
		return 0.0
	}

	if entry.Status == models.EntryStatusWithdrawn {
		return 0.0
	}

	// Simple calculation based on current position
	if entry.CurrentPosition == 0 {
		return 0.5 // No position data yet
	}

	if entry.CurrentPosition <= 70 {
		return 0.8
	} else if entry.CurrentPosition <= 100 {
		return 0.5
	}

	return 0.2
}

// calculateSimpleCutProbability provides a fallback cut probability calculation
func (gps *GolfProjectionService) calculateSimpleCutProbability(player types.PlayerInterface, tournament *models.GolfTournament) float64 {
	baseProbability := 0.5 // Start at 50%

	// Adjust based on player salary (proxy for ranking/skill)
	salary := player.GetSalaryDK()
	if salary == 0 {
		salary = player.GetSalaryFD()
	}

	if salary > 10000 {
		baseProbability += 0.2
	} else if salary > 8000 {
		baseProbability += 0.1
	} else if salary < 6000 {
		baseProbability -= 0.1
	}

	// Field strength adjustment
	if tournament.FieldStrength > 1.2 {
		baseProbability -= 0.1 // Stronger field = harder cuts
	} else if tournament.FieldStrength < 0.8 {
		baseProbability += 0.1 // Weaker field = easier cuts
	}

	return math.Max(0.1, math.Min(0.95, baseProbability))
}

// calculatePositionProbability calculates probability of finishing in top N positions
func (gps *GolfProjectionService) calculatePositionProbability(player types.PlayerInterface, tournament *models.GolfTournament, cutProb float64, position int) float64 {
	if cutProb < 0.3 {
		return 0.01 // Very unlikely to achieve good finish if unlikely to make cut
	}

	// Base probability varies by position difficulty
	var baseProbability float64
	switch position {
	case 1: // Win
		baseProbability = 0.02
	case 5: // Top 5
		baseProbability = 0.08
	case 10: // Top 10
		baseProbability = 0.15
	case 25: // Top 25
		baseProbability = 0.35
	default:
		baseProbability = 0.1
	}

	// Adjust based on player salary/skill
	salary := player.GetSalaryDK()
	if salary == 0 {
		salary = player.GetSalaryFD()
	}

	skillMultiplier := 1.0
	if salary > 11000 {
		skillMultiplier = 2.0
	} else if salary > 9000 {
		skillMultiplier = 1.5
	} else if salary < 7000 {
		skillMultiplier = 0.5
	}

	// Factor in cut probability - can't finish well without making cut
	adjustedProb := baseProbability * skillMultiplier * (cutProb / 0.7)

	return math.Max(0.001, math.Min(0.5, adjustedProb))
}

// calculateExpectedFinish calculates expected finish position
func (gps *GolfProjectionService) calculateExpectedFinish(player types.PlayerInterface, tournament *models.GolfTournament, cutProb float64) float64 {
	// Base expected finish based on skill level
	salary := player.GetSalaryDK()
	if salary == 0 {
		salary = player.GetSalaryFD()
	}

	var baseFinish float64
	if salary > 11000 {
		baseFinish = 15.0 // Elite players
	} else if salary > 9000 {
		baseFinish = 30.0 // Very good players
	} else if salary > 7000 {
		baseFinish = 50.0 // Good players
	} else {
		baseFinish = 75.0 // Average to below average players
	}

	// Adjust for cut probability
	if cutProb < 0.5 {
		baseFinish += 30.0 // Likely to miss cut or finish poorly
	} else if cutProb > 0.8 {
		baseFinish -= 10.0 // Very likely to make cut
	}

	// Field strength adjustment
	baseFinish *= tournament.FieldStrength

	return math.Max(1.0, baseFinish)
}

// calculateStrategyFit calculates how well a player fits different strategies
func (gps *GolfProjectionService) calculateStrategyFit(projection *models.GolfProjection) float64 {
	// Strategy fit is based on balance of cut probability, upside, and consistency
	cutWeight := 0.4
	upsideWeight := 0.3
	consistencyWeight := 0.3

	cutScore := projection.FinalCutProbability
	upsideScore := (projection.Top5Probability + projection.Top10Probability) / 2.0
	consistencyScore := 1.0 - projection.VarianceScore/20.0 // Normalize variance

	return cutWeight*cutScore + upsideWeight*upsideScore + consistencyWeight*consistencyScore
}

// calculateRiskReward calculates risk/reward ratio for a player
func (gps *GolfProjectionService) calculateRiskReward(projection *models.GolfProjection) float64 {
	// Risk/reward is ratio of upside potential to downside risk
	upside := projection.Top10Probability + (projection.WinProbability * 2.0)
	downside := 1.0 - projection.FinalCutProbability

	if downside == 0 {
		return upside * 10.0 // Very high ratio if no downside risk
	}

	return upside / downside
}

// calculateVarianceScore calculates variance/volatility score for a player
func (gps *GolfProjectionService) calculateVarianceScore(player types.PlayerInterface, projection *models.GolfProjection) float64 {
	// Variance based on salary tier and probability spread
	salary := player.GetSalaryDK()
	if salary == 0 {
		salary = player.GetSalaryFD()
	}

	// Higher salary players tend to have lower variance (more consistent)
	baseVariance := 15.0
	if salary > 10000 {
		baseVariance = 12.0
	} else if salary < 7000 {
		baseVariance = 18.0
	}

	// Adjust based on probability spread
	probabilitySpread := projection.Top5Probability - projection.FinalCutProbability + 0.5
	variance := baseVariance * probabilitySpread

	return math.Max(5.0, math.Min(25.0, variance))
}
