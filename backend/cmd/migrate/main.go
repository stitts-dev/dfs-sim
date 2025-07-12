package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: migrate [up|down|seed]")
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := database.NewConnection(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		logrus.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	command := os.Args[1]

	switch command {
	case "up":
		if err := runMigrations(db); err != nil {
			logrus.Fatalf("Failed to run migrations: %v", err)
		}
		logrus.Info("Migrations completed successfully")

	case "down":
		if err := dropTables(db); err != nil {
			logrus.Fatalf("Failed to drop tables: %v", err)
		}
		logrus.Info("Tables dropped successfully")

	case "seed":
		if err := seedData(db); err != nil {
			logrus.Fatalf("Failed to seed data: %v", err)
		}
		logrus.Info("Data seeded successfully")

	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

func runMigrations(db *database.DB) error {
	// Enable UUID extension for PostgreSQL
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return fmt.Errorf("failed to create UUID extension: %w", err)
	}

	// Auto migrate all models
	if err := db.AutoMigrate(
		&models.Contest{},
		&models.Player{},
		&models.Lineup{},
		&models.LineupPlayer{},
		&models.SimulationResult{},
		&models.PlayerMetadata{},
		&models.TeamInfo{},
		&models.GlossaryTerm{},
		&models.AIRecommendation{},
		&models.UserPreferences{},
	); err != nil {
		return fmt.Errorf("failed to migrate models: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_players_contest_sport ON players(contest_id, sport)",
		"CREATE INDEX IF NOT EXISTS idx_players_team ON players(team)",
		"CREATE INDEX IF NOT EXISTS idx_players_position ON players(position)",
		"CREATE INDEX IF NOT EXISTS idx_players_salary ON players(salary)",
		"CREATE INDEX IF NOT EXISTS idx_lineups_user_contest ON lineups(user_id, contest_id)",
		"CREATE INDEX IF NOT EXISTS idx_lineups_projected_points ON lineups(projected_points DESC)",
		"CREATE INDEX IF NOT EXISTS idx_simulation_results_lineup ON simulation_results(lineup_id)",
		"CREATE INDEX IF NOT EXISTS idx_player_metadata_player_id ON player_metadata(player_id)",
		"CREATE INDEX IF NOT EXISTS idx_glossary_category ON glossary_terms(category)",
		"CREATE INDEX IF NOT EXISTS idx_glossary_difficulty ON glossary_terms(difficulty)",
		"CREATE INDEX IF NOT EXISTS idx_glossary_sport ON glossary_terms(sport)",
		"CREATE INDEX IF NOT EXISTS idx_glossary_search ON glossary_terms USING gin(to_tsvector('english', term || ' ' || definition))",
		"CREATE INDEX IF NOT EXISTS idx_ai_recommendations_user ON ai_recommendations(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_ai_recommendations_contest ON ai_recommendations(contest_id)",
	}

	for _, index := range indexes {
		if err := db.Exec(index).Error; err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

func dropTables(db *database.DB) error {
	// Drop tables in reverse order to handle foreign key constraints
	tables := []string{
		"user_preferences",
		"ai_recommendations",
		"glossary_terms",
		"team_infos",
		"player_metadata",
		"simulation_results",
		"lineup_players",
		"lineups",
		"players",
		"contests",
	}

	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)).Error; err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}

func seedData(db *database.DB) error {
	// Create sample contest
	contest := &models.Contest{
		Platform:             "draftkings",
		Sport:                "nba",
		ContestType:          "gpp",
		Name:                 "NBA $100K Tournament",
		EntryFee:             20,
		PrizePool:            100000,
		MaxEntries:           10000,
		TotalEntries:         0,
		SalaryCap:            50000,
		StartTime:            time.Now().Add(2 * time.Hour), // Contest starts in 2 hours
		IsActive:             true,
		IsMultiEntry:         true,
		MaxLineupsPerUser:    20,
		PositionRequirements: models.GetPositionRequirements("nba", "draftkings"),
	}

	if err := db.Create(contest).Error; err != nil {
		return fmt.Errorf("failed to create contest: %w", err)
	}

	// Create sample players
	samplePlayers := []models.Player{
		// Point Guards
		{ExternalID: "dk_001", Name: "Luka Doncic", Team: "DAL", Opponent: "LAL", Position: "PG", Salary: 11200, ProjectedPoints: 55.5, FloorPoints: 45.0, CeilingPoints: 70.0, Ownership: 25.5, Sport: "nba", ContestID: contest.ID},
		{ExternalID: "dk_002", Name: "Trae Young", Team: "ATL", Opponent: "BOS", Position: "PG", Salary: 9800, ProjectedPoints: 48.0, FloorPoints: 38.0, CeilingPoints: 62.0, Ownership: 18.0, Sport: "nba", ContestID: contest.ID},
		{ExternalID: "dk_003", Name: "Ja Morant", Team: "MEM", Opponent: "GSW", Position: "PG", Salary: 9500, ProjectedPoints: 46.5, FloorPoints: 36.0, CeilingPoints: 60.0, Ownership: 15.5, Sport: "nba", ContestID: contest.ID},
		// Shooting Guards
		{ExternalID: "dk_004", Name: "Devin Booker", Team: "PHX", Opponent: "DEN", Position: "SG", Salary: 8800, ProjectedPoints: 42.0, FloorPoints: 32.0, CeilingPoints: 55.0, Ownership: 12.0, Sport: "nba", ContestID: contest.ID},
		{ExternalID: "dk_005", Name: "Jaylen Brown", Team: "BOS", Opponent: "ATL", Position: "SG", Salary: 8200, ProjectedPoints: 38.5, FloorPoints: 28.0, CeilingPoints: 50.0, Ownership: 10.5, Sport: "nba", ContestID: contest.ID},
		// Small Forwards
		{ExternalID: "dk_006", Name: "LeBron James", Team: "LAL", Opponent: "DAL", Position: "SF", Salary: 10500, ProjectedPoints: 50.0, FloorPoints: 40.0, CeilingPoints: 65.0, Ownership: 20.0, Sport: "nba", ContestID: contest.ID},
		{ExternalID: "dk_007", Name: "Jayson Tatum", Team: "BOS", Opponent: "ATL", Position: "SF", Salary: 10200, ProjectedPoints: 48.5, FloorPoints: 38.0, CeilingPoints: 62.0, Ownership: 18.5, Sport: "nba", ContestID: contest.ID},
		// Power Forwards
		{ExternalID: "dk_008", Name: "Giannis Antetokounmpo", Team: "MIL", Opponent: "CHI", Position: "PF", Salary: 11800, ProjectedPoints: 58.0, FloorPoints: 48.0, CeilingPoints: 72.0, Ownership: 28.0, Sport: "nba", ContestID: contest.ID},
		{ExternalID: "dk_009", Name: "Kevin Durant", Team: "PHX", Opponent: "DEN", Position: "PF", Salary: 10800, ProjectedPoints: 52.0, FloorPoints: 42.0, CeilingPoints: 66.0, Ownership: 22.0, Sport: "nba", ContestID: contest.ID},
		// Centers
		{ExternalID: "dk_010", Name: "Nikola Jokic", Team: "DEN", Opponent: "PHX", Position: "C", Salary: 12000, ProjectedPoints: 60.0, FloorPoints: 50.0, CeilingPoints: 75.0, Ownership: 30.0, Sport: "nba", ContestID: contest.ID},
		{ExternalID: "dk_011", Name: "Joel Embiid", Team: "PHI", Opponent: "NYK", Position: "C", Salary: 11500, ProjectedPoints: 56.0, FloorPoints: 46.0, CeilingPoints: 70.0, Ownership: 26.0, Sport: "nba", ContestID: contest.ID},
		// Utility players
		{ExternalID: "dk_012", Name: "Anthony Davis", Team: "LAL", Opponent: "DAL", Position: "C", Salary: 10000, ProjectedPoints: 48.0, FloorPoints: 38.0, CeilingPoints: 62.0, Ownership: 16.0, Sport: "nba", ContestID: contest.ID},
		{ExternalID: "dk_013", Name: "Damian Lillard", Team: "MIL", Opponent: "CHI", Position: "PG", Salary: 9200, ProjectedPoints: 44.0, FloorPoints: 34.0, CeilingPoints: 58.0, Ownership: 14.0, Sport: "nba", ContestID: contest.ID},
		// More value players
		{ExternalID: "dk_014", Name: "Tyrese Haliburton", Team: "IND", Opponent: "CLE", Position: "PG", Salary: 8500, ProjectedPoints: 40.0, FloorPoints: 30.0, CeilingPoints: 52.0, Ownership: 8.0, Sport: "nba", ContestID: contest.ID},
		{ExternalID: "dk_015", Name: "Darius Garland", Team: "CLE", Opponent: "IND", Position: "PG", Salary: 7800, ProjectedPoints: 36.0, FloorPoints: 26.0, CeilingPoints: 48.0, Ownership: 6.0, Sport: "nba", ContestID: contest.ID},
		{ExternalID: "dk_016", Name: "CJ McCollum", Team: "NOP", Opponent: "OKC", Position: "SG", Salary: 7200, ProjectedPoints: 34.0, FloorPoints: 24.0, CeilingPoints: 45.0, Ownership: 5.5, Sport: "nba", ContestID: contest.ID},
		{ExternalID: "dk_017", Name: "Brandon Ingram", Team: "NOP", Opponent: "OKC", Position: "SF", Salary: 8000, ProjectedPoints: 38.0, FloorPoints: 28.0, CeilingPoints: 50.0, Ownership: 7.0, Sport: "nba", ContestID: contest.ID},
		{ExternalID: "dk_018", Name: "Zion Williamson", Team: "NOP", Opponent: "OKC", Position: "PF", Salary: 8800, ProjectedPoints: 42.0, FloorPoints: 32.0, CeilingPoints: 55.0, Ownership: 9.0, Sport: "nba", ContestID: contest.ID},
		{ExternalID: "dk_019", Name: "Alperen Sengun", Team: "HOU", Opponent: "SAS", Position: "C", Salary: 7500, ProjectedPoints: 36.0, FloorPoints: 26.0, CeilingPoints: 48.0, Ownership: 6.5, Sport: "nba", ContestID: contest.ID},
		{ExternalID: "dk_020", Name: "Walker Kessler", Team: "UTA", Opponent: "POR", Position: "C", Salary: 5800, ProjectedPoints: 28.0, FloorPoints: 20.0, CeilingPoints: 38.0, Ownership: 3.5, Sport: "nba", ContestID: contest.ID},
	}

	// Set game times for all players (same day, different times)
	baseTime := contest.StartTime
	for i := range samplePlayers {
		samplePlayers[i].GameTime = baseTime.Add(time.Duration(i/4) * time.Hour)
	}

	if err := db.Create(&samplePlayers).Error; err != nil {
		return fmt.Errorf("failed to create players: %w", err)
	}

	logrus.Infof("Seeded %d players for contest %s", len(samplePlayers), contest.Name)

	// Seed team information
	teams := []models.TeamInfo{
		{Abbreviation: "DAL", FullName: "Dallas Mavericks", Stadium: "American Airlines Center", Outdoor: false, Timezone: "America/Chicago"},
		{Abbreviation: "LAL", FullName: "Los Angeles Lakers", Stadium: "Crypto.com Arena", Outdoor: false, Timezone: "America/Los_Angeles"},
		{Abbreviation: "ATL", FullName: "Atlanta Hawks", Stadium: "State Farm Arena", Outdoor: false, Timezone: "America/New_York"},
		{Abbreviation: "BOS", FullName: "Boston Celtics", Stadium: "TD Garden", Outdoor: false, Timezone: "America/New_York"},
		{Abbreviation: "MEM", FullName: "Memphis Grizzlies", Stadium: "FedExForum", Outdoor: false, Timezone: "America/Chicago"},
		{Abbreviation: "GSW", FullName: "Golden State Warriors", Stadium: "Chase Center", Outdoor: false, Timezone: "America/Los_Angeles"},
		{Abbreviation: "PHX", FullName: "Phoenix Suns", Stadium: "Footprint Center", Outdoor: false, Timezone: "America/Phoenix"},
		{Abbreviation: "DEN", FullName: "Denver Nuggets", Stadium: "Ball Arena", Outdoor: false, Timezone: "America/Denver"},
		{Abbreviation: "MIL", FullName: "Milwaukee Bucks", Stadium: "Fiserv Forum", Outdoor: false, Timezone: "America/Chicago"},
		{Abbreviation: "CHI", FullName: "Chicago Bulls", Stadium: "United Center", Outdoor: false, Timezone: "America/Chicago"},
		{Abbreviation: "PHI", FullName: "Philadelphia 76ers", Stadium: "Wells Fargo Center", Outdoor: false, Timezone: "America/New_York"},
		{Abbreviation: "NYK", FullName: "New York Knicks", Stadium: "Madison Square Garden", Outdoor: false, Timezone: "America/New_York"},
		{Abbreviation: "IND", FullName: "Indiana Pacers", Stadium: "Gainbridge Fieldhouse", Outdoor: false, Timezone: "America/New_York"},
		{Abbreviation: "CLE", FullName: "Cleveland Cavaliers", Stadium: "Rocket Mortgage FieldHouse", Outdoor: false, Timezone: "America/New_York"},
		{Abbreviation: "NOP", FullName: "New Orleans Pelicans", Stadium: "Smoothie King Center", Outdoor: false, Timezone: "America/Chicago"},
		{Abbreviation: "OKC", FullName: "Oklahoma City Thunder", Stadium: "Paycom Center", Outdoor: false, Timezone: "America/Chicago"},
		{Abbreviation: "HOU", FullName: "Houston Rockets", Stadium: "Toyota Center", Outdoor: false, Timezone: "America/Chicago"},
		{Abbreviation: "SAS", FullName: "San Antonio Spurs", Stadium: "AT&T Center", Outdoor: false, Timezone: "America/Chicago"},
		{Abbreviation: "UTA", FullName: "Utah Jazz", Stadium: "Delta Center", Outdoor: false, Timezone: "America/Denver"},
		{Abbreviation: "POR", FullName: "Portland Trail Blazers", Stadium: "Moda Center", Outdoor: false, Timezone: "America/Los_Angeles"},
	}

	if err := db.Create(&teams).Error; err != nil {
		logrus.Warnf("Failed to seed teams (may already exist): %v", err)
	}

	// Seed glossary terms
	glossaryTerms := []models.GlossaryTerm{
		// General DFS terms
		{Term: "DFS", Category: "general", Definition: "Daily Fantasy Sports - A type of fantasy sports where contests last one day or one week, rather than an entire season.", Difficulty: "beginner"},
		{Term: "GPP", Category: "general", Definition: "Guaranteed Prize Pool - Large tournaments with set prize pools regardless of entries. High risk, high reward contests.", Difficulty: "beginner"},
		{Term: "Cash Game", Category: "general", Definition: "Contests where roughly 50% of entrants win (50/50s, Double-Ups, Head-to-Heads). Lower risk than GPPs.", Difficulty: "beginner"},
		{Term: "Salary Cap", Category: "general", Definition: "The maximum amount of virtual dollars you can spend on your lineup. Each player has a salary based on their expected performance.", Difficulty: "beginner"},
		{Term: "Ownership", Category: "general", Definition: "The percentage of lineups that include a specific player. Important for GPP strategy.", Difficulty: "intermediate"},
		{Term: "Stacking", Category: "strategy", Definition: "Selecting multiple players from the same team or game to maximize correlation.", Difficulty: "intermediate"},
		{Term: "Correlation", Category: "strategy", Definition: "The relationship between players' performances. Correlated players tend to score well together.", Difficulty: "advanced"},
		{Term: "Chalk", Category: "general", Definition: "Popular, highly-owned players. Playing chalk is safer but limits upside in GPPs.", Difficulty: "intermediate"},
		{Term: "Fade", Category: "strategy", Definition: "To avoid selecting a player, typically one with high projected ownership.", Difficulty: "intermediate"},
		{Term: "Pivot", Category: "strategy", Definition: "Selecting a less popular player at the same position as a chalky play.", Difficulty: "intermediate"},
		{Term: "Floor", Category: "general", Definition: "A player's expected minimum points based on their role and matchup. Important for cash games.", Difficulty: "intermediate"},
		{Term: "Ceiling", Category: "general", Definition: "A player's maximum scoring potential. Important for GPP tournaments.", Difficulty: "intermediate"},
		{Term: "Leverage", Category: "strategy", Definition: "Gaining an advantage by being contrarian when a popular player fails.", Difficulty: "advanced"},
		{Term: "Late Swap", Category: "strategy", Definition: "Changing players in your lineup after contests start but before their games begin.", Difficulty: "intermediate"},
		{Term: "Multi-Entry", Category: "general", Definition: "Contests allowing multiple lineup entries from the same user.", Difficulty: "beginner"},
		{Term: "Single Entry", Category: "general", Definition: "Contests limited to one lineup per user. More casual-friendly.", Difficulty: "beginner"},
		{Term: "Optimizer", Category: "general", Definition: "Software that generates optimal lineups based on projections and settings.", Difficulty: "beginner"},
		{Term: "Exposure", Category: "strategy", Definition: "The percentage of your lineups containing a specific player.", Difficulty: "intermediate"},
		{Term: "Value", Category: "general", Definition: "Points per dollar. A $5,000 player projecting 25 points = 5x value.", Difficulty: "beginner"},
		{Term: "Punt", Category: "strategy", Definition: "Selecting a very cheap player to afford expensive players elsewhere.", Difficulty: "intermediate"},
	}

	for i := range glossaryTerms {
		glossaryTerms[i].Examples = datatypes.JSON(`[]`)
		glossaryTerms[i].RelatedTerms = []string{}
	}

	if err := db.Create(&glossaryTerms).Error; err != nil {
		logrus.Warnf("Failed to seed glossary terms: %v", err)
	}

	logrus.Info("Seeded team information and glossary terms")

	// All static golf contest and player seeding logic has been removed. See new DraftKings provider for live data.

	return nil
}
