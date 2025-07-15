package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/services/api-gateway/internal/proxy"
	"github.com/stitts-dev/dfs-sim/shared/pkg/database"
)

type HealthHandler struct {
	db           *database.DB
	redis        *redis.Client
	serviceProxy *proxy.ServiceProxy
	logger       *logrus.Logger
}

func NewHealthHandler(db *database.DB, redis *redis.Client, serviceProxy *proxy.ServiceProxy, logger *logrus.Logger) *HealthHandler {
	return &HealthHandler{
		db:           db,
		redis:        redis,
		serviceProxy: serviceProxy,
		logger:       logger,
	}
}

type HealthResponse struct {
	Service   string                 `json:"service"`
	Status    string                 `json:"status"`
	Timestamp int64                  `json:"timestamp"`
	Version   string                 `json:"version,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

type ServiceStatus struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Status    string `json:"status"`
	Latency   string `json:"latency,omitempty"`
	Error     string `json:"error,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func (h *HealthHandler) GetHealth(c *gin.Context) {
	// Basic health check - just verify the gateway is running
	response := HealthResponse{
		Service:   "api-gateway",
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
		Version:   "1.0.0",
	}

	c.JSON(http.StatusOK, response)
}

func (h *HealthHandler) GetReady(c *gin.Context) {
	details := make(map[string]interface{})
	overall := "ready"

	// Check database connection
	if err := h.db.HealthCheck(); err != nil {
		details["database"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		overall = "not_ready"
	} else {
		details["database"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Check Redis connection
	if err := h.redis.Ping(c.Request.Context()).Err(); err != nil {
		details["redis"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		overall = "not_ready"
	} else {
		details["redis"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	response := HealthResponse{
		Service:   "api-gateway",
		Status:    overall,
		Timestamp: time.Now().Unix(),
		Details:   details,
	}

	statusCode := http.StatusOK
	if overall != "ready" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

func (h *HealthHandler) GetMetrics(c *gin.Context) {
	// Basic metrics - in a real implementation, you'd integrate with Prometheus or similar
	metrics := gin.H{
		"service":           "api-gateway",
		"uptime_seconds":    time.Since(time.Now()).Seconds(), // This would be tracked from startup
		"memory_usage_mb":   0,                                // Would get actual memory usage
		"goroutines":        0,                                // Would get actual goroutine count
		"requests_total":    0,                                // Would track request counts
		"requests_per_sec":  0,                                // Would calculate RPS
		"response_time_avg": 0,                                // Would track average response time
		"timestamp":         time.Now().Unix(),
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *HealthHandler) GetServiceStatus(c *gin.Context) {
	services := []ServiceStatus{
		h.checkService("golf-service", "http://golf-service:8081/health"),
		h.checkService("optimization-service", "http://optimization-service:8082/health"),
		h.checkService("user-service", "http://user-service:8083/health"),
	}

	overall := "healthy"
	for _, service := range services {
		if service.Status != "healthy" {
			overall = "degraded"
			break
		}
	}

	response := gin.H{
		"gateway_status": overall,
		"services":       services,
		"timestamp":      time.Now().Unix(),
	}

	c.JSON(http.StatusOK, response)
}

func (h *HealthHandler) GetCircuitBreakerStatus(c *gin.Context) {
	// Get circuit breaker status from service proxy
	status := h.serviceProxy.GetCircuitBreakerStatus()
	
	c.JSON(http.StatusOK, gin.H{
		"circuit_breakers": status,
		"timestamp":        time.Now().Unix(),
	})
}

func (h *HealthHandler) checkService(name, url string) ServiceStatus {
	start := time.Now()
	
	// Create a simple HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	resp, err := client.Get(url)
	latency := time.Since(start)
	
	status := ServiceStatus{
		Name:      name,
		URL:       url,
		Timestamp: time.Now().Unix(),
		Latency:   latency.String(),
	}
	
	if err != nil {
		status.Status = "unhealthy"
		status.Error = err.Error()
		return status
	}
	
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK {
		status.Status = "healthy"
	} else {
		status.Status = "unhealthy"
		status.Error = "Non-200 response code"
	}
	
	return status
}