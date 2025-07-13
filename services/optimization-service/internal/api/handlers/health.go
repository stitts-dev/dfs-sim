package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// HealthHandler handles health check endpoints for optimization service
type HealthHandler struct {
	db     *database.DB
	redis  *redis.Client
	logger *logrus.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(
	db *database.DB,
	redis *redis.Client,
	logger *logrus.Logger,
) *HealthHandler {
	return &HealthHandler{
		db:     db,
		redis:  redis,
		logger: logger,
	}
}

// GetHealth returns the basic health status
func (h *HealthHandler) GetHealth(c *gin.Context) {
	response := types.HealthStatus{
		Status:    "ok",
		Service:   "optimization-service",
		Timestamp: time.Now(),
		Checks:    make(map[string]string),
	}

	// Check database connection (optional for optimization service)
	if h.db != nil {
		if err := h.db.HealthCheck(); err != nil {
			response.Status = "degraded"
			response.Checks["database"] = "failed: " + err.Error()
		} else {
			response.Checks["database"] = "ok"
		}
	} else {
		response.Checks["database"] = "not_configured"
	}

	// Check Redis connection (critical for optimization service)
	if err := h.redis.Ping(c.Request.Context()).Err(); err != nil {
		response.Status = "unhealthy"
		response.Checks["redis"] = "failed: " + err.Error()
	} else {
		response.Checks["redis"] = "ok"
	}

	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if response.Status == "degraded" {
		statusCode = http.StatusPartialContent
	}

	c.JSON(statusCode, response)
}

// GetReady returns the readiness status
func (h *HealthHandler) GetReady(c *gin.Context) {
	response := types.HealthStatus{
		Status:    "ready",
		Service:   "optimization-service",
		Timestamp: time.Now(),
		Checks:    make(map[string]string),
	}

	// Redis is critical for caching optimization results
	if err := h.redis.Ping(c.Request.Context()).Err(); err != nil {
		response.Status = "not_ready"
		response.Checks["redis"] = "failed: " + err.Error()
	} else {
		response.Checks["redis"] = "ok"
	}

	// Database check (if configured)
	if h.db != nil {
		if err := h.db.HealthCheck(); err != nil {
			response.Checks["database"] = "failed: " + err.Error()
			// Database failure doesn't make optimization service not ready
			// as it can still perform optimizations without database
		} else {
			response.Checks["database"] = "ok"
		}
	}

	statusCode := http.StatusOK
	if response.Status != "ready" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// GetMetrics returns optimization service metrics
func (h *HealthHandler) GetMetrics(c *gin.Context) {
	metrics := map[string]interface{}{
		"service":   "optimization-service",
		"timestamp": time.Now(),
		"uptime":    time.Since(time.Now()).Seconds(), // Would track actual uptime
	}

	// Add Redis metrics
	if info, err := h.redis.Info(c.Request.Context()).Result(); err == nil {
		metrics["redis"] = map[string]interface{}{
			"connected": true,
			"info_available": len(info) > 0,
		}
	}

	// Add cache metrics
	if dbSize, err := h.redis.DBSize(c.Request.Context()).Result(); err == nil {
		metrics["cache"] = map[string]interface{}{
			"total_keys": dbSize,
		}

		// Get optimization-specific metrics
		if optimizationKeys, err := h.redis.Keys(c.Request.Context(), "optimization:*").Result(); err == nil {
			metrics["optimization_cache"] = map[string]interface{}{
				"cached_results": len(optimizationKeys),
			}
		}

		if simulationKeys, err := h.redis.Keys(c.Request.Context(), "simulation:*").Result(); err == nil {
			metrics["simulation_cache"] = map[string]interface{}{
				"cached_results": len(simulationKeys),
			}
		}
	}

	// Add database connection metrics if available
	if h.db != nil {
		if sqlDB, err := h.db.DB.DB(); err == nil {
			stats := sqlDB.Stats()
			metrics["database"] = map[string]interface{}{
				"open_connections": stats.OpenConnections,
				"in_use":          stats.InUse,
				"idle":            stats.Idle,
			}
		}
	}

	c.JSON(http.StatusOK, metrics)
}