package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/jstittsworth/dfs-optimizer/internal/optimizer"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/jstittsworth/dfs-optimizer/pkg/database"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Connect to database
	db, err := database.NewConnection(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	fmt.Println("=== DFS Optimizer Data Validation ===")
	fmt.Println()

	// Check contests
	var contests []models.Contest
	if err := db.Find(&contests).Error; err != nil {
		log.Fatal("Failed to fetch contests:", err)
	}

	fmt.Printf("Found %d contests\n", len(contests))
	fmt.Println("====================================================")

	for _, contest := range contests {
		fmt.Printf("\nContest %d: %s\n", contest.ID, contest.Name)
		fmt.Printf("  Sport: %s, Platform: %s\n", contest.Sport, contest.Platform)
		fmt.Printf("  Salary Cap: $%d\n", contest.SalaryCap)
		fmt.Printf("  Entry Fee: $%.2f, Prize Pool: $%.2f\n", contest.EntryFee, contest.PrizePool)

		// Check players
		var playerCount int64
		db.Model(&models.Player{}).Where("contest_id = ?", contest.ID).Count(&playerCount)
		fmt.Printf("  Players: %d\n", playerCount)

		// Check by position
		var positions []struct {
			Position string
			Count    int64
		}
		db.Model(&models.Player{}).
			Where("contest_id = ?", contest.ID).
			Select("position, COUNT(*) as count").
			Group("position").
			Order("position").
			Scan(&positions)

		fmt.Printf("\n  By Position:\n")
		for _, p := range positions {
			fmt.Printf("    %s: %d\n", p.Position, p.Count)
		}

		// Check salary distribution
		var salaryStats struct {
			MinSalary int
			MaxSalary int
			AvgSalary float64
		}
		db.Model(&models.Player{}).
			Where("contest_id = ?", contest.ID).
			Select("MIN(salary) as min_salary, MAX(salary) as max_salary, AVG(salary) as avg_salary").
			Scan(&salaryStats)

		fmt.Printf("\n  Salary Range: $%d - $%d (avg: $%.0f)\n",
			salaryStats.MinSalary, salaryStats.MaxSalary, salaryStats.AvgSalary)

		// Sample players for each position
		fmt.Printf("\n  Sample Players by Position:\n")
		for _, p := range positions {
			var samplePlayers []models.Player
			db.Where("contest_id = ? AND position = ?", contest.ID, p.Position).
				Limit(2).
				Order("projected_points DESC").
				Find(&samplePlayers)

			for _, player := range samplePlayers {
				fmt.Printf("    %s: %s (%s) - $%d, %.1f pts\n",
					player.Position, player.Name, player.Team,
					player.Salary, player.ProjectedPoints)
			}
		}

		// Check lineup feasibility
		fmt.Printf("\n  Lineup Feasibility Check:\n")
		checkLineupFeasibility(db, &contest)

		// Check for potential issues
		if playerCount == 0 {
			fmt.Printf("\n  ⚠️  WARNING: No players found for this contest!\n")
		}

		if contest.Sport == "" {
			fmt.Printf("  ⚠️  WARNING: Sport is empty!\n")
		}

		if contest.Platform == "" {
			fmt.Printf("  ⚠️  WARNING: Platform is empty!\n")
		}

		fmt.Println("\n  " + "========================================")
	}

	// Overall statistics
	fmt.Println("\n=== Overall Statistics ===")

	var totalPlayers int64
	db.Model(&models.Player{}).Count(&totalPlayers)
	fmt.Printf("Total Players: %d\n", totalPlayers)

	// Players by sport
	var sportStats []struct {
		Sport string
		Count int64
	}
	db.Table("players").
		Joins("JOIN contests ON players.contest_id = contests.id").
		Select("contests.sport, COUNT(*) as count").
		Group("contests.sport").
		Scan(&sportStats)

	fmt.Println("\nPlayers by Sport:")
	for _, stat := range sportStats {
		fmt.Printf("  %s: %d\n", stat.Sport, stat.Count)
	}

	// Check for missing position data
	var missingPositions int64
	db.Model(&models.Player{}).Where("position = '' OR position IS NULL").Count(&missingPositions)
	if missingPositions > 0 {
		fmt.Printf("\n⚠️  WARNING: %d players have missing position data!\n", missingPositions)
	}

	// Check for invalid salaries
	var invalidSalaries int64
	db.Model(&models.Player{}).Where("salary <= 0").Count(&invalidSalaries)
	if invalidSalaries > 0 {
		fmt.Printf("⚠️  WARNING: %d players have invalid salaries!\n", invalidSalaries)
	}

	fmt.Println("\n=== Data Validation Complete ===")

	os.Exit(0)
}

func checkLineupFeasibility(db *database.DB, contest *models.Contest) {
	// Get position slots for this contest
	slots := optimizer.GetPositionSlots(contest.Sport, contest.Platform)

	fmt.Printf("    Required Slots (%s %s):\n", contest.Sport, contest.Platform)
	for _, slot := range slots {
		fmt.Printf("      %s: %v\n", slot.SlotName, slot.AllowedPositions)
	}

	// Count available players by position
	var positions []struct {
		Position string
		Count    int64
	}
	db.Model(&models.Player{}).
		Where("contest_id = ?", contest.ID).
		Select("position, COUNT(*) as count").
		Group("position").
		Scan(&positions)

	posMap := make(map[string]int64)
	for _, p := range positions {
		posMap[p.Position] = p.Count
	}

	// Check if we can fill each slot
	fmt.Printf("\n    Slot Fill Analysis:\n")
	canFormLineup := true
	usedCounts := make(map[string]int)

	for i, slot := range slots {
		available := 0
		for _, pos := range slot.AllowedPositions {
			available += int(posMap[pos]) - usedCounts[pos]
		}

		status := "✅"
		if available <= 0 {
			status = "❌"
			canFormLineup = false
		}

		fmt.Printf("      Slot %d (%s): Need 1, Have %d available %s\n",
			i+1, slot.SlotName, available, status)

		// Mark one player as used for concrete positions
		if available > 0 && len(slot.AllowedPositions) == 1 {
			usedCounts[slot.AllowedPositions[0]]++
		}
	}

	if canFormLineup {
		fmt.Printf("\n    ✅ Can form valid lineups\n")
	} else {
		fmt.Printf("\n    ❌ CANNOT form valid lineups - insufficient players at required positions\n")
	}
}
