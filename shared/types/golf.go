package types

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
	Temperature int    `json:"temperature"`
	WindSpeed   int    `json:"wind_speed"`
	WindDir     string `json:"wind_direction"`
	Conditions  string `json:"conditions"`
	Humidity    int    `json:"humidity"`
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
}

// GolfPlayerEntry represents a player's entry in a golf tournament
type GolfPlayerEntry struct {
	ID               uuid.UUID         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PlayerID         uint              `gorm:"not null;uniqueIndex:idx_player_tournament,priority:1" json:"player_id"`
	TournamentID     uuid.UUID         `gorm:"not null;uniqueIndex:idx_player_tournament,priority:2;index:idx_tournament_status,priority:1" json:"tournament_id"`
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
}

// GolfRoundScore represents scoring data for a single round
type GolfRoundScore struct {
	ID             uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	EntryID        uuid.UUID   `gorm:"not null" json:"entry_id"`
	RoundNumber    int         `gorm:"not null;check:round_number BETWEEN 1 AND 4" json:"round_number"`
	HolesCompleted int         `gorm:"default:0" json:"holes_completed"`
	Score          int         `json:"score"`
	Strokes        int         `json:"strokes"`
	Birdies        int         `gorm:"default:0" json:"birdies"`
	Eagles         int         `gorm:"default:0" json:"eagles"`
	Bogeys         int         `gorm:"default:0" json:"bogeys"`
	DoubleBogeys   int         `gorm:"default:0" json:"double_bogeys"`
	HoleScores     map[string]int `gorm:"type:jsonb" json:"hole_scores"`
	StartedAt      *time.Time  `json:"started_at"`
	CompletedAt    *time.Time  `json:"completed_at"`
	CreatedAt      time.Time   `json:"created_at"`
}

// GolfCourseHistory represents a player's historical performance at a specific course
type GolfCourseHistory struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PlayerID           uint       `gorm:"not null" json:"player_id"`
	CourseID           string     `gorm:"not null" json:"course_id"`
	TournamentsPlayed  int        `gorm:"default:0" json:"tournaments_played"`
	RoundsPlayed       int        `gorm:"default:0" json:"rounds_played"`
	TotalStrokes       int        `json:"total_strokes"`
	ScoringAvg         float64    `json:"scoring_avg"`
	AdjScoringAvg      float64    `json:"adj_scoring_avg"`
	BestFinish         int        `json:"best_finish"`
	WorstFinish        int        `json:"worst_finish"`
	CutsMade           int        `json:"cuts_made"`
	MissedCuts         int        `json:"missed_cuts"`
	Top10s             int        `json:"top_10s"`
	Top25s             int        `json:"top_25s"`
	Wins               int        `json:"wins"`
	StrokesGainedTotal float64    `json:"strokes_gained_total"`
	SGTeeToGreen       float64    `json:"sg_tee_to_green"`
	SGPutting          float64    `json:"sg_putting"`
	LastPlayed         *time.Time `json:"last_played"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// GolfProjection represents projected performance for a golf player
type GolfProjection struct {
	PlayerID         string  `json:"player_id"`
	TournamentID     string  `json:"tournament_id"`
	ExpectedScore    float64 `json:"expected_score"`
	DKPoints         float64 `json:"dk_points"`
	FDPoints         float64 `json:"fd_points"`
	Confidence       float64 `json:"confidence"`

	// Cut probability modeling
	BaseCutProbability      float64 `json:"base_cut_probability"`
	CourseCutProbability    float64 `json:"course_cut_probability"`
	WeatherAdjustedCut      float64 `json:"weather_adjusted_cut"`
	FinalCutProbability     float64 `json:"final_cut_probability"`
	CutConfidence          float64 `json:"cut_confidence"`
	
	// Position probabilities
	Top5Probability        float64 `json:"top5_probability"`
	Top10Probability       float64 `json:"top10_probability"`
	Top25Probability       float64 `json:"top25_probability"`
	WinProbability         float64 `json:"win_probability"`
	ExpectedFinishPosition float64 `json:"expected_finish_position"`
	
	// Weather impact
	WeatherAdvantage       float64 `json:"weather_advantage"`
	TeeTimeAdvantage      float64 `json:"tee_time_advantage"`
	WeatherImpactScore    float64 `json:"weather_impact_score"`
	
	// Strategy-specific scores
	StrategyFitScore      float64 `json:"strategy_fit_score"`
	RiskRewardRatio       float64 `json:"risk_reward_ratio"`
	VarianceScore         float64 `json:"variance_score"`
}

// HoleScore represents the score for a single hole
type HoleScore struct {
	Hole  int `json:"hole"`
	Par   int `json:"par"`
	Score int `json:"score"`
	Yards int `json:"yards"`
}

// GolfTournamentSyncRequest represents a request to sync tournament data
type GolfTournamentSyncRequest struct {
	TournamentID string `json:"tournament_id"`
	ForceRefresh bool   `json:"force_refresh"`
}

// GolfLeaderboardResponse represents leaderboard data
type GolfLeaderboardResponse struct {
	TournamentID   string                     `json:"tournament_id"`
	TournamentName string                     `json:"tournament_name"`
	CurrentRound   int                        `json:"current_round"`
	CutLine        int                        `json:"cut_line"`
	LastUpdated    time.Time                  `json:"last_updated"`
	Leaderboard    []GolfLeaderboardEntry     `json:"leaderboard"`
}

// GolfLeaderboardEntry represents a single leaderboard entry
type GolfLeaderboardEntry struct {
	Position    int    `json:"position"`
	PlayerName  string `json:"player_name"`
	PlayerID    string `json:"player_id"`
	TotalScore  int    `json:"total_score"`
	ThruHoles   int    `json:"thru_holes"`
	RoundScore  int    `json:"round_score"`
	TeeTime     string `json:"tee_time"`
	Status      string `json:"status"`
}

// GolfPlayerProjectionRequest represents a request for player projections
type GolfPlayerProjectionRequest struct {
	TournamentID   string   `json:"tournament_id"`
	PlayerIDs      []string `json:"player_ids,omitempty"`
	Platform       string   `json:"platform"` // "draftkings" or "fanduel"
	UseWeather     bool     `json:"use_weather"`
	UseCourseHistory bool   `json:"use_course_history"`
}

// GolfPlayerProjectionResponse represents player projection data
type GolfPlayerProjectionResponse struct {
	TournamentID string            `json:"tournament_id"`
	Platform     string            `json:"platform"`
	GeneratedAt  time.Time         `json:"generated_at"`
	Projections  []GolfProjection  `json:"projections"`
}

// Enhanced Data Models for DataGolf Integration

// StrokesGainedMetrics represents comprehensive strokes gained data
type StrokesGainedMetrics struct {
	PlayerID           int64     `json:"player_id"`
	TournamentID       string    `json:"tournament_id"`
	SGOffTheTee       float64   `json:"sg_off_the_tee"`
	SGApproach        float64   `json:"sg_approach"`
	SGAroundTheGreen  float64   `json:"sg_around_the_green"`
	SGPutting         float64   `json:"sg_putting"`
	SGTotal           float64   `json:"sg_total"`
	Consistency       float64   `json:"consistency_rating"`
	VolatilityIndex   float64   `json:"volatility_index"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CourseAnalytics represents comprehensive course data for optimization
type CourseAnalytics struct {
	CourseID              string                 `json:"course_id"`
	DifficultyRating      float64               `json:"difficulty_rating"`
	Length                int                   `json:"length"`
	Par                   int                   `json:"par"`
	PlayerTypeAdvantages  map[string]float64    `json:"player_type_advantages"`
	WeatherSensitivity    map[string]float64    `json:"weather_sensitivity"`
	HistoricalScoring     ScoreDistribution     `json:"historical_scoring"`
	KeyHoles              []int                 `json:"key_holes"`
	SkillPremiums         SkillPremiumWeights   `json:"skill_premiums"`
}

// SkillPremiumWeights represents the importance of different skills for a course
type SkillPremiumWeights struct {
	DrivingDistance    float64 `json:"driving_distance"`
	DrivingAccuracy    float64 `json:"driving_accuracy"`
	ApproachPrecision  float64 `json:"approach_precision"`
	ShortGameSkill     float64 `json:"short_game_skill"`
	PuttingConsistency float64 `json:"putting_consistency"`
}

// ScoreDistribution represents historical scoring patterns
type ScoreDistribution struct {
	MeanScore      float64 `json:"mean_score"`
	MedianScore    float64 `json:"median_score"`
	StandardDev    float64 `json:"standard_deviation"`
	WinningScore   float64 `json:"winning_score"`
	CutScore       float64 `json:"cut_score"`
}

// TournamentPredictions represents enhanced pre-tournament predictions
type TournamentPredictions struct {
	TournamentID  string                      `json:"tournament_id"`
	GeneratedAt   time.Time                   `json:"generated_at"`
	Predictions   []EnhancedPlayerPrediction  `json:"predictions"`
	CourseModel   CourseModelData             `json:"course_model"`
	WeatherModel  WeatherModelData            `json:"weather_model"`
}

// EnhancedPlayerPrediction represents comprehensive player prediction data
type EnhancedPlayerPrediction struct {
	PlayerID              int     `json:"player_id"`
	PlayerName            string  `json:"player_name"`
	WinProbability        float64 `json:"win_probability"`
	Top5Probability       float64 `json:"top5_probability"`
	Top10Probability      float64 `json:"top10_probability"`
	Top20Probability      float64 `json:"top20_probability"`
	MakeCutProbability    float64 `json:"make_cut_probability"`
	ProjectedScore        float64 `json:"projected_score"`
	ProjectedFinish       float64 `json:"projected_finish"`
	CourseFit             float64 `json:"course_fit"`
	WeatherAdvantage      float64 `json:"weather_advantage"`
	VolatilityRating      float64 `json:"volatility_rating"`
	StrategyFit           map[string]float64 `json:"strategy_fit"`
}

// CourseModelData represents course modeling information
type CourseModelData struct {
	ModelVersion     string             `json:"model_version"`
	Accuracy         float64            `json:"accuracy"`
	KeyFactors       []string           `json:"key_factors"`
	PlayerTypeBonus  map[string]float64 `json:"player_type_bonus"`
}

// WeatherModelData represents weather modeling for tournament prediction
type WeatherModelData struct {
	CurrentConditions WeatherConditions  `json:"current_conditions"`
	Forecast          []WeatherForecast  `json:"forecast"`
	ImpactScore       float64            `json:"impact_score"`
}

// WeatherForecast represents weather forecast data
type WeatherForecast struct {
	Date        time.Time `json:"date"`
	Temperature int       `json:"temperature"`
	WindSpeed   int       `json:"wind_speed"`
	WindDir     string    `json:"wind_direction"`
	Conditions  string    `json:"conditions"`
	Humidity    int       `json:"humidity"`
}

// LiveTournamentData represents real-time tournament information
type LiveTournamentData struct {
	TournamentID     string                 `json:"tournament_id"`
	CurrentRound     int                    `json:"current_round"`
	CutLine          int                    `json:"cut_line"`
	CutMade          bool                   `json:"cut_made"`
	LeaderScore      int                    `json:"leader_score"`
	LastUpdated      time.Time              `json:"last_updated"`
	LiveLeaderboard  []LiveLeaderboardEntry `json:"live_leaderboard"`
	WeatherUpdate    WeatherConditions      `json:"weather_update"`
	PlaySuspended    bool                   `json:"play_suspended"`
}

// LiveLeaderboardEntry represents a live leaderboard entry
type LiveLeaderboardEntry struct {
	PlayerID         int     `json:"player_id"`
	PlayerName       string  `json:"player_name"`
	Position         int     `json:"position"`
	TotalScore       int     `json:"total_score"`
	ThruHoles        int     `json:"thru_holes"`
	RoundScore       int     `json:"round_score"`
	MovementIndicator string `json:"movement_indicator"`
	TeeTime          string  `json:"tee_time"`
	IsOnCourse       bool    `json:"is_on_course"`
}

// PlayerCourseHistory represents historical performance at a specific course
type PlayerCourseHistory struct {
	PlayerID           int                    `json:"player_id"`
	CourseID           string                 `json:"course_id"`
	TotalAppearances   int                    `json:"total_appearances"`
	AveragingScore     float64                `json:"averaging_score"`
	BestFinish         int                    `json:"best_finish"`
	RecentForm         []CourseHistoryEntry   `json:"recent_form"`
	StrokesGainedAvg   StrokesGainedMetrics   `json:"strokes_gained_avg"`
	CourseFitScore     float64                `json:"course_fit_score"`
}

// CourseHistoryEntry represents a single tournament appearance at a course
type CourseHistoryEntry struct {
	Year           int     `json:"year"`
	Position       int     `json:"position"`
	Score          int     `json:"score"`
	MadeCut        bool    `json:"made_cut"`
	RoundsPlayed   int     `json:"rounds_played"`
}

// WeatherImpactAnalysis represents weather impact on tournament play
type WeatherImpactAnalysis struct {
	TournamentID       string             `json:"tournament_id"`
	AnalysisDate       time.Time          `json:"analysis_date"`
	OverallImpact      float64            `json:"overall_impact"`
	PlayerImpacts      []PlayerWeatherImpact `json:"player_impacts"`
	CourseAdjustments  CourseWeatherAdjustment `json:"course_adjustments"`
	OptimalStrategy    WeatherStrategy    `json:"optimal_strategy"`
}

// PlayerWeatherImpact represents how weather affects an individual player
type PlayerWeatherImpact struct {
	PlayerID          int     `json:"player_id"`
	PlayerName        string  `json:"player_name"`
	WeatherAdvantage  float64 `json:"weather_advantage"`
	ImpactCategories  map[string]float64 `json:"impact_categories"`
	AdjustedProjection float64 `json:"adjusted_projection"`
}

// CourseWeatherAdjustment represents course-level weather adjustments
type CourseWeatherAdjustment struct {
	ScoreImpact        float64 `json:"score_impact"`
	DistanceReduction  float64 `json:"distance_reduction"`
	VarianceMultiplier float64 `json:"variance_multiplier"`
	SoftConditions     bool    `json:"soft_conditions"`
}

// WeatherStrategy represents optimal strategy based on weather conditions
type WeatherStrategy struct {
	StrategyType         string             `json:"strategy_type"`
	PlayerTypePreference []string           `json:"player_type_preference"`
	RecommendedWeights   map[string]float64 `json:"recommended_weights"`
}

// WeatherImpact represents the impact of weather on golf performance
type WeatherImpact struct {
	ScoreImpact        float64 `json:"score_impact"`
	DistanceReduction  float64 `json:"distance_reduction"`
	VarianceMultiplier float64 `json:"variance_multiplier"`
	SoftConditions     bool    `json:"soft_conditions"`
	WindAdvantage      float64 `json:"wind_advantage"`
	TeeTimeAdvantage   float64 `json:"tee_time_advantage"`
}

// GolfTournamentData represents basic tournament information for API providers
type GolfTournamentData struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	Status       string    `json:"status"`
	CurrentRound int       `json:"current_round"`
	CourseID     string    `json:"course_id"`
	CourseName   string    `json:"course_name"`
	CoursePar    int       `json:"course_par"`
	CourseYards  int       `json:"course_yards"`
	Purse        float64   `json:"purse"`
	CutLine      int       `json:"cut_line"`
}

// GolfPlayer represents a golf player with enhanced data
type GolfPlayer struct {
	Player
	// Golf-specific fields
	TeeTime         *string  `json:"tee_time,omitempty"`
	CutProbability  *float64 `json:"cut_probability,omitempty"`
	// TODO: Add more golf-specific fields as needed
}