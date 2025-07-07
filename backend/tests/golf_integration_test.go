package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jstittsworth/dfs-optimizer/internal/api"
	"github.com/jstittsworth/dfs-optimizer/internal/api/handlers"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/optimizer"
	"github.com/jstittsworth/dfs-optimizer/internal/providers"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEnvironment(t *testing.T) (*gin.Engine, *database.DB, func()) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test database
	cfg := &config.Config{
		DatabaseURL: "postgres://postgres:postgres@localhost:5432/dfs_optimizer_test?sslmode=disable",
		JWTSecret:   "test-secret",
		Logger:      logrus.New(),
	}

	db, err := database.New(cfg.DatabaseURL)
	require.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(
		&models.Contest{},
		&models.Player{},
		&models.Lineup{},
		&models.GolfTournament{},
		&models.GolfPlayerEntry{},
		&models.GolfRoundScore{},
		&models.GolfCourseHistory{},
	)
	require.NoError(t, err)

	// Create test router
	router := gin.New()
	cache := services.NewCacheService(nil, cfg.Logger)
	wsHub := services.NewWebSocketHub(cfg.Logger)
	
	// Mock aggregator and data fetcher
	aggregator := &services.DataAggregator{}
	dataFetcher := &services.DataFetcherService{}

	// Setup routes
	apiGroup := router.Group("/api/v1")
	api.SetupRoutes(apiGroup, db, cache, wsHub, cfg, aggregator, dataFetcher)

	// Cleanup function
	cleanup := func() {
		// Drop test data
		db.Exec("DROP TABLE IF EXISTS golf_round_scores CASCADE")
		db.Exec("DROP TABLE IF EXISTS golf_player_entries CASCADE")
		db.Exec("DROP TABLE IF EXISTS golf_course_history CASCADE")
		db.Exec("DROP TABLE IF EXISTS golf_tournaments CASCADE")
		db.Close()
	}

	return router, db, cleanup
}

func TestGolfTournamentEndpoints(t *testing.T) {
	router, db, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create test tournament
	tournament := createTestTournament(t, db)

	t.Run("List Tournaments", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/golf/tournaments", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Tournaments []models.GolfTournament `json:"tournaments"`
			Total       int64                   `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(response.Tournaments), 1)
	})

	t.Run("Get Tournament", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/golf/tournaments/%s", tournament.ID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.GolfTournament
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, tournament.ID, response.ID)
		assert.Equal(t, tournament.Name, response.Name)
	})

	t.Run("Get Tournament Leaderboard", func(t *testing.T) {
		// Create test players and entries
		players := createTestPlayers(t, db, 10)
		createTestEntries(t, db, tournament, players)

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/golf/tournaments/%s/leaderboard", tournament.ID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Tournament models.GolfTournament    `json:"tournament"`
			Entries    []models.GolfPlayerEntry `json:"entries"`
			CutLine    int                      `json:"cut_line"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, tournament.ID, response.Tournament.ID)
		assert.GreaterOrEqual(t, len(response.Entries), 1)
	})
}

func TestGolfOptimization(t *testing.T) {
	router, db, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create test data
	tournament := createTestTournament(t, db)
	contest := createTestGolfContest(t, db, tournament)
	players := createTestPlayers(t, db, 50)
	createTestEntries(t, db, tournament, players)

	t.Run("Optimize Golf Lineup", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"contest_id":   contest.ID,
			"num_lineups":  5,
			"constraints": map[string]interface{}{
				"min_cut_probability": 0.5,
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/optimize", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Lineups []models.Lineup `json:"lineups"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Lineups, 5)

		// Validate each lineup
		for _, lineup := range response.Lineups {
			assert.Len(t, lineup.Players, 6) // Golf lineups have 6 players
			assert.LessOrEqual(t, lineup.TotalSalary, contest.SalaryCap)
			
			// All players should be golfers
			for _, player := range lineup.Players {
				assert.Equal(t, "G", player.Position)
			}
		}
	})
}

func TestGolfProjections(t *testing.T) {
	router, db, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tournament := createTestTournament(t, db)
	players := createTestPlayers(t, db, 20)
	createTestEntries(t, db, tournament, players)

	t.Run("Get Golf Projections", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/golf/tournaments/%s/projections", tournament.ID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Tournament   models.GolfTournament                      `json:"tournament"`
			Projections  map[string]*models.GolfProjection          `json:"projections"`
			Correlations map[string]map[string]float64              `json:"correlations"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotEmpty(t, response.Projections)
		assert.NotEmpty(t, response.Correlations)

		// Validate projections
		for playerID, projection := range response.Projections {
			assert.NotEmpty(t, playerID)
			assert.Greater(t, projection.CutProbability, 0.0)
			assert.LessOrEqual(t, projection.CutProbability, 1.0)
			assert.GreaterOrEqual(t, projection.DKPoints, 0.0)
		}
	})
}

func TestGolfCorrelations(t *testing.T) {
	// Test golf-specific correlations
	players := []models.Player{
		{ID: 1, Name: "Player 1", Team: "USA", Position: "G", Salary: 11000},
		{ID: 2, Name: "Player 2", Team: "USA", Position: "G", Salary: 10500},
		{ID: 3, Name: "Player 3", Team: "EUR", Position: "G", Salary: 9000},
		{ID: 4, Name: "Player 4", Team: "EUR", Position: "G", Salary: 8500},
	}

	builder := optimizer.NewGolfCorrelationBuilder(players)
	correlations := builder.BuildCorrelationMatrix()

	// Test same country correlation
	assert.Greater(t, correlations[1][2], 0.0, "Same country players should have positive correlation")
	
	// Test similar skill level correlation
	assert.Greater(t, correlations[1][2], correlations[1][4], "Similar salary players should have higher correlation")
}

func TestGolfStacking(t *testing.T) {
	players := []models.Player{
		{ID: 1, Name: "Star 1", Team: "USA", Position: "G", Salary: 11500, ProjectedPoints: 75, Ownership: 25},
		{ID: 2, Name: "Star 2", Team: "USA", Position: "G", Salary: 11000, ProjectedPoints: 72, Ownership: 22},
		{ID: 3, Name: "Mid 1", Team: "EUR", Position: "G", Salary: 8500, ProjectedPoints: 55, Ownership: 15},
		{ID: 4, Name: "Mid 2", Team: "EUR", Position: "G", Salary: 8000, ProjectedPoints: 52, Ownership: 12},
		{ID: 5, Name: "Value 1", Team: "USA", Position: "G", Salary: 6500, ProjectedPoints: 42, Ownership: 8},
		{ID: 6, Name: "Value 2", Team: "AUS", Position: "G", Salary: 6000, ProjectedPoints: 38, Ownership: 5},
	}

	stackBuilder := optimizer.NewStackBuilder(players, "golf")
	stacks := stackBuilder.GetOptimalStacks()

	assert.NotEmpty(t, stacks, "Should generate golf stacks")

	// Verify stack types
	hasCountryStack := false
	hasValueStack := false
	hasOwnershipStack := false

	for _, stack := range stacks {
		if stack.Type == optimizer.TeamStack {
			hasCountryStack = true
		}
		if stack.Type == optimizer.MiniStack {
			// Check if it's a value or ownership stack
			if len(stack.Players) >= 2 {
				salaryDiff := stack.Players[0].Salary - stack.Players[1].Salary
				if salaryDiff > 3000 {
					hasValueStack = true
				}
				
				ownershipDiff := stack.Players[0].Ownership - stack.Players[1].Ownership
				if ownershipDiff > 10 {
					hasOwnershipStack = true
				}
			}
		}
	}

	assert.True(t, hasCountryStack, "Should have country-based stacks")
	assert.True(t, hasValueStack || hasOwnershipStack, "Should have value or ownership-based stacks")
}

// Helper functions

func createTestTournament(t *testing.T, db *database.DB) *models.GolfTournament {
	tournament := &models.GolfTournament{
		ID:           uuid.New(),
		ExternalID:   "test-masters-2024",
		Name:         "Masters Tournament",
		StartDate:    time.Now(),
		EndDate:      time.Now().AddDate(0, 0, 4),
		Status:       models.TournamentScheduled,
		CourseName:   "Augusta National",
		CoursePar:    72,
		CourseYards:  7475,
		Purse:        15000000,
	}

	err := db.Create(tournament).Error
	require.NoError(t, err)
	return tournament
}

func createTestGolfContest(t *testing.T, db *database.DB, tournament *models.GolfTournament) *models.Contest {
	contest := &models.Contest{
		Name:        fmt.Sprintf("Golf GPP - %s", tournament.Name),
		Sport:       "golf",
		Platform:    "draftkings",
		ContestType: "gpp",
		EntryFee:    20,
		PrizePool:   100000,
		MaxEntries:  10000,
		SalaryCap:   50000,
		StartTime:   tournament.StartDate,
		IsActive:    true,
	}

	err := db.Create(contest).Error
	require.NoError(t, err)
	return contest
}

func createTestPlayers(t *testing.T, db *database.DB, count int) []models.Player {
	players := make([]models.Player, count)
	countries := []string{"USA", "EUR", "AUS", "JPN", "CAN"}

	for i := 0; i < count; i++ {
		player := models.Player{
			ExternalID:      fmt.Sprintf("golf-player-%d", i+1),
			Name:            fmt.Sprintf("Test Golfer %d", i+1),
			Team:            countries[i%len(countries)],
			Position:        "G",
			Sport:           "golf",
			Salary:          6000 + (i * 500), // Range from 6000 to high values
			ProjectedPoints: 40 + float64(i*2),
			FloorPoints:     30 + float64(i),
			CeilingPoints:   50 + float64(i*3),
			Ownership:       5 + float64(i%20),
		}
		
		err := db.Create(&player).Error
		require.NoError(t, err)
		players[i] = player
	}

	return players
}

func createTestEntries(t *testing.T, db *database.DB, tournament *models.GolfTournament, players []models.Player) {
	for i, player := range players {
		entry := &models.GolfPlayerEntry{
			ID:               uuid.New(),
			PlayerID:         player.ID,
			TournamentID:     tournament.ID,
			Status:           models.EntryStatusEntered,
			CurrentPosition:  i + 1,
			TotalScore:       -5 + i,
			DKSalary:         player.Salary,
			FDSalary:         player.Salary + 1000,
			DKOwnership:      player.Ownership,
			FDOwnership:      player.Ownership * 0.9,
		}

		err := db.Create(entry).Error
		require.NoError(t, err)
	}
}