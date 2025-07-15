package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/api/handlers"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/models"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/services"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
)

type IntegrationTestSuite struct {
	suite.Suite
	db     *gorm.DB
	router *gin.Engine
	cfg    *config.Config
	logger *logrus.Logger
}

func (suite *IntegrationTestSuite) SetupSuite() {
	// Use test database or in-memory SQLite for testing
	dsn := "host=localhost user=postgres password=postgres dbname=dfs_optimizer_test port=5432 sslmode=disable TimeZone=UTC"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		// Fallback to SQLite for CI/local testing without Postgres
		suite.T().Skip("PostgreSQL not available for integration tests")
		return
	}
	
	suite.db = db
	suite.cfg = &config.Config{
		ClaudeAPIKey:      "test-key",
		AIRateLimit:       5,
		AICacheExpiration: 3600,
	}
	
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.ErrorLevel)

	// Auto-migrate test tables
	err = suite.db.AutoMigrate(
		&models.AIRecommendation{},
		&models.OwnershipSnapshot{},
		&models.RecommendationFeedback{},
		&models.RealtimeDataPoint{},
		&models.LeverageOpportunity{},
		&models.PromptTemplate{},
		&models.ModelPerformance{},
		&models.UserAIPreferences{},
	)
	suite.Require().NoError(err)

	// Set up Gin router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	
	// Mock services for testing
	mockAIEngine := &MockAIEngine{}
	mockOwnershipAnalyzer := &MockOwnershipAnalyzer{}
	
	// Initialize handlers
	recommendationHandler := handlers.NewRecommendationHandler(
		suite.db,
		mockAIEngine,
		nil, // No WebSocket hub needed for HTTP tests
		suite.cfg,
		suite.logger,
	)
	
	ownershipHandler := handlers.NewOwnershipHandler(
		mockOwnershipAnalyzer,
		suite.cfg,
		suite.logger,
	)
	
	// Setup routes
	apiV1 := suite.router.Group("/api/v1")
	{
		apiV1.POST("/recommendations/players", recommendationHandler.GetPlayerRecommendations)
		apiV1.GET("/ownership/:contestId", ownershipHandler.GetOwnershipData)
		apiV1.GET("/ownership/:contestId/leverage", ownershipHandler.GetLeverageOpportunities)
	}
}

func (suite *IntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		// Clean up test data
		sqlDB, _ := suite.db.DB()
		sqlDB.Close()
	}
}

func (suite *IntegrationTestSuite) SetupTest() {
	// Clean up data before each test
	if suite.db != nil {
		suite.db.Exec("TRUNCATE ai_recommendations, ownership_snapshots, recommendation_feedback RESTART IDENTITY CASCADE")
	}
}

func (suite *IntegrationTestSuite) TestPlayerRecommendations_Success() {
	request := models.PlayerRecommendationRequest{
		ContestID:   123,
		Sport:       "golf",
		ContestType: "gpp",
		MaxPlayers:  6,
		Budget:      50000,
		Players: []models.PlayerRecommendation{
			{
				ID:              1,
				Name:            "Rory McIlroy",
				Position:        "G",
				Salary:          9500,
				ProjectedPoints: 65.0,
				Ownership:       25.5,
				Team:            "NIR",
			},
			{
				ID:              2,
				Name:            "Scottie Scheffler",
				Position:        "G",
				Salary:          10000,
				ProjectedPoints: 68.0,
				Ownership:       30.0,
				Team:            "USA",
			},
		},
		Preferences: models.UserPreferences{
			RiskTolerance:      "medium",
			OwnershipStrategy:  "balanced",
			OptimizationGoal:   "roi",
		},
	}

	jsonBody, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "/api/v1/recommendations/players", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "success", response["status"])
	assert.Contains(suite.T(), response, "data")
}

func (suite *IntegrationTestSuite) TestPlayerRecommendations_InvalidRequest() {
	// Test with invalid contest ID
	request := models.PlayerRecommendationRequest{
		ContestID: -1, // Invalid
		Sport:     "golf",
	}

	jsonBody, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "/api/v1/recommendations/players", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *IntegrationTestSuite) TestOwnershipData_Success() {
	req, _ := http.NewRequest("GET", "/api/v1/ownership/123", nil)
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "success", response["status"])
	assert.Contains(suite.T(), response, "data")
}

func (suite *IntegrationTestSuite) TestLeverageOpportunities_Success() {
	req, _ := http.NewRequest("GET", "/api/v1/ownership/123/leverage?contest_type=gpp&min_leverage_score=0.5", nil)
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "success", response["status"])
	assert.Contains(suite.T(), response, "data")
}

func (suite *IntegrationTestSuite) TestOwnershipData_InvalidContestID() {
	req, _ := http.NewRequest("GET", "/api/v1/ownership/invalid", nil)
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// Mock implementations for testing

type MockAIEngine struct{}

func (m *MockAIEngine) GeneratePlayerRecommendations(ctx context.Context, request models.PlayerRecommendationRequest) (*models.AIRecommendationResponse, error) {
	return &models.AIRecommendationResponse{
		RecommendationID: "test-123",
		Recommendations: []models.PlayerRecommendation{
			{
				ID:              1,
				Name:            "Rory McIlroy",
				Position:        "G",
				Salary:          9500,
				ProjectedPoints: 65.0,
				Ownership:       25.5,
				Team:            "NIR",
				RecommendationScore: 8.5,
				Reasoning:       "Strong recent form and favorable course history",
			},
		},
		ModelUsed:      "claude-sonnet-4",
		Confidence:     85.0,
		TokensUsed:     250,
		ResponseTimeMs: 1500,
		Analysis: models.RecommendationAnalysis{
			OverallStrategy: "Balanced approach with value plays",
			RiskAssessment:  "Medium risk with high upside potential",
			KeyInsights:     []string{"Weather favors longer hitters", "Recent form is strong"},
		},
	}, nil
}

func (m *MockAIEngine) GenerateLineupRecommendations(ctx context.Context, request models.LineupRecommendationRequest) (*models.AIRecommendationResponse, error) {
	return &models.AIRecommendationResponse{
		RecommendationID: "lineup-test-123",
		ModelUsed:        "claude-sonnet-4",
		Confidence:       80.0,
		TokensUsed:       500,
		ResponseTimeMs:   2000,
	}, nil
}

func (m *MockAIEngine) GenerateSwapRecommendations(ctx context.Context, request models.SwapRecommendationRequest) (*models.AIRecommendationResponse, error) {
	return &models.AIRecommendationResponse{
		RecommendationID: "swap-test-123",
		ModelUsed:        "claude-sonnet-4",
		Confidence:       75.0,
		TokensUsed:       150,
		ResponseTimeMs:   800,
	}, nil
}

type MockOwnershipAnalyzer struct{}

func (m *MockOwnershipAnalyzer) GetOwnershipInsights(contestID uint) (*services.OwnershipAnalysis, error) {
	return &services.OwnershipAnalysis{
		ContestID:    contestID,
		TotalPlayers: 100,
		AvgOwnership: 12.5,
		TopOwnedPlayers: []services.PlayerOwnership{
			{PlayerID: 1, Name: "Test Player", Ownership: 35.5, Trend: "rising"},
		},
		LowOwnedValues: []services.PlayerOwnership{
			{PlayerID: 2, Name: "Value Player", Ownership: 5.2, Trend: "stable"},
		},
		LastUpdated: "2024-01-01T12:00:00Z",
	}, nil
}

func (m *MockOwnershipAnalyzer) CalculateLeverageOpportunities(contestID uint, players []models.PlayerRecommendation, contestType string, lineups []models.LineupReference) ([]services.LeveragePlay, error) {
	return []services.LeveragePlay{
		{
			PlayerID:           1,
			PlayerName:         "Test Player",
			LeverageScore:      0.75,
			LeverageType:       "contrarian",
			CurrentOwnership:   8.5,
			ProjectedOwnership: 12.0,
			ValueRating:        8.2,
			Reasoning:          "Low ownership with high upside",
		},
	}, nil
}

func (m *MockOwnershipAnalyzer) GetOwnershipTrends(contestID uint, playerIDs []uint) ([]services.OwnershipTrend, error) {
	return []services.OwnershipTrend{
		{
			PlayerID:        1,
			PlayerName:      "Test Player",
			CurrentOwnership: 15.5,
			TrendDirection:  "rising",
			ChangePercent:   2.3,
			Volatility:      "medium",
		},
	}, nil
}

// Run the test suite
func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}