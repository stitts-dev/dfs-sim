package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/api"
	"github.com/jstittsworth/dfs-optimizer/internal/api/handlers"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptimizerEndpoint_AllSports(t *testing.T) {
	// Setup test environment
	gin.SetMode(gin.TestMode)

	// Create test database
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(db)

	// Create test config
	cfg := &config.Config{
		MaxLineups:          150,
		OptimizationTimeout: 30,
	}

	// Create cache service
	cache := services.NewCacheService(cfg)

	// Setup router
	router := gin.New()
	optimizerHandler := handlers.NewOptimizerHandler(db, cache, cfg)
	api.SetupRoutes(router, db, cache, cfg)

	// Test each sport
	sports := []struct {
		sport     string
		platform  string
		positions []string
		salaryCap int
	}{
		{"nba", "draftkings", []string{"PG", "SG", "SF", "PF", "C"}, 50000},
		{"nfl", "draftkings", []string{"QB", "RB", "WR", "TE", "DST"}, 50000},
		{"mlb", "draftkings", []string{"P", "C", "1B", "2B", "3B", "SS", "OF"}, 35000},
		{"nhl", "draftkings", []string{"C", "W", "D", "G"}, 50000},
		{"golf", "draftkings", []string{"G"}, 50000},
	}

	for _, sport := range sports {
		t.Run(sport.sport+"_"+sport.platform, func(t *testing.T) {
			// Create contest and players
			contest := setupContestWithPlayers(t, db, sport.sport, sport.platform, sport.positions, sport.salaryCap)

			// Create optimization request
			req := handlers.OptimizeRequest{
				ContestID:           contest.ID,
				NumLineups:          5,
				MinDifferentPlayers: 2,
				UseCorrelations:     false,
			}

			// Marshal request
			body, err := json.Marshal(req)
			require.NoError(t, err)

			// Create HTTP request
			httpReq := httptest.NewRequest("POST", "/api/v1/optimize", bytes.NewBuffer(body))
			httpReq.Header.Set("Content-Type", "application/json")

			// Perform request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httpReq)

			// Check response
			assert.Equal(t, http.StatusOK, w.Code,
				"Should return 200 OK for %s optimization", sport.sport)

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Check response structure
			assert.True(t, response["success"].(bool),
				"Response should indicate success for %s", sport.sport)

			data, ok := response["data"].(map[string]interface{})
			require.True(t, ok, "Response should have data field")

			lineups, ok := data["lineups"].([]interface{})
			require.True(t, ok, "Data should have lineups array")

			assert.Greater(t, len(lineups), 0,
				"Should return lineups for %s", sport.sport)

			// Verify first lineup structure
			if len(lineups) > 0 {
				firstLineup, ok := lineups[0].(map[string]interface{})
				require.True(t, ok, "Lineup should be a map")

				// Check required fields
				assert.Contains(t, firstLineup, "players")
				assert.Contains(t, firstLineup, "total_salary")
				assert.Contains(t, firstLineup, "projected_points")

				// Verify salary cap constraint
				totalSalary := int(firstLineup["total_salary"].(float64))
				assert.LessOrEqual(t, totalSalary, sport.salaryCap,
					"Lineup should not exceed salary cap for %s", sport.sport)
			}
		})
	}
}

func TestOptimizerEndpoint_Constraints(t *testing.T) {
	// Setup similar to above
	gin.SetMode(gin.TestMode)
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(db)

	cfg := &config.Config{
		MaxLineups:          150,
		OptimizationTimeout: 30,
	}
	cache := services.NewCacheService(cfg)
	router := gin.New()
	api.SetupRoutes(router, db, cache, cfg)

	// Create test contest
	contest := setupContestWithPlayers(t, db, "nba", "draftkings",
		[]string{"PG", "SG", "SF", "PF", "C"}, 50000)

	// Test with locked players
	t.Run("with_locked_players", func(t *testing.T) {
		// Get some player IDs to lock
		var players []models.Player
		db.Where("contest_id = ?", contest.ID).Limit(2).Find(&players)
		require.Len(t, players, 2)

		lockedIDs := []uint{players[0].ID, players[1].ID}

		req := handlers.OptimizeRequest{
			ContestID:     contest.ID,
			NumLineups:    3,
			LockedPlayers: lockedIDs,
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/api/v1/optimize", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify locked players are in all lineups
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].(map[string]interface{})
		lineups := data["lineups"].([]interface{})

		for _, lineup := range lineups {
			lineupMap := lineup.(map[string]interface{})
			players := lineupMap["players"].([]interface{})

			// Check that locked players are present
			foundLocked := 0
			for _, player := range players {
				playerMap := player.(map[string]interface{})
				playerID := uint(playerMap["id"].(float64))
				for _, lockedID := range lockedIDs {
					if playerID == lockedID {
						foundLocked++
					}
				}
			}
			assert.Equal(t, len(lockedIDs), foundLocked,
				"All locked players should be in lineup")
		}
	})

	// Test with excluded players
	t.Run("with_excluded_players", func(t *testing.T) {
		// Get some player IDs to exclude
		var players []models.Player
		db.Where("contest_id = ?", contest.ID).Limit(3).Find(&players)
		require.Len(t, players, 3)

		excludedIDs := []uint{players[0].ID, players[1].ID, players[2].ID}

		req := handlers.OptimizeRequest{
			ContestID:       contest.ID,
			NumLineups:      3,
			ExcludedPlayers: excludedIDs,
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/api/v1/optimize", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify excluded players are not in any lineup
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].(map[string]interface{})
		lineups := data["lineups"].([]interface{})

		for _, lineup := range lineups {
			lineupMap := lineup.(map[string]interface{})
			players := lineupMap["players"].([]interface{})

			// Check that excluded players are not present
			for _, player := range players {
				playerMap := player.(map[string]interface{})
				playerID := uint(playerMap["id"].(float64))
				for _, excludedID := range excludedIDs {
					assert.NotEqual(t, playerID, excludedID,
						"Excluded player should not be in lineup")
				}
			}
		}
	})
}

func setupContestWithPlayers(t *testing.T, db *database.DB, sport, platform string, positions []string, salaryCap int) *models.Contest {
	// Create contest
	contest := &models.Contest{
		Name:      "Test " + sport + " Contest",
		Sport:     sport,
		Platform:  platform,
		SalaryCap: salaryCap,
		PositionRequirements: models.PositionRequirements{
			TotalPlayers: getPlayerCountForSport(sport),
		},
	}

	err := db.Create(contest).Error
	require.NoError(t, err)

	// Create players
	players := createTestPlayers(contest.ID, positions)
	for i := range players {
		err := db.Create(&players[i]).Error
		require.NoError(t, err)
	}

	return contest
}

func setupTestDatabase(t *testing.T) *database.DB {
	// This would typically connect to a test database
	// For now, we'll use an in-memory SQLite database
	cfg := &config.Config{
		DatabaseURL: "sqlite::memory:",
	}

	db, err := database.NewConnection(cfg)
	require.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(
		&models.Contest{},
		&models.Player{},
		&models.Lineup{},
		&models.LineupPlayer{},
	)
	require.NoError(t, err)

	return db
}

func cleanupTestDatabase(db *database.DB) {
	// Drop all tables
	db.Exec("DROP TABLE IF EXISTS lineup_players")
	db.Exec("DROP TABLE IF EXISTS lineups")
	db.Exec("DROP TABLE IF EXISTS players")
	db.Exec("DROP TABLE IF EXISTS contests")
}
