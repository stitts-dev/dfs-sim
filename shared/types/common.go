package types

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
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

// CacheProvider defines the interface for caching services
type CacheProvider interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) bool
	// Convenience methods for backward compatibility
	SetSimple(key string, value interface{}, expiration time.Duration) error
	GetSimple(key string, dest interface{}) error
}

// Sport represents different sports types
type Sport string

const (
	SportGolf       Sport = "golf"
	SportNBA        Sport = "nba"
	SportNFL        Sport = "nfl"
	SportMLB        Sport = "mlb"
	SportNHL        Sport = "nhl"
	SportSoccer     Sport = "soccer"
	SportTennis     Sport = "tennis"
)

// PlayerData represents player data from external providers
type PlayerData struct {
	ID              string            `json:"id"`
	ExternalID      string            `json:"external_id"`
	Name            string            `json:"name"`
	Team            string            `json:"team"`
	Position        string            `json:"position"`
	Salary          int               `json:"salary"`
	ProjectedPoints float64           `json:"projected_points"`
	FloorPoints     float64           `json:"floor_points"`
	CeilingPoints   float64           `json:"ceiling_points"`
	Ownership       float64           `json:"ownership"`
	Sport           Sport             `json:"sport"`
	GameTime        time.Time         `json:"game_time"`
	IsInjured       bool              `json:"is_injured"`
	InjuryStatus    string            `json:"injury_status"`
	ImageURL        string            `json:"image_url"`
	Metadata        map[string]string `json:"metadata"`
	Stats           interface{}       `json:"stats,omitempty"`
	LastUpdated     time.Time         `json:"last_updated"`
	Source          string            `json:"source"`
}

// PlayerInterface defines the common interface for all player types
type PlayerInterface interface {
	GetID() uuid.UUID
	GetExternalID() string
	GetName() string
	GetTeam() string
	GetPosition() string
	GetSalaryDK() int
	GetSalaryFD() int
	GetProjectedPoints() float64
	GetFloorPoints() float64
	GetCeilingPoints() float64
	GetOwnershipDK() float64
	GetOwnershipFD() float64
	GetGameTime() time.Time
	IsPlayerInjured() bool
	GetInjuryStatus() string
	GetImageURL() string
}

// Player represents a DFS player (shared across all services)
type Player struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SportID         uuid.UUID  `gorm:"type:uuid;not null" json:"sport_id"`
	ExternalID      string     `gorm:"not null" json:"external_id"`
	Name            string     `gorm:"not null" json:"name"`
	Team            string     `gorm:"not null" json:"team"`
	Opponent        string     `gorm:"not null" json:"opponent"`
	Position        string     `gorm:"not null" json:"position"`
	SalaryDK        int        `json:"salary_dk"`
	SalaryFD        int        `json:"salary_fd"`
	ProjectedPoints float64    `gorm:"not null" json:"projected_points"`
	FloorPoints     float64    `gorm:"not null" json:"floor_points"`
	CeilingPoints   float64    `gorm:"not null" json:"ceiling_points"`
	OwnershipDK     float64    `json:"ownership_dk"`
	OwnershipFD     float64    `json:"ownership_fd"`
	ContestID       *uuid.UUID `gorm:"type:uuid;index" json:"contest_id,omitempty"`
	GameTime        time.Time  `gorm:"not null" json:"game_time"`
	IsInjured       bool       `gorm:"default:false" json:"is_injured"`
	InjuryStatus    string     `json:"injury_status,omitempty"`
	ImageURL        string     `json:"image_url,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Implement PlayerInterface for types.Player
func (p Player) GetID() uuid.UUID              { return p.ID }
func (p Player) GetExternalID() string         { return p.ExternalID }
func (p Player) GetName() string               { return p.Name }
func (p Player) GetTeam() string               { return p.Team }
func (p Player) GetPosition() string           { return p.Position }
func (p Player) GetSalaryDK() int              { return p.SalaryDK }
func (p Player) GetSalaryFD() int              { return p.SalaryFD }
func (p Player) GetProjectedPoints() float64   { return p.ProjectedPoints }
func (p Player) GetFloorPoints() float64       { return p.FloorPoints }
func (p Player) GetCeilingPoints() float64     { return p.CeilingPoints }
func (p Player) GetOwnershipDK() float64       { return p.OwnershipDK }
func (p Player) GetOwnershipFD() float64       { return p.OwnershipFD }
func (p Player) GetGameTime() time.Time        { return p.GameTime }
func (p Player) IsPlayerInjured() bool         { return p.IsInjured }
func (p Player) GetInjuryStatus() string       { return p.InjuryStatus }
func (p Player) GetImageURL() string           { return p.ImageURL }

// SimulationResult represents the result of a Monte Carlo simulation
type SimulationResult struct {
	ID               string                     `json:"id"`
	Iterations       int                        `json:"iterations"`
	ExecutionTime    time.Duration              `json:"execution_time"`
	LineupResults    []LineupSimulationResult   `json:"lineup_results"`
	OverallStats     SimulationStats            `json:"overall_stats"`
	ContestType      string                     `json:"contest_type"`
	CreatedAt        time.Time                  `json:"created_at"`
}

// LineupSimulationResult represents simulation results for a single lineup
type LineupSimulationResult struct {
	LineupID         string  `json:"lineup_id"`
	ExpectedScore    float64 `json:"expected_score"`
	ScoreVariance    float64 `json:"score_variance"`
	CashRate         float64 `json:"cash_rate"`
	ROI              float64 `json:"roi"`
	Top1Percent      float64 `json:"top_1_percent"`
	Top10Percent     float64 `json:"top_10_percent"`
	MedianFinish     int     `json:"median_finish"`
	Ceiling          float64 `json:"ceiling"`
	Floor            float64 `json:"floor"`
}

// SimulationStats represents overall statistics for a simulation
type SimulationStats struct {
	TotalLineups     int     `json:"total_lineups"`
	AverageROI       float64 `json:"average_roi"`
	BestROI          float64 `json:"best_roi"`
	WorstROI         float64 `json:"worst_roi"`
	AverageCashRate  float64 `json:"average_cash_rate"`
	PortfolioROI     float64 `json:"portfolio_roi"`
	Sharpe           float64 `json:"sharpe"`
}

// Contest represents a DFS contest (shared across all services)
type Contest struct {
	ID                   uuid.UUID            `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SportID              uuid.UUID            `gorm:"type:uuid;not null" json:"sport_id"`
	Platform             string               `gorm:"not null" json:"platform"`
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
	LastDataUpdate       *time.Time           `json:"last_data_update,omitempty"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
	TournamentID         *uuid.UUID           `gorm:"type:uuid" json:"tournament_id,omitempty"`
	ExternalID           string               `gorm:"index" json:"external_id"`
	DraftGroupID         string               `gorm:"index" json:"draft_group_id"`
	LastSyncTime         *time.Time           `json:"last_sync_time,omitempty"`
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

// GetPositionRequirements returns position requirements for a given sport and platform
func GetPositionRequirements(sport, platform string) PositionRequirements {
	switch sport {
	case "golf":
		// Golf typically uses 6 golfers for DraftKings, FanDuel, etc.
		return PositionRequirements{
			"G": 6, // 6 golfers
		}
	case "nba":
		if platform == "draftkings" {
			return PositionRequirements{
				"PG": 1, "SG": 1, "SF": 1, "PF": 1, "C": 1,
				"G": 1, "F": 1, "UTIL": 1,
			}
		}
		// FanDuel NBA
		return PositionRequirements{
			"PG": 2, "SG": 2, "SF": 2, "PF": 2, "C": 1,
		}
	case "nfl":
		if platform == "draftkings" {
			return PositionRequirements{
				"QB": 1, "RB": 2, "WR": 3, "TE": 1, "K": 1, "DST": 1,
			}
		}
		// FanDuel NFL
		return PositionRequirements{
			"QB": 1, "RB": 2, "WR": 3, "TE": 1, "K": 1, "DST": 1,
		}
	default:
		return PositionRequirements{}
	}
}

// LineupPlayer represents a player in an optimized lineup
type LineupPlayer struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Team            string    `json:"team"`
	Position        string    `json:"position"`
	Salary          int       `json:"salary"`
	ProjectedPoints float64   `json:"projected_points"`
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
	ContestID   uuid.UUID               `json:"contest_id"`
	PlayerPool  []OptimizationPlayer    `json:"player_pool"`
	Constraints OptimizationConstraints `json:"constraints"`
	Settings    OptimizationSettings    `json:"settings"`
	UserID      uuid.UUID               `json:"user_id,omitempty"`
}

// OptimizationPlayer represents a player for optimization
type OptimizationPlayer struct {
	ID              uuid.UUID `json:"id"`
	ExternalID      string    `json:"external_id"`
	Name            string    `json:"name"`
	Team            string    `json:"team"`
	Position        string    `json:"position"`
	Salary          int       `json:"salary"`
	ProjectedPoints float64   `json:"projected_points"`
	FloorPoints     float64   `json:"floor_points"`
	CeilingPoints   float64   `json:"ceiling_points"`
	Ownership       float64   `json:"ownership"`
	TeeTime         string    `json:"tee_time,omitempty"`     // Golf specific
	CutProbability  float64   `json:"cut_probability,omitempty"` // Golf specific
}

// OptimizationConstraints represents constraints for optimization
type OptimizationConstraints struct {
	SalaryCap            int                      `json:"salary_cap"`
	PositionRequirements PositionRequirements     `json:"position_requirements"`
	MaxExposure          map[string]float64       `json:"max_exposure,omitempty"`
	MinExposure          map[string]float64       `json:"min_exposure,omitempty"`
	TeamStacks           []TeamStackConstraint    `json:"team_stacks,omitempty"`
	GameStacks           []GameStackConstraint    `json:"game_stacks,omitempty"`
	MustInclude          []uuid.UUID              `json:"must_include,omitempty"`
	MustExclude          []uuid.UUID              `json:"must_exclude,omitempty"`
}

// OptimizationSettings represents settings for optimization
type OptimizationSettings struct {
	MaxLineups          int                  `json:"max_lineups"`
	MinDifferentPlayers int                  `json:"min_different_players"`
	UseCorrelations     bool                 `json:"use_correlations"`
	CorrelationWeight   float64             `json:"correlation_weight"`
	StackingRules       []StackingRule      `json:"stacking_rules"`
	LockedPlayers       []uuid.UUID         `json:"locked_players"`
	ExcludedPlayers     []uuid.UUID         `json:"excluded_players"`
	MinExposure         map[uuid.UUID]float64 `json:"min_exposure"`
	MaxExposure         map[uuid.UUID]float64 `json:"max_exposure"`
	UniquenessFactor    float64             `json:"uniqueness_factor"`
	RandomnessLevel     float64             `json:"randomness_level"`
	Timeout             int                 `json:"timeout"`
}

// StackingRule represents a stacking rule for optimization
type StackingRule struct {
	Type       string   `json:"type"` // "team", "game", "mini"
	MinPlayers int      `json:"min_players"`
	MaxPlayers int      `json:"max_players"`
	Teams      []string `json:"teams,omitempty"`
}

// OptimizationMeta represents metadata about an optimization
type OptimizationMeta struct {
	ExecutionTime     time.Duration          `json:"execution_time"`
	TotalCombinations int64                  `json:"total_combinations"`
	ValidCombinations int64                  `json:"valid_combinations"`
	SettingsUsed      map[string]interface{} `json:"settings_used"`
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

// User represents a user in the system
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	PhoneNumber  string    `gorm:"uniqueIndex;not null" json:"phone_number"`
	Email        string    `json:"email,omitempty"`
	FirstName    string    `json:"first_name,omitempty"`
	LastName     string    `json:"last_name,omitempty"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserPreferences represents user preferences for optimization
type UserPreferences struct {
	ID                   uuid.UUID              `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID               uuid.UUID              `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	BeginnerMode         bool                   `gorm:"default:true" json:"beginner_mode"`
	DefaultSport         string                 `gorm:"default:'nfl'" json:"default_sport"`
	DefaultContestType   string                 `gorm:"default:'cash'" json:"default_contest_type"`
	MaxExposure          float64                `gorm:"default:20.0" json:"max_exposure"`
	MinStackSize         int                    `gorm:"default:2" json:"min_stack_size"`
	MaxStackSize         int                    `gorm:"default:4" json:"max_stack_size"`
	AutoOptimize         bool                   `gorm:"default:false" json:"auto_optimize"`
	NotificationSettings map[string]interface{} `gorm:"type:jsonb" json:"notification_settings"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// Lineup represents a saved lineup
type Lineup struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	Name            string         `gorm:"not null" json:"name"`
	Sport           string         `gorm:"not null" json:"sport"`
	Platform        string         `gorm:"not null" json:"platform"`
	ContestID       *uuid.UUID     `gorm:"type:uuid" json:"contest_id,omitempty"`
	Players         []LineupPlayer `gorm:"-" json:"players"` // Use LineupPlayer slice instead of map
	TotalSalary     int            `json:"total_salary"`
	ProjectedPoints float64        `json:"projected_points"`
	ActualPoints    *float64       `json:"actual_points,omitempty"`
	IsLocked        bool           `gorm:"default:false" json:"is_locked"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}