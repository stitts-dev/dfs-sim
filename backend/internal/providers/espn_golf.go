package providers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/dfs"
	"github.com/sirupsen/logrus"
)

// ESPNGolfClient implements golf-specific data fetching from ESPN
type ESPNGolfClient struct {
	httpClient  *http.Client
	cache       dfs.CacheProvider
	logger      *logrus.Logger
	baseURL     string
	rateLimiter *time.Ticker
}

// NewESPNGolfClient creates a new ESPN Golf API client
func NewESPNGolfClient(cache dfs.CacheProvider, logger *logrus.Logger) *ESPNGolfClient {
	return &ESPNGolfClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:       cache,
		logger:      logger,
		baseURL:     "https://site.api.espn.com/apis/site/v2/sports/golf",
		rateLimiter: time.NewTicker(time.Second), // 1 request per second
	}
}

// ESPN Golf API response structures
type espnGolfLeaderboardResponse struct {
	Events []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Date         string `json:"date"`
		EndDate      string `json:"endDate"`
		Status       espnEventStatus `json:"status"`
		Purse        string `json:"purse"`
		Competitions []struct {
			ID         string `json:"id"`
			Competitors []espnGolfCompetitor `json:"competitors"`
			Status     espnEventStatus `json:"status"`
			Format     struct {
				NumberOfRounds int `json:"numberOfRounds"`
			} `json:"format"`
			Course espnGolfCourse `json:"course"`
		} `json:"competitions"`
	} `json:"events"`
}

type espnEventStatus struct {
	Type struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		State       string `json:"state"`
		Completed   bool   `json:"completed"`
		Description string `json:"description"`
	} `json:"type"`
	Period int `json:"period"`
}

type espnGolfCompetitor struct {
	ID       string `json:"id"`
	Athlete  espnGolfAthlete `json:"athlete"`
	Status   string `json:"status"`
	Score    string `json:"score"`
	Position string `json:"position"`
	Movement string `json:"movement"`
	Rounds   []espnGolfRound `json:"rounds"`
	Stats    []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"statistics"`
}

type espnGolfAthlete struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	ShortName   string `json:"shortName"`
	Headshot    struct {
		Href string `json:"href"`
	} `json:"headshot"`
	Flag struct {
		Href string `json:"href"`
		Alt  string `json:"alt"`
	} `json:"flag"`
}

type espnGolfRound struct {
	Number     int    `json:"number"`
	Score      string `json:"score"`
	Strokes    int    `json:"strokes"`
	TeeTime    string `json:"teeTime"`
	ThruHoles  int    `json:"thru"`
}

type espnGolfCourse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	City     string `json:"city"`
	State    string `json:"state"`
	Nation   string `json:"nation"`
	Yardage  int    `json:"yardage"`
	Par      int    `json:"par"`
}

// GetPlayers fetches all players for a golf tournament
func (c *ESPNGolfClient) GetPlayers(sport dfs.Sport, date string) ([]dfs.PlayerData, error) {
	if sport != dfs.SportGolf {
		return nil, fmt.Errorf("ESPNGolfClient only supports golf")
	}

	cacheKey := fmt.Sprintf("espn:golf:players:%s", date)
	
	// Check cache first
	var cachedPlayers []dfs.PlayerData
	err := c.cache.GetSimple(cacheKey, &cachedPlayers)
	if err == nil {
		return cachedPlayers, nil
	}

	// Rate limiting
	<-c.rateLimiter.C

	// Fetch current tournament leaderboard
	url := fmt.Sprintf("%s/pga/leaderboard", c.baseURL)
	var leaderboard espnGolfLeaderboardResponse
	
	if err := c.makeRequest(url, &leaderboard); err != nil {
		return nil, fmt.Errorf("failed to fetch golf leaderboard: %w", err)
	}

	var players []dfs.PlayerData
	
	// Process the first (current) tournament
	if len(leaderboard.Events) > 0 && len(leaderboard.Events[0].Competitions) > 0 {
		competition := leaderboard.Events[0].Competitions[0]
		
		for _, competitor := range competition.Competitors {
			// Skip players who have withdrawn
			if competitor.Status == "withdrawn" {
				continue
			}
			
			player := dfs.PlayerData{
				ExternalID:  competitor.ID,
				Name:        competitor.Athlete.DisplayName,
				Team:        c.extractCountry(competitor.Athlete),
				Position:    "G", // All golfers have position "G"
				Stats:       c.extractGolfStats(competitor),
				ImageURL:    competitor.Athlete.Headshot.Href,
				LastUpdated: time.Now(),
				Source:      "espn_golf",
			}
			
			players = append(players, player)
		}
	}

	// Cache for 30 minutes (golf tournaments update less frequently)
	if len(players) > 0 {
		c.cache.SetSimple(cacheKey, players, 30*time.Minute)
	}

	return players, nil
}

// GetPlayer fetches a specific golf player (not directly supported by ESPN)
func (c *ESPNGolfClient) GetPlayer(sport dfs.Sport, externalID string) (*dfs.PlayerData, error) {
	return nil, fmt.Errorf("direct player lookup not supported by ESPN Golf API")
}

// GetTeamRoster - for golf, this returns players by country
func (c *ESPNGolfClient) GetTeamRoster(sport dfs.Sport, teamID string) ([]dfs.PlayerData, error) {
	// For golf, teamID would be a country code
	// We'll fetch all players and filter by country
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

// GetCurrentTournament fetches the current PGA tournament details
func (c *ESPNGolfClient) GetCurrentTournament() (*GolfTournamentData, error) {
	cacheKey := "espn:golf:current_tournament"
	
	// Check cache first
	var cachedTournament GolfTournamentData
	err := c.cache.GetSimple(cacheKey, &cachedTournament)
	if err == nil {
		return &cachedTournament, nil
	}

	// Rate limiting
	<-c.rateLimiter.C

	url := fmt.Sprintf("%s/pga/leaderboard", c.baseURL)
	var leaderboard espnGolfLeaderboardResponse
	
	if err := c.makeRequest(url, &leaderboard); err != nil {
		return nil, fmt.Errorf("failed to fetch tournament data: %w", err)
	}

	if len(leaderboard.Events) == 0 {
		return nil, fmt.Errorf("no active tournaments found")
	}

	event := leaderboard.Events[0]
	competition := event.Competitions[0]
	
	tournament := &GolfTournamentData{
		ID:           event.ID,
		Name:         event.Name,
		StartDate:    c.parseDate(event.Date),
		EndDate:      c.parseDate(event.EndDate),
		Status:       c.mapEventStatus(event.Status),
		CurrentRound: event.Status.Period,
		CourseID:     competition.Course.ID,
		CourseName:   competition.Course.Name,
		CoursePar:    competition.Course.Par,
		CourseYards:  competition.Course.Yardage,
		Purse:        c.parsePurse(event.Purse),
	}

	// Cache for 5 minutes during active tournament, 1 hour otherwise
	cacheDuration := 5 * time.Minute
	if tournament.Status == "completed" {
		cacheDuration = 1 * time.Hour
	}
	c.cache.SetSimple(cacheKey, tournament, cacheDuration)

	return tournament, nil
}

// Helper methods

func (c *ESPNGolfClient) makeRequest(url string, target interface{}) error {
	var resp *http.Response
	var err error
	
	// Implement exponential backoff
	for attempt := 0; attempt < 3; attempt++ {
		resp, err = c.httpClient.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		
		if resp != nil {
			resp.Body.Close()
		}
		
		// Exponential backoff
		waitTime := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		c.logger.Warnf("Request failed (attempt %d), waiting %v: %v", attempt+1, waitTime, err)
		time.Sleep(waitTime)
	}
	
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return json.NewDecoder(resp.Body).Decode(target)
}

func (c *ESPNGolfClient) extractGolfStats(competitor espnGolfCompetitor) map[string]float64 {
	stats := make(map[string]float64)
	
	// Parse score (relative to par)
	if competitor.Score != "" && competitor.Score != "E" {
		score, err := strconv.ParseFloat(strings.TrimPrefix(competitor.Score, "+"), 64)
		if err == nil {
			stats["score"] = score
		}
	}
	
	// Parse position
	if competitor.Position != "" && competitor.Position != "T" {
		position := strings.TrimPrefix(competitor.Position, "T")
		if pos, err := strconv.ParseFloat(position, 64); err == nil {
			stats["position"] = pos
		}
	}
	
	// Calculate total strokes from rounds
	totalStrokes := 0
	roundsPlayed := 0
	for _, round := range competitor.Rounds {
		if round.Strokes > 0 {
			totalStrokes += round.Strokes
			roundsPlayed++
		}
		
		// Store individual round scores
		stats[fmt.Sprintf("round_%d_score", round.Number)] = float64(round.Strokes)
	}
	
	if roundsPlayed > 0 {
		stats["total_strokes"] = float64(totalStrokes)
		stats["rounds_played"] = float64(roundsPlayed)
		stats["avg_score"] = float64(totalStrokes) / float64(roundsPlayed)
	}
	
	// Extract other statistics
	for _, stat := range competitor.Stats {
		if val, err := strconv.ParseFloat(stat.Value, 64); err == nil {
			stats[strings.ToLower(strings.ReplaceAll(stat.Name, " ", "_"))] = val
		}
	}
	
	return stats
}

func (c *ESPNGolfClient) extractCountry(athlete espnGolfAthlete) string {
	// Extract country from flag alt text
	if athlete.Flag.Alt != "" {
		return athlete.Flag.Alt
	}
	return "USA" // Default to USA if not specified
}

func (c *ESPNGolfClient) parseDate(dateStr string) time.Time {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		// Try alternative format
		t, _ = time.Parse("2006-01-02T15:04:05Z", dateStr)
	}
	return t
}

func (c *ESPNGolfClient) mapEventStatus(status espnEventStatus) string {
	switch status.Type.State {
	case "pre":
		return "scheduled"
	case "in":
		return "in_progress"
	case "post":
		return "completed"
	default:
		return status.Type.State
	}
}

func (c *ESPNGolfClient) parsePurse(purseStr string) float64 {
	// Remove $ and commas, then parse
	cleaned := strings.ReplaceAll(strings.TrimPrefix(purseStr, "$"), ",", "")
	purse, _ := strconv.ParseFloat(cleaned, 64)
	return purse
}

// GolfTournamentData represents tournament information
type GolfTournamentData struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	Status       string    `json:"status"`
	CurrentRound int       `json:"current_round"`
	CourseID     string    `json:"course_id"`
	CourseName   string    `json:"course_name"`
	CoursePar    int       `json:"course_par"`
	CourseYards  int       `json:"course_yards"`
	Purse        float64   `json:"purse"`
	CutLine      int       `json:"cut_line"`
}