# PRP: AI Recommendations Intelligence Upgrade

## Feature: AI-Powered DFS Recommendations with Real-Time Intelligence

### Context Summary
Transform the legacy AI recommendation system from `backend.deprecated/internal/services/ai_recommendations.go` into an intelligent microservice with real-time data integration, advanced ownership analysis, and sophisticated dynamic prompting. This service will provide professional-grade DFS insights that adapt dynamically to contest conditions and market movements.

### Research References
- **Claude API Documentation**: https://www.anthropic.com/api
- **Dynamic Context Injection**: https://palospublishing.com/dynamic-context-injection-for-rapid-llm-workflows/
- **SportsDataIO API**: https://sportsdata.io/fantasy-sports-api
- **Prompt Engineering Best Practices**: https://www.multimodal.dev/post/llm-prompting

## Implementation Blueprint

### 1. Service Architecture

#### New Microservice Structure
```
services/ai-recommendations-service/
├── cmd/server/main.go                    # Service entry point
├── internal/
│   ├── api/
│   │   └── handlers/
│   │       ├── recommendations.go        # AI recommendation endpoints
│   │       ├── analysis.go              # Lineup analysis endpoints
│   │       ├── ownership.go             # Ownership intelligence endpoints
│   │       └── health.go                # Health check
│   ├── models/
│   │   ├── recommendations.go           # Data models
│   │   └── context.go                   # Context structures
│   ├── services/
│   │   ├── ai_engine.go                 # Core AI orchestration
│   │   ├── claude_client.go             # Claude API integration
│   │   ├── prompt_builder.go            # Dynamic prompt engine
│   │   ├── realtime_aggregator.go       # Real-time data pipeline
│   │   ├── ownership_analyzer.go        # Ownership intelligence
│   │   ├── cache.go                     # Redis caching layer
│   │   └── circuit_breaker.go          # API resilience
│   └── websocket/
│       └── hub.go                       # Real-time recommendation updates
├── migrations/
│   └── 001_create_ai_recommendations.sql
└── go.mod
```

#### API Gateway Integration
Follow the pattern from `services/api-gateway/internal/proxy/service_proxy.go`:

```go
// Add to API Gateway configuration
type Config struct {
    // ... existing fields
    AIRecommendationsServiceURL string `mapstructure:"AI_RECOMMENDATIONS_SERVICE_URL"`
}

// Add proxy method
func (sp *ServiceProxy) ProxyAIRecommendationsRequest(c *gin.Context) {
    sp.proxyRequest(c, sp.aiRecommendationsClient, "ai-recommendations-service")
}

// Add circuit breaker
aiRecommendationsCB := gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "ai-recommendations-service",
    MaxRequests: 3,
    Interval:    60 * time.Second,
    Timeout:     15 * time.Second, // Allow more time for AI processing
})
```

### 2. Core AI Engine Implementation

#### Claude Integration with Latest Model
Reference the existing pattern but upgrade to Claude Sonnet 4:

```go
// internal/services/claude_client.go
type ClaudeClient struct {
    apiKey         string
    httpClient     *http.Client
    rateLimiter    *RateLimiter
    circuitBreaker *gobreaker.CircuitBreaker
    logger         *logrus.Logger
}

type ClaudeConfig struct {
    Model         string  // "claude-sonnet-4-20250514" or latest
    MaxTokens     int     // Dynamic based on complexity
    Temperature   float64 // Vary by recommendation type (0.3-0.7)
    APIEndpoint   string  // https://api.anthropic.com/v1/messages
    PromptCache   bool    // Enable prompt caching for cost savings
}

// Follow existing HTTP patterns from golf service RapidAPI client
func (c *ClaudeClient) SendMessage(ctx context.Context, prompt string, config ClaudeConfig) (*ClaudeResponse, error) {
    // Implement with rate limiting, circuit breaker, and retry logic
    // Reference: services/golf-service/internal/providers/rapidapi_golf.go
}
```

#### Dynamic Prompt Builder
Implement sophisticated context injection:

```go
// internal/services/prompt_builder.go
type PromptBuilder struct {
    templates      map[string]*PromptTemplate
    sportModifiers map[string]SportModifier
    cache          *CacheService
    logger         *logrus.Logger
}

type PromptContext struct {
    Sport              string
    ContestType        string
    OptimizationGoal   string
    RealTimeData       []RealTimeDataPoint
    UserProfile        *UserAnalytics
    ExistingLineups    []types.Lineup
    TimeToLock         time.Duration
    OwnershipStrategy  string
}

func (pb *PromptBuilder) BuildRecommendationPrompt(ctx PromptContext, players []Player) string {
    // 1. Select base template based on sport/contest
    // 2. Inject real-time data points
    // 3. Add user personalization
    // 4. Include ownership intelligence
    // 5. Format for optimal Claude performance
}
```

### 3. Real-Time Data Integration

#### Data Aggregator Service
Follow the pattern from `services/golf-service/internal/services/data_fetcher.go`:

```go
// internal/services/realtime_aggregator.go
type RealtimeAggregator struct {
    weatherService    *WeatherAPI
    injuryService     *InjuryAPI
    oddsService       *OddsAPI
    ownershipService  *OwnershipTracker
    newsService       *NewsAggregator
    cache             *CacheService
    logger            *logrus.Logger
}

type RealTimeDataPoint struct {
    PlayerID      uint
    DataType      string    // "injury", "weather", "ownership", "odds"
    Value         interface{}
    Confidence    float64   // 0-1 reliability score
    Timestamp     time.Time
    Source        string
    ImpactRating  float64   // -5 to +5 DFS impact
}

// Implement streaming updates using channels
func (ra *RealtimeAggregator) StreamUpdates(ctx context.Context, contestID uint) <-chan RealTimeDataPoint {
    updates := make(chan RealTimeDataPoint, 100)
    
    // Launch goroutines for each data source
    go ra.streamWeatherUpdates(ctx, contestID, updates)
    go ra.streamInjuryReports(ctx, contestID, updates)
    go ra.streamOwnershipChanges(ctx, contestID, updates)
    
    return updates
}
```

### 4. Advanced Ownership Analysis

#### Ownership Intelligence Engine
```go
// internal/services/ownership_analyzer.go
type OwnershipAnalyzer struct {
    liveTracker       *OwnershipTracker
    historicalDB      *database.DB
    patternAnalyzer   *PatternEngine
    leverageCalc      *LeverageCalculator
    cache             *CacheService
}

type OwnershipInsight struct {
    PlayerID           uint
    CurrentOwnership   float64
    ProjectedOwnership float64
    OwnershipTrend     string    // "rising", "falling", "stable"
    LeverageScore      float64   // Contrarian opportunity rating
    ChalkFactor        float64   // How "chalky" this play is
    StackOwnership     map[string]float64
    ConfidenceInterval float64
}

// Implement leverage detection algorithm
func (oa *OwnershipAnalyzer) CalculateLeverageOpportunities(
    players []Player, 
    contestType string,
    existingLineups []Lineup,
) []LeveragePlay {
    // 1. Analyze ownership projections vs actual value
    // 2. Identify contrarian stacks
    // 3. Calculate tournament leverage
    // 4. Consider user's portfolio exposure
}
```

### 5. WebSocket Real-Time Updates

Follow the pattern from `services/optimization-service/internal/websocket/hub.go`:

```go
// internal/websocket/hub.go
type RecommendationHub struct {
    clients    map[string]map[*Client]bool
    broadcast  chan *RecommendationUpdate
    register   chan *Client
    unregister chan *Client
    mutex      sync.RWMutex
}

type RecommendationUpdate struct {
    Type         string    // "insight", "ownership_alert", "late_swap"
    UserID       string
    PlayerID     uint
    Confidence   float64
    Message      string
    Data         interface{}
    Timestamp    time.Time
}

// Broadcast AI insights to specific users
func (h *RecommendationHub) BroadcastInsight(userID string, update *RecommendationUpdate) {
    h.mutex.RLock()
    defer h.mutex.RUnlock()
    
    if clients, ok := h.clients[userID]; ok {
        for client := range clients {
            select {
            case client.send <- update:
            default:
                close(client.send)
                delete(h.clients[userID], client)
            }
        }
    }
}
```

### 6. Caching Strategy

Implement tiered caching following `services/golf-service/internal/services/cache.go`:

```go
const (
    // Cache TTLs
    PlayerInsightTTL    = 6 * time.Hour    // During active games
    HistoricalAnalysisTTL = 7 * 24 * time.Hour
    ModelResponseTTL    = 24 * time.Hour
    OwnershipSnapshotTTL = 5 * time.Minute
)

// Cache key patterns
func buildCacheKey(elements ...string) string {
    return fmt.Sprintf("ai-recommendations:%s", strings.Join(elements, ":"))
}

// Example keys:
// ai-recommendations:player:12345:insight:gpp
// ai-recommendations:contest:98765:ownership:snapshot
// ai-recommendations:model:response:hash123abc
```

### 7. API Endpoints

#### Recommendation Endpoints
```go
// GET /api/v1/ai-recommendations/players
// Request smart player recommendations with real-time context
type SmartRecommendationRequest struct {
    ContestID            int                    `json:"contest_id"`
    Sport                string                 `json:"sport"`
    ContestType          string                 `json:"contest_type"`
    RemainingBudget      float64                `json:"remaining_budget"`
    CurrentLineup        []int                  `json:"current_lineup"`
    PositionsNeeded      []string               `json:"positions_needed"`
    OptimizeFor          string                 `json:"optimize_for"`
    IncludeRealTimeData  bool                   `json:"include_realtime"`
    OwnershipStrategy    string                 `json:"ownership_strategy"`
    ExistingLineupIDs    []string               `json:"existing_lineup_ids"`
    TimeToLock           string                 `json:"time_to_lock"`
    RiskTolerance        string                 `json:"risk_tolerance"`
}

// POST /api/v1/ai-recommendations/analyze
// Analyze existing lineup with AI insights

// GET /api/v1/ai-recommendations/ownership/:contestId
// Get real-time ownership intelligence

// WebSocket: /ws/ai-recommendations/:userId
// Stream real-time AI insights and alerts
```

### 8. Database Schema

```sql
-- migrations/001_create_ai_recommendations.sql
CREATE TABLE ai_recommendations (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    contest_id INTEGER NOT NULL,
    request JSONB NOT NULL,
    response JSONB NOT NULL,
    model_used VARCHAR(50) NOT NULL,
    confidence FLOAT NOT NULL,
    tokens_used INTEGER,
    response_time_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_user_contest (user_id, contest_id),
    INDEX idx_created_at (created_at)
);

CREATE TABLE ownership_snapshots (
    id SERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    ownership_percentage FLOAT NOT NULL,
    projected_ownership FLOAT,
    trend VARCHAR(20),
    snapshot_time TIMESTAMP NOT NULL,
    source VARCHAR(50),
    
    INDEX idx_contest_player (contest_id, player_id),
    INDEX idx_snapshot_time (snapshot_time)
);

CREATE TABLE recommendation_feedback (
    id SERIAL PRIMARY KEY,
    recommendation_id INTEGER REFERENCES ai_recommendations(id),
    user_id INTEGER NOT NULL,
    feedback_type VARCHAR(50), -- 'followed', 'ignored', 'partial'
    lineup_result JSONB,
    roi FLOAT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Implementation Steps

### Phase 1: Service Setup (Days 1-3)
1. Create new microservice structure
2. Set up API Gateway routing
3. Configure database connections (Supabase)
4. Implement health checks
5. Set up Redis connection (DB 4)

### Phase 2: Claude Integration (Days 4-6)
1. Implement Claude API client with latest model
2. Add rate limiting (follow RapidAPI pattern)
3. Implement circuit breaker
4. Create prompt templates for each sport
5. Test API integration

### Phase 3: Real-Time Data Pipeline (Days 7-10)
1. Create data aggregator interfaces
2. Implement weather service integration
3. Add injury report monitoring
4. Create ownership tracking system
5. Set up WebSocket hub

### Phase 4: Ownership Intelligence (Days 11-13)
1. Build ownership analyzer
2. Implement leverage calculations
3. Create contrarian detection
4. Add stack ownership analysis

### Phase 5: Dynamic Prompting (Days 14-16)
1. Build prompt builder with templates
2. Implement context injection
3. Add personalization layer
4. Create sport-specific modifiers

### Phase 6: Integration & Testing (Days 17-20)
1. Connect all components
2. Implement caching layers
3. Add comprehensive logging
4. Create integration tests
5. Performance optimization

## Validation Gates

```bash
# Service health check
curl http://localhost:8084/health

# Test Claude integration
go test ./internal/services/claude_client_test.go -v

# Test real-time data pipeline
go test ./internal/services/realtime_aggregator_test.go -v

# Integration tests
go test ./tests/ai_recommendations_integration_test.go -v

# Load testing
hey -n 1000 -c 50 -m POST -d '{"contest_id": 123}' \
  http://localhost:8084/recommendations

# Verify WebSocket connections
wscat -c ws://localhost:8084/ws/ai-recommendations/user123
```

## Error Handling Strategy

1. **Claude API Failures**: Fallback to cached insights, then simplified recommendations
2. **Rate Limiting**: Queue requests, notify users of delays
3. **Data Source Failures**: Use stale data with confidence adjustments
4. **Circuit Breaker Pattern**: Automatic recovery with exponential backoff

## Performance Targets

- Recommendation latency: < 2 seconds (cached), < 5 seconds (fresh)
- WebSocket broadcast latency: < 100ms
- Cache hit rate: > 85% for repeated queries
- API availability: 99.5% uptime

## Security Considerations

1. Sanitize all LLM inputs to prevent prompt injection
2. Implement rate limiting per user
3. Use structured outputs to prevent data leakage
4. Monitor for unusual API usage patterns

## Future Enhancements

1. Multi-model support (GPT-4, Gemini)
2. A/B testing framework for prompts
3. Reinforcement learning from user feedback
4. Sport-specific fine-tuning

---

**Confidence Score**: 9/10

This PRP provides comprehensive context for implementing the AI recommendations intelligence upgrade. The implementation follows established patterns from the codebase while incorporating modern AI best practices and real-time data integration capabilities.