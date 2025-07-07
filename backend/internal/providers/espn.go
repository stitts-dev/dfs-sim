package providers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/dfs"
	"github.com/sirupsen/logrus"
)

// ESPNClient implements the Provider interface for ESPN API
type ESPNClient struct {
	httpClient *http.Client
	cache      dfs.CacheProvider
	logger     *logrus.Logger
}

// NewESPNClient creates a new ESPN API client
func NewESPNClient(cache dfs.CacheProvider, logger *logrus.Logger) *ESPNClient {
	return &ESPNClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:  cache,
		logger: logger,
	}
}

// ESPN API response structures
type espnScoreboardResponse struct {
	Events []struct {
		ID           string `json:"id"`
		Date         string `json:"date"`
		Name         string `json:"name"`
		Competitions []struct {
			ID         string `json:"id"`
			Competitors []struct {
				ID         string `json:"id"`
				HomeAway   string `json:"homeAway"`
				Team       espnTeam `json:"team"`
				Statistics []map[string]interface{} `json:"statistics"`
			} `json:"competitors"`
		} `json:"competitions"`
	} `json:"events"`
}

type espnTeam struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	DisplayName  string `json:"displayName"`
}

type espnTeamResponse struct {
	Team struct {
		ID           string `json:"id"`
		Location     string `json:"location"`
		Name         string `json:"name"`
		Abbreviation string `json:"abbreviation"`
		DisplayName  string `json:"displayName"`
		Athletes     []struct {
			ID          string `json:"id"`
			FirstName   string `json:"firstName"`
			LastName    string `json:"lastName"`
			FullName    string `json:"fullName"`
			DisplayName string `json:"displayName"`
			Jersey      string `json:"jersey"`
			Position    struct {
				Abbreviation string `json:"abbreviation"`
				Name         string `json:"name"`
			} `json:"position"`
			Headshot struct {
				Href string `json:"href"`
			} `json:"headshot"`
			Statistics struct {
				Splits struct {
					Categories []struct {
						Name  string `json:"name"`
						Stats []struct {
							Name  string  `json:"name"`
							Value float64 `json:"value"`
						} `json:"stats"`
					} `json:"categories"`
				} `json:"splits"`
			} `json:"statistics"`
		} `json:"athletes"`
	} `json:"team"`
}

// GetPlayers fetches players for a specific sport and date
func (c *ESPNClient) GetPlayers(sport dfs.Sport, date string) ([]dfs.PlayerData, error) {
	cacheKey := fmt.Sprintf("espn:%s:players:%s", sport, date)
	
	// Check cache first
	var cachedPlayers []dfs.PlayerData
	err := c.cache.GetSimple(cacheKey, &cachedPlayers)
	if err == nil {
		return cachedPlayers, nil
	}

	// Fetch scoreboard data to get active teams
	scoreboardURL := c.getScoreboardURL(sport)
	teams, err := c.fetchActiveTeams(scoreboardURL, date)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch active teams: %w", err)
	}

	// Fetch rosters for each team
	var allPlayers []dfs.PlayerData
	for _, teamID := range teams {
		roster, err := c.GetTeamRoster(sport, teamID)
		if err != nil {
			c.logger.Warnf("Failed to fetch roster for team %s: %v", teamID, err)
			continue
		}
		allPlayers = append(allPlayers, roster...)
	}

	// Cache for 15 minutes
	if len(allPlayers) > 0 {
		c.cache.SetSimple(cacheKey, allPlayers, 15*time.Minute)
	}

	return allPlayers, nil
}

// GetPlayer fetches a specific player
func (c *ESPNClient) GetPlayer(sport dfs.Sport, externalID string) (*dfs.PlayerData, error) {
	// ESPN doesn't have a direct player endpoint, would need to search through team rosters
	// This is a limitation of the ESPN API
	return nil, fmt.Errorf("direct player lookup not supported by ESPN API")
}

// GetTeamRoster fetches the roster for a specific team
func (c *ESPNClient) GetTeamRoster(sport dfs.Sport, teamID string) ([]dfs.PlayerData, error) {
	cacheKey := fmt.Sprintf("espn:%s:roster:%s", sport, teamID)
	
	// Check cache first
	var cachedRoster []dfs.PlayerData
	err := c.cache.GetSimple(cacheKey, &cachedRoster)
	if err == nil {
		return cachedRoster, nil
	}

	url := c.getTeamURL(sport, teamID)
	var teamResp espnTeamResponse
	
	err = c.makeRequest(url, &teamResp)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch team roster: %w", err)
	}

	var players []dfs.PlayerData
	for _, athlete := range teamResp.Team.Athletes {
		player := dfs.PlayerData{
			ExternalID:  athlete.ID,
			Name:        athlete.DisplayName,
			Team:        teamResp.Team.Abbreviation,
			Position:    athlete.Position.Abbreviation,
			Stats:       c.extractPlayerStats(athlete.Statistics),
			ImageURL:    athlete.Headshot.Href,
			LastUpdated: time.Now(),
			Source:      "espn",
		}
		players = append(players, player)
	}

	// Cache for 2 hours
	if len(players) > 0 {
		c.cache.SetSimple(cacheKey, players, 2*time.Hour)
	}

	return players, nil
}

// makeRequest performs HTTP request with exponential backoff
func (c *ESPNClient) makeRequest(url string, target interface{}) error {
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
		return fmt.Errorf("request failed after retries: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	defer resp.Body.Close()
	
	return json.NewDecoder(resp.Body).Decode(target)
}

// fetchActiveTeams gets team IDs from the scoreboard
func (c *ESPNClient) fetchActiveTeams(scoreboardURL, date string) ([]string, error) {
	url := scoreboardURL
	if date != "" {
		url = fmt.Sprintf("%s?dates=%s", scoreboardURL, date)
	}

	var scoreboard espnScoreboardResponse
	err := c.makeRequest(url, &scoreboard)
	if err != nil {
		return nil, err
	}

	teamMap := make(map[string]bool)
	for _, event := range scoreboard.Events {
		for _, competition := range event.Competitions {
			for _, competitor := range competition.Competitors {
				teamMap[competitor.Team.ID] = true
			}
		}
	}

	teams := make([]string, 0, len(teamMap))
	for teamID := range teamMap {
		teams = append(teams, teamID)
	}

	return teams, nil
}

// extractPlayerStats converts ESPN statistics to our format
func (c *ESPNClient) extractPlayerStats(statistics interface{}) map[string]float64 {
	stats := make(map[string]float64)
	
	// ESPN stats structure is complex and varies by sport
	// This is a simplified extraction
	// In a real implementation, you'd parse the nested statistics structure
	
	return stats
}

// getScoreboardURL returns the scoreboard URL for a sport
func (c *ESPNClient) getScoreboardURL(sport dfs.Sport) string {
	baseURL := "http://site.api.espn.com/apis/site/v2/sports"
	
	switch sport {
	case dfs.SportNBA:
		return fmt.Sprintf("%s/basketball/nba/scoreboard", baseURL)
	case dfs.SportNFL:
		return fmt.Sprintf("%s/football/nfl/scoreboard", baseURL)
	case dfs.SportMLB:
		return fmt.Sprintf("%s/baseball/mlb/scoreboard", baseURL)
	default:
		return ""
	}
}

// getTeamURL returns the team detail URL for a sport
func (c *ESPNClient) getTeamURL(sport dfs.Sport, teamID string) string {
	baseURL := "http://site.api.espn.com/apis/site/v2/sports"
	
	switch sport {
	case dfs.SportNBA:
		return fmt.Sprintf("%s/basketball/nba/teams/%s?enable=roster,stats", baseURL, teamID)
	case dfs.SportNFL:
		return fmt.Sprintf("%s/football/nfl/teams/%s?enable=roster,stats", baseURL, teamID)
	case dfs.SportMLB:
		return fmt.Sprintf("%s/baseball/mlb/teams/%s?enable=roster,stats", baseURL, teamID)
	default:
		return ""
	}
}