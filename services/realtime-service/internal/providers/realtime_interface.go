package providers

import (
	"context"
	"time"

	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/models"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// RealTimeProvider defines the interface for real-time data providers
type RealTimeProvider interface {
	// Basic provider info
	GetProviderName() string
	GetSupportedSports() []string
	IsHealthy() bool

	// Real-time subscription management
	Subscribe(ctx context.Context, subscription *Subscription) (<-chan models.RealTimeEvent, error)
	Unsubscribe(ctx context.Context, subscriptionID string) error
	GetActiveSubscriptions() []Subscription

	// Event streaming
	StreamEvents(ctx context.Context, eventTypes []models.EventType) (<-chan models.RealTimeEvent, error)
	StreamOwnership(ctx context.Context, contestIDs []string) (<-chan models.OwnershipSnapshot, error)

	// Health and monitoring
	GetConnectionStatus() ConnectionStatus
	GetMetrics() ProviderMetrics
}

// DataProvider extends the existing provider interface with real-time capabilities
type DataProvider interface {
	// Static data methods (from existing GolfProvider interface)
	GetPlayers(sport types.Sport, date string) ([]types.PlayerData, error)
	GetCurrentTournament() (*GolfTournamentData, error)
	GetTournamentSchedule() ([]GolfTournamentData, error)

	// Real-time capabilities
	RealTimeProvider
}

// Subscription represents a real-time data subscription
type Subscription struct {
	ID             string                `json:"id"`
	UserID         string                `json:"user_id"`
	EventTypes     []models.EventType    `json:"event_types"`
	Sports         []string              `json:"sports"`
	ContestIDs     []string              `json:"contest_ids,omitempty"`
	PlayerIDs      []uint                `json:"player_ids,omitempty"`
	ImpactFilter   float64               `json:"impact_filter"` // Minimum impact rating
	CreatedAt      time.Time             `json:"created_at"`
	LastActivity   time.Time             `json:"last_activity"`
	IsActive       bool                  `json:"is_active"`
}

// ConnectionStatus represents the health of a real-time provider connection
type ConnectionStatus struct {
	IsConnected       bool      `json:"is_connected"`
	LastConnected     time.Time `json:"last_connected"`
	ConnectionUptime  time.Duration `json:"connection_uptime"`
	ReconnectAttempts int       `json:"reconnect_attempts"`
	LatencyMs         float64   `json:"latency_ms"`
	ErrorRate         float64   `json:"error_rate"`
}

// ProviderMetrics contains performance and usage metrics
type ProviderMetrics struct {
	EventsReceived     int64     `json:"events_received"`
	EventsProcessed    int64     `json:"events_processed"`
	EventsFiltered     int64     `json:"events_filtered"`
	EventsErrored      int64     `json:"events_errored"`
	SubscriptionsActive int      `json:"subscriptions_active"`
	DataLatencyMs      float64   `json:"data_latency_ms"`
	ThroughputPerSec   float64   `json:"throughput_per_sec"`
	LastEventTime      time.Time `json:"last_event_time"`
	UptimePercentage   float64   `json:"uptime_percentage"`
}

// GolfTournamentData represents golf tournament data (maintaining compatibility)
type GolfTournamentData struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Course      string    `json:"course"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	Status      string    `json:"status"`
	PrizePool   float64   `json:"prize_pool"`
	PlayerCount int       `json:"player_count"`
}

// RealTimeConfig represents configuration for real-time providers
type RealTimeConfig struct {
	// Connection settings
	MaxConnections     int           `json:"max_connections"`
	ConnectionTimeout  time.Duration `json:"connection_timeout"`
	ReconnectInterval  time.Duration `json:"reconnect_interval"`
	MaxReconnectAttempts int         `json:"max_reconnect_attempts"`

	// Rate limiting
	EventsPerSecond    float64       `json:"events_per_second"`
	BurstSize          int           `json:"burst_size"`
	RateLimitWindow    time.Duration `json:"rate_limit_window"`

	// Circuit breaker
	FailureThreshold   int           `json:"failure_threshold"`
	RecoveryTimeout    time.Duration `json:"recovery_timeout"`
	MaxRetries         int           `json:"max_retries"`

	// Data processing
	EventBufferSize    int           `json:"event_buffer_size"`
	ProcessingTimeout  time.Duration `json:"processing_timeout"`
	EnableFiltering    bool          `json:"enable_filtering"`
	MinImpactRating    float64       `json:"min_impact_rating"`
}

// EventFilter defines criteria for filtering real-time events
type EventFilter struct {
	EventTypes      []models.EventType `json:"event_types"`
	Sports          []string           `json:"sports"`
	MinImpactRating float64            `json:"min_impact_rating"`
	MaxAge          time.Duration      `json:"max_age"`
	PlayerIDs       []uint             `json:"player_ids,omitempty"`
	ContestIDs      []string           `json:"contest_ids,omitempty"`
}

// ProviderError represents errors from real-time providers
type ProviderError struct {
	Provider    string    `json:"provider"`
	ErrorType   string    `json:"error_type"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	IsRetryable bool      `json:"is_retryable"`
}

func (pe *ProviderError) Error() string {
	return pe.Message
}

// ProviderStatus represents the overall status of all providers
type ProviderStatus struct {
	TotalProviders   int               `json:"total_providers"`
	HealthyProviders int               `json:"healthy_providers"`
	FailedProviders  []string          `json:"failed_providers"`
	LastUpdate       time.Time         `json:"last_update"`
	Providers        map[string]ConnectionStatus `json:"providers"`
}