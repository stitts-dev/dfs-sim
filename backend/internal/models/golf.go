package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// TournamentStatus represents the status of a golf tournament
type TournamentStatus string

const (
	TournamentScheduled  TournamentStatus = "scheduled"
	TournamentInProgress TournamentStatus = "in_progress"
	TournamentCompleted  TournamentStatus = "completed"
	TournamentPostponed  TournamentStatus = "postponed"
	TournamentCancelled  TournamentStatus = "cancelled"
)

// PlayerEntryStatus represents a player's status in a tournament
type PlayerEntryStatus string

const (
	EntryStatusEntered   PlayerEntryStatus = "entered"
	EntryStatusWithdrawn PlayerEntryStatus = "withdrawn"
	EntryStatusCut       PlayerEntryStatus = "cut"
	EntryStatusActive    PlayerEntryStatus = "active"
	EntryStatusCompleted PlayerEntryStatus = "completed"
)

// WeatherConditions represents weather data for a golf tournament
type WeatherConditions struct {
	Temperature  int    `json:"temperature"`
	WindSpeed    int    `json:"wind_speed"`
	WindDir      string `json:"wind_direction"`
	Conditions   string `json:"conditions"`
	Humidity     int    `json:"humidity"`
}

// Value implements driver.Valuer for database storage
func (w WeatherConditions) Value() (driver.Value, error) {
	return json.Marshal(w)
}

// Scan implements sql.Scanner for database retrieval
func (w *WeatherConditions) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, w)
}

// GolfTournament represents a PGA Tour golf tournament
type GolfTournament struct {
	ID                uuid.UUID         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ExternalID        string            `gorm:"uniqueIndex;not null" json:"external_id"`
	Name              string            `gorm:"not null" json:"name"`
	StartDate         time.Time         `gorm:"not null;index" json:"start_date"`
	EndDate           time.Time         `gorm:"not null" json:"end_date"`
	Purse             float64           `json:"purse"`
	WinnerShare       float64           `json:"winner_share"`
	FedexPoints       int               `json:"fedex_points"`
	CourseID          string            `gorm:"index" json:"course_id"`
	CourseName        string            `json:"course_name"`
	CoursePar         int               `json:"course_par"`
	CourseYards       int               `json:"course_yards"`
	Status            TournamentStatus  `gorm:"type:varchar(50);default:'scheduled';index:idx_active,where:status IN ('in_progress','scheduled')" json:"status"`
	CurrentRound      int               `gorm:"default:0" json:"current_round"`
	CutLine           int               `json:"cut_line"`
	CutRule           string            `json:"cut_rule"`
	WeatherConditions WeatherConditions `gorm:"type:jsonb" json:"weather_conditions"`
	FieldStrength     float64           `json:"field_strength"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`

	// Associations
	PlayerEntries []GolfPlayerEntry `gorm:"foreignKey:TournamentID" json:"player_entries,omitempty"`
}

// TableName specifies the table name for GORM
func (GolfTournament) TableName() string {
	return "golf_tournaments"
}

// GolfPlayerEntry represents a player's entry in a golf tournament
type GolfPlayerEntry struct {
	ID               uuid.UUID         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PlayerID         uint              `gorm:"not null;uniqueIndex:idx_player_tournament,priority:1" json:"player_id"`
	Player           *Player           `gorm:"foreignKey:PlayerID" json:"player,omitempty"`
	TournamentID     uuid.UUID         `gorm:"not null;uniqueIndex:idx_player_tournament,priority:2;index:idx_tournament_status,priority:1" json:"tournament_id"`
	Tournament       *GolfTournament   `gorm:"foreignKey:TournamentID" json:"tournament,omitempty"`
	Status           PlayerEntryStatus `gorm:"type:varchar(50);default:'entered';index:idx_tournament_status,priority:2" json:"status"`
	StartingPosition int               `json:"starting_position"`
	CurrentPosition  int               `gorm:"index:idx_position,where:status = 'active'" json:"current_position"`
	TotalScore       int               `json:"total_score"`
	ThruHoles        int               `json:"thru_holes"`
	RoundsScores     pq.Int64Array     `gorm:"type:integer[]" json:"rounds_scores"`
	TeeTimes         pq.StringArray    `gorm:"type:timestamp[]" json:"tee_times"`
	PlayingPartners  pq.StringArray    `gorm:"type:uuid[]" json:"playing_partners"`
	DKSalary         int               `json:"dk_salary"`
	FDSalary         int               `json:"fd_salary"`
	DKOwnership      float64           `json:"dk_ownership"`
	FDOwnership      float64           `json:"fd_ownership"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`

	// Associations
	RoundScores []GolfRoundScore `gorm:"foreignKey:EntryID" json:"round_scores,omitempty"`
}

// TableName specifies the table name for GORM
func (GolfPlayerEntry) TableName() string {
	return "golf_player_entries"
}

// GetProjectedScore calculates expected score based on history
func (e *GolfPlayerEntry) GetProjectedScore(courseHistory *GolfCourseHistory) float64 {
	if courseHistory == nil || courseHistory.ScoringAvg == 0 {
		return 280.0 // Default 4-round score
	}

	// Adjust for recent form and course history
	baseScore := courseHistory.ScoringAvg * 4
	// This is a simplified calculation - would be enhanced with more factors
	return baseScore
}

// GolfRoundScore represents scoring data for a single round
type GolfRoundScore struct {
	ID             uuid.UUID           `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	EntryID        uuid.UUID           `gorm:"not null" json:"entry_id"`
	Entry          *GolfPlayerEntry    `gorm:"foreignKey:EntryID" json:"entry,omitempty"`
	RoundNumber    int                 `gorm:"not null;check:round_number BETWEEN 1 AND 4" json:"round_number"`
	HolesCompleted int                 `gorm:"default:0" json:"holes_completed"`
	Score          int                 `json:"score"`
	Strokes        int                 `json:"strokes"`
	Birdies        int                 `gorm:"default:0" json:"birdies"`
	Eagles         int                 `gorm:"default:0" json:"eagles"`
	Bogeys         int                 `gorm:"default:0" json:"bogeys"`
	DoubleBogeys   int                 `gorm:"default:0" json:"double_bogeys"`
	HoleScores     map[string]int      `gorm:"type:jsonb" json:"hole_scores"`
	StartedAt      *time.Time          `json:"started_at"`
	CompletedAt    *time.Time          `json:"completed_at"`
	CreatedAt      time.Time           `json:"created_at"`
}

// TableName specifies the table name for GORM
func (GolfRoundScore) TableName() string {
	return "golf_round_scores"
}

// GolfCourseHistory represents a player's historical performance at a specific course
type GolfCourseHistory struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PlayerID          uint       `gorm:"not null" json:"player_id"`
	Player            *Player    `gorm:"foreignKey:PlayerID" json:"player,omitempty"`
	CourseID          string     `gorm:"not null" json:"course_id"`
	TournamentsPlayed int        `gorm:"default:0" json:"tournaments_played"`
	RoundsPlayed      int        `gorm:"default:0" json:"rounds_played"`
	TotalStrokes      int        `json:"total_strokes"`
	ScoringAvg        float64    `json:"scoring_avg"`
	AdjScoringAvg     float64    `json:"adj_scoring_avg"`
	BestFinish        int        `json:"best_finish"`
	WorstFinish       int        `json:"worst_finish"`
	CutsMade          int        `json:"cuts_made"`
	MissedCuts        int        `json:"missed_cuts"`
	Top10s            int        `json:"top_10s"`
	Top25s            int        `json:"top_25s"`
	Wins              int        `json:"wins"`
	StrokesGainedTotal float64   `json:"strokes_gained_total"`
	SGTeeToGreen      float64    `json:"sg_tee_to_green"`
	SGPutting         float64    `json:"sg_putting"`
	LastPlayed        *time.Time `json:"last_played"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (GolfCourseHistory) TableName() string {
	return "golf_course_history"
}

// GolfProjection represents projected performance for a golf player
type GolfProjection struct {
	PlayerID         string  `json:"player_id"`
	TournamentID     string  `json:"tournament_id"`
	ExpectedScore    float64 `json:"expected_score"`
	CutProbability   float64 `json:"cut_probability"`
	Top10Probability float64 `json:"top10_probability"`
	Top25Probability float64 `json:"top25_probability"`
	WinProbability   float64 `json:"win_probability"`
	DKPoints         float64 `json:"dk_points"`
	FDPoints         float64 `json:"fd_points"`
	Confidence       float64 `json:"confidence"`
}

// HoleScore represents the score for a single hole
type HoleScore struct {
	Hole   int `json:"hole"`
	Par    int `json:"par"`
	Score  int `json:"score"`
	Yards  int `json:"yards"`
}