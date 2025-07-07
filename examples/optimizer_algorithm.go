package main

import (
	"fmt"
	"sort"
)

// Player represents a DFS player
type Player struct {
	ID              int
	Name            string
	Position        string
	Team            string
	Salary          int
	ProjectedPoints float64
}

// Lineup represents a DFS lineup
type Lineup struct {
	Players         []Player
	TotalSalary     int
	ProjectedPoints float64
}

// OptimizeLineups demonstrates a basic knapsack algorithm for DFS optimization
func OptimizeLineups(players []Player, salaryCap int, positionReqs map[string]int) []Lineup {
	// Sort players by value (points per dollar)
	sort.Slice(players, func(i, j int) bool {
		valueI := players[i].ProjectedPoints / float64(players[i].Salary)
		valueJ := players[j].ProjectedPoints / float64(players[j].Salary)
		return valueI > valueJ
	})

	// Use dynamic programming approach
	var lineups []Lineup
	
	// Simplified example: Generate one optimal lineup
	lineup := generateOptimalLineup(players, salaryCap, positionReqs)
	if lineup != nil {
		lineups = append(lineups, *lineup)
	}

	return lineups
}

func generateOptimalLineup(players []Player, salaryCap int, positionReqs map[string]int) *Lineup {
	lineup := &Lineup{
		Players: make([]Player, 0),
	}

	// Group players by position
	playersByPosition := make(map[string][]Player)
	for _, player := range players {
		playersByPosition[player.Position] = append(playersByPosition[player.Position], player)
	}

	// Fill each position requirement
	for position, required := range positionReqs {
		positionPlayers := playersByPosition[position]
		added := 0

		for _, player := range positionPlayers {
			if lineup.TotalSalary+player.Salary <= salaryCap && added < required {
				lineup.Players = append(lineup.Players, player)
				lineup.TotalSalary += player.Salary
				lineup.ProjectedPoints += player.ProjectedPoints
				added++
			}
		}

		if added < required {
			fmt.Printf("Warning: Could not fill all %s positions\n", position)
			return nil
		}
	}

	return lineup
}

// Example usage
func main() {
	// Sample NBA players
	players := []Player{
		{1, "Luka Doncic", "PG", "DAL", 11200, 55.5},
		{2, "Trae Young", "PG", "ATL", 9800, 48.0},
		{3, "Devin Booker", "SG", "PHX", 8800, 42.0},
		{4, "Jaylen Brown", "SG", "BOS", 8200, 38.5},
		{5, "LeBron James", "SF", "LAL", 10500, 50.0},
		{6, "Jayson Tatum", "SF", "BOS", 10200, 48.5},
		{7, "Giannis Antetokounmpo", "PF", "MIL", 11800, 58.0},
		{8, "Kevin Durant", "PF", "PHX", 10800, 52.0},
		{9, "Nikola Jokic", "C", "DEN", 12000, 60.0},
		{10, "Joel Embiid", "C", "PHI", 11500, 56.0},
		// Add more players for complete lineup...
	}

	// NBA DraftKings position requirements
	positionReqs := map[string]int{
		"PG": 1,
		"SG": 1,
		"SF": 1,
		"PF": 1,
		"C":  1,
		// Simplified - would need to handle G, F, UTIL positions
	}

	salaryCap := 50000

	// Generate optimized lineups
	lineups := OptimizeLineups(players, salaryCap, positionReqs)

	// Display results
	for i, lineup := range lineups {
		fmt.Printf("\nLineup %d:\n", i+1)
		fmt.Printf("Total Salary: $%d\n", lineup.TotalSalary)
		fmt.Printf("Projected Points: %.1f\n", lineup.ProjectedPoints)
		fmt.Println("Players:")
		for _, player := range lineup.Players {
			fmt.Printf("  %s (%s) - %s - $%d - %.1f pts\n",
				player.Name, player.Position, player.Team, player.Salary, player.ProjectedPoints)
		}
	}
}