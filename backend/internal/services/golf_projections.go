package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/optimizer"
	"github.com/jstittsworth/dfs-optimizer/internal/providers"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/sirupsen/logrus"
)

// GolfProjectionService handles golf-specific player projections
type GolfProjectionService struct {
	db             *database.DB
	cache          *CacheService
	logger         *logrus.Logger
	golfProvider   *providers.ESPNGolfClient
	weatherService *WeatherService
}

// NewGolfProjectionService creates a new golf projection service
func NewGolfProjectionService(
	db *database.DB,
	cache *CacheService,
	logger *logrus.Logger,
) *GolfProjectionService {
	return &GolfProjectionService{
		db:           db,
		cache:        cache,
		logger:       logger,
		golfProvider: providers.NewESPNGolfClient(cache, logger),
	}
}

// SetWeatherService sets the weather service for weather-adjusted projections
func (gps *GolfProjectionService) SetWeatherService(ws *WeatherService) {
	gps.weatherService = ws
}

// GenerateProjections generates projections for golf players in a tournament
func (gps *GolfProjectionService) GenerateProjections(
	ctx context.Context,
	players []models.Player,
	tournamentID string,
) (map[uint]*models.GolfProjection, map[uint]map[uint]float64, error) {
	projections := make(map[uint]*models.GolfProjection)

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

	courseHistory, err := gps.getCourseHistory(ctx, tournament.CourseID, players)
	if err != nil {
		gps.logger.Warn("Failed to get course history", "error", err)
	}

	// Generate projections for each player
	for _, player := range players {
		projection := gps.generatePlayerProjection(player, tournament, entries[player.ID], courseHistory[player.ID])
		projections[player.ID] = projection
	}

	// Generate correlation matrix
	correlations := gps.generateCorrelations(players, entries)

	return projections, correlations, nil
}

// generatePlayerProjection generates projection for a single player
func (gps *GolfProjectionService) generatePlayerProjection(
	player models.Player,
	tournament *models.GolfTournament,
	entry *models.GolfPlayerEntry,
	history *models.GolfCourseHistory,
) *models.GolfProjection {
	projection := &models.GolfProjection{
		PlayerID:     fmt.Sprintf("%d", player.ID),
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

	// Weather adjustment
	if gps.weatherService != nil && tournament.WeatherConditions.WindSpeed > 0 {
		weatherImpact := gps.calculateWeatherImpact(tournament.WeatherConditions)
		projection.ExpectedScore *= weatherImpact
	}

	// Calculate probabilities
	projection.CutProbability = gps.calculateCutProbability(player, tournament, entry, history)
	projection.Top10Probability = gps.calculateTop10Probability(player, tournament, projection.CutProbability)
	projection.Top25Probability = gps.calculateTop25Probability(player, tournament, projection.CutProbability)
	projection.WinProbability = gps.calculateWinProbability(player, tournament, projection.Top10Probability)

	// Calculate DFS points
	projection.DKPoints = gps.calculateDKPoints(projection, tournament)
	projection.FDPoints = gps.calculateFDPoints(projection, tournament)

	// Cap confidence at 0.9
	projection.Confidence = math.Min(projection.Confidence, 0.9)

	return projection
}

// calculateBaseScore calculates base expected 4-round score
func (gps *GolfProjectionService) calculateBaseScore(player models.Player, tournament *models.GolfTournament) float64 {
	// Base score relative to par (4 rounds)
	coursePar := float64(tournament.CoursePar * 4) // 4-round total par

	// Use player's projected points as a proxy for skill level
	// Assuming projected points correlate with expected finish position
	skillFactor := (100 - player.ProjectedPoints) / 100 // Higher projected points = lower score

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
	player models.Player,
	tournament *models.GolfTournament,
	entry *models.GolfPlayerEntry,
	history *models.GolfCourseHistory,
) float64 {
	baseProbability := 0.5 // Start at 50%

	// Adjust based on player salary (proxy for ranking/skill)
	if player.Salary > 10000 {
		baseProbability += 0.2
	} else if player.Salary > 8000 {
		baseProbability += 0.1
	} else if player.Salary < 6000 {
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
	player models.Player,
	tournament *models.GolfTournament,
	cutProbability float64,
) float64 {
	if cutProbability < 0.3 {
		return 0.01 // Very unlikely to top 10 if unlikely to make cut
	}

	// Base on salary/skill level
	baseProbability := 0.1 // 10% base chance

	if player.Salary > 11000 {
		baseProbability = 0.25
	} else if player.Salary > 9000 {
		baseProbability = 0.15
	} else if player.Salary < 7000 {
		baseProbability = 0.05
	}

	// Factor in cut probability
	return baseProbability * (cutProbability / 0.5)
}

func (gps *GolfProjectionService) calculateTop25Probability(
	player models.Player,
	tournament *models.GolfTournament,
	cutProbability float64,
) float64 {
	// Top 25 is more achievable than top 10
	top10Prob := gps.calculateTop10Probability(player, tournament, cutProbability)
	return math.Min(cutProbability*0.6, top10Prob*2.5)
}

func (gps *GolfProjectionService) calculateWinProbability(
	player models.Player,
	tournament *models.GolfTournament,
	top10Probability float64,
) float64 {
	if top10Probability < 0.1 {
		return 0.001
	}

	// Only elite players have realistic win probability
	if player.Salary > 11500 {
		return top10Probability * 0.15
	} else if player.Salary > 10000 {
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

func (gps *GolfProjectionService) getPlayerEntries(ctx context.Context, tournamentID interface{}) (map[uint]*models.GolfPlayerEntry, error) {
	var entries []models.GolfPlayerEntry

	err := gps.db.WithContext(ctx).
		Where("tournament_id = ?", tournamentID).
		Find(&entries).Error

	if err != nil {
		return nil, err
	}

	entryMap := make(map[uint]*models.GolfPlayerEntry)
	for i := range entries {
		entryMap[entries[i].PlayerID] = &entries[i]
	}

	return entryMap, nil
}

func (gps *GolfProjectionService) getCourseHistory(ctx context.Context, courseID string, players []models.Player) (map[uint]*models.GolfCourseHistory, error) {
	playerIDs := make([]uint, len(players))
	for i, p := range players {
		playerIDs[i] = p.ID
	}

	var histories []models.GolfCourseHistory

	err := gps.db.WithContext(ctx).
		Where("course_id = ? AND player_id IN ?", courseID, playerIDs).
		Find(&histories).Error

	if err != nil {
		return nil, err
	}

	historyMap := make(map[uint]*models.GolfCourseHistory)
	for i := range histories {
		historyMap[histories[i].PlayerID] = &histories[i]
	}

	return historyMap, nil
}

func (gps *GolfProjectionService) generateCorrelations(players []models.Player, entries map[uint]*models.GolfPlayerEntry) map[uint]map[uint]float64 {
	// Use the golf correlation builder
	builder := optimizer.NewGolfCorrelationBuilder(players)

	// Convert map to slice for the builder
	var entrySlice []models.GolfPlayerEntry
	for _, entry := range entries {
		if entry != nil {
			entrySlice = append(entrySlice, *entry)
		}
	}

	builder.SetPlayerEntries(entrySlice)

	return builder.BuildCorrelationMatrix()
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
