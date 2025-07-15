package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
)

type AdminHandler struct {
	startupManager *services.StartupManager
}

func NewAdminHandler(startupManager *services.StartupManager) *AdminHandler {
	return &AdminHandler{
		startupManager: startupManager,
	}
}

// TriggerGolfSync manually triggers golf tournament synchronization
func (a *AdminHandler) TriggerGolfSync(c *gin.Context) {
	if err := a.startupManager.TriggerGolfSync(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to trigger golf sync",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "started",
		"operation": "golf_sync",
		"message":   "Golf tournament sync has been triggered",
	})
}

// TriggerDataFetch manually triggers data fetching operations
func (a *AdminHandler) TriggerDataFetch(c *gin.Context) {
	if err := a.startupManager.TriggerDataFetch(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to trigger data fetch",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "started",
		"operation": "data_fetch",
		"message":   "Data fetch operation has been triggered",
	})
}

// GetSystemStatus returns comprehensive system status for admin monitoring
func (a *AdminHandler) GetSystemStatus(c *gin.Context) {
	status := a.startupManager.GetStatus()

	c.JSON(http.StatusOK, gin.H{
		"system_status": status,
		"admin_info": gin.H{
			"available_operations": []string{
				"POST /api/v1/admin/sync/golf - Trigger golf tournament sync",
				"POST /api/v1/admin/sync/data - Trigger data fetch operation",
				"GET /api/v1/admin/status - Get detailed system status",
			},
		},
	})
}
