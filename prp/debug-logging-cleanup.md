# PRP: Debug Logging Cleanup

## FEATURE:

Remove debug logging statements and implement proper structured logging throughout the optimization algorithm

## CONTEXT & MOTIVATION:

The optimization algorithm in `backend/internal/optimizer/algorithm.go` currently contains 20+ `fmt.Printf` debug statements that are polluting production logs and degrading performance. These synchronous print statements occur during the critical optimization path, which can slow down lineup generation. The codebase needs structured logging with proper log levels, contextual information, and production-ready output formatting.

**Current Pain Points:**
- Debug statements appear in production logs without log level control
- Printf statements are synchronous and impact optimization performance  
- Unstructured output makes debugging and monitoring difficult
- No contextual information (request IDs, user context) in logs
- Risk of accidentally logging sensitive data (salaries, user data)

## EXAMPLES:

**Current problematic code in `backend/internal/optimizer/algorithm.go`:**
```go
// Lines 55, 71, 129, 151, 181, 212, 239, 274, 283, 355, etc.
fmt.Printf("DEBUG: Starting optimization with %d players, sport=%s, platform=%s\n", 
    len(players), config.Contest.Sport, config.Contest.Platform)
fmt.Printf("DEBUG: After generation, found %d valid lineups\n", len(validLineups))
fmt.Printf("DEBUG: Player filtering - Total: %d, Excluded: %d, Injured: %d, Available: %d\n", 
    len(players), excludedCount, injuredCount, len(filtered))
```

**Should be replaced with structured logging:**
```go
log.WithFields(log.Fields{
    "total_players":    len(players),
    "sport":           config.Contest.Sport,
    "platform":        config.Contest.Platform,
    "optimization_id": optimizationID,
}).Debug("Starting optimization")

log.WithFields(log.Fields{
    "valid_lineups":   len(validLineups),
    "optimization_id": optimizationID,
}).Debug("Generated valid lineups")
```

## CURRENT STATE ANALYSIS:

**Existing Infrastructure:**
- Logrus already available as dependency (`github.com/sirupsen/logrus v1.9.3`)
- Basic `log.Printf` used in some handlers (`backend/internal/api/handlers/optimizer.go`)
- No centralized logging configuration
- No structured logging patterns established

**Affected Components:**
- `backend/internal/optimizer/algorithm.go` (primary target)
- `backend/internal/optimizer/slots.go` (minor logging statements)
- `backend/internal/api/handlers/optimizer.go` (upgrade existing logs)
- Migration scripts in `backend/cmd/migrate/main.go`

**Current Debug Statements Count:** 20+ fmt.Printf statements in optimization algorithm

## TECHNICAL REQUIREMENTS:

### Backend Requirements:
- [x] Logrus dependency available (already in go.mod)
- [ ] Initialize structured logger with proper configuration
- [ ] Replace all fmt.Printf debug statements with structured logging
- [ ] Implement log level configuration (ENV: LOG_LEVEL)
- [ ] Add contextual fields (optimization_id, request_id, sport, platform)
- [ ] Ensure no sensitive data logging (salaries, personal info)
- [ ] Performance optimization: async logging for debug level
- [ ] JSON formatting for production environments

### Frontend Requirements:
- [ ] No frontend changes required

### Infrastructure Requirements:
- [ ] LOG_LEVEL environment variable
- [ ] LOG_FORMAT environment variable (json/text)
- [ ] Log output configuration (stdout/file)
- [ ] Performance monitoring for logging impact

## IMPLEMENTATION APPROACH:

### Phase 1: Logger Setup & Configuration
```go
// backend/pkg/logger/logger.go
package logger

import (
    "os"
    "github.com/sirupsen/logrus"
)

func InitLogger() *logrus.Logger {
    log := logrus.New()
    
    // Set log level from environment
    if level := os.Getenv("LOG_LEVEL"); level != "" {
        if l, err := logrus.ParseLevel(level); err == nil {
            log.SetLevel(l)
        }
    }
    
    // Set formatter based on environment
    if os.Getenv("LOG_FORMAT") == "json" {
        log.SetFormatter(&logrus.JSONFormatter{})
    } else {
        log.SetFormatter(&logrus.TextFormatter{
            FullTimestamp: true,
        })
    }
    
    return log
}
```

### Phase 2: Replace Debug Statements
**Priority Order (based on performance impact):**
1. Core optimization loop statements (lines 212, 239, 274, 283)
2. Player filtering and organization (lines 129, 151)  
3. Initialization and summary statements (lines 55, 71, 355, 363)
4. Slot validation statements (lines 181, 291, 295)

**Structured Field Patterns:**
```go
// For optimization progress
log.WithFields(log.Fields{
    "optimization_id": optimizationID,
    "slot_index":     slotIndex,
    "slot_name":      slot.SlotName,
    "allowed_positions": slot.AllowedPositions,
}).Debug("Processing slot")

// For player stats
log.WithFields(log.Fields{
    "optimization_id": optimizationID,
    "total_players":   len(players),
    "excluded_count":  excludedCount,
    "injured_count":   injuredCount,
    "available_count": len(filtered),
}).Debug("Player filtering complete")
```

### Phase 3: Integration & Performance Optimization
- Add optimization_id generation using UUID
- Integrate with existing handler logging
- Performance testing with/without debug logging
- Context propagation for request tracing

## DOCUMENTATION:

**External References:**
- Logrus GitHub: https://github.com/sirupsen/logrus
- Go Logging Best Practices: https://betterstack.com/community/guides/logging/best-golang-logging-libraries/
- Logrus Performance Guide: https://signoz.io/guides/golang-logrus/
- Structured Logging Standards: https://gosolve.io/golang-logging-best-practices/

**Existing Code Patterns:**
- Handler logging: `backend/internal/api/handlers/optimizer.go:line 82-84`
- Error logging: `backend/cmd/migrate/main.go:line 15-18`
- Basic log usage: `backend/internal/optimizer/slots.go:line 25`

## TESTING STRATEGY:

### Unit Tests:
- [ ] Test logger initialization with different LOG_LEVEL values
- [ ] Test log output format (JSON vs text) based on LOG_FORMAT
- [ ] Test structured field inclusion in log entries
- [ ] Test performance impact of logging vs no-logging scenarios
- [ ] Test sensitive data exclusion (mock player salaries)

### Integration Tests:
- [ ] Test full optimization flow with debug logging enabled
- [ ] Test log output during actual optimization requests
- [ ] Verify optimization_id consistency across log statements
- [ ] Test log level filtering in different environments

### Performance Tests:
- [ ] Benchmark optimization performance with/without debug logging
- [ ] Memory usage comparison with structured vs printf logging
- [ ] Concurrent optimization logging safety tests

## POTENTIAL CHALLENGES & RISKS:

**Performance Concerns:**
- Structured logging may have slight overhead vs printf
- Risk: Debug logs in production if LOG_LEVEL not properly set
- Mitigation: Default to INFO level, performance benchmarking

**Breaking Changes:**
- Existing log parsing scripts may break if they depend on printf format
- Development debugging workflows may need adjustment
- Mitigation: Maintain backward compatibility option via environment flag

**Memory Usage:**
- Structured logging creates more objects than simple printf
- Risk: Increased GC pressure during high-volume optimizations  
- Mitigation: Use logrus.SetNoLock() for single-threaded optimization paths

**Contextual Data:**
- Need to propagate optimization_id through function calls
- Risk: Breaking existing function signatures
- Mitigation: Use context.Context pattern for request-scoped data

## SUCCESS CRITERIA:

- [ ] Zero fmt.Printf statements remain in optimization algorithm
- [ ] All log statements use structured fields with logrus
- [ ] LOG_LEVEL environment variable controls output
- [ ] No sensitive data appears in logs (verified via audit)
- [ ] Optimization performance impact < 5% with debug logging enabled
- [ ] All tests pass with new logging implementation
- [ ] Production logs show structured JSON format
- [ ] Request tracing works via optimization_id field

## OTHER CONSIDERATIONS:

**Common AI Assistant Gotchas:**
- Don't remove legitimate error logging (only debug printf statements)
- Preserve log.Printf usage in migration scripts and utilities (only target algorithm debug statements)
- Don't break existing handler logging patterns - enhance them
- Ensure optimization_id is generated once per request, not per function call
- Don't add logging to tight loops that could impact performance

**Security Considerations:**
- Never log player salaries, user IDs, or personal information
- Sanitize team/player names that might contain sensitive tournament data
- Use log level filtering to prevent debug data in production

**Performance Optimization:**
- Consider SetNoLock() for single-threaded optimization scenarios
- Use string constants for frequently logged field names
- Avoid creating unnecessary log.Fields{} objects in hot paths

## MONITORING & OBSERVABILITY:

**New Log Fields for Monitoring:**
- `optimization_id`: Unique identifier for tracking single requests
- `optimization_duration_ms`: Performance tracking
- `lineup_generation_time_ms`: Algorithm performance
- `player_count_by_position`: Data quality monitoring
- `salary_cap_utilization`: Optimization effectiveness

**Recommended Log Levels:**
- **DEBUG**: Player-by-player optimization details, slot processing
- **INFO**: Optimization start/completion, summary statistics
- **WARN**: Unusual conditions, missing players for positions
- **ERROR**: Optimization failures, constraint violations

## ROLLBACK PLAN:

**Phase 1 Rollback:** 
- Revert logger package, maintain existing printf statements
- Environment variables can be removed without impact

**Phase 2 Rollback:**
- Git revert individual commits per file
- Printf statements can be restored from git history
- No database or external service dependencies

**Emergency Rollback:**
- Feature flag: `USE_LEGACY_LOGGING=true` to bypass structured logging
- Minimal code path maintains printf statements as fallback

## VALIDATION GATES:

**Syntax/Style Validation:**
```bash
# Go formatting and linting  
cd backend
go fmt ./...
golangci-lint run ./internal/optimizer/...

# Build verification
go build ./cmd/server/
```

**Unit Tests:**
```bash
cd backend
go test ./internal/optimizer/... -v
go test ./pkg/logger/... -v
```

**Integration Tests:**
```bash
cd backend
go test ./tests/optimizer_integration_test.go -v
```

**Performance Validation:**
```bash
cd backend
go test -bench=BenchmarkOptimization ./internal/optimizer/... -benchtime=10s
```

**Log Output Validation:**
```bash
# Test structured logging output
LOG_LEVEL=debug LOG_FORMAT=json go run cmd/server/main.go 2>&1 | jq '.'

# Test production logging  
LOG_LEVEL=info LOG_FORMAT=json go run cmd/server/main.go
```

---

**PRP Confidence Score: 9/10**

This PRP provides comprehensive context for one-pass implementation success:
- ✅ All necessary technical context included
- ✅ Existing codebase patterns referenced with specific file/line numbers
- ✅ External documentation URLs provided for deep research
- ✅ Executable validation gates for immediate feedback
- ✅ Performance considerations and benchmarking approach
- ✅ Security considerations for sensitive data handling
- ✅ Clear implementation phases with specific code examples
- ✅ Rollback strategy for safe deployment
- ✅ Testing strategy covers unit, integration, and performance scenarios

High confidence due to:
- Logrus dependency already available
- Clear problem scope (20+ identifiable printf statements)
- Existing logging patterns to build upon
- Comprehensive external research providing best practices
- Executable validation steps for immediate feedback