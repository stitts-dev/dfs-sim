## FEATURE:

Fix AI Recommendations Database Table and API Integration Issues

## CONTEXT & MOTIVATION:

The dfs_backend service is stuck in an infinite loop, repeatedly failing API calls and creating invalid database records. This is causing significant compute waste and preventing the AI recommendations feature from functioning properly. The main issues are:
1. BallDontLie API JSON unmarshalling error with the `next_cursor` field
2. ESPN API returning 400 errors for active teams endpoint
3. Players being inserted with 0 projections due to failed data fetches

## EXAMPLES:

Error logs showing the issues:
```
level=warning msg="Provider balldontlie failed: json: cannot unmarshal number into Go struct field ballDontLieMeta.meta.next_cursor of type string"
level=warning msg="Provider espn failed: failed to fetch active teams: unexpected status code: 400"
```

## CURRENT STATE ANALYSIS:

- The aggregator service is continuously retrying failed API calls with exponential backoff
- Players are being inserted into the database with all projection values set to 0
- The system is processing contests 3205 (Golf) and 3964 (WNBA) in a loop
- External IDs are being stored in scientific notation (e.g., '1.23207e+06')
- The database is being flooded with invalid player records

## TECHNICAL REQUIREMENTS:

### Backend Requirements:
- [x] Fix BallDontLie API struct to handle `next_cursor` as `interface{}` or `json.Number`
- [x] Debug ESPN API endpoint URL/parameters for active teams
- [x] Add circuit breaker pattern to prevent infinite retry loops
- [x] Implement data validation before inserting players
- [x] Fix external ID formatting (prevent scientific notation)

### Frontend Requirements:
- [ ] No frontend changes required for this fix

### Infrastructure Requirements:
- [x] Add monitoring for API failure rates
- [x] Implement rate limiting for external API calls
- [x] Add health check endpoint that includes external API status

## IMPLEMENTATION APPROACH:

### Phase 1: Immediate Fixes
1. Update BallDontLie provider struct in `backend/internal/providers/balldontlie.go`
2. Fix ESPN provider URL construction in `backend/internal/providers/espn.go`
3. Add validation in aggregator to prevent inserting players with 0 projections

### Phase 2: Resilience Improvements
1. Implement circuit breaker for external API calls
2. Add exponential backoff with maximum retry limit
3. Create fallback data source when APIs fail

### Phase 3: Monitoring & Alerting
1. Add structured logging for API failures
2. Implement metrics collection for API success/failure rates
3. Create alerts for sustained API failures

## DOCUMENTATION:

- BallDontLie API Documentation: https://www.balldontlie.io/home.html#introduction
- ESPN API endpoints and expected responses
- Go JSON unmarshalling with interface{} types
- Circuit breaker pattern implementation in Go

## TESTING STRATEGY:

### Unit Tests:
- [x] Test JSON unmarshalling with various `next_cursor` types
- [x] Mock API responses for both success and failure cases
- [x] Validate player data before insertion

### Integration Tests:
- [x] Test API provider fallback behavior
- [x] Verify circuit breaker opens after repeated failures
- [x] Ensure no invalid data reaches the database

### E2E Tests:
- [x] Verify system recovers gracefully from API outages
- [x] Test that valid player data is eventually fetched and stored

## POTENTIAL CHALLENGES & RISKS:

- API schema changes without notice (like the next_cursor type change)
- Rate limiting from external APIs not properly handled
- Database performance impact from repeated failed insertions
- Missing API keys or expired credentials

## SUCCESS CRITERIA:

- Backend service runs without infinite loops
- API failures are logged but don't crash the service
- Valid player data is fetched and stored with proper projections
- System gracefully handles API outages with appropriate fallbacks
- No players inserted with 0 projections when data is unavailable

## OTHER CONSIDERATIONS:

- Consider caching successful API responses more aggressively
- Implement a dead letter queue for failed API calls
- Add API response schema validation
- Consider using a message queue for asynchronous data fetching
- The external ID scientific notation issue suggests a type conversion problem

## MONITORING & OBSERVABILITY:

- Log all API request/response pairs at debug level
- Track metrics: API call count, success rate, response time
- Alert on: >50% API failure rate over 5 minutes
- Dashboard showing real-time API health status

## ROLLBACK PLAN:

1. Keep previous provider implementations as fallback
2. Feature flag to disable problematic providers
3. Manual override to use cached/static data
4. Database migration to clean up invalid player records