package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// Sport represents different sports types
type Sport struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name         string    `gorm:"uniqueIndex;not null" json:"name"`
	Abbreviation string    `gorm:"uniqueIndex;not null" json:"abbreviation"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	SeasonStart  *time.Time `json:"season_start,omitempty"`
	SeasonEnd    *time.Time `json:"season_end,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (Sport) TableName() string {
	return "sports"
}

// PositionRequirements defines how many players needed for each position
type PositionRequirements map[string]int

// Contest represents a DFS contest
type Contest struct {
	ID                   uuid.UUID             `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SportID              uuid.UUID             `gorm:"type:uuid;not null" json:"sport_id"`
	Sport                *Sport                `gorm:"foreignKey:SportID" json:"sport,omitempty"`
	Platform             string                `gorm:"not null" json:"platform"`
	ContestType          string                `gorm:"not null" json:"contest_type"`
	Name                 string                `gorm:"not null" json:"name"`
	EntryFee             float64               `json:"entry_fee"`
	PrizePool            float64               `json:"prize_pool"`
	MaxEntries           int                   `json:"max_entries"`
	TotalEntries         int                   `json:"total_entries"`
	SalaryCap            int                   `gorm:"not null" json:"salary_cap"`
	StartTime            time.Time             `gorm:"not null" json:"start_time"`
	IsActive             bool                  `gorm:"default:true" json:"is_active"`
	IsMultiEntry         bool                  `gorm:"default:false" json:"is_multi_entry"`
	MaxLineupsPerUser    int                   `gorm:"default:1" json:"max_lineups_per_user"`
	LastDataUpdate       *time.Time            `json:"last_data_update,omitempty"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`
	TournamentID         *uuid.UUID            `gorm:"type:uuid" json:"tournament_id,omitempty"`
	ExternalID           string                `gorm:"index" json:"external_id"`
	DraftGroupID         string                `gorm:"index" json:"draft_group_id"`
	LastSyncTime         *time.Time            `json:"last_sync_time,omitempty"`
	PositionRequirements PositionRequirements  `gorm:"column:roster_positions;type:jsonb" json:"roster_positions"`
	
	// Associations
	Players []Player `gorm:"foreignKey:ContestID" json:"players,omitempty"`
}

// TableName specifies the table name for GORM
func (Contest) TableName() string {
	return "contests"
}

// Player represents a DFS player
type Player struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SportID         uuid.UUID `gorm:"type:uuid;not null" json:"sport_id"`
	Sport           *Sport    `gorm:"foreignKey:SportID" json:"sport,omitempty"`
	ExternalID      string    `gorm:"not null" json:"external_id"`
	Name            string    `gorm:"not null" json:"name"`
	Team            string    `gorm:"not null" json:"team"`
	Opponent        string    `gorm:"not null" json:"opponent"`
	Position        string    `gorm:"not null" json:"position"`
	SalaryDK        int       `json:"salary_dk"`
	SalaryFD        int       `json:"salary_fd"`
	ProjectedPoints float64   `gorm:"not null" json:"projected_points"`
	FloorPoints     float64   `gorm:"not null" json:"floor_points"`
	CeilingPoints   float64   `gorm:"not null" json:"ceiling_points"`
	OwnershipDK     float64   `json:"ownership_dk"`
	OwnershipFD     float64   `json:"ownership_fd"`
	ContestID       *uuid.UUID `gorm:"type:uuid;index" json:"contest_id,omitempty"`
	Contest         *Contest  `gorm:"foreignKey:ContestID" json:"contest,omitempty"`
	GameTime        time.Time `gorm:"not null" json:"game_time"`
	IsInjured       bool      `gorm:"default:false" json:"is_injured"`
	InjuryStatus    string    `json:"injury_status,omitempty"`
	ImageURL        string    `json:"image_url,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (Player) TableName() string {
	return "players"
}

// Verify that models.Player implements types.PlayerInterface at compile time
var _ types.PlayerInterface = (*Player)(nil)

// Implement types.PlayerInterface for models.Player
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

// Lineup represents a saved lineup
type Lineup struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	ContestID       *uuid.UUID `gorm:"type:uuid;index" json:"contest_id,omitempty"`
	Contest         *Contest  `gorm:"foreignKey:ContestID" json:"contest,omitempty"`
	Name            string    `json:"name"`
	Sport           string    `gorm:"not null" json:"sport"`
	Platform        string    `gorm:"not null" json:"platform"`
	TotalSalary     int       `gorm:"not null" json:"total_salary"`
	ProjectedPoints float64   `gorm:"not null" json:"projected_points"`
	ActualPoints    *float64  `json:"actual_points,omitempty"`
	SimulatedCeiling *float64 `json:"simulated_ceiling,omitempty"`
	SimulatedFloor  *float64  `json:"simulated_floor,omitempty"`
	SimulatedMean   *float64  `json:"simulated_mean,omitempty"`
	Ownership       *float64  `json:"ownership,omitempty"`
	IsSubmitted     bool      `gorm:"default:false" json:"is_submitted"`
	IsOptimized     bool      `gorm:"default:false" json:"is_optimized"`
	OptimizationRank *int     `json:"optimization_rank,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	
	// Associations
	LineupPlayers []LineupPlayer `gorm:"foreignKey:LineupID" json:"lineup_players,omitempty"`
}

// TableName specifies the table name for GORM
func (Lineup) TableName() string {
	return "lineups"
}

// LineupPlayer represents a player in a lineup
type LineupPlayer struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	LineupID        *uuid.UUID `gorm:"type:uuid" json:"lineup_id,omitempty"`
	PlayerID        uuid.UUID `gorm:"type:uuid;not null" json:"player_id"`
	Position        string    `gorm:"not null" json:"position"`
	Salary          int       `gorm:"not null" json:"salary"`
	ProjectedPoints float64   `json:"projected_points"`
	ActualPoints    *float64  `json:"actual_points,omitempty"`
	Lineup          *Lineup   `gorm:"foreignKey:LineupID" json:"lineup,omitempty"`
	Player          *Player   `gorm:"foreignKey:PlayerID" json:"player,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// TableName specifies the table name for GORM
func (LineupPlayer) TableName() string {
	return "lineup_players"
}

// SimulationResult represents the result of a Monte Carlo simulation
type SimulationResult struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID            uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	LineupID          *uuid.UUID     `gorm:"type:uuid" json:"lineup_id,omitempty"`
	Lineup            *Lineup        `gorm:"foreignKey:LineupID" json:"lineup,omitempty"`
	ContestID         *uuid.UUID     `gorm:"type:uuid;index" json:"contest_id,omitempty"`
	Contest           *Contest       `gorm:"foreignKey:ContestID" json:"contest,omitempty"`
	SimulationsRun    int            `gorm:"not null" json:"simulations_run"`
	ProjectedROI      float64        `json:"projected_roi"`
	WinProbability    float64        `json:"win_probability"`
	Top1Percent       float64        `json:"top_1_percent"`
	Top10Percent      float64        `json:"top_10_percent"`
	AvgScore          float64        `json:"avg_score"`
	MedianScore       float64        `json:"median_score"`
	StdDeviation      float64        `json:"std_deviation"`
	ExecutionTimeMs   int            `json:"execution_time_ms"`
	CreatedAt         time.Time      `json:"created_at"`
}

// TableName specifies the table name for GORM
func (SimulationResult) TableName() string {
	return "simulation_results"
}

// User represents a user in the system (extends auth.users)
type User struct {
	ID                        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	PhoneNumber               string    `gorm:"uniqueIndex;not null" json:"phone_number"`
	PhoneVerified             bool      `gorm:"default:false" json:"phone_verified"`
	Email                     string    `json:"email,omitempty"`
	EmailVerified             bool      `gorm:"default:false" json:"email_verified"`
	FirstName                 string    `json:"first_name,omitempty"`
	LastName                  string    `json:"last_name,omitempty"`
	SubscriptionTier          string    `gorm:"default:'free'" json:"subscription_tier"`
	SubscriptionStatus        string    `gorm:"default:'active'" json:"subscription_status"`
	SubscriptionExpiresAt     *time.Time `json:"subscription_expires_at,omitempty"`
	StripeCustomerID          string    `json:"stripe_customer_id,omitempty"`
	MonthlyOptimizationsUsed  int       `gorm:"default:0" json:"monthly_optimizations_used"`
	MonthlySimulationsUsed    int       `gorm:"default:0" json:"monthly_simulations_used"`
	UsageResetDate            time.Time `gorm:"default:CURRENT_DATE" json:"usage_reset_date"`
	IsActive                  bool      `gorm:"default:true" json:"is_active"`
	LastLoginAt               *time.Time `json:"last_login_at,omitempty"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (User) TableName() string {
	return "users"
}

// UserPreferences represents user preferences for optimization
type UserPreferences struct {
	ID                      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID                  uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	User                    *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	SportPreferences        pq.StringArray `gorm:"type:jsonb;default:'[\"nba\", \"nfl\", \"mlb\", \"golf\"]'" json:"sport_preferences"`
	PlatformPreferences     pq.StringArray `gorm:"type:jsonb;default:'[\"draftkings\", \"fanduel\"]'" json:"platform_preferences"`
	ContestTypePreferences  pq.StringArray `gorm:"type:jsonb;default:'[\"gpp\", \"cash\"]'" json:"contest_type_preferences"`
	Theme                   string    `gorm:"default:'light'" json:"theme"`
	Language                string    `gorm:"default:'en'" json:"language"`
	NotificationsEnabled    bool      `gorm:"default:true" json:"notifications_enabled"`
	TutorialCompleted       bool      `gorm:"default:false" json:"tutorial_completed"`
	BeginnerMode            bool      `gorm:"default:true" json:"beginner_mode"`
	TooltipsEnabled         bool      `gorm:"default:true" json:"tooltips_enabled"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (UserPreferences) TableName() string {
	return "user_preferences"
}

// SubscriptionTier represents a subscription tier configuration
type SubscriptionTier struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	Name                  string    `gorm:"uniqueIndex;not null" json:"name"`
	PriceCents            int       `gorm:"not null;default:0" json:"price_cents"`
	Currency              string    `gorm:"default:'USD'" json:"currency"`
	MonthlyOptimizations  int       `gorm:"default:10" json:"monthly_optimizations"`
	MonthlySimulations    int       `gorm:"default:5" json:"monthly_simulations"`
	AIRecommendations     bool      `gorm:"default:false" json:"ai_recommendations"`
	BankVerification      bool      `gorm:"default:false" json:"bank_verification"`
	PrioritySupport       bool      `gorm:"default:false" json:"priority_support"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (SubscriptionTier) TableName() string {
	return "subscription_tiers"
}

// PhoneVerificationCode represents a phone verification code for OTP
type PhoneVerificationCode struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PhoneNumber string    `gorm:"not null;index" json:"phone_number"`
	Code        string    `gorm:"not null" json:"code"`
	ExpiresAt   time.Time `gorm:"not null;index" json:"expires_at"`
	Attempts    int       `gorm:"default:0" json:"attempts"`
	Verified    bool      `gorm:"default:false" json:"verified"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableName specifies the table name for GORM
func (PhoneVerificationCode) TableName() string {
	return "phone_verification_codes"
}