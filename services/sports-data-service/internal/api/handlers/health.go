package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/services/sports-data-service/internal/services"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db             *database.DB
	redis          *redis.Client
	startupManager *services.StartupManager
	logger         *logrus.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(
	db *database.DB,
	redis *redis.Client,
	startupManager *services.StartupManager,
	logger *logrus.Logger,
) *HealthHandler {
	return &HealthHandler{
		db:             db,
		redis:          redis,
		startupManager: startupManager,
		logger:         logger,
	}
}

// GetHealth returns the basic health status
func (h *HealthHandler) GetHealth(c *gin.Context) {
	response := types.HealthStatus{
		Status:    "ok",
		Service:   "sports-data-service",
		Timestamp: time.Now(),
		Checks:    make(map[string]string),
	}

	// Check database connection
	if err := h.db.HealthCheck(); err != nil {
		response.Status = "unhealthy"
		response.Checks["database"] = "failed: " + err.Error()
	} else {
		response.Checks["database"] = "ok"
	}

	// Check Redis connection
	if err := h.redis.Ping(c.Request.Context()).Err(); err != nil {
		response.Status = "unhealthy"
		response.Checks["redis"] = "failed: " + err.Error()
	} else {
		response.Checks["redis"] = "ok"
	}

	statusCode := http.StatusOK
	if response.Status != "ok" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// GetReady returns the readiness status (includes startup dependencies)
func (h *HealthHandler) GetReady(c *gin.Context) {
	response := types.HealthStatus{
		Status:    "ready",
		Service:   "sports-data-service",
		Timestamp: time.Now(),
		Checks:    make(map[string]string),
	}

	// Check basic health first
	if err := h.db.HealthCheck(); err != nil {
		response.Status = "not_ready"
		response.Checks["database"] = "failed: " + err.Error()
	} else {
		response.Checks["database"] = "ok"
	}

	if err := h.redis.Ping(c.Request.Context()).Err(); err != nil {
		response.Status = "not_ready"
		response.Checks["redis"] = "failed: " + err.Error()
	} else {
		response.Checks["redis"] = "ok"
	}

	// Check startup manager status
	if h.startupManager != nil {
		statusMap := h.startupManager.GetStatus()
		
		// Extract the phase from the status map
		if phase, ok := statusMap["phase"].(string); ok {
			response.Checks["startup_phase"] = phase
			
			// Service is ready when critical services are started
			if phase == "starting" {
				response.Status = "not_ready"
			}
		} else {
			response.Checks["startup_phase"] = "unknown"
		}
	}

	statusCode := http.StatusOK
	if response.Status != "ready" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// GetMetrics returns basic metrics for monitoring
func (h *HealthHandler) GetMetrics(c *gin.Context) {
	metrics := map[string]interface{}{
		"service":   "sports-data-service",
		"timestamp": time.Now(),
		"uptime":    time.Since(time.Now()).Seconds(), // Would track actual uptime
	}

	// Add database connection metrics if available
	if sqlDB, err := h.db.DB.DB(); err == nil {
		stats := sqlDB.Stats()
		metrics["database"] = map[string]interface{}{
			"open_connections": stats.OpenConnections,
			"in_use":          stats.InUse,
			"idle":            stats.Idle,
		}
	}

	c.JSON(http.StatusOK, metrics)
}