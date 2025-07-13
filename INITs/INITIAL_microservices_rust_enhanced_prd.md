## FEATURE:

Rust-Enhanced Microservices Architecture for High-Performance DFS Golf Simulation

## CONTEXT & MOTIVATION:

The DFS optimizer requires extreme computational performance for optimization algorithms and Monte Carlo simulations while maintaining rapid development velocity for API endpoints and data management. A hybrid Go/Rust microservices architecture leverages each language's strengths: Go's excellent web/database ecosystem for API services, and Rust's superior performance for compute-intensive workloads. This approach will deliver 5-10x performance improvements for core algorithms while maintaining development productivity.

The golf simulation workload involves complex mathematical operations:
- Knapsack optimization with correlation matrices (CPU-bound)
- Monte Carlo simulations with 100k+ iterations (parallel compute-bound) 
- Statistical analysis with large player datasets (memory-intensive)

Rust's zero-cost abstractions, SIMD capabilities, and lack of garbage collection make it ideal for these workloads, while Go remains perfect for HTTP APIs, database operations, and external service integrations.

## EXAMPLES:

**Performance Targets:**
- Golf lineup optimization (150 players, 50 lineups): 2-4 seconds (vs 8-12s in Go)
- Monte Carlo simulation (100k iterations): 5-8 seconds (vs 25-40s in Go)
- Correlation matrix calculation (150x150): 0.3-0.5 seconds (vs 2-3s in Go)
- Memory usage for optimization: 50-100MB (vs 200-400MB in Go)

**Service Examples:**
- Rust optimization engine processing 500 optimization requests/minute
- Parallel Monte Carlo simulation using all CPU cores efficiently
- Go API gateway handling 10k+ requests/minute with sub-100ms latency
- Golf data service managing external API rate limits (20 req/day RapidAPI)

## CURRENT STATE ANALYSIS:

**Existing Go Monolith Analysis:**
- `internal/optimizer/`: Knapsack algorithms, correlation, stacking (→ Rust)
- `internal/simulator/`: Monte Carlo simulation engine (→ Rust)
- `internal/api/handlers/golf.go`: Golf data management (→ Go service)
- `internal/api/router.go`: Authentication, routing (→ Go gateway)
- `internal/providers/`: External API integrations (→ Go service)

**Performance Bottlenecks to Address:**
- Optimization algorithms blocking other requests (single-threaded)
- Monte Carlo simulations consuming excessive memory (GC pressure)
- Correlation matrix calculations taking 2-3 seconds
- Concurrent optimization requests causing memory spikes

**Integration Points:**
- Shared PostgreSQL database (initially, migrate to service-owned later)
- Redis caching for optimization results
- WebSocket hub for real-time progress updates
- External API rate limiting state management

## TECHNICAL REQUIREMENTS:

### Backend Requirements:
- [ ] **Rust Optimization Engine Service**: Port knapsack solver, correlation matrix, stacking
- [ ] **Rust Simulation Service**: Monte Carlo with Rayon parallelism, statistical distributions
- [ ] **Go Golf Data Service**: Tournament data, external APIs, database operations
- [ ] **Go API Gateway Service**: Authentication, routing, WebSocket hub, non-compute endpoints
- [ ] HTTP APIs for inter-service communication with JSON serialization
- [ ] Shared database access with service-specific table ownership
- [ ] Redis caching layer for Rust services
- [ ] Service health checks and graceful shutdown
- [ ] Request correlation IDs for distributed tracing

### Frontend Requirements:
- [ ] No changes required - API contracts remain identical
- [ ] Enhanced performance monitoring dashboard
- [ ] Real-time optimization progress indicators
- [ ] Service health status indicators
- [ ] Performance metrics visualization

### Infrastructure Requirements:
- [ ] Multi-language Docker containers (Go + Rust)
- [ ] NGINX load balancer with service routing
- [ ] Environment variable management per service
- [ ] Resource allocation (CPU/memory) per service type
- [ ] Container health checks and restart policies
- [ ] Inter-service networking configuration
- [ ] Development environment with hot reload

## IMPLEMENTATION APPROACH:

### Phase 1: Foundation & Go Services (2 hours)
**Setup Infrastructure:**
- Create monorepo structure with Git worktrees
- Configure parent docker-compose.yml with multi-language support
- Setup NGINX routing configuration
- Create shared types package for API contracts

**Extract Golf Data Service (Go):**
- Move golf handlers, models, providers to standalone service
- Implement health checks and graceful shutdown
- Configure external API rate limiting and caching
- Test tournament data loading and synchronization

**Create API Gateway (Go):**
- Implement service routing and authentication
- Maintain WebSocket hub functionality
- Setup request correlation and logging
- Configure upstream service discovery

### Phase 2: Rust Optimization Engine (2 hours)
**Port Core Algorithms:**
- Create Rust optimization-engine service with Cargo.toml
- Port knapsack solver with SIMD optimizations
- Implement correlation matrix calculations with `nalgebra`
- Add stacking algorithms with zero-copy data structures

**Dependencies & Performance:**
```toml
[dependencies]
tokio = { version = "1.0", features = ["full"] }
serde = { version = "1.0", features = ["derive"] }
nalgebra = "0.32"
rayon = "1.7"
reqwest = "0.11"
redis = "0.23"
```

**HTTP API Implementation:**
- Async HTTP server with `tokio` and `warp`/`axum`
- JSON serialization/deserialization
- Error handling and proper HTTP status codes
- Request validation and response caching

### Phase 3: Rust Simulation Service (2 hours)
**Monte Carlo Implementation:**
- Create simulation-service with parallel processing
- Implement statistical distributions with `statrs`
- Use Rayon for data-parallel simulation batches
- Memory-efficient result aggregation

**Performance Optimizations:**
- Worker pool architecture for concurrent simulations
- SIMD vectorization for statistical calculations
- Zero-allocation result collection
- Efficient random number generation

**Integration Features:**
- HTTP API matching existing simulation endpoints
- Progress reporting via HTTP polling
- Result caching with Redis integration
- Graceful handling of large simulation batches

### Phase 4: Integration & Testing (1.5 hours)
**Service Communication:**
- Test HTTP API contracts between all services
- Implement proper error handling and retries
- Setup service discovery and health monitoring
- Configure request timeouts and circuit breakers

**End-to-End Validation:**
- Complete golf optimization workflow
- Monte Carlo simulation with progress tracking
- Authentication flow through gateway
- WebSocket real-time updates
- Performance benchmarking vs monolith

### Phase 5: Deployment & Monitoring (0.5 hours)
**Production Deployment:**
- Deploy multi-service stack to production
- Configure resource limits and auto-scaling
- Setup monitoring and alerting
- Verify performance improvements

## DOCUMENTATION:

**Rust Service Documentation:**
- Cargo.toml dependencies and feature flags
- Performance benchmarking methodology
- SIMD optimization techniques used
- Memory allocation patterns and profiling

**API Documentation:**
- OpenAPI specs for all Rust services
- Service discovery and routing configuration
- Error codes and handling strategies
- Request/response examples with curl commands

**Deployment Documentation:**
- Multi-language Docker build process
- Resource allocation guidelines
- Service scaling recommendations
- Health check endpoint specifications

## TESTING STRATEGY:

### Unit Tests:
- [ ] Rust optimization algorithms with property-based testing
- [ ] Monte Carlo simulation accuracy and performance
- [ ] Go service API endpoints and business logic
- [ ] Service health check functionality

### Integration Tests:
- [ ] HTTP communication between Go and Rust services
- [ ] Database access patterns and connection pooling
- [ ] Redis caching behavior across services
- [ ] External API integration with rate limiting

### Performance Tests:
- [ ] Optimization engine throughput and latency
- [ ] Monte Carlo simulation scalability
- [ ] Memory usage profiling under load
- [ ] Concurrent request handling capacity

### E2E Tests:
- [ ] Complete golf simulation workflow
- [ ] Multi-service optimization request flow
- [ ] Real-time WebSocket updates through gateway
- [ ] Service failure and recovery scenarios

## POTENTIAL CHALLENGES & RISKS:

**Technical Challenges:**
- Rust learning curve for algorithm porting
- JSON serialization performance between services
- Memory management differences between Go and Rust
- Debugging distributed requests across language boundaries

**Performance Risks:**
- Network latency overhead between services
- Serialization/deserialization costs
- Rust compilation time during development
- Resource contention between compute services

**Integration Complexity:**
- Error handling consistency across languages
- Logging and tracing correlation
- Service discovery and routing configuration
- Development environment complexity

**Migration Risks:**
- Algorithm correctness during Rust porting
- Floating-point precision differences
- Race conditions in parallel processing
- Breaking changes in optimization results

## SUCCESS CRITERIA:

**Performance Benchmarks:**
- [ ] Optimization requests: <4 seconds for 50 lineups (vs 8-12s baseline)
- [ ] Monte Carlo simulations: <8 seconds for 100k iterations (vs 25-40s baseline)
- [ ] Memory usage: <100MB for optimization (vs 200-400MB baseline)
- [ ] Concurrent throughput: 500+ optimization requests/minute

**Functional Requirements:**
- [ ] All existing golf simulation features working
- [ ] API compatibility maintained for frontend
- [ ] Real-time WebSocket updates functioning
- [ ] External API rate limiting preserved
- [ ] Authentication and authorization working

**Operational Requirements:**
- [ ] All services deploy independently
- [ ] Health checks responding correctly
- [ ] Monitoring and alerting functional
- [ ] Service discovery and routing working
- [ ] Graceful shutdown and restart capability

## OTHER CONSIDERATIONS:

**Rust-Specific Considerations:**
- Use `#[cfg(feature = "simd")]` for SIMD optimizations
- Implement proper error types with `thiserror`
- Use `tokio::spawn` for CPU-intensive tasks
- Profile with `perf` and `valgrind` for optimization

**Development Workflow:**
- Rust services with `cargo watch` for hot reload
- Shared API contracts in separate repository/package
- Performance regression testing in CI/CD
- Cross-language debugging strategies

**Future Migration Path:**
- Database per service with event sourcing
- gRPC for lower-latency inter-service communication
- WebAssembly compilation for browser-side optimization
- Kubernetes deployment with horizontal pod autoscaling

## MONITORING & OBSERVABILITY:

**Rust Service Metrics:**
- Algorithm execution time percentiles
- Memory allocation patterns and peak usage
- CPU utilization and SIMD instruction usage
- Request throughput and error rates

**Cross-Service Tracing:**
- Distributed tracing with correlation IDs
- Request flow visualization across services
- Performance bottleneck identification
- Error propagation tracking

**Performance Monitoring:**
- Optimization algorithm regression detection
- Monte Carlo simulation accuracy validation
- Resource utilization trends
- Service communication latency

**Alerts:**
- Rust service performance degradation
- Memory usage threshold violations
- Algorithm correctness validation failures
- Inter-service communication errors

## ROLLBACK PLAN:

**Immediate Rollback (< 5 minutes):**
- Switch NGINX routing back to monolithic Go service
- Preserve all optimization results and data
- Maintain service logs for debugging
- Keep Rust services running for comparison

**Graceful Migration Back:**
- Export optimization results from Rust services
- Validate algorithm correctness before switch
- Migrate cached data back to monolith
- Preserve performance benchmarking data

**Prevention Measures:**
- Comprehensive algorithm validation suite
- Performance regression testing
- Canary deployment with traffic splitting
- Automated rollback triggers on error thresholds

**Data Consistency:**
- Rust services remain stateless for easy rollback
- Database transactions handled by Go services
- Cache invalidation strategies for rollback
- Result validation between Rust and Go implementations