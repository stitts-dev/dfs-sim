package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/models"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/services"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/sirupsen/logrus"
)

// MockCacheService for testing
type MockCacheService struct {
	mock.Mock
}

func (m *MockCacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheService) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCacheService) Del(ctx context.Context, keys ...string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

func (m *MockCacheService) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheService) TTL(ctx context.Context, key string) (time.Duration, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(time.Duration), args.Error(1)
}

func TestClaudeClient_GenerateRecommendation(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		systemPrompt   string
		expectedError  bool
		expectedResult bool
	}{
		{
			name:           "Valid golf recommendation request",
			prompt:         "Analyze these golf players for a tournament: Player A (9500, 65.0 proj), Player B (8500, 58.0 proj). Which offers better value?",
			systemPrompt:   "You are a DFS golf expert providing strategic recommendations.",
			expectedError:  false,
			expectedResult: true,
		},
		{
			name:           "Empty prompt should fail",
			prompt:         "",
			systemPrompt:   "You are a DFS expert.",
			expectedError:  true,
			expectedResult: false,
		},
		{
			name:           "Long complex prompt",
			prompt:         "Analyze the following 20 players for a GPP tournament with correlation considerations and weather factors...",
			systemPrompt:   "You are a DFS expert with deep knowledge of player correlations and external factors.",
			expectedError:  false,
			expectedResult: true,
		},
	}

	// Set up test configuration
	cfg := &config.Config{
		ClaudeAPIKey:    "test-api-key",
		AIRateLimit:     5,
		AICacheExpiration: 3600,
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This would require actual Claude API key for integration testing
			// For unit testing, we would mock the HTTP client
			client := services.NewClaudeClient(cfg, logger)
			
			// Skip actual API calls in unit tests unless we have a test API key
			if cfg.ClaudeAPIKey == "test-api-key" {
				t.Skip("Skipping Claude API test - no test API key provided")
				return
			}

			result, err := client.GenerateRecommendation(context.Background(), tt.prompt, tt.systemPrompt)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, result.Response)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result.Response)
				assert.Greater(t, result.TokensUsed, 0)
				assert.Greater(t, result.ResponseTimeMs, int64(0))
			}
		})
	}
}

func TestClaudeClient_RateLimiting(t *testing.T) {
	cfg := &config.Config{
		ClaudeAPIKey:    "test-api-key",
		AIRateLimit:     2, // Low limit for testing
		AICacheExpiration: 3600,
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	client := services.NewClaudeClient(cfg, logger)

	// Skip if no test API key
	if cfg.ClaudeAPIKey == "test-api-key" {
		t.Skip("Skipping rate limiting test - no test API key provided")
		return
	}

	ctx := context.Background()
	prompt := "Quick test prompt"
	systemPrompt := "You are a DFS expert."

	// Make rapid requests to test rate limiting
	start := time.Now()
	for i := 0; i < 3; i++ {
		_, err := client.GenerateRecommendation(ctx, prompt, systemPrompt)
		if i >= 2 { // Should hit rate limit after 2 requests
			// Rate limiting should introduce delay
			elapsed := time.Since(start)
			assert.True(t, elapsed > time.Second, "Rate limiting should introduce delay")
		}
		assert.NoError(t, err) // Should not error, just delay
	}
}

func TestClaudeClient_CircuitBreaker(t *testing.T) {
	cfg := &config.Config{
		ClaudeAPIKey:    "invalid-key", // Invalid key to trigger failures
		AIRateLimit:     5,
		AICacheExpiration: 3600,
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	client := services.NewClaudeClient(cfg, logger)
	ctx := context.Background()

	// Make multiple failed requests to trip circuit breaker
	for i := 0; i < 6; i++ {
		_, err := client.GenerateRecommendation(ctx, "test", "test")
		assert.Error(t, err) // Should fail with invalid key
	}

	// Circuit breaker should now be open - this should fail quickly
	start := time.Now()
	_, err := client.GenerateRecommendation(ctx, "test", "test")
	elapsed := time.Since(start)
	
	assert.Error(t, err)
	assert.True(t, elapsed < time.Second, "Circuit breaker should fail fast")
}

func TestClaudeClient_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		ClaudeAPIKey:    "test-api-key",
		AIRateLimit:     5,
		AICacheExpiration: 3600,
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	client := services.NewClaudeClient(cfg, logger)
	
	health := client.HealthCheck(context.Background())
	
	// Should return health status regardless of API key validity
	assert.NotNil(t, health)
	assert.Contains(t, []string{"healthy", "degraded", "unhealthy"}, health.Status)
	assert.NotEmpty(t, health.Details)
}