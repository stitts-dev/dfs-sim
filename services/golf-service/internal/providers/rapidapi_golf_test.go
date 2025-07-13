package providers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/dfs"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCacheProvider implements a simple in-memory cache for testing
type MockCacheProvider struct {
	data map[string]interface{}
}

func NewMockCacheProvider() *MockCacheProvider {
	return &MockCacheProvider{
		data: make(map[string]interface{}),
	}
}

func (m *MockCacheProvider) SetSimple(key string, value interface{}, expiration time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *MockCacheProvider) GetSimple(key string, dest interface{}) error {
	val, exists := m.data[key]
	if !exists {
		return redis.Nil
	}

	// Marshal and unmarshal to simulate real cache behavior
	data, _ := json.Marshal(val)
	return json.Unmarshal(data, dest)
}

func TestNewRapidAPIGolfClient(t *testing.T) {
	cache := NewMockCacheProvider()
	logger := logrus.New()
	client := NewRapidAPIGolfClient("test-api-key", cache, logger)

	assert.NotNil(t, client)
	assert.Equal(t, "test-api-key", client.apiKey)
	assert.Equal(t, "live-golf-data.p.rapidapi.com", client.apiHost)
	assert.Equal(t, "https://live-golf-data.p.rapidapi.com", client.baseURL)
	assert.NotNil(t, client.espnFallback)
	assert.NotNil(t, client.requestTracker)
	assert.Equal(t, 20, client.requestTracker.dailyLimit)
	assert.Equal(t, 250, client.requestTracker.monthlyLimit)
}

func TestRapidAPIGolfClient_GetPlayers_CacheHit(t *testing.T) {
	cache := NewMockCacheProvider()
	logger := logrus.New()
	client := NewRapidAPIGolfClient("test-api-key", cache, logger)

	// Pre-populate cache
	expectedPlayers := []dfs.PlayerData{
		{
			ExternalID: "rapidapi_123",
			Name:       "Test Player",
			Team:       "USA",
			Position:   "G",
			Stats:      map[string]float64{"score": -5},
		},
	}
	cacheKey := "rapidapi:golf:players:2025-01-08"
	cache.SetSimple(cacheKey, expectedPlayers, 1*time.Hour)

	// Test cache hit
	players, err := client.GetPlayers(dfs.SportGolf, "2025-01-08")

	require.NoError(t, err)
	assert.Len(t, players, 1)
	assert.Equal(t, expectedPlayers[0].Name, players[0].Name)
	assert.Equal(t, 0, client.requestTracker.dailyCount) // No API call made
}

func TestRapidAPIGolfClient_GetPlayers_APIResponse(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Equal(t, "test-api-key", r.Header.Get("X-RapidAPI-Key"))
		assert.Equal(t, "live-golf-data.p.rapidapi.com", r.Header.Get("X-RapidAPI-Host"))

		// Handle different endpoints
		switch r.URL.Path {
		case "/tournament":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"results": map[string]interface{}{
					"tournament": map[string]interface{}{
						"id":         123,
						"name":       "Test Tournament",
						"start_date": "2025-01-08",
						"end_date":   "2025-01-11",
						"status":     "active",
					},
				},
			})
		case "/leaderboard":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"results": map[string]interface{}{
					"leaderboard": []map[string]interface{}{
						{
							"player_id":    456,
							"position":     1,
							"first_name":   "Tiger",
							"last_name":    "Woods",
							"country":      "USA",
							"score":        -10,
							"strokes":      270,
							"hole_num":     18,
							"fedex_points": 500,
						},
					},
					"tournament": map[string]interface{}{
						"status": "active",
					},
				},
			})
		}
	}))
	defer server.Close()

	cache := NewMockCacheProvider()
	logger := logrus.New()
	client := NewRapidAPIGolfClient("test-api-key", cache, logger)
	client.baseURL = server.URL

	players, err := client.GetPlayers(dfs.SportGolf, "2025-01-08")

	require.NoError(t, err)
	assert.Len(t, players, 1)
	assert.Equal(t, "Tiger Woods", players[0].Name)
	assert.Equal(t, "USA", players[0].Team)
	assert.Equal(t, "rapidapi_456", players[0].ExternalID)
	assert.Equal(t, float64(-10), players[0].Stats["score"])
	assert.Equal(t, float64(500), players[0].Stats["fedex_points"])
	assert.Equal(t, 2, client.requestTracker.dailyCount) // 2 API calls (tournament + leaderboard)
}

func TestRapidAPIGolfClient_DailyLimitExceeded(t *testing.T) {
	cache := NewMockCacheProvider()
	logger := logrus.New()
	client := NewRapidAPIGolfClient("test-api-key", cache, logger)

	// Manually set daily count to limit
	client.requestTracker.dailyCount = 20

	// Should fall back to ESPN
	players, err := client.GetPlayers(dfs.SportGolf, "2025-01-08")

	// ESPN mock returns empty, but no error
	assert.NoError(t, err)
	assert.Len(t, players, 0)
	assert.Equal(t, 20, client.requestTracker.dailyCount) // No additional calls
}

func TestRapidAPIGolfClient_GetCurrentTournament(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": map[string]interface{}{
				"tournament": map[string]interface{}{
					"id":            789,
					"name":          "Masters Tournament",
					"start_date":    "2025-04-10",
					"end_date":      "2025-04-13",
					"status":        "upcoming",
					"course":        "Augusta National",
					"prize_fund":    "$20,000,000",
					"fund_currency": "USD",
				},
			},
		})
	}))
	defer server.Close()

	cache := NewMockCacheProvider()
	logger := logrus.New()
	client := NewRapidAPIGolfClient("test-api-key", cache, logger)
	client.baseURL = server.URL

	tournament, err := client.GetCurrentTournament()

	require.NoError(t, err)
	assert.NotNil(t, tournament)
	assert.Equal(t, "789", tournament.ID)
	assert.Equal(t, "Masters Tournament", tournament.Name)
	assert.Equal(t, "Augusta National", tournament.CourseName)
	assert.Equal(t, float64(20000000), tournament.Purse)
	assert.Equal(t, "scheduled", tournament.Status)

	// Verify it was cached
	var cachedTournament GolfTournamentData
	err = cache.GetSimple("rapidapi:golf:current_tournament", &cachedTournament)
	assert.NoError(t, err)
	assert.Equal(t, tournament.ID, cachedTournament.ID)
}

func TestRapidAPIGolfClient_RetryLogic(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Success on 3rd attempt
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": map[string]interface{}{
				"tournament": map[string]interface{}{
					"id":     999,
					"name":   "Retry Test Tournament",
					"status": "active",
				},
			},
		})
	}))
	defer server.Close()

	cache := NewMockCacheProvider()
	logger := logrus.New()
	client := NewRapidAPIGolfClient("test-api-key", cache, logger)
	client.baseURL = server.URL

	tournament, err := client.GetCurrentTournament()

	require.NoError(t, err)
	assert.NotNil(t, tournament)
	assert.Equal(t, "999", tournament.ID)
	assert.Equal(t, 3, attempts) // Should have retried until success
}

func TestRapidAPIGolfClient_GetPlayer(t *testing.T) {
	cache := NewMockCacheProvider()
	logger := logrus.New()
	client := NewRapidAPIGolfClient("test-api-key", cache, logger)

	// Pre-populate cache with players
	players := []dfs.PlayerData{
		{ExternalID: "rapidapi_123", Name: "Player One"},
		{ExternalID: "rapidapi_456", Name: "Player Two"},
	}
	cache.SetSimple("rapidapi:golf:players:"+time.Now().Format("2006-01-02"), players, 1*time.Hour)

	player, err := client.GetPlayer(dfs.SportGolf, "rapidapi_456")

	require.NoError(t, err)
	assert.NotNil(t, player)
	assert.Equal(t, "Player Two", player.Name)
}

func TestRapidAPIGolfClient_GetTeamRoster(t *testing.T) {
	cache := NewMockCacheProvider()
	logger := logrus.New()
	client := NewRapidAPIGolfClient("test-api-key", cache, logger)

	// Pre-populate cache with players
	players := []dfs.PlayerData{
		{ExternalID: "1", Name: "US Player 1", Team: "USA"},
		{ExternalID: "2", Name: "US Player 2", Team: "USA"},
		{ExternalID: "3", Name: "UK Player", Team: "GBR"},
	}
	cache.SetSimple("rapidapi:golf:players:"+time.Now().Format("2006-01-02"), players, 1*time.Hour)

	teamPlayers, err := client.GetTeamRoster(dfs.SportGolf, "USA")

	require.NoError(t, err)
	assert.Len(t, teamPlayers, 2)
	assert.Equal(t, "US Player 1", teamPlayers[0].Name)
	assert.Equal(t, "US Player 2", teamPlayers[1].Name)
}

func TestRapidAPIGolfClient_WarmCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tournament":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"results": map[string]interface{}{
					"tournament": map[string]interface{}{
						"id":     111,
						"name":   "Warm Cache Tournament",
						"status": "in_progress",
					},
				},
			})
		case "/leaderboard":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"results": map[string]interface{}{
					"leaderboard": []map[string]interface{}{},
					"tournament": map[string]interface{}{
						"status": "in_progress",
					},
				},
			})
		}
	}))
	defer server.Close()

	cache := NewMockCacheProvider()
	logger := logrus.New()
	client := NewRapidAPIGolfClient("test-api-key", cache, logger)
	client.baseURL = server.URL

	err := client.WarmCache()

	require.NoError(t, err)
	assert.Equal(t, 2, client.requestTracker.dailyCount) // Tournament + Leaderboard

	// Verify data was cached
	var cachedTournament GolfTournamentData
	err = cache.GetSimple("rapidapi:golf:current_tournament", &cachedTournament)
	assert.NoError(t, err)
	assert.Equal(t, "111", cachedTournament.ID)
}

func TestRapidAPIGolfClient_GetDailyUsage(t *testing.T) {
	cache := NewMockCacheProvider()
	logger := logrus.New()
	client := NewRapidAPIGolfClient("test-api-key", cache, logger)

	// Make some tracking updates
	client.requestTracker.dailyCount = 15
	client.requestTracker.monthlyCount = 150

	daily, monthly, limit := client.GetDailyUsage()

	assert.Equal(t, 15, daily)
	assert.Equal(t, 150, monthly)
	assert.Equal(t, 20, limit)
}
