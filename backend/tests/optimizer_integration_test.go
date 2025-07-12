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
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type OptimizerIntegrationTestSuite struct {
	suite.Suite
	db      *database.DB
	router  *gin.Engine
	handler *handlers.OptimizerHandler
	cache   *services.CacheService
}

func (s *OptimizerIntegrationTestSuite) SetupSuite() {
	// Setup in-memory database
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	s.Require().NoError(err)

	s.db = &database.DB{DB: gormDB}

	// Auto-migrate schemas
	err = s.db.AutoMigrate(
		&models.Contest{},
		&models.Player{},
		&models.Lineup{},
		&models.LineupPlayer{},
	)
	s.Require().NoError(err)

	// Setup cache
	s.cache = services.NewCacheService(nil) // In-memory cache

	// Setup handler
	cfg := &config.Config{
		MaxLineups:          150,
		OptimizationTimeout: 30,
	}
	s.handler = handlers.NewOptimizerHandler(s.db, s.cache, cfg)

	// Setup router
	gin.SetMode(gin.TestMode)
	s.router = gin.New()
	s.router.POST("/optimize", s.handler.OptimizeLineups)
}

func (s *OptimizerIntegrationTestSuite) SetupTest() {
	// Clean database before each test
	s.db.Exec("DELETE FROM lineup_players")
	s.db.Exec("DELETE FROM lineups")
	s.db.Exec("DELETE FROM players")
	s.db.Exec("DELETE FROM contests")
}

func (s *OptimizerIntegrationTestSuite) TestNBAOptimization_SavesCorrectPositions() {
	// Create NBA contest
	contest := models.Contest{
		ID:        1,
		Platform:  "draftkings",
		Sport:     "nba",
		Name:      "NBA $100K Tournament",
		SalaryCap: 50000,
		StartTime: time.Now().Add(24 * time.Hour),
		PositionRequirements: models.PositionRequirements{
			"PG":   1,
			"SG":   1,
			"SF":   1,
			"PF":   1,
			"C":    1,
			"G":    1,
			"F":    1,
			"UTIL": 1,
		},
	}
	s.Require().NoError(s.db.Create(&contest).Error)

	// Create NBA players
	players := []models.Player{
		{ContestID: 1, Name: "Stephen Curry", Position: "PG", Salary: 10000, ProjectedPoints: 50.5, Team: "GSW"},
		{ContestID: 1, Name: "Ja Morant", Position: "PG", Salary: 8500, ProjectedPoints: 42.0, Team: "MEM"},
		{ContestID: 1, Name: "James Harden", Position: "SG", Salary: 9500, ProjectedPoints: 48.0, Team: "PHI"},
		{ContestID: 1, Name: "Devin Booker", Position: "SG", Salary: 8000, ProjectedPoints: 40.0, Team: "PHX"},
		{ContestID: 1, Name: "LeBron James", Position: "SF", Salary: 11000, ProjectedPoints: 52.0, Team: "LAL"},
		{ContestID: 1, Name: "Jimmy Butler", Position: "SF", Salary: 8000, ProjectedPoints: 41.0, Team: "MIA"},
		{ContestID: 1, Name: "Anthony Davis", Position: "PF", Salary: 10500, ProjectedPoints: 51.0, Team: "LAL"},
		{ContestID: 1, Name: "Pascal Siakam", Position: "PF", Salary: 7500, ProjectedPoints: 38.0, Team: "TOR"},
		{ContestID: 1, Name: "Nikola Jokic", Position: "C", Salary: 11500, ProjectedPoints: 55.0, Team: "DEN"},
		{ContestID: 1, Name: "Joel Embiid", Position: "C", Salary: 11000, ProjectedPoints: 53.0, Team: "PHI"},
		{ContestID: 1, Name: "Karl-Anthony Towns", Position: "C", Salary: 8500, ProjectedPoints: 43.0, Team: "MIN"},
	}

	for i := range players {
		s.Require().NoError(s.db.Create(&players[i]).Error)
	}

	// Create optimization request
	req := gin.H{
		"contest_id":            1,
		"num_lineups":           2,
		"min_different_players": 2,
		"use_correlations":      false,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/optimize", bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")

	// Execute request
	s.router.ServeHTTP(w, request)

	// Check response
	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.True(response["success"].(bool))

	// Check saved lineups
	var savedLineups []models.Lineup
	s.db.Find(&savedLineups)
	s.Greater(len(savedLineups), 0, "Should have saved lineups")

	// Check lineup players with positions
	for _, lineup := range savedLineups {
		var lineupPlayers []models.LineupPlayer
		s.db.Where("lineup_id = ?", lineup.ID).Find(&lineupPlayers)

		s.Len(lineupPlayers, 8, "NBA lineup should have 8 players")

		// Track positions used
		positionsUsed := make(map[string]bool)

		for _, lp := range lineupPlayers {
			s.NotEmpty(lp.Position, "Position should not be empty")
			s.False(positionsUsed[lp.Position], "Position %s already used", lp.Position)
			positionsUsed[lp.Position] = true

			// Verify position is valid
			validPositions := []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UTIL"}
			s.Contains(validPositions, lp.Position, "Position %s should be valid", lp.Position)

			// Verify player can fill the position
			var player models.Player
			s.db.First(&player, lp.PlayerID)

			// Check position compatibility
			compatible := s.verifyPositionCompatibility(player.Position, lp.Position)
			s.True(compatible, "Player %s (%s) should be compatible with position %s",
				player.Name, player.Position, lp.Position)
		}

		// Verify all required positions are filled
		requiredPositions := []string{"PG", "SG", "SF", "PF", "C", "G", "F", "UTIL"}
		for _, pos := range requiredPositions {
			s.True(positionsUsed[pos], "Position %s should be filled", pos)
		}
	}
}

func (s *OptimizerIntegrationTestSuite) TestGolfOptimization_ContinuesWorking() {
	// Create Golf contest
	contest := models.Contest{
		ID:        2,
		Platform:  "draftkings",
		Sport:     "golf",
		Name:      "PGA $50K Birdie Maker",
		SalaryCap: 50000,
		StartTime: time.Now().Add(24 * time.Hour),
		PositionRequirements: models.PositionRequirements{
			"G": 6,
		},
	}
	s.Require().NoError(s.db.Create(&contest).Error)

	// Create Golf players
	golfers := []models.Player{
		{ContestID: 2, Name: "Rory McIlroy", Position: "G", Salary: 11500, ProjectedPoints: 65.0, Team: "NIR"},
		{ContestID: 2, Name: "Scottie Scheffler", Position: "G", Salary: 12000, ProjectedPoints: 68.0, Team: "USA"},
		{ContestID: 2, Name: "Jon Rahm", Position: "G", Salary: 11200, ProjectedPoints: 64.0, Team: "ESP"},
		{ContestID: 2, Name: "Patrick Cantlay", Position: "G", Salary: 9500, ProjectedPoints: 58.0, Team: "USA"},
		{ContestID: 2, Name: "Viktor Hovland", Position: "G", Salary: 10200, ProjectedPoints: 61.0, Team: "NOR"},
		{ContestID: 2, Name: "Xander Schauffele", Position: "G", Salary: 9800, ProjectedPoints: 59.0, Team: "USA"},
		{ContestID: 2, Name: "Jordan Spieth", Position: "G", Salary: 8800, ProjectedPoints: 55.0, Team: "USA"},
		{ContestID: 2, Name: "Tony Finau", Position: "G", Salary: 8500, ProjectedPoints: 53.0, Team: "USA"},
	}

	for i := range golfers {
		s.Require().NoError(s.db.Create(&golfers[i]).Error)
	}

	// Create optimization request
	req := gin.H{
		"contest_id":            2,
		"num_lineups":           2,
		"min_different_players": 2,
		"use_correlations":      false,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/optimize", bytes.NewBuffer(body))
	request.Header.Set("Content-Type", "application/json")

	// Execute request
	s.router.ServeHTTP(w, request)

	// Check response
	s.Equal(http.StatusOK, w.Code)

	// Check saved lineups
	var savedLineups []models.Lineup
	s.db.Where("contest_id = ?", 2).Find(&savedLineups)
	s.Greater(len(savedLineups), 0, "Should have saved golf lineups")

	// Check lineup players
	for _, lineup := range savedLineups {
		var lineupPlayers []models.LineupPlayer
		s.db.Where("lineup_id = ?", lineup.ID).Find(&lineupPlayers)

		s.Len(lineupPlayers, 6, "Golf lineup should have 6 players")

		// All positions should be "G"
		for _, lp := range lineupPlayers {
			s.Equal("G", lp.Position, "All golf positions should be 'G'")
		}
	}
}

func (s *OptimizerIntegrationTestSuite) TestAllSports_PositionValidation() {
	sports := []struct {
		sport     string
		platform  string
		positions map[string]int
		players   []models.Player
	}{
		{
			sport:    "nfl",
			platform: "draftkings",
			positions: map[string]int{
				"QB": 1, "RB": 2, "WR": 3, "TE": 1, "FLEX": 1, "DST": 1,
			},
			players: []models.Player{
				{Name: "Mahomes", Position: "QB", Salary: 8000, ProjectedPoints: 25.0},
				{Name: "Henry", Position: "RB", Salary: 7500, ProjectedPoints: 20.0},
				{Name: "Cook", Position: "RB", Salary: 7000, ProjectedPoints: 18.0},
				{Name: "Hill", Position: "WR", Salary: 8500, ProjectedPoints: 22.0},
				{Name: "Jefferson", Position: "WR", Salary: 8000, ProjectedPoints: 21.0},
				{Name: "Chase", Position: "WR", Salary: 7500, ProjectedPoints: 19.0},
				{Name: "Kelce", Position: "TE", Salary: 7000, ProjectedPoints: 18.0},
				{Name: "Diggs", Position: "WR", Salary: 7500, ProjectedPoints: 19.5}, // For FLEX
				{Name: "49ers", Position: "DST", Salary: 5000, ProjectedPoints: 12.0},
			},
		},
		{
			sport:    "mlb",
			platform: "draftkings",
			positions: map[string]int{
				"P": 2, "C": 1, "1B": 1, "2B": 1, "3B": 1, "SS": 1, "OF": 3,
			},
			players: []models.Player{
				{Name: "Cole", Position: "P", Salary: 9000, ProjectedPoints: 20.0},
				{Name: "Scherzer", Position: "P", Salary: 8500, ProjectedPoints: 19.0},
				{Name: "Realmuto", Position: "C", Salary: 5500, ProjectedPoints: 12.0},
				{Name: "Freeman", Position: "1B", Salary: 5000, ProjectedPoints: 11.0},
				{Name: "Altuve", Position: "2B", Salary: 5000, ProjectedPoints: 11.0},
				{Name: "Machado", Position: "3B", Salary: 5000, ProjectedPoints: 11.0},
				{Name: "Turner", Position: "SS", Salary: 5000, ProjectedPoints: 11.0},
				{Name: "Judge", Position: "OF", Salary: 6000, ProjectedPoints: 13.0},
				{Name: "Betts", Position: "OF", Salary: 5500, ProjectedPoints: 12.0},
				{Name: "Acuna", Position: "OF", Salary: 5500, ProjectedPoints: 12.0},
			},
		},
	}

	for i, sport := range sports {
		contestID := uint(i + 10)

		// Create contest
		contest := models.Contest{
			ID:                   contestID,
			Platform:             sport.platform,
			Sport:                sport.sport,
			Name:                 fmt.Sprintf("%s Test Contest", sport.sport),
			SalaryCap:            50000,
			StartTime:            time.Now().Add(24 * time.Hour),
			PositionRequirements: models.PositionRequirements(sport.positions),
		}
		s.Require().NoError(s.db.Create(&contest).Error)

		// Create players
		for j := range sport.players {
			sport.players[j].ContestID = contestID
			s.Require().NoError(s.db.Create(&sport.players[j]).Error)
		}

		// Run optimization
		req := gin.H{
			"contest_id":  contestID,
			"num_lineups": 1,
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		request, _ := http.NewRequest("POST", "/optimize", bytes.NewBuffer(body))
		request.Header.Set("Content-Type", "application/json")

		s.router.ServeHTTP(w, request)

		// Verify success
		s.Equal(http.StatusOK, w.Code, "Optimization should succeed for %s", sport.sport)

		// Verify lineup saved with correct positions
		var lineup models.Lineup
		s.db.Where("contest_id = ?", contestID).First(&lineup)

		var lineupPlayers []models.LineupPlayer
		s.db.Where("lineup_id = ?", lineup.ID).Find(&lineupPlayers)

		// Count total expected players
		totalPlayers := 0
		for _, count := range sport.positions {
			totalPlayers += count
		}

		s.Len(lineupPlayers, totalPlayers, "%s should have %d players", sport.sport, totalPlayers)
	}
}

func (s *OptimizerIntegrationTestSuite) verifyPositionCompatibility(playerPos, slotPos string) bool {
	// Define position compatibility rules
	compatibility := map[string][]string{
		// NBA
		"G":    {"PG", "SG"},
		"F":    {"SF", "PF"},
		"UTIL": {"PG", "SG", "SF", "PF", "C"},
		// NFL
		"FLEX": {"RB", "WR", "TE"},
		// Direct matches always compatible
		"PG":  {"PG"},
		"SG":  {"SG"},
		"SF":  {"SF"},
		"PF":  {"PF"},
		"C":   {"C"},
		"QB":  {"QB"},
		"RB":  {"RB"},
		"WR":  {"WR"},
		"TE":  {"TE"},
		"DST": {"DST"},
	}

	// Direct match
	if playerPos == slotPos {
		return true
	}

	// Check compatibility
	if allowedPositions, exists := compatibility[slotPos]; exists {
		for _, allowed := range allowedPositions {
			if playerPos == allowed {
				return true
			}
		}
	}

	return false
}

func TestOptimizerIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(OptimizerIntegrationTestSuite))
}
