package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/api"
	"github.com/jstittsworth/dfs-optimizer/internal/dfs"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/providers"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RapidAPIGolfIntegrationSuite struct {
	suite.Suite
	db          *database.DB
	cache       *services.CacheService
	router      *gin.Engine
	mockServer  *httptest.Server
	client      *providers.RapidAPIGolfClient
	redisClient *redis.Client
}

func (suite *RapidAPIGolfIntegrationSuite) SetupSuite() {
	// Set test environment
	os.Setenv("ENV", "test")
	os.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/dfs_optimizer_test?sslmode=disable")
	os.Setenv("REDIS_URL", "redis://localhost:6379/1")
	os.Setenv("RAPIDAPI_KEY", "test-rapidapi-key")

	// Load config
	cfg, err := config.LoadConfig()
	require.NoError(suite.T(), err)

	// Setup database
	suite.db, err = database.NewConnection(cfg.DatabaseURL, true)
	require.NoError(suite.T(), err)

	// Setup Redis
	opt, err := redis.ParseURL(cfg.RedisURL)
	require.NoError(suite.T(), err)
	suite.redisClient = redis.NewClient(opt)

	// Setup cache service
	suite.cache = services.NewCacheService(suite.redisClient)

	// Create mock RapidAPI server
	suite.setupMockServer()

	// Create RapidAPI client with mock server URL
	logger := logrus.New()
	suite.client = providers.NewRapidAPIGolfClient("test-rapidapi-key", suite.cache, logger)
	suite.client.baseURL = suite.mockServer.URL

	// Setup router
	suite.router = gin.New()
	suite.router.Use(gin.Recovery())

	// Setup routes with RapidAPI provider
	wsHub := services.NewWebSocketHub()
	go wsHub.Run()

	aggregator := services.NewDataAggregator(suite.db, suite.cache, logger, cfg.BallDontLieAPIKey)
	dataFetcher := services.NewDataFetcherService(suite.db, suite.cache, aggregator, logger, 2*time.Hour)

	apiV1 := suite.router.Group("/api/v1")
	api.SetupRoutes(apiV1, suite.db, suite.cache, wsHub, cfg, aggregator, dataFetcher)
}

func (suite *RapidAPIGolfIntegrationSuite) setupMockServer() {
	suite.mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify RapidAPI headers
		assert.Equal(suite.T(), "test-rapidapi-key", r.Header.Get("X-RapidAPI-Key"))
		assert.Equal(suite.T(), "live-golf-data.p.rapidapi.com", r.Header.Get("X-RapidAPI-Host"))

		switch r.URL.Path {
		case "/tournament":
			suite.serveTournamentData(w)
		case "/leaderboard":
			suite.serveLeaderboardData(w)
		case "/schedule":
			suite.serveScheduleData(w)
		case "/stats":
			suite.serveStatsData(w)
		case "/points":
			suite.servePointsData(w)
		case "/earnings":
			suite.serveEarningsData(w)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func (suite *RapidAPIGolfIntegrationSuite) serveTournamentData(w http.ResponseWriter) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"meta": map[string]interface{}{
			"title":       "Current Tournament",
			"description": "Live golf tournament data",
		},
		"results": map[string]interface{}{
			"tournament": map[string]interface{}{
				"id":            1001,
				"name":          "Integration Test Open",
				"country":       "USA",
				"course":        "Test National Golf Club",
				"start_date":    time.Now().Format("2006-01-02"),
				"end_date":      time.Now().Add(3 * 24 * time.Hour).Format("2006-01-02"),
				"prize_fund":    "$10,000,000",
				"fund_currency": "USD",
				"status":        "active",
				"tour_name":     "PGA Tour",
				"season":        2025,
				"timezone":      "America/New_York",
				"updated_at":    time.Now().Format(time.RFC3339),
			},
		},
	})
}

func (suite *RapidAPIGolfIntegrationSuite) serveLeaderboardData(w http.ResponseWriter) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"meta": map[string]interface{}{
			"title": "Tournament Leaderboard",
		},
		"results": map[string]interface{}{
			"leaderboard": []map[string]interface{}{
				{
					"player_id":    2001,
					"position":     1,
					"first_name":   "John",
					"last_name":    "Smith",
					"country":      "USA",
					"hole_num":     18,
					"start_hole":   1,
					"group_id":     1,
					"score":        -12,
					"strokes":      268,
					"prize_money":  "1,800,000",
					"fedex_points": 500,
					"rounds_breakdown": []map[string]interface{}{
						{"round_number": 1, "score_to_par": -3, "strokes": 67},
						{"round_number": 2, "score_to_par": -4, "strokes": 66},
						{"round_number": 3, "score_to_par": -2, "strokes": 68},
						{"round_number": 4, "score_to_par": -3, "strokes": 67},
					},
				},
				{
					"player_id":    2002,
					"position":     2,
					"first_name":   "Jane",
					"last_name":    "Doe",
					"country":      "CAN",
					"hole_num":     18,
					"score":        -10,
					"strokes":      270,
					"prize_money":  "1,080,000",
					"fedex_points": 300,
				},
			},
			"tournament": map[string]interface{}{
				"id":     1001,
				"status": "active",
			},
		},
	})
}

func (suite *RapidAPIGolfIntegrationSuite) serveScheduleData(w http.ResponseWriter) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": map[string]interface{}{
			"schedule": []map[string]interface{}{
				{
					"id":         1002,
					"name":       "Future Championship",
					"start_date": time.Now().Add(7 * 24 * time.Hour).Format("2006-01-02"),
					"end_date":   time.Now().Add(10 * 24 * time.Hour).Format("2006-01-02"),
					"status":     "upcoming",
					"prize_fund": "$8,000,000",
				},
			},
		},
	})
}

func (suite *RapidAPIGolfIntegrationSuite) serveStatsData(w http.ResponseWriter) {
	playerID := suite.T().Name() // Just for testing
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": map[string]interface{}{
			"stats": map[string]interface{}{
				"player_id": playerID,
				"season":    2025,
				"stats": map[string]interface{}{
					"scoring_average":      70.5,
					"driving_distance":     295.8,
					"driving_accuracy":     65.4,
					"greens_in_regulation": 72.1,
					"putts_per_round":      28.9,
				},
			},
		},
	})
}

func (suite *RapidAPIGolfIntegrationSuite) servePointsData(w http.ResponseWriter) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": map[string]interface{}{
			"points": []map[string]interface{}{
				{"player_id": 2001, "points": 500, "rank": 1},
				{"player_id": 2002, "points": 300, "rank": 2},
			},
		},
	})
}

func (suite *RapidAPIGolfIntegrationSuite) serveEarningsData(w http.ResponseWriter) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": map[string]interface{}{
			"earnings": []map[string]interface{}{
				{"player_id": 2001, "earnings": "$1,800,000", "position": 1},
				{"player_id": 2002, "earnings": "$1,080,000", "position": 2},
			},
		},
	})
}

func (suite *RapidAPIGolfIntegrationSuite) TearDownSuite() {
	if suite.mockServer != nil {
		suite.mockServer.Close()
	}
	if suite.redisClient != nil {
		suite.redisClient.FlushDB(suite.redisClient.Context())
		suite.redisClient.Close()
	}
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *RapidAPIGolfIntegrationSuite) SetupTest() {
	// Clean database before each test
	suite.db.Exec("TRUNCATE TABLE golf_tournaments, golf_tournament_players, golf_player_stats CASCADE")

	// Clear Redis cache
	suite.redisClient.FlushDB(suite.redisClient.Context())
}

func (suite *RapidAPIGolfIntegrationSuite) TestGetTournamentData() {
	tournament, err := suite.client.GetCurrentTournament()

	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), tournament)
	assert.Equal(suite.T(), "1001", tournament.ID)
	assert.Equal(suite.T(), "Integration Test Open", tournament.Name)
	assert.Equal(suite.T(), "Test National Golf Club", tournament.CourseName)
	assert.Equal(suite.T(), float64(10000000), tournament.Purse)
	assert.Equal(suite.T(), "in_progress", tournament.Status)
}

func (suite *RapidAPIGolfIntegrationSuite) TestGetPlayersData() {
	players, err := suite.client.GetPlayers(dfs.SportGolf, time.Now().Format("2006-01-02"))

	require.NoError(suite.T(), err)
	assert.Len(suite.T(), players, 2)

	// Check first player
	assert.Equal(suite.T(), "John Smith", players[0].Name)
	assert.Equal(suite.T(), "USA", players[0].Team)
	assert.Equal(suite.T(), "rapidapi_2001", players[0].ExternalID)
	assert.Equal(suite.T(), float64(-12), players[0].Stats["score"])
	assert.Equal(suite.T(), float64(500), players[0].Stats["fedex_points"])

	// Check second player
	assert.Equal(suite.T(), "Jane Doe", players[1].Name)
	assert.Equal(suite.T(), "CAN", players[1].Team)
}

func (suite *RapidAPIGolfIntegrationSuite) TestCacheBehavior() {
	// First call - should hit API
	_, err := suite.client.GetCurrentTournament()
	require.NoError(suite.T(), err)
	initialCount := suite.client.requestTracker.dailyCount

	// Second call - should hit cache
	_, err = suite.client.GetCurrentTournament()
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), initialCount, suite.client.requestTracker.dailyCount)
}

func (suite *RapidAPIGolfIntegrationSuite) TestDailyLimitEnforcement() {
	// Set daily count to limit
	suite.client.requestTracker.dailyCount = 19

	// This should succeed (20th request)
	_, err := suite.client.GetCurrentTournament()
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 20, suite.client.requestTracker.dailyCount)

	// This should fail and fall back to ESPN
	players, err := suite.client.GetPlayers(dfs.SportGolf, time.Now().Format("2006-01-02"))
	assert.NoError(suite.T(), err) // ESPN fallback returns empty but no error
	assert.Len(suite.T(), players, 0)
	assert.Equal(suite.T(), 20, suite.client.requestTracker.dailyCount) // No increase
}

func (suite *RapidAPIGolfIntegrationSuite) TestGolfAPIEndpoint() {
	// Create a tournament in the database
	tournament := models.GolfTournament{
		ExternalID: "1001",
		Name:       "Integration Test Open",
		StartDate:  time.Now(),
		EndDate:    time.Now().Add(3 * 24 * time.Hour),
		Status:     "in_progress",
		Purse:      10000000,
	}
	suite.db.Create(&tournament)

	// Test the API endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/golf/tournaments", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response struct {
		Tournaments []models.GolfTournament `json:"tournaments"`
		Total       int                     `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), response.Tournaments, 1)
	assert.Equal(suite.T(), "Integration Test Open", response.Tournaments[0].Name)
}

func (suite *RapidAPIGolfIntegrationSuite) TestGetTournamentSchedule() {
	schedule, err := suite.client.GetTournamentSchedule()

	require.NoError(suite.T(), err)
	assert.Len(suite.T(), schedule, 1)
	assert.Equal(suite.T(), "1002", schedule[0].ID)
	assert.Equal(suite.T(), "Future Championship", schedule[0].Name)
	assert.Equal(suite.T(), "scheduled", schedule[0].Status)
}

func (suite *RapidAPIGolfIntegrationSuite) TestWarmCache() {
	err := suite.client.WarmCache()

	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), suite.client.requestTracker.dailyCount, 2)

	// Verify data was cached
	var cachedTournament providers.GolfTournamentData
	err = suite.cache.GetSimple("rapidapi:golf:current_tournament", &cachedTournament)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "1001", cachedTournament.ID)
}

func TestRapidAPIGolfIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(RapidAPIGolfIntegrationSuite))
}
