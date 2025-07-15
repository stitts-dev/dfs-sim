# AI Recommendations Service - Testing Guide

This document provides comprehensive testing instructions for the AI Recommendations Service.

## Table of Contents

- [Test Setup](#test-setup)
- [Unit Tests](#unit-tests)
- [Integration Tests](#integration-tests)
- [Validation Scripts](#validation-scripts)
- [Performance Testing](#performance-testing)
- [Test Data](#test-data)
- [Troubleshooting](#troubleshooting)

## Test Setup

### Prerequisites

- Go 1.21+
- PostgreSQL (for integration tests)
- Redis (for caching tests)
- Docker & Docker Compose (for containerized testing)
- Node.js (optional, for WebSocket testing)

### Environment Setup

1. **Local Testing Environment**:
   ```bash
   # Set environment variables
   export DATABASE_URL="postgres://postgres:postgres@localhost:5432/dfs_optimizer_test"
   export REDIS_URL="redis://localhost:6379/4"
   export CLAUDE_API_KEY="your-claude-api-key"
   export AI_RECOMMENDATIONS_SERVICE_URL="http://localhost:8084"
   export API_GATEWAY_URL="http://localhost:8080"
   ```

2. **Create Test Database**:
   ```bash
   createdb dfs_optimizer_test
   ```

3. **Start Required Services**:
   ```bash
   # Start Redis
   docker run -d -p 6379:6379 redis:7-alpine
   
   # Or use docker-compose
   docker-compose up -d redis
   ```

## Unit Tests

### Running Unit Tests

```bash
# Run all unit tests
cd services/ai-recommendations-service
go test ./... -v

# Run tests with coverage
go test ./... -v -cover -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out
```

### Test Categories

1. **Service Layer Tests** (`internal/services/*_test.go`):
   - Claude API client testing
   - Cache service testing
   - Ownership analyzer testing
   - Prompt builder testing

2. **Handler Tests** (`internal/api/handlers/*_test.go`):
   - HTTP endpoint testing
   - Request/response validation
   - Error handling testing

3. **Model Tests** (`internal/models/*_test.go`):
   - Data structure validation
   - JSON marshaling/unmarshaling

### Running Specific Test Suites

```bash
# Test Claude API client
go test ./internal/services -run TestClaudeClient -v

# Test API handlers
go test ./internal/api/handlers -run TestIntegration -v

# Test with race condition detection
go test ./... -race -v
```

## Integration Tests

### Full Service Integration Tests

```bash
# Run integration test suite
go test ./internal/api/handlers -run TestIntegrationSuite -v

# Run with test database
DATABASE_URL="postgres://postgres:postgres@localhost:5432/dfs_optimizer_test" \
go test ./internal/api/handlers -run TestIntegrationSuite -v
```

### Docker Integration Testing

```bash
# Build and test the service in Docker
docker-compose -f docker-compose.yml up -d ai-recommendations-service

# Wait for service to be ready
sleep 10

# Run validation script
./validate-ai-recommendations-service.sh
```

## Validation Scripts

### End-to-End Validation

The comprehensive validation script tests all service functionality:

```bash
# Make script executable (if not already)
chmod +x validate-ai-recommendations-service.sh

# Run validation with default settings
./validate-ai-recommendations-service.sh

# Run with custom configuration
AI_RECOMMENDATIONS_SERVICE_URL="http://localhost:8084" \
API_GATEWAY_URL="http://localhost:8080" \
TEST_USER_ID="550e8400-e29b-41d4-a716-446655440000" \
./validate-ai-recommendations-service.sh
```

### Validation Test Coverage

The script tests:

1. **Health Checks**:
   - Service health endpoint
   - Readiness checks
   - Metrics endpoint
   - API Gateway integration

2. **AI Recommendation Endpoints**:
   - Player recommendations
   - Lineup recommendations
   - Swap recommendations
   - Response validation

3. **Ownership Analysis**:
   - Ownership data retrieval
   - Leverage opportunities
   - Trend analysis
   - Historical data

4. **Analysis Features**:
   - Lineup analysis
   - Contest analysis
   - Trend insights

5. **WebSocket Functionality**:
   - Real-time connection testing
   - Message broadcasting

6. **Error Handling**:
   - Invalid input handling
   - Malformed request testing
   - Service unavailability scenarios

7. **Performance Metrics**:
   - Response time validation
   - Throughput testing

8. **Integration Testing**:
   - Service discovery
   - Redis connectivity
   - Database connectivity

## Performance Testing

### Load Testing

```bash
# Install Apache Bench (if not available)
# On macOS: brew install httpie
# On Ubuntu: apt-get install apache2-utils

# Test player recommendations endpoint
ab -n 100 -c 10 -T application/json -p test-data/player-request.json \
   http://localhost:8084/api/v1/recommendations/players

# Test ownership endpoint
ab -n 200 -c 20 http://localhost:8084/api/v1/ownership/123
```

### Memory and CPU Profiling

```bash
# Run service with profiling enabled
go run -tags debug cmd/server/main.go

# Generate CPU profile
go tool pprof http://localhost:8084/debug/pprof/profile?seconds=30

# Generate memory profile
go tool pprof http://localhost:8084/debug/pprof/heap
```

### Benchmark Tests

```bash
# Run benchmark tests
go test ./internal/services -bench=. -benchmem

# Run specific benchmarks
go test ./internal/services -bench=BenchmarkClaudeClient -benchmem
```

## Test Data

### Sample Request Payloads

Located in `test-config.json`:

- Golf player data for recommendations
- Contest configurations
- User preference settings
- Expected response formats

### Creating Test Data

```bash
# Generate test data using the provided script
go run scripts/generate-test-data.go

# Load test data into database
psql dfs_optimizer_test < test-data/sample-data.sql
```

### Mock Data for Offline Testing

The test suite includes comprehensive mocks for:

- Claude API responses
- Database queries
- Redis cache operations
- External service calls

## Test Configuration

### Configuration Files

- `test-config.json`: Comprehensive test configuration
- `.env.test`: Environment variables for testing
- `docker-compose.test.yml`: Docker configuration for testing

### Test Environment Variables

```bash
# Required for testing
export CLAUDE_API_KEY="your-test-api-key"
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/dfs_optimizer_test"
export REDIS_URL="redis://localhost:6379/4"

# Optional for enhanced testing
export TEST_USER_ID="550e8400-e29b-41d4-a716-446655440000"
export TEST_CONTEST_ID="123"
export ENABLE_DEBUG_LOGGING="true"
```

## Troubleshooting

### Common Issues

1. **Database Connection Errors**:
   ```bash
   # Check PostgreSQL is running
   pg_isready -h localhost -p 5432
   
   # Create test database if missing
   createdb dfs_optimizer_test
   ```

2. **Redis Connection Errors**:
   ```bash
   # Check Redis is running
   redis-cli ping
   
   # Start Redis if not running
   redis-server
   ```

3. **Claude API Errors**:
   ```bash
   # Verify API key is set
   echo $CLAUDE_API_KEY
   
   # Test API connectivity
   curl -H "Authorization: Bearer $CLAUDE_API_KEY" \
        https://api.anthropic.com/v1/messages
   ```

4. **Port Conflicts**:
   ```bash
   # Check if port 8084 is in use
   lsof -i :8084
   
   # Kill conflicting processes
   kill -9 $(lsof -t -i:8084)
   ```

### Debug Mode

```bash
# Run tests with debug output
LOG_LEVEL=debug go test ./... -v

# Run service with debug logging
LOG_LEVEL=debug go run cmd/server/main.go
```

### Test Isolation

```bash
# Run tests with fresh database
dropdb dfs_optimizer_test && createdb dfs_optimizer_test
go test ./... -v

# Run tests in isolation
go test ./internal/services -v -count=1
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: AI Recommendations Service Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: dfs_optimizer_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: '1.21'
    
    - name: Run tests
      env:
        DATABASE_URL: postgres://postgres:postgres@localhost:5432/dfs_optimizer_test
        REDIS_URL: redis://localhost:6379/4
      run: |
        cd services/ai-recommendations-service
        go test ./... -v -cover
        
    - name: Run validation script
      run: ./validate-ai-recommendations-service.sh
```

## Test Metrics and Reports

### Coverage Reports

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# View coverage summary
go tool cover -func=coverage.out
```

### Test Reports

```bash
# Generate JSON test report
go test ./... -json > test-results.json

# Generate XML test report (requires go-junit-report)
go install github.com/jstemmer/go-junit-report/v2@latest
go test ./... -v | go-junit-report > test-results.xml
```

## Additional Resources

- [Go Testing Package Documentation](https://pkg.go.dev/testing)
- [Testify Assertion Library](https://github.com/stretchr/testify)
- [Docker Compose Testing Guide](https://docs.docker.com/compose/testing/)
- [PostgreSQL Testing Best Practices](https://www.postgresql.org/docs/current/regress.html)