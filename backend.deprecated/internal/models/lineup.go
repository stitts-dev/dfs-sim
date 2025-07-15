package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Lineup struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"not null;index:idx_user_contest" json:"user_id"`
	ContestID        uint      `gorm:"not null;index:idx_user_contest" json:"contest_id"`
	Name             string    `json:"name"`
	TotalSalary      int       `gorm:"not null" json:"total_salary"`
	ProjectedPoints  float64   `gorm:"not null" json:"projected_points"`
	ActualPoints     *float64  `json:"actual_points,omitempty"` // Null until contest completes
	SimulatedCeiling float64   `json:"simulated_ceiling"`
	SimulatedFloor   float64   `json:"simulated_floor"`
	SimulatedMean    float64   `json:"simulated_mean"`
	Ownership        float64   `json:"ownership"`
	IsSubmitted      bool      `gorm:"default:false" json:"is_submitted"`
	IsOptimized      bool      `gorm:"default:false" json:"is_optimized"`
	OptimizationRank int       `json:"optimization_rank,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// Associations
	Contest Contest  `gorm:"foreignKey:ContestID" json:"contest,omitempty"`
	Players []Player `gorm:"-" json:"players"`

	// Position assignments for each player (not stored in DB, used for saving)
	PlayerPositions map[uint]string `gorm:"-" json:"player_positions,omitempty"`
}

// TableName specifies the table name for GORM
func (Lineup) TableName() string {
	return "lineups"
}

// LineupPlayer represents the join table for lineup-player relationships
type LineupPlayer struct {
	LineupID uint   `gorm:"primaryKey"`
	PlayerID uint   `gorm:"primaryKey"`
	Position string `gorm:"not null"` // The position the player fills in this lineup
}

func (LineupPlayer) TableName() string {
	return "lineup_players"
}

// CalculateTotalSalary calculates the total salary of all players in the lineup
func (l *Lineup) CalculateTotalSalary() int {
	total := 0
	for _, player := range l.Players {
		total += player.Salary
	}
	l.TotalSalary = total
	return total
}

// CalculateProjectedPoints calculates the total projected points for the lineup
func (l *Lineup) CalculateProjectedPoints() float64 {
	total := 0.0
	for _, player := range l.Players {
		total += player.ProjectedPoints
	}
	l.ProjectedPoints = total
	return total
}

// CalculateOwnership calculates the average ownership of the lineup
func (l *Lineup) CalculateOwnership() float64 {
	if len(l.Players) == 0 {
		return 0
	}

	total := 0.0
	for _, player := range l.Players {
		total += player.Ownership
	}
	l.Ownership = total / float64(len(l.Players))
	return l.Ownership
}

// ValidateSalaryCap checks if the lineup is under the salary cap
func (l *Lineup) ValidateSalaryCap(salaryCap int) error {
	if l.CalculateTotalSalary() > salaryCap {
		return fmt.Errorf("lineup exceeds salary cap: %d > %d", l.TotalSalary, salaryCap)
	}
	return nil
}

// ValidatePositions checks if the lineup meets position requirements
func (l *Lineup) ValidatePositions(requirements PositionRequirements) error {
	positionCounts := make(map[string]int)

	// Count players by position
	for _, player := range l.Players {
		positionCounts[player.Position]++
	}

	// Check if all requirements are met
	for position, required := range requirements {
		if position == "UTIL" || position == "FLEX" {
			// Handle flex positions separately
			continue
		}

		actual := positionCounts[position]
		if actual != required {
			return fmt.Errorf("position %s requires %d players, got %d", position, required, actual)
		}
	}

	return nil
}

// GetPlayersByPosition returns all players in the lineup for a specific position
func (l *Lineup) GetPlayersByPosition(position string) []Player {
	var players []Player
	for _, player := range l.Players {
		if player.Position == position {
			players = append(players, player)
		}
	}
	return players
}

// HasPlayer checks if a player is already in the lineup
func (l *Lineup) HasPlayer(playerID uint) bool {
	for _, player := range l.Players {
		if player.ID == playerID {
			return true
		}
	}
	return false
}

// GetTeamExposure returns a map of team abbreviations to player count
func (l *Lineup) GetTeamExposure() map[string]int {
	exposure := make(map[string]int)
	for _, player := range l.Players {
		exposure[player.Team]++
	}
	return exposure
}

// GetGameExposure returns a map of game matchups to player count
func (l *Lineup) GetGameExposure() map[string]int {
	exposure := make(map[string]int)
	games := make(map[string]bool)

	for _, player := range l.Players {
		// Create a unique game identifier
		var gameKey string
		if player.Team < player.Opponent {
			gameKey = fmt.Sprintf("%s@%s", player.Team, player.Opponent)
		} else {
			gameKey = fmt.Sprintf("%s@%s", player.Opponent, player.Team)
		}

		if !games[gameKey] {
			games[gameKey] = true
			exposure[gameKey] = 0
		}
		exposure[gameKey]++
	}

	return exposure
}

// Clone creates a deep copy of the lineup
func (l *Lineup) Clone() *Lineup {
	clone := &Lineup{
		UserID:          l.UserID,
		ContestID:       l.ContestID,
		Name:            l.Name,
		TotalSalary:     l.TotalSalary,
		ProjectedPoints: l.ProjectedPoints,
		IsOptimized:     l.IsOptimized,
		Players:         make([]Player, len(l.Players)),
	}

	copy(clone.Players, l.Players)
	return clone
}

// LoadPlayers loads the players for this lineup from the database
func (l *Lineup) LoadPlayers(db *gorm.DB) error {
	var lineupPlayers []LineupPlayer
	if err := db.Where("lineup_id = ?", l.ID).Find(&lineupPlayers).Error; err != nil {
		return err
	}

	playerIDs := make([]uint, len(lineupPlayers))
	l.PlayerPositions = make(map[uint]string)

	for i, lp := range lineupPlayers {
		playerIDs[i] = lp.PlayerID
		l.PlayerPositions[lp.PlayerID] = lp.Position
	}

	return db.Where("id IN ?", playerIDs).Find(&l.Players).Error
}
