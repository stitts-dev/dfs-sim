package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ServiceResponse represents a standard response from microservices
type ServiceResponse struct {
	StatusCode int         `json:"status_code"`
	Body       interface{} `json:"body"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// HealthStatus represents the health status of a service
type HealthStatus struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// Player represents a DFS player (shared across all services)
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
}

// Contest represents a DFS contest (shared across all services)
type Contest struct {
	ID                   uint                 `gorm:"primaryKey" json:"id"`
	Platform             string               `gorm:"not null" json:"platform"`
	Sport                string               `gorm:"not null" json:"sport"`
	ContestType          string               `gorm:"not null" json:"contest_type"`
	Name                 string               `gorm:"not null" json:"name"`
	EntryFee             float64              `json:"entry_fee"`
	PrizePool            float64              `json:"prize_pool"`
	MaxEntries           int                  `json:"max_entries"`
	TotalEntries         int                  `json:"total_entries"`
	SalaryCap            int                  `gorm:"not null" json:"salary_cap"`
	StartTime            time.Time            `gorm:"not null" json:"start_time"`
	IsActive             bool                 `gorm:"default:true" json:"is_active"`
	IsMultiEntry         bool                 `gorm:"default:false" json:"is_multi_entry"`
	MaxLineupsPerUser    int                  `gorm:"default:1" json:"max_lineups_per_user"`
	LastDataUpdate       time.Time            `json:"last_data_update"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
	TournamentID         *string              `gorm:"type:uuid" json:"tournament_id,omitempty"`
	ExternalID           string               `gorm:"index" json:"external_id"`
	DraftGroupID         string               `gorm:"index" json:"draft_group_id"`
	LastSyncTime         time.Time            `json:"last_sync_time"`
	PositionRequirements PositionRequirements `gorm:"type:jsonb" json:"position_requirements"`
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

// GetTotalPlayers returns the total number of players required for a lineup
func (pr PositionRequirements) GetTotalPlayers() int {
	total := 0
	for _, count := range pr {
		total += count
	}
	return total
}

// LineupPlayer represents a player in an optimized lineup
type LineupPlayer struct {
	ID              uint    `json:"id"`
	Name            string  `json:"name"`
	Team            string  `json:"team"`
	Position        string  `json:"position"`
	Salary          int     `json:"salary"`
	ProjectedPoints float64 `json:"projected_points"`
}

// GeneratedLineup represents an optimized lineup
type GeneratedLineup struct {
	ID               string         `json:"id"`
	Players          []LineupPlayer `json:"players"`
	TotalSalary      int            `json:"total_salary"`
	ProjectedPoints  float64        `json:"projected_points"`
	Exposure         float64        `json:"exposure"`
	StackDescription string         `json:"stack_description,omitempty"`
}

// OptimizationRequest represents a request to optimize lineups
type OptimizationRequest struct {
	ContestID   uint                    `json:"contest_id"`
	PlayerPool  []OptimizationPlayer    `json:"player_pool"`
	Constraints OptimizationConstraints `json:"constraints"`
	Settings    OptimizationSettings    `json:"settings"`
	UserID      uint                    `json:"user_id,omitempty"`
}

// OptimizationPlayer represents a player for optimization
type OptimizationPlayer struct {
	ID              uint    `json:"id"`
	ExternalID      string  `json:"external_id"`
	Name            string  `json:"name"`
	Team            string  `json:"team"`
	Position        string  `json:"position"`
	Salary          int     `json:"salary"`
	ProjectedPoints float64 `json:"projected_points"`
	FloorPoints     float64 `json:"floor_points"`
	CeilingPoints   float64 `json:"ceiling_points"`
	Ownership       float64 `json:"ownership"`
	TeeTime         string  `json:"tee_time,omitempty"`     // Golf specific
	CutProbability  float64 `json:"cut_probability,omitempty"` // Golf specific
}

// OptimizationConstraints represents constraints for optimization
type OptimizationConstraints struct {
	SalaryCap            int                      `json:"salary_cap"`
	PositionRequirements PositionRequirements     `json:"position_requirements"`
	MaxExposure          map[string]float64       `json:"max_exposure,omitempty"`
	MinExposure          map[string]float64       `json:"min_exposure,omitempty"`
	TeamStacks           []TeamStackConstraint    `json:"team_stacks,omitempty"`
	GameStacks           []GameStackConstraint    `json:"game_stacks,omitempty"`
	MustInclude          []uint                   `json:"must_include,omitempty"`
	MustExclude          []uint                   `json:"must_exclude,omitempty"`
}

// OptimizationSettings represents settings for optimization
type OptimizationSettings struct {
	MaxLineups       int     `json:"max_lineups"`
	UniquenessFactor float64 `json:"uniqueness_factor"`
	RandomnessLevel  float64 `json:"randomness_level"`
	Timeout          int     `json:"timeout"`
}

// TeamStackConstraint represents a team stacking constraint
type TeamStackConstraint struct {
	Team      string `json:"team"`
	MinPlayers int    `json:"min_players"`
	MaxPlayers int    `json:"max_players"`
}

// GameStackConstraint represents a game stacking constraint
type GameStackConstraint struct {
	Game       string `json:"game"`        // "TeamA@TeamB"
	MinPlayers int    `json:"min_players"`
	MaxPlayers int    `json:"max_players"`
}

// OptimizationResult represents the result of an optimization
type OptimizationResult struct {
	Lineups           []GeneratedLineup         `json:"lineups"`
	Metadata          OptimizationMetadata      `json:"metadata"`
	CorrelationMatrix map[string]float64        `json:"correlation_matrix,omitempty"`
}

// OptimizationMetadata represents metadata about the optimization
type OptimizationMetadata struct {
	TotalLineups     int           `json:"total_lineups"`
	ExecutionTime    time.Duration `json:"execution_time"`
	AverageUniqueess float64       `json:"average_uniqueness"`
	TopProjection    float64       `json:"top_projection"`
	AverageProjection float64      `json:"average_projection"`
	StacksGenerated  int           `json:"stacks_generated"`
}

// ProgressUpdate represents a progress update for optimization/simulation
type ProgressUpdate struct {
	Type        string  `json:"type"`        // "optimization" or "simulation"
	Progress    float64 `json:"progress"`    // 0.0 to 1.0
	Message     string  `json:"message"`
	CurrentStep string  `json:"current_step"`
	TotalSteps  int     `json:"total_steps"`
	Timestamp   time.Time `json:"timestamp"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// Success response for API endpoints
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}