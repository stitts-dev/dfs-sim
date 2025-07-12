package dfs

import (
	"time"
)

// PlayerData represents player data from external APIs
type PlayerData struct {
	ExternalID  string             `json:"external_id"`
	Name        string             `json:"name"`
	Team        string             `json:"team"`
	Position    string             `json:"position"`
	Stats       map[string]float64 `json:"stats"`
	ImageURL    string             `json:"image_url,omitempty"`
	LastUpdated time.Time          `json:"last_updated"`
	Source      string             `json:"source"` // "espn", "thesportsdb", "balldontlie"
}

// AggregatedPlayer combines data from multiple sources
type AggregatedPlayer struct {
	PlayerID        string             `json:"player_id"`
	Name            string             `json:"name"`
	Team            string             `json:"team"`
	Position        string             `json:"position"`
	Salary          float64            `json:"salary"`
	ProjectedPoints float64            `json:"projected_points"`
	Confidence      float64            `json:"confidence"` // Based on data availability
	ESPNData        *PlayerData        `json:"espn_data,omitempty"`
	TheSportsDBData *PlayerData        `json:"thesportsdb_data,omitempty"`
	BallDontLieData *PlayerData        `json:"balldontlie_data,omitempty"`
	DraftKingsData  *PlayerData        `json:"draftkings_data,omitempty"`
	Stats           map[string]float64 `json:"stats"`
	LastUpdated     time.Time          `json:"last_updated"`
}

// Sport represents the sport type
type Sport string

const (
	SportNBA  Sport = "nba"
	SportNFL  Sport = "nfl"
	SportMLB  Sport = "mlb"
	SportNHL  Sport = "nhl"
	SportGolf Sport = "golf"
)

// Provider interface for all external data providers
type Provider interface {
	GetPlayers(sport Sport, date string) ([]PlayerData, error)
	GetPlayer(sport Sport, externalID string) (*PlayerData, error)
	GetTeamRoster(sport Sport, teamID string) ([]PlayerData, error)
}

// CacheProvider interface for cache operations
type CacheProvider interface {
	SetSimple(key string, value interface{}, expiration time.Duration) error
	GetSimple(key string, dest interface{}) error
}
