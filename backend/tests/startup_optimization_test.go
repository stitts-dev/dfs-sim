package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/api/handlers"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/jstittsworth/dfs-optimizer/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestStartupConfiguration(t *testing.T) {
	// Test environment variable parsing works correctly
	os.Setenv("SKIP_INITIAL_GOLF_SYNC", "true")
	os.Setenv("SKIP_INITIAL_DATA_FETCH", "true")
	os.Setenv("STARTUP_DELAY_SECONDS", "5")

	cfg, err := config.LoadConfig()
	assert.NoError(t, err)
	assert.True(t, cfg.SkipInitialGolfSync)
	assert.True(t, cfg.SkipInitialDataFetch)
	assert.Equal(t, 5, cfg.StartupDelaySeconds)

	// Clean up
	os.Unsetenv("SKIP_INITIAL_GOLF_SYNC")
	os.Unsetenv("SKIP_INITIAL_DATA_FETCH")
	os.Unsetenv("STARTUP_DELAY_SECONDS")
}

func TestCircuitBreakerIntegration(t *testing.T) {
	// Test circuit breaker wraps external API calls
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests

	cb := services.NewCircuitBreakerService(5, 10*time.Second, logger)

	// Simulate successful call with configured service
	result, err := cb.Execute("rapidapi", func() (interface{}, error) {
		return "success", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "success", result)

	// Check state after successful call
	state := cb.GetState("rapidapi")
	counts := cb.GetCounts("rapidapi")
	assert.Equal(t, "closed", state.String())
	assert.Equal(t, uint32(1), counts.Requests)
	assert.Equal(t, uint32(0), counts.TotalFailures)
}

func TestHealthEndpoints(t *testing.T) {
	// Test health check endpoints return correct status
	cfg := &config.Config{
		SkipInitialGolfSync:     true,
		SkipInitialDataFetch:    true,
		StartupDelaySeconds:     0,
		ExternalAPITimeout:      10 * time.Second,
		CircuitBreakerThreshold: 5,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests

	// Create minimal startup manager for testing
	cb := services.NewCircuitBreakerService(cfg.CircuitBreakerThreshold, cfg.ExternalAPITimeout, logger)
	startupManager := services.NewStartupManager(cfg, logger, nil, nil, cb)

	// Start critical services
	err := startupManager.StartCriticalServices()
	assert.NoError(t, err)

	healthHandler := handlers.NewHealthHandler(startupManager)

	// Setup test router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", healthHandler.GetHealth)
	router.GET("/ready", healthHandler.GetReady)
	router.GET("/startup-status", healthHandler.GetStartupStatus)

	// Test /health endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response["status"])

	// Test /ready endpoint
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/ready", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ready", response["status"])

	// Test /startup-status endpoint
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/startup-status", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "startup_phase")
	assert.Contains(t, response, "background_jobs")
	assert.Contains(t, response, "external_services")
}

func TestAdminEndpoints(t *testing.T) {
	// Test admin endpoints
	cfg := &config.Config{
		SkipInitialGolfSync:     true,
		SkipInitialDataFetch:    true,
		StartupDelaySeconds:     0,
		ExternalAPITimeout:      10 * time.Second,
		CircuitBreakerThreshold: 5,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests

	cb := services.NewCircuitBreakerService(cfg.CircuitBreakerThreshold, cfg.ExternalAPITimeout, logger)
	startupManager := services.NewStartupManager(cfg, logger, nil, nil, cb)

	adminHandler := handlers.NewAdminHandler(startupManager)

	// Setup test router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/status", adminHandler.GetSystemStatus)
	router.POST("/admin/sync/golf", adminHandler.TriggerGolfSync)
	router.POST("/admin/sync/data", adminHandler.TriggerDataFetch)

	// Test /admin/status endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/status", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "system_status")
	assert.Contains(t, response, "admin_info")

	// Test manual golf sync trigger
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/admin/sync/golf", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "started", response["status"])
	assert.Equal(t, "golf_sync", response["operation"])
}
