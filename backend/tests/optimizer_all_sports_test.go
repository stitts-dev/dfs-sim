package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/api/handlers"
	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/optimizer"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOptimizerAllSports(t *testing.T) {
	// Setup in-memory database
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	db := &database.DB{DB: gormDB}

	// Auto-migrate schemas
	err = db.AutoMigrate(
		&models.Contest{},
		&models.Player{},
		&models.Lineup{},
		&models.LineupPlayer{},
	)
	assert.NoError(t, err)

	// Setup cache and handler
	cache := services.NewCacheService(nil)
	cfg := &config.Config{
		MaxLineups:          150,
		OptimizationTimeout: 30,
	}
	handler := handlers.NewOptimizerHandler(db, cache, cfg)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/optimize", handler.OptimizeLineups)

	sports := []struct {
		sport     string
		platform  string
		positions []string
	}{
		{"nba", "draftkings", []string{"PG", "SG", "SF", "PF", "C"}},
		{"nfl", "draftkings", []string{"QB", "RB", "WR", "TE", "DST"}},
		{"mlb", "draftkings", []string{"P", "C", "1B", "2B", "3B", "SS", "OF"}},
		{"nhl", "draftkings", []string{"C", "W", "D", "G"}},
		{"golf", "draftkings", []string{"G"}},
	}

	for _, test := range sports {
		t.Run(fmt.Sprintf("%s_%s", test.sport, test.platform), func(t *testing.T) {
			// Clean database
			db.Exec("DELETE FROM lineup_players")
			db.Exec("DELETE FROM lineups")
			db.Exec("DELETE FROM players")
			db.Exec("DELETE FROM contests")

			// Setup test data
			contest := createTestContest(test.sport, test.platform)
			assert.NoError(t, db.Create(&contest).Error)

			players := createTestPlayers(contest.ID, test.positions)
			for i := range players {
				assert.NoError(t, db.Create(&players[i]).Error)
			}

			// Run optimizer
			req := map[string]interface{}{
				"contest_id":  contest.ID,
				"num_lineups": 5,
			}

			body, _ := json.Marshal(req)
			w := httptest.NewRecorder()
			request, _ := http.NewRequest("POST", "/optimize", bytes.NewBuffer(body))
			request.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(w, request)

			// Verify results
			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.True(t, response["success"].(bool))

			data := response["data"].(map[string]interface{})
			lineups := data["lineups"].([]interface{})
			assert.NotEmpty(t, lineups, "Expected lineups for %s/%s", test.sport, test.platform)

			// Verify positions filled correctly
			for _, lineup := range lineups {
				lineupMap := lineup.(map[string]interface{})
				players := lineupMap["players"].([]interface{})
				verifyLineupPositions(t, players, test.sport, test.platform)
			}
		})
	}
}

func createTestContest(sport, platform string) models.Contest {
	salaryCap := 50000
	if sport == "nfl" {
		salaryCap = 60000
	}

	posReqs := make(models.PositionRequirements)

	switch sport {
	case "nba":
		posReqs = models.PositionRequirements{
			"PG": 1, "SG": 1, "SF": 1, "PF": 1, "C": 1,
			"G": 1, "F": 1, "UTIL": 1,
		}
	case "nfl":
		posReqs = models.PositionRequirements{
			"QB": 1, "RB": 2, "WR": 3, "TE": 1, "FLEX": 1, "DST": 1,
		}
	case "mlb":
		posReqs = models.PositionRequirements{
			"P": 2, "C": 1, "1B": 1, "2B": 1, "3B": 1, "SS": 1, "OF": 3,
		}
	case "nhl":
		posReqs = models.PositionRequirements{
			"C": 2, "W": 3, "D": 2, "G": 1, "UTIL": 1,
		}
	case "golf":
		posReqs = models.PositionRequirements{
			"G": 6,
		}
	}

	return models.Contest{
		ID:                   1,
		Platform:             platform,
		Sport:                sport,
		Name:                 fmt.Sprintf("%s Test Contest", sport),
		SalaryCap:            salaryCap,
		StartTime:            time.Now().Add(24 * time.Hour),
		PositionRequirements: posReqs,
		EntryFee:             10.0,
		PrizePool:            1000.0,
		MaxEntries:           150,
		Active:               true,
	}
}

func createTestPlayers(contestID uint, positions []string) []models.Player {
	players := []models.Player{}
	teams := []string{"LAL", "GSW", "BOS", "MIA", "DEN", "PHX", "MIL", "PHI"}

	// Create multiple players for each position
	for i, pos := range positions {
		// Create 3-4 players per position
		numPlayers := 3
		if pos == "OF" || pos == "WR" {
			numPlayers = 5 // Need more for multiple slots
		}

		for j := 0; j < numPlayers; j++ {
			player := models.Player{
				ContestID:       contestID,
				Name:            fmt.Sprintf("Player %s-%d", pos, j+1),
				Position:        pos,
				Team:            teams[j%len(teams)],
				Salary:          5000 + (i * 1000) + (j * 500),
				ProjectedPoints: 20.0 + float64(i*2) + float64(j),
				ActualPoints:    0,
			}
			players = append(players, player)
		}
	}

	return players
}

func verifyLineupPositions(t *testing.T, players []interface{}, sport, platform string) {
	slots := optimizer.GetPositionSlots(sport, platform)
	assert.Equal(t, len(slots), len(players),
		"Lineup should have correct number of players for %s", sport)

	// Track which positions are filled
	filledPositions := make(map[string]int)
	for _, player := range players {
		playerMap := player.(map[string]interface{})
		position := playerMap["position"].(string)
		filledPositions[position]++
	}

	// Verify all required slots are filled (basic check)
	assert.Greater(t, len(filledPositions), 0,
		"Lineup should have players in positions for %s", sport)
}

func TestOptimizerEndpoint_AllSports(t *testing.T) {
	// Setup similar to above but focused on API endpoint testing
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	db := &database.DB{DB: gormDB}

	// Auto-migrate schemas
	err = db.AutoMigrate(
		&models.Contest{},
		&models.Player{},
		&models.Lineup{},
		&models.LineupPlayer{},
	)
	assert.NoError(t, err)

	// Setup cache and handler
	cache := services.NewCacheService(nil)
	cfg := &config.Config{
		MaxLineups:          150,
		OptimizationTimeout: 30,
	}
	handler := handlers.NewOptimizerHandler(db, cache, cfg)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/optimize", handler.OptimizeLineups)

	// Test each sport with minimal request
	for _, sport := range []string{"nba", "nfl", "mlb", "nhl", "golf"} {
		contest := setupContestWithPlayers(db, sport, "draftkings")

		req := map[string]interface{}{
			"contest_id":  contest.ID,
			"num_lineups": 5,
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		request, _ := http.NewRequest("POST", "/optimize", bytes.NewBuffer(body))
		request.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, request)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].(map[string]interface{})
		lineups := data["lineups"].([]interface{})
		assert.Greater(t, len(lineups), 0,
			"Should return lineups for %s", sport)
	}
}

func setupContestWithPlayers(db *database.DB, sport, platform string) models.Contest {
	// Clean up first
	db.Exec("DELETE FROM players WHERE contest_id > 100")
	db.Exec("DELETE FROM contests WHERE id > 100")

	contestID := uint(101)
	if sport == "nfl" {
		contestID = 102
	} else if sport == "mlb" {
		contestID = 103
	} else if sport == "nhl" {
		contestID = 104
	} else if sport == "golf" {
		contestID = 105
	}

	contest := createTestContest(sport, platform)
	contest.ID = contestID
	db.Create(&contest)

	positions := []string{}
	switch sport {
	case "nba":
		positions = []string{"PG", "SG", "SF", "PF", "C"}
	case "nfl":
		positions = []string{"QB", "RB", "WR", "TE", "DST"}
	case "mlb":
		positions = []string{"P", "C", "1B", "2B", "3B", "SS", "OF"}
	case "nhl":
		positions = []string{"C", "W", "D", "G"}
	case "golf":
		positions = []string{"G"}
	}

	players := createTestPlayers(contestID, positions)
	for i := range players {
		db.Create(&players[i])
	}

	return contest
}
