package lateswap

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// DecisionTree implements automated decision making for late swap approvals
type DecisionTree struct {
	config *RecommendationConfig
	logger *logrus.Logger
	
	// Decision rules and thresholds
	autoApprovalRules []AutoApprovalRule
	riskThresholds    *RiskThresholds
	userPreferences   map[int]*UserSwapPreferences
}

// AutoApprovalRule defines a rule for automatic swap approval
type AutoApprovalRule struct {
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	Conditions          []DecisionCondition    `json:"conditions"`
	RequiredConfidence  float64                `json:"required_confidence"`
	MaxRiskScore        float64                `json:"max_risk_score"`
	MinImpactScore      float64                `json:"min_impact_score"`
	AllowedEventTypes   []RecommendationType   `json:"allowed_event_types"`
	TimeConstraints     *TimeConstraints       `json:"time_constraints"`
	IsActive            bool                   `json:"is_active"`
	Priority            int                    `json:"priority"`
}

// DecisionCondition represents a condition in the decision tree
type DecisionCondition struct {
	Field       string      `json:"field"`        // Field to evaluate
	Operator    string      `json:"operator"`     // Comparison operator
	Value       interface{} `json:"value"`        // Expected value
	Weight      float64     `json:"weight"`       // Weight in decision (0-1)
	IsMandatory bool        `json:"is_mandatory"` // Must be satisfied
}

// RiskThresholds defines risk tolerance levels
type RiskThresholds struct {
	Conservative RiskLevel `json:"conservative"`
	Moderate     RiskLevel `json:"moderate"`
	Aggressive   RiskLevel `json:"aggressive"`
}

// RiskLevel defines thresholds for a risk tolerance level
type RiskLevel struct {
	MaxRiskScore       float64 `json:"max_risk_score"`
	MinConfidence      float64 `json:"min_confidence"`
	MinImpactScore     float64 `json:"min_impact_score"`
	AutoApprovalLimit  int     `json:"auto_approval_limit"` // Max auto-approvals per day
}

// UserSwapPreferences stores user-specific swap preferences
type UserSwapPreferences struct {
	UserID               int                    `json:"user_id"`
	RiskTolerance        string                 `json:"risk_tolerance"`        // "conservative", "moderate", "aggressive"
	AutoSwapEnabled      bool                   `json:"auto_swap_enabled"`
	MaxAutoSwapsPerDay   int                    `json:"max_auto_swaps_per_day"`
	AllowedEventTypes    []RecommendationType   `json:"allowed_event_types"`
	MinImpactThreshold   float64                `json:"min_impact_threshold"`
	RequireConfirmation  bool                   `json:"require_confirmation"`
	NotificationSettings *NotificationSettings  `json:"notification_settings"`
	LastUpdated          time.Time              `json:"last_updated"`
}

// NotificationSettings defines how users want to be notified
type NotificationSettings struct {
	SendEmail         bool `json:"send_email"`
	SendPush          bool `json:"send_push"`
	SendSMS           bool `json:"send_sms"`
	SendWebSocket     bool `json:"send_websocket"`
	RequireAck        bool `json:"require_ack"`        // Require acknowledgment
	TimeoutMinutes    int  `json:"timeout_minutes"`    // Auto-reject if no response
}

// TimeConstraints defines time-based constraints for decisions
type TimeConstraints struct {
	MinTimeToLock   time.Duration `json:"min_time_to_lock"`   // Minimum time before contest lock
	MaxTimeToLock   time.Duration `json:"max_time_to_lock"`   // Maximum time before contest lock
	AllowedHours    []int         `json:"allowed_hours"`      // Hours of day when auto-approval allowed
	BlockedDays     []string      `json:"blocked_days"`       // Days when auto-approval blocked
}

// DecisionResult represents the result of a decision tree evaluation
type DecisionResult struct {
	Decision             DecisionType        `json:"decision"`
	Confidence           float64             `json:"confidence"`
	Reason               string              `json:"reason"`
	RuleName             string              `json:"rule_name"`
	RequiresApproval     bool                `json:"requires_approval"`
	SuggestedAction      string              `json:"suggested_action"`
	WarningMessages      []string            `json:"warning_messages"`
	TimeoutMinutes       int                 `json:"timeout_minutes"`
	NextReviewTime       *time.Time          `json:"next_review_time"`
	AlternativeOptions   []AlternativeOption `json:"alternative_options"`
}

// DecisionType represents different types of decisions
type DecisionType string

const (
	DecisionAutoApprove  DecisionType = "auto_approve"
	DecisionRequireApproval DecisionType = "require_approval"
	DecisionReject       DecisionType = "reject"
	DecisionHold         DecisionType = "hold"
	DecisionEscalate     DecisionType = "escalate"
)

// AlternativeOption represents alternative recommendations
type AlternativeOption struct {
	PlayerID      uint    `json:"player_id"`
	PlayerName    string  `json:"player_name"`
	ImpactScore   float64 `json:"impact_score"`
	RiskScore     float64 `json:"risk_score"`
	Reason        string  `json:"reason"`
}

// NewDecisionTree creates a new decision tree
func NewDecisionTree(config *RecommendationConfig, logger *logrus.Logger) *DecisionTree {
	dt := &DecisionTree{
		config:          config,
		logger:          logger,
		userPreferences: make(map[int]*UserSwapPreferences),
	}
	
	// Initialize default auto-approval rules
	dt.initializeDefaultRules()
	
	// Initialize risk thresholds
	dt.initializeRiskThresholds()
	
	return dt
}

// initializeDefaultRules sets up the default auto-approval rules
func (dt *DecisionTree) initializeDefaultRules() {
	dt.autoApprovalRules = []AutoApprovalRule{
		{
			Name:        "High Confidence Injury Swap",
			Description: "Auto-approve high-confidence injury swaps",
			Conditions: []DecisionCondition{
				{
					Field:       "event_type",
					Operator:    "equals",
					Value:       RecommendationTypeInjury,
					Weight:      1.0,
					IsMandatory: true,
				},
				{
					Field:       "impact_score",
					Operator:    "greater_than",
					Value:       7.0,
					Weight:      0.8,
					IsMandatory: true,
				},
			},
			RequiredConfidence: 0.8,
			MaxRiskScore:       0.3,
			MinImpactScore:     7.0,
			AllowedEventTypes:  []RecommendationType{RecommendationTypeInjury},
			TimeConstraints: &TimeConstraints{
				MinTimeToLock: 15 * time.Minute,
			},
			IsActive: true,
			Priority: 1,
		},
		{
			Name:        "Weather Impact Swap",
			Description: "Auto-approve weather-related swaps for outdoor positions",
			Conditions: []DecisionCondition{
				{
					Field:       "event_type",
					Operator:    "equals",
					Value:       RecommendationTypeWeather,
					Weight:      1.0,
					IsMandatory: true,
				},
				{
					Field:       "impact_score",
					Operator:    "greater_than",
					Value:       5.0,
					Weight:      0.7,
					IsMandatory: true,
				},
			},
			RequiredConfidence: 0.7,
			MaxRiskScore:       0.4,
			MinImpactScore:     5.0,
			AllowedEventTypes:  []RecommendationType{RecommendationTypeWeather},
			TimeConstraints: &TimeConstraints{
				MinTimeToLock: 30 * time.Minute,
			},
			IsActive: true,
			Priority: 2,
		},
		{
			Name:        "Low Risk Value Swap",
			Description: "Auto-approve low-risk value swaps",
			Conditions: []DecisionCondition{
				{
					Field:       "risk_score",
					Operator:    "less_than",
					Value:       0.2,
					Weight:      0.9,
					IsMandatory: true,
				},
				{
					Field:       "expected_value_gain",
					Operator:    "greater_than",
					Value:       1.0,
					Weight:      0.7,
					IsMandatory: false,
				},
			},
			RequiredConfidence: 0.75,
			MaxRiskScore:       0.2,
			MinImpactScore:     4.0,
			AllowedEventTypes:  []RecommendationType{RecommendationTypeValue, RecommendationTypeProjection},
			TimeConstraints: &TimeConstraints{
				MinTimeToLock: 20 * time.Minute,
			},
			IsActive: true,
			Priority: 3,
		},
	}
}

// initializeRiskThresholds sets up default risk thresholds
func (dt *DecisionTree) initializeRiskThresholds() {
	dt.riskThresholds = &RiskThresholds{
		Conservative: RiskLevel{
			MaxRiskScore:      0.2,
			MinConfidence:     0.9,
			MinImpactScore:    6.0,
			AutoApprovalLimit: 2,
		},
		Moderate: RiskLevel{
			MaxRiskScore:      0.4,
			MinConfidence:     0.7,
			MinImpactScore:    4.0,
			AutoApprovalLimit: 5,
		},
		Aggressive: RiskLevel{
			MaxRiskScore:      0.6,
			MinConfidence:     0.6,
			MinImpactScore:    3.0,
			AutoApprovalLimit: 10,
		},
	}
}

// EvaluateRecommendation evaluates a swap recommendation using the decision tree
func (dt *DecisionTree) EvaluateRecommendation(recommendation *SwapRecommendation) *DecisionResult {
	result := &DecisionResult{
		Decision:         DecisionRequireApproval, // Default to requiring approval
		Confidence:       0.5,
		WarningMessages:  make([]string, 0),
		TimeoutMinutes:   30, // Default timeout
	}
	
	// Get user preferences
	userPrefs := dt.getUserPreferences(recommendation.UserID)
	
	// Check if auto-swap is enabled for user
	if !userPrefs.AutoSwapEnabled {
		result.Decision = DecisionRequireApproval
		result.Reason = "Auto-swap disabled by user preference"
		return result
	}
	
	// Check user's daily auto-swap limit
	if dt.hasExceededDailyLimit(recommendation.UserID, userPrefs) {
		result.Decision = DecisionRequireApproval
		result.Reason = "Daily auto-swap limit exceeded"
		result.WarningMessages = append(result.WarningMessages, "User has reached daily auto-swap limit")
		return result
	}
	
	// Check time constraints
	if !dt.checkTimeConstraints(recommendation) {
		result.Decision = DecisionRequireApproval
		result.Reason = "Time constraints not met"
		result.WarningMessages = append(result.WarningMessages, "Not enough time before contest lock")
		return result
	}
	
	// Get user's risk tolerance
	riskLevel := dt.getRiskLevel(userPrefs.RiskTolerance)
	
	// Check if recommendation meets basic risk requirements
	if !dt.meetsRiskRequirements(recommendation, riskLevel) {
		result.Decision = DecisionRequireApproval
		result.Reason = "Risk tolerance exceeded"
		result.WarningMessages = append(result.WarningMessages, fmt.Sprintf("Risk score %.2f exceeds limit %.2f", recommendation.RiskScore, riskLevel.MaxRiskScore))
		return result
	}
	
	// Evaluate against auto-approval rules
	bestRule, ruleScore := dt.evaluateAutoApprovalRules(recommendation, userPrefs)
	
	if bestRule != nil && ruleScore >= 0.8 {
		result.Decision = DecisionAutoApprove
		result.Confidence = ruleScore
		result.RuleName = bestRule.Name
		result.Reason = fmt.Sprintf("Auto-approved by rule: %s", bestRule.Name)
		result.RequiresApproval = false
	} else if ruleScore >= 0.6 {
		result.Decision = DecisionRequireApproval
		result.Confidence = ruleScore
		result.Reason = "Meets partial criteria, manual approval required"
		result.RequiresApproval = true
		result.SuggestedAction = "APPROVE"
	} else {
		result.Decision = DecisionRequireApproval
		result.Confidence = ruleScore
		result.Reason = "Does not meet auto-approval criteria"
		result.RequiresApproval = true
		result.SuggestedAction = "REVIEW"
	}
	
	// Set timeout based on user preferences
	if userPrefs.NotificationSettings != nil {
		result.TimeoutMinutes = userPrefs.NotificationSettings.TimeoutMinutes
	}
	
	return result
}

// IsAutoApprovalEligible checks if a recommendation is eligible for auto-approval
func (dt *DecisionTree) IsAutoApprovalEligible(recommendation *SwapRecommendation) bool {
	result := dt.EvaluateRecommendation(recommendation)
	return result.Decision == DecisionAutoApprove
}

// evaluateAutoApprovalRules evaluates the recommendation against all auto-approval rules
func (dt *DecisionTree) evaluateAutoApprovalRules(recommendation *SwapRecommendation, userPrefs *UserSwapPreferences) (*AutoApprovalRule, float64) {
	var bestRule *AutoApprovalRule
	bestScore := 0.0
	
	for _, rule := range dt.autoApprovalRules {
		if !rule.IsActive {
			continue
		}
		
		// Check if event type is allowed
		if !dt.isEventTypeAllowed(recommendation.RecommendationType, rule.AllowedEventTypes) {
			continue
		}
		
		// Check if event type is allowed by user
		if !dt.isEventTypeAllowed(recommendation.RecommendationType, userPrefs.AllowedEventTypes) {
			continue
		}
		
		// Evaluate rule conditions
		score := dt.evaluateRuleConditions(recommendation, rule)
		
		if score > bestScore {
			bestScore = score
			bestRule = &rule
		}
	}
	
	return bestRule, bestScore
}

// evaluateRuleConditions evaluates all conditions for a specific rule
func (dt *DecisionTree) evaluateRuleConditions(recommendation *SwapRecommendation, rule AutoApprovalRule) float64 {
	totalWeight := 0.0
	satisfiedWeight := 0.0
	mandatoryFailed := false
	
	for _, condition := range rule.Conditions {
		totalWeight += condition.Weight
		
		satisfied := dt.evaluateCondition(recommendation, condition)
		
		if satisfied {
			satisfiedWeight += condition.Weight
		} else if condition.IsMandatory {
			mandatoryFailed = true
			break
		}
	}
	
	if mandatoryFailed {
		return 0.0
	}
	
	if totalWeight == 0 {
		return 0.0
	}
	
	baseScore := satisfiedWeight / totalWeight
	
	// Apply additional checks
	if recommendation.ConfidenceScore < rule.RequiredConfidence {
		baseScore *= 0.5
	}
	
	if recommendation.RiskScore > rule.MaxRiskScore {
		baseScore *= 0.3
	}
	
	if recommendation.ImpactScore < rule.MinImpactScore {
		baseScore *= 0.7
	}
	
	return baseScore
}

// evaluateCondition evaluates a single condition
func (dt *DecisionTree) evaluateCondition(recommendation *SwapRecommendation, condition DecisionCondition) bool {
	fieldValue := dt.getFieldValue(recommendation, condition.Field)
	
	switch condition.Operator {
	case "equals":
		return fieldValue == condition.Value
	case "not_equals":
		return fieldValue != condition.Value
	case "greater_than":
		return dt.compareNumeric(fieldValue, condition.Value, ">")
	case "less_than":
		return dt.compareNumeric(fieldValue, condition.Value, "<")
	case "greater_than_or_equal":
		return dt.compareNumeric(fieldValue, condition.Value, ">=")
	case "less_than_or_equal":
		return dt.compareNumeric(fieldValue, condition.Value, "<=")
	case "contains":
		return dt.containsString(fieldValue, condition.Value)
	case "in":
		return dt.inSlice(fieldValue, condition.Value)
	default:
		dt.logger.WithField("operator", condition.Operator).Warn("Unknown condition operator")
		return false
	}
}

// Helper methods for condition evaluation
func (dt *DecisionTree) getFieldValue(recommendation *SwapRecommendation, field string) interface{} {
	switch field {
	case "impact_score":
		return recommendation.ImpactScore
	case "confidence_score":
		return recommendation.ConfidenceScore
	case "risk_score":
		return recommendation.RiskScore
	case "expected_value_gain":
		return recommendation.ExpectedValueGain
	case "event_type":
		return recommendation.RecommendationType
	case "time_to_lock":
		return recommendation.TimeToLock.Minutes()
	case "auto_approval_eligible":
		return recommendation.AutoApprovalEligible
	default:
		return nil
	}
}

func (dt *DecisionTree) compareNumeric(a, b interface{}, operator string) bool {
	aFloat, aOk := a.(float64)
	bFloat, bOk := b.(float64)
	
	if !aOk || !bOk {
		return false
	}
	
	switch operator {
	case ">":
		return aFloat > bFloat
	case "<":
		return aFloat < bFloat
	case ">=":
		return aFloat >= bFloat
	case "<=":
		return aFloat <= bFloat
	default:
		return false
	}
}

func (dt *DecisionTree) containsString(a, b interface{}) bool {
	aStr, aOk := a.(string)
	bStr, bOk := b.(string)
	
	if !aOk || !bOk {
		return false
	}
	
	return fmt.Sprintf("%s", aStr) == fmt.Sprintf("%s", bStr)
}

func (dt *DecisionTree) inSlice(a, b interface{}) bool {
	// This would check if value a is in slice b
	// Implementation depends on the specific types
	return false
}

// getUserPreferences gets user swap preferences with defaults
func (dt *DecisionTree) getUserPreferences(userID int) *UserSwapPreferences {
	if prefs, exists := dt.userPreferences[userID]; exists {
		return prefs
	}
	
	// Return default preferences
	return &UserSwapPreferences{
		UserID:              userID,
		RiskTolerance:       "moderate",
		AutoSwapEnabled:     false, // Disabled by default
		MaxAutoSwapsPerDay:  3,
		AllowedEventTypes:   []RecommendationType{RecommendationTypeInjury, RecommendationTypeWeather},
		MinImpactThreshold:  5.0,
		RequireConfirmation: true,
		NotificationSettings: &NotificationSettings{
			SendEmail:      true,
			SendPush:       true,
			SendWebSocket:  true,
			RequireAck:     true,
			TimeoutMinutes: 30,
		},
		LastUpdated: time.Now(),
	}
}

// getRiskLevel gets the risk level configuration
func (dt *DecisionTree) getRiskLevel(tolerance string) RiskLevel {
	switch tolerance {
	case "conservative":
		return dt.riskThresholds.Conservative
	case "aggressive":
		return dt.riskThresholds.Aggressive
	default:
		return dt.riskThresholds.Moderate
	}
}

// checkTimeConstraints validates time-based constraints
func (dt *DecisionTree) checkTimeConstraints(recommendation *SwapRecommendation) bool {
	// Check minimum time to lock
	if recommendation.TimeToLock < dt.config.LockTimeBuffer {
		return false
	}
	
	// Check if current time is within allowed hours (if configured)
	now := time.Now()
	hour := now.Hour()
	
	// Simple validation - can be expanded with more sophisticated time rules
	if hour < 6 || hour > 23 {
		return false // Don't auto-approve very early or very late
	}
	
	return true
}

// meetsRiskRequirements checks if recommendation meets risk requirements
func (dt *DecisionTree) meetsRiskRequirements(recommendation *SwapRecommendation, riskLevel RiskLevel) bool {
	if recommendation.RiskScore > riskLevel.MaxRiskScore {
		return false
	}
	
	if recommendation.ConfidenceScore < riskLevel.MinConfidence {
		return false
	}
	
	if recommendation.ImpactScore < riskLevel.MinImpactScore {
		return false
	}
	
	return true
}

// isEventTypeAllowed checks if an event type is in the allowed list
func (dt *DecisionTree) isEventTypeAllowed(eventType RecommendationType, allowedTypes []RecommendationType) bool {
	if len(allowedTypes) == 0 {
		return true // No restrictions
	}
	
	for _, allowed := range allowedTypes {
		if allowed == eventType {
			return true
		}
	}
	
	return false
}

// hasExceededDailyLimit checks if user has exceeded daily auto-swap limit
func (dt *DecisionTree) hasExceededDailyLimit(userID int, userPrefs *UserSwapPreferences) bool {
	// TODO: Implement actual daily limit checking using database/Redis
	// For now, return false (no limit exceeded)
	return false
}

// SetUserPreferences updates user swap preferences
func (dt *DecisionTree) SetUserPreferences(userID int, preferences *UserSwapPreferences) {
	preferences.UserID = userID
	preferences.LastUpdated = time.Now()
	dt.userPreferences[userID] = preferences
	
	dt.logger.WithFields(logrus.Fields{
		"user_id":         userID,
		"auto_enabled":    preferences.AutoSwapEnabled,
		"risk_tolerance":  preferences.RiskTolerance,
		"max_daily_swaps": preferences.MaxAutoSwapsPerDay,
	}).Info("Updated user swap preferences")
}

// GetUserPreferences returns current user preferences
func (dt *DecisionTree) GetUserPreferences(userID int) *UserSwapPreferences {
	return dt.getUserPreferences(userID)
}

// AddAutoApprovalRule adds a new auto-approval rule
func (dt *DecisionTree) AddAutoApprovalRule(rule AutoApprovalRule) {
	dt.autoApprovalRules = append(dt.autoApprovalRules, rule)
	
	dt.logger.WithFields(logrus.Fields{
		"rule_name":    rule.Name,
		"priority":     rule.Priority,
		"is_active":    rule.IsActive,
	}).Info("Added auto-approval rule")
}

// GetDecisionStats returns statistics about decision tree performance
func (dt *DecisionTree) GetDecisionStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_rules":        len(dt.autoApprovalRules),
		"active_rules":       0,
		"user_preferences":   len(dt.userPreferences),
		"auto_enabled_users": 0,
	}
	
	for _, rule := range dt.autoApprovalRules {
		if rule.IsActive {
			stats["active_rules"] = stats["active_rules"].(int) + 1
		}
	}
	
	for _, prefs := range dt.userPreferences {
		if prefs.AutoSwapEnabled {
			stats["auto_enabled_users"] = stats["auto_enabled_users"].(int) + 1
		}
	}
	
	return stats
}