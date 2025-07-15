package optimizer

import (
	"fmt"

	"github.com/stitts-dev/dfs-sim/shared/types"
)

// PositionConstraint defines constraints for a specific position
type PositionConstraint struct {
	Position      string
	MinRequired   int
	MaxAllowed    int
	EligibleSlots []string // Other positions this can fill (e.g., PG can fill G slot)
}

// LineupConstraints holds all constraints for lineup validation
type LineupConstraints struct {
	SalaryCap           int
	PositionConstraints map[string]PositionConstraint
	MinPlayersPerTeam   int
	MaxPlayersPerTeam   int
	MinPlayersPerGame   int
	MaxPlayersPerGame   int
	MinUniqueTeams      int
	MinUniqueGames      int
}

// GetConstraintsForContest returns the constraints for a specific contest
func GetConstraintsForContest(contest *types.Contest) *LineupConstraints {
	constraints := &LineupConstraints{
		SalaryCap:           contest.SalaryCap,
		PositionConstraints: make(map[string]PositionConstraint),
		MinPlayersPerTeam:   0,
		MaxPlayersPerTeam:   4, // Default max
		MinPlayersPerGame:   0,
		MaxPlayersPerGame:   6, // Default max
		MinUniqueTeams:      2, // At least 2 different teams
		MinUniqueGames:      1, // At least 1 game
	}

	// Set position constraints based on sport and platform
	sportName := getSportNameFromID(contest.SportID)
	switch sportName {
	case "nba":
		constraints.setupNBAConstraints(contest.Platform)
	case "nfl":
		constraints.setupNFLConstraints(contest.Platform)
	case "mlb":
		constraints.setupMLBConstraints(contest.Platform)
	case "nhl":
		constraints.setupNHLConstraints(contest.Platform)
	case "golf":
		constraints.setupGolfConstraints(contest.Platform)
	}

	return constraints
}

func (lc *LineupConstraints) setupNBAConstraints(platform string) {
	if platform == "draftkings" {
		lc.PositionConstraints = map[string]PositionConstraint{
			"PG": {Position: "PG", MinRequired: 1, MaxAllowed: 3, EligibleSlots: []string{"G", "UTIL"}},
			"SG": {Position: "SG", MinRequired: 1, MaxAllowed: 3, EligibleSlots: []string{"G", "UTIL"}},
			"SF": {Position: "SF", MinRequired: 1, MaxAllowed: 3, EligibleSlots: []string{"F", "UTIL"}},
			"PF": {Position: "PF", MinRequired: 1, MaxAllowed: 3, EligibleSlots: []string{"F", "UTIL"}},
			"C":  {Position: "C", MinRequired: 1, MaxAllowed: 2, EligibleSlots: []string{"UTIL"}},
		}
		lc.MaxPlayersPerTeam = 4
	} else if platform == "fanduel" {
		lc.PositionConstraints = map[string]PositionConstraint{
			"PG": {Position: "PG", MinRequired: 2, MaxAllowed: 2},
			"SG": {Position: "SG", MinRequired: 2, MaxAllowed: 2},
			"SF": {Position: "SF", MinRequired: 2, MaxAllowed: 2},
			"PF": {Position: "PF", MinRequired: 2, MaxAllowed: 2},
			"C":  {Position: "C", MinRequired: 1, MaxAllowed: 1},
		}
		lc.MaxPlayersPerTeam = 4
	}
}

func (lc *LineupConstraints) setupNFLConstraints(platform string) {
	if platform == "draftkings" {
		lc.PositionConstraints = map[string]PositionConstraint{
			"QB":  {Position: "QB", MinRequired: 1, MaxAllowed: 1},
			"RB":  {Position: "RB", MinRequired: 2, MaxAllowed: 3, EligibleSlots: []string{"FLEX"}},
			"WR":  {Position: "WR", MinRequired: 3, MaxAllowed: 4, EligibleSlots: []string{"FLEX"}},
			"TE":  {Position: "TE", MinRequired: 1, MaxAllowed: 2, EligibleSlots: []string{"FLEX"}},
			"DST": {Position: "DST", MinRequired: 1, MaxAllowed: 1},
		}
		lc.MaxPlayersPerTeam = 8 // Can stack entire offense
	} else if platform == "fanduel" {
		lc.PositionConstraints = map[string]PositionConstraint{
			"QB":   {Position: "QB", MinRequired: 1, MaxAllowed: 1},
			"RB":   {Position: "RB", MinRequired: 2, MaxAllowed: 3, EligibleSlots: []string{"FLEX"}},
			"WR":   {Position: "WR", MinRequired: 3, MaxAllowed: 4, EligibleSlots: []string{"FLEX"}},
			"TE":   {Position: "TE", MinRequired: 1, MaxAllowed: 2, EligibleSlots: []string{"FLEX"}},
			"D/ST": {Position: "D/ST", MinRequired: 1, MaxAllowed: 1},
		}
		lc.MaxPlayersPerTeam = 8
	}
}

func (lc *LineupConstraints) setupMLBConstraints(platform string) {
	if platform == "draftkings" {
		lc.PositionConstraints = map[string]PositionConstraint{
			"P":  {Position: "P", MinRequired: 2, MaxAllowed: 2},
			"C":  {Position: "C", MinRequired: 1, MaxAllowed: 1},
			"1B": {Position: "1B", MinRequired: 1, MaxAllowed: 1},
			"2B": {Position: "2B", MinRequired: 1, MaxAllowed: 1},
			"3B": {Position: "3B", MinRequired: 1, MaxAllowed: 1},
			"SS": {Position: "SS", MinRequired: 1, MaxAllowed: 1},
			"OF": {Position: "OF", MinRequired: 3, MaxAllowed: 3},
		}
		lc.MaxPlayersPerTeam = 5
	} else if platform == "fanduel" {
		lc.PositionConstraints = map[string]PositionConstraint{
			"P":    {Position: "P", MinRequired: 1, MaxAllowed: 1},
			"C/1B": {Position: "C/1B", MinRequired: 1, MaxAllowed: 1},
			"2B":   {Position: "2B", MinRequired: 1, MaxAllowed: 1},
			"3B":   {Position: "3B", MinRequired: 1, MaxAllowed: 1},
			"SS":   {Position: "SS", MinRequired: 1, MaxAllowed: 1},
			"OF":   {Position: "OF", MinRequired: 3, MaxAllowed: 3},
			"UTIL": {Position: "UTIL", MinRequired: 1, MaxAllowed: 1},
		}
		lc.MaxPlayersPerTeam = 4
	}
}

func (lc *LineupConstraints) setupNHLConstraints(platform string) {
	if platform == "draftkings" {
		lc.PositionConstraints = map[string]PositionConstraint{
			"C":    {Position: "C", MinRequired: 2, MaxAllowed: 3, EligibleSlots: []string{"UTIL"}},
			"W":    {Position: "W", MinRequired: 3, MaxAllowed: 4, EligibleSlots: []string{"UTIL"}},
			"D":    {Position: "D", MinRequired: 2, MaxAllowed: 3, EligibleSlots: []string{"UTIL"}},
			"G":    {Position: "G", MinRequired: 1, MaxAllowed: 1},
			"UTIL": {Position: "UTIL", MinRequired: 1, MaxAllowed: 1},
		}
		lc.MaxPlayersPerTeam = 8
	} else if platform == "fanduel" {
		lc.PositionConstraints = map[string]PositionConstraint{
			"C": {Position: "C", MinRequired: 2, MaxAllowed: 2},
			"W": {Position: "W", MinRequired: 4, MaxAllowed: 4},
			"D": {Position: "D", MinRequired: 2, MaxAllowed: 2},
			"G": {Position: "G", MinRequired: 1, MaxAllowed: 1},
		}
		lc.MaxPlayersPerTeam = 8
	}
}

func (lc *LineupConstraints) setupGolfConstraints(platform string) {
	// Golf has no position constraints - just 6 golfers
	lc.PositionConstraints = map[string]PositionConstraint{
		"G": {Position: "G", MinRequired: 6, MaxAllowed: 6}, // G for Golfer
	}

	// Golf-specific adjustments
	lc.MaxPlayersPerTeam = 6 // In golf, "team" represents country
	lc.MinUniqueTeams = 1    // Can have all players from same country
	lc.MinUniqueGames = 1    // All players in same tournament
	lc.MaxPlayersPerGame = 6 // All players in same tournament

	// Platform-specific salary caps
	if platform == "draftkings" {
		lc.SalaryCap = 50000
	} else if platform == "fanduel" {
		lc.SalaryCap = 60000
	}
}

// ValidateLineup performs comprehensive lineup validation
func (lc *LineupConstraints) ValidateLineup(lineup *types.GeneratedLineup) error {
	// Salary cap validation
	if err := lc.validateSalaryCap(lineup); err != nil {
		return err
	}

	// Position requirements validation
	if err := lc.validatePositions(lineup); err != nil {
		return err
	}

	// Team diversity validation
	if err := lc.validateTeamDiversity(lineup); err != nil {
		return err
	}

	// Game diversity validation
	if err := lc.validateGameDiversity(lineup); err != nil {
		return err
	}

	return nil
}

func (lc *LineupConstraints) validateSalaryCap(lineup *types.GeneratedLineup) error {
	if lineup.TotalSalary > lc.SalaryCap {
		return fmt.Errorf("lineup exceeds salary cap: %d > %d", lineup.TotalSalary, lc.SalaryCap)
	}

	if lineup.TotalSalary < int(float64(lc.SalaryCap)*0.95) {
		return fmt.Errorf("lineup leaves too much salary on table: %d < %d", lineup.TotalSalary, int(float64(lc.SalaryCap)*0.95))
	}

	return nil
}

func (lc *LineupConstraints) validatePositions(lineup *types.GeneratedLineup) error {
	positionCounts := make(map[string]int)

	// Count players by position
	for _, player := range lineup.Players {
		positionCounts[player.Position]++
	}

	// Check each position constraint
	for position, constraint := range lc.PositionConstraints {
		count := positionCounts[position]

		if count < constraint.MinRequired {
			return fmt.Errorf("position %s requires at least %d players, got %d", position, constraint.MinRequired, count)
		}

		if count > constraint.MaxAllowed {
			return fmt.Errorf("position %s allows at most %d players, got %d", position, constraint.MaxAllowed, count)
		}
	}

	return nil
}

func (lc *LineupConstraints) validateTeamDiversity(lineup *types.GeneratedLineup) error {
	teamCounts := make(map[string]int)
	for _, player := range lineup.Players {
		teamCounts[player.Team]++
	}

	// Check min/max players per team
	for team, count := range teamCounts {
		if count > lc.MaxPlayersPerTeam {
			return fmt.Errorf("too many players from team %s: %d > %d", team, count, lc.MaxPlayersPerTeam)
		}

		if lc.MinPlayersPerTeam > 0 && count < lc.MinPlayersPerTeam {
			return fmt.Errorf("too few players from team %s: %d < %d", team, count, lc.MinPlayersPerTeam)
		}
	}

	// Check minimum unique teams
	if len(teamCounts) < lc.MinUniqueTeams {
		return fmt.Errorf("lineup needs players from at least %d teams, got %d", lc.MinUniqueTeams, len(teamCounts))
	}

	return nil
}

func (lc *LineupConstraints) validateGameDiversity(lineup *types.GeneratedLineup) error {
	gameCounts := make(map[string]int)
	for _, player := range lineup.Players {
		// Create game identifier from team and opponent
		game := fmt.Sprintf("%s@%s", player.Team, "OPP") // TODO: Get actual opponent from player data
		gameCounts[game]++
	}

	// Check min/max players per game
	for game, count := range gameCounts {
		if count > lc.MaxPlayersPerGame {
			return fmt.Errorf("too many players from game %s: %d > %d", game, count, lc.MaxPlayersPerGame)
		}

		if lc.MinPlayersPerGame > 0 && count < lc.MinPlayersPerGame {
			return fmt.Errorf("too few players from game %s: %d < %d", game, count, lc.MinPlayersPerGame)
		}
	}

	// Check minimum unique games
	if len(gameCounts) < lc.MinUniqueGames {
		return fmt.Errorf("lineup needs players from at least %d games, got %d", lc.MinUniqueGames, len(gameCounts))
	}

	return nil
}

// CanFillPosition checks if a player can fill a specific lineup position
func (lc *LineupConstraints) CanFillPosition(playerPosition, lineupPosition string) bool {
	// Direct match
	if playerPosition == lineupPosition {
		return true
	}

	// Check if player's position can fill the lineup position
	if constraint, exists := lc.PositionConstraints[playerPosition]; exists {
		for _, eligible := range constraint.EligibleSlots {
			if eligible == lineupPosition {
				return true
			}
		}
	}

	return false
}

// GetRequiredPositions returns a list of all required positions for the lineup
func (lc *LineupConstraints) GetRequiredPositions() []string {
	positions := make([]string, 0)

	for position, constraint := range lc.PositionConstraints {
		for i := 0; i < constraint.MinRequired; i++ {
			positions = append(positions, position)
		}
	}

	return positions
}

// GetFlexPositions returns positions that can fill flex spots
func GetFlexPositions(sport, flexType string) []string {
	switch sport {
	case "nba":
		switch flexType {
		case "G":
			return []string{"PG", "SG"}
		case "F":
			return []string{"SF", "PF"}
		case "UTIL":
			return []string{"PG", "SG", "SF", "PF", "C"}
		}
	case "nfl":
		if flexType == "FLEX" {
			return []string{"RB", "WR", "TE"}
		}
	case "mlb":
		if flexType == "UTIL" {
			return []string{"C", "1B", "2B", "3B", "SS", "OF"}
		}
	case "nhl":
		if flexType == "UTIL" {
			return []string{"C", "W", "D"}
		}
	case "golf":
		// Golf has no flex positions - all players are "G" (Golfer)
		return []string{}
	}

	return []string{}
}
