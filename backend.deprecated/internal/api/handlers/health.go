package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
)

type HealthHandler struct {
	startupManager *services.StartupManager
}

func NewHealthHandler(startupManager *services.StartupManager) *HealthHandler {
	return &HealthHandler{
		startupManager: startupManager,
	}
}

// GetHealth returns basic health status - always returns 200 if server is running
// This is used for basic liveness probes
func (h *HealthHandler) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": "ok",
		"service":   "dfs-optimizer",
	})
}

// GetReady returns readiness status - only returns 200 when critical services are ready
// This is used for readiness probes in container orchestration
func (h *HealthHandler) GetReady(c *gin.Context) {
	if h.startupManager.IsReady() {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"phase":  string(h.startupManager.GetPhase()),
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"phase":  string(h.startupManager.GetPhase()),
		})
	}
}

// GetStartupStatus returns detailed startup status including background jobs
// This provides comprehensive visibility into startup phases and job status
func (h *HealthHandler) GetStartupStatus(c *gin.Context) {
	status := h.startupManager.GetStatus()

	// Return appropriate HTTP status based on phase
	phase := h.startupManager.GetPhase()
	switch phase {
	case services.PhaseCriticalReady, services.PhaseBackgroundInit, services.PhaseFullyReady:
		c.JSON(http.StatusOK, status)
	default:
		c.JSON(http.StatusServiceUnavailable, status)
	}
}
