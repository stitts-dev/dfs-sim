## FEATURE:

Backend Startup Optimization - Eliminate Blocking Operations and Improve Scalability

## CONTEXT & MOTIVATION:

The backend currently experiences severe startup blocking due to synchronous external API calls and bulk database operations during initialization. This creates several critical issues:

1. **Deployment Risk**: Startup can take 30+ seconds or fail entirely if external APIs are down
2. **Scalability Concerns**: Each instance startup triggers dozens of API calls and database operations
3. **Development Experience**: Local development startup is slow and unpredictable
4. **Production Readiness**: No clear separation between critical startup and background tasks

The current startup process makes external API calls to RapidAPI, ESPN, BallDontLie, and others, creates multiple database contests per tournament, and attempts to sync all data immediately. This is fundamentally incompatible with modern containerized deployment patterns and horizontal scaling.

## EXAMPLES:

Current problematic startup sequence:
1. Golf Tournament Sync: Makes 2-5 RapidAPI calls, creates 4 DB contests per tournament
2. Data Fetcher: Queries all upcoming contests, calls multiple external APIs
3. Contest Discovery: Checks 6 different sports for new contests

## CURRENT STATE ANALYSIS:

### Existing Components:
- `cmd/server/main.go`: Main entry point with blocking operations
- `services/golf_tournament_sync.go`: Makes multiple API calls on startup
- `services/data_fetcher.go`: Auto-starts with immediate fetch operations
- `services/aggregator.go`: Calls multiple external providers
- `providers/rapidapi_golf.go`: Rate-limited API with 20 req/day limit

### Current Blocking Operations:
1. **Line 117-127 in main.go**: Initial golf tournament sync in goroutine
2. **Line 112-115 in main.go**: Data fetcher auto-start with immediate fetch
3. **Line 92-93 in data_fetcher.go**: Immediate `fetchAllContests()` call
4. **Line 294-319 in data_fetcher.go**: Contest discovery for all sports

### Constraints:
- RapidAPI Basic plan: 20 requests/day limit
- Multiple external APIs with different rate limits
- Database operations must remain consistent
- Existing scheduled jobs should continue working
- WebSocket functionality depends on data availability

## TECHNICAL REQUIREMENTS:

### Backend Requirements:
- [x] Add startup configuration environment variables
- [x] Implement deferred background task initialization
- [x] Add circuit breakers for external API calls
- [x] Create health check endpoints for startup phases
- [x] Add startup timing metrics and logging
- [x] Implement graceful fallback mechanisms
- [x] Add admin endpoints for manual sync operations

### Frontend Requirements:
- [ ] Update health check polling to use new endpoints
- [ ] Add loading states for when background services are initializing
- [ ] Handle cases where initial data may not be immediately available

### Infrastructure Requirements:
- [x] New environment variables for startup control
- [x] Docker configuration updates for health checks
- [x] Enhanced logging for startup phases
- [x] Monitoring for startup duration and success rates

## IMPLEMENTATION APPROACH:

### Phase 1: Foundation (Immediate Fixes)
1. **Environment Variable Controls**
   - `SKIP_INITIAL_GOLF_SYNC=true` - Skip golf sync on startup
   - `SKIP_INITIAL_DATA_FETCH=true` - Skip immediate data fetching
   - `SKIP_INITIAL_CONTEST_DISCOVERY=true` - Skip contest discovery on startup
   - `STARTUP_DELAY_SECONDS=30` - Configurable delay before background tasks

2. **Conditional Startup Logic**
   - Make all background operations optional
   - Add startup phase logging
   - Separate critical vs. background initialization

### Phase 2: Integration (Performance & Reliability)
1. **Circuit Breakers & Timeouts**
   - Add timeout controls for all external API calls
   - Implement retry logic with exponential backoff
   - Add fallback mechanisms when APIs are unavailable

2. **Health Check Endpoints**
   - `/health` - Basic server health (existing)
   - `/ready` - Indicates background services are ready
   - `/startup-status` - Detailed startup phase information

### Phase 3: Enhancement (Monitoring & Control)
1. **Admin Control Endpoints**
   - `POST /api/v1/admin/sync/golf` - Manual golf tournament sync
   - `POST /api/v1/admin/sync/contests` - Manual contest discovery
   - `GET /api/v1/admin/status` - Background service status

2. **Enhanced Monitoring**
   - Startup duration metrics
   - API call success/failure rates
   - Background job status tracking

## DOCUMENTATION:

- Environment variable configuration documentation
- Health check endpoint specifications
- Admin API documentation for manual operations
- Troubleshooting guide for startup issues
- Best practices for production deployment

## TESTING STRATEGY:

### Unit Tests:
- [x] Test startup configuration parsing
- [x] Test conditional initialization logic
- [x] Test circuit breaker functionality
- [x] Test health check endpoint responses

### Integration Tests:
- [x] Test startup with various environment configurations
- [x] Test behavior when external APIs are unavailable
- [x] Test manual sync operations through admin endpoints
- [x] Test health check accuracy during startup phases

### E2E Tests:
- [x] Full startup timing tests with different configurations
- [x] Production-like environment testing
- [x] Container startup and scaling tests

## POTENTIAL CHALLENGES & RISKS:

### Technical Challenges:
1. **Data Availability**: Initial app usage may have limited data
2. **Background Job Coordination**: Ensuring jobs don't interfere with each other
3. **API Rate Limits**: Managing limited RapidAPI quota across multiple instances
4. **Cache Warming**: Deciding when and how to warm caches

### Mitigation Strategies:
1. Implement smart cache-first strategies
2. Add database-first fallbacks for critical data
3. Use distributed locking for shared resources
4. Add comprehensive logging for debugging

### Breaking Changes:
- Environment variable defaults may change behavior
- Health check endpoints need to be integrated into deployment scripts
- Some data may not be immediately available on fresh deploys

## SUCCESS CRITERIA:

1. **Startup Time**: Backend startup completes in <5 seconds (critical path only)
2. **Reliability**: Startup succeeds even when external APIs are down
3. **Scalability**: Multiple instances can start simultaneously without conflicts
4. **Observability**: Clear visibility into startup phases and background job status
5. **Control**: Ability to manually trigger sync operations when needed
6. **Data Availability**: Core functionality works with cached/fallback data

## OTHER CONSIDERATIONS:

### Common AI Assistant Gotchas:
1. **Environment Variable Defaults**: Ensure backward compatibility
2. **Goroutine Management**: Proper cleanup on shutdown
3. **Database Connection Pooling**: Don't exhaust connections during bulk operations
4. **Cache Invalidation**: Clear strategy for when manual syncs occur
5. **Error Propagation**: Don't let background errors crash the main process

### Production Deployment:
- Use health checks for container orchestration
- Set appropriate timeouts for load balancers
- Configure monitoring alerts for background job failures
- Plan for graceful degradation when external services are down

## MONITORING & OBSERVABILITY:

### Metrics to Track:
- Startup duration (by phase)
- Background job success/failure rates
- External API call latency and success rates
- Database operation timing
- Cache hit/miss rates

### Logging Requirements:
- Structured startup phase logging
- External API request/response logging (with rate limit info)
- Background job execution timing
- Error context for failed operations

### Alerts to Set Up:
- Startup duration exceeding thresholds
- Background job failure rates
- External API rate limit approaching
- Database connection pool exhaustion

## ROLLBACK PLAN:

### Immediate Rollback:
1. Set all `SKIP_INITIAL_*` environment variables to `false`
2. Restart services to restore original behavior
3. Monitor for startup success rates

### Gradual Rollback:
1. Disable specific optimizations one by one
2. Monitor impact on startup time and reliability
3. Adjust environment variables as needed

### Validation Steps:
1. Verify core functionality works
2. Check that scheduled jobs continue operating
3. Confirm data freshness is acceptable
4. Validate external API integration still works