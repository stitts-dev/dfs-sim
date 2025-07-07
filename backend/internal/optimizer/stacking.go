package optimizer

import (
	"sort"

	"github.com/jstittsworth/dfs-optimizer/internal/models"
)

// StackType represents different types of stacks
type StackType string

const (
	TeamStack StackType = "team"
	GameStack StackType = "game"
	MiniStack StackType = "mini"
	BringBack StackType = "bringback"
)

// Stack represents a group of correlated players
type Stack struct {
	Type             StackType
	Players          []models.Player
	TotalSalary      int
	ProjectedPoints  float64
	CorrelationScore float64
	Team             string // For team stacks
	Game             string // For game stacks
}

// StackBuilder helps build optimal stacks
type StackBuilder struct {
	players      []models.Player
	sport        string
	correlations *CorrelationMatrix
}

// NewStackBuilder creates a new stack builder
func NewStackBuilder(players []models.Player, sport string) *StackBuilder {
	return &StackBuilder{
		players:      players,
		sport:        sport,
		correlations: NewCorrelationMatrix(players),
	}
}

// BuildTeamStacks creates stacks of players from the same team
func (sb *StackBuilder) BuildTeamStacks(minSize, maxSize int) []Stack {
	stacks := make([]Stack, 0)

	// Group players by team
	teamPlayers := make(map[string][]models.Player)
	for _, player := range sb.players {
		teamPlayers[player.Team] = append(teamPlayers[player.Team], player)
	}

	// Build stacks for each team
	for team, players := range teamPlayers {
		if len(players) < minSize {
			continue
		}

		// Generate combinations of team stacks
		teamStacks := sb.generateTeamStacks(team, players, minSize, maxSize)
		stacks = append(stacks, teamStacks...)
	}

	// Sort by projected points + correlation bonus
	sort.Slice(stacks, func(i, j int) bool {
		scoreI := stacks[i].ProjectedPoints + stacks[i].CorrelationScore*10
		scoreJ := stacks[j].ProjectedPoints + stacks[j].CorrelationScore*10
		return scoreI > scoreJ
	})

	return stacks
}

// BuildGameStacks creates stacks of players from the same game
func (sb *StackBuilder) BuildGameStacks(minSize, maxSize int) []Stack {
	stacks := make([]Stack, 0)

	// Group players by game
	gamePlayers := make(map[string][]models.Player)
	for _, player := range sb.players {
		gameKey := getGameKey(player.Team, player.Opponent)
		gamePlayers[gameKey] = append(gamePlayers[gameKey], player)
	}

	// Build stacks for each game
	for game, players := range gamePlayers {
		if len(players) < minSize {
			continue
		}

		// Must have players from both teams for game stack
		teams := make(map[string]int)
		for _, p := range players {
			teams[p.Team]++
		}

		if len(teams) < 2 {
			continue
		}

		// Generate game stack combinations
		gameStacks := sb.generateGameStacks(game, players, minSize, maxSize)
		stacks = append(stacks, gameStacks...)
	}

	// Sort by value
	sort.Slice(stacks, func(i, j int) bool {
		scoreI := stacks[i].ProjectedPoints + stacks[i].CorrelationScore*15
		scoreJ := stacks[j].ProjectedPoints + stacks[j].CorrelationScore*15
		return scoreI > scoreJ
	})

	return stacks
}

// GetOptimalStacks returns the best stacks based on sport-specific rules
func (sb *StackBuilder) GetOptimalStacks() []Stack {
	switch sb.sport {
	case "nba":
		return sb.getNBAStacks()
	case "nfl":
		return sb.getNFLStacks()
	case "mlb":
		return sb.getMLBStacks()
	case "nhl":
		return sb.getNHLStacks()
	case "golf":
		return sb.getGolfStacks()
	default:
		return []Stack{}
	}
}

func (sb *StackBuilder) getNBAStacks() []Stack {
	stacks := make([]Stack, 0)

	// Team stacks (2-4 players)
	teamStacks := sb.BuildTeamStacks(2, 4)
	stacks = append(stacks, teamStacks[:min(20, len(teamStacks))]...)

	// Game stacks (3-5 players)
	gameStacks := sb.BuildGameStacks(3, 5)
	stacks = append(stacks, gameStacks[:min(15, len(gameStacks))]...)

	return stacks
}

func (sb *StackBuilder) getNFLStacks() []Stack {
	stacks := make([]Stack, 0)

	// QB stacks are crucial in NFL
	qbStacks := sb.buildQBStacks()
	stacks = append(stacks, qbStacks...)

	// RB + DEF mini stacks
	rbDefStacks := sb.buildRBDefenseStacks()
	stacks = append(stacks, rbDefStacks...)

	return stacks
}

func (sb *StackBuilder) getMLBStacks() []Stack {
	stacks := make([]Stack, 0)

	// Batting order stacks (2-5 consecutive batters)
	battingStacks := sb.BuildTeamStacks(2, 5)

	// Filter for likely consecutive batters
	for i := range battingStacks {
		// Exclude pitchers from stacks
		filtered := make([]models.Player, 0)
		for _, p := range battingStacks[i].Players {
			if p.Position != "P" {
				filtered = append(filtered, p)
			}
		}
		battingStacks[i].Players = filtered
	}

	stacks = append(stacks, battingStacks[:min(20, len(battingStacks))]...)

	return stacks
}

func (sb *StackBuilder) getNHLStacks() []Stack {
	stacks := make([]Stack, 0)

	// Line stacks (C + W from same line)
	lineStacks := sb.buildNHLLineStacks()
	stacks = append(stacks, lineStacks...)

	// Power play stacks
	ppStacks := sb.BuildTeamStacks(3, 4)
	stacks = append(stacks, ppStacks[:min(10, len(ppStacks))]...)

	return stacks
}

func (sb *StackBuilder) getGolfStacks() []Stack {
	stacks := make([]Stack, 0)

	// Golf stacking is unique - focus on ownership and correlation strategies
	
	// Country stacks (2-3 players from same country)
	countryStacks := sb.buildGolfCountryStacks()
	stacks = append(stacks, countryStacks...)

	// Ownership leverage stacks (high-owned + low-owned)
	ownershipStacks := sb.buildGolfOwnershipStacks()
	stacks = append(stacks, ownershipStacks...)

	// Stars and scrubs stacks
	valueStacks := sb.buildGolfValueStacks()
	stacks = append(stacks, valueStacks...)

	return stacks
}

func (sb *StackBuilder) buildQBStacks() []Stack {
	stacks := make([]Stack, 0)

	// Find all QBs
	qbs := make([]models.Player, 0)
	for _, p := range sb.players {
		if p.Position == "QB" {
			qbs = append(qbs, p)
		}
	}

	// For each QB, find best stacking partners
	for _, qb := range qbs {
		// Get teammates
		teammates := make([]models.Player, 0)
		for _, p := range sb.players {
			if p.Team == qb.Team && (p.Position == "WR" || p.Position == "TE") {
				teammates = append(teammates, p)
			}
		}

		// Create QB + 1 stacks
		for _, teammate := range teammates {
			stack := Stack{
				Type:            TeamStack,
				Players:         []models.Player{qb, teammate},
				Team:            qb.Team,
				TotalSalary:     qb.Salary + teammate.Salary,
				ProjectedPoints: qb.ProjectedPoints + teammate.ProjectedPoints,
			}
			stack.CorrelationScore = sb.correlations.GetCorrelation(qb.ID, teammate.ID)
			stacks = append(stacks, stack)
		}

		// Create QB + 2 stacks
		if len(teammates) >= 2 {
			for i := 0; i < len(teammates)-1; i++ {
				for j := i + 1; j < len(teammates); j++ {
					stack := Stack{
						Type:            TeamStack,
						Players:         []models.Player{qb, teammates[i], teammates[j]},
						Team:            qb.Team,
						TotalSalary:     qb.Salary + teammates[i].Salary + teammates[j].Salary,
						ProjectedPoints: qb.ProjectedPoints + teammates[i].ProjectedPoints + teammates[j].ProjectedPoints,
					}
					stack.CorrelationScore = sb.correlations.CalculateLineupCorrelation(stack.Players)
					stacks = append(stacks, stack)
				}
			}
		}

		// Bring-back stacks (QB + teammate + opponent)
		opponents := make([]models.Player, 0)
		for _, p := range sb.players {
			if p.Opponent == qb.Team && (p.Position == "WR" || p.Position == "TE" || p.Position == "RB") {
				opponents = append(opponents, p)
			}
		}

		for _, teammate := range teammates {
			for _, opp := range opponents {
				stack := Stack{
					Type:            GameStack,
					Players:         []models.Player{qb, teammate, opp},
					Game:            getGameKey(qb.Team, qb.Opponent),
					TotalSalary:     qb.Salary + teammate.Salary + opp.Salary,
					ProjectedPoints: qb.ProjectedPoints + teammate.ProjectedPoints + opp.ProjectedPoints,
				}
				stack.CorrelationScore = sb.correlations.CalculateLineupCorrelation(stack.Players)
				stacks = append(stacks, stack)
			}
		}
	}

	return stacks
}

func (sb *StackBuilder) buildRBDefenseStacks() []Stack {
	stacks := make([]Stack, 0)

	// RB + Defense from same team (game script correlation)
	for _, player := range sb.players {
		if player.Position == "RB" {
			// Find team defense
			for _, def := range sb.players {
				if (def.Position == "DST" || def.Position == "D/ST") && def.Team == player.Team {
					stack := Stack{
						Type:            MiniStack,
						Players:         []models.Player{player, def},
						Team:            player.Team,
						TotalSalary:     player.Salary + def.Salary,
						ProjectedPoints: player.ProjectedPoints + def.ProjectedPoints,
					}
					stack.CorrelationScore = 0.3 // Positive game script correlation
					stacks = append(stacks, stack)
				}
			}
		}
	}

	return stacks
}

func (sb *StackBuilder) buildNHLLineStacks() []Stack {
	stacks := make([]Stack, 0)

	// Group by team
	teamPlayers := make(map[string][]models.Player)
	for _, p := range sb.players {
		if p.Position == "C" || p.Position == "W" {
			teamPlayers[p.Team] = append(teamPlayers[p.Team], p)
		}
	}

	// Build line combinations
	for team, players := range teamPlayers {
		centers := make([]models.Player, 0)
		wingers := make([]models.Player, 0)

		for _, p := range players {
			if p.Position == "C" {
				centers = append(centers, p)
			} else {
				wingers = append(wingers, p)
			}
		}

		// Pair centers with wingers
		for _, c := range centers {
			for i := 0; i < len(wingers)-1; i++ {
				for j := i + 1; j < len(wingers); j++ {
					stack := Stack{
						Type:            TeamStack,
						Players:         []models.Player{c, wingers[i], wingers[j]},
						Team:            team,
						TotalSalary:     c.Salary + wingers[i].Salary + wingers[j].Salary,
						ProjectedPoints: c.ProjectedPoints + wingers[i].ProjectedPoints + wingers[j].ProjectedPoints,
					}
					stack.CorrelationScore = 0.4 // Line mate correlation
					stacks = append(stacks, stack)
				}
			}
		}
	}

	return stacks
}

func (sb *StackBuilder) generateTeamStacks(team string, players []models.Player, minSize, maxSize int) []Stack {
	stacks := make([]Stack, 0)

	// Generate all combinations of the specified size range
	for size := minSize; size <= maxSize && size <= len(players); size++ {
		sb.generateCombinations(players, size, func(combo []models.Player) {
			stack := Stack{
				Type:            TeamStack,
				Team:            team,
				Players:         make([]models.Player, len(combo)),
				TotalSalary:     0,
				ProjectedPoints: 0,
			}

			copy(stack.Players, combo)

			for _, p := range combo {
				stack.TotalSalary += p.Salary
				stack.ProjectedPoints += p.ProjectedPoints
			}

			stack.CorrelationScore = sb.correlations.CalculateLineupCorrelation(combo)
			stacks = append(stacks, stack)
		})
	}

	return stacks
}

func (sb *StackBuilder) generateGameStacks(game string, players []models.Player, minSize, maxSize int) []Stack {
	stacks := make([]Stack, 0)

	// Ensure we have players from both teams
	teamCounts := make(map[string][]models.Player)
	for _, p := range players {
		teamCounts[p.Team] = append(teamCounts[p.Team], p)
	}

	if len(teamCounts) < 2 {
		return stacks
	}

	// Generate combinations that include players from both teams
	for size := minSize; size <= maxSize && size <= len(players); size++ {
		sb.generateCombinations(players, size, func(combo []models.Player) {
			// Check if combo has players from multiple teams
			teams := make(map[string]bool)
			for _, p := range combo {
				teams[p.Team] = true
			}

			if len(teams) >= 2 {
				stack := Stack{
					Type:            GameStack,
					Game:            game,
					Players:         make([]models.Player, len(combo)),
					TotalSalary:     0,
					ProjectedPoints: 0,
				}

				copy(stack.Players, combo)

				for _, p := range combo {
					stack.TotalSalary += p.Salary
					stack.ProjectedPoints += p.ProjectedPoints
				}

				stack.CorrelationScore = sb.correlations.CalculateLineupCorrelation(combo)
				stacks = append(stacks, stack)
			}
		})
	}

	return stacks
}

func (sb *StackBuilder) generateCombinations(players []models.Player, k int, callback func([]models.Player)) {
	n := len(players)
	if k > n {
		return
	}

	// Generate combinations using binary representation
	for i := 0; i < (1 << n); i++ {
		if countBits(i) == k {
			combo := make([]models.Player, 0, k)
			for j := 0; j < n; j++ {
				if (i>>j)&1 == 1 {
					combo = append(combo, players[j])
				}
			}
			callback(combo)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Golf-specific stacking methods

func (sb *StackBuilder) buildGolfCountryStacks() []Stack {
	stacks := make([]Stack, 0)

	// Group players by country (team field in golf represents country)
	countryPlayers := make(map[string][]models.Player)
	for _, player := range sb.players {
		if player.Position == "G" && player.Team != "" {
			countryPlayers[player.Team] = append(countryPlayers[player.Team], player)
		}
	}

	// Build country stacks (2-3 players)
	for country, players := range countryPlayers {
		if len(players) < 2 {
			continue
		}

		// 2-player country stacks
		for i := 0; i < len(players)-1; i++ {
			for j := i + 1; j < len(players); j++ {
				stack := Stack{
					Type:            TeamStack, // Using TeamStack type for country
					Team:            country,
					Players:         []models.Player{players[i], players[j]},
					TotalSalary:     players[i].Salary + players[j].Salary,
					ProjectedPoints: players[i].ProjectedPoints + players[j].ProjectedPoints,
				}
				stack.CorrelationScore = 0.15 // Country correlation bonus
				stacks = append(stacks, stack)
			}
		}

		// 3-player country stacks
		if len(players) >= 3 {
			for i := 0; i < len(players)-2; i++ {
				for j := i + 1; j < len(players)-1; j++ {
					for k := j + 1; k < len(players); k++ {
						stack := Stack{
							Type:            TeamStack,
							Team:            country,
							Players:         []models.Player{players[i], players[j], players[k]},
							TotalSalary:     players[i].Salary + players[j].Salary + players[k].Salary,
							ProjectedPoints: players[i].ProjectedPoints + players[j].ProjectedPoints + players[k].ProjectedPoints,
						}
						stack.CorrelationScore = 0.20 // Slightly higher for 3-player stacks
						stacks = append(stacks, stack)
					}
				}
			}
		}
	}

	return stacks
}

func (sb *StackBuilder) buildGolfOwnershipStacks() []Stack {
	stacks := make([]Stack, 0)

	// Separate high and low ownership players
	highOwned := make([]models.Player, 0)
	lowOwned := make([]models.Player, 0)
	
	for _, player := range sb.players {
		if player.Position == "G" {
			if player.Ownership > 20 {
				highOwned = append(highOwned, player)
			} else if player.Ownership < 10 {
				lowOwned = append(lowOwned, player)
			}
		}
	}

	// Pair high-owned with low-owned players
	for _, high := range highOwned {
		for _, low := range lowOwned {
			// Only stack if combined salary is reasonable
			if high.Salary+low.Salary <= 18000 { // Avg 9k per player
				stack := Stack{
					Type:            MiniStack,
					Players:         []models.Player{high, low},
					TotalSalary:     high.Salary + low.Salary,
					ProjectedPoints: high.ProjectedPoints + low.ProjectedPoints,
				}
				// Bonus for ownership leverage
				ownershipDiff := high.Ownership - low.Ownership
				stack.CorrelationScore = 0.05 + (ownershipDiff / 100)
				stacks = append(stacks, stack)
			}
		}
	}

	return stacks
}

func (sb *StackBuilder) buildGolfValueStacks() []Stack {
	stacks := make([]Stack, 0)

	// Separate expensive and cheap players
	stars := make([]models.Player, 0)
	scrubs := make([]models.Player, 0)
	
	for _, player := range sb.players {
		if player.Position == "G" {
			if player.Salary >= 10000 {
				stars = append(stars, player)
			} else if player.Salary <= 7000 {
				scrubs = append(scrubs, player)
			}
		}
	}

	// Build stars and scrubs combinations
	for _, star := range stars {
		// Find 2-3 cheap players to pair with each star
		affordableScrubs := make([]models.Player, 0)
		for _, scrub := range scrubs {
			if star.Salary+scrub.Salary <= 16000 { // Leave room for others
				affordableScrubs = append(affordableScrubs, scrub)
			}
		}

		// Create 1 star + 2 scrubs stacks
		if len(affordableScrubs) >= 2 {
			for i := 0; i < len(affordableScrubs)-1; i++ {
				for j := i + 1; j < len(affordableScrubs); j++ {
					totalSalary := star.Salary + affordableScrubs[i].Salary + affordableScrubs[j].Salary
					if totalSalary <= 23000 { // ~7.7k average
						stack := Stack{
							Type:            MiniStack,
							Players:         []models.Player{star, affordableScrubs[i], affordableScrubs[j]},
							TotalSalary:     totalSalary,
							ProjectedPoints: star.ProjectedPoints + affordableScrubs[i].ProjectedPoints + affordableScrubs[j].ProjectedPoints,
						}
						// Value correlation bonus
						valueScore := (star.ProjectedPoints/float64(star.Salary)*1000 + 
									  affordableScrubs[i].ProjectedPoints/float64(affordableScrubs[i].Salary)*1000 +
									  affordableScrubs[j].ProjectedPoints/float64(affordableScrubs[j].Salary)*1000) / 3.0
						stack.CorrelationScore = valueScore * 0.1
						stacks = append(stacks, stack)
					}
				}
			}
		}
	}

	// Sort by value score
	sort.Slice(stacks, func(i, j int) bool {
		valueI := stacks[i].ProjectedPoints / float64(stacks[i].TotalSalary)
		valueJ := stacks[j].ProjectedPoints / float64(stacks[j].TotalSalary)
		return valueI > valueJ
	})

	// Return top value stacks
	if len(stacks) > 20 {
		return stacks[:20]
	}
	return stacks
}
