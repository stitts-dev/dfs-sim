package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
)

// EventType represents different types of real-time events
type EventType string

const (
	EventTypePlayerInjury     EventType = "player_injury"
	EventTypePlayerStatus     EventType = "player_status"
	EventTypeWeatherUpdate    EventType = "weather_update"
	EventTypeOwnershipChange  EventType = "ownership_change"
	EventTypeContestUpdate    EventType = "contest_update"
	EventTypeLineupChange     EventType = "lineup_change"
	EventTypePriceChange      EventType = "price_change"
	EventTypeNewsUpdate       EventType = "news_update"
)

// DeliveryChannel represents alert delivery methods
type DeliveryChannel string

const (
	DeliveryChannelWebSocket DeliveryChannel = "websocket"
	DeliveryChannelEmail     DeliveryChannel = "email"
	DeliveryChannelPush      DeliveryChannel = "push"
	DeliveryChannelSMS       DeliveryChannel = "sms"
)

// SwapAction represents user response to late swap recommendations
type SwapAction string

const (
	SwapActionPending  SwapAction = "pending"
	SwapActionAccepted SwapAction = "accepted"
	SwapActionRejected SwapAction = "rejected"
)

// RealTimeEvent represents a real-time event in the system
type RealTimeEvent struct {
	EventID        uuid.UUID      `json:"event_id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	EventType      EventType      `json:"event_type" gorm:"index:idx_event_type;size:50;not null"`
	PlayerID       *uint          `json:"player_id,omitempty" gorm:"index:idx_player_events"`
	GameID         *string        `json:"game_id,omitempty" gorm:"size:100"`
	TournamentID   *string        `json:"tournament_id,omitempty" gorm:"size:100"`
	Timestamp      time.Time      `json:"timestamp" gorm:"index:idx_timestamp;default:CURRENT_TIMESTAMP"`
	Source         string         `json:"source" gorm:"index:idx_source;size:50;not null"`
	Data           datatypes.JSON `json:"data" gorm:"type:jsonb;not null"`
	ImpactRating   float64        `json:"impact_rating" gorm:"default:0.0"`    // -10 to +10 DFS impact
	Confidence     float64        `json:"confidence" gorm:"default:1.0"`       // 0-1 data reliability
	ExpirationTime *time.Time     `json:"expiration_time,omitempty"`
	ProcessedAt    *time.Time     `json:"processed_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// TableName returns the table name for RealTimeEvent
func (RealTimeEvent) TableName() string {
	return "realtime_events"
}

// IsActive returns true if the event hasn't expired
func (rte *RealTimeEvent) IsActive() bool {
	if rte.ExpirationTime == nil {
		return true
	}
	return time.Now().Before(*rte.ExpirationTime)
}

// IsHighImpact returns true if the event has significant DFS impact
func (rte *RealTimeEvent) IsHighImpact() bool {
	return rte.ImpactRating >= 7.0 || rte.ImpactRating <= -7.0
}

// OwnershipSnapshot represents ownership data at a point in time
type OwnershipSnapshot struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	ContestID       string         `json:"contest_id" gorm:"index:idx_contest_time;size:100;not null"`
	Timestamp       time.Time      `json:"timestamp" gorm:"index:idx_contest_time;default:CURRENT_TIMESTAMP"`
	PlayerOwnership datatypes.JSON `json:"player_ownership" gorm:"type:jsonb;not null"` // map[uint]float64
	StackOwnership  datatypes.JSON `json:"stack_ownership" gorm:"type:jsonb"`          // map[string]float64
	TotalEntries    int            `json:"total_entries" gorm:"default:0"`
	TimeToLock      *time.Duration `json:"time_to_lock,omitempty"`
	CreatedAt       time.Time      `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// TableName returns the table name for OwnershipSnapshot
func (OwnershipSnapshot) TableName() string {
	return "ownership_snapshots"
}

// GetPlayerOwnership returns ownership percentage for a specific player
func (os *OwnershipSnapshot) GetPlayerOwnership(playerID uint) float64 {
	var ownership map[string]float64
	if err := json.Unmarshal(os.PlayerOwnership, &ownership); err != nil {
		return 0.0
	}
	
	playerIDStr := string(rune(playerID))
	return ownership[playerIDStr]
}

// AlertRule represents user-specific alert configuration
type AlertRule struct {
	ID               uint                    `json:"id" gorm:"primaryKey"`
	UserID           int                     `json:"user_id" gorm:"index:idx_user_alerts;not null"`
	RuleID           string                  `json:"rule_id" gorm:"uniqueIndex;size:100;not null"`
	EventTypes       pq.StringArray          `json:"event_types" gorm:"type:text[];not null"`
	ImpactThreshold  float64                 `json:"impact_threshold" gorm:"default:0.0"`
	Sports           pq.StringArray          `json:"sports" gorm:"type:text[];default:ARRAY['golf']"`
	DeliveryChannels pq.StringArray          `json:"delivery_channels" gorm:"type:text[];default:ARRAY['websocket']"`
	IsActive         bool                    `json:"is_active" gorm:"default:true"`
	CreatedAt        time.Time               `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt        time.Time               `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// TableName returns the table name for AlertRule
func (AlertRule) TableName() string {
	return "alert_rules"
}

// ShouldAlert determines if an event should trigger this alert rule
func (ar *AlertRule) ShouldAlert(event *RealTimeEvent) bool {
	if !ar.IsActive {
		return false
	}
	
	// Check impact threshold
	if event.ImpactRating < ar.ImpactThreshold {
		return false
	}
	
	// Check event type
	eventTypeStr := string(event.EventType)
	for _, allowedType := range ar.EventTypes {
		if allowedType == eventTypeStr {
			return true
		}
	}
	
	return false
}

// EventLog represents audit trail for event processing
type EventLog struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	EventID   uuid.UUID      `json:"event_id" gorm:"index:idx_event_log;type:uuid"`
	Action    string         `json:"action" gorm:"index:idx_action_time;size:50;not null"`
	Details   datatypes.JSON `json:"details,omitempty" gorm:"type:jsonb"`
	UserID    *int           `json:"user_id,omitempty"`
	CreatedAt time.Time      `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// TableName returns the table name for EventLog
func (EventLog) TableName() string {
	return "event_log"
}

// LateSwapRecommendation represents intelligent late swap suggestions
type LateSwapRecommendation struct {
	ID                    uint       `json:"id" gorm:"primaryKey"`
	UserID                int        `json:"user_id" gorm:"index:idx_user_contest;not null"`
	ContestID             string     `json:"contest_id" gorm:"index:idx_user_contest;size:100;not null"`
	OriginalPlayerID      uint       `json:"original_player_id" gorm:"not null"`
	RecommendedPlayerID   uint       `json:"recommended_player_id" gorm:"not null"`
	SwapReason            string     `json:"swap_reason" gorm:"size:255;not null"`
	ImpactScore           float64    `json:"impact_score" gorm:"not null"`
	ConfidenceScore       float64    `json:"confidence_score" gorm:"not null"`
	AutoApproved          bool       `json:"auto_approved" gorm:"default:false"`
	UserAction            *SwapAction `json:"user_action,omitempty" gorm:"size:20"`
	CreatedAt             time.Time  `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	ExpiresAt             time.Time  `json:"expires_at" gorm:"index:idx_active;not null"`
}

// TableName returns the table name for LateSwapRecommendation
func (LateSwapRecommendation) TableName() string {
	return "late_swap_recommendations"
}

// IsActive returns true if the recommendation hasn't expired and is pending
func (lsr *LateSwapRecommendation) IsActive() bool {
	return time.Now().Before(lsr.ExpiresAt) && lsr.UserAction == nil
}

// IsHighConfidence returns true if the recommendation has high confidence
func (lsr *LateSwapRecommendation) IsHighConfidence() bool {
	return lsr.ConfidenceScore >= 0.8
}

// WebSocketMessage represents real-time messages sent via WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	EventType EventType   `json:"event_type,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	UserID    *int        `json:"user_id,omitempty"`
}

// SubscriptionRequest represents WebSocket subscription requests
type SubscriptionRequest struct {
	Type      string      `json:"type"`
	DataTypes []EventType `json:"data_types"`
	Sports    []string    `json:"sports,omitempty"`
}

// DataSubscription represents user's real-time data subscription
type DataSubscription struct {
	UserID    string      `json:"user_id"`
	DataTypes []EventType `json:"data_types"`
	Sports    []string    `json:"sports"`
	Channel   chan RealTimeEvent `json:"-"`
	CreatedAt time.Time   `json:"created_at"`
	LastSeen  time.Time   `json:"last_seen"`
}

// OwnershipTrend represents ownership change trends
type OwnershipTrend struct {
	PlayerID         uint    `json:"player_id"`
	ContestID        string  `json:"contest_id"`
	CurrentOwnership float64 `json:"current_ownership"`
	PrevOwnership    float64 `json:"prev_ownership"`
	OwnershipChange  float64 `json:"ownership_change"`
	Velocity         float64 `json:"velocity"` // Rate of change per hour
	TrendDirection   string  `json:"trend_direction"` // "up", "down", "stable"
}

// Alert represents a generated alert for delivery
type Alert struct {
	ID          uuid.UUID       `json:"id"`
	UserID      int             `json:"user_id"`
	RuleID      string          `json:"rule_id"`
	EventID     uuid.UUID       `json:"event_id"`
	Title       string          `json:"title"`
	Message     string          `json:"message"`
	Priority    string          `json:"priority"` // "low", "medium", "high", "critical"
	Channels    []DeliveryChannel `json:"channels"`
	CreatedAt   time.Time       `json:"created_at"`
	DeliveredAt *time.Time      `json:"delivered_at,omitempty"`
}