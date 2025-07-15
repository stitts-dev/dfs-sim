name: "Backend Startup Optimization - Eliminate Blocking Operations and Improve Scalability"
description: |

## Purpose
Template optimized for AI agents to implement features with sufficient context and self-validation capabilities to achieve working code through iterative refinement.

## Core Principles
1. **Context is King**: Include ALL necessary documentation, examples, and caveats
2. **Validation Loops**: Provide executable tests/lints the AI can run and fix
3. **Information Dense**: Use keywords and patterns from the codebase
4. **Progressive Success**: Start simple, validate, then enhance
5. **Global rules**: Be sure to follow all rules in CLAUDE.md

---

## Goal
Eliminate blocking startup operations in the Go DFS Optimizer backend by making external API calls and bulk database operations optional and configurable. Implement circuit breakers, health check endpoints, and graceful fallback mechanisms to ensure reliable startup even when external services are unavailable. Target <5 second critical path startup time while maintaining data freshness through background processes.

## Why
- **Business value**: Prevents deployment failures and enables horizontal scaling without coordination issues
- **Integration with existing features**: Maintains all current functionality while adding resilience
- **Problems this solves**: 
  - 30+ second startup times that can fail entirely if external APIs are down
  - Inability to scale horizontally due to startup coordination issues  
  - Poor developer experience with slow local development startup
  - Production deployment risk from synchronous external API dependencies during startup

## What
Implement configurable startup behavior with environment variables, circuit breaker patterns for external API calls, comprehensive health check endpoints for container orchestration, and admin endpoints for manual sync operations. All existing functionality continues working but becomes more resilient and scalable.

### Success Criteria
- [x] Backend startup completes critical path in <5 seconds
- [x] Startup succeeds even when external APIs (RapidAPI, ESPN, BallDontLie) are down
- [x] Multiple instances can start simultaneously without conflicts  
- [x] Clear visibility into startup phases and background job status via health endpoints
- [x] Ability to manually trigger sync operations when needed via admin endpoints
- [x] Core optimization functionality works with cached/fallback data

## All Needed Context

### Documentation & References
```yaml
# MUST READ - Include these in your context window
- url: https://github.com/sony/gobreaker
  why: Circuit breaker implementation patterns for Go, state management examples
  
- url: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
  why: Health check endpoint specifications for container orchestration
  
- url: https://microservices.io/patterns/reliability/circuit-breaker.html
  section: Circuit breaker pattern fundamentals
  critical: Three states (Closed/Open/Half-Open) and timeout handling
  
- file: backend/cmd/server/main.go
  why: Current startup sequence with blocking operations on lines 112-127
  
- file: backend/internal/services/data_fetcher.go  
  why: Immediate fetch operation on line 93, scheduled job patterns to mirror
  
- file: backend/internal/services/golf_tournament_sync.go
  why: Tournament sync patterns that make API calls, error handling examples
  
- file: backend/pkg/config/config.go
  why: Environment variable pattern using viper, existing configuration structure
  
- file: backend/internal/api/middleware/logger.go
  why: Structured logging patterns to follow for startup phases
```

### Current Codebase tree (run `tree` in the root of the project) to get an overview of the codebase
```bash
backend/
├── cmd/
│   ├── server/main.go      # Main entry point with blocking ops (lines 112-127)
│   └── migrate/main.go     # Database migration tool
├── internal/
│   ├── api/               # HTTP handlers and routes
│   │   ├── handlers/      # API endpoint handlers
│   │   ├── middleware/    # Request middleware (logging, CORS, auth)
│   │   └── router.go      # Route configuration
│   ├── services/          # Business logic services
│   │   ├── data_fetcher.go        # Scheduled data updates (line 93 blocking)
│   │   ├── golf_tournament_sync.go # Golf API sync (makes 2-5 API calls)
│   │   ├── contest_discovery.go   # Contest discovery for 6 sports
│   │   └── aggregator.go          # Multi-provider data aggregation
│   ├── providers/         # External API clients
│   │   ├── rapidapi_golf.go       # Rate-limited (20 req/day)
│   │   ├── espn_golf.go          # Fallback provider
│   │   └── balldontlie.go        # NBA data
│   └── models/            # Database models
└── pkg/
    ├── config/            # Environment configuration
    └── database/          # Database connection
```

### Desired Codebase tree with files to be added and responsibility of file
```bash
backend/
├── internal/
│   ├── api/
│   │   └── handlers/
│   │       └── health.go          # NEW: /health, /ready, /startup-status endpoints
│   │       └── admin.go           # NEW: /admin/sync/* manual operation endpoints
│   ├── services/
│   │   └── circuit_breaker.go     # NEW: Circuit breaker service wrapper
│   │   └── startup_manager.go     # NEW: Manages startup phases and background init
│   └── pkg/
│       └── startup/               # NEW: Startup orchestration package
│           ├── config.go          # NEW: Startup-specific configuration
│           ├── health.go          # NEW: Health check logic
│           └── phases.go          # NEW: Startup phase management
```

### Known Gotchas of our codebase & Library Quirks
```go
// CRITICAL: RapidAPI Golf provider has rate limit of 20 requests/day (Basic plan)
// Example: Must implement aggressive caching and fallback to ESPN Golf
// Location: backend/internal/providers/rapidapi_golf.go

// CRITICAL: Data fetcher auto-starts with immediate fetch on line 93
// Example: dataFetcher.Start() triggers fetchAllContests() immediately in goroutine
// Location: backend/internal/services/data_fetcher.go:93

// CRITICAL: Golf tournament sync creates 4 database contests per tournament
// Example: SyncAllActiveTournaments() -> 2 platforms × 2 contest types = 4 DB writes
// Location: backend/internal/services/golf_tournament_sync.go:245-270

// CRITICAL: Viper configuration uses SetDefault pattern
// Example: All new environment variables must follow viper.SetDefault() pattern
// Location: backend/pkg/config/config.go:54-73

// CRITICAL: Structured logging uses logrus.Fields pattern  
// Example: logger.WithFields(logrus.Fields{"phase": "startup", "operation": "golf_sync"})
// Location: backend/internal/api/middleware/logger.go:34-41

// CRITICAL: Contest discovery checks 6 different sports synchronously
// Example: Loop through all sports making external API calls
// Location: backend/internal/services/data_fetcher.go:298-319
```

## Implementation Blueprint

### Data models and structure

Create startup configuration and health check models to ensure type safety and consistency.
```go
// Startup configuration structure
type StartupConfig struct {
    SkipInitialGolfSync        bool          `mapstructure:"SKIP_INITIAL_GOLF_SYNC"`
    SkipInitialDataFetch       bool          `mapstructure:"SKIP_INITIAL_DATA_FETCH"`  
    SkipInitialContestDiscovery bool         `mapstructure:"SKIP_INITIAL_CONTEST_DISCOVERY"`
    StartupDelaySeconds        int           `mapstructure:"STARTUP_DELAY_SECONDS"`
    ExternalAPITimeout         time.Duration `mapstructure:"EXTERNAL_API_TIMEOUT"`
    CircuitBreakerThreshold    int           `mapstructure:"CIRCUIT_BREAKER_THRESHOLD"`
}

// Health check status model
type HealthStatus struct {
    Status           string                 `json:"status"`
    Timestamp        time.Time              `json:"timestamp"`
    StartupPhase     string                 `json:"startup_phase"`
    BackgroundJobs   map[string]JobStatus   `json:"background_jobs"`
    ExternalServices map[string]ServiceStatus `json:"external_services"`
}
```

### List of tasks to be completed to fulfill the PRP in the order they should be completed

```yaml
Task 1 - Add Startup Configuration:
MODIFY backend/pkg/config/config.go:
  - FIND pattern: "type Config struct"
  - INJECT new fields for startup control after existing fields
  - ADD viper.SetDefault() calls for new environment variables
  - PRESERVE existing configuration structure and patterns

Task 2 - Create Circuit Breaker Service:
CREATE backend/internal/services/circuit_breaker.go:
  - MIRROR pattern from: existing services with constructor pattern
  - IMPLEMENT sony/gobreaker integration for external API calls
  - KEEP error handling and logging patterns from other services

Task 3 - Create Startup Manager:
CREATE backend/internal/services/startup_manager.go:
  - MIRROR pattern from: data_fetcher.go for background job management
  - IMPLEMENT phase tracking and background initialization
  - PRESERVE existing service initialization patterns

Task 4 - Add Health Check Endpoints:
CREATE backend/internal/api/handlers/health.go:
  - MIRROR pattern from: existing handlers with gin.Context
  - IMPLEMENT /health, /ready, /startup-status endpoints
  - KEEP JSON response patterns from other handlers

Task 5 - Create Admin Control Endpoints:
CREATE backend/internal/api/handlers/admin.go:
  - MIRROR pattern from: existing handlers for POST operations
  - IMPLEMENT manual sync operation endpoints
  - PRESERVE authentication and error handling patterns

Task 6 - Modify Main Server Startup:
MODIFY backend/cmd/server/main.go:
  - FIND pattern: "// Initial sync of golf tournaments" (line 117)
  - REPLACE blocking operations with conditional startup manager
  - PRESERVE existing service initialization order and error handling

Task 7 - Update Data Fetcher Service:
MODIFY backend/internal/services/data_fetcher.go:
  - FIND pattern: "// Run initial fetch" (line 93)
  - MODIFY to respect startup configuration flags
  - KEEP existing cron job patterns and error handling

Task 8 - Add Circuit Breakers to Providers:
MODIFY backend/internal/providers/rapidapi_golf.go:
  - FIND pattern: external API call methods
  - WRAP calls with circuit breaker service
  - PRESERVE existing caching and rate limiting logic

Task 9 - Update API Router:
MODIFY backend/internal/api/router.go:
  - FIND pattern: route registration
  - ADD new health and admin endpoints
  - KEEP existing middleware and authentication patterns

Task 10 - Add Integration Tests:
CREATE backend/tests/startup_optimization_test.go:
  - MIRROR pattern from: existing integration tests
  - TEST various startup configurations and health endpoints
  - PRESERVE existing test setup and teardown patterns
```

### Per task pseudocode as needed added to each task

```go
// Task 2: Circuit Breaker Service Pseudocode
type CircuitBreakerService struct {
    breakers map[string]*gobreaker.CircuitBreaker
    logger   *logrus.Logger
}

func NewCircuitBreakerService(config *Config) *CircuitBreakerService {
    // PATTERN: Follow constructor pattern from other services
    settings := gobreaker.Settings{
        Name:        "external-api",
        MaxRequests: uint32(config.CircuitBreakerThreshold),
        Timeout:     config.ExternalAPITimeout,
        // CRITICAL: ReadyToTrip function determines when to open circuit
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            return counts.ConsecutiveFailures > 5
        },
    }
    // GOTCHA: Create separate breakers for each external service
    breakers := map[string]*gobreaker.CircuitBreaker{
        "rapidapi": gobreaker.NewCircuitBreaker(settings),
        "espn":     gobreaker.NewCircuitBreaker(settings),
    }
}

// Task 3: Startup Manager Pseudocode  
type StartupManager struct {
    phase        string
    backgroundJobs map[string]bool
    mu           sync.RWMutex
}

func (sm *StartupManager) StartBackgroundInitialization() {
    // PATTERN: Use goroutines like existing services
    go func() {
        // CRITICAL: Respect startup delay configuration
        time.Sleep(time.Duration(config.StartupDelaySeconds) * time.Second)
        
        if !config.SkipInitialGolfSync {
            sm.startGolfSync()
        }
        // GOTCHA: Update phase atomically for health checks
        sm.updatePhase("background_ready")
    }()
}

// Task 4: Health Check Endpoints Pseudocode
func (h *HealthHandler) GetStartupStatus(c *gin.Context) {
    // PATTERN: Use gin.H for JSON responses like other handlers
    status := h.startupManager.GetStatus()
    
    // CRITICAL: Different HTTP status codes for different states
    if status.Phase == "critical_ready" {
        c.JSON(http.StatusOK, gin.H{
            "status": "ready",
            "phase": status.Phase,
            "background_jobs": status.BackgroundJobs,
        })
    } else {
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "status": "starting",
            "phase": status.Phase,
        })
    }
}
```

### Integration Points
```yaml
CONFIGURATION:
  - add to: backend/pkg/config/config.go
  - pattern: "viper.SetDefault('SKIP_INITIAL_GOLF_SYNC', false)"
  - new fields: StartupConfig embedded in Config struct

LOGGING:
  - add to: all new services  
  - pattern: "logrus.WithFields(logrus.Fields{'component': 'startup_manager'})"
  - structured: Include startup phase and operation context

ROUTES:
  - add to: backend/internal/api/router.go
  - pattern: "r.GET('/health', handlers.NewHealthHandler().GetHealth)"
  - endpoints: /health, /ready, /startup-status, /api/v1/admin/sync/*

DATABASE:
  - no migrations: Uses existing contest and tournament tables
  - pattern: Follow existing GORM patterns for queries
  - critical: Don't modify database schema, only add startup behavior

EXTERNAL_APIS:
  - modify: All providers in backend/internal/providers/
  - pattern: Wrap existing API calls with circuit breaker.Execute()
  - fallback: Implement graceful degradation when circuits open
```

## Validation Loop

### Level 1: Syntax & Style
```bash
# Run these FIRST - fix any errors before proceeding  
cd backend
go mod tidy                     # Ensure dependencies are correct
go fmt ./...                    # Format code
go vet ./...                    # Static analysis
golangci-lint run               # Comprehensive linting (if available)

# Expected: No errors. If errors, READ the error and fix.
```

### Level 2: Unit Tests each new feature/file/function use existing test patterns
```go
// CREATE backend/tests/startup_optimization_test.go with these test cases:
func TestStartupConfiguration(t *testing.T) {
    """Test environment variable parsing works correctly"""
    os.Setenv("SKIP_INITIAL_GOLF_SYNC", "true")
    cfg, err := config.LoadConfig()
    assert.NoError(t, err)
    assert.True(t, cfg.SkipInitialGolfSync)
}

func TestCircuitBreakerIntegration(t *testing.T) {
    """Test circuit breaker wraps external API calls"""
    cb := services.NewCircuitBreakerService(testConfig)
    
    // Simulate failures to trip the circuit
    for i := 0; i < 6; i++ {
        _, err := cb.Execute("test", func() (interface{}, error) {
            return nil, errors.New("simulated failure")
        })
        assert.Error(t, err)
    }
    
    // Circuit should now be open
    _, err := cb.Execute("test", func() (interface{}, error) {
        return "success", nil
    })
    assert.Contains(t, err.Error(), "circuit breaker is open")
}

func TestHealthEndpoints(t *testing.T) {
    """Test health check endpoints return correct status"""
    // Setup test server with health endpoints
    router := setupTestRouter()
    
    // Test /health endpoint
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/health", nil)
    router.ServeHTTP(w, req)
    
    assert.Equal(t, 200, w.Code)
    
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.Equal(t, "ok", response["status"])
}
```

```bash
# Run and iterate until passing:
cd backend
go test ./tests/ -v
go test ./internal/services/ -v    # Test new services
# If failing: Read error, understand root cause, fix code, re-run (never mock to pass)
```

### Level 3: Integration Test
```bash
# Test startup with different configurations
cd backend

# Test 1: Fast startup (skip all initial operations)
export SKIP_INITIAL_GOLF_SYNC=true
export SKIP_INITIAL_DATA_FETCH=true  
export SKIP_INITIAL_CONTEST_DISCOVERY=true
go run cmd/server/main.go &
SERVER_PID=$!

# Verify fast startup
curl -s http://localhost:8080/health | jq .
curl -s http://localhost:8080/ready | jq .
curl -s http://localhost:8080/startup-status | jq .

kill $SERVER_PID

# Test 2: Manual sync operations
export SKIP_INITIAL_GOLF_SYNC=true
go run cmd/server/main.go &
SERVER_PID=$!

# Test manual golf sync
curl -X POST http://localhost:8080/api/v1/admin/sync/golf \
  -H "Content-Type: application/json"

# Expected: {"status": "started", "operation": "golf_sync"}
kill $SERVER_PID

# Test 3: Circuit breaker behavior (simulate API failures)
# This would require more complex setup with mock servers
```

## Final validation Checklist
- [x] All tests pass: `go test ./...` 
- [x] No linting errors: `golangci-lint run`
- [x] No static analysis issues: `go vet ./...`
- [x] Manual startup test successful with SKIP flags enabled
- [x] Health endpoints respond correctly during startup phases
- [x] Circuit breakers properly handle external API failures
- [x] Background jobs start correctly after startup delay
- [x] Admin endpoints allow manual sync operations
- [x] Logs are informative showing startup phases
- [x] Existing functionality unchanged when flags are disabled

---

## Anti-Patterns to Avoid
- ❌ Don't create new patterns when existing service patterns work
- ❌ Don't skip validation because "startup should work"  
- ❌ Don't ignore failing circuit breaker tests - fix the underlying issue
- ❌ Don't use synchronous operations in startup manager - use goroutines
- ❌ Don't hardcode timeouts - make them configurable
- ❌ Don't break existing API contracts - maintain backward compatibility
- ❌ Don't ignore error handling - implement graceful fallbacks
- ❌ Don't modify database schema - work with existing models

## Context-Specific Implementation Notes

### Circuit Breaker Integration Points:
```go
// CRITICAL: Wrap all external API calls in existing providers
// Pattern to follow in rapidapi_golf.go:
func (c *RapidAPIGolfClient) GetCurrentTournament() (*GolfTournamentData, error) {
    result, err := c.circuitBreaker.Execute("rapidapi", func() (interface{}, error) {
        // Existing API call logic here
        return c.makeAPICall("/tournaments/current")
    })
    
    if err != nil {
        // FALLBACK: Use ESPN Golf provider or cached data
        return c.fallbackProvider.GetCurrentTournament()
    }
    
    return result.(*GolfTournamentData), nil
}
```

### Health Check Implementation:
```go
// CRITICAL: Health checks must differentiate between critical and background readiness
// /health - Always returns 200 if server is running (basic liveness)
// /ready - Returns 200 only when critical services are ready (readiness probe)  
// /startup-status - Detailed information about startup phases and background jobs
```

### Environment Variable Defaults:
```go
// CRITICAL: Maintain backward compatibility - all flags default to current behavior
viper.SetDefault("SKIP_INITIAL_GOLF_SYNC", false)        // Keep current behavior by default
viper.SetDefault("SKIP_INITIAL_DATA_FETCH", false)       // Keep current behavior by default
viper.SetDefault("STARTUP_DELAY_SECONDS", 0)             // No delay by default
viper.SetDefault("EXTERNAL_API_TIMEOUT", "10s")          // Conservative timeout
viper.SetDefault("CIRCUIT_BREAKER_THRESHOLD", 5)         // Fail after 5 consecutive failures
```

This comprehensive PRP provides all necessary context, patterns, and validation steps for successful one-pass implementation of backend startup optimization while maintaining full backward compatibility and existing functionality.