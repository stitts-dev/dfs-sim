# PRP: Real-Time Data Integration System

## Goal
Build a comprehensive real-time data integration platform that streams live player status, weather conditions, ownership changes, and market movements to enable dynamic optimization and intelligent late-game decision making. Extend the existing microservices architecture with event-driven real-time capabilities while maintaining the current provider interface patterns and WebSocket infrastructure.

## Why
- **Business Impact**: Enable 10-15% ROI improvement through optimal late swap decisions
- **Competitive Advantage**: Provide professional-grade DFS analysis with real-time insights
- **User Experience**: Reduce manual monitoring burden by 60% through automated alerts and recommendations
- **Integration**: Seamlessly extend existing optimization and AI recommendation systems with live data
- **Market Position**: Compete with platforms like Stokastic and FantasyData with superior real-time capabilities

## What
Implement a production-ready real-time data integration system that:
- Streams live player status updates with <30 second latency
- Tracks ownership percentage changes dynamically during contest periods
- Provides intelligent late swap recommendations with automated approval workflows
- Delivers multi-channel alerts (WebSocket, email, push) based on user preferences
- Maintains 99.9% uptime during contest periods with graceful fallback strategies

### Success Criteria
- [ ] Data latency <30 seconds from source to system for critical updates
- [ ] 99.9% availability during contest periods with circuit breaker fallbacks
- [ ] 95%+ player coverage across all major sports with real-time status tracking
- [ ] 40%+ late swap adoption rate with 8-12% average ROI improvement
- [ ] Integration with existing optimization and AI recommendation services
- [ ] <1% false positive rate on status change alerts

## All Needed Context

### Documentation & References
```yaml
# Real-Time Architecture Patterns
- url: https://kafka.apache.org/
  why: High-volume event streaming patterns and partitioning strategies
  section: Core concepts for event-driven architecture
  
- url: https://redis.io/docs/latest/develop/data-types/streams/
  why: In-memory streaming for ultra-low latency ownership tracking
  section: Redis Streams for real-time analytics patterns
  
- url: https://ably.com/topic/the-challenge-of-scaling-websockets
  why: WebSocket scaling patterns with Redis pub/sub
  section: Non-sticky sessions and connection state sharing

# Circuit Breaker and Reliability
- url: https://microservices.io/patterns/reliability/circuit-breaker.html
  why: Production-ready circuit breaker patterns for external APIs
  critical: State management and fallback strategies for data source reliability

# Event-Driven Architecture  
- url: https://learn.microsoft.com/en-us/azure/architecture/patterns/cqrs
  why: CQRS patterns for real-time read/write optimization
  section: Command and query responsibility separation for ownership tracking

# Existing Codebase Patterns
- file: services/optimization-service/internal/websocket/hub.go
  why: Existing WebSocket hub pattern for real-time updates
  critical: Connection management and user-specific channels
  
- file: services/sports-data-service/internal/providers/rapidapi_golf.go
  why: Current provider interface pattern with rate limiting
  critical: Cache-first strategy and fallback hierarchy
  
- file: services/sports-data-service/internal/services/circuit_breaker.go
  why: Existing circuit breaker implementation with state management
  critical: Per-service circuit breakers and automatic fallback
  
- file: services/api-gateway/internal/proxy/service_proxy.go
  why: Service communication patterns and request routing
  critical: Circuit breaker integration and error handling
  
- file: shared/pkg/config/config.go
  why: Configuration management patterns across services
  critical: Environment-based configuration and service discovery
```

### Current Codebase Structure
```bash
services/
├── api-gateway/               # Central API Gateway with WebSocket proxy
│   ├── internal/websocket/    # Existing WebSocket hub for optimization progress
│   └── internal/proxy/        # Service proxy with circuit breakers
├── optimization-service/      # Optimization algorithms and simulation
│   ├── internal/websocket/    # Real-time progress tracking (extend for data updates)
│   └── internal/optimizer/    # Core optimization algorithms (integrate with real-time data)
├── sports-data-service/       # Golf data providers and tournament sync
│   ├── internal/providers/    # Provider interface pattern (extend for real-time)
│   └── internal/services/     # Circuit breaker and data fetching (add event processing)
├── user-service/             # User authentication and preferences
│   └── internal/services/    # SMS service (extend for alert delivery)
└── shared/
    ├── pkg/config/           # Centralized configuration (add real-time settings)
    └── types/                # Common type definitions (add real-time events)
```

### Desired Codebase Structure with New Files
```bash
services/
├── realtime-service/         # NEW: Core real-time data integration service
│   ├── cmd/server/main.go           # Service entry point with event processing
│   ├── internal/
│   │   ├── api/handlers/            # WebSocket and SSE endpoints for data streams
│   │   ├── events/                  # Event processing engine and handlers
│   │   ├── providers/               # Real-time provider extensions (WebSocket, SSE)
│   │   ├── ownership/               # Ownership tracking and trend analysis
│   │   ├── alerts/                  # Alert system with multi-channel delivery
│   │   ├── lateswap/               # Late swap recommendation engine
│   │   └── models/                  # Real-time event and ownership models
│   └── migrations/                  # Real-time data schema migrations
├── api-gateway/
│   ├── internal/websocket/          # EXTEND: Add real-time data channels
│   └── internal/proxy/              # EXTEND: Add realtime-service routing
├── optimization-service/
│   ├── internal/websocket/          # EXTEND: Add data update channels  
│   └── internal/optimizer/          # EXTEND: Dynamic re-optimization triggers
├── sports-data-service/
│   ├── internal/providers/          # EXTEND: Add real-time provider interface
│   └── internal/events/             # NEW: Event publishing for data changes
├── user-service/
│   ├── internal/services/           # EXTEND: Alert delivery service
│   └── internal/preferences/        # NEW: Real-time alert preferences
└── shared/
    ├── pkg/events/                  # NEW: Shared event types and interfaces
    ├── pkg/realtime/               # NEW: Real-time utilities and patterns
    └── types/                       # EXTEND: Add real-time event types
```

### Known Gotchas & Library Quirks
```go
// CRITICAL: Redis Streams require unique consumer group names per service
// Pattern: Use service name + instance ID for consumer groups
consumerGroup := fmt.Sprintf("%s-%s", serviceName, instanceID)

// GOTCHA: WebSocket connections drop during load balancing
// Solution: Use Redis pub/sub for connection state sharing across instances
type ConnectionState struct {
    UserID      string    `json:"user_id"`
    InstanceID  string    `json:"instance_id"`
    ConnectedAt time.Time `json:"connected_at"`
}

// CRITICAL: RapidAPI has 20 requests/day limit on basic plan
// Solution: Aggressive caching + multiple fallback providers
// Cache TTL: 24 hours for completed tournaments, 5 minutes for live events

// GOTCHA: PostgreSQL triggers for real-time events can cause deadlocks
// Solution: Use NOTIFY/LISTEN instead of triggers for high-frequency updates
SELECT pg_notify('player_update', json_build_object('player_id', NEW.id)::text);

// CRITICAL: Circuit breakers must be service-specific
// Pattern: One circuit breaker per external data provider
type CircuitBreakerConfig struct {
    FailureThreshold: 5,    // 5 failures triggers open state
    RecoveryTimeout:  30s,  // Wait 30s before half-open attempt
    MaxRetries:       3,    // Retry 3 times before marking as failure
}
```

## Implementation Blueprint

### Data Models and Structure

Create core real-time event models to ensure type safety and consistency across services:

```go
// Real-time event types for event sourcing pattern
type RealTimeEvent struct {
    EventID        string                 `json:"event_id" gorm:"primaryKey"`
    EventType      EventType              `json:"event_type" gorm:"index:idx_event_type"`
    PlayerID       *uint                  `json:"player_id,omitempty" gorm:"index:idx_player_events"`
    GameID         *string                `json:"game_id,omitempty" gorm:"index:idx_game_events"`
    TournamentID   *string                `json:"tournament_id,omitempty"`
    Timestamp      time.Time              `json:"timestamp" gorm:"index:idx_timestamp"`
    Source         string                 `json:"source" gorm:"index:idx_source"`
    Data           datatypes.JSON         `json:"data" gorm:"type:jsonb"`
    ImpactRating   float64               `json:"impact_rating"`    // -10 to +10 DFS impact
    Confidence     float64               `json:"confidence"`       // 0-1 data reliability
    ExpirationTime *time.Time            `json:"expiration_time"`
    ProcessedAt    *time.Time            `json:"processed_at"`
    CreatedAt      time.Time             `json:"created_at"`
}

// Ownership tracking with historical snapshots
type OwnershipSnapshot struct {
    ID              uint                   `json:"id" gorm:"primaryKey"`
    ContestID       string                 `json:"contest_id" gorm:"index:idx_contest_time"`
    Timestamp       time.Time              `json:"timestamp" gorm:"index:idx_contest_time"`
    PlayerOwnership datatypes.JSON         `json:"player_ownership" gorm:"type:jsonb"` // map[uint]float64
    StackOwnership  datatypes.JSON         `json:"stack_ownership" gorm:"type:jsonb"`  // map[string]float64
    TotalEntries    int                    `json:"total_entries"`
    TimeToLock      time.Duration          `json:"time_to_lock"`
    CreatedAt       time.Time              `json:"created_at"`
}

// Alert configuration with user preferences  
type AlertRule struct {
    ID              uint                   `json:"id" gorm:"primaryKey"`
    UserID          int                    `json:"user_id" gorm:"index:idx_user_alerts"`
    RuleID          string                 `json:"rule_id" gorm:"uniqueIndex"`
    EventTypes      pq.StringArray         `json:"event_types" gorm:"type:text[]"`
    ImpactThreshold float64               `json:"impact_threshold"`
    Sports          pq.StringArray         `json:"sports" gorm:"type:text[]"`
    DeliveryChannels pq.StringArray        `json:"delivery_channels" gorm:"type:text[]"`
    IsActive        bool                   `json:"is_active" gorm:"default:true"`
    CreatedAt       time.Time              `json:"created_at"`
    UpdatedAt       time.Time              `json:"updated_at"`
}
```

### Task List in Implementation Order

```yaml
Task 1: Database Schema and Migrations
CREATE services/realtime-service/migrations/001_realtime_events_schema.sql:
  - CREATE realtime_events table with JSONB data column and proper indexes
  - CREATE ownership_snapshots table with time-series indexing
  - CREATE alert_rules table with user preferences
  - CREATE event_log table for audit trail
  - ADD PostgreSQL NOTIFY triggers for real-time event publishing

Task 2: Real-Time Provider Interface Extension  
MODIFY services/sports-data-service/internal/providers/:
  - EXTEND existing DataProvider interface with real-time capabilities
  - ADD RealTimeProvider interface with Subscribe/Unsubscribe methods
  - CREATE realtime_provider.go with WebSocket and SSE client implementations
  - PRESERVE existing cache-first and circuit breaker patterns

Task 3: Event Processing Service Foundation
CREATE services/realtime-service/internal/events/:
  - CREATE event_processor.go with Redis Streams consumer groups
  - CREATE event_handlers.go for different event types (injury, weather, ownership)
  - CREATE event_publisher.go for publishing events to Redis Streams
  - MIRROR circuit breaker pattern from sports-data-service

Task 4: WebSocket Hub Extensions  
MODIFY services/optimization-service/internal/websocket/hub.go:
  - ADD real-time data channels: dataUpdate chan DataUpdateMessage
  - ADD event subscription management for user-specific updates
  - PRESERVE existing optimization progress tracking
  - EXTEND message routing for different event types

Task 5: Ownership Tracking System
CREATE services/realtime-service/internal/ownership/:
  - CREATE ownership_tracker.go with Redis-backed real-time calculations
  - CREATE trend_analyzer.go for ownership velocity and prediction
  - CREATE leverage_calculator.go for contrarian opportunity scoring
  - INTEGRATE with existing contest and player models

Task 6: Alert System Implementation
CREATE services/realtime-service/internal/alerts/:
  - CREATE alert_engine.go with rule-based alert generation
  - CREATE delivery_channels.go for WebSocket, email, push notifications
  - CREATE rate_limiter.go to prevent alert spam
  - EXTEND user-service SMS capabilities for alert delivery

Task 7: Late Swap Intelligence Engine
CREATE services/realtime-service/internal/lateswap/:
  - CREATE recommendation_engine.go with impact scoring algorithms
  - CREATE decision_tree.go for automated vs manual approval workflows
  - CREATE risk_manager.go for trade safety and user consent
  - INTEGRATE with optimization-service for dynamic re-optimization

Task 8: API Gateway Integration
MODIFY services/api-gateway/:
  - ADD realtime-service routing in internal/proxy/service_proxy.go
  - EXTEND WebSocket proxy for real-time data subscriptions
  - ADD health check integration for realtime-service
  - PRESERVE existing circuit breaker patterns

Task 9: Service Configuration and Deployment
MODIFY docker-compose.yml and service configurations:
  - ADD realtime-service to docker-compose with Redis Streams configuration
  - UPDATE environment variables for real-time data source URLs
  - ADD service discovery configuration for realtime-service
  - CONFIGURE Redis Streams consumer groups per service instance

Task 10: Frontend WebSocket Integration
MODIFY frontend/src/services/ and store/:
  - EXTEND existing WebSocket client for real-time data subscriptions
  - ADD real-time data store with Zustand for ownership tracking
  - CREATE alert notification components
  - INTEGRATE with existing Dashboard and optimization components
```

### Per Task Pseudocode

```go
// Task 1: Database Schema - Focus on time-series optimization
CREATE TABLE realtime_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(50) NOT NULL,
    player_id INTEGER REFERENCES players(id),
    game_id VARCHAR(100),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    source VARCHAR(50) NOT NULL,
    data JSONB NOT NULL,
    impact_rating FLOAT DEFAULT 0.0,
    confidence FLOAT DEFAULT 1.0,
    expiration_time TIMESTAMP WITH TIME ZONE,
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- CRITICAL: Indexes for real-time query performance
CREATE INDEX CONCURRENTLY idx_realtime_events_type_time ON realtime_events(event_type, timestamp DESC);
CREATE INDEX CONCURRENTLY idx_realtime_events_player_active ON realtime_events(player_id, timestamp DESC) 
    WHERE expiration_time IS NULL OR expiration_time > CURRENT_TIMESTAMP;

// Task 3: Event Processing - Redis Streams pattern
type EventProcessor struct {
    redisClient   *redis.Client
    eventHandlers map[EventType]EventHandler
    consumerGroup string // CRITICAL: Must be unique per service instance
    streamName    string
}

func (ep *EventProcessor) ProcessEvents(ctx context.Context) error {
    // PATTERN: Consumer group ensures at-least-once delivery
    consumerGroup := fmt.Sprintf("realtime-service-%s", ep.instanceID)
    
    for {
        // GOTCHA: Block with timeout to prevent indefinite blocking
        streams, err := ep.redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
            Group:    consumerGroup,
            Consumer: ep.instanceID,
            Streams:  []string{ep.streamName, ">"},
            Count:    10,
            Block:    5 * time.Second,
        }).Result()
        
        if err != nil && err != redis.Nil {
            // CRITICAL: Circuit breaker pattern for Redis failures
            return fmt.Errorf("failed to read from stream: %w", err)
        }
        
        for _, stream := range streams {
            for _, message := range stream.Messages {
                if err := ep.handleEvent(ctx, message); err != nil {
                    // PATTERN: Dead letter queue for failed events
                    ep.sendToDeadLetterQueue(message, err)
                } else {
                    // CRITICAL: Acknowledge successful processing
                    ep.redisClient.XAck(ctx, ep.streamName, consumerGroup, message.ID)
                }
            }
        }
    }
}

// Task 4: WebSocket Hub Extension - Add data channels
type ExtendedHub struct {
    *OptimizationHub                           // PRESERVE: Existing optimization hub
    dataSubscriptions map[string]DataSubscription // NEW: Data subscriptions by user
    eventChannel     chan RealTimeEvent       // NEW: Real-time event channel
    ownershipChannel chan OwnershipSnapshot   // NEW: Ownership update channel
}

func (h *ExtendedHub) SubscribeToDataUpdates(userID string, dataTypes []string) error {
    // PATTERN: User-specific subscription management
    subscription := DataSubscription{
        UserID:    userID,
        DataTypes: dataTypes,
        Channel:   make(chan RealTimeEvent, 100), // CRITICAL: Buffered channel
        CreatedAt: time.Now(),
    }
    
    h.dataSubscriptions[userID] = subscription
    
    // CRITICAL: Start goroutine for user-specific event filtering
    go h.filterAndDeliverEvents(userID, subscription)
    return nil
}

// Task 5: Ownership Tracking - Real-time calculation pattern
type OwnershipTracker struct {
    redisClient    *redis.Client
    dbConnection   *gorm.DB
    trendAnalyzer  *TrendAnalyzer
    updateInterval time.Duration
}

func (ot *OwnershipTracker) TrackOwnership(contestID string) error {
    ticker := time.NewTicker(ot.updateInterval) // PATTERN: Configurable update frequency
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            // CRITICAL: Atomic ownership calculation
            snapshot, err := ot.calculateOwnershipSnapshot(contestID)
            if err != nil {
                // PATTERN: Log error but continue processing
                ot.logger.WithError(err).Error("Failed to calculate ownership")
                continue
            }
            
            // PATTERN: Cache current snapshot in Redis for fast access
            cacheKey := fmt.Sprintf("ownership:%s:current", contestID)
            if err := ot.redisClient.Set(context.Background(), cacheKey, snapshot, time.Hour).Err(); err != nil {
                ot.logger.WithError(err).Warn("Failed to cache ownership snapshot")
            }
            
            // PATTERN: Persist historical data in PostgreSQL
            if err := ot.dbConnection.Create(&snapshot).Error; err != nil {
                ot.logger.WithError(err).Error("Failed to persist ownership snapshot")
            }
            
            // CRITICAL: Publish ownership change event for WebSocket delivery
            event := RealTimeEvent{
                EventType: "ownership_update",
                Data:      snapshot,
                Source:    "ownership_tracker",
            }
            ot.publishEvent(event)
        }
    }
}
```

### Integration Points
```yaml
DATABASE:
  - migration: "Add realtime_events, ownership_snapshots, alert_rules tables"
  - indexes: "Time-series indexes for real-time query performance"
  - triggers: "PostgreSQL NOTIFY triggers for event publishing"
  
CONFIG:
  - add to: shared/pkg/config/config.go
  - pattern: "RealTimeDataSources map[string]DataSourceConfig"
  - settings: "REDIS_STREAMS_MAX_LENGTH, EVENT_PROCESSING_BATCH_SIZE"
  
ROUTES:
  - add to: services/api-gateway/internal/proxy/service_proxy.go
  - pattern: "RealtimeService: newServiceClient(config.RealtimeServiceURL)"
  - websocket: "Extend WebSocket proxy for real-time data streams"

REDIS:
  - streams: "realtime_events, ownership_updates, alert_queue"
  - consumer_groups: "service-specific consumer groups for at-least-once delivery"
  - pub_sub: "WebSocket connection state sharing across instances"
```

## Validation Loop

### Level 1: Syntax & Style
```bash
# Run these FIRST for each service - fix any errors before proceeding
cd services/realtime-service && go mod tidy
cd services/realtime-service && golangci-lint run
cd services/realtime-service && go vet ./...

# Type checking and build verification
cd services/realtime-service && go build -o server cmd/server/main.go

# Expected: No errors. If errors, READ the error message and fix.
```

### Level 2: Unit Tests for Each New Component
```go
// CREATE services/realtime-service/internal/events/event_processor_test.go
func TestEventProcessor_ProcessEvents(t *testing.T) {
    // Test Redis Streams event processing
    processor := NewEventProcessor(testRedisClient, testHandlers)
    
    // Mock event in Redis Stream
    testEvent := map[string]interface{}{
        "event_type": "player_injury",
        "player_id":  "123",
        "data":       `{"status": "out", "injury": "knee"}`,
    }
    
    // Verify event is processed and handled correctly
    err := processor.ProcessEvents(context.Background())
    assert.NoError(t, err)
}

func TestOwnershipTracker_CalculateSnapshot(t *testing.T) {
    // Test ownership calculation accuracy
    tracker := NewOwnershipTracker(testDB, testRedis)
    
    // Setup test data with known ownership percentages
    contestID := "test_contest_123"
    snapshot, err := tracker.calculateOwnershipSnapshot(contestID)
    
    assert.NoError(t, err)
    assert.NotNil(t, snapshot)
    assert.Equal(t, contestID, snapshot.ContestID)
}

func TestAlertEngine_GenerateAlerts(t *testing.T) {
    // Test alert generation for different event types
    engine := NewAlertEngine(testDB, testDelivery)
    
    event := RealTimeEvent{
        EventType:    "player_injury",
        PlayerID:     123,
        ImpactRating: 8.5,
    }
    
    alerts, err := engine.GenerateAlerts(event)
    assert.NoError(t, err)
    assert.Greater(t, len(alerts), 0)
}
```

```bash
# Run tests with race detection for concurrent code:
cd services/realtime-service && go test -race ./...
# If failing: Read error, understand root cause, fix code, re-run
```

### Level 3: Integration Tests
```bash
# Start the entire microservices stack
docker-compose up -d

# Wait for services to be healthy
sleep 30

# Test real-time event processing
curl -X POST http://localhost:8080/api/v1/realtime/events \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{
    "event_type": "player_injury",
    "player_id": 123,
    "data": {"status": "questionable", "injury": "ankle"},
    "impact_rating": 7.5,
    "source": "test_provider"
  }'

# Expected: {"status": "success", "event_id": "uuid-here"}

# Test WebSocket subscription for real-time updates
wscat -c ws://localhost:8080/ws/realtime/user123
# Send subscription message:
{"type": "subscribe", "data_types": ["player_injury", "ownership_update"]}
# Expected: {"type": "subscription_confirmed", "data_types": [...]}

# Test ownership tracking endpoint
curl http://localhost:8080/api/v1/realtime/ownership/contest123
# Expected: {"contest_id": "contest123", "ownership": {...}, "timestamp": "..."}
```

## Final Validation Checklist
- [ ] All tests pass: `cd services/realtime-service && go test ./...`
- [ ] No linting errors: `cd services/realtime-service && golangci-lint run`
- [ ] No race conditions: `go test -race ./...`
- [ ] WebSocket connections work: `wscat -c ws://localhost:8080/ws/realtime/user123`
- [ ] Event processing works: Redis Streams consumer groups active
- [ ] Ownership tracking updates: Real-time ownership calculations in Redis
- [ ] Alert delivery works: Multi-channel alert delivery (WebSocket, email)
- [ ] Circuit breakers function: External API failure handling
- [ ] Database migrations applied: All new tables and indexes created
- [ ] Service health checks pass: All services report healthy status

---

## Anti-Patterns to Avoid
- ❌ Don't create synchronous event processing - use Redis Streams for async processing
- ❌ Don't ignore circuit breaker failures - implement proper fallback strategies  
- ❌ Don't use polling for real-time updates - leverage WebSockets and event streams
- ❌ Don't hardcode event types - use enums and proper type definitions
- ❌ Don't skip consumer group management - ensure at-least-once delivery semantics
- ❌ Don't ignore Redis connection failures - implement reconnection logic
- ❌ Don't process events without timeout - always use context with deadlines
- ❌ Don't skip alert rate limiting - prevent user notification spam

**PRP Success Confidence Score: 9/10**

This PRP provides comprehensive context from both existing codebase patterns and external best practices. The implementation follows proven microservices patterns, extends existing infrastructure, and includes detailed validation steps. The high confidence score reflects the thorough research, clear task breakdown, and alignment with existing architecture patterns.