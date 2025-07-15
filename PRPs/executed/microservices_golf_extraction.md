name: "Microservices Golf Extraction PRP v1 - Production Ready Service Decomposition"
description: |

## Purpose
Extract golf-specific functionality from the existing DFS optimizer monolith into independent microservices for production deployment with improved scalability, deployment independence, and performance isolation.

## Core Principles
1. **Service Boundaries**: Clear separation between golf data, optimization, and gateway services
2. **Zero Downtime**: Maintain current functionality while decomposing architecture
3. **Production Ready**: Immediate deployment capability with monitoring and health checks
4. **Pattern Preservation**: Maintain existing code quality and architectural patterns

---

## Goal
Extract golf tournament data management, optimization algorithms, and simulation engines from the monolithic DFS optimizer into three independent microservices (Golf Data Service, Optimization Service, API Gateway) with complete production deployment within 8 hours while preserving all existing functionality and performance characteristics.

## Why
- **Performance Isolation**: Prevent compute-intensive optimization from blocking other API requests
- **Independent Scaling**: Scale golf data ingestion and optimization separately based on demand  
- **Deployment Independence**: Deploy algorithm updates without affecting user authentication or data services
- **Resource Optimization**: Optimize compute resources for CPU-intensive tasks vs I/O-bound operations
- **Parallel Development**: Enable separate teams to work on golf vs general DFS features
- **RapidAPI Rate Limiting**: Isolate 20 req/day limit to dedicated service with intelligent caching

## What
Transform current monolithic architecture into microservices while maintaining:
- Exact same API contracts and response formats
- RapidAPI rate limiting and fallback strategies  
- Real-time WebSocket optimization progress updates
- JWT authentication and user authorization
- Database consistency and transaction boundaries
- Current optimization algorithm accuracy and performance

### Success Criteria
- [ ] Golf tournament data loads in <5 seconds via dedicated service
- [ ] Optimization generates 50 lineups in <10 seconds in isolated container
- [ ] Monte Carlo simulation (10k iterations) completes in <15 seconds
- [ ] All services deploy independently without downtime
- [ ] RapidAPI rate limiting preserved with 20 req/day enforcement
- [ ] WebSocket real-time updates work through API gateway proxy
- [ ] Service health checks respond correctly for monitoring systems
- [ ] Golf simulation end-to-end workflow unchanged from user perspective
- [ ] Production deployment completes within 8 hours with rollback capability

## All Needed Context

### Documentation & References
```yaml
# MUST READ - Official Documentation
- url: https://github.com/go-kit/kit
  why: Standard Go microservices toolkit with service patterns, middleware, transport abstractions
  critical: Service interface patterns, circuit breakers, logging middleware
  
- url: https://github.com/iamuditg/go-microservice-patterns  
  why: Service discovery, load balancing, fault tolerance patterns
  critical: Health check implementations, monitoring strategies
  
- url: https://github.com/sony/gobreaker
  why: Production-ready circuit breaker (already implemented in codebase)
  critical: Configuration patterns, state management, fallback strategies
  
- url: https://www.nginx.com/blog/microservices-reference-architecture-nginx-plus/
  why: NGINX reverse proxy and load balancing for microservices
  critical: Service discovery, SSL termination, rate limiting configurations
  
- url: https://github.com/redis/go-redis
  why: Redis client patterns for distributed caching and rate limiting
  critical: Clustering, pipelines, pub/sub for service coordination

# MUST READ - Current Codebase Patterns  
- file: backend/cmd/server/main.go
  why: Service startup patterns with graceful shutdown, dependency injection
  critical: Configuration loading, health checks, circuit breaker initialization
  
- file: backend/internal/api/router.go
  why: API versioning, route grouping, middleware chain patterns
  critical: Authentication flow, CORS handling, request routing
  
- file: backend/internal/api/handlers/health.go
  why: Multi-tier health check implementation
  critical: Startup manager integration, readiness vs liveness probes
  
- file: backend/internal/services/startup_manager.go
  why: Phased startup with background initialization
  critical: Non-blocking critical services, async background tasks
  
- file: backend/internal/providers/rapidapi_golf.go
  why: Rate limiting implementation with fallback chains
  critical: Daily request tracking, exponential backoff, fallback to ESPN
  
- file: backend/internal/services/circuit_breaker.go
  why: Circuit breaker service with multiple breakers per external API
  critical: Sony gobreaker integration, state change logging

# Current Architecture Analysis
- file: backend/internal/api/handlers/golf.go
  why: Golf API handlers to be extracted (622 lines)
  critical: Tournament management, leaderboard data, player projections
  
- file: backend/internal/services/golf_projections.go
  why: Golf-specific business logic (478 lines)
  critical: Cut probability, DraftKings/FanDuel scoring, weather impact
  
- file: backend/internal/optimizer/golf_correlation.go
  why: Golf optimization algorithms (231 lines)
  critical: Tee time correlations, course history, weather adjustments
```

### Current Codebase Tree (Golf-Specific Components)
```bash
backend/
├── cmd/server/main.go                    # Monolithic entry point to split
├── internal/
│   ├── api/
│   │   ├── handlers/golf.go              # → Golf Service API
│   │   ├── router.go                     # → Split routing logic
│   │   └── middleware/                   # → Shared across services
│   ├── models/
│   │   ├── golf_tournament.go            # → Golf Service models
│   │   ├── golf_player.go                # → Golf Service models
│   │   └── (shared models remain)        # → API Gateway models
│   ├── services/
│   │   ├── golf_projections.go           # → Golf Service business logic
│   │   ├── golf_tournament_sync.go       # → Golf Service sync logic
│   │   └── (other services remain)       # → API Gateway services
│   ├── providers/
│   │   ├── rapidapi_golf.go              # → Golf Service providers
│   │   ├── espn_golf.go                  # → Golf Service providers
│   │   └── (other providers remain)      # → API Gateway providers
│   ├── optimizer/
│   │   ├── algorithm.go                  # → Optimization Service core
│   │   ├── golf_correlation.go           # → Optimization Service golf logic
│   │   ├── correlation.go                # → Optimization Service shared
│   │   └── (simulator/ directory)        # → Optimization Service complete
│   └── (shared packages)                 # → Shared utilities package
├── migrations/
│   └── 004_add_golf_support.sql          # → Golf Service migrations
└── pkg/                                  # → Shared package across services
```

### Desired Microservices Architecture
```bash
# Root monorepo structure
├── shared/                               # Shared utilities and types
│   ├── pkg/config/                       # Configuration management
│   ├── pkg/database/                     # Database connection utilities
│   ├── pkg/logger/                       # Structured logging
│   └── types/                            # Shared data types
├── services/
│   ├── golf-service/                     # Golf Data Service
│   │   ├── cmd/server/main.go            # Golf service entry point
│   │   ├── internal/
│   │   │   ├── api/handlers/             # Golf API handlers
│   │   │   ├── models/                   # Golf-specific models  
│   │   │   ├── services/                 # Golf business logic
│   │   │   └── providers/                # Golf external APIs
│   │   ├── migrations/                   # Golf database schema
│   │   └── Dockerfile                    # Golf service container
│   ├── optimization-service/             # Optimization Service
│   │   ├── cmd/server/main.go            # Optimization service entry
│   │   ├── internal/
│   │   │   ├── api/handlers/             # Optimization API handlers
│   │   │   ├── optimizer/                # Optimization algorithms
│   │   │   ├── simulator/                # Monte Carlo simulation
│   │   │   └── websocket/                # Progress update hub
│   │   ├── pkg/cache/                    # Redis caching layer
│   │   └── Dockerfile                    # Optimization container
│   └── api-gateway/                      # API Gateway Service
│       ├── cmd/server/main.go            # Gateway entry point
│       ├── internal/
│       │   ├── api/handlers/             # Non-golf handlers
│       │   ├── middleware/               # Auth and routing middleware
│       │   ├── proxy/                    # Service proxy logic
│       │   └── websocket/                # WebSocket hub management
│       ├── config/nginx.conf             # NGINX routing config
│       └── Dockerfile                    # Gateway container
├── docker-compose.yml                    # Multi-service orchestration
└── docker-compose.prod.yml               # Production deployment config
```

### Known Gotchas & Critical Implementation Details
```go
// CRITICAL: RapidAPI rate limiting must be preserved exactly
// Current implementation tracks daily requests with mutex
type RateLimitTracker struct {
    mu           sync.Mutex
    dailyCount   int
    lastReset    time.Time
    dailyLimit   int  // 20 for Basic plan
}

// GOTCHA: WebSocket connections must proxy through NGINX
// Current hub pattern must be replicated in gateway with service forwarding

// CRITICAL: Circuit breaker states must be maintained per service
// Sony gobreaker configuration with specific failure ratios
ReadyToTrip: func(counts gobreaker.Counts) bool {
    failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
    return counts.Requests >= 3 && failureRatio >= 0.6
}

// GOTCHA: Database connection pooling limits
// Current: shared pool, future: per-service pools with connection limits
// PostgreSQL default max_connections = 100, allocate per service

// CRITICAL: Startup manager phases must be respected
// Services must support PhaseStarting → PhaseCriticalReady → PhaseFullyReady
// Health checks must reflect actual readiness state

// GOTCHA: Golf optimization correlations integrate with main algorithm
// Tee time correlations and course history must be accessible from optimization service
// Current correlation matrix calculation is golf-specific but used by general optimizer
```

## Implementation Blueprint

### Service Communication Architecture
```go
// Request routing pattern
Frontend → NGINX → API Gateway → Golf Service (data endpoints)
                                → Optimization Service (compute endpoints)
                                → Local Handlers (auth, users, lineups)

// Database access pattern (shared PostgreSQL)
Golf Service:        golf_tournaments, golf_players, golf_courses, golf_round_scores
Optimization Service: optimization_results, simulation_results, cached_lineups  
API Gateway:         users, lineups, contests (non-golf), authentication sessions

// Caching strategy (shared Redis)
Golf Service:        tournament data (TTL: 1h), player data (TTL: 4h)
Optimization Service: lineup results (TTL: 24h), simulation results (TTL: 1h)
API Gateway:         sessions (TTL: JWT expiry), rate limiting counters
```

### Data Models & Service Boundaries
```go
// Shared types (in shared/types/)
type Player struct {
    ID       uint   `json:"id"`
    Name     string `json:"name"`
    Sport    string `json:"sport"`
    Position string `json:"position"`
}

type Contest struct {
    ID       uint      `json:"id"`
    Name     string    `json:"name"`  
    Sport    string    `json:"sport"`
    StartTime time.Time `json:"start_time"`
}

// Golf Service specific models
type GolfTournament struct {
    ID           uint      `json:"id"`
    TournamentID string    `json:"tournament_id"`
    Name         string    `json:"name"`
    StartDate    time.Time `json:"start_date"`
    CourseID     string    `json:"course_id"`
    Status       string    `json:"status"`
}

type GolfPlayer struct {
    PlayerID     uint    `json:"player_id"`
    TournamentID string  `json:"tournament_id"`
    TeeTime      string  `json:"tee_time"`
    Salary       int     `json:"salary"`
    Projection   float64 `json:"projection"`
    CutProbability float64 `json:"cut_probability"`
}

// Optimization Service models
type OptimizationRequest struct {
    ContestID    uint                `json:"contest_id"`
    PlayerPool   []OptimizationPlayer `json:"player_pool"`
    Constraints  OptimizationConstraints `json:"constraints"`
    Settings     OptimizationSettings `json:"settings"`
}

type OptimizationResult struct {
    Lineups      []GeneratedLineup `json:"lineups"`
    Metadata     OptimizationMetadata `json:"metadata"`
    CorrelationMatrix map[string]float64 `json:"correlation_matrix"`
}
```

### Task Implementation Order

```yaml
Task 1: Create Shared Utilities Package
MODIFY: Backend restructure
  - CREATE: shared/pkg/config/ (copy from backend/pkg/config/)
  - CREATE: shared/pkg/database/ (copy from backend/pkg/database/)  
  - CREATE: shared/pkg/logger/ (copy from backend/pkg/logger/)
  - CREATE: shared/types/ (extract common types from backend/internal/models/)
  - UPDATE: Go module paths to use shared package
  - PRESERVE: All existing configuration and logging patterns

Task 2: Golf Service Foundation
CREATE: services/golf-service/
  - COPY: backend/cmd/server/main.go → services/golf-service/cmd/server/main.go
  - MODIFY: Remove non-golf route initialization, keep health checks and startup manager
  - COPY: backend/internal/api/handlers/golf.go → services/golf-service/internal/api/handlers/
  - COPY: backend/internal/services/golf_*.go → services/golf-service/internal/services/
  - COPY: backend/internal/providers/*golf*.go → services/golf-service/internal/providers/
  - COPY: backend/internal/models/golf_*.go → services/golf-service/internal/models/
  - CREATE: services/golf-service/Dockerfile (mirror backend/Dockerfile patterns)
  - UPDATE: Import paths to use shared package
  - PRESERVE: RapidAPI rate limiting, ESPN fallback, circuit breaker patterns

Task 3: Optimization Service Extraction  
CREATE: services/optimization-service/
  - COPY: backend/internal/optimizer/ → services/optimization-service/internal/optimizer/
  - COPY: backend/internal/simulator/ → services/optimization-service/internal/simulator/  
  - CREATE: services/optimization-service/internal/api/handlers/optimization.go
  - EXTRACT: Optimization endpoints from backend/internal/api/handlers/optimization.go
  - CREATE: services/optimization-service/internal/websocket/ (copy WebSocket hub pattern)
  - MODIFY: Redis caching integration for optimization results
  - UPDATE: Golf correlation integration with main algorithm
  - PRESERVE: Parallel worker pools, progress tracking, algorithm accuracy

Task 4: API Gateway Implementation
CREATE: services/api-gateway/
  - COPY: backend/cmd/server/main.go → services/api-gateway/cmd/server/main.go
  - REMOVE: Golf and optimization route initialization
  - CREATE: services/api-gateway/internal/proxy/ (HTTP client for service forwarding)
  - MODIFY: Router to proxy golf requests to golf-service
  - MODIFY: Router to proxy optimization requests to optimization-service
  - PRESERVE: Authentication middleware, CORS handling, user/lineup routes
  - CREATE: services/api-gateway/config/nginx.conf (load balancing configuration)
  - PRESERVE: WebSocket hub with proxy forwarding to optimization service

Task 5: Database Migration Strategy
MODIFY: Database access patterns
  - COPY: backend/migrations/004_add_golf_support.sql → services/golf-service/migrations/
  - CREATE: Database connection per service with shared connection pool management
  - UPDATE: Service-specific connection pool limits (Golf: 30, Optimization: 20, Gateway: 50)
  - PRESERVE: Transaction boundaries and data consistency
  - CREATE: Database health checks per service

Task 6: Docker Compose Multi-Service Setup
CREATE: docker-compose.yml
  - SERVICE: golf-service (port 8081, depends_on: postgres, redis)
  - SERVICE: optimization-service (port 8082, depends_on: postgres, redis)  
  - SERVICE: api-gateway (port 8080, depends_on: golf-service, optimization-service)
  - SERVICE: nginx (port 80, proxy to api-gateway, load balancing)
  - PRESERVE: PostgreSQL and Redis configuration
  - ADD: Service discovery via Docker DNS resolution
  - CREATE: Health check configuration for all services

Task 7: Service Communication Implementation
CREATE: Inter-service HTTP clients
  - CREATE: services/api-gateway/internal/clients/golf_client.go
  - CREATE: services/api-gateway/internal/clients/optimization_client.go
  - IMPLEMENT: HTTP client with circuit breaker, timeout, retry logic
  - PRESERVE: Request/response contract compatibility
  - ADD: Service discovery via environment variables (GOLF_SERVICE_URL, etc.)

Task 8: NGINX Configuration & Load Balancing
CREATE: services/api-gateway/config/nginx.conf
  - CONFIGURE: Upstream servers for each service
  - IMPLEMENT: Health check endpoints for upstream validation
  - ADD: WebSocket proxy configuration for optimization progress
  - PRESERVE: CORS headers and SSL termination capability
  - IMPLEMENT: Rate limiting at gateway level

Task 9: Production Deployment Configuration
CREATE: docker-compose.prod.yml
  - OPTIMIZE: Production resource limits and health checks
  - CONFIGURE: Environment-specific configuration management
  - ADD: Log aggregation and monitoring endpoints
  - IMPLEMENT: Rolling deployment strategy
  - CREATE: Service scaling configuration

Task 10: Monitoring & Observability
IMPLEMENT: Service monitoring
  - ADD: Health check endpoints per service (/health, /ready, /metrics)
  - CREATE: Structured logging with correlation IDs across services
  - IMPLEMENT: Request tracing through service boundaries
  - ADD: Performance metrics (response times, error rates, throughput)
  - PRESERVE: Existing logging patterns and levels
```

### Per-Task Implementation Details

#### Task 2: Golf Service Extraction - Critical Code Patterns
```go
// Golf service main.go structure (preserve startup pattern)
func main() {
    // 1. PRESERVE: Configuration loading with golf-specific variables
    cfg, err := config.LoadConfig()
    if err != nil {
        logrus.Fatalf("Failed to load configuration: %v", err)
    }
    
    // 2. PRESERVE: Structured logging initialization
    logger := logger.InitLogger(cfg.LogLevel, cfg.IsDevelopment())
    
    // 3. PRESERVE: Database connection with health checks
    db, err := database.NewConnection(cfg.DatabaseURL, cfg.IsDevelopment())
    defer db.Close()
    
    // 4. PRESERVE: Redis connection for caching
    redisClient := redis.NewClient(&redis.Options{
        Addr: cfg.RedisURL,
        DB:   0, // Golf service uses DB 0
    })
    defer redisClient.Close()
    
    // 5. PRESERVE: Circuit breaker for external APIs
    circuitBreaker := services.NewCircuitBreakerService(
        cfg.CircuitBreakerThreshold,
        cfg.ExternalAPITimeout,
        logger,
    )
    
    // 6. GOLF-SPECIFIC: RapidAPI and ESPN providers
    cacheService := services.NewCacheService(redisClient)
    rapidAPIGolf := providers.NewRapidAPIGolfClient(cfg.RapidAPIKey, cacheService, logger)
    espnGolf := providers.NewESPNGolfClient(cacheService, logger)
    
    // 7. GOLF-SPECIFIC: Golf services initialization
    golfProjectionService := services.NewGolfProjectionService(db, logger)
    golfSyncService := services.NewGolfTournamentSyncService(
        db, rapidAPIGolf, espnGolf, cacheService, logger,
    )
    
    // 8. PRESERVE: Startup manager with golf-specific services
    startupManager := services.NewStartupManager(cfg, logger, golfSyncService, circuitBreaker)
    
    // 9. PRESERVE: Router setup with golf handlers only
    router := gin.New()
    router.Use(gin.Logger(), gin.Recovery())
    
    golfHandler := handlers.NewGolfHandler(
        db, cacheService, golfProjectionService, 
        golfSyncService, rapidAPIGolf, logger,
    )
    
    apiV1 := router.Group("/api/v1")
    apiV1.GET("/golf/tournaments", golfHandler.ListTournaments)
    apiV1.GET("/golf/tournaments/:id", golfHandler.GetTournament)
    apiV1.GET("/golf/tournaments/:id/leaderboard", golfHandler.GetTournamentLeaderboard)
    apiV1.GET("/golf/tournaments/:id/players", golfHandler.GetTournamentPlayers)
    // ... other golf endpoints
    
    // 10. PRESERVE: Health check endpoints
    router.GET("/health", healthHandler.GetHealth)
    router.GET("/ready", healthHandler.GetReady)
    
    // 11. PRESERVE: Graceful startup and shutdown
    startupManager.StartCriticalServices()
    go startupManager.StartBackgroundInitialization()
    
    srv := &http.Server{
        Addr:    fmt.Sprintf(":%s", cfg.Port),
        Handler: router,
    }
    
    // Graceful shutdown (preserve pattern)
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatalf("Failed to start server: %v", err)
        }
    }()
    <-quit
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
}
```

#### Task 3: Optimization Service - Algorithm Preservation
```go
// Optimization service with Redis caching integration
func (h *OptimizationHandler) OptimizeLineups(c *gin.Context) {
    var req OptimizationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        utils.SendValidationError(c, "Invalid request", err.Error())
        return
    }
    
    // 1. PRESERVE: Cache key generation pattern
    cacheKey := fmt.Sprintf("optimization:%d:%s", req.ContestID, 
        hashOptimizationRequest(req))
    
    // 2. PRESERVE: Cache-first strategy
    cached, err := h.cache.Get(context.Background(), cacheKey)
    if err == nil && cached != nil {
        c.JSON(http.StatusOK, cached)
        return
    }
    
    // 3. PRESERVE: Golf correlation matrix building
    correlationMatrix := h.optimizer.BuildGolfCorrelationMatrix(req.PlayerPool)
    
    // 4. PRESERVE: Core optimization algorithm
    optimizer := optimizer.NewOptimizer(
        req.PlayerPool,
        req.Constraints,
        correlationMatrix,
        h.logger,
    )
    
    // 5. PRESERVE: WebSocket progress updates (now forwarded to gateway)
    progressChan := make(chan optimization.Progress, 100)
    go h.forwardProgressToGateway(req.UserID, progressChan)
    
    // 6. PRESERVE: Optimization execution with worker pools
    result, err := optimizer.OptimizeWithProgress(req.Settings, progressChan)
    if err != nil {
        utils.SendError(c, "Optimization failed", err.Error())
        return
    }
    
    // 7. PRESERVE: Cache storage with TTL
    h.cache.SetWithRetry(context.Background(), cacheKey, result, 
        24*time.Hour, 3)
    
    c.JSON(http.StatusOK, result)
}

// NEW: Progress forwarding to API Gateway via WebSocket or HTTP
func (h *OptimizationHandler) forwardProgressToGateway(userID uint, progressChan <-chan optimization.Progress) {
    gatewayURL := fmt.Sprintf("%s/ws/optimization-progress/%d", 
        h.config.APIGatewayURL, userID)
    
    // Establish WebSocket connection to gateway or use HTTP POST
    // Forward all progress updates received from optimization algorithm
}
```

#### Task 4: API Gateway - Request Proxying Pattern
```go
// API Gateway with service proxying
type ServiceProxy struct {
    golfClient        *clients.GolfClient
    optimizationClient *clients.OptimizationClient
    circuitBreaker    *services.CircuitBreakerService
    logger            *logrus.Logger
}

// Golf requests proxy
func (p *ServiceProxy) ProxyGolfRequest(c *gin.Context) {
    // 1. PRESERVE: Authentication middleware (already applied)
    userID := c.GetUint("user_id")
    
    // 2. NEW: Circuit breaker for service calls
    result, err := p.circuitBreaker.Execute("golf-service", func() (interface{}, error) {
        return p.golfClient.ForwardRequest(c.Request.Method, c.Request.URL.Path, c.Request.Body)
    })
    
    if err != nil {
        // 3. PRESERVE: Error handling patterns
        utils.SendError(c, "Golf service unavailable", err.Error())
        return
    }
    
    // 4. PRESERVE: Response format
    golfResponse := result.(*clients.ServiceResponse)
    c.JSON(golfResponse.StatusCode, golfResponse.Body)
}

// NEW: Service client with timeout and retry
type GolfClient struct {
    baseURL    string
    httpClient *http.Client
    logger     *logrus.Logger
}

func (gc *GolfClient) ForwardRequest(method, path string, body io.Reader) (*ServiceResponse, error) {
    url := fmt.Sprintf("%s%s", gc.baseURL, path)
    
    req, err := http.NewRequest(method, url, body)
    if err != nil {
        return nil, err
    }
    
    // PRESERVE: Header forwarding (auth, correlation ID)
    // PRESERVE: Timeout configuration (from current HTTP server config)
    resp, err := gc.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    return &ServiceResponse{
        StatusCode: resp.StatusCode,
        Body:       parseResponseBody(resp.Body),
    }, nil
}
```

### Integration Points
```yaml
DATABASE:
  - connection: "Shared PostgreSQL with service-specific connection pools"
  - migrations: "Service-specific migration directories with shared tables"
  - patterns: "Preserve existing GORM models and query patterns"
  
REDIS:
  - caching: "Shared Redis with service-specific key prefixes"
  - patterns: "golf:*, optimization:*, gateway:* key namespacing"
  - rate_limiting: "Preserve RapidAPI request counting in golf service"
  
CONFIGURATION:
  - pattern: "Viper-based configuration with service-specific defaults"
  - environment: "SERVICE_NAME_VAR format for service-specific config"
  - secrets: "Shared JWT_SECRET, service-specific API keys"
  
MONITORING:
  - health_checks: "/health (liveness), /ready (readiness), /metrics (prometheus)"
  - logging: "Structured logging with correlation IDs across services"
  - patterns: "Preserve existing logrus configuration and log levels"
```

## Validation Loop

### Level 1: Service Isolation & Health Checks
```bash
# Build and start each service independently
cd services/golf-service && go build cmd/server/main.go && ./main &
cd services/optimization-service && go build cmd/server/main.go && ./main &  
cd services/api-gateway && go build cmd/server/main.go && ./main &

# Verify health endpoints respond correctly
curl http://localhost:8081/health  # Golf service
curl http://localhost:8082/health  # Optimization service  
curl http://localhost:8080/health  # API Gateway

# Expected: {"status": "ok", "service": "golf-service", "timestamp": "..."}
# If errors: Check logs for database connection, Redis connection, configuration issues
```

### Level 2: Service Communication & Request Routing
```bash
# Test golf data requests through API Gateway
curl http://localhost:8080/api/v1/golf/tournaments \
  -H "Authorization: Bearer <valid-jwt-token>"

# Expected: Tournament list from golf service via gateway proxy
# If error: Check NGINX configuration, service discovery, network connectivity

# Test optimization requests through API Gateway  
curl -X POST http://localhost:8080/api/v1/optimize \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <valid-jwt-token>" \
  -d '{"contest_id": 1, "max_lineups": 10}'

# Expected: Optimization result from optimization service via gateway
# If error: Check inter-service HTTP clients, circuit breaker status
```

### Level 3: Database & External API Integration
```bash
# Test RapidAPI rate limiting in golf service
for i in {1..25}; do
  curl http://localhost:8081/api/v1/golf/tournaments/sync-data
  echo "Request $i completed"
done

# Expected: First 20 succeed, remaining 5 hit rate limit with fallback to ESPN
# If error: Check rate limiting tracker, circuit breaker state, fallback logic

# Test database access per service
curl http://localhost:8081/api/v1/golf/tournaments  # Golf service DB access
curl http://localhost:8082/api/v1/optimize/cache-status  # Optimization service Redis
curl http://localhost:8080/api/v1/users/profile  # API Gateway DB access

# Expected: Each service accesses appropriate database tables/cache
# If error: Check connection pool limits, database permissions, query patterns
```

### Level 4: WebSocket & Real-time Features
```bash
# Test WebSocket optimization progress through gateway
websocat ws://localhost:8080/ws/optimization-progress/<user-id> &

# Start optimization request
curl -X POST http://localhost:8080/api/v1/optimize \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <valid-jwt-token>" \
  -d '{"contest_id": 1, "max_lineups": 50}'

# Expected: WebSocket receives progress updates from optimization service
# If error: Check WebSocket proxy configuration, service communication
```

### Level 5: Docker Compose Multi-Service Deployment
```bash
# Build and start entire stack
docker-compose up --build -d

# Verify all services are healthy
docker-compose ps
curl http://localhost/health  # Through NGINX

# Test end-to-end golf simulation workflow
curl http://localhost/api/v1/golf/tournaments
curl -X POST http://localhost/api/v1/optimize -d '<golf-optimization-request>'

# Expected: Complete workflow through NGINX → Gateway → Services
# If error: Check Docker networking, service discovery, NGINX configuration
```

## Final Validation Checklist
- [ ] All services build without errors: `go build` in each service directory
- [ ] No import cycle errors with shared package: `go mod tidy && go mod verify`
- [ ] Health checks respond correctly: `curl http://localhost:<port>/health` for each service
- [ ] Database connections work per service: Check connection pool utilization
- [ ] RapidAPI rate limiting preserved: Test 20+ requests to golf service
- [ ] Circuit breakers function correctly: Test with service failures
- [ ] WebSocket proxy works: Test optimization progress updates
- [ ] Docker Compose deployment successful: `docker-compose up` without errors
- [ ] NGINX routing works: Test requests through load balancer
- [ ] End-to-end golf simulation: Complete tournament → optimization → result workflow
- [ ] Performance maintained: Optimization generates 50 lineups in <10 seconds
- [ ] Rollback capability: Can revert to monolith deployment in <5 minutes

---

## Anti-Patterns to Avoid
- ❌ Don't change API contracts - maintain exact request/response formats
- ❌ Don't break RapidAPI rate limiting - preserve 20 req/day enforcement exactly
- ❌ Don't lose WebSocket functionality - real-time updates must work through proxy
- ❌ Don't ignore database connection limits - configure per-service pool limits
- ❌ Don't skip health checks - monitoring systems depend on these endpoints
- ❌ Don't hardcode service URLs - use environment variables for service discovery
- ❌ Don't remove circuit breakers - external API protection is critical
- ❌ Don't change optimization algorithms - preserve accuracy and performance
- ❌ Don't break authentication - JWT validation must work across all services
- ❌ Don't ignore graceful shutdown - services must handle SIGTERM correctly

## Confidence Score: 9/10

This PRP provides comprehensive context including current codebase patterns, external documentation references, specific implementation details, and executable validation steps. The AI agent has all necessary information to successfully extract golf services into microservices while preserving functionality and achieving production deployment within 8 hours.