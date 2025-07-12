package services

import (
	"context"
	"testing"
	"time"

	"github.com/jstittsworth/dfs-optimizer/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCacheService for testing
type MockCacheService struct {
	mock.Mock
}

func (m *MockCacheService) Get(ctx context.Context, key string, dest interface{}) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}

func (m *MockCacheService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheService) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// TestFuzzyPlayerMatching tests the fuzzy matching functionality
func TestFuzzyPlayerMatching(t *testing.T) {
	// Setup test service
	service := &AIRecommendationService{
		logger: logrus.New(),
	}

	// Test data
	players := []models.Player{
		{
			ID:   1,
			Name: "Jon Rahm",
			Team: "Spain",
		},
		{
			ID:   2,
			Name: "Scottie Scheffler",
			Team: "USA",
		},
		{
			ID:   3,
			Name: "Tiger Woods",
			Team: "United States",
		},
		{
			ID:   4,
			Name: "Rory McIlroy Jr.",
			Team: "Northern Ireland",
		},
	}

	tests := []struct {
		name          string
		aiName        string
		aiTeam        string
		expectedID    uint
		minConfidence float64
		shouldMatch   bool
	}{
		{
			name:          "Exact match",
			aiName:        "Jon Rahm",
			aiTeam:        "Spain",
			expectedID:    1,
			minConfidence: 0.99,
			shouldMatch:   true,
		},
		{
			name:          "Initial variation",
			aiName:        "J. Rahm",
			aiTeam:        "Spain",
			expectedID:    1,
			minConfidence: 0.85,
			shouldMatch:   true,
		},
		{
			name:          "Team normalization",
			aiName:        "Tiger Woods",
			aiTeam:        "USA",
			expectedID:    3,
			minConfidence: 0.75,
			shouldMatch:   true,
		},
		{
			name:          "Junior suffix variation",
			aiName:        "Rory McIlroy",
			aiTeam:        "Northern Ireland",
			expectedID:    4,
			minConfidence: 0.90,
			shouldMatch:   true,
		},
		{
			name:          "Partial name match",
			aiName:        "S. Scheffler",
			aiTeam:        "USA",
			expectedID:    2,
			minConfidence: 0.80,
			shouldMatch:   true,
		},
		{
			name:          "No match - completely different",
			aiName:        "Random Player",
			aiTeam:        "Unknown",
			expectedID:    0,
			minConfidence: 0.0,
			shouldMatch:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player, confidence := service.fuzzyMatchPlayer(tt.aiName, tt.aiTeam, players)

			if tt.shouldMatch {
				assert.NotNil(t, player, "Expected to find a matching player")
				assert.Equal(t, tt.expectedID, player.ID, "Player ID should match")
				assert.GreaterOrEqual(t, confidence, tt.minConfidence, "Confidence should be above minimum")
			} else {
				assert.True(t, player == nil || confidence < 0.6, "Should not match with high confidence")
			}
		})
	}
}

// TestLevenshteinDistance tests the Levenshtein distance calculation
func TestLevenshteinDistance(t *testing.T) {
	service := &AIRecommendationService{
		logger: logrus.New(),
	}

	tests := []struct {
		name     string
		s1       string
		s2       string
		expected int
	}{
		{
			name:     "Identical strings",
			s1:       "hello",
			s2:       "hello",
			expected: 0,
		},
		{
			name:     "One character difference",
			s1:       "hello",
			s2:       "hallo",
			expected: 1,
		},
		{
			name:     "Length difference",
			s1:       "hello",
			s2:       "hell",
			expected: 1,
		},
		{
			name:     "Complete difference",
			s1:       "abc",
			s2:       "xyz",
			expected: 3,
		},
		{
			name:     "Empty strings",
			s1:       "",
			s2:       "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.levenshteinDistance(tt.s1, tt.s2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestInitialVariation tests initial variation detection
func TestInitialVariation(t *testing.T) {
	service := &AIRecommendationService{
		logger: logrus.New(),
	}

	tests := []struct {
		name1    string
		name2    string
		expected bool
	}{
		{"J. Smith", "John Smith", true},
		{"John Smith", "J. Smith", true},
		{"J. R. Smith", "John Robert Smith", true},
		{"John Smith", "Jane Smith", false},
		{"J. Smith", "Jim Smith", true}, // J could be short for Jim
		{"John", "J.", true},
	}

	for _, tt := range tests {
		t.Run(tt.name1+" vs "+tt.name2, func(t *testing.T) {
			result := service.isInitialVariation(tt.name1, tt.name2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSuffixVariation tests suffix variation detection
func TestSuffixVariation(t *testing.T) {
	service := &AIRecommendationService{
		logger: logrus.New(),
	}

	tests := []struct {
		name1    string
		name2    string
		expected bool
	}{
		{"John Smith Jr.", "John Smith", true},
		{"John Smith", "John Smith Jr.", true},
		{"John Smith Sr.", "John Smith", true},
		{"John Smith III", "John Smith", true},
		{"John Smith", "Jane Smith", false},
		{"John Jr. Smith", "John Smith", false}, // Jr. in middle
	}

	for _, tt := range tests {
		t.Run(tt.name1+" vs "+tt.name2, func(t *testing.T) {
			result := service.isSuffixVariation(tt.name1, tt.name2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTeamNormalization tests team name normalization
func TestTeamNormalization(t *testing.T) {
	service := &AIRecommendationService{
		logger: logrus.New(),
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"usa", "united states"}, // Function works on lowercase
		{"united states", "usa"},
		{"liv", "liv golf"},
		{"liv golf", "liv"},
		{"spain", "spain"}, // No mapping
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := service.normalizeTeam(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCacheKeyGeneration tests intelligent cache key generation
func TestCacheKeyGeneration(t *testing.T) {
	service := &AIRecommendationService{
		logger: logrus.New(),
	}

	req := PlayerRecommendationRequest{
		ContestID:       123,
		RemainingBudget: 45000,
		PositionsNeeded: []string{"G", "G"},
		OptimizeFor:     "ceiling",
		Sport:           "golf",
	}

	userID := 456

	// Test that similar requests generate same cache key
	key1 := service.generateCacheKey(req, userID)
	key2 := service.generateCacheKey(req, userID)
	assert.Equal(t, key1, key2, "Same request should generate same cache key")

	// Test that budget rounding works
	req.RemainingBudget = 45500 // Should round to same 45000
	key3 := service.generateCacheKey(req, userID)
	assert.Equal(t, key1, key3, "Budget rounding should make cache keys equal")

	// Test that different users generate different keys
	key4 := service.generateCacheKey(req, 789)
	assert.NotEqual(t, key1, key4, "Different users should generate different cache keys")

	// Test that different sports generate different keys
	req.Sport = "nba"
	key5 := service.generateCacheKey(req, userID)
	assert.NotEqual(t, key1, key5, "Different sports should generate different cache keys")
}

// TestPositionHashing tests position array hashing
func TestPositionHashing(t *testing.T) {
	service := &AIRecommendationService{
		logger: logrus.New(),
	}

	tests := []struct {
		name      string
		positions []string
		expected  string
	}{
		{
			name:      "Empty positions",
			positions: []string{},
			expected:  "all",
		},
		{
			name:      "Single position",
			positions: []string{"G"},
			expected:  "G",
		},
		{
			name:      "Multiple positions (sorted)",
			positions: []string{"G", "G", "G"},
			expected:  "G,G,G",
		},
		{
			name:      "Mixed positions",
			positions: []string{"QB", "RB", "WR"},
			expected:  "QB,RB,WR",
		},
		{
			name:      "Unsorted positions (should be sorted)",
			positions: []string{"WR", "QB", "RB"},
			expected:  "QB,RB,WR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.hashPositions(tt.positions)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCircuitBreakerStates tests circuit breaker functionality
func TestCircuitBreakerStates(t *testing.T) {
	service := &AIRecommendationService{
		logger: logrus.New(),
		circuitBreaker: &CircuitBreaker{
			state:     "closed",
			threshold: 2,
			timeout:   time.Minute,
		},
	}

	// Initially should allow calls
	assert.True(t, service.canMakeAPICall(), "Circuit breaker should be closed initially")

	// Record failures
	service.recordAPIFailure()
	assert.True(t, service.canMakeAPICall(), "Should still allow calls after first failure")

	service.recordAPIFailure()
	assert.False(t, service.canMakeAPICall(), "Should block calls after threshold reached")

	// Test that half-open state transitions to closed on success
	service.circuitBreaker.state = "half-open"
	service.recordAPISuccess()
	assert.True(t, service.canMakeAPICall(), "Success should reset circuit breaker")
}

// TestPlayerFormCalculation tests player form calculation
func TestPlayerFormCalculation(t *testing.T) {
	service := &AIRecommendationService{
		logger: logrus.New(),
	}

	tests := []struct {
		name            string
		projectedPoints float64
		salary          int
		expectedForm    string
	}{
		{
			name:            "Hot player",
			projectedPoints: 60.0,
			salary:          10000,
			expectedForm:    "Hot",
		},
		{
			name:            "Good player",
			projectedPoints: 45.0,
			salary:          10000,
			expectedForm:    "Good",
		},
		{
			name:            "Average player",
			projectedPoints: 35.0,
			salary:          10000,
			expectedForm:    "Average",
		},
		{
			name:            "Cold player",
			projectedPoints: 25.0,
			salary:          10000,
			expectedForm:    "Cold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := models.Player{
				ProjectedPoints: tt.projectedPoints,
				Salary:          tt.salary,
			}
			result := service.getPlayerForm(player)
			assert.Equal(t, tt.expectedForm, result)
		})
	}
}

// TestGolfPromptGeneration tests golf-specific prompt generation
func TestGolfPromptGeneration(t *testing.T) {
	service := &AIRecommendationService{
		logger: logrus.New(),
	}

	req := PlayerRecommendationRequest{
		Sport:        "golf",
		ContestType:  "GPP",
		OptimizeFor:  "ceiling",
		BeginnerMode: true,
	}

	contest := models.Contest{
		Name: "Test Golf Tournament",
	}

	players := []models.Player{
		{
			Name:            "Test Player",
			Team:            "USA",
			Salary:          10000,
			ProjectedPoints: 45.0,
			Ownership:       15.5,
		},
	}

	prompt := service.buildGolfRecommendationPrompt(req, contest, players)

	// Check that golf-specific content is included
	assert.Contains(t, prompt, "Golf DFS expert", "Should contain golf DFS expert reference")
	assert.Contains(t, prompt, "Strokes Gained", "Should contain strokes gained analysis")
	assert.Contains(t, prompt, "Course Fit", "Should contain course fit assessment")
	assert.Contains(t, prompt, "Cut Probability", "Should contain cut probability analysis")
	assert.Contains(t, prompt, "BEGINNER MODE", "Should contain beginner mode content")
	assert.Contains(t, prompt, "GPP: High ceiling", "Should contain GPP strategy for ceiling optimization")
	assert.Contains(t, prompt, "Test Player", "Should contain player information")
}
