package providers

import (
	"time"
)

// DataGolf API Response Types - Based on actual API testing
// All types are validated against live DataGolf API responses

// ========================================
// General Use Endpoint Response Types
// ========================================

// PlayerListResponse represents the response from /get-player-list
type PlayerListResponse []DataGolfPlayerListItem

type DataGolfPlayerListItem struct {
	Amateur     int    `json:"amateur"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
	DGID        int    `json:"dg_id"`
	PlayerName  string `json:"player_name"`
}

// ScheduleResponse represents the response from /get-schedule
type ScheduleResponse struct {
	CurrentSeason int                       `json:"current_season"`
	Schedule      []DataGolfScheduleItem    `json:"schedule"`
}

type DataGolfScheduleItem struct {
	Course      string  `json:"course"`
	CourseKey   string  `json:"course_key"`
	EventID     int     `json:"event_id"`
	EventName   string  `json:"event_name"`
	Latitude    float64 `json:"latitude"`
	Location    string  `json:"location"`
	Longitude   float64 `json:"longitude"`
	Purse       int64   `json:"purse"`
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
	Year        int     `json:"year"`
}

// FieldUpdatesResponse represents the response from /field-updates
type FieldUpdatesResponse struct {
	CurrentRound int                        `json:"current_round"`
	EventID      int                        `json:"event_id"`
	EventName    string                     `json:"event_name"`
	Field        []DataGolfFieldUpdateItem  `json:"field"`
}

type DataGolfFieldUpdateItem struct {
	Am          int     `json:"am"`
	Country     string  `json:"country"`
	Course      string  `json:"course"`
	DGID        int     `json:"dg_id"`
	DKID        string  `json:"dk_id,omitempty"`
	DKSalary    int     `json:"dk_salary,omitempty"`
	EarlyLate   string  `json:"early_late,omitempty"`
	FDID        string  `json:"fd_id,omitempty"`
	FDSalary    int     `json:"fd_salary,omitempty"`
	PlayerName  string  `json:"player_name"`
	Status      string  `json:"status,omitempty"`
	TeeTime     string  `json:"tee_time,omitempty"`
	YahooID     string  `json:"yahoo_id,omitempty"`
	YahooSalary int     `json:"yahoo_salary,omitempty"`
}

// ========================================
// Model Predictions Endpoint Response Types
// ========================================

// RankingsResponse represents the response from /preds/get-dg-rankings
type RankingsResponse struct {
	LastUpdated string                  `json:"last_updated"`
	Notes       string                  `json:"notes"`
	Rankings    []DataGolfRankingItem   `json:"rankings"`
}

type DataGolfRankingItem struct {
	Am                int     `json:"am"`
	Country           string  `json:"country"`
	DatagolfRank      int     `json:"datagolf_rank"`
	DGID              int     `json:"dg_id"`
	DGSkillEstimate   float64 `json:"dg_skill_estimate"`
	OWGRRank          int     `json:"owgr_rank"`
	PlayerName        string  `json:"player_name"`
	PrimaryTour       string  `json:"primary_tour"`
}

// PreTournamentResponse represents the response from /preds/pre-tournament
type PreTournamentResponse struct {
	Baseline   DataGolfPredictionSet `json:"baseline"`
	Event      DataGolfEventInfo     `json:"event"`
	EventCompleted bool               `json:"event_completed"`
}

type DataGolfPredictionSet struct {
	Predictions []DataGolfPredictionItem `json:"predictions"`
}

type DataGolfPredictionItem struct {
	DGID            int     `json:"dg_id"`
	PlayerName      string  `json:"player_name"`
	MakeCutProb     float64 `json:"make_cut"`
	Top5Prob        float64 `json:"top_5"`
	Top10Prob       float64 `json:"top_10"`
	Top20Prob       float64 `json:"top_20"`
	WinProb         float64 `json:"win"`
}

type DataGolfEventInfo struct {
	Course    string `json:"course"`
	EventID   int    `json:"event_id"`
	EventName string `json:"event_name"`
	Year      int    `json:"year"`
}

// SkillDecompositionsResponse represents the response from /preds/player-decompositions
type SkillDecompositionsResponse struct {
	Decompositions []DataGolfSkillDecomposition `json:"decompositions"`
	Event          DataGolfEventInfo            `json:"event"`
}

type DataGolfSkillDecomposition struct {
	DGID                 int     `json:"dg_id"`
	PlayerName           string  `json:"player_name"`
	SGApproach           float64 `json:"sg_approach"`
	SGAroundTheGreen     float64 `json:"sg_around_the_green"`
	SGOffTheTee          float64 `json:"sg_off_the_tee"`
	SGPutting            float64 `json:"sg_putting"`
	SGTotal              float64 `json:"sg_total"`
}

// SkillRatingsResponse represents the response from /preds/skill-ratings
type SkillRatingsResponse struct {
	LastUpdated string                     `json:"last_updated"`
	Ratings     []DataGolfSkillRatingItem  `json:"ratings"`
}

type DataGolfSkillRatingItem struct {
	DGID                 int     `json:"dg_id"`
	PlayerName           string  `json:"player_name"`
	SGApproach           float64 `json:"sg_approach"`
	SGAroundTheGreen     float64 `json:"sg_around_the_green"`
	SGOffTheTee          float64 `json:"sg_off_the_tee"`
	SGPutting            float64 `json:"sg_putting"`
	SGTotal              float64 `json:"sg_total"`
}

// ApproachSkillResponse represents the response from /preds/approach-skill
type ApproachSkillResponse struct {
	LastUpdated string                      `json:"last_updated"`
	Stats       []DataGolfApproachSkillItem `json:"stats"`
}

type DataGolfApproachSkillItem struct {
	DGID       int     `json:"dg_id"`
	PlayerName string  `json:"player_name"`
	// Proximity stats by distance ranges
	Prox100125 float64 `json:"prox_100_125"`
	Prox125150 float64 `json:"prox_125_150"`
	Prox150175 float64 `json:"prox_150_175"`
	Prox175200 float64 `json:"prox_175_200"`
	Prox200    float64 `json:"prox_200_plus"`
	ProxRgh    float64 `json:"prox_rgh"`
	ProxFw     float64 `json:"prox_fw"`
	// Additional approach metrics
	SGApproach float64 `json:"sg_approach"`
	GIR        float64 `json:"gir"`
	Accuracy   float64 `json:"accuracy"`
}

// FantasyProjectionsResponse represents the response from /preds/fantasy-projection-defaults
type FantasyProjectionsResponse struct {
	Event       DataGolfEventInfo             `json:"event"`
	Projections []DataGolfFantasyProjection   `json:"projections"`
	Site        string                        `json:"site"`
	Slate       string                        `json:"slate"`
}

type DataGolfFantasyProjection struct {
	DGID        int     `json:"dg_id"`
	PlayerName  string  `json:"player_name"`
	Projection  float64 `json:"projection"`
	Salary      int     `json:"salary"`
	Ownership   float64 `json:"ownership,omitempty"`
}

// ========================================
// Live Model Endpoint Response Types
// ========================================

// LivePredictionsResponse represents the response from /preds/in-play
type LivePredictionsResponse struct {
	Event         DataGolfEventInfo        `json:"event"`
	Predictions   []DataGolfLivePrediction `json:"predictions"`
	UpdatedAt     string                   `json:"updated_at"`
}

type DataGolfLivePrediction struct {
	DGID            int     `json:"dg_id"`
	PlayerName      string  `json:"player_name"`
	Position        string  `json:"position"`
	Score           string  `json:"score"`
	Thru            string  `json:"thru"`
	WinProb         float64 `json:"win"`
	Top5Prob        float64 `json:"top_5"`
	Top10Prob       float64 `json:"top_10"`
	Top20Prob       float64 `json:"top_20"`
	MakeCutProb     float64 `json:"make_cut"`
}

// LiveTournamentStatsResponse represents the response from /preds/live-tournament-stats
type LiveTournamentStatsResponse struct {
	Event        DataGolfEventInfo         `json:"event"`
	Stats        []DataGolfLiveStat        `json:"stats"`
	UpdatedAt    string                    `json:"updated_at"`
}

type DataGolfLiveStat struct {
	DGID                 int     `json:"dg_id"`
	PlayerName           string  `json:"player_name"`
	SGApproach           float64 `json:"sg_app,omitempty"`
	SGAroundTheGreen     float64 `json:"sg_arg,omitempty"`
	SGOffTheTee          float64 `json:"sg_ott,omitempty"`
	SGPutting            float64 `json:"sg_putt,omitempty"`
	SGTotal              float64 `json:"sg_total,omitempty"`
	SGTeeToGreen         float64 `json:"sg_t2g,omitempty"`
	Scrambling           float64 `json:"scrambling,omitempty"`
	GIR                  float64 `json:"gir,omitempty"`
	DrivingDistance      float64 `json:"distance,omitempty"`
	DrivingAccuracy      float64 `json:"accuracy,omitempty"`
}

// LiveHoleStatsResponse represents the response from /preds/live-hole-stats
type LiveHoleStatsResponse struct {
	Event         DataGolfEventInfo    `json:"event"`
	HoleStats     []DataGolfHoleStat   `json:"hole_stats"`
	UpdatedAt     string               `json:"updated_at"`
}

type DataGolfHoleStat struct {
	Hole         int                    `json:"hole"`
	Par          int                    `json:"par"`
	Difficulty   float64                `json:"difficulty"`
	ScoringStats DataGolfScoringStats   `json:"scoring"`
}

type DataGolfScoringStats struct {
	Average     float64 `json:"average"`
	Eagles      int     `json:"eagles"`
	Birdies     int     `json:"birdies"`
	Pars        int     `json:"pars"`
	Bogeys      int     `json:"bogeys"`
	DoublePlus  int     `json:"double_plus"`
}

// ========================================
// Historical Data Endpoint Response Types
// ========================================

// HistoricalEventListResponse represents the response from /historical-raw-data/event-list
type HistoricalEventListResponse struct {
	Events []DataGolfHistoricalEvent `json:"events"`
}

type DataGolfHistoricalEvent struct {
	Course      string `json:"course"`
	EventID     string `json:"event_id"`
	EventName   string `json:"event_name"`
	Tour        string `json:"tour"`
	Year        int    `json:"year"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
}

// HistoricalDFSEventListResponse represents the response from /historical-dfs-data/event-list
type HistoricalDFSEventListResponse struct {
	Events []DataGolfDFSEvent `json:"events"`
}

type DataGolfDFSEvent struct {
	Course      string `json:"course"`
	EventID     string `json:"event_id"`
	EventName   string `json:"event_name"`
	Tour        string `json:"tour"`
	Year        int    `json:"year"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
}

// HistoricalRoundsResponse represents the response from /historical-raw-data/rounds
type HistoricalRoundsResponse struct {
	Rounds []DataGolfHistoricalRound `json:"rounds"`
}

type DataGolfHistoricalRound struct {
	DGID               int     `json:"dg_id"`
	PlayerName         string  `json:"player_name"`
	EventID            string  `json:"event_id"`
	EventName          string  `json:"event_name"`
	Year               int     `json:"year"`
	Round              int     `json:"round"`
	Score              int     `json:"score"`
	SGApproach         float64 `json:"sg_app,omitempty"`
	SGAroundTheGreen   float64 `json:"sg_arg,omitempty"`
	SGOffTheTee        float64 `json:"sg_ott,omitempty"`
	SGPutting          float64 `json:"sg_putt,omitempty"`
	SGTotal            float64 `json:"sg_total,omitempty"`
	TeeTime            string  `json:"tee_time,omitempty"`
}

// HistoricalDFSResponse represents the response from /historical-dfs-data/points
type HistoricalDFSResponse struct {
	Results []DataGolfDFSResult `json:"results"`
}

type DataGolfDFSResult struct {
	DGID         int     `json:"dg_id"`
	PlayerName   string  `json:"player_name"`
	EventID      string  `json:"event_id"`
	Year         int     `json:"year"`
	Site         string  `json:"site"`
	Salary       int     `json:"salary"`
	Ownership    float64 `json:"ownership"`
	Points       float64 `json:"points"`
	Position     int     `json:"position"`
	MadeCut      bool    `json:"made_cut"`
}

// ========================================
// Betting Tools Endpoint Response Types
// ========================================

// OutrightsResponse represents the response from /betting-tools/outrights
type OutrightsResponse struct {
	Event     DataGolfEventInfo      `json:"event"`
	Market    string                 `json:"market"`
	Odds      []DataGolfOutrightOdd  `json:"odds"`
	UpdatedAt string                 `json:"updated_at"`
}

type DataGolfOutrightOdd struct {
	DGID         int                       `json:"dg_id"`
	PlayerName   string                    `json:"player_name"`
	ModelProb    float64                   `json:"model_prob"`
	BookOdds     map[string]DataGolfOdds   `json:"book_odds"`
}

type DataGolfOdds struct {
	Odds        float64 `json:"odds"`
	ImpliedProb float64 `json:"implied_prob"`
}

// MatchupsResponse represents the response from /betting-tools/matchups
type MatchupsResponse struct {
	Event      DataGolfEventInfo       `json:"event"`
	Market     string                  `json:"market"`
	Matchups   []DataGolfMatchup       `json:"matchups"`
	UpdatedAt  string                  `json:"updated_at"`
}

type DataGolfMatchup struct {
	MatchupID   string                    `json:"matchup_id"`
	Players     []string                  `json:"players"`
	ModelProbs  []float64                 `json:"model_probs"`
	BookOdds    map[string][]DataGolfOdds `json:"book_odds"`
}

// ========================================
// Enhanced Data Structures for Internal Use
// ========================================

// GolfTournamentData represents a standardized tournament structure
type GolfTournamentData struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	Status       string    `json:"status"`
	Purse        float64   `json:"purse"`
	CourseName   string    `json:"course_name"`
	CourseID     string    `json:"course_id,omitempty"`
	// Legacy fields for backward compatibility with RapidAPI and ESPN providers
	CurrentRound int       `json:"current_round,omitempty"`
	CoursePar    int       `json:"course_par,omitempty"`
	CourseYards  int       `json:"course_yards,omitempty"`
	CutLine      int       `json:"cut_line,omitempty"`
}

// ErrorResponse represents DataGolf API error responses
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}