package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := database.NewOptimizationServiceConnection(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Get SQL file path
	sqlFile := "../../deploy-missing-datagolf-tables.sql"
	if len(os.Args) > 1 {
		sqlFile = os.Args[1]
	}

	// Make path absolute
	absPath, err := filepath.Abs(sqlFile)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	// Read SQL file
	sqlContent, err := ioutil.ReadFile(absPath)
	if err != nil {
		log.Fatalf("Failed to read SQL file %s: %v", absPath, err)
	}

	log.Printf("Executing SQL from file: %s", absPath)
	
	// Execute SQL
	sqlDB, err := db.DB.DB()
	if err != nil {
		log.Fatalf("Failed to get SQL DB: %v", err)
	}

	// Execute the SQL content
	_, err = sqlDB.ExecContext(context.Background(), string(sqlContent))
	if err != nil {
		log.Fatalf("Failed to execute SQL: %v", err)
	}

	log.Println("DataGolf analytics schema deployment completed successfully!")
}