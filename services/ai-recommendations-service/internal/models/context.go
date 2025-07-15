package models

import (
	"time"
)

// PromptContext represents the dynamic context for AI prompt generation
type PromptContext struct {
	Sport              string                 `json:"sport"`
	ContestType        string                 `json:"contest_type"`
	OptimizationGoal   string                 `json:"optimization_goal"`
	RealTimeData       []RealtimeDataPoint    `json:"realtime_data"`
	UserProfile        *UserAnalytics         `json:"user_profile"`
	ExistingLineups    []LineupReference      `json:"existing_lineups"`
	TimeToLock         time.Duration          `json:"time_to_lock"`
	OwnershipStrategy  string                 `json:"ownership_strategy"`
	RiskTolerance      string                 `json:"risk_tolerance"`
	ContestMeta        *ContestMetadata       `json:"contest_meta"`
}

// UserAnalytics represents user behavior and performance analytics
type UserAnalytics struct {
	UserID              uint                   `json:"user_id"`
	HistoricalROI       float64                `json:"historical_roi"`
	PreferredStrategies []string               `json:"preferred_strategies"`
	SuccessfulPatterns  map[string]float64     `json:"successful_patterns"`
	RiskProfile         string                 `json:"risk_profile"`
	SportExpertise      map[string]float64     `json:"sport_expertise"` // sport -> expertise level 0-1
	RecentPerformance   []PerformanceMetric    `json:"recent_performance"`
}

// PerformanceMetric represents a user's performance in a specific context
type PerformanceMetric struct {
	ContestType string    `json:"contest_type"`
	Sport       string    `json:"sport"`
	ROI         float64   `json:"roi"`
	Rank        int       `json:"rank"`
	TotalEntry  int       `json:"total_entries"`
	Date        time.Time `json:"date"`
}

// LineupReference represents a simplified lineup for context
type LineupReference struct {
	ID          string            `json:"id"`
	PlayerIDs   []uint            `json:"player_ids"`
	TotalSalary float64           `json:"total_salary"`
	Projection  float64           `json:"projection"`
	Strategy    string            `json:"strategy"`
	CreatedAt   time.Time         `json:"created_at"`
}

// ContestMetadata represents additional contest information
type ContestMetadata struct {
	ContestID       uint      `json:"contest_id"`
	ContestName     string    `json:"contest_name"`
	EntryFee        float64   `json:"entry_fee"`
	TotalPrize      float64   `json:"total_prize"`
	MaxEntries      int       `json:"max_entries"`
	CurrentEntries  int       `json:"current_entries"`
	SalaryCap       float64   `json:"salary_cap"`
	StartTime       time.Time `json:"start_time"`
	IsLive          bool      `json:"is_live"`
	PayoutStructure string    `json:"payout_structure"` // "top_heavy", "flat", "winner_take_all"
}

// SmartRecommendationRequest represents the request for AI recommendations
type SmartRecommendationRequest struct {
	ContestID            int      `json:"contest_id" binding:"required"`
	Sport                string   `json:"sport" binding:"required"`
	ContestType          string   `json:"contest_type" binding:"required"`
	RemainingBudget      float64  `json:"remaining_budget"`
	CurrentLineup        []int    `json:"current_lineup"`
	PositionsNeeded      []string `json:"positions_needed"`
	OptimizeFor          string   `json:"optimize_for"`
	IncludeRealTimeData  bool     `json:"include_realtime"`
	OwnershipStrategy    string   `json:"ownership_strategy"`
	ExistingLineupIDs    []string `json:"existing_lineup_ids"`
	TimeToLock           string   `json:"time_to_lock"`
	RiskTolerance        string   `json:"risk_tolerance"`
	MaxRecommendations   int      `json:"max_recommendations"`
	ExcludePlayers       []int    `json:"exclude_players"`
	MustIncludePlayers   []int    `json:"must_include_players"`
}

// SmartRecommendationResponse represents the AI-generated recommendations
type SmartRecommendationResponse struct {
	Recommendations    []PlayerRecommendation `json:"recommendations"`
	ContextInsights    []ContextInsight       `json:"context_insights"`
	OwnershipAnalysis  *OwnershipAnalysis     `json:"ownership_analysis"`
	StackSuggestions   []StackSuggestion      `json:"stack_suggestions"`
	LeverageOpportunities []LeveragePlay      `json:"leverage_opportunities"`
	RealTimeAlerts     []RealTimeAlert        `json:"realtime_alerts"`
	Confidence         float64                `json:"confidence"`
	ReasoningPath      []string               `json:"reasoning_path"`
	ModelUsed          string                 `json:"model_used"`
	TimestampGenerated time.Time              `json:"timestamp_generated"`
}

// PlayerRecommendation represents a recommended player with context
type PlayerRecommendation struct {
	PlayerID         uint     `json:"player_id"`
	PlayerName       string   `json:"player_name"`
	Position         string   `json:"position"`
	Team             string   `json:"team"`
	Opponent         string   `json:"opponent"`
	Salary           float64  `json:"salary"`
	Projection       float64  `json:"projection"`
	Ownership        float64  `json:"ownership"`
	Value            float64  `json:"value"`
	RecommendReason  string   `json:"recommend_reason"`
	Confidence       float64  `json:"confidence"`
	RiskLevel        string   `json:"risk_level"`
	Tags             []string `json:"tags"`
	RealTimeFactors  []string `json:"realtime_factors"`
}

// ContextInsight represents AI-generated insights about the contest/situation
type ContextInsight struct {
	InsightType string  `json:"insight_type"` // "weather", "injury", "matchup", "ownership", "variance"
	Message     string  `json:"message"`
	Impact      string  `json:"impact"` // "positive", "negative", "neutral"
	Confidence  float64 `json:"confidence"`
	AffectedPlayers []uint `json:"affected_players"`
}

// OwnershipAnalysis represents ownership intelligence for the contest
type OwnershipAnalysis struct {
	HighOwnership    []PlayerOwnership `json:"high_ownership"`
	LowOwnership     []PlayerOwnership `json:"low_ownership"`
	OwnershipTrends  []OwnershipTrend  `json:"ownership_trends"`
	ChalkPlays       []PlayerOwnership `json:"chalk_plays"`
	ContrianPlays    []PlayerOwnership `json:"contrarian_plays"`
	StackOwnership   map[string]float64 `json:"stack_ownership"`
}

// PlayerOwnership represents ownership data for a specific player
type PlayerOwnership struct {
	PlayerID   uint    `json:"player_id"`
	PlayerName string  `json:"player_name"`
	Ownership  float64 `json:"ownership"`
	Trend      string  `json:"trend"`
	Projection float64 `json:"projection"`
	Value      float64 `json:"value"`
}

// OwnershipTrend represents ownership movement over time
type OwnershipTrend struct {
	PlayerID        uint      `json:"player_id"`
	PlayerName      string    `json:"player_name"`
	PreviousOwnership float64 `json:"previous_ownership"`
	CurrentOwnership  float64 `json:"current_ownership"`
	TrendDirection   string   `json:"trend_direction"` // "rising", "falling", "stable"
	TrendStrength    float64  `json:"trend_strength"`
	LastUpdated      time.Time `json:"last_updated"`
}

// StackSuggestion represents AI-recommended stacking strategies
type StackSuggestion struct {
	StackType       string   `json:"stack_type"` // "game", "team", "qb", "mini"
	Players         []uint   `json:"players"`
	TotalSalary     float64  `json:"total_salary"`
	TotalProjection float64  `json:"total_projection"`
	Reasoning       string   `json:"reasoning"`
	Confidence      float64  `json:"confidence"`
	OwnershipImpact float64  `json:"ownership_impact"`
	Tags            []string `json:"tags"`
}

// LeveragePlay represents a contrarian opportunity
type LeveragePlay struct {
	PlayerID      uint    `json:"player_id"`
	PlayerName    string  `json:"player_name"`
	LeverageType  string  `json:"leverage_type"` // "contrarian", "pivot", "fade"
	OpportunityScore float64 `json:"opportunity_score"`
	OwnershipGap  float64 `json:"ownership_gap"`
	ValueRating   float64 `json:"value_rating"`
	RiskLevel     string  `json:"risk_level"`
	Reasoning     string  `json:"reasoning"`
	Confidence    float64 `json:"confidence"`
}

// RealTimeAlert represents time-sensitive information affecting recommendations
type RealTimeAlert struct {
	AlertType   string    `json:"alert_type"` // "injury", "weather", "lineup", "news"
	Severity    string    `json:"severity"`   // "low", "medium", "high", "critical"
	Message     string    `json:"message"`
	PlayerID    *uint     `json:"player_id"`
	Impact      string    `json:"impact"`
	ActionNeeded string   `json:"action_needed"`
	Timestamp   time.Time `json:"timestamp"`
}