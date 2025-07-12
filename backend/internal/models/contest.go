package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type Contest struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Platform          string    `gorm:"not null" json:"platform"`     // "draftkings" or "fanduel"
	Sport             string    `gorm:"not null" json:"sport"`        // "nba", "nfl", "mlb", "nhl", "golf"
	ContestType       string    `gorm:"not null" json:"contest_type"` // "gpp" or "cash"
	Name              string    `gorm:"not null" json:"name"`
	EntryFee          float64   `json:"entry_fee"`
	PrizePool         float64   `json:"prize_pool"`
	MaxEntries        int       `json:"max_entries"`
	TotalEntries      int       `json:"total_entries"`
	SalaryCap         int       `gorm:"not null" json:"salary_cap"`
	StartTime         time.Time `gorm:"not null" json:"start_time"`
	IsActive          bool      `gorm:"default:true" json:"is_active"`
	IsMultiEntry      bool      `gorm:"default:false" json:"is_multi_entry"`
	MaxLineupsPerUser int       `gorm:"default:1" json:"max_lineups_per_user"`
	LastDataUpdate    time.Time `json:"last_data_update"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	TournamentID      *string   `gorm:"type:uuid" json:"tournament_id,omitempty"`
	ExternalID        string    `gorm:"index" json:"external_id"`    // DraftKings contest ID
	DraftGroupID      string    `gorm:"index" json:"draft_group_id"` // DraftKings draft group ID
	LastSyncTime      time.Time `json:"last_sync_time"`              // Last time contest was synced from external source

	// Relationships
	Tournament *GolfTournament `gorm:"foreignKey:TournamentID" json:"tournament,omitempty"`

	// Position requirements stored as JSON
	PositionRequirements PositionRequirements `gorm:"type:jsonb" json:"position_requirements"`
}

// TableName specifies the table name for GORM
func (Contest) TableName() string {
	return "contests"
}

// PositionRequirements defines how many players needed for each position
type PositionRequirements map[string]int

// Scan implements the sql.Scanner interface for JSONB
func (pr *PositionRequirements) Scan(value interface{}) error {
	if value == nil {
		*pr = make(PositionRequirements)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into PositionRequirements", value)
	}

	var result map[string]int
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*pr = PositionRequirements(result)
	return nil
}

// Value implements the driver.Valuer interface for JSONB
func (pr PositionRequirements) Value() (driver.Value, error) {
	if pr == nil {
		return nil, nil
	}
	return json.Marshal(pr)
}

// GetPositionRequirements returns position requirements for each sport/platform
func GetPositionRequirements(sport, platform string) PositionRequirements {
	requirements := make(PositionRequirements)

	switch sport {
	case "nba":
		if platform == "draftkings" {
			requirements = PositionRequirements{
				"PG":   1,
				"SG":   1,
				"SF":   1,
				"PF":   1,
				"C":    1,
				"G":    1, // Guard (PG or SG)
				"F":    1, // Forward (SF or PF)
				"UTIL": 1, // Any position
			}
		} else if platform == "fanduel" {
			requirements = PositionRequirements{
				"PG": 2,
				"SG": 2,
				"SF": 2,
				"PF": 2,
				"C":  1,
			}
		}
	case "nfl":
		if platform == "draftkings" {
			requirements = PositionRequirements{
				"QB":   1,
				"RB":   2,
				"WR":   3,
				"TE":   1,
				"FLEX": 1, // RB/WR/TE
				"DST":  1,
			}
		} else if platform == "fanduel" {
			requirements = PositionRequirements{
				"QB":   1,
				"RB":   2,
				"WR":   3,
				"TE":   1,
				"FLEX": 1,
				"D/ST": 1,
			}
		}
	case "mlb":
		if platform == "draftkings" {
			requirements = PositionRequirements{
				"P":  2,
				"C":  1,
				"1B": 1,
				"2B": 1,
				"3B": 1,
				"SS": 1,
				"OF": 3,
			}
		} else if platform == "fanduel" {
			requirements = PositionRequirements{
				"P":    1,
				"C/1B": 1,
				"2B":   1,
				"3B":   1,
				"SS":   1,
				"OF":   3,
				"UTIL": 1,
			}
		}
	case "nhl":
		if platform == "draftkings" {
			requirements = PositionRequirements{
				"C":    2,
				"W":    3,
				"D":    2,
				"G":    1,
				"UTIL": 1,
			}
		} else if platform == "fanduel" {
			requirements = PositionRequirements{
				"C": 2,
				"W": 4,
				"D": 2,
				"G": 1,
			}
		}
	case "golf":
		// Both DraftKings and FanDuel use 6 golfers for PGA
		requirements = PositionRequirements{
			"G": 6, // 6 golfers
		}
	}

	return requirements
}

// GetTotalPlayers returns the total number of players required for a lineup
func (pr PositionRequirements) GetTotalPlayers() int {
	total := 0
	for _, count := range pr {
		total += count
	}
	return total
}

// ValidateLineupSize checks if the lineup has the correct number of players
func (c *Contest) ValidateLineupSize(playerCount int) bool {
	return playerCount == c.PositionRequirements.GetTotalPlayers()
}
