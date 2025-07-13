## FEATURE:

Rapid Microservices Transition for Golf Simulation Production Deployment

## CONTEXT & MOTIVATION:

The current DFS optimizer monolith needs to be split into microservices for improved scalability, deployment independence, and team productivity. The primary goal is to deploy a production-ready golf simulation system today while establishing a clean microservices foundation for future growth. This transition will enable independent scaling of compute-intensive services, reduce deployment risks, and improve code maintainability.

**PARALLEL DEVELOPMENT CONTEXT:**
This PRD focuses on extracting golf-specific services from the existing monolith while maintaining current user authentication. A separate parallel workstream (INITIAL_supabase_user_migration.md) handles migrating user authentication and data to Supabase. The two workstreams are designed to be independent and can be executed by separate agents simultaneously, with integration coordination happening after both are complete.

**SERVICE COORDINATION:**
- Golf services will maintain current PostgreSQL for contest/player/optimization data
- User services will migrate to Supabase for authentication and user-specific data
- API Gateway will support both authentication systems during migration phase
- Final integration will connect Supabase users to golf optimization services

## EXAMPLES:

- Golf tournament optimization with 156 players taking <5 seconds
- Monte Carlo simulation of 100,000 iterations completing in <10 seconds  
- Independent deployment of optimization engine without affecting data services
- Load balancing optimization requests across multiple worker instances

## CURRENT STATE ANALYSIS:

**Existing Monolith Structure:**
- Single Go backend with all functionality in `backend/internal/`
- Shared PostgreSQL database with all entities
- Single Docker container deployment
- All services tightly coupled through direct function calls

**Components to Split:**
- Golf data management (`handlers/golf.go`, `services/golf_*`, `providers/rapidapi_golf.go`)
- Optimization engine (`internal/optimizer/`, `internal/simulator/`)
- Authentication and routing (`api/router.go`, `middleware/auth.go`)
- External API providers with rate limiting

**Current Constraints:**
- RapidAPI rate limit of 20 requests/day for golf data
- Compute-intensive optimization algorithms blocking other requests
- Monolithic deployment preventing independent scaling
- Shared database causing data coupling

## TECHNICAL REQUIREMENTS:

### Backend Requirements:
- [ ] Golf Data Service: Extract golf handlers, models, and providers
- [ ] Optimization Service: Move optimization and simulation engines
- [ ] API Gateway Service: Maintain auth, routing, and non-golf endpoints
- [ ] Service discovery and health check endpoints
- [ ] HTTP APIs for inter-service communication
- [ ] Shared database access with service boundaries
- [ ] Rate limiting preservation for external APIs
- [ ] WebSocket hub for real-time optimization progress

### Frontend Requirements:
- [ ] Update API client to route requests through gateway
- [ ] Maintain existing golf simulation UI components
- [ ] Handle service-specific error states
- [ ] Real-time updates via WebSocket connection to gateway
- [ ] No breaking changes to existing user flows

### Infrastructure Requirements:
- [ ] NGINX load balancer configuration
- [ ] Multi-service Docker Compose setup
- [ ] Environment variable management per service
- [ ] Service-to-service authentication
- [ ] Health check and monitoring setup
- [ ] Redis caching for optimization service
- [ ] PostgreSQL shared database with service schemas

## IMPLEMENTATION APPROACH:

### Phase 1: Foundation (1 hour)
- Create monorepo structure with Git worktrees
- Setup parent docker-compose.yml with NGINX
- Configure environment management for multiple services
- Create shared types/utilities package

### Phase 2: Golf Data Service (2 hours)  
- Extract golf handlers, models, and providers
- Create standalone service with health checks
- Implement external API management with rate limiting
- Setup database connection and migrations
- Test golf tournament data loading and sync

### Phase 3: Optimization Service (2 hours)
- Move optimization engine and simulator to separate service
- Setup Redis caching layer for optimization results
- Expose HTTP APIs for optimization and simulation
- Implement parallel processing with worker pools
- Test optimization performance and accuracy

### Phase 4: API Gateway (2 hours)
- Implement request routing to downstream services  
- Maintain authentication and WebSocket functionality
- Setup NGINX load balancing configuration
- Handle service discovery and health checks
- Test end-to-end request flow

### Phase 5: Integration & Deployment (1 hour)
- Test inter-service communication
- Verify golf simulation end-to-end functionality
- Deploy to production environment
- Monitor service health and performance
- Setup alerts and logging

## DOCUMENTATION:

- Docker Compose configuration with service definitions
- NGINX routing configuration
- Service API contracts and OpenAPI specs
- Environment variable documentation per service
- Deployment and rollback procedures
- Service health check endpoints
- Inter-service authentication setup

## TESTING STRATEGY:

### Unit Tests:
- [ ] Golf data service API endpoints
- [ ] Optimization service algorithms
- [ ] API gateway routing logic
- [ ] Service health check functionality

### Integration Tests:
- [ ] Service-to-service communication
- [ ] Database access from each service
- [ ] External API integration with rate limiting
- [ ] WebSocket functionality through gateway

### E2E Tests:
- [ ] Complete golf simulation workflow
- [ ] Multi-service optimization request
- [ ] User authentication across services
- [ ] Real-time updates and WebSocket communication

## POTENTIAL CHALLENGES & RISKS:

**Technical Challenges:**
- Service discovery and routing complexity
- Database transaction boundaries across services
- WebSocket connection management through proxy
- Rate limiting state sharing between instances

**Dependencies:**
- NGINX configuration expertise
- Docker networking between services
- PostgreSQL connection pooling limits
- Redis availability for optimization caching

**Performance Concerns:**
- Network latency between services
- Database connection overhead per service
- Optimization request routing delays
- Memory usage across multiple containers

**Breaking Changes:**
- API endpoint restructuring
- Authentication token validation changes
- WebSocket connection endpoint changes

## SUCCESS CRITERIA:

- [ ] Golf tournament data loads within 5 seconds
- [ ] Optimization generates 50 lineups in <10 seconds
- [ ] Monte Carlo simulation (10k iterations) completes in <15 seconds
- [ ] All services deploy independently without downtime
- [ ] Frontend functionality remains unchanged
- [ ] RapidAPI rate limiting preserved and functioning
- [ ] WebSocket real-time updates working
- [ ] Service health checks responding correctly
- [ ] Production deployment completes within 8 hours

## OTHER CONSIDERATIONS:

**Service Boundaries:**
- Golf Data Service owns golf-specific tables
- Optimization Service handles compute-only operations
- API Gateway manages users, lineups, and general contests

**Future Migration Path:**
- Database per service in next iteration
- Event-driven communication for data sync
- Circuit breakers for service resilience
- Horizontal scaling of optimization workers

**Development Workflow:**
- Git worktrees for independent service development
- Shared utilities package for common code
- Service-specific environment configuration
- Independent CI/CD pipelines per service

## MONITORING & OBSERVABILITY:

**Logging Requirements:**
- Structured logging with correlation IDs across services
- Request tracing through service boundaries
- Rate limiting and quota tracking logs
- Optimization performance metrics

**Metrics to Track:**
- Service response times and error rates
- Database connection pool utilization
- Optimization request queue length
- External API quota usage
- Memory and CPU utilization per service

**Alerts to Set Up:**
- Service health check failures
- Database connection pool exhaustion
- RapidAPI quota threshold warnings
- Optimization request timeout alerts

## ROLLBACK PLAN:

**Immediate Rollback:**
- Revert to monolithic deployment within 5 minutes
- Switch NGINX routing back to single backend
- Preserve database state during rollback

**Graceful Rollback:**
- Drain optimization requests from service
- Complete in-flight golf data syncs
- Export service logs for debugging
- Maintain data consistency across rollback

**Prevention Measures:**
- Blue-green deployment strategy
- Database migration reversibility  
- Service health check requirements
- Automated rollback triggers on failure thresholds