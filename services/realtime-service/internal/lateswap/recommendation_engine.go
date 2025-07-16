package lateswap

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/models"
)

// RecommendationEngine generates intelligent late swap recommendations
type RecommendationEngine struct {
	db             *gorm.DB
	redisClient    *redis.Client
	logger         *logrus.Logger
	decisionTree   *DecisionTree
	riskManager    *RiskManager
	
	// Configuration
	config         *RecommendationConfig
	
	// Statistics
	stats          *RecommendationStats
	
	// Active recommendations tracking
	activeRecs     map[string]*models.LateSwapRecommendation
	recMutex       sync.RWMutex
}

// RecommendationConfig contains configuration for the recommendation engine
type RecommendationConfig struct {
	MinImpactThreshold    float64       `json:"min_impact_threshold"`    // Minimum impact to trigger recommendation
	MaxRecommendations    int           `json:"max_recommendations"`     // Max recommendations per user per contest
	RecommendationTTL     time.Duration `json:"recommendation_ttl"`      // How long recommendations are valid
	AutoApprovalThreshold float64       `json:"auto_approval_threshold"` // Impact threshold for auto-approval
	MinConfidenceScore    float64       `json:"min_confidence_score"`    // Minimum confidence for recommendations
	MaxRiskScore          float64       `json:"max_risk_score"`          // Maximum risk score allowed
	EnableAutoSwap        bool          `json:"enable_auto_swap"`        // Enable automatic swaps
	LockTimeBuffer        time.Duration `json:"lock_time_buffer"`        // Time buffer before contest lock
}

// RecommendationStats tracks recommendation engine performance
type RecommendationStats struct {
	RecommendationsGenerated int64     `json:"recommendations_generated"`
	RecommendationsAccepted  int64     `json:"recommendations_accepted"`
	RecommendationsRejected  int64     `json:"recommendations_rejected"`
	AutoApprovalsExecuted    int64     `json:"auto_approvals_executed"`
	AverageImpactScore       float64   `json:"average_impact_score"`
	AverageConfidenceScore   float64   `json:"average_confidence_score"`
	LastGeneratedTime        time.Time `json:"last_generated_time"`
	SuccessRate              float64   `json:"success_rate"`
}

// SwapRecommendation represents a late swap recommendation with detailed analysis
type SwapRecommendation struct {
	ID                    string                 `json:"id"`
	UserID                int                    `json:"user_id"`
	ContestID             string                 `json:"contest_id"`
	OriginalPlayerID      uint                   `json:"original_player_id"`
	RecommendedPlayerID   uint                   `json:"recommended_player_id"`
	
	// Scoring and analysis
	ImpactScore           float64                `json:"impact_score"`          // -10 to +10
	ConfidenceScore       float64                `json:"confidence_score"`      // 0-1
	RiskScore             float64                `json:"risk_score"`            // 0-1
	ExpectedValueGain     float64                `json:"expected_value_gain"`   // Expected EV improvement
	
	// Swap analysis details
	SwapReason            string                 `json:"swap_reason"`
	AnalysisDetails       map[string]interface{} `json:"analysis_details"`
	ProjectionComparison  *ProjectionComparison  `json:"projection_comparison"`
	OwnershipAnalysis     *OwnershipAnalysis     `json:"ownership_analysis"`
	
	// Decision and timing
	RecommendationType    RecommendationType     `json:"recommendation_type"`
	AutoApprovalEligible  bool                   `json:"auto_approval_eligible"`
	TimeToLock            time.Duration          `json:"time_to_lock"`
	ExpiresAt             time.Time              `json:"expires_at"`
	
	// Status tracking
	Status                SwapStatus             `json:"status"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

// ProjectionComparison compares projections between original and recommended players
type ProjectionComparison struct {
	OriginalProjection    float64 `json:"original_projection"`
	RecommendedProjection float64 `json:"recommended_projection"`
	ProjectionDifference  float64 `json:"projection_difference"`
	ValueDifference       float64 `json:"value_difference"`      // Points per dollar
	CeilingComparison     float64 `json:"ceiling_comparison"`    // Upside potential
	FloorComparison       float64 `json:"floor_comparison"`      // Downside protection
}

// OwnershipAnalysis analyzes ownership implications of the swap
type OwnershipAnalysis struct {
	OriginalOwnership     float64 `json:"original_ownership"`
	RecommendedOwnership  float64 `json:"recommended_ownership"`
	OwnershipDifference   float64 `json:"ownership_difference"`
	LeverageGain          float64 `json:"leverage_gain"`
	ContrarianValue       float64 `json:"contrarian_value"`
	StackingImpact        float64 `json:"stacking_impact"`
}

// RecommendationType represents different types of swap recommendations
type RecommendationType string

const (
	RecommendationTypeInjury       RecommendationType = "injury"
	RecommendationTypeWeather      RecommendationType = "weather"
	RecommendationTypeOwnership    RecommendationType = "ownership"
	RecommendationTypeProjection   RecommendationType = "projection"
	RecommendationTypeValue        RecommendationType = "value"
	RecommendationTypeNews         RecommendationType = "news"
	RecommendationTypeStack        RecommendationType = "stack"
)

// SwapStatus represents the status of a swap recommendation
type SwapStatus string

const (
	SwapStatusPending     SwapStatus = "pending"
	SwapStatusApproved    SwapStatus = "approved"
	SwapStatusRejected    SwapStatus = "rejected"
	SwapStatusExecuted    SwapStatus = "executed"
	SwapStatusExpired     SwapStatus = "expired"
	SwapStatusCancelled   SwapStatus = "cancelled"
)

// NewRecommendationEngine creates a new recommendation engine
func NewRecommendationEngine(db *gorm.DB, redisClient *redis.Client, logger *logrus.Logger) *RecommendationEngine {
	config := &RecommendationConfig{
		MinImpactThreshold:    3.0,
		MaxRecommendations:    5,
		RecommendationTTL:     30 * time.Minute,
		AutoApprovalThreshold: 7.0,
		MinConfidenceScore:    0.7,
		MaxRiskScore:          0.5,
		EnableAutoSwap:        false, // Disabled by default for safety
		LockTimeBuffer:        10 * time.Minute,
	}
	
	engine := &RecommendationEngine{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
		config:      config,
		stats:       &RecommendationStats{},
		activeRecs:  make(map[string]*models.LateSwapRecommendation),
	}
	
	// Initialize sub-components
	engine.decisionTree = NewDecisionTree(config, logger)
	engine.riskManager = NewRiskManager(db, logger)
	
	return engine
}

// GenerateRecommendations generates swap recommendations based on real-time events
func (re *RecommendationEngine) GenerateRecommendations(ctx context.Context, event models.RealTimeEvent) ([]*SwapRecommendation, error) {
	recommendations := make([]*SwapRecommendation, 0)
	
	// Determine if this event warrants swap recommendations
	if !re.shouldGenerateRecommendations(event) {
		return recommendations, nil
	}
	
	// Get affected users (those who have the affected player in active lineups)
	affectedUsers, err := re.getAffectedUsers(event)
	if err != nil {
		return nil, fmt.Errorf("failed to get affected users: %w", err)
	}
	
	// Generate recommendations for each affected user
	for _, userID := range affectedUsers {
		userRecs, err := re.generateUserRecommendations(ctx, userID, event)
		if err != nil {
			re.logger.WithError(err).WithField("user_id", userID).Error("Failed to generate user recommendations")
			continue
		}
		
		recommendations = append(recommendations, userRecs...)
	}
	
	// Update statistics
	re.updateStats(recommendations)
	
	re.logger.WithFields(logrus.Fields{
		"event_type":         event.EventType,
		"event_id":          event.EventID,
		"recommendations":    len(recommendations),
		"affected_users":     len(affectedUsers),
	}).Info("Generated late swap recommendations")
	
	return recommendations, nil
}

// shouldGenerateRecommendations determines if an event should trigger recommendations
func (re *RecommendationEngine) shouldGenerateRecommendations(event models.RealTimeEvent) bool {
	// Only generate recommendations for high-impact events
	if event.ImpactRating < re.config.MinImpactThreshold {
		return false
	}
	
	// Only generate for certain event types
	switch event.EventType {
	case models.EventTypePlayerInjury, 
		 models.EventTypeWeatherUpdate, 
		 models.EventTypeOwnershipChange,
		 models.EventTypeNewsUpdate:
		return true
	default:
		return false
	}
}

// getAffectedUsers finds users who have the affected player in active lineups
func (re *RecommendationEngine) getAffectedUsers(event models.RealTimeEvent) ([]int, error) {
	// This would query the database for users with active lineups containing the affected player
	// For now, we'll return a mock list
	
	if event.PlayerID == nil {
		return []int{}, nil
	}
	
	// TODO: Implement actual database query
	// SELECT DISTINCT user_id FROM lineups l 
	// JOIN lineup_players lp ON l.id = lp.lineup_id 
	// WHERE lp.player_id = ? AND l.contest_lock_time > NOW()
	
	// Mock affected users
	affectedUsers := []int{1001, 1002, 1003}
	
	return affectedUsers, nil
}

// generateUserRecommendations generates recommendations for a specific user
func (re *RecommendationEngine) generateUserRecommendations(ctx context.Context, userID int, event models.RealTimeEvent) ([]*SwapRecommendation, error) {
	recommendations := make([]*SwapRecommendation, 0)
	
	// Get user's active lineups that could be affected
	affectedLineups, err := re.getUserAffectedLineups(userID, event)
	if err != nil {
		return nil, fmt.Errorf("failed to get affected lineups: %w", err)
	}
	
	// Generate recommendations for each affected lineup
	for _, lineup := range affectedLineups {
		lineupRecs, err := re.generateLineupRecommendations(ctx, userID, lineup, event)
		if err != nil {
			re.logger.WithError(err).WithFields(logrus.Fields{
				"user_id":   userID,
				"lineup_id": lineup.ID,
			}).Error("Failed to generate lineup recommendations")
			continue
		}
		
		recommendations = append(recommendations, lineupRecs...)
	}
	
	// Limit recommendations per user
	if len(recommendations) > re.config.MaxRecommendations {
		// Sort by impact score and keep the best ones
		sort.Slice(recommendations, func(i, j int) bool {
			return recommendations[i].ImpactScore > recommendations[j].ImpactScore
		})
		recommendations = recommendations[:re.config.MaxRecommendations]
	}
	
	return recommendations, nil
}

// generateLineupRecommendations generates recommendations for a specific lineup
func (re *RecommendationEngine) generateLineupRecommendations(ctx context.Context, userID int, lineup *Lineup, event models.RealTimeEvent) ([]*SwapRecommendation, error) {
	recommendations := make([]*SwapRecommendation, 0)
	
	// Find the affected player in the lineup
	affectedPlayer := re.findAffectedPlayerInLineup(lineup, event)
	if affectedPlayer == nil {
		return recommendations, nil
	}
	
	// Get potential replacement players
	replacementCandidates, err := re.getReplacementCandidates(lineup, affectedPlayer, event)
	if err != nil {
		return nil, fmt.Errorf("failed to get replacement candidates: %w", err)
	}
	
	// Evaluate each replacement candidate
	for _, candidate := range replacementCandidates {
		recommendation := re.evaluateSwapCandidate(userID, lineup, affectedPlayer, candidate, event)
		
		// Apply filters
		if recommendation.ImpactScore < re.config.MinImpactThreshold {
			continue
		}
		
		if recommendation.ConfidenceScore < re.config.MinConfidenceScore {
			continue
		}
		
		if recommendation.RiskScore > re.config.MaxRiskScore {
			continue
		}
		
		recommendations = append(recommendations, recommendation)
	}
	
	// Sort by impact score
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].ImpactScore > recommendations[j].ImpactScore
	})
	
	return recommendations, nil
}

// evaluateSwapCandidate evaluates a potential player swap
func (re *RecommendationEngine) evaluateSwapCandidate(userID int, lineup *Lineup, originalPlayer, candidatePlayer *Player, event models.RealTimeEvent) *SwapRecommendation {
	recommendation := &SwapRecommendation{
		ID:                  generateRecommendationID(),
		UserID:              userID,
		ContestID:           lineup.ContestID,
		OriginalPlayerID:    originalPlayer.ID,
		RecommendedPlayerID: candidatePlayer.ID,
		Status:              SwapStatusPending,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	
	// Calculate impact score
	recommendation.ImpactScore = re.calculateImpactScore(originalPlayer, candidatePlayer, event)
	
	// Calculate confidence score
	recommendation.ConfidenceScore = re.calculateConfidenceScore(originalPlayer, candidatePlayer, event)
	
	// Calculate risk score
	recommendation.RiskScore = re.riskManager.CalculateSwapRisk(userID, originalPlayer, candidatePlayer, lineup)
	
	// Calculate expected value gain
	recommendation.ExpectedValueGain = re.calculateExpectedValueGain(originalPlayer, candidatePlayer)
	
	// Determine recommendation type
	recommendation.RecommendationType = re.determineRecommendationType(event)
	
	// Generate swap reason
	recommendation.SwapReason = re.generateSwapReason(originalPlayer, candidatePlayer, event)
	
	// Projection comparison
	recommendation.ProjectionComparison = re.compareProjections(originalPlayer, candidatePlayer)
	
	// Ownership analysis
	recommendation.OwnershipAnalysis = re.analyzeOwnership(originalPlayer, candidatePlayer)
	
	// Auto-approval eligibility
	recommendation.AutoApprovalEligible = re.decisionTree.IsAutoApprovalEligible(recommendation)
	
	// Set expiration
	recommendation.ExpiresAt = time.Now().Add(re.config.RecommendationTTL)
	
	// Time to lock
	recommendation.TimeToLock = time.Until(lineup.LockTime)
	
	return recommendation
}

// calculateImpactScore calculates the impact score of a swap
func (re *RecommendationEngine) calculateImpactScore(original, candidate *Player, event models.RealTimeEvent) float64 {
	score := 0.0
	
	// Base projection difference
	projectionDiff := candidate.Projection - original.Projection
	score += projectionDiff * 2.0 // Scale projection difference
	
	// Event-specific adjustments
	switch event.EventType {
	case models.EventTypePlayerInjury:
		if original.ID == *event.PlayerID {
			// Heavily penalize injured player
			score += event.ImpactRating * -1.0
		}
	case models.EventTypeWeatherUpdate:
		// Weather affects certain positions more
		if original.Position == "WR" || original.Position == "QB" {
			score += event.ImpactRating * 0.5
		}
	case models.EventTypeOwnershipChange:
		// Ownership changes affect leverage
		ownershipDiff := candidate.Ownership - original.Ownership
		if ownershipDiff < 0 { // Lower ownership = better leverage
			score += math.Abs(ownershipDiff) * 0.3
		}
	}
	
	// Salary efficiency
	if candidate.Salary > 0 && original.Salary > 0 {
		originalValue := original.Projection / float64(original.Salary) * 1000
		candidateValue := candidate.Projection / float64(candidate.Salary) * 1000
		valueDiff := candidateValue - originalValue
		score += valueDiff * 5.0
	}
	
	// Cap the score between -10 and +10
	if score > 10 {
		score = 10
	}
	if score < -10 {
		score = -10
	}
	
	return score
}

// calculateConfidenceScore calculates confidence in the recommendation
func (re *RecommendationEngine) calculateConfidenceScore(original, candidate *Player, event models.RealTimeEvent) float64 {
	confidence := 0.5 // Base confidence
	
	// Event confidence
	confidence += event.Confidence * 0.3
	
	// Projection reliability
	if candidate.ProjectionVariance > 0 && original.ProjectionVariance > 0 {
		// Lower variance = higher confidence
		varianceRatio := original.ProjectionVariance / candidate.ProjectionVariance
		if varianceRatio > 1 {
			confidence += 0.2
		}
	}
	
	// Historical performance correlation
	// This would use historical data to determine confidence
	confidence += 0.1 // Mock addition
	
	// Cap between 0 and 1
	if confidence > 1 {
		confidence = 1
	}
	if confidence < 0 {
		confidence = 0
	}
	
	return confidence
}

// calculateExpectedValueGain calculates expected DFS value improvement
func (re *RecommendationEngine) calculateExpectedValueGain(original, candidate *Player) float64 {
	// Simple expected value calculation
	// In practice, this would consider:
	// - Projection differences
	// - Ownership leverag
	// - Ceiling/floor analysis
	// - Game theory optimal play
	
	projectionGain := candidate.Projection - original.Projection
	ownershipLeverage := (original.Ownership - candidate.Ownership) * 0.1
	
	return projectionGain + ownershipLeverage
}

// Additional helper methods and types would be defined here...

// Mock types for compilation
type Lineup struct {
	ID        uint
	ContestID string
	LockTime  time.Time
	Players   []*Player
}

type Player struct {
	ID                 uint
	Name               string
	Position           string
	Team               string
	Salary             int
	Projection         float64
	ProjectionVariance float64
	Ownership          float64
	IsInjured          bool
}

func (re *RecommendationEngine) getUserAffectedLineups(userID int, event models.RealTimeEvent) ([]*Lineup, error) {
	// Mock implementation
	return []*Lineup{
		{
			ID:        1,
			ContestID: "contest_123",
			LockTime:  time.Now().Add(time.Hour),
		},
	}, nil
}

func (re *RecommendationEngine) findAffectedPlayerInLineup(lineup *Lineup, event models.RealTimeEvent) *Player {
	// Mock implementation
	if event.PlayerID != nil {
		return &Player{
			ID:         *event.PlayerID,
			Name:       "Affected Player",
			Position:   "WR",
			Salary:     8500,
			Projection: 15.2,
			Ownership:  25.5,
		}
	}
	return nil
}

func (re *RecommendationEngine) getReplacementCandidates(lineup *Lineup, affectedPlayer *Player, event models.RealTimeEvent) ([]*Player, error) {
	// Mock replacement candidates
	return []*Player{
		{
			ID:         2001,
			Name:       "Replacement 1",
			Position:   affectedPlayer.Position,
			Salary:     8400,
			Projection: 16.1,
			Ownership:  18.3,
		},
		{
			ID:         2002,
			Name:       "Replacement 2", 
			Position:   affectedPlayer.Position,
			Salary:     8600,
			Projection: 15.8,
			Ownership:  22.1,
		},
	}, nil
}

func (re *RecommendationEngine) determineRecommendationType(event models.RealTimeEvent) RecommendationType {
	switch event.EventType {
	case models.EventTypePlayerInjury:
		return RecommendationTypeInjury
	case models.EventTypeWeatherUpdate:
		return RecommendationTypeWeather
	case models.EventTypeOwnershipChange:
		return RecommendationTypeOwnership
	case models.EventTypeNewsUpdate:
		return RecommendationTypeNews
	default:
		return RecommendationTypeProjection
	}
}

func (re *RecommendationEngine) generateSwapReason(original, candidate *Player, event models.RealTimeEvent) string {
	switch event.EventType {
	case models.EventTypePlayerInjury:
		return fmt.Sprintf("%s injury update - swap to %s for safer floor", original.Name, candidate.Name)
	case models.EventTypeWeatherUpdate:
		return fmt.Sprintf("Weather concerns - %s has better matchup indoors", candidate.Name)
	case models.EventTypeOwnershipChange:
		return fmt.Sprintf("Leverage opportunity - %s at lower ownership (%.1f%%)", candidate.Name, candidate.Ownership)
	default:
		return fmt.Sprintf("Projection update favors %s (+%.1f pts)", candidate.Name, candidate.Projection-original.Projection)
	}
}

func (re *RecommendationEngine) compareProjections(original, candidate *Player) *ProjectionComparison {
	return &ProjectionComparison{
		OriginalProjection:    original.Projection,
		RecommendedProjection: candidate.Projection,
		ProjectionDifference:  candidate.Projection - original.Projection,
		ValueDifference:       (candidate.Projection/float64(candidate.Salary) - original.Projection/float64(original.Salary)) * 1000,
		CeilingComparison:     candidate.Projection * 1.3 - original.Projection * 1.3, // Mock ceiling calc
		FloorComparison:       candidate.Projection * 0.7 - original.Projection * 0.7, // Mock floor calc
	}
}

func (re *RecommendationEngine) analyzeOwnership(original, candidate *Player) *OwnershipAnalysis {
	return &OwnershipAnalysis{
		OriginalOwnership:     original.Ownership,
		RecommendedOwnership:  candidate.Ownership,
		OwnershipDifference:   candidate.Ownership - original.Ownership,
		LeverageGain:          (original.Ownership - candidate.Ownership) * 0.1, // Mock leverage calc
		ContrarianValue:       math.Max(0, original.Ownership-candidate.Ownership) * 0.2,
		StackingImpact:        0.0, // Would calculate team/game stack implications
	}
}

func (re *RecommendationEngine) updateStats(recommendations []*SwapRecommendation) {
	re.stats.RecommendationsGenerated += int64(len(recommendations))
	re.stats.LastGeneratedTime = time.Now()
	
	if len(recommendations) > 0 {
		totalImpact := 0.0
		totalConfidence := 0.0
		
		for _, rec := range recommendations {
			totalImpact += rec.ImpactScore
			totalConfidence += rec.ConfidenceScore
		}
		
		re.stats.AverageImpactScore = totalImpact / float64(len(recommendations))
		re.stats.AverageConfidenceScore = totalConfidence / float64(len(recommendations))
	}
}

func generateRecommendationID() string {
	return fmt.Sprintf("rec_%d", time.Now().UnixNano())
}