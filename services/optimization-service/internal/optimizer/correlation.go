package optimizer

import (
	"math"

	"github.com/stitts-dev/dfs-sim/shared/types"
)

// CorrelationMatrix represents correlations between players
type CorrelationMatrix struct {
	correlations map[uint]map[uint]float64
	byTeam       map[string][]uint
	byGame       map[string][]uint
	byPosition   map[string][]uint
}

// NewCorrelationMatrix creates a new correlation matrix from players
func NewCorrelationMatrix(players []OptimizationPlayer) *CorrelationMatrix {
	cm := &CorrelationMatrix{
		correlations: make(map[uint]map[uint]float64),
		byTeam:       make(map[string][]uint),
		byGame:       make(map[string][]uint),
		byPosition:   make(map[string][]uint),
	}

	// Organize players
	for _, player := range players {
		playerID := uint(player.ID.ID())
		
		// Direct field access since OptimizationPlayer has concrete fields
		team := player.Team
		position := player.Position
		opponent := player.Opponent
		
		cm.byTeam[team] = append(cm.byTeam[team], playerID)
		cm.byPosition[position] = append(cm.byPosition[position], playerID)

		gameKey := getGameKey(team, opponent)
		cm.byGame[gameKey] = append(cm.byGame[gameKey], playerID)
	}

	// Calculate correlations
	cm.calculateCorrelations(players)

	return cm
}

func (cm *CorrelationMatrix) calculateCorrelations(players []OptimizationPlayer) {
	playerMap := make(map[uint]OptimizationPlayer)
	for _, p := range players {
		playerMap[uint(p.ID.ID())] = p
	}

	// Calculate correlations for each player pair
	for i := 0; i < len(players); i++ {
		p1 := players[i]

		p1ID := uint(p1.ID.ID())
		if cm.correlations[p1ID] == nil {
			cm.correlations[p1ID] = make(map[uint]float64)
		}

		for j := i + 1; j < len(players); j++ {
			p2 := players[j]

			p2ID := uint(p2.ID.ID())
			if cm.correlations[p2ID] == nil {
				cm.correlations[p2ID] = make(map[uint]float64)
			}

			corr := cm.calculatePairCorrelation(p1, p2)
			cm.correlations[p1ID][p2ID] = corr
			cm.correlations[p2ID][p1ID] = corr
		}
	}
}

func (cm *CorrelationMatrix) calculatePairCorrelation(p1, p2 OptimizationPlayer) float64 {
	correlation := 0.0

	// Same team correlation (teammates)
	p1Team := p1.Team
	p2Team := p2.Team
	p1Position := p1.Position
	p2Position := p2.Position
	
	if p1Team == p2Team && p1Team != "" {
		// Use sport name from position - golf players are "G"
		sport := "golf" // Default to golf for now, could be enhanced
		if p1Position != "G" {
			sport = "nba" // Simple heuristic for other sports
		}
		correlation += cm.getTeammateCorrelation(p1Position, p2Position, sport)
	}

	// Same game correlation
	p1Opponent := p1.Opponent
	p2Opponent := p2.Opponent
	
	if getGameKey(p1Team, p1Opponent) == getGameKey(p2Team, p2Opponent) {
		if p1Team != p2Team {
			// Opponents
			sport := "golf" // Default to golf for now
			if p1Position != "G" {
				sport = "nba" // Simple heuristic for other sports
			}
			correlation += cm.getOpponentCorrelation(p1Position, p2Position, sport)
		}
	}

	// Cap correlation between -1 and 1
	return math.Max(-1.0, math.Min(1.0, correlation))
}

func (cm *CorrelationMatrix) getTeammateCorrelation(pos1, pos2, sport string) float64 {
	switch sport {
	case "nba":
		return cm.getNBATeammateCorrelation(pos1, pos2)
	case "nfl":
		return cm.getNFLTeammateCorrelation(pos1, pos2)
	case "mlb":
		return cm.getMLBTeammateCorrelation(pos1, pos2)
	case "nhl":
		return cm.getNHLTeammateCorrelation(pos1, pos2)
	case "golf":
		return cm.getGolfTeammateCorrelation(pos1, pos2)
	default:
		return 0.2 // Default positive correlation for teammates
	}
}

func (cm *CorrelationMatrix) getNBATeammateCorrelation(pos1, pos2 string) float64 {
	// NBA correlations based on position pairs
	correlations := map[string]map[string]float64{
		"PG": {"PG": 0.0, "SG": 0.35, "SF": 0.25, "PF": 0.20, "C": 0.30},
		"SG": {"PG": 0.35, "SG": 0.0, "SF": 0.20, "PF": 0.15, "C": 0.25},
		"SF": {"PG": 0.25, "SG": 0.20, "SF": 0.0, "PF": 0.20, "C": 0.20},
		"PF": {"PG": 0.20, "SG": 0.15, "SF": 0.20, "PF": 0.0, "C": 0.35},
		"C":  {"PG": 0.30, "SG": 0.25, "SF": 0.20, "PF": 0.35, "C": 0.0},
	}

	if corr, exists := correlations[pos1][pos2]; exists {
		return corr
	}
	return 0.2
}

func (cm *CorrelationMatrix) getNFLTeammateCorrelation(pos1, pos2 string) float64 {
	// NFL correlations - QB stacks are key
	correlations := map[string]map[string]float64{
		"QB":  {"QB": 0.0, "RB": 0.10, "WR": 0.50, "TE": 0.40, "DST": -0.20},
		"RB":  {"QB": 0.10, "RB": -0.30, "WR": -0.10, "TE": -0.05, "DST": 0.15},
		"WR":  {"QB": 0.50, "RB": -0.10, "WR": 0.25, "TE": 0.10, "DST": -0.10},
		"TE":  {"QB": 0.40, "RB": -0.05, "WR": 0.10, "TE": 0.0, "DST": -0.05},
		"DST": {"QB": -0.20, "RB": 0.15, "WR": -0.10, "TE": -0.05, "DST": 0.0},
	}

	if corr, exists := correlations[pos1][pos2]; exists {
		return corr
	}
	return 0.1
}

func (cm *CorrelationMatrix) getMLBTeammateCorrelation(pos1, pos2 string) float64 {
	// MLB correlations - batting order matters
	correlations := map[string]map[string]float64{
		"P":  {"P": -0.50, "C": 0.20, "1B": 0.0, "2B": 0.0, "3B": 0.0, "SS": 0.0, "OF": 0.0},
		"C":  {"P": 0.20, "C": 0.0, "1B": 0.10, "2B": 0.10, "3B": 0.10, "SS": 0.10, "OF": 0.10},
		"1B": {"P": 0.0, "C": 0.10, "1B": 0.0, "2B": 0.25, "3B": 0.20, "SS": 0.20, "OF": 0.30},
		"2B": {"P": 0.0, "C": 0.10, "1B": 0.25, "2B": 0.0, "3B": 0.25, "SS": 0.30, "OF": 0.25},
		"3B": {"P": 0.0, "C": 0.10, "1B": 0.20, "2B": 0.25, "3B": 0.0, "SS": 0.25, "OF": 0.25},
		"SS": {"P": 0.0, "C": 0.10, "1B": 0.20, "2B": 0.30, "3B": 0.25, "SS": 0.0, "OF": 0.25},
		"OF": {"P": 0.0, "C": 0.10, "1B": 0.30, "2B": 0.25, "3B": 0.25, "SS": 0.25, "OF": 0.35},
	}

	if corr, exists := correlations[pos1][pos2]; exists {
		return corr
	}
	return 0.15
}

func (cm *CorrelationMatrix) getNHLTeammateCorrelation(pos1, pos2 string) float64 {
	// NHL correlations - line mates are key
	correlations := map[string]map[string]float64{
		"C": {"C": 0.20, "W": 0.45, "D": 0.25, "G": 0.30},
		"W": {"C": 0.45, "W": 0.40, "D": 0.20, "G": 0.30},
		"D": {"C": 0.25, "W": 0.20, "D": 0.35, "G": 0.35},
		"G": {"C": 0.30, "W": 0.30, "D": 0.35, "G": 0.0},
	}

	if corr, exists := correlations[pos1][pos2]; exists {
		return corr
	}
	return 0.2
}

func (cm *CorrelationMatrix) getGolfTeammateCorrelation(pos1, pos2 string) float64 {
	// Golf players don't have traditional teammates, but can have course/tee time correlations
	// Return small positive correlation for similar tee times/conditions
	return 0.1
}

func (cm *CorrelationMatrix) getOpponentCorrelation(pos1, pos2, sport string) float64 {
	switch sport {
	case "nba":
		// Game environment correlation
		return 0.15
	case "nfl":
		// QB vs opposing pass catchers
		if pos1 == "QB" && (pos2 == "WR" || pos2 == "TE") {
			return 0.25
		}
		if pos2 == "QB" && (pos1 == "WR" || pos1 == "TE") {
			return 0.25
		}
		// RB vs DEF negative
		if (pos1 == "RB" && pos2 == "DST") || (pos1 == "DST" && pos2 == "RB") {
			return -0.30
		}
		return 0.10
	case "mlb":
		// Pitcher vs hitters negative
		if pos1 == "P" || pos2 == "P" {
			return -0.25
		}
		return 0.10
	case "nhl":
		// Goalie vs skaters negative
		if pos1 == "G" || pos2 == "G" {
			return -0.20
		}
		return 0.15
	case "golf":
		// Golf doesn't have direct opponents
		return cm.getGolfOpponentCorrelation(pos1, pos2)
	default:
		return 0.05
	}
}

func (cm *CorrelationMatrix) getGolfOpponentCorrelation(pos1, pos2 string) float64 {
	// Golf doesn't have direct opponents like other sports
	// Return small positive correlation for playing in same tournament conditions
	return 0.05
}

// GetCorrelation returns the correlation between two players
func (cm *CorrelationMatrix) GetCorrelation(player1ID, player2ID uint) float64 {
	if player1ID == player2ID {
		return 1.0
	}

	if corr, exists := cm.correlations[player1ID][player2ID]; exists {
		return corr
	}

	return 0.0
}

// GetTeammates returns all teammates for a player
func (cm *CorrelationMatrix) GetTeammates(playerID uint, players []types.Player) []uint {
	var playerTeam string
	for _, p := range players {
		if uint(p.ID.ID()) == playerID {
			if p.Team != nil {
				playerTeam = *p.Team
			}
			break
		}
	}

	teammates := make([]uint, 0)
	for _, id := range cm.byTeam[playerTeam] {
		if id != playerID {
			teammates = append(teammates, id)
		}
	}

	return teammates
}

// GetGamePartners returns all players in the same game
func (cm *CorrelationMatrix) GetGamePartners(playerID uint, players []types.Player) []uint {
	var gameKey string
	for _, p := range players {
		if uint(p.ID.ID()) == playerID {
			team := ""
			if p.Team != nil {
				team = *p.Team
			}
			opponent := ""
			if p.Opponent != nil {
				opponent = *p.Opponent
			}
			gameKey = getGameKey(team, opponent)
			break
		}
	}

	partners := make([]uint, 0)
	for _, id := range cm.byGame[gameKey] {
		if id != playerID {
			partners = append(partners, id)
		}
	}

	return partners
}

// CalculateLineupCorrelation calculates the total correlation score for a lineup
func (cm *CorrelationMatrix) CalculateLineupCorrelation(lineup []OptimizationPlayer) float64 {
	if len(lineup) < 2 {
		return 0.0
	}

	totalCorrelation := 0.0
	count := 0

	// Sum all pairwise correlations
	for i := 0; i < len(lineup); i++ {
		for j := i + 1; j < len(lineup); j++ {
			player1ID := uint(lineup[i].ID.ID())
			player2ID := uint(lineup[j].ID.ID())
			corr := cm.GetCorrelation(player1ID, player2ID)
			totalCorrelation += corr
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	// Return average correlation
	return totalCorrelation / float64(count)
}

// GetStronglyCorrelatedPlayers returns players with high correlation to the given player
func (cm *CorrelationMatrix) GetStronglyCorrelatedPlayers(playerID uint, threshold float64) []uint {
	correlated := make([]uint, 0)

	if playerCorrs, exists := cm.correlations[playerID]; exists {
		for otherID, corr := range playerCorrs {
			if corr >= threshold {
				correlated = append(correlated, otherID)
			}
		}
	}

	return correlated
}

// GetNegativelyCorrelatedPlayers returns players with negative correlation to the given player
func (cm *CorrelationMatrix) GetNegativelyCorrelatedPlayers(playerID uint, threshold float64) []uint {
	negCorrelated := make([]uint, 0)

	if playerCorrs, exists := cm.correlations[playerID]; exists {
		for otherID, corr := range playerCorrs {
			if corr <= -threshold {
				negCorrelated = append(negCorrelated, otherID)
			}
		}
	}

	return negCorrelated
}
