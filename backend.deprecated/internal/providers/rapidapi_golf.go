package providers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/dfs"
	"github.com/sirupsen/logrus"
)

// RapidAPIGolfClient implements golf-specific data fetching from RapidAPI Live Golf Data
type RapidAPIGolfClient struct {
	httpClient     *http.Client
	cache          dfs.CacheProvider
	logger         *logrus.Logger
	apiKey         string
	apiHost        string
	baseURL        string
	rateLimiter    *time.Ticker
	retryAttempts  int
	requestTracker *RateLimitTracker
	espnFallback   *ESPNGolfClient
	mu             sync.Mutex
}

// RateLimitTracker tracks daily and monthly API usage
type RateLimitTracker struct {
	mu           sync.Mutex
	dailyCount   int
	monthlyCount int
	lastReset    time.Time
	dailyLimit   int
	monthlyLimit int
}

// NewRapidAPIGolfClient creates a new RapidAPI Golf client with rate limiting
func NewRapidAPIGolfClient(apiKey string, cache dfs.CacheProvider, logger *logrus.Logger) *RapidAPIGolfClient {
	// Create ESPN fallback client
	espnFallback := NewESPNGolfClient(cache, logger)

	// Initialize rate limit tracker
	tracker := &RateLimitTracker{
		dailyLimit:   20,  // Basic plan limit
		monthlyLimit: 250, // Basic plan limit
		lastReset:    time.Now(),
	}

	return &RapidAPIGolfClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:          cache,
		logger:         logger,
		apiKey:         apiKey,
		apiHost:        "live-golf-data.p.rapidapi.com",
		baseURL:        "https://live-golf-data.p.rapidapi.com",
		rateLimiter:    time.NewTicker(3 * time.Second), // 1 request per 3 seconds (safe for 20/day)
		retryAttempts:  3,
		requestTracker: tracker,
		espnFallback:   espnFallback,
	}
}

// RapidAPI Response Structures
type rapidAPITournamentResponse struct {
	Meta struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"meta"`
	Results struct {
		Tournament rapidAPITournament `json:"tournament"`
	} `json:"results"`
}

type rapidAPITournament struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Country      string    `json:"country"`
	Course       string    `json:"course"`
	StartDate    string    `json:"start_date"`
	EndDate      string    `json:"end_date"`
	Prize        string    `json:"prize_fund"`
	FundCurrency string    `json:"fund_currency"`
	Status       string    `json:"status"`
	Tour         string    `json:"tour_name"`
	Season       int       `json:"season"`
	Timezone     string    `json:"timezone"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type rapidAPILeaderboardResponse struct {
	Meta struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"meta"`
	Results struct {
		Leaderboard []rapidAPILeaderboardEntry `json:"leaderboard"`
		Tournament  rapidAPITournament         `json:"tournament"`
	} `json:"results"`
}

type rapidAPILeaderboardEntry struct {
	PlayerID    int             `json:"player_id"`
	Position    int             `json:"position"`
	FirstName   string          `json:"first_name"`
	LastName    string          `json:"last_name"`
	Country     string          `json:"country"`
	HoleNum     int             `json:"hole_num"`
	StartHole   int             `json:"start_hole"`
	GroupID     int             `json:"group_id"`
	StartRound  int             `json:"start_round"`
	CourseName  string          `json:"course_name"`
	Score       int             `json:"score"`
	Strokes     int             `json:"strokes"`
	UpdatedAt   string          `json:"updated_at"`
	PrizeMoney  string          `json:"prize_money"`
	FedexPoints int             `json:"fedex_points"`
	Rounds      []rapidAPIRound `json:"rounds_breakdown"`
}

type rapidAPIRound struct {
	RoundNumber int    `json:"round_number"`
	CourseName  string `json:"course_name"`
	Score       int    `json:"score_to_par"`
	Strokes     int    `json:"strokes"`
}

type rapidAPIPlayerStats struct {
	PlayerID  int                    `json:"player_id"`
	Season    int                    `json:"season"`
	Stats     map[string]interface{} `json:"stats"`
	UpdatedAt string                 `json:"updated_at"`
}

// GetPlayers fetches all players for a golf tournament using cache-first strategy
func (c *RapidAPIGolfClient) GetPlayers(sport dfs.Sport, date string) ([]dfs.PlayerData, error) {
	if sport != dfs.SportGolf {
		return nil, fmt.Errorf("RapidAPIGolfClient only supports golf")
	}

	cacheKey := fmt.Sprintf("rapidapi:golf:players:%s", date)

	// ALWAYS check cache first (critical for Basic plan)
	var cachedPlayers []dfs.PlayerData
	err := c.cache.GetSimple(cacheKey, &cachedPlayers)
	if err == nil {
		c.logger.WithField("source", "cache").Info("Returning cached golf players")
		return cachedPlayers, nil
	}

	// Check if we've hit our daily limit
	if c.isOverDailyLimit() {
		c.logger.Warn("RapidAPI daily limit reached, falling back to ESPN")
		return c.espnFallback.GetPlayers(sport, date)
	}

	// Get current tournament first
	tournament, err := c.GetCurrentTournament()
	if err != nil {
		c.logger.WithError(err).Warn("Failed to get tournament, falling back to ESPN")
		return c.espnFallback.GetPlayers(sport, date)
	}

	// Fetch leaderboard (more efficient than /players endpoint)
	leaderboard, err := c.getLeaderboard(tournament.ID)
	if err != nil {
		c.logger.WithError(err).Warn("Failed to get leaderboard, falling back to ESPN")
		return c.espnFallback.GetPlayers(sport, date)
	}

	var players []dfs.PlayerData
	for _, entry := range leaderboard.Results.Leaderboard {
		player := dfs.PlayerData{
			ExternalID:  fmt.Sprintf("rapidapi_%d", entry.PlayerID),
			Name:        fmt.Sprintf("%s %s", entry.FirstName, entry.LastName),
			Team:        entry.Country,
			Position:    "G", // All golfers have position "G"
			Stats:       c.extractPlayerStats(entry),
			LastUpdated: time.Now(),
			Source:      "rapidapi_golf",
		}
		players = append(players, player)
	}

	// Cache for extended duration (2 hours active, 24 hours completed)
	cacheDuration := 2 * time.Hour
	if tournament.Status == "completed" {
		cacheDuration = 24 * time.Hour
	}

	if len(players) > 0 {
		c.cache.SetSimple(cacheKey, players, cacheDuration)
		c.logger.WithField("count", len(players)).Info("Cached golf players from RapidAPI")
	}

	return players, nil
}

// GetPlayer fetches a specific golf player - return from cached data to save requests
func (c *RapidAPIGolfClient) GetPlayer(sport dfs.Sport, externalID string) (*dfs.PlayerData, error) {
	// Try to get from cached players first
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

// GetTeamRoster - for golf, this returns players by country
func (c *RapidAPIGolfClient) GetTeamRoster(sport dfs.Sport, teamID string) ([]dfs.PlayerData, error) {
	// For golf, teamID would be a country code
	players, err := c.GetPlayers(sport, time.Now().Format("2006-01-02"))
	if err != nil {
		return nil, err
	}

	var teamPlayers []dfs.PlayerData
	for _, player := range players {
		if player.Team == teamID {
			teamPlayers = append(teamPlayers, player)
		}
	}

	return teamPlayers, nil
}

// GetCurrentTournament fetches the current tournament details with aggressive caching
func (c *RapidAPIGolfClient) GetCurrentTournament() (*GolfTournamentData, error) {
	cacheKey := "rapidapi:golf:current_tournament"

	// Check cache first (24 hour TTL for tournament data)
	var cachedTournament GolfTournamentData
	err := c.cache.GetSimple(cacheKey, &cachedTournament)
	if err == nil {
		return &cachedTournament, nil
	}

	// Check rate limit
	if c.isOverDailyLimit() {
		c.logger.Warn("RapidAPI daily limit reached, using ESPN fallback")
		return c.espnFallback.GetCurrentTournament()
	}

	// Make API request
	url := fmt.Sprintf("%s/tournament", c.baseURL)
	var response rapidAPITournamentResponse

	if err := c.makeRequest(url, &response); err != nil {
		c.logger.WithError(err).Error("Failed to fetch tournament from RapidAPI")
		return c.espnFallback.GetCurrentTournament()
	}

	tournament := &GolfTournamentData{
		ID:         strconv.Itoa(response.Results.Tournament.ID),
		Name:       response.Results.Tournament.Name,
		StartDate:  c.parseDate(response.Results.Tournament.StartDate),
		EndDate:    c.parseDate(response.Results.Tournament.EndDate),
		Status:     c.mapTournamentStatus(response.Results.Tournament.Status),
		CourseName: response.Results.Tournament.Course,
		Purse:      c.parsePrize(response.Results.Tournament.Prize),
	}

	// Cache for 24 hours (tournament data rarely changes)
	c.cache.SetSimple(cacheKey, tournament, 24*time.Hour)
	c.logger.WithField("tournament", tournament.Name).Info("Cached tournament data from RapidAPI")

	return tournament, nil
}

// getLeaderboard fetches the tournament leaderboard
func (c *RapidAPIGolfClient) getLeaderboard(tournamentID string) (*rapidAPILeaderboardResponse, error) {
	cacheKey := fmt.Sprintf("rapidapi:golf:leaderboard:%s", tournamentID)

	// Check cache first
	var cachedLeaderboard rapidAPILeaderboardResponse
	err := c.cache.GetSimple(cacheKey, &cachedLeaderboard)
	if err == nil {
		return &cachedLeaderboard, nil
	}

	// Check rate limit
	if c.isOverDailyLimit() {
		return nil, fmt.Errorf("daily API limit reached")
	}

	// Make API request
	url := fmt.Sprintf("%s/leaderboard", c.baseURL)
	var response rapidAPILeaderboardResponse

	if err := c.makeRequest(url, &response); err != nil {
		return nil, err
	}

	// Cache for 2 hours during active tournament, 24 hours if completed
	cacheDuration := 2 * time.Hour
	if response.Results.Tournament.Status == "completed" {
		cacheDuration = 24 * time.Hour
	}

	c.cache.SetSimple(cacheKey, response, cacheDuration)
	c.logger.Info("Cached leaderboard data from RapidAPI")

	return &response, nil
}

// makeRequest handles HTTP requests with rate limiting and retries
func (c *RapidAPIGolfClient) makeRequest(url string, target interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Rate limiting
	<-c.rateLimiter.C

	// Track request
	if err := c.trackRequest(); err != nil {
		return err
	}

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

		// Set required RapidAPI headers
		req.Header.Set("X-RapidAPI-Key", c.apiKey)
		req.Header.Set("X-RapidAPI-Host", c.apiHost)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		// Check rate limit headers
		c.updateRateLimitInfo(resp.Header)

		if resp.StatusCode == http.StatusOK {
			if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}
			return nil
		}

		// Handle specific error codes
		switch resp.StatusCode {
		case http.StatusForbidden:
			return fmt.Errorf("invalid API credentials")
		case http.StatusTooManyRequests:
			return fmt.Errorf("rate limit exceeded")
		default:
			lastErr = fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", c.retryAttempts, lastErr)
}

// trackRequest tracks API usage and checks limits
func (c *RapidAPIGolfClient) trackRequest() error {
	c.requestTracker.mu.Lock()
	defer c.requestTracker.mu.Unlock()

	// Reset daily counter if needed
	now := time.Now()
	if now.Day() != c.requestTracker.lastReset.Day() {
		c.requestTracker.dailyCount = 0
		c.requestTracker.lastReset = now
		c.logger.Info("Reset daily request counter")
	}

	// Check if we're at the limit
	if c.requestTracker.dailyCount >= c.requestTracker.dailyLimit {
		return fmt.Errorf("daily request limit reached (%d/%d)",
			c.requestTracker.dailyCount, c.requestTracker.dailyLimit)
	}

	// Warn when approaching limit
	if c.requestTracker.dailyCount >= 15 {
		c.logger.WithFields(logrus.Fields{
			"daily_count": c.requestTracker.dailyCount,
			"daily_limit": c.requestTracker.dailyLimit,
		}).Warn("Approaching daily API limit")
	}

	c.requestTracker.dailyCount++
	c.requestTracker.monthlyCount++

	c.logger.WithFields(logrus.Fields{
		"daily_count":   c.requestTracker.dailyCount,
		"monthly_count": c.requestTracker.monthlyCount,
	}).Debug("Tracked API request")

	return nil
}

// isOverDailyLimit checks if we've exceeded the daily limit
func (c *RapidAPIGolfClient) isOverDailyLimit() bool {
	c.requestTracker.mu.Lock()
	defer c.requestTracker.mu.Unlock()

	// Reset if it's a new day
	now := time.Now()
	if now.Day() != c.requestTracker.lastReset.Day() {
		c.requestTracker.dailyCount = 0
		c.requestTracker.lastReset = now
	}

	return c.requestTracker.dailyCount >= c.requestTracker.dailyLimit
}

// updateRateLimitInfo updates rate limit tracking from response headers
func (c *RapidAPIGolfClient) updateRateLimitInfo(headers http.Header) {
	if limit := headers.Get("x-ratelimit-requests-limit"); limit != "" {
		c.logger.WithField("limit", limit).Debug("Rate limit header")
	}
	if remaining := headers.Get("x-ratelimit-requests-remaining"); remaining != "" {
		c.logger.WithField("remaining", remaining).Debug("Rate limit remaining")
	}
}

// extractPlayerStats extracts stats from leaderboard entry
func (c *RapidAPIGolfClient) extractPlayerStats(entry rapidAPILeaderboardEntry) map[string]float64 {
	stats := make(map[string]float64)

	stats["position"] = float64(entry.Position)
	stats["score"] = float64(entry.Score)
	stats["strokes"] = float64(entry.Strokes)
	stats["holes_played"] = float64(entry.HoleNum)

	if entry.FedexPoints > 0 {
		stats["fedex_points"] = float64(entry.FedexPoints)
	}

	// Parse prize money if available
	if prizeMoney := c.parsePrizeMoney(entry.PrizeMoney); prizeMoney > 0 {
		stats["prize_money"] = prizeMoney
	}

	// Add round scores
	for i, round := range entry.Rounds {
		stats[fmt.Sprintf("round_%d_score", i+1)] = float64(round.Score)
		stats[fmt.Sprintf("round_%d_strokes", i+1)] = float64(round.Strokes)
	}

	return stats
}

// Helper functions
func (c *RapidAPIGolfClient) parseDate(dateStr string) time.Time {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Now()
	}
	return t
}

func (c *RapidAPIGolfClient) mapTournamentStatus(status string) string {
	switch strings.ToLower(status) {
	case "upcoming":
		return "scheduled"
	case "active", "in progress":
		return "in_progress"
	case "finished", "complete":
		return "completed"
	default:
		return status
	}
}

func (c *RapidAPIGolfClient) parsePrize(prizeStr string) float64 {
	// Remove currency symbols and commas
	cleaned := strings.ReplaceAll(prizeStr, ",", "")
	cleaned = strings.TrimPrefix(cleaned, "$")
	cleaned = strings.TrimPrefix(cleaned, "€")
	cleaned = strings.TrimPrefix(cleaned, "£")

	prize, _ := strconv.ParseFloat(cleaned, 64)
	return prize
}

func (c *RapidAPIGolfClient) parsePrizeMoney(moneyStr string) float64 {
	if moneyStr == "" || moneyStr == "0" {
		return 0
	}
	return c.parsePrize(moneyStr)
}

// GetTournamentSchedule fetches the tournament schedule with aggressive caching
func (c *RapidAPIGolfClient) GetTournamentSchedule() ([]GolfTournamentData, error) {
	cacheKey := "rapidapi:golf:schedule"

	// Check cache first (7 day TTL - schedule rarely changes)
	var cachedSchedule []GolfTournamentData
	err := c.cache.GetSimple(cacheKey, &cachedSchedule)
	if err == nil {
		c.logger.Info("Returning cached tournament schedule")
		return cachedSchedule, nil
	}

	// Check rate limit
	if c.isOverDailyLimit() {
		c.logger.Warn("RapidAPI daily limit reached for schedule")
		return nil, fmt.Errorf("daily API limit reached")
	}

	// Make API request
	url := fmt.Sprintf("%s/schedule", c.baseURL)
	var response struct {
		Results struct {
			Schedule []rapidAPITournament `json:"schedule"`
		} `json:"results"`
	}

	if err := c.makeRequest(url, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch schedule: %w", err)
	}

	var schedule []GolfTournamentData
	for _, t := range response.Results.Schedule {
		tournament := GolfTournamentData{
			ID:        strconv.Itoa(t.ID),
			Name:      t.Name,
			StartDate: c.parseDate(t.StartDate),
			EndDate:   c.parseDate(t.EndDate),
			Status:    c.mapTournamentStatus(t.Status),
			Purse:     c.parsePrize(t.Prize),
		}
		schedule = append(schedule, tournament)
	}

	// Cache for 7 days
	c.cache.SetSimple(cacheKey, schedule, 7*24*time.Hour)
	c.logger.WithField("count", len(schedule)).Info("Cached tournament schedule")

	return schedule, nil
}

// GetPlayerStats fetches player statistics with long-term caching
func (c *RapidAPIGolfClient) GetPlayerStats(playerID string, year int) (*rapidAPIPlayerStats, error) {
	cacheKey := fmt.Sprintf("rapidapi:golf:stats:%s:%d", playerID, year)

	// Check cache first (7 day TTL - stats change infrequently)
	var cachedStats rapidAPIPlayerStats
	err := c.cache.GetSimple(cacheKey, &cachedStats)
	if err == nil {
		return &cachedStats, nil
	}

	// Only fetch if we have requests available
	if c.isOverDailyLimit() {
		return nil, fmt.Errorf("daily API limit reached")
	}

	// Make API request
	url := fmt.Sprintf("%s/stats?player_id=%s&year=%d", c.baseURL, playerID, year)
	var response struct {
		Results struct {
			Stats rapidAPIPlayerStats `json:"stats"`
		} `json:"results"`
	}

	if err := c.makeRequest(url, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch player stats: %w", err)
	}

	// Cache for 7 days
	c.cache.SetSimple(cacheKey, response.Results.Stats, 7*24*time.Hour)

	return &response.Results.Stats, nil
}

// GetScorecard fetches hole-by-hole scorecard (only for specific user requests)
func (c *RapidAPIGolfClient) GetScorecard(tournamentID string, playerID string) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("rapidapi:golf:scorecard:%s:%s", tournamentID, playerID)

	// Check cache first
	var cachedScorecard map[string]interface{}
	err := c.cache.GetSimple(cacheKey, &cachedScorecard)
	if err == nil {
		return cachedScorecard, nil
	}

	// Only fetch if explicitly needed and we have requests
	if c.isOverDailyLimit() {
		return nil, fmt.Errorf("daily API limit reached")
	}

	// Warn about using limited requests
	c.logger.WithFields(logrus.Fields{
		"tournament_id": tournamentID,
		"player_id":     playerID,
		"daily_usage":   c.requestTracker.dailyCount,
	}).Warn("Using limited API request for scorecard")

	// Make API request
	url := fmt.Sprintf("%s/scorecard?tournament_id=%s&player_id=%s",
		c.baseURL, tournamentID, playerID)
	var response map[string]interface{}

	if err := c.makeRequest(url, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch scorecard: %w", err)
	}

	// Cache for 1 hour (active) or 7 days (completed)
	cacheDuration := 1 * time.Hour
	if tournament, _ := c.GetCurrentTournament(); tournament != nil && tournament.Status == "completed" {
		cacheDuration = 7 * 24 * time.Hour
	}

	c.cache.SetSimple(cacheKey, response, cacheDuration)

	return response, nil
}

// GetFedExPoints fetches FedEx Cup points with caching
func (c *RapidAPIGolfClient) GetFedExPoints(tournamentID string) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("rapidapi:golf:points:%s", tournamentID)

	// Check cache first (6 hours active, 24 hours completed)
	var cachedPoints map[string]interface{}
	err := c.cache.GetSimple(cacheKey, &cachedPoints)
	if err == nil {
		return cachedPoints, nil
	}

	// Check rate limit
	if c.isOverDailyLimit() {
		return nil, fmt.Errorf("daily API limit reached")
	}

	// Make API request
	url := fmt.Sprintf("%s/points?tournament_id=%s", c.baseURL, tournamentID)
	var response map[string]interface{}

	if err := c.makeRequest(url, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch points: %w", err)
	}

	// Cache based on tournament status
	cacheDuration := 6 * time.Hour
	if tournament, _ := c.GetCurrentTournament(); tournament != nil && tournament.Status == "completed" {
		cacheDuration = 24 * time.Hour
	}

	c.cache.SetSimple(cacheKey, response, cacheDuration)

	return response, nil
}

// GetPrizeMoney fetches prize money distribution
func (c *RapidAPIGolfClient) GetPrizeMoney(tournamentID string) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("rapidapi:golf:earnings:%s", tournamentID)

	// Check cache first
	var cachedEarnings map[string]interface{}
	err := c.cache.GetSimple(cacheKey, &cachedEarnings)
	if err == nil {
		return cachedEarnings, nil
	}

	// Check rate limit
	if c.isOverDailyLimit() {
		return nil, fmt.Errorf("daily API limit reached")
	}

	// Make API request
	url := fmt.Sprintf("%s/earnings?tournament_id=%s", c.baseURL, tournamentID)
	var response map[string]interface{}

	if err := c.makeRequest(url, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch earnings: %w", err)
	}

	// Cache for appropriate duration
	cacheDuration := 6 * time.Hour
	if tournament, _ := c.GetCurrentTournament(); tournament != nil && tournament.Status == "completed" {
		cacheDuration = 24 * time.Hour
	}

	c.cache.SetSimple(cacheKey, response, cacheDuration)

	return response, nil
}

// GetOrganizations fetches golf tour organizations with long-term caching
func (c *RapidAPIGolfClient) GetOrganizations() (map[string]interface{}, error) {
	cacheKey := "rapidapi:golf:organizations"

	// Check cache first (30 day TTL - static data)
	var cachedOrgs map[string]interface{}
	err := c.cache.GetSimple(cacheKey, &cachedOrgs)
	if err == nil {
		return cachedOrgs, nil
	}

	// Only fetch if we have spare requests
	if c.requestTracker.dailyCount >= 18 {
		c.logger.Info("Skipping organizations fetch to preserve API quota")
		return nil, fmt.Errorf("preserving API quota")
	}

	// Make API request
	url := fmt.Sprintf("%s/organizations", c.baseURL)
	var response map[string]interface{}

	if err := c.makeRequest(url, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch organizations: %w", err)
	}

	// Cache for 30 days (static data)
	c.cache.SetSimple(cacheKey, response, 30*24*time.Hour)

	return response, nil
}

// WarmCache performs morning cache warming with minimal API calls
func (c *RapidAPIGolfClient) WarmCache() error {
	c.logger.Info("Starting RapidAPI cache warming")

	// Priority 1: Current tournament (1 request)
	tournament, err := c.GetCurrentTournament()
	if err != nil {
		c.logger.WithError(err).Error("Failed to warm tournament cache")
		return err
	}

	// Priority 2: Leaderboard if tournament is active (1-2 requests)
	if tournament.Status == "in_progress" {
		_, err = c.GetPlayers(dfs.SportGolf, time.Now().Format("2006-01-02"))
		if err != nil {
			c.logger.WithError(err).Error("Failed to warm leaderboard cache")
		}
	}

	c.logger.WithField("requests_used", c.requestTracker.dailyCount).
		Info("Cache warming completed")

	return nil
}

// GetDailyUsage returns current API usage stats
func (c *RapidAPIGolfClient) GetDailyUsage() (daily, monthly int, limit int) {
	c.requestTracker.mu.Lock()
	defer c.requestTracker.mu.Unlock()

	return c.requestTracker.dailyCount, c.requestTracker.monthlyCount, c.requestTracker.dailyLimit
}
