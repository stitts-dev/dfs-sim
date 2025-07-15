package lateswap

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/models"
)

// RiskManager handles risk assessment and management for late swap recommendations
type RiskManager struct {
	db     *gorm.DB
	logger *logrus.Logger
	
	// Risk calculation parameters
	config         *RiskConfig
	
	// User risk profiles
	userProfiles   map[int]*UserRiskProfile
	profilesMutex  sync.RWMutex
	
	// Risk assessment statistics
	stats          *RiskStats
	statsMutex     sync.Mutex
}

// RiskConfig contains configuration for risk management
type RiskConfig struct {
	// Base risk thresholds
	MaxRiskScore           float64 `json:"max_risk_score"`            // Maximum allowed risk score (0-1)
	HighRiskThreshold      float64 `json:"high_risk_threshold"`       // Threshold for high-risk classification
	MediumRiskThreshold    float64 `json:"medium_risk_threshold"`     // Threshold for medium-risk classification
	
	// Variance and volatility settings
	MaxVarianceRatio       float64 `json:"max_variance_ratio"`        // Max allowed variance ratio
	MaxVolatilityScore     float64 `json:"max_volatility_score"`      // Max allowed volatility
	
	// Portfolio impact settings
	MaxPortfolioImpact     float64 `json:"max_portfolio_impact"`      // Max % impact on portfolio
	MaxCorrelationRisk     float64 `json:"max_correlation_risk"`      // Max correlation-based risk
	MaxExposureRisk        float64 `json:"max_exposure_risk"`         // Max exposure concentration risk
	
	// Time-based risk factors
	HighRiskTimeBuffer     time.Duration `json:"high_risk_time_buffer"`   // Time buffer for high-risk swaps
	LastMinuteRiskMultiplier float64     `json:"last_minute_risk_multiplier"` // Risk multiplier for last-minute swaps
}

// UserRiskProfile represents a user's risk tolerance and history
type UserRiskProfile struct {
	UserID              int                    `json:"user_id"`
	RiskTolerance       string                 `json:"risk_tolerance"`       // "conservative", "moderate", "aggressive"
	MaxDailyRiskScore   float64                `json:"max_daily_risk_score"` // Maximum daily cumulative risk
	CurrentDailyRisk    float64                `json:"current_daily_risk"`   // Current daily risk exposure
	HistoricalPerformance *PerformanceMetrics   `json:"historical_performance"`
	LastUpdated         time.Time              `json:"last_updated"`
	
	// Custom risk settings
	CustomRiskLimits    map[string]float64     `json:"custom_risk_limits"`   // Custom risk type limits
	RiskBudget          float64                `json:"risk_budget"`          // Available risk budget
	UsedRiskBudget      float64                `json:"used_risk_budget"`     // Used risk budget today
}

// PerformanceMetrics tracks user's historical swap performance
type PerformanceMetrics struct {
	TotalSwaps          int     `json:"total_swaps"`
	SuccessfulSwaps     int     `json:"successful_swaps"`
	FailedSwaps         int     `json:"failed_swaps"`
	AverageImpact       float64 `json:"average_impact"`
	AverageRiskTaken    float64 `json:"average_risk_taken"`
	RiskAdjustedReturn  float64 `json:"risk_adjusted_return"`
	SharpeRatio         float64 `json:"sharpe_ratio"`
	MaxDrawdown         float64 `json:"max_drawdown"`
	LastCalculated      time.Time `json:"last_calculated"`
}

// RiskAssessment represents a comprehensive risk assessment
type RiskAssessment struct {
	SwapID             string                 `json:"swap_id"`
	UserID             int                    `json:"user_id"`
	OverallRiskScore   float64                `json:"overall_risk_score"`    // 0-1 scale
	RiskLevel          string                 `json:"risk_level"`            // "low", "medium", "high", "extreme"
	RiskFactors        map[string]float64     `json:"risk_factors"`          // Individual risk factor scores
	RiskReasons        []string               `json:"risk_reasons"`          // Human-readable risk explanations
	Recommendation     string                 `json:"recommendation"`        // "approve", "review", "reject"
	ConfidenceScore    float64                `json:"confidence_score"`      // Confidence in risk assessment
	
	// Detailed risk breakdown
	PlayerRisk         *PlayerRiskAnalysis    `json:"player_risk"`
	PortfolioRisk      *PortfolioRiskAnalysis `json:"portfolio_risk"`
	TimingRisk         *TimingRiskAnalysis    `json:"timing_risk"`
	MarketRisk         *MarketRiskAnalysis    `json:"market_risk"`
	
	// Risk mitigation suggestions
	MitigationOptions  []RiskMitigation       `json:"mitigation_options"`
	
	CreatedAt          time.Time              `json:"created_at"`
}

// PlayerRiskAnalysis analyzes player-specific risks
type PlayerRiskAnalysis struct {
	InjuryRisk         float64 `json:"injury_risk"`          // Risk of player injury
	ProjectionRisk     float64 `json:"projection_risk"`      // Risk of projection inaccuracy
	VolatilityRisk     float64 `json:"volatility_risk"`      // Player performance volatility
	NewsRisk           float64 `json:"news_risk"`            // Risk from recent news
	WeatherRisk        float64 `json:"weather_risk"`         // Weather-related risk
	MatchupRisk        float64 `json:"matchup_risk"`         // Matchup difficulty risk
}

// PortfolioRiskAnalysis analyzes portfolio-level risks
type PortfolioRiskAnalysis struct {
	ConcentrationRisk  float64 `json:"concentration_risk"`   // Risk from overexposure
	CorrelationRisk    float64 `json:"correlation_risk"`     // Risk from player correlations
	DiversificationRisk float64 `json:"diversification_risk"` // Risk from lack of diversification
	LeverageRisk       float64 `json:"leverage_risk"`        // Risk from leverage/stacking
}

// TimingRiskAnalysis analyzes timing-related risks
type TimingRiskAnalysis struct {
	TimeToLockRisk     float64 `json:"time_to_lock_risk"`    // Risk based on time remaining
	LastMinuteRisk     float64 `json:"last_minute_risk"`     // Additional risk for last-minute swaps
	ExecutionRisk      float64 `json:"execution_risk"`       // Risk of execution failure
}

// MarketRiskAnalysis analyzes market condition risks
type MarketRiskAnalysis struct {
	OwnershipRisk      float64 `json:"ownership_risk"`       // Risk from ownership patterns
	LineMovementRisk   float64 `json:"line_movement_risk"`   // Risk from betting line changes
	VolumeRisk         float64 `json:"volume_risk"`          // Risk from contest volume
}

// RiskMitigation represents a risk mitigation option
type RiskMitigation struct {
	Type               string  `json:"type"`                 // Type of mitigation
	Description        string  `json:"description"`          // Description of mitigation
	RiskReduction      float64 `json:"risk_reduction"`       // Expected risk reduction (0-1)
	ImplementationCost float64 `json:"implementation_cost"`  // Cost to implement (0-1)
	Feasibility        float64 `json:"feasibility"`          // How feasible it is (0-1)
}

// RiskStats tracks risk management statistics
type RiskStats struct {
	AssessmentsPerformed   int64     `json:"assessments_performed"`
	HighRiskBlocked        int64     `json:"high_risk_blocked"`
	MediumRiskApproved     int64     `json:"medium_risk_approved"`
	LowRiskAutoApproved    int64     `json:"low_risk_auto_approved"`
	AverageRiskScore       float64   `json:"average_risk_score"`
	LastAssessmentTime     time.Time `json:"last_assessment_time"`
}

// NewRiskManager creates a new risk manager
func NewRiskManager(db *gorm.DB, logger *logrus.Logger) *RiskManager {
	config := &RiskConfig{
		MaxRiskScore:             0.7,  // 70% max risk
		HighRiskThreshold:        0.6,  // 60% high risk threshold
		MediumRiskThreshold:      0.3,  // 30% medium risk threshold
		MaxVarianceRatio:         2.0,  // Max 2x variance ratio
		MaxVolatilityScore:       0.8,  // Max 80% volatility
		MaxPortfolioImpact:       0.15, // Max 15% portfolio impact
		MaxCorrelationRisk:       0.5,  // Max 50% correlation risk
		MaxExposureRisk:          0.4,  // Max 40% exposure risk
		HighRiskTimeBuffer:       30 * time.Minute,
		LastMinuteRiskMultiplier: 1.5,  // 50% risk increase for last-minute
	}
	
	return &RiskManager{
		db:           db,
		logger:       logger,
		config:       config,
		userProfiles: make(map[int]*UserRiskProfile),
		stats:        &RiskStats{},
	}
}

// CalculateSwapRisk calculates the risk score for a potential swap
func (rm *RiskManager) CalculateSwapRisk(userID int, originalPlayer, candidatePlayer *Player, lineup *Lineup) float64 {
	ctx := context.Background()
	
	// Get user risk profile
	profile := rm.getUserRiskProfile(userID)
	
	// Perform comprehensive risk assessment
	assessment := rm.AssessSwapRisk(ctx, userID, originalPlayer, candidatePlayer, lineup)
	
	// Update statistics
	rm.updateRiskStats(assessment)
	
	return assessment.OverallRiskScore
}

// AssessSwapRisk performs a comprehensive risk assessment for a swap
func (rm *RiskManager) AssessSwapRisk(ctx context.Context, userID int, originalPlayer, candidatePlayer *Player, lineup *Lineup) *RiskAssessment {
	assessment := &RiskAssessment{
		SwapID:        generateSwapID(),
		UserID:        userID,
		RiskFactors:   make(map[string]float64),
		RiskReasons:   make([]string, 0),
		CreatedAt:     time.Now(),
	}
	
	// Analyze player-specific risks
	assessment.PlayerRisk = rm.analyzePlayerRisk(originalPlayer, candidatePlayer)
	playerRiskScore := rm.calculatePlayerRiskScore(assessment.PlayerRisk)
	assessment.RiskFactors["player_risk"] = playerRiskScore
	
	// Analyze portfolio risks
	assessment.PortfolioRisk = rm.analyzePortfolioRisk(userID, originalPlayer, candidatePlayer, lineup)
	portfolioRiskScore := rm.calculatePortfolioRiskScore(assessment.PortfolioRisk)
	assessment.RiskFactors["portfolio_risk"] = portfolioRiskScore
	
	// Analyze timing risks
	assessment.TimingRisk = rm.analyzeTimingRisk(lineup)
	timingRiskScore := rm.calculateTimingRiskScore(assessment.TimingRisk)
	assessment.RiskFactors["timing_risk"] = timingRiskScore
	
	// Analyze market risks
	assessment.MarketRisk = rm.analyzeMarketRisk(originalPlayer, candidatePlayer)
	marketRiskScore := rm.calculateMarketRiskScore(assessment.MarketRisk)
	assessment.RiskFactors["market_risk"] = marketRiskScore
	
	// Calculate overall risk score (weighted average)
	assessment.OverallRiskScore = rm.calculateOverallRiskScore(assessment.RiskFactors)
	
	// Determine risk level
	assessment.RiskLevel = rm.determineRiskLevel(assessment.OverallRiskScore)
	
	// Generate risk reasons
	assessment.RiskReasons = rm.generateRiskReasons(assessment)
	
	// Generate recommendation
	assessment.Recommendation = rm.generateRiskRecommendation(assessment)
	
	// Calculate confidence score
	assessment.ConfidenceScore = rm.calculateConfidenceScore(assessment)
	
	// Generate mitigation options
	assessment.MitigationOptions = rm.generateMitigationOptions(assessment)
	
	return assessment
}

// analyzePlayerRisk analyzes player-specific risk factors
func (rm *RiskManager) analyzePlayerRisk(original, candidate *Player) *PlayerRiskAnalysis {
	analysis := &PlayerRiskAnalysis{}
	
	// Injury risk (higher for injured players)
	if original.IsInjured {
		analysis.InjuryRisk = 0.8
	} else {
		analysis.InjuryRisk = 0.1
	}
	
	// Projection risk (based on variance)
	if candidate.ProjectionVariance > 0 && original.ProjectionVariance > 0 {
		varianceRatio := candidate.ProjectionVariance / original.ProjectionVariance
		analysis.ProjectionRisk = math.Min(varianceRatio/3.0, 1.0) // Cap at 1.0
	} else {
		analysis.ProjectionRisk = 0.3 // Default moderate risk
	}
	
	// Volatility risk (mock implementation)
	analysis.VolatilityRisk = 0.2 // Would calculate from historical data
	
	// News risk (mock implementation)
	analysis.NewsRisk = 0.1 // Would analyze recent news sentiment
	
	// Weather risk (position-dependent)
	if candidate.Position == "WR" || candidate.Position == "QB" {
		analysis.WeatherRisk = 0.3 // Higher weather sensitivity
	} else {
		analysis.WeatherRisk = 0.1
	}
	
	// Matchup risk (mock implementation)
	analysis.MatchupRisk = 0.2 // Would analyze opponent strength
	
	return analysis
}

// analyzePortfolioRisk analyzes portfolio-level risks
func (rm *RiskManager) analyzePortfolioRisk(userID int, original, candidate *Player, lineup *Lineup) *PortfolioRiskAnalysis {
	analysis := &PortfolioRiskAnalysis{}
	
	// Concentration risk (team/position concentration)
	analysis.ConcentrationRisk = rm.calculateConcentrationRisk(candidate, lineup)
	
	// Correlation risk (player correlations in lineup)
	analysis.CorrelationRisk = rm.calculateCorrelationRisk(original, candidate, lineup)
	
	// Diversification risk
	analysis.DiversificationRisk = rm.calculateDiversificationRisk(lineup)
	
	// Leverage risk (from stacking)
	analysis.LeverageRisk = rm.calculateLeverageRisk(candidate, lineup)
	
	return analysis
}

// analyzeTimingRisk analyzes timing-related risks
func (rm *RiskManager) analyzeTimingRisk(lineup *Lineup) *TimingRiskAnalysis {
	analysis := &TimingRiskAnalysis{}
	
	timeToLock := time.Until(lineup.LockTime)
	
	// Time to lock risk (higher as lock approaches)
	if timeToLock < 15*time.Minute {
		analysis.TimeToLockRisk = 0.8
	} else if timeToLock < 30*time.Minute {
		analysis.TimeToLockRisk = 0.5
	} else {
		analysis.TimeToLockRisk = 0.2
	}
	
	// Last minute risk
	if timeToLock < 5*time.Minute {
		analysis.LastMinuteRisk = 0.9
	} else {
		analysis.LastMinuteRisk = 0.1
	}
	
	// Execution risk (risk of failed execution)
	analysis.ExecutionRisk = analysis.TimeToLockRisk * 0.5
	
	return analysis
}

// analyzeMarketRisk analyzes market condition risks
func (rm *RiskManager) analyzeMarketRisk(original, candidate *Player) *MarketRiskAnalysis {
	analysis := &MarketRiskAnalysis{}
	
	// Ownership risk (contrarian plays are higher risk)
	ownershipDiff := math.Abs(candidate.Ownership - original.Ownership)
	analysis.OwnershipRisk = ownershipDiff / 100.0 // Normalize to 0-1
	
	// Line movement risk (mock implementation)
	analysis.LineMovementRisk = 0.2 // Would track betting line changes
	
	// Volume risk (mock implementation)
	analysis.VolumeRisk = 0.1 // Would analyze contest entry volume
	
	return analysis
}

// Helper methods for risk score calculations
func (rm *RiskManager) calculatePlayerRiskScore(analysis *PlayerRiskAnalysis) float64 {
	// Weighted average of player risk factors
	weights := map[string]float64{
		"injury":     0.25,
		"projection": 0.20,
		"volatility": 0.15,
		"news":       0.15,
		"weather":    0.15,
		"matchup":    0.10,
	}
	
	score := analysis.InjuryRisk*weights["injury"] +
		analysis.ProjectionRisk*weights["projection"] +
		analysis.VolatilityRisk*weights["volatility"] +
		analysis.NewsRisk*weights["news"] +
		analysis.WeatherRisk*weights["weather"] +
		analysis.MatchupRisk*weights["matchup"]
	
	return score
}

func (rm *RiskManager) calculatePortfolioRiskScore(analysis *PortfolioRiskAnalysis) float64 {
	// Weighted average of portfolio risk factors
	weights := map[string]float64{
		"concentration":   0.30,
		"correlation":     0.25,
		"diversification": 0.25,
		"leverage":        0.20,
	}
	
	score := analysis.ConcentrationRisk*weights["concentration"] +
		analysis.CorrelationRisk*weights["correlation"] +
		analysis.DiversificationRisk*weights["diversification"] +
		analysis.LeverageRisk*weights["leverage"]
	
	return score
}

func (rm *RiskManager) calculateTimingRiskScore(analysis *TimingRiskAnalysis) float64 {
	// Weighted average of timing risk factors
	weights := map[string]float64{
		"time_to_lock": 0.40,
		"last_minute":  0.35,
		"execution":    0.25,
	}
	
	score := analysis.TimeToLockRisk*weights["time_to_lock"] +
		analysis.LastMinuteRisk*weights["last_minute"] +
		analysis.ExecutionRisk*weights["execution"]
	
	return score
}

func (rm *RiskManager) calculateMarketRiskScore(analysis *MarketRiskAnalysis) float64 {
	// Weighted average of market risk factors
	weights := map[string]float64{
		"ownership":     0.40,
		"line_movement": 0.35,
		"volume":        0.25,
	}
	
	score := analysis.OwnershipRisk*weights["ownership"] +
		analysis.LineMovementRisk*weights["line_movement"] +
		analysis.VolumeRisk*weights["volume"]
	
	return score
}

func (rm *RiskManager) calculateOverallRiskScore(riskFactors map[string]float64) float64 {
	// Weighted average of all risk categories
	weights := map[string]float64{
		"player_risk":    0.30,
		"portfolio_risk": 0.25,
		"timing_risk":    0.25,
		"market_risk":    0.20,
	}
	
	totalScore := 0.0
	totalWeight := 0.0
	
	for factor, score := range riskFactors {
		if weight, exists := weights[factor]; exists {
			totalScore += score * weight
			totalWeight += weight
		}
	}
	
	if totalWeight > 0 {
		return totalScore / totalWeight
	}
	
	return 0.5 // Default moderate risk
}

// Additional helper methods
func (rm *RiskManager) calculateConcentrationRisk(player *Player, lineup *Lineup) float64 {
	// Mock implementation - would calculate actual team/position concentration
	return 0.2
}

func (rm *RiskManager) calculateCorrelationRisk(original, candidate *Player, lineup *Lineup) float64 {
	// Mock implementation - would calculate player correlations
	return 0.3
}

func (rm *RiskManager) calculateDiversificationRisk(lineup *Lineup) float64 {
	// Mock implementation - would analyze lineup diversification
	return 0.2
}

func (rm *RiskManager) calculateLeverageRisk(player *Player, lineup *Lineup) float64 {
	// Mock implementation - would calculate stacking leverage
	return 0.25
}

func (rm *RiskManager) determineRiskLevel(score float64) string {
	if score >= rm.config.HighRiskThreshold {
		return "high"
	} else if score >= rm.config.MediumRiskThreshold {
		return "medium"
	} else {
		return "low"
	}
}

func (rm *RiskManager) generateRiskReasons(assessment *RiskAssessment) []string {
	reasons := make([]string, 0)
	
	if assessment.PlayerRisk.InjuryRisk > 0.5 {
		reasons = append(reasons, "High injury risk for original player")
	}
	if assessment.PortfolioRisk.ConcentrationRisk > 0.4 {
		reasons = append(reasons, "High team/position concentration")
	}
	if assessment.TimingRisk.TimeToLockRisk > 0.6 {
		reasons = append(reasons, "Limited time remaining before lock")
	}
	if assessment.MarketRisk.OwnershipRisk > 0.3 {
		reasons = append(reasons, "Significant ownership differential")
	}
	
	return reasons
}

func (rm *RiskManager) generateRiskRecommendation(assessment *RiskAssessment) string {
	if assessment.OverallRiskScore > rm.config.HighRiskThreshold {
		return "reject"
	} else if assessment.OverallRiskScore > rm.config.MediumRiskThreshold {
		return "review"
	} else {
		return "approve"
	}
}

func (rm *RiskManager) calculateConfidenceScore(assessment *RiskAssessment) float64 {
	// Higher confidence for more data points and consistent risk factors
	return 0.75 // Mock confidence score
}

func (rm *RiskManager) generateMitigationOptions(assessment *RiskAssessment) []RiskMitigation {
	options := make([]RiskMitigation, 0)
	
	if assessment.TimingRisk.TimeToLockRisk > 0.5 {
		options = append(options, RiskMitigation{
			Type:               "timing",
			Description:        "Wait for more information closer to lock",
			RiskReduction:      0.2,
			ImplementationCost: 0.1,
			Feasibility:        0.8,
		})
	}
	
	return options
}

func (rm *RiskManager) getUserRiskProfile(userID int) *UserRiskProfile {
	rm.profilesMutex.RLock()
	profile, exists := rm.userProfiles[userID]
	rm.profilesMutex.RUnlock()
	
	if !exists {
		// Create default profile
		profile = &UserRiskProfile{
			UserID:           userID,
			RiskTolerance:    "moderate",
			MaxDailyRiskScore: 0.5,
			CurrentDailyRisk: 0.0,
			CustomRiskLimits: make(map[string]float64),
			RiskBudget:       1.0,
			UsedRiskBudget:   0.0,
			LastUpdated:      time.Now(),
		}
		
		rm.profilesMutex.Lock()
		rm.userProfiles[userID] = profile
		rm.profilesMutex.Unlock()
	}
	
	return profile
}

func (rm *RiskManager) updateRiskStats(assessment *RiskAssessment) {
	rm.statsMutex.Lock()
	defer rm.statsMutex.Unlock()
	
	rm.stats.AssessmentsPerformed++
	rm.stats.LastAssessmentTime = time.Now()
	
	switch assessment.RiskLevel {
	case "high":
		rm.stats.HighRiskBlocked++
	case "medium":
		rm.stats.MediumRiskApproved++
	case "low":
		rm.stats.LowRiskAutoApproved++
	}
	
	// Update average risk score
	totalAssessments := float64(rm.stats.AssessmentsPerformed)
	rm.stats.AverageRiskScore = (rm.stats.AverageRiskScore*(totalAssessments-1) + assessment.OverallRiskScore) / totalAssessments
}

// GetRiskStats returns current risk management statistics
func (rm *RiskManager) GetRiskStats() RiskStats {
	rm.statsMutex.Lock()
	defer rm.statsMutex.Unlock()
	return *rm.stats
}

// SetUserRiskProfile sets a custom risk profile for a user
func (rm *RiskManager) SetUserRiskProfile(profile *UserRiskProfile) {
	rm.profilesMutex.Lock()
	profile.LastUpdated = time.Now()
	rm.userProfiles[profile.UserID] = profile
	rm.profilesMutex.Unlock()
	
	rm.logger.WithFields(logrus.Fields{
		"user_id":        profile.UserID,
		"risk_tolerance": profile.RiskTolerance,
		"max_daily_risk": profile.MaxDailyRiskScore,
	}).Info("Updated user risk profile")
}

// Utility function to generate swap IDs
func generateSwapID() string {
	return fmt.Sprintf("swap_%d", time.Now().UnixNano())
}