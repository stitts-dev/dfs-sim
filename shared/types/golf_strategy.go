package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// TournamentPositionStrategy represents different optimization strategies for golf tournaments
type TournamentPositionStrategy string

const (
	WinStrategy       TournamentPositionStrategy = "win"
	TopFiveStrategy   TournamentPositionStrategy = "top_5"
	TopTenStrategy    TournamentPositionStrategy = "top_10"
	TopTwentyFive     TournamentPositionStrategy = "top_25"
	CutStrategy       TournamentPositionStrategy = "make_cut"
	BalancedStrategy  TournamentPositionStrategy = "balanced"
)

// WeatherImpact represents weather effects on golf performance
type WeatherImpact struct {
	ScoreImpact         float64 `json:"score_impact"`
	VarianceMultiplier  float64 `json:"variance_multiplier"`
	SoftConditions      bool    `json:"soft_conditions"`
	DistanceReduction   float64 `json:"distance_reduction"`
	WindAdvantage       float64 `json:"wind_advantage"`
	TeeTimeAdvantage    float64 `json:"tee_time_advantage"`
}

// Value implements driver.Valuer for database storage
func (w WeatherImpact) Value() (driver.Value, error) {
	return json.Marshal(w)
}

// Scan implements sql.Scanner for database retrieval
func (w *WeatherImpact) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, w)
}

// CutProbability represents comprehensive cut probability analysis
type CutProbability struct {
	PlayerID           string  `json:"player_id"`
	TournamentID       string  `json:"tournament_id"`
	BaseCutProb        float64 `json:"base_cut_probability"`
	CourseCutProb      float64 `json:"course_cut_probability"`
	WeatherAdjusted    float64 `json:"weather_adjusted_cut"`
	FinalCutProb       float64 `json:"final_cut_probability"`
	Confidence         float64 `json:"cut_confidence"`
	FieldStrengthAdj   float64 `json:"field_strength_adjustment"`
	RecentFormAdj      float64 `json:"recent_form_adjustment"`
}

// GolfOptimizationRequest extends the base optimization request with golf-specific parameters
type GolfOptimizationRequest struct {
	OptimizationRequest
	TournamentStrategy    TournamentPositionStrategy `json:"tournament_strategy"`
	CutOptimization      bool                       `json:"enable_cut_optimization"`
	WeatherConsideration bool                       `json:"include_weather"`
	CourseHistory        bool                       `json:"use_course_history"`
	TeeTimeCorrelations  bool                       `json:"tee_time_correlations"`
	RiskTolerance        float64                    `json:"risk_tolerance"`
	MinCutProbability    float64                    `json:"min_cut_probability"`
	WeightCutProbability float64                    `json:"weight_cut_probability"`
	PreferHighVariance   bool                       `json:"prefer_high_variance"`
}

// TournamentState represents live tournament tracking data
type TournamentState struct {
	TournamentID      string                 `json:"tournament_id"`
	CurrentRound      int                    `json:"current_round"`
	CutLine           int                    `json:"cut_line"`
	ProjectedCutLine  int                    `json:"projected_cut_line"`
	PlayersActive     int                    `json:"players_active"`
	PlayersWithdrawn  int                    `json:"players_withdrawn"`
	WeatherUpdate     *WeatherConditions     `json:"weather_update,omitempty"`
	LeaderPosition    *GolfLeaderboardEntry  `json:"leader_position,omitempty"`
	LastUpdate        string                 `json:"last_update"`
}

// LateSwapRecommendation represents recommendations for lineup changes
type LateSwapRecommendation struct {
	PlayerOut         string  `json:"player_out"`
	PlayerIn          string  `json:"player_in"`
	ReasonCode        string  `json:"reason_code"`
	Reasoning         string  `json:"reasoning"`
	ImpactScore       float64 `json:"impact_score"`
	Confidence        float64 `json:"confidence"`
	SwapDeadline      string  `json:"swap_deadline"`
	WeatherRelated    bool    `json:"weather_related"`
	TeeTimeAdvantage  bool    `json:"tee_time_advantage"`
}

// PlayerStats represents golf-specific player statistics for weather analysis
type PlayerStats struct {
	PlayerID                string  `json:"player_id"`
	StrokesGainedTotal      float64 `json:"strokes_gained_total"`
	StrokesGainedTeeToGreen float64 `json:"strokes_gained_tee_to_green"`
	StrokesGainedPutting    float64 `json:"strokes_gained_putting"`
	StrokesGainedDriving    float64 `json:"strokes_gained_driving"`
	StrokesGainedApproach   float64 `json:"strokes_gained_approach"`
	StrokesGainedAroundGreen float64 `json:"strokes_gained_around_green"`
	DrivingAccuracy         float64 `json:"driving_accuracy"`
	DrivingDistance         float64 `json:"driving_distance"`
	GreensInRegulation      float64 `json:"greens_in_regulation"`
	ScoringAverage          float64 `json:"scoring_average"`
	PuttingAverage          float64 `json:"putting_average"`
	BounceBackPercentage    float64 `json:"bounce_back_percentage"`
	WindResistance          float64 `json:"wind_resistance"` // Custom metric for weather analysis
	WetConditionsPerformance float64 `json:"wet_conditions_performance"`
}

// GolfStrategyAnalytics represents detailed analytics for golf optimization
type GolfStrategyAnalytics struct {
	TournamentID         string                   `json:"tournament_id"`
	Strategy             TournamentPositionStrategy `json:"strategy"`
	AvgCutProbability    float64                  `json:"avg_cut_probability"`
	AvgExpectedFinish    float64                  `json:"avg_expected_finish"`
	WeatherImpactScore   float64                  `json:"weather_impact_score"`
	FieldStrengthRating  float64                  `json:"field_strength_rating"`
	CorrelationStrength  float64                  `json:"correlation_strength"`
	RiskRewardRatio      float64                  `json:"risk_reward_ratio"`
	StackingUtilization  float64                  `json:"stacking_utilization"`
	ExposureDistribution map[string]float64       `json:"exposure_distribution"`
}