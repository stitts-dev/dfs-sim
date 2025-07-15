## FEATURE:

Extract Golf Services from Monolith for Production Deployment

## CONTEXT & MOTIVATION:

The current DFS optimizer monolith contains golf-specific functionality that needs to be extracted into independent microservices for production deployment. This extraction will enable independent scaling of golf data ingestion, optimization algorithms, and simulation engines while maintaining current authentication systems. The primary goal is to deploy production-ready golf simulation capabilities with proper service boundaries and improved performance isolation.

**PARALLEL DEVELOPMENT CONTEXT:**
This PRD focuses on extracting golf-specific services (data, optimization, simulation) from the existing monolith while maintaining current PostgreSQL and authentication systems. A separate parallel workstream (INITIAL_supabase_user_migration.md) handles user service migration to Supabase. Both workstreams are independent and can be executed simultaneously, with integration coordination after completion.

**SERVICE EXTRACTION STRATEGY:**
Extract services with minimal changes to existing business logic, preserving current database schemas and authentication flows to enable immediate production deployment.

## EXAMPLES:

- Golf tournament data loading in isolated service with RapidAPI rate limiting
- Optimization engine processing 156-player tournaments in dedicated containers
- Monte Carlo simulations running parallel workers without blocking other services
- Independent deployment of optimization service for algorithm updates
- Load balancing golf requests across multiple service instances

## CURRENT STATE ANALYSIS:

**Monolith Components to Extract:**
- Golf handlers: `backend/internal/api/handlers/golf.go`
- Golf services: `backend/internal/services/golf_*`
- Golf providers: `backend/internal/providers/rapidapi_golf.go`, `backend/internal/providers/espn_golf.go`
- Optimization engine: `backend/internal/optimizer/`, `backend/internal/simulator/`
- Golf models: Golf-specific database models and migrations

**Current Constraints:**
- RapidAPI rate limit of 20 requests/day for golf data
- Compute-intensive optimization blocking other API requests
- Single deployment unit preventing independent service scaling
- Shared database causing coupling between unrelated features

**Performance Issues to Address:**
- Golf tournament loading blocking user authentication requests
- Optimization algorithms consuming excessive memory in shared container
- RapidAPI rate limiting affecting service availability
- WebSocket optimization progress updates causing connection overhead

## TECHNICAL REQUIREMENTS:

### Golf Data Service Requirements:
- [ ] Extract golf handlers, models, and external API providers
- [ ] Maintain RapidAPI integration with existing rate limiting
- [ ] Preserve golf database schema and migrations
- [ ] Implement service health checks and metrics
- [ ] Setup independent deployment and scaling
- [ ] Maintain existing golf API contract

### Optimization Service Requirements:
- [ ] Extract optimization and simulation engines
- [ ] Implement Redis caching for optimization results
- [ ] Setup parallel worker pools for simulations
- [ ] Expose HTTP APIs for optimization requests
- [ ] Implement progress tracking and WebSocket updates
- [ ] Preserve existing algorithm accuracy and performance

### API Gateway Requirements:
- [ ] Route golf requests to dedicated golf service
- [ ] Route optimization requests to optimization service
- [ ] Maintain existing authentication and middleware
- [ ] Implement service discovery and health checks
- [ ] Setup WebSocket proxy for real-time updates
- [ ] Preserve existing API contracts and responses

### Infrastructure Requirements:
- [ ] Multi-service Docker Compose configuration
- [ ] NGINX load balancer for service routing
- [ ] Shared PostgreSQL access with service boundaries
- [ ] Redis caching layer for optimization service
- [ ] Service-to-service communication setup
- [ ] Independent service health monitoring

## IMPLEMENTATION APPROACH:

### Phase 1: Service Foundation (2 hours)
- Create Git worktree structure for independent services
- Setup shared types and utilities package
- Create base service template with health checks
- Setup Docker Compose with NGINX routing
- Configure environment management per service

### Phase 2: Golf Data Service Extraction (3 hours)
- Extract golf handlers and create standalone service
- Move golf models and database migration logic
- Setup RapidAPI and ESPN provider integrations
- Implement external API rate limiting and caching
- Create golf service with existing API contracts
- Test golf tournament data loading and sync

### Phase 3: Optimization Service Extraction (3 hours)
- Extract optimization and simulation engines
- Setup Redis caching for optimization results
- Implement HTTP APIs for optimization requests
- Setup parallel worker pools for Monte Carlo simulations
- Create WebSocket endpoints for progress updates
- Test optimization performance and result accuracy

### Phase 4: API Gateway Implementation (2 hours)
- Create API gateway with request routing logic
- Implement service discovery and health checks
- Setup NGINX configuration for load balancing
- Maintain existing authentication and middleware
- Setup WebSocket proxy for real-time features
- Test end-to-end request routing and responses

### Phase 5: Integration & Deployment (2 hours)
- Test inter-service communication and data flow
- Verify golf simulation end-to-end functionality
- Setup monitoring and logging for all services
- Deploy to production environment
- Performance testing and optimization
- Documentation and runbook creation

## SERVICE ARCHITECTURE DESIGN:

### Golf Data Service:
```
golf-data-service/
├── cmd/
│   └── server/main.go
├── internal/
│   ├── handlers/          # Golf API handlers
│   ├── models/           # Golf-specific models
│   ├── providers/        # RapidAPI, ESPN integrations
│   └── services/         # Golf business logic
├── migrations/           # Golf database migrations
└── Dockerfile
```

### Optimization Service:
```
optimization-service/
├── cmd/
│   └── server/main.go
├── internal/
│   ├── handlers/         # Optimization API handlers
│   ├── optimizer/        # Optimization algorithms
│   ├── simulator/        # Monte Carlo simulation
│   └── websocket/        # Progress update hub
├── pkg/
│   └── cache/           # Redis caching layer
└── Dockerfile
```

### API Gateway:
```
api-gateway/
├── cmd/
│   └── server/main.go
├── internal/
│   ├── handlers/         # Non-golf handlers
│   ├── middleware/       # Auth and routing middleware
│   ├── proxy/           # Service proxy logic
│   └── websocket/       # WebSocket hub management
├── config/
│   └── nginx.conf       # NGINX routing config
└── Dockerfile
```

## SERVICE COMMUNICATION:

### Request Routing:
```
Frontend → NGINX → API Gateway → Golf Data Service
                                → Optimization Service
```

### Database Access:
- Golf Data Service: golf_tournaments, golf_players, golf_courses tables
- Optimization Service: optimization_results, simulation_results tables  
- API Gateway: users, lineups, contests (non-golf) tables

### Caching Strategy:
- Golf Data Service: Redis cache for tournament data (TTL: 1 hour)
- Optimization Service: Redis cache for lineup results (TTL: 24 hours)
- API Gateway: Session and authentication caching

## DOCUMENTATION:

### Service Documentation:
- API contracts and OpenAPI specifications per service
- Service deployment and configuration guides
- Database schema ownership per service
- Inter-service communication protocols

### Operations Documentation:
- Docker Compose multi-service setup
- NGINX routing and load balancing configuration
- Service health check and monitoring setup
- Environment variable management per service

### Development Documentation:
- Git worktree workflow for independent development
- Shared package dependency management
- Service testing strategies and test data
- Local development environment setup

## TESTING STRATEGY:

### Unit Tests:
- [ ] Golf data service API endpoints and external integrations
- [ ] Optimization service algorithms and simulation accuracy
- [ ] API gateway routing logic and middleware functionality
- [ ] Service health check and metrics endpoints

### Integration Tests:
- [ ] Service-to-service communication and data flow
- [ ] Database access patterns and transaction boundaries
- [ ] External API integration with rate limiting validation
- [ ] WebSocket functionality through API gateway proxy

### Performance Tests:
- [ ] Golf tournament data loading under rate limits
- [ ] Optimization performance with 156 player tournaments
- [ ] Monte Carlo simulation scaling with worker pools
- [ ] Service response times under concurrent load

### E2E Tests:
- [ ] Complete golf simulation workflow across services
- [ ] Real-time optimization progress updates via WebSocket
- [ ] Service failure and recovery scenarios
- [ ] Cross-service authentication and authorization

## POTENTIAL CHALLENGES & RISKS:

**Technical Challenges:**
- Service discovery and request routing complexity
- Database connection pooling across multiple services
- WebSocket connection management through NGINX proxy
- RapidAPI rate limiting state sharing between service instances

**Performance Considerations:**
- Network latency between services for optimization requests
- Database query performance with service-specific access patterns
- Memory usage optimization across multiple containers
- Redis caching strategy for cross-service data sharing

**Operational Complexity:**
- Multi-service deployment coordination
- Service health monitoring and alerting
- Log aggregation and distributed tracing
- Environment configuration management per service

**Breaking Changes:**
- Internal API restructuring for service boundaries
- WebSocket endpoint changes for proxy routing
- Authentication token validation across services
- Database migration coordination between services

## SUCCESS CRITERIA:

- [ ] Golf tournament data loads in <5 seconds via dedicated service
- [ ] Optimization generates 50 lineups in <10 seconds in isolation
- [ ] Monte Carlo simulation (10k iterations) completes in <15 seconds
- [ ] All services deploy independently without affecting others
- [ ] RapidAPI rate limiting preserved and functioning correctly
- [ ] WebSocket real-time updates work through API gateway
- [ ] Service health checks respond correctly for monitoring
- [ ] Golf simulation end-to-end functionality maintained
- [ ] Production deployment completes within 8 hours
- [ ] No performance regression from monolith baseline

## OTHER CONSIDERATIONS:

**Database Strategy:**
- Maintain current PostgreSQL with logical service boundaries
- Service-specific connection pools and query optimization
- Future migration path to service-specific databases
- Transaction boundary management across service calls

**Service Boundaries:**
- Golf Data Service: owns golf_* tables and external API integrations
- Optimization Service: owns optimization/simulation results and algorithms
- API Gateway: owns users, lineups, general contests, and service orchestration

**Future Evolution:**
- Event-driven communication for real-time data synchronization
- Circuit breakers and service resilience patterns
- Horizontal scaling of optimization workers
- Database per service migration strategy

**Development Workflow:**
- Git worktrees for independent service development
- Shared utilities package for common functionality
- Service-specific environment configuration
- Independent CI/CD pipelines per service

## MONITORING & OBSERVABILITY:

**Service Health Metrics:**
- Service response times and error rates per endpoint
- Database connection pool utilization per service
- External API quota usage and rate limit adherence
- Service dependency health and availability

**Performance Metrics:**
- Golf data loading times and external API latency
- Optimization request processing times and queue length
- Monte Carlo simulation throughput and worker utilization
- WebSocket connection counts and message throughput

**Business Metrics:**
- Golf tournament data freshness and accuracy
- Optimization result quality and algorithm performance
- User engagement with golf simulation features
- Service usage patterns and capacity planning

## ROLLBACK PLAN:

**Immediate Rollback:**
- Switch NGINX routing back to monolithic backend
- Revert to single-container deployment within 5 minutes
- Preserve database state and user data consistency
- Maintain service availability during rollback process

**Graceful Rollback:**
- Drain optimization requests from dedicated service
- Complete in-flight golf data synchronization
- Export service logs and metrics for analysis
- Coordinate database transaction completion

**Prevention Measures:**
- Blue-green deployment strategy with health checks
- Automated rollback triggers on service failure thresholds
- Database migration reversibility and backup procedures
- Service health validation before traffic routing

**Post-Rollback Analysis:**
- Performance comparison between monolith and microservices
- Service failure root cause analysis
- Optimization opportunities for future extraction attempts
- Documentation updates based on rollback learnings