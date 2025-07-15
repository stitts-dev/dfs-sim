package models

import (
	"time"
)

type Player struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ExternalID      string    `gorm:"uniqueIndex:idx_external_contest;not null" json:"external_id"`
	Name            string    `gorm:"not null" json:"name"`
	Team            string    `gorm:"not null" json:"team"`
	Opponent        string    `gorm:"not null" json:"opponent"`
	Position        string    `gorm:"not null" json:"position"`
	Salary          int       `gorm:"not null" json:"salary"`
	ProjectedPoints float64   `gorm:"not null" json:"projected_points"`
	FloorPoints     float64   `gorm:"not null" json:"floor_points"`
	CeilingPoints   float64   `gorm:"not null" json:"ceiling_points"`
	Ownership       float64   `json:"ownership"`
	Sport           string    `gorm:"not null" json:"sport"`
	ContestID       uint      `gorm:"uniqueIndex:idx_external_contest;not null" json:"contest_id"`
	GameTime        time.Time `gorm:"not null" json:"game_time"`
	IsInjured       bool      `gorm:"default:false" json:"is_injured"`
	InjuryStatus    string    `json:"injury_status,omitempty"`
	ImageURL        string    `json:"image_url,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Associations
	Contest Contest `gorm:"foreignKey:ContestID" json:"-"`
}

// TableName specifies the table name for GORM
func (Player) TableName() string {
	return "players"
}

// PlayerPool represents a collection of players for optimization
type PlayerPool struct {
	Players      []Player            `json:"players"`
	ByPosition   map[string][]Player `json:"-"`
	TotalPlayers int                 `json:"total_players"`
}

// NewPlayerPool creates a new player pool and organizes players by position
func NewPlayerPool(players []Player) *PlayerPool {
	pool := &PlayerPool{
		Players:      players,
		ByPosition:   make(map[string][]Player),
		TotalPlayers: len(players),
	}

	for _, player := range players {
		pool.ByPosition[player.Position] = append(pool.ByPosition[player.Position], player)
	}

	return pool
}

// GetPlayersForPosition returns all players for a specific position
func (pp *PlayerPool) GetPlayersForPosition(position string) []Player {
	return pp.ByPosition[position]
}

// FilterByTeam returns players from a specific team
func (pp *PlayerPool) FilterByTeam(team string) []Player {
	var filtered []Player
	for _, player := range pp.Players {
		if player.Team == team {
			filtered = append(filtered, player)
		}
	}
	return filtered
}

// FilterBySalaryRange returns players within a salary range
func (pp *PlayerPool) FilterBySalaryRange(minSalary, maxSalary int) []Player {
	var filtered []Player
	for _, player := range pp.Players {
		if player.Salary >= minSalary && player.Salary <= maxSalary {
			filtered = append(filtered, player)
		}
	}
	return filtered
}
