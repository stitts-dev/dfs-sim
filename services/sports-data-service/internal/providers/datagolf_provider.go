package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/managers"
	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/models"
	"github.com/stitts-dev/dfs-sim/shared/types"
	"gorm.io/gorm"
)

// DataGolfClient implements golf-specific data fetching from DataGolf API
type DataGolfClient struct {
	httpClient     *http.Client
	cache          types.CacheProvider
	logger         *logrus.Logger
	apiKey         string
	baseURL        string
	rateLimiter    *time.Ticker
	retryAttempts  int
	requestTracker *DataGolfRateLimitTracker
	db             *gorm.DB
	sportsManager  *managers.SportsManager
	mu             sync.Mutex
}

// DataGolfRateLimitTracker tracks API usage (DataGolf has much higher limits)
type DataGolfRateLimitTracker struct {
	mu           sync.Mutex
	requestCount int
	lastReset    time.Time
}

// NewDataGolfClient creates a new DataGolf client
func NewDataGolfClient(apiKey string, db *gorm.DB, cache types.CacheProvider, logger *logrus.Logger) *DataGolfClient {
	// Initialize rate limit tracker (much more generous than RapidAPI)
	tracker := &DataGolfRateLimitTracker{
		lastReset: time.Now(),
	}

	// Create sports manager only if db is provided
	var sportsManager *managers.SportsManager
	if db != nil {
		sportsManager = managers.NewSportsManager(db, cache, logger)
	}

	return &DataGolfClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:          cache,
		logger:         logger,
		apiKey:         apiKey,
		baseURL:        "https://feeds.datagolf.com",
		rateLimiter:    time.NewTicker(1 * time.Second), // Conservative 1 req/second
		retryAttempts:  3,
		requestTracker: tracker,
		db:             db,
		sportsManager:  sportsManager,
	}
}

// Enhanced DataGolf API Response Structures

// DataGolf Enhanced Provider Interface - implements advanced golf analytics
type EnhancedGolfDataProvider interface {
	// Standard methods
	GetPlayers(sport types.Sport, date string) ([]types.PlayerData, error)
	GetCurrentTournament() (*GolfTournamentData, error)
	GetTournamentSchedule() ([]GolfTournamentData, error)

	// DataGolf-specific advanced methods
	GetStrokesGainedData(playerID string, tournamentID string) (*StrokesGainedMetrics, error)
	GetCourseAnalytics(courseID string) (*CourseAnalytics, error)
	GetPreTournamentPredictions(tournamentID string) (*TournamentPredictions, error)
	GetLiveTournamentData(tournamentID string) (*LiveTournamentData, error)
	GetPlayerCourseHistory(playerID, courseID string) (*PlayerCourseHistory, error)
	GetWeatherImpactData(tournamentID string) (*WeatherImpactAnalysis, error)
}

// Enhanced Data Models
type StrokesGainedMetrics struct {
	PlayerID           int64     `json:"player_id"`
	TournamentID       string    `json:"tournament_id"`
	SGOffTheTee       float64   `json:"sg_off_the_tee"`
	SGApproach        float64   `json:"sg_approach"`
	SGAroundTheGreen  float64   `json:"sg_around_the_green"`
	SGPutting         float64   `json:"sg_putting"`
	SGTotal           float64   `json:"sg_total"`
	Consistency       float64   `json:"consistency_rating"`
	VolatilityIndex   float64   `json:"volatility_index"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type CourseAnalytics struct {
	CourseID              string                 `json:"course_id"`
	DifficultyRating      float64               `json:"difficulty_rating"`
	Length                int                   `json:"length"`
	Par                   int                   `json:"par"`
	PlayerTypeAdvantages  map[string]float64    `json:"player_type_advantages"`
	WeatherSensitivity    map[string]float64    `json:"weather_sensitivity"`
	HistoricalScoring     ScoreDistribution     `json:"historical_scoring"`
	KeyHoles              []int                 `json:"key_holes"`
	SkillPremiums         SkillPremiumWeights   `json:"skill_premiums"`
	// Course-specific skill premium ratings for optimization algorithms
	DrivingPremium        float64               `json:"driving_premium"`
	ApproachPremium       float64               `json:"approach_premium"`
	ShortGamePremium      float64               `json:"short_game_premium"`
	PuttingPremium        float64               `json:"putting_premium"`
}

type SkillPremiumWeights struct {
	DrivingDistance    float64 `json:"driving_distance"`
	DrivingAccuracy    float64 `json:"driving_accuracy"`
	ApproachPrecision  float64 `json:"approach_precision"`
	ShortGameSkill     float64 `json:"short_game_skill"`
	PuttingConsistency float64 `json:"putting_consistency"`
}

type ScoreDistribution struct {
	MeanScore      float64 `json:"mean_score"`
	MedianScore    float64 `json:"median_score"`
	StandardDev    float64 `json:"standard_deviation"`
	WinningScore   float64 `json:"winning_score"`
	CutScore       float64 `json:"cut_score"`
}

type TournamentPredictions struct {
	TournamentID  string                      `json:"tournament_id"`
	GeneratedAt   time.Time                   `json:"generated_at"`
	Predictions   []EnhancedPlayerPrediction  `json:"predictions"`
	CourseModel   CourseModelData             `json:"course_model"`
	WeatherModel  WeatherModelData            `json:"weather_model"`
}

type EnhancedPlayerPrediction struct {
	PlayerID              int     `json:"player_id"`
	PlayerName            string  `json:"player_name"`
	WinProbability        float64 `json:"win_probability"`
	Top5Probability       float64 `json:"top5_probability"`
	Top10Probability      float64 `json:"top10_probability"`
	Top20Probability      float64 `json:"top20_probability"`
	MakeCutProbability    float64 `json:"make_cut_probability"`
	ProjectedScore        float64 `json:"projected_score"`
	ProjectedFinish       float64 `json:"projected_finish"`
	CourseFit             float64 `json:"course_fit"`
	WeatherAdvantage      float64 `json:"weather_advantage"`
	VolatilityRating      float64 `json:"volatility_rating"`
	StrategyFit           map[string]float64 `json:"strategy_fit"`
	// Strokes gained metrics for optimization algorithms
	SGOffTee              float64 `json:"sg_off_tee"`
	SGApproach            float64 `json:"sg_approach"`
	SGAroundGreen         float64 `json:"sg_around_green"`
	SGPutting             float64 `json:"sg_putting"`
}

type CourseModelData struct {
	ModelVersion     string             `json:"model_version"`
	Accuracy         float64            `json:"accuracy"`
	KeyFactors       []string           `json:"key_factors"`
	PlayerTypeBonus  map[string]float64 `json:"player_type_bonus"`
}

type WeatherModelData struct {
	CurrentConditions WeatherConditions  `json:"current_conditions"`
	Forecast          []WeatherForecast  `json:"forecast"`
	ImpactScore       float64            `json:"impact_score"`
}

type WeatherForecast struct {
	Date        time.Time `json:"date"`
	Temperature int       `json:"temperature"`
	WindSpeed   int       `json:"wind_speed"`
	WindDir     string    `json:"wind_direction"`
	Conditions  string    `json:"conditions"`
	Humidity    int       `json:"humidity"`
}

type WeatherConditions struct {
	Temperature int    `json:"temperature"`
	WindSpeed   int    `json:"wind_speed"`
	WindDir     string `json:"wind_direction"`
	Conditions  string `json:"conditions"`
	Humidity    int    `json:"humidity"`
}

type LiveTournamentData struct {
	TournamentID     string                 `json:"tournament_id"`
	CurrentRound     int                    `json:"current_round"`
	CutLine          int                    `json:"cut_line"`
	CutMade          bool                   `json:"cut_made"`
	LeaderScore      int                    `json:"leader_score"`
	LastUpdated      time.Time              `json:"last_updated"`
	LiveLeaderboard  []LiveLeaderboardEntry `json:"live_leaderboard"`
	WeatherUpdate    WeatherConditions      `json:"weather_update"`
	PlaySuspended    bool                   `json:"play_suspended"`
}

type LiveLeaderboardEntry struct {
	PlayerID         int     `json:"player_id"`
	PlayerName       string  `json:"player_name"`
	Position         int     `json:"position"`
	TotalScore       int     `json:"total_score"`
	ThruHoles        int     `json:"thru_holes"`
	RoundScore       int     `json:"round_score"`
	MovementIndicator string `json:"movement_indicator"`
	TeeTime          string  `json:"tee_time"`
	IsOnCourse       bool    `json:"is_on_course"`
}

type PlayerCourseHistory struct {
	PlayerID           int                    `json:"player_id"`
	CourseID           string                 `json:"course_id"`
	TotalAppearances   int                    `json:"total_appearances"`
	AveragingScore     float64                `json:"averaging_score"`
	BestFinish         int                    `json:"best_finish"`
	RecentForm         []CourseHistoryEntry   `json:"recent_form"`
	StrokesGainedAvg   StrokesGainedMetrics   `json:"strokes_gained_avg"`
	CourseFitScore     float64                `json:"course_fit_score"`
}

type CourseHistoryEntry struct {
	Year           int     `json:"year"`
	Position       int     `json:"position"`
	Score          int     `json:"score"`
	MadeCut        bool    `json:"made_cut"`
	RoundsPlayed   int     `json:"rounds_played"`
}

type WeatherImpactAnalysis struct {
	TournamentID       string             `json:"tournament_id"`
	AnalysisDate       time.Time          `json:"analysis_date"`
	OverallImpact      float64            `json:"overall_impact"`
	PlayerImpacts      []PlayerWeatherImpact `json:"player_impacts"`
	CourseAdjustments  CourseWeatherAdjustment `json:"course_adjustments"`
	OptimalStrategy    WeatherStrategy    `json:"optimal_strategy"`
}

type PlayerWeatherImpact struct {
	PlayerID          int     `json:"player_id"`
	PlayerName        string  `json:"player_name"`
	WeatherAdvantage  float64 `json:"weather_advantage"`
	ImpactCategories  map[string]float64 `json:"impact_categories"`
	AdjustedProjection float64 `json:"adjusted_projection"`
}

type CourseWeatherAdjustment struct {
	ScoreImpact        float64 `json:"score_impact"`
	DistanceReduction  float64 `json:"distance_reduction"`
	VarianceMultiplier float64 `json:"variance_multiplier"`
	SoftConditions     bool    `json:"soft_conditions"`
}

type WeatherStrategy struct {
	StrategyType         string             `json:"strategy_type"`
	PlayerTypePreference []string           `json:"player_type_preference"`
	RecommendedWeights   map[string]float64 `json:"recommended_weights"`
}

// DataGolf API Response Structures
type dataGolfScheduleResponse struct {
	Schedule []dataGolfTournament `json:"schedule"`
}

type dataGolfTournament struct {
	EventID     FlexibleID `json:"event_id"`
	EventName   string     `json:"event_name"`
	StartDate   string     `json:"start_date"`
	EndDate     string     `json:"end_date"`
	Tour        string     `json:"tour"`
	Country     string     `json:"country"`
	Course      string     `json:"course"`
	Year        int        `json:"year"`
	Status      string     `json:"status"`
	Purse       int64      `json:"purse"`
}

// FlexibleID handles both string and number IDs from DataGolf API
type FlexibleID string

// UnmarshalJSON handles unmarshaling of both string and number event IDs
func (f *FlexibleID) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*f = FlexibleID(str)
		return nil
	}
	
	// If string fails, try as number
	var num int64
	if err := json.Unmarshal(data, &num); err == nil {
		*f = FlexibleID(fmt.Sprintf("%d", num))
		return nil
	}
	
	return fmt.Errorf("event_id must be string or number")
}

// String returns the string representation
func (f FlexibleID) String() string {
	return string(f)
}

type dataGolfFieldResponse struct {
	Field []dataGolfPlayer `json:"field"`
}

type dataGolfPlayer struct {
	DGPlayerID    int     `json:"dg_id"`
	PlayerName    string  `json:"player_name"`
	Country       string  `json:"country"`
	FantasyPoints float64 `json:"fantasy_points,omitempty"`
	DKSalary      int     `json:"dk_salary,omitempty"`
	FDSalary      int     `json:"fd_salary,omitempty"`
	SGTotal       float64 `json:"sg_total,omitempty"`
	SGOffTheTee   float64 `json:"sg_ott,omitempty"`
	SGApproach    float64 `json:"sg_app,omitempty"`
	SGAroundGreen float64 `json:"sg_arg,omitempty"`
	SGPutting     float64 `json:"sg_putt,omitempty"`
}

type dataGolfPreTournamentResponse struct {
	Predictions []dataGolfPrediction `json:"predictions"`
}

type dataGolfPrediction struct {
	DGPlayerID        int     `json:"dg_id"`
	PlayerName        string  `json:"player_name"`
	WinProbability    float64 `json:"win_prob"`
	Top5Probability   float64 `json:"top_5_prob"`
	Top10Probability  float64 `json:"top_10_prob"`
	Top20Probability  float64 `json:"top_20_prob"`
	MakeCutProbability float64 `json:"make_cut_prob"`
	ProjectedScore    float64 `json:"projected_score"`
	ProjectedFinish   float64 `json:"projected_finish"`
}

// GetPlayers fetches all players for a golf tournament using DataGolf API
func (c *DataGolfClient) GetPlayers(sport types.Sport, date string) ([]types.PlayerData, error) {
	if sport != types.SportGolf {
		return nil, fmt.Errorf("DataGolfClient only supports golf")
	}

	cacheKey := fmt.Sprintf("datagolf:golf:players:%s", date)

	// Check cache first
	var cachedPlayers []types.PlayerData
	err := c.cache.GetSimple(cacheKey, &cachedPlayers)
	if err == nil {
		c.logger.WithField("source", "cache").Info("Returning cached golf players from DataGolf")
		return cachedPlayers, nil
	}

	// Get current tournament first
	tournament, err := c.GetCurrentTournament()
	if err != nil {
		return nil, fmt.Errorf("failed to get current tournament: %w", err)
	}

	// Fetch field data from DataGolf
	fieldData, err := c.getField(tournament.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get field data: %w", err)
	}

	// Get pre-tournament predictions for additional data
	predictions, err := c.getPreTournamentPredictions(tournament.ID)
	if err != nil {
		c.logger.WithError(err).Warn("Failed to get predictions, continuing with field data only")
		predictions = &dataGolfPreTournamentResponse{}
	}

	// Create prediction lookup map
	predictionMap := make(map[int]*dataGolfPrediction)
	for i := range predictions.Predictions {
		pred := &predictions.Predictions[i]
		predictionMap[pred.DGPlayerID] = pred
	}

	var players []types.PlayerData
	for _, player := range fieldData.Field {
		pred := predictionMap[player.DGPlayerID]

		playerData := types.PlayerData{
			ExternalID:  fmt.Sprintf("datagolf_%d", player.DGPlayerID),
			Name:        player.PlayerName,
			Team:        player.Country,
			Position:    "G", // All golfers have position "G"
			Stats:       c.extractPlayerStats(player, pred),
			LastUpdated: time.Now(),
			Source:      "datagolf",
		}
		players = append(players, playerData)
	}

	// Cache for appropriate duration
	cacheDuration := 2 * time.Hour
	if tournament.Status == "completed" {
		cacheDuration = 24 * time.Hour
	}

	if len(players) > 0 {
		// Save players to database
		if err := c.savePlayersToDatabase(context.Background(), players, tournament); err != nil {
			c.logger.WithError(err).Warn("Failed to save players to database")
		}

		c.cache.SetSimple(cacheKey, players, cacheDuration)
		c.logger.WithField("count", len(players)).Info("Cached golf players from DataGolf")
	}

	return players, nil
}

// GetPlayer fetches a specific golf player from cached data
func (c *DataGolfClient) GetPlayer(sport types.Sport, externalID string) (*types.PlayerData, error) {
	players, err := c.GetPlayers(sport, time.Now().Format("2006-01-02"))
	if err == nil {
		for _, player := range players {
			if player.ExternalID == externalID {
				return &player, nil
			}
		}
	}

	return nil, fmt.Errorf("player not found in cached data")
}

// GetTeamRoster returns players by country for golf
func (c *DataGolfClient) GetTeamRoster(sport types.Sport, teamID string) ([]types.PlayerData, error) {
	players, err := c.GetPlayers(sport, time.Now().Format("2006-01-02"))
	if err != nil {
		return nil, err
	}

	var teamPlayers []types.PlayerData
	for _, player := range players {
		if player.Team == teamID {
			teamPlayers = append(teamPlayers, player)
		}
	}

	return teamPlayers, nil
}

// GetCurrentTournament fetches the current tournament details
func (c *DataGolfClient) GetCurrentTournament() (*GolfTournamentData, error) {
	cacheKey := "datagolf:golf:current_tournament"

	// Check cache first
	var cachedTournament GolfTournamentData
	err := c.cache.GetSimple(cacheKey, &cachedTournament)
	if err == nil {
		return &cachedTournament, nil
	}

	// Get schedule to find current tournament
	schedule, err := c.GetTournamentSchedule()
	if err != nil {
		return nil, fmt.Errorf("failed to get tournament schedule: %w", err)
	}

	// Find current tournament from schedule
	now := time.Now()
	var currentTournament *GolfTournamentData
	for _, tournament := range schedule {
		if tournament.StartDate.Before(now.Add(7*24*time.Hour)) && tournament.EndDate.After(now.Add(-7*24*time.Hour)) {
			currentTournament = &tournament
			break
		}
	}

	if currentTournament == nil {
		return nil, fmt.Errorf("no current tournament found")
	}

	// Save tournament to database
	if err := c.saveTournamentToDatabase(context.Background(), currentTournament); err != nil {
		c.logger.WithError(err).Warn("Failed to save tournament to database")
	}

	// Cache for 24 hours
	c.cache.SetSimple(cacheKey, currentTournament, 24*time.Hour)
	c.logger.WithField("tournament", currentTournament.Name).Info("Cached tournament data from DataGolf")

	return currentTournament, nil
}

// GetTournamentSchedule fetches the tournament schedule from DataGolf
func (c *DataGolfClient) GetTournamentSchedule() ([]GolfTournamentData, error) {
	cacheKey := "datagolf:golf:schedule"

	// Check cache first
	var cachedSchedule []GolfTournamentData
	err := c.cache.GetSimple(cacheKey, &cachedSchedule)
	if err == nil {
		c.logger.Info("Returning cached tournament schedule from DataGolf")
		return cachedSchedule, nil
	}

	// Fetch from DataGolf API
	currentYear := time.Now().Year()
	url := fmt.Sprintf("%s/get-schedule?tour=pga&year=%d&key=%s", c.baseURL, currentYear, c.apiKey)

	var response dataGolfScheduleResponse
	if err := c.makeRequest(url, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch schedule: %w", err)
	}

	var schedule []GolfTournamentData
	for _, t := range response.Schedule {
		startDate := c.parseDate(t.StartDate)
		endDate := c.parseDate(t.EndDate)

		tournament := GolfTournamentData{
			ID:        t.EventID.String(),
			Name:      t.EventName,
			StartDate: startDate,
			EndDate:   endDate,
			Status:    c.mapTournamentStatus(t.Status, startDate, endDate),
			Purse:     float64(t.Purse),
			CourseName: t.Course,
		}
		schedule = append(schedule, tournament)
	}

	// Cache for 7 days
	c.cache.SetSimple(cacheKey, schedule, 7*24*time.Hour)
	c.logger.WithField("count", len(schedule)).Info("Cached tournament schedule from DataGolf")

	return schedule, nil
}

// getField fetches the field for a specific tournament
func (c *DataGolfClient) getField(eventID string) (*dataGolfFieldResponse, error) {
	cacheKey := fmt.Sprintf("datagolf:golf:field:%s", eventID)

	// Check cache first
	var cachedField dataGolfFieldResponse
	err := c.cache.GetSimple(cacheKey, &cachedField)
	if err == nil {
		return &cachedField, nil
	}

	// Fetch from DataGolf API
	url := fmt.Sprintf("%s/field-updates?tour=pga&event_id=%s&key=%s", c.baseURL, eventID, c.apiKey)

	var response dataGolfFieldResponse
	if err := c.makeRequest(url, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch field: %w", err)
	}

	// Cache for 2 hours during active tournament, 24 hours if completed
	cacheDuration := 2 * time.Hour
	c.cache.SetSimple(cacheKey, response, cacheDuration)

	return &response, nil
}

// getPreTournamentPredictions fetches pre-tournament predictions
func (c *DataGolfClient) getPreTournamentPredictions(eventID string) (*dataGolfPreTournamentResponse, error) {
	cacheKey := fmt.Sprintf("datagolf:golf:predictions:%s", eventID)

	// Check cache first
	var cachedPredictions dataGolfPreTournamentResponse
	err := c.cache.GetSimple(cacheKey, &cachedPredictions)
	if err == nil {
		return &cachedPredictions, nil
	}

	// Fetch from DataGolf API
	url := fmt.Sprintf("%s/preds/pre-tournament?tour=pga&file_format=json&key=%s", c.baseURL, eventID, c.apiKey)

	var response dataGolfPreTournamentResponse
	if err := c.makeRequest(url, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch predictions: %w", err)
	}

	// Cache for 4 hours
	c.cache.SetSimple(cacheKey, response, 4*time.Hour)

	return &response, nil
}

// makeRequest handles HTTP requests with rate limiting and retries
func (c *DataGolfClient) makeRequest(url string, target interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Rate limiting
	<-c.rateLimiter.C

	// Track request
	c.trackRequest()

	var lastErr error
	for attempt := 0; attempt < c.retryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}

		// Set headers
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "dfs-sports-data-service/1.0.0")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response body: %w", err)
			}

			// Log response for debugging (truncated)
			if len(body) > 500 {
				c.logger.WithField("response_preview", string(body[:500])+"...").Debug("DataGolf API response preview")
			} else {
				c.logger.WithField("response_body", string(body)).Debug("DataGolf API response body")
			}

			if err := json.Unmarshal(body, target); err != nil {
				c.logger.WithFields(map[string]interface{}{
					"url": url,
					"response_length": len(body),
					"error": err.Error(),
				}).Error("Failed to decode JSON response from DataGolf")
				return fmt.Errorf("failed to decode response: %w", err)
			}
			return nil
		}

		// Handle specific error codes
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("invalid API key")
		case http.StatusForbidden:
			return fmt.Errorf("access forbidden - check subscription")
		case http.StatusTooManyRequests:
			return fmt.Errorf("rate limit exceeded")
		default:
			lastErr = fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", c.retryAttempts, lastErr)
}

// trackRequest tracks API usage
func (c *DataGolfClient) trackRequest() {
	c.requestTracker.mu.Lock()
	defer c.requestTracker.mu.Unlock()

	// Reset counter if needed (not as critical for DataGolf)
	now := time.Now()
	if now.Day() != c.requestTracker.lastReset.Day() {
		c.requestTracker.requestCount = 0
		c.requestTracker.lastReset = now
		c.logger.Info("Reset daily DataGolf request counter")
	}

	c.requestTracker.requestCount++

	c.logger.WithField("request_count", c.requestTracker.requestCount).Debug("Tracked DataGolf API request")
}

// extractPlayerStats extracts stats from DataGolf player and prediction data
func (c *DataGolfClient) extractPlayerStats(player dataGolfPlayer, prediction *dataGolfPrediction) interface{} {
	stats := make(map[string]interface{})

	// Basic player data
	if player.FantasyPoints > 0 {
		stats["fantasy_points"] = player.FantasyPoints
	}
	if player.DKSalary > 0 {
		stats["dk_salary"] = float64(player.DKSalary)
	}
	if player.FDSalary > 0 {
		stats["fd_salary"] = float64(player.FDSalary)
	}

	// Strokes gained metrics
	if player.SGTotal != 0 {
		stats["sg_total"] = player.SGTotal
	}
	if player.SGOffTheTee != 0 {
		stats["sg_off_the_tee"] = player.SGOffTheTee
	}
	if player.SGApproach != 0 {
		stats["sg_approach"] = player.SGApproach
	}
	if player.SGAroundGreen != 0 {
		stats["sg_around_green"] = player.SGAroundGreen
	}
	if player.SGPutting != 0 {
		stats["sg_putting"] = player.SGPutting
	}

	// Prediction data if available
	if prediction != nil {
		stats["win_probability"] = prediction.WinProbability
		stats["top5_probability"] = prediction.Top5Probability
		stats["top10_probability"] = prediction.Top10Probability
		stats["top20_probability"] = prediction.Top20Probability
		stats["make_cut_probability"] = prediction.MakeCutProbability
		stats["projected_score"] = prediction.ProjectedScore
		stats["projected_finish"] = prediction.ProjectedFinish
	}

	return stats
}

// Helper functions
func (c *DataGolfClient) parseDate(dateStr string) time.Time {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.logger.WithField("date", dateStr).Warn("Failed to parse date, using current time")
		return time.Now()
	}
	return t
}

func (c *DataGolfClient) mapTournamentStatus(status string, startDate, endDate time.Time) string {
	now := time.Now()

	// Use date logic if status is not explicit
	if now.Before(startDate) {
		return "scheduled"
	} else if now.After(endDate) {
		return "completed"
	} else {
		return "in_progress"
	}
}

// Database integration methods (same interface as RapidAPI)
func (c *DataGolfClient) saveTournamentToDatabase(ctx context.Context, tournamentData *GolfTournamentData) error {
	if c.db == nil || c.sportsManager == nil {
		c.logger.Debug("Skipping database save - no database configured")
		return nil
	}

	// Ensure golf sport exists
	sport, err := c.sportsManager.EnsureGolfSport(ctx)
	if err != nil {
		return fmt.Errorf("failed to ensure golf sport: %w", err)
	}

	// Check if tournament already exists
	var existingTournament models.GolfTournament
	err = c.db.WithContext(ctx).Where("external_id = ?", tournamentData.ID).First(&existingTournament).Error

	if err == nil {
		// Tournament exists, update it
		existingTournament.Name = tournamentData.Name
		existingTournament.StartDate = tournamentData.StartDate
		existingTournament.EndDate = tournamentData.EndDate
		existingTournament.Status = models.TournamentStatus(tournamentData.Status)
		existingTournament.CourseName = tournamentData.CourseName
		existingTournament.Purse = tournamentData.Purse

		if err := c.db.WithContext(ctx).Save(&existingTournament).Error; err != nil {
			return fmt.Errorf("failed to update tournament: %w", err)
		}

		c.logger.WithFields(logrus.Fields{
			"tournament_id":   existingTournament.ID,
			"tournament_name": tournamentData.Name,
		}).Info("Updated existing golf tournament in database (DataGolf)")

		return nil
	}

	// Tournament doesn't exist, create new one
	tournament := models.GolfTournament{
		ID:           uuid.New(),
		ExternalID:   tournamentData.ID,
		Name:         tournamentData.Name,
		StartDate:    tournamentData.StartDate,
		EndDate:      tournamentData.EndDate,
		Status:       models.TournamentStatus(tournamentData.Status),
		CourseName:   tournamentData.CourseName,
		Purse:        tournamentData.Purse,
	}

	if err := c.db.WithContext(ctx).Create(&tournament).Error; err != nil {
		return fmt.Errorf("failed to create tournament: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"tournament_id":   tournament.ID,
		"tournament_name": tournamentData.Name,
		"sport_id":        sport.ID,
	}).Info("Created new golf tournament in database (DataGolf)")

	return nil
}

func (c *DataGolfClient) savePlayersToDatabase(ctx context.Context, playersData []types.PlayerData, tournament *GolfTournamentData) error {
	if c.db == nil || c.sportsManager == nil {
		c.logger.Debug("Skipping player database save - no database configured")
		return nil
	}

	// Ensure golf sport exists
	sport, err := c.sportsManager.EnsureGolfSport(ctx)
	if err != nil {
		return fmt.Errorf("failed to ensure golf sport: %w", err)
	}

	// Get tournament from database to link players
	var dbTournament models.GolfTournament
	err = c.db.WithContext(ctx).Where("external_id = ?", tournament.ID).First(&dbTournament).Error
	if err != nil {
		c.logger.WithError(err).Warn("Tournament not found in database, skipping player save")
		return nil
	}

	for _, playerData := range playersData {
		// Check if player already exists
		var existingPlayer models.Player
		err = c.db.WithContext(ctx).Where("external_id = ? AND sport_id = ?", playerData.ExternalID, sport.ID).First(&existingPlayer).Error

		if err == nil {
			// Player exists, update basic info
			existingPlayer.Name = playerData.Name
			existingPlayer.Team = playerData.Team
			existingPlayer.Position = playerData.Position
			existingPlayer.GameTime = tournament.StartDate

			if err := c.db.WithContext(ctx).Save(&existingPlayer).Error; err != nil {
				c.logger.WithError(err).WithField("player", playerData.Name).Warn("Failed to update player (DataGolf)")
				continue
			}
		} else {
			// Create new player
			player := models.Player{
				ID:              uuid.New(),
				SportID:         sport.ID,
				ExternalID:      playerData.ExternalID,
				Name:            playerData.Name,
				Team:            playerData.Team,
				Position:        playerData.Position,
				ProjectedPoints: 0,
				FloorPoints:     0,
				CeilingPoints:   0,
				GameTime:        tournament.StartDate,
				ImageURL:        playerData.ImageURL,
			}

			if err := c.db.WithContext(ctx).Create(&player).Error; err != nil {
				c.logger.WithError(err).WithField("player", playerData.Name).Warn("Failed to create player (DataGolf)")
				continue
			}

			c.logger.WithFields(logrus.Fields{
				"player_id":   player.ID,
				"player_name": playerData.Name,
				"sport_id":    sport.ID,
			}).Debug("Created new golf player in database (DataGolf)")
		}
	}

	c.logger.WithFields(logrus.Fields{
		"tournament_name": tournament.Name,
		"player_count":    len(playersData),
		"sport_id":        sport.ID,
	}).Info("Saved golf players to database (DataGolf)")

	return nil
}

// WarmCache performs cache warming with DataGolf data
func (c *DataGolfClient) WarmCache() error {
	c.logger.Info("Starting DataGolf cache warming")

	// Current tournament
	tournament, err := c.GetCurrentTournament()
	if err != nil {
		c.logger.WithError(err).Error("Failed to warm tournament cache (DataGolf)")
		return err
	}

	// Players if tournament is active
	if tournament.Status == "in_progress" || tournament.Status == "scheduled" {
		_, err = c.GetPlayers(types.SportGolf, time.Now().Format("2006-01-02"))
		if err != nil {
			c.logger.WithError(err).Error("Failed to warm players cache (DataGolf)")
		}
	}

	c.logger.WithField("requests_used", c.requestTracker.requestCount).
		Info("DataGolf cache warming completed")

	return nil
}

// Enhanced DataGolf API Methods Implementation

// GetStrokesGainedData fetches strokes gained metrics for a specific player and tournament
func (c *DataGolfClient) GetStrokesGainedData(playerID string, tournamentID string) (*StrokesGainedMetrics, error) {
	cacheKey := fmt.Sprintf("datagolf:sg:%s:%s", playerID, tournamentID)

	// Check cache first
	var cachedSG StrokesGainedMetrics
	err := c.cache.GetSimple(cacheKey, &cachedSG)
	if err == nil {
		return &cachedSG, nil
	}

	// Fetch from DataGolf API - using skill decompositions endpoint
	url := fmt.Sprintf("%s/preds/player-decompositions?tour=pga&file_format=json&key=%s",
		c.baseURL, c.apiKey)

	var decompositionResponse SkillDecompositionsResponse

	if err := c.makeRequest(url, &decompositionResponse); err != nil {
		return nil, fmt.Errorf("failed to fetch strokes gained data: %w", err)
	}

	// Find the specific player's data
	playerIDInt, err := strconv.Atoi(playerID)
	if err != nil {
		return nil, fmt.Errorf("invalid player ID: %w", err)
	}

	for _, decomp := range decompositionResponse.Decompositions {
		if decomp.DGID == playerIDInt {
			sgMetrics := &StrokesGainedMetrics{
				PlayerID:           int64(decomp.DGID),
				TournamentID:       tournamentID,
				SGOffTheTee:        decomp.SGOffTheTee,
				SGApproach:         decomp.SGApproach,
				SGAroundTheGreen:   decomp.SGAroundTheGreen,
				SGPutting:          decomp.SGPutting,
				SGTotal:            decomp.SGTotal,
				Consistency:        0.8, // TODO: Calculate from historical data
				VolatilityIndex:    0.2, // TODO: Calculate from historical data
				UpdatedAt:          time.Now(),
			}

			// Cache for 6 hours
			c.cache.SetSimple(cacheKey, sgMetrics, 6*time.Hour)

			return sgMetrics, nil
		}
	}

	return nil, fmt.Errorf("player not found in strokes gained data")
}

// GetCourseAnalytics fetches comprehensive course analytics
// TODO: This would be built from historical round data analysis, not a direct API endpoint
func (c *DataGolfClient) GetCourseAnalytics(courseID string) (*CourseAnalytics, error) {
	cacheKey := fmt.Sprintf("datagolf:course_analytics:%s", courseID)

	// Check cache first - course analytics change infrequently
	var cachedAnalytics CourseAnalytics
	err := c.cache.GetSimple(cacheKey, &cachedAnalytics)
	if err == nil {
		return &cachedAnalytics, nil
	}

	// TODO: Implement course analytics by analyzing historical round data
	// For now, return placeholder data based on course characteristics
	analytics := &CourseAnalytics{
		CourseID:         courseID,
		DifficultyRating: 72.5, // Placeholder - would be calculated from historical scoring
		Length:           7200,  // Placeholder - would come from course database
		Par:              72,    // Placeholder - would come from course database
		PlayerTypeAdvantages: map[string]float64{
			"bomber":       1.2,  // Long hitters advantage
			"accurate":     0.9,  // Accuracy players slight disadvantage
			"short_game":   1.0,  // Neutral short game course
			"putter":       1.1,  // Putting slightly more important
		},
		WeatherSensitivity: map[string]float64{
			"wind":         1.5,  // High wind sensitivity
			"rain":         1.2,  // Moderate rain impact
			"temperature":  0.8,  // Low temperature sensitivity
		},
		HistoricalScoring: ScoreDistribution{
			MeanScore:    71.2,
			MedianScore:  71.0,
			StandardDev:  2.8,
			WinningScore: -12.0,
			CutScore:     1.0,
		},
		KeyHoles: []int{8, 12, 15, 17}, // Placeholder scoring holes
		SkillPremiums: SkillPremiumWeights{
			DrivingDistance:    1.3,
			DrivingAccuracy:    0.8,
			ApproachPrecision:  1.1,
			ShortGameSkill:     1.0,
			PuttingConsistency: 1.2,
		},
	}

	// Cache for 24 hours - course analytics are relatively static
	c.cache.SetSimple(cacheKey, analytics, 24*time.Hour)

	return analytics, nil
}

// GetPreTournamentPredictions fetches enhanced pre-tournament predictions with course fit and weather
func (c *DataGolfClient) GetPreTournamentPredictions(tournamentID string) (*TournamentPredictions, error) {
	cacheKey := fmt.Sprintf("datagolf:enhanced_predictions:%s", tournamentID)

	// Check cache first
	var cachedPredictions TournamentPredictions
	err := c.cache.GetSimple(cacheKey, &cachedPredictions)
	if err == nil {
		return &cachedPredictions, nil
	}

	// Fetch predictions from DataGolf API using actual endpoint
	url := fmt.Sprintf("%s/preds/pre-tournament?tour=pga&file_format=json&key=%s",
		c.baseURL, c.apiKey)

	var apiResponse PreTournamentResponse

	if err := c.makeRequest(url, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to fetch enhanced predictions: %w", err)
	}

	// Convert DataGolf response to our enhanced format
	var enhancedPredictions []EnhancedPlayerPrediction
	for _, pred := range apiResponse.Baseline.Predictions {
		enhancedPred := EnhancedPlayerPrediction{
			PlayerID:              pred.DGID,
			PlayerName:            pred.PlayerName,
			WinProbability:        pred.WinProb,
			Top5Probability:       pred.Top5Prob,
			Top10Probability:      pred.Top10Prob,
			Top20Probability:      pred.Top20Prob,
			MakeCutProbability:    pred.MakeCutProb,
			ProjectedScore:        -8.0, // TODO: Calculate from probabilities
			ProjectedFinish:       15.0, // TODO: Calculate from probabilities
			CourseFit:             0.85, // TODO: Calculate from course analytics
			WeatherAdvantage:      0.95, // TODO: Calculate from weather data
			VolatilityRating:      0.75, // TODO: Calculate from historical performance
			StrategyFit:           map[string]float64{
				"win":      pred.WinProb,
				"top5":     pred.Top5Prob,
				"top10":    pred.Top10Prob,
				"top20":    pred.Top20Prob,
				"cut":      pred.MakeCutProb,
				"balanced": (pred.WinProb + pred.Top10Prob + pred.MakeCutProb) / 3.0,
			},
		}
		enhancedPredictions = append(enhancedPredictions, enhancedPred)
	}

	response := TournamentPredictions{
		TournamentID: tournamentID,
		GeneratedAt:  time.Now(),
		Predictions:  enhancedPredictions,
		CourseModel: CourseModelData{
			ModelVersion: "datagolf_v1",
			Accuracy:     0.72,
			KeyFactors:   []string{"strokes_gained_total", "course_history", "recent_form"},
			PlayerTypeBonus: map[string]float64{
				"bomber":     1.15,
				"accurate":   0.95,
				"short_game": 1.05,
				"putter":     1.10,
			},
		},
		WeatherModel: WeatherModelData{
			CurrentConditions: WeatherConditions{
				Temperature: 75,
				WindSpeed:   8,
				WindDir:     "SW",
				Conditions:  "partly_cloudy",
				Humidity:    65,
			},
			Forecast: []WeatherForecast{
				{Date: time.Now(), Temperature: 75, WindSpeed: 8, WindDir: "SW", Conditions: "partly_cloudy", Humidity: 65},
				{Date: time.Now().AddDate(0, 0, 1), Temperature: 73, WindSpeed: 12, WindDir: "W", Conditions: "cloudy", Humidity: 70},
				{Date: time.Now().AddDate(0, 0, 2), Temperature: 71, WindSpeed: 15, WindDir: "NW", Conditions: "breezy", Humidity: 60},
			},
			ImpactScore: 0.25, // Moderate weather impact
		},
	}

	// Cache for 4 hours during tournament week, 12 hours otherwise
	cacheDuration := 12 * time.Hour
	if c.isTournamentWeek(tournamentID) {
		cacheDuration = 4 * time.Hour
	}

	c.cache.SetSimple(cacheKey, response, cacheDuration)

	return &response, nil
}

// GetLiveTournamentData fetches real-time tournament data during active play
func (c *DataGolfClient) GetLiveTournamentData(tournamentID string) (*LiveTournamentData, error) {
	cacheKey := fmt.Sprintf("datagolf:live_data:%s", tournamentID)

	// Check cache first - very short cache for live data
	var cachedLive LiveTournamentData
	err := c.cache.GetSimple(cacheKey, &cachedLive)
	if err == nil {
		// Only return cached if it's very recent (< 5 minutes)
		if time.Since(cachedLive.LastUpdated) < 5*time.Minute {
			return &cachedLive, nil
		}
	}

	// Fetch live data from DataGolf API using in-play predictions
	url := fmt.Sprintf("%s/preds/in-play?tour=pga&file_format=json&key=%s",
		c.baseURL, c.apiKey)

	var liveResponse LivePredictionsResponse

	if err := c.makeRequest(url, &liveResponse); err != nil {
		return nil, fmt.Errorf("failed to fetch live tournament data: %w", err)
	}

	// Convert to our LiveTournamentData format
	var leaderboard []LiveLeaderboardEntry
	for _, pred := range liveResponse.Predictions {
		entry := LiveLeaderboardEntry{
			PlayerID:          pred.DGID,
			PlayerName:        pred.PlayerName,
			Position:          0, // TODO: Parse position from string
			TotalScore:        0, // TODO: Parse score from string
			ThruHoles:         0, // TODO: Parse thru from string
			RoundScore:        0, // TODO: Calculate from total score
			MovementIndicator: "", // TODO: Calculate movement
			TeeTime:          "",  // Not available in live predictions
			IsOnCourse:       pred.Thru != "F",
		}
		leaderboard = append(leaderboard, entry)
	}

	response := LiveTournamentData{
		TournamentID:    tournamentID,
		CurrentRound:    1, // TODO: Determine current round
		CutLine:         1, // TODO: Calculate cut line
		CutMade:         false, // TODO: Determine if cut has been made
		LeaderScore:     -12, // TODO: Calculate leader score
		LastUpdated:     time.Now(),
		LiveLeaderboard: leaderboard,
		WeatherUpdate: WeatherConditions{
			Temperature: 75,
			WindSpeed:   10,
			WindDir:     "SW",
			Conditions:  "partly_cloudy",
			Humidity:    65,
		},
		PlaySuspended: false, // TODO: Determine from tournament status
	}

	// Cache for very short duration - 2 minutes for live data
	c.cache.SetSimple(cacheKey, response, 2*time.Minute)

	return &response, nil
}

// GetPlayerCourseHistory fetches historical performance data for a player at a specific course
func (c *DataGolfClient) GetPlayerCourseHistory(playerID, courseID string) (*PlayerCourseHistory, error) {
	cacheKey := fmt.Sprintf("datagolf:course_history:%s:%s", playerID, courseID)

	// Check cache first - course history changes infrequently
	var cachedHistory PlayerCourseHistory
	err := c.cache.GetSimple(cacheKey, &cachedHistory)
	if err == nil {
		return &cachedHistory, nil
	}

	// Fetch from DataGolf API
	url := fmt.Sprintf("%s/player-course-history?player_id=%s&course_id=%s&key=%s",
		c.baseURL, playerID, courseID, c.apiKey)

	var response PlayerCourseHistory

	if err := c.makeRequest(url, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch player course history: %w", err)
	}

	// Cache for 7 days - historical data changes infrequently
	c.cache.SetSimple(cacheKey, response, 7*24*time.Hour)

	return &response, nil
}

// GetWeatherImpactData fetches weather impact analysis for tournament optimization
func (c *DataGolfClient) GetWeatherImpactData(tournamentID string) (*WeatherImpactAnalysis, error) {
	cacheKey := fmt.Sprintf("datagolf:weather_impact:%s", tournamentID)

	// Check cache first
	var cachedWeather WeatherImpactAnalysis
	err := c.cache.GetSimple(cacheKey, &cachedWeather)
	if err == nil {
		// Return cached if it's recent (< 2 hours)
		if time.Since(cachedWeather.AnalysisDate) < 2*time.Hour {
			return &cachedWeather, nil
		}
	}

	// Fetch weather impact from DataGolf API
	url := fmt.Sprintf("%s/weather-impact-analysis?tour=pga&event_id=%s&key=%s",
		c.baseURL, tournamentID, c.apiKey)

	var response WeatherImpactAnalysis

	if err := c.makeRequest(url, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch weather impact analysis: %w", err)
	}

	// Set analysis timestamp
	response.AnalysisDate = time.Now()

	// Cache for 2 hours - weather conditions can change frequently
	c.cache.SetSimple(cacheKey, response, 2*time.Hour)

	return &response, nil
}

// Enhanced helper methods

// isTournamentWeek determines if we're in an active tournament week for caching optimization
func (c *DataGolfClient) isTournamentWeek(tournamentID string) bool {
	// Get current tournament to check dates
	tournament, err := c.GetCurrentTournament()
	if err != nil {
		return false
	}

	now := time.Now()
	// Consider it tournament week if we're within 3 days of start or during tournament
	return tournament.ID == tournamentID &&
		   (now.After(tournament.StartDate.Add(-3*24*time.Hour)) &&
		    now.Before(tournament.EndDate.Add(24*time.Hour)))
}

// Enhanced cache warming for DataGolf advanced features
func (c *DataGolfClient) WarmAdvancedCache() error {
	c.logger.Info("Starting DataGolf advanced cache warming")

	// Get current tournament
	tournament, err := c.GetCurrentTournament()
	if err != nil {
		c.logger.WithError(err).Error("Failed to get tournament for advanced cache warming")
		return err
	}

	// Warm essential advanced data
	if tournament.Status == "in_progress" || tournament.Status == "scheduled" {
		// Pre-tournament predictions
		_, err = c.GetPreTournamentPredictions(tournament.ID)
		if err != nil {
			c.logger.WithError(err).Warn("Failed to warm predictions cache")
		}

		// Course analytics if we have course info
		if tournament.CourseID != "" {
			_, err = c.GetCourseAnalytics(tournament.CourseID)
			if err != nil {
				c.logger.WithError(err).Warn("Failed to warm course analytics cache")
			}
		}

		// Weather impact analysis
		_, err = c.GetWeatherImpactData(tournament.ID)
		if err != nil {
			c.logger.WithError(err).Warn("Failed to warm weather impact cache")
		}

		// Live data if tournament is in progress
		if tournament.Status == "in_progress" {
			_, err = c.GetLiveTournamentData(tournament.ID)
			if err != nil {
				c.logger.WithError(err).Warn("Failed to warm live tournament data cache")
			}
		}
	}

	c.logger.WithField("requests_used", c.requestTracker.requestCount).
		Info("DataGolf advanced cache warming completed")

	return nil
}

// GetDailyUsage returns current API usage stats
func (c *DataGolfClient) GetDailyUsage() (daily, monthly int, limit int) {
	c.requestTracker.mu.Lock()
	defer c.requestTracker.mu.Unlock()

	// DataGolf has much higher limits, so we return -1 for "unlimited"
	return c.requestTracker.requestCount, c.requestTracker.requestCount, -1
}
