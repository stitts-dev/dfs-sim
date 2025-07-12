package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/dfs"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// BallDontLieClient implements the Provider interface for BALLDONTLIE API
type BallDontLieClient struct {
	httpClient  *http.Client
	cache       dfs.CacheProvider
	logger      *logrus.Logger
	rateLimiter *rate.Limiter
	apiKey      string
}

// NewBallDontLieClient creates a new BALLDONTLIE API client
func NewBallDontLieClient(apiKey string, cache dfs.CacheProvider, logger *logrus.Logger) *BallDontLieClient {
	return &BallDontLieClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:       cache,
		logger:      logger,
		rateLimiter: rate.NewLimiter(rate.Every(12*time.Second), 1), // 5 requests per minute for free tier
		apiKey:      apiKey,
	}
}

// BALLDONTLIE API response structures
type ballDontLiePlayersResponse struct {
	Data []ballDontLiePlayer `json:"data"`
	Meta ballDontLieMeta     `json:"meta"`
}

type ballDontLiePlayer struct {
	ID           int             `json:"id"`
	FirstName    string          `json:"first_name"`
	LastName     string          `json:"last_name"`
	Position     string          `json:"position"`
	Height       string          `json:"height"`
	Weight       string          `json:"weight"`
	JerseyNumber string          `json:"jersey_number"`
	College      string          `json:"college"`
	Country      string          `json:"country"`
	DraftYear    int             `json:"draft_year"`
	DraftRound   int             `json:"draft_round"`
	DraftNumber  int             `json:"draft_number"`
	Team         ballDontLieTeam `json:"team"`
}

type ballDontLieTeam struct {
	ID           int    `json:"id"`
	Conference   string `json:"conference"`
	Division     string `json:"division"`
	City         string `json:"city"`
	Name         string `json:"name"`
	FullName     string `json:"full_name"`
	Abbreviation string `json:"abbreviation"`
}

type ballDontLieMeta struct {
	NextCursor string `json:"next_cursor"`
	PerPage    int    `json:"per_page"`
}

type ballDontLieStatsResponse struct {
	Data []ballDontLieStats `json:"data"`
	Meta ballDontLieMeta    `json:"meta"`
}

type ballDontLieStats struct {
	ID       int               `json:"id"`
	Player   ballDontLiePlayer `json:"player"`
	Game     ballDontLieGame   `json:"game"`
	Team     ballDontLieTeam   `json:"team"`
	Min      string            `json:"min"`
	Fgm      int               `json:"fgm"`
	Fga      int               `json:"fga"`
	FgPct    float64           `json:"fg_pct"`
	Fg3m     int               `json:"fg3m"`
	Fg3a     int               `json:"fg3a"`
	Fg3Pct   float64           `json:"fg3_pct"`
	Ftm      int               `json:"ftm"`
	Fta      int               `json:"fta"`
	FtPct    float64           `json:"ft_pct"`
	Oreb     int               `json:"oreb"`
	Dreb     int               `json:"dreb"`
	Reb      int               `json:"reb"`
	Ast      int               `json:"ast"`
	Stl      int               `json:"stl"`
	Blk      int               `json:"blk"`
	Turnover int               `json:"turnover"`
	Pf       int               `json:"pf"`
	Pts      int               `json:"pts"`
}

type ballDontLieGame struct {
	ID               int    `json:"id"`
	Date             string `json:"date"`
	Season           int    `json:"season"`
	Status           string `json:"status"`
	Period           int    `json:"period"`
	Time             string `json:"time"`
	Postseason       bool   `json:"postseason"`
	HomeTeamScore    int    `json:"home_team_score"`
	VisitorTeamScore int    `json:"visitor_team_score"`
}

type ballDontLieSeasonAveragesResponse struct {
	Data []ballDontLieSeasonAverage `json:"data"`
}

type ballDontLieSeasonAverage struct {
	PlayerID    int     `json:"player_id"`
	Season      int     `json:"season"`
	GamesPlayed int     `json:"games_played"`
	Min         string  `json:"min"`
	Fgm         float64 `json:"fgm"`
	Fga         float64 `json:"fga"`
	Fg3m        float64 `json:"fg3m"`
	Fg3a        float64 `json:"fg3a"`
	Ftm         float64 `json:"ftm"`
	Fta         float64 `json:"fta"`
	Oreb        float64 `json:"oreb"`
	Dreb        float64 `json:"dreb"`
	Reb         float64 `json:"reb"`
	Ast         float64 `json:"ast"`
	Stl         float64 `json:"stl"`
	Blk         float64 `json:"blk"`
	Turnover    float64 `json:"turnover"`
	Pf          float64 `json:"pf"`
	Pts         float64 `json:"pts"`
	FgPct       float64 `json:"fg_pct"`
	Fg3Pct      float64 `json:"fg3_pct"`
	FtPct       float64 `json:"ft_pct"`
}

// GetPlayers fetches NBA players (date parameter is ignored as API doesn't support date filtering)
func (c *BallDontLieClient) GetPlayers(sport dfs.Sport, date string) ([]dfs.PlayerData, error) {
	if sport != dfs.SportNBA {
		return nil, fmt.Errorf("BALLDONTLIE only supports NBA")
	}

	cacheKey := "balldontlie:nba:allplayers"

	// Check cache first
	var cachedPlayers []dfs.PlayerData
	err := c.cache.GetSimple(cacheKey, &cachedPlayers)
	if err == nil {
		return cachedPlayers, nil
	}

	// Fetch all active players (with pagination)
	var allPlayers []dfs.PlayerData
	cursor := ""

	for {
		// Rate limiting
		ctx := context.Background()
		err := c.rateLimiter.Wait(ctx)
		if err != nil {
			return nil, err
		}

		players, nextCursor, err := c.fetchPlayersPage(cursor)
		if err != nil {
			return nil, err
		}

		allPlayers = append(allPlayers, players...)

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	// Cache for 6 hours
	if len(allPlayers) > 0 {
		c.cache.SetSimple(cacheKey, allPlayers, 6*time.Hour)
	}

	return allPlayers, nil
}

// GetPlayer fetches a specific NBA player
func (c *BallDontLieClient) GetPlayer(sport dfs.Sport, externalID string) (*dfs.PlayerData, error) {
	if sport != dfs.SportNBA {
		return nil, fmt.Errorf("BALLDONTLIE only supports NBA")
	}

	cacheKey := fmt.Sprintf("balldontlie:player:%s", externalID)

	// Check cache first
	var cachedPlayer dfs.PlayerData
	err := c.cache.GetSimple(cacheKey, &cachedPlayer)
	if err == nil {
		return &cachedPlayer, nil
	}

	// Rate limiting
	ctx := context.Background()
	err = c.rateLimiter.Wait(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.balldontlie.io/v1/players/%s", externalID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var playerResp struct {
		Data ballDontLiePlayer `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&playerResp)
	if err != nil {
		return nil, err
	}

	// Get season averages for the player
	stats, err := c.getPlayerSeasonAverages(playerResp.Data.ID)
	if err != nil {
		c.logger.Warnf("Failed to get season averages for player %d: %v", playerResp.Data.ID, err)
	}

	playerData := c.convertToPlayerData(playerResp.Data, stats)

	// Cache for 24 hours
	c.cache.SetSimple(cacheKey, playerData, 24*time.Hour)

	return &playerData, nil
}

// GetTeamRoster fetches all players for a specific NBA team
func (c *BallDontLieClient) GetTeamRoster(sport dfs.Sport, teamID string) ([]dfs.PlayerData, error) {
	if sport != dfs.SportNBA {
		return nil, fmt.Errorf("BALLDONTLIE only supports NBA")
	}

	cacheKey := fmt.Sprintf("balldontlie:roster:%s", teamID)

	// Check cache first
	var cachedRoster []dfs.PlayerData
	err := c.cache.GetSimple(cacheKey, &cachedRoster)
	if err == nil {
		return cachedRoster, nil
	}

	// Search for players by team
	// Note: BALLDONTLIE API requires filtering players by team_ids parameter
	var roster []dfs.PlayerData
	cursor := ""

	for {
		// Rate limiting
		ctx := context.Background()
		err := c.rateLimiter.Wait(ctx)
		if err != nil {
			return nil, err
		}

		url := fmt.Sprintf("https://api.balldontlie.io/v1/players?team_ids[]=%s&per_page=100", teamID)
		if cursor != "" {
			url += fmt.Sprintf("&cursor=%s", cursor)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var playersResp ballDontLiePlayersResponse
		err = json.NewDecoder(resp.Body).Decode(&playersResp)
		if err != nil {
			return nil, err
		}

		for _, player := range playersResp.Data {
			roster = append(roster, c.convertToPlayerData(player, nil))
		}

		if playersResp.Meta.NextCursor == "" {
			break
		}
		cursor = playersResp.Meta.NextCursor
	}

	// Cache for 6 hours
	if len(roster) > 0 {
		c.cache.SetSimple(cacheKey, roster, 6*time.Hour)
	}

	return roster, nil
}

// fetchPlayersPage fetches a single page of players
func (c *BallDontLieClient) fetchPlayersPage(cursor string) ([]dfs.PlayerData, string, error) {
	url := "https://api.balldontlie.io/v1/players?per_page=100"
	if cursor != "" {
		url += fmt.Sprintf("&cursor=%s", cursor)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var playersResp ballDontLiePlayersResponse
	err = json.NewDecoder(resp.Body).Decode(&playersResp)
	if err != nil {
		return nil, "", err
	}

	var players []dfs.PlayerData
	for _, player := range playersResp.Data {
		players = append(players, c.convertToPlayerData(player, nil))
	}

	return players, playersResp.Meta.NextCursor, nil
}

// getPlayerSeasonAverages fetches season averages for a player
func (c *BallDontLieClient) getPlayerSeasonAverages(playerID int) (map[string]float64, error) {
	// Rate limiting
	ctx := context.Background()
	err := c.rateLimiter.Wait(ctx)
	if err != nil {
		return nil, err
	}

	currentYear := time.Now().Year()
	season := currentYear - 1 // NBA season typically runs Oct-June

	url := fmt.Sprintf("https://api.balldontlie.io/v1/season_averages?season=%d&player_ids[]=%d", season, playerID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var averagesResp ballDontLieSeasonAveragesResponse
	err = json.NewDecoder(resp.Body).Decode(&averagesResp)
	if err != nil {
		return nil, err
	}

	if len(averagesResp.Data) == 0 {
		return nil, nil
	}

	avg := averagesResp.Data[0]
	return map[string]float64{
		"pts":     avg.Pts,
		"reb":     avg.Reb,
		"ast":     avg.Ast,
		"stl":     avg.Stl,
		"blk":     avg.Blk,
		"fg_pct":  avg.FgPct,
		"fg3_pct": avg.Fg3Pct,
		"ft_pct":  avg.FtPct,
		"min":     c.parseMinutes(avg.Min),
		"games":   float64(avg.GamesPlayed),
	}, nil
}

// convertToPlayerData converts BALLDONTLIE player to our format
func (c *BallDontLieClient) convertToPlayerData(player ballDontLiePlayer, stats map[string]float64) dfs.PlayerData {
	if stats == nil {
		stats = make(map[string]float64)
	}

	// Normalize position with better distribution
	position := c.normalizePositionWithStats(player, stats)

	return dfs.PlayerData{
		ExternalID:  strconv.Itoa(player.ID),
		Name:        fmt.Sprintf("%s %s", player.FirstName, player.LastName),
		Team:        player.Team.Abbreviation,
		Position:    position,
		Stats:       stats,
		ImageURL:    "", // BALLDONTLIE doesn't provide player images
		LastUpdated: time.Now(),
		Source:      "balldontlie",
	}
}

// normalizePositionWithStats uses player stats to better determine position
func (c *BallDontLieClient) normalizePositionWithStats(player ballDontLiePlayer, stats map[string]float64) string {
	position := strings.ToUpper(player.Position)

	// If position is already specific, use it
	if position == "PG" || position == "SG" || position == "SF" || position == "PF" || position == "C" {
		return position
	}

	// For generic positions, use stats and player attributes
	switch position {
	case "G":
		// Use assists and player ID for distribution
		// High assist players are more likely PG
		if stats["ast"] > 0 {
			if stats["ast"] > 5.0 {
				return "PG"
			}
			return "SG"
		}
		// Use player ID for even distribution when no stats
		if player.ID%2 == 0 {
			return "PG"
		}
		return "SG"

	case "F":
		// Use rebounds and blocks to differentiate
		// Higher rebounds/blocks suggest PF
		if stats["reb"] > 0 || stats["blk"] > 0 {
			if stats["reb"] > 7.0 || stats["blk"] > 1.0 {
				return "PF"
			}
			return "SF"
		}
		// Use player ID for even distribution when no stats
		if player.ID%2 == 0 {
			return "SF"
		}
		return "PF"

	default:
		// Fall back to original normalization
		return c.normalizePosition(position)
	}
}

// normalizePosition converts position to standard format
func (c *BallDontLieClient) normalizePosition(position string) string {
	// BALLDONTLIE uses G, F, C format
	// We need to distribute generic positions better
	switch strings.ToUpper(position) {
	case "G":
		// Distribute guards between PG and SG
		// In production, we'd use player stats/role to determine
		// For now, alternate based on player ID or name hash
		return "PG" // Will be improved with smarter distribution
	case "F":
		// Distribute forwards between SF and PF
		return "SF" // Will be improved with smarter distribution
	case "C":
		return "C"
	case "G-F":
		return "SG" // Combo guard-forward typically plays SG
	case "F-C":
		return "PF" // Forward-center typically plays PF
	case "F-G":
		return "SF" // Forward-guard typically plays SF
	default:
		// Check for specific positions
		if strings.Contains(strings.ToUpper(position), "PG") {
			return "PG"
		} else if strings.Contains(strings.ToUpper(position), "SG") {
			return "SG"
		} else if strings.Contains(strings.ToUpper(position), "SF") {
			return "SF"
		} else if strings.Contains(strings.ToUpper(position), "PF") {
			return "PF"
		} else if strings.Contains(strings.ToUpper(position), "C") {
			return "C"
		}
		return position
	}
}

// parseMinutes converts "MM:SS" to float64 minutes
func (c *BallDontLieClient) parseMinutes(minStr string) float64 {
	parts := strings.Split(minStr, ":")
	if len(parts) != 2 {
		return 0
	}

	minutes, _ := strconv.ParseFloat(parts[0], 64)
	seconds, _ := strconv.ParseFloat(parts[1], 64)

	return minutes + (seconds / 60)
}
