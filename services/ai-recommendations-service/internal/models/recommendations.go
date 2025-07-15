package models

import (
	"time"
	"encoding/json"
)

// AIRecommendation represents a stored AI recommendation
type AIRecommendation struct {
	ID             uint            `json:"id" gorm:"primaryKey"`
	UserID         uint            `json:"user_id" gorm:"not null"`
	ContestID      uint            `json:"contest_id" gorm:"not null"`
	Request        json.RawMessage `json:"request" gorm:"type:jsonb"`
	Response       json.RawMessage `json:"response" gorm:"type:jsonb"`
	ModelUsed      string          `json:"model_used" gorm:"size:50;not null"`
	Confidence     float64         `json:"confidence" gorm:"not null"`
	TokensUsed     *int            `json:"tokens_used"`
	ResponseTimeMs *int            `json:"response_time_ms"`
	CreatedAt      time.Time       `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// OwnershipSnapshot represents real-time ownership data
type OwnershipSnapshot struct {
	ID                  uint       `json:"id" gorm:"primaryKey"`
	ContestID           uint       `json:"contest_id" gorm:"not null"`
	PlayerID            uint       `json:"player_id" gorm:"not null"`
	OwnershipPercentage float64    `json:"ownership_percentage" gorm:"not null"`
	ProjectedOwnership  *float64   `json:"projected_ownership"`
	Trend               *string    `json:"trend" gorm:"size:20"`
	LeverageScore       *float64   `json:"leverage_score"`
	ChalkFactor         *float64   `json:"chalk_factor"`
	SnapshotTime        time.Time  `json:"snapshot_time" gorm:"not null"`
	Source              *string    `json:"source" gorm:"size:50"`
	ConfidenceInterval  *float64   `json:"confidence_interval"`
}

// RecommendationFeedback tracks user interactions with AI recommendations
type RecommendationFeedback struct {
	ID               uint            `json:"id" gorm:"primaryKey"`
	RecommendationID uint            `json:"recommendation_id" gorm:"not null"`
	UserID           uint            `json:"user_id" gorm:"not null"`
	FeedbackType     string          `json:"feedback_type" gorm:"size:50"` // 'followed', 'ignored', 'partial'
	LineupResult     json.RawMessage `json:"lineup_result" gorm:"type:jsonb"`
	ROI              *float64        `json:"roi"`
	SatisfactionScore *int           `json:"satisfaction_score"` // 1-5 rating
	Notes            *string         `json:"notes" gorm:"type:text"`
	CreatedAt        time.Time       `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	
	AIRecommendation AIRecommendation `json:"ai_recommendation" gorm:"foreignKey:RecommendationID"`
}

// RealtimeDataPoint represents live data affecting recommendations
type RealtimeDataPoint struct {
	ID           uint            `json:"id" gorm:"primaryKey"`
	PlayerID     uint            `json:"player_id" gorm:"not null"`
	ContestID    uint            `json:"contest_id" gorm:"not null"`
	DataType     string          `json:"data_type" gorm:"size:50;not null"` // 'injury', 'weather', 'ownership', 'odds', 'news'
	Value        json.RawMessage `json:"value" gorm:"type:jsonb;not null"`
	Confidence   float64         `json:"confidence" gorm:"not null"` // 0-1 reliability score
	ImpactRating *float64        `json:"impact_rating"` // -5 to +5 DFS impact
	Source       string          `json:"source" gorm:"size:100;not null"`
	Timestamp    time.Time       `json:"timestamp" gorm:"not null"`
	ExpiresAt    *time.Time      `json:"expires_at"`
}

// LeverageOpportunity represents contrarian play opportunities
type LeverageOpportunity struct {
	ID                    uint      `json:"id" gorm:"primaryKey"`
	ContestID             uint      `json:"contest_id" gorm:"not null"`
	PlayerID              uint      `json:"player_id" gorm:"not null"`
	LeverageType          string    `json:"leverage_type" gorm:"size:50;not null"` // 'contrarian', 'stack', 'pivot'
	OpportunityScore      float64   `json:"opportunity_score" gorm:"not null"`
	OwnershipDifferential *float64  `json:"ownership_differential"`
	ValueRating           *float64  `json:"value_rating"`
	RiskRating            *float64  `json:"risk_rating"`
	Reasoning             *string   `json:"reasoning" gorm:"type:text"`
	ExpiresAt             time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt             time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// PromptTemplate represents dynamic AI prompt templates
type PromptTemplate struct {
	ID          uint            `json:"id" gorm:"primaryKey"`
	Name        string          `json:"name" gorm:"size:100;not null;uniqueIndex"`
	Sport       string          `json:"sport" gorm:"size:50;not null"`
	ContestType *string         `json:"contest_type" gorm:"size:50"` // 'gpp', 'cash', 'satellite'
	Template    string          `json:"template" gorm:"type:text;not null"`
	Variables   json.RawMessage `json:"variables" gorm:"type:jsonb"` // Template variables and their descriptions
	Version     int             `json:"version" gorm:"default:1"`
	IsActive    bool            `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time       `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// ModelPerformance tracks AI model effectiveness
type ModelPerformance struct {
	ID                       uint      `json:"id" gorm:"primaryKey"`
	ModelName                string    `json:"model_name" gorm:"size:50;not null"`
	Sport                    string    `json:"sport" gorm:"size:50;not null"`
	ContestType              *string   `json:"contest_type" gorm:"size:50"`
	TotalRecommendations     int       `json:"total_recommendations" gorm:"default:0"`
	SuccessfulRecommendations int      `json:"successful_recommendations" gorm:"default:0"`
	AverageConfidence        *float64  `json:"average_confidence"`
	AverageResponseTimeMs    *int      `json:"average_response_time_ms"`
	AverageROI               *float64  `json:"average_roi"`
	LastUpdated              time.Time `json:"last_updated" gorm:"default:CURRENT_TIMESTAMP"`
}

// UserAIPreferences stores user personalization settings
type UserAIPreferences struct {
	ID                         uint            `json:"id" gorm:"primaryKey"`
	UserID                     uint            `json:"user_id" gorm:"not null;uniqueIndex"`
	RiskTolerance              string          `json:"risk_tolerance" gorm:"size:20;default:'medium'"` // 'conservative', 'medium', 'aggressive'
	OwnershipStrategy          string          `json:"ownership_strategy" gorm:"size:50;default:'balanced'"` // 'contrarian', 'balanced', 'chalk'
	OptimizationGoal           string          `json:"optimization_goal" gorm:"size:50;default:'roi'"` // 'roi', 'ceiling', 'floor', 'balanced'
	IncludeRealtimeData        bool            `json:"include_realtime_data" gorm:"default:true"`
	MaxRecommendationFrequency int             `json:"max_recommendation_frequency" gorm:"default:5"` // per hour
	PreferredModels            json.RawMessage `json:"preferred_models" gorm:"type:jsonb;default:'[]'"`
	BlacklistedPlayers         json.RawMessage `json:"blacklisted_players" gorm:"type:jsonb;default:'[]'"`
	CreatedAt                  time.Time       `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt                  time.Time       `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// OwnershipInsight represents processed ownership intelligence
type OwnershipInsight struct {
	PlayerID           uint                   `json:"player_id"`
	CurrentOwnership   float64                `json:"current_ownership"`
	ProjectedOwnership float64                `json:"projected_ownership"`
	OwnershipTrend     string                 `json:"ownership_trend"` // "rising", "falling", "stable"
	LeverageScore      float64                `json:"leverage_score"`
	ChalkFactor        float64                `json:"chalk_factor"`
	StackOwnership     map[string]float64     `json:"stack_ownership"`
	ConfidenceInterval float64                `json:"confidence_interval"`
}

// RecommendationUpdate represents real-time updates sent via WebSocket
type RecommendationUpdate struct {
	Type         string      `json:"type"`         // "insight", "ownership_alert", "late_swap"
	UserID       string      `json:"user_id"`
	PlayerID     uint        `json:"player_id"`
	Confidence   float64     `json:"confidence"`
	Message      string      `json:"message"`
	Data         interface{} `json:"data"`
	Timestamp    time.Time   `json:"timestamp"`
}