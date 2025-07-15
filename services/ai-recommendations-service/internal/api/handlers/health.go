package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/services"
	"gorm.io/gorm"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db           *gorm.DB
	redisClient  *redis.Client
	claudeClient *services.ClaudeClient
	logger       *logrus.Logger
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Service   string                 `json:"service"`
	Version   string                 `json:"version"`
	Uptime    string                 `json:"uptime"`
	Checks    map[string]HealthCheck `json:"checks"`
}

// HealthCheck represents an individual health check
type HealthCheck struct {
	Status  string    `json:"status"`
	Message string    `json:"message,omitempty"`
	Latency string    `json:"latency,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// ReadinessResponse represents the readiness check response
type ReadinessResponse struct {
	Ready     bool      `json:"ready"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
	Checks    map[string]bool `json:"checks"`
}

// MetricsResponse represents service metrics
type MetricsResponse struct {
	Service          string                 `json:"service"`
	Timestamp        time.Time              `json:"timestamp"`
	DatabaseMetrics  DatabaseMetrics        `json:"database_metrics"`
	RedisMetrics     RedisMetrics           `json:"redis_metrics"`
	ClaudeMetrics    ClaudeMetrics          `json:"claude_metrics"`
	ServiceMetrics   ServiceMetrics         `json:"service_metrics"`
}

// DatabaseMetrics represents database health metrics
type DatabaseMetrics struct {
	Status           string        `json:"status"`
	OpenConnections  int           `json:"open_connections"`
	IdleConnections  int           `json:"idle_connections"`
	ConnectionLatency time.Duration `json:"connection_latency_ms"`
}

// RedisMetrics represents Redis health metrics
type RedisMetrics struct {
	Status           string        `json:"status"`
	ConnectedClients int           `json:"connected_clients"`
	UsedMemory       int64         `json:"used_memory_bytes"`
	CommandLatency   time.Duration `json:"command_latency_ms"`
	HitRate          float64       `json:"hit_rate_percent"`
}

// ClaudeMetrics represents Claude API health metrics
type ClaudeMetrics struct {
	Status            string  `json:"status"`
	CircuitBreakerState string `json:"circuit_breaker_state"`
	RequestsPerMinute int64   `json:"requests_per_minute"`
	TokensPerHour     int64   `json:"tokens_per_hour"`
	RequestLimit      int64   `json:"request_limit"`
	TokenLimit        int64   `json:"token_limit"`
	SuccessRate       float64 `json:"success_rate_percent"`
}

// ServiceMetrics represents general service metrics
type ServiceMetrics struct {
	StartTime           time.Time `json:"start_time"`
	UptimeSeconds       int64     `json:"uptime_seconds"`
	TotalRequests       int64     `json:"total_requests"`
	ActiveWebSockets    int       `json:"active_websockets"`
	RecommendationsGenerated int64 `json:"recommendations_generated"`
	CacheHitRate        float64   `json:"cache_hit_rate_percent"`
}

var startTime = time.Now()

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *gorm.DB, redisClient *redis.Client, claudeClient *services.ClaudeClient, logger *logrus.Logger) *HealthHandler {
	return &HealthHandler{
		db:           db,
		redisClient:  redisClient,
		claudeClient: claudeClient,
		logger:       logger,
	}
}

// GetHealth performs comprehensive health checks
func (h *HealthHandler) GetHealth(c *gin.Context) {
	h.logger.Debug("Health check requested")

	checks := make(map[string]HealthCheck)
	overallStatus := "healthy"

	// Check database health
	dbCheck := h.checkDatabase()
	checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// Check Redis health
	redisCheck := h.checkRedis()
	checks["redis"] = redisCheck
	if redisCheck.Status != "healthy" {
		overallStatus = "degraded"
	}

	// Check Claude API health
	claudeCheck := h.checkClaude()
	checks["claude_api"] = claudeCheck
	if claudeCheck.Status != "healthy" {
		overallStatus = "degraded"
	}

	// Check external dependencies
	externalCheck := h.checkExternalDependencies()
	checks["external_services"] = externalCheck
	if externalCheck.Status != "healthy" {
		overallStatus = "degraded"
	}

	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Service:   "ai-recommendations-service",
		Version:   "1.0.0",
		Uptime:    time.Since(startTime).String(),
		Checks:    checks,
	}

	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if overallStatus == "degraded" {
		statusCode = http.StatusOK // Still serving requests but with issues
	}

	c.JSON(statusCode, response)
}

// GetReady checks if the service is ready to serve requests
func (h *HealthHandler) GetReady(c *gin.Context) {
	h.logger.Debug("Readiness check requested")

	checks := map[string]bool{
		"database":    h.isDatabaseReady(),
		"redis":       h.isRedisReady(),
		"claude_api":  h.isClaudeReady(),
	}

	ready := true
	for _, check := range checks {
		if !check {
			ready = false
			break
		}
	}

	response := ReadinessResponse{
		Ready:     ready,
		Timestamp: time.Now(),
		Service:   "ai-recommendations-service",
		Checks:    checks,
	}

	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// GetMetrics returns detailed service metrics
func (h *HealthHandler) GetMetrics(c *gin.Context) {
	h.logger.Debug("Metrics requested")

	response := MetricsResponse{
		Service:         "ai-recommendations-service",
		Timestamp:       time.Now(),
		DatabaseMetrics: h.getDatabaseMetrics(),
		RedisMetrics:    h.getRedisMetrics(),
		ClaudeMetrics:   h.getClaudeMetrics(),
		ServiceMetrics:  h.getServiceMetrics(),
	}

	c.JSON(http.StatusOK, response)
}

// Individual health check methods

func (h *HealthHandler) checkDatabase() HealthCheck {
	start := time.Now()
	
	sqlDB, err := h.db.DB()
	if err != nil {
		return HealthCheck{
			Status:    "unhealthy",
			Message:   "Failed to get database instance",
			CheckedAt: time.Now(),
		}
	}

	if err := sqlDB.Ping(); err != nil {
		return HealthCheck{
			Status:    "unhealthy",
			Message:   "Database ping failed: " + err.Error(),
			CheckedAt: time.Now(),
		}
	}

	latency := time.Since(start)
	status := "healthy"
	if latency > 100*time.Millisecond {
		status = "slow"
	}

	return HealthCheck{
		Status:    status,
		Latency:   latency.String(),
		CheckedAt: time.Now(),
	}
}

func (h *HealthHandler) checkRedis() HealthCheck {
	start := time.Now()
	
	_, err := h.redisClient.Ping(context.Background()).Result()
	if err != nil {
		return HealthCheck{
			Status:    "unhealthy",
			Message:   "Redis ping failed: " + err.Error(),
			CheckedAt: time.Now(),
		}
	}

	latency := time.Since(start)
	status := "healthy"
	if latency > 50*time.Millisecond {
		status = "slow"
	}

	return HealthCheck{
		Status:    status,
		Latency:   latency.String(),
		CheckedAt: time.Now(),
	}
}

func (h *HealthHandler) checkClaude() HealthCheck {
	status := "healthy"
	message := ""

	if !h.claudeClient.IsHealthy() {
		status = "unhealthy"
		message = "Claude API circuit breaker is open"
	}

	// Check rate limits
	requestsPerMinute, tokensPerHour, requestLimit, tokenLimit := h.claudeClient.GetUsageStats()
	
	if requestsPerMinute >= requestLimit {
		status = "degraded"
		message = "Claude API request rate limit reached"
	} else if tokensPerHour >= tokenLimit {
		status = "degraded"
		message = "Claude API token rate limit reached"
	}

	return HealthCheck{
		Status:    status,
		Message:   message,
		CheckedAt: time.Now(),
	}
}

func (h *HealthHandler) checkExternalDependencies() HealthCheck {
	// Check if we can reach external services (placeholder)
	return HealthCheck{
		Status:    "healthy",
		CheckedAt: time.Now(),
	}
}

// Readiness check methods

func (h *HealthHandler) isDatabaseReady() bool {
	sqlDB, err := h.db.DB()
	if err != nil {
		return false
	}
	return sqlDB.Ping() == nil
}

func (h *HealthHandler) isRedisReady() bool {
	_, err := h.redisClient.Ping(context.Background()).Result()
	return err == nil
}

func (h *HealthHandler) isClaudeReady() bool {
	return h.claudeClient.IsHealthy()
}

// Metrics collection methods

func (h *HealthHandler) getDatabaseMetrics() DatabaseMetrics {
	metrics := DatabaseMetrics{
		Status: "healthy",
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		metrics.Status = "unhealthy"
		return metrics
	}

	stats := sqlDB.Stats()
	metrics.OpenConnections = stats.OpenConnections
	metrics.IdleConnections = stats.Idle

	// Measure connection latency
	start := time.Now()
	if err := sqlDB.Ping(); err != nil {
		metrics.Status = "unhealthy"
	} else {
		metrics.ConnectionLatency = time.Since(start)
	}

	return metrics
}

func (h *HealthHandler) getRedisMetrics() RedisMetrics {
	metrics := RedisMetrics{
		Status: "healthy",
	}

	// Measure command latency
	start := time.Now()
	_, err := h.redisClient.Ping(context.Background()).Result()
	if err != nil {
		metrics.Status = "unhealthy"
		return metrics
	}
	metrics.CommandLatency = time.Since(start)

	// Get Redis info
	_, err = h.redisClient.Info(context.Background(), "memory", "clients", "stats").Result()
	if err == nil {
		// TODO: Implement actual Redis info parsing instead of using placeholder values
		// Parse Redis info string to extract:
		// - connected_clients from Clients section
		// - used_memory from Memory section  
		// - keyspace_hits/keyspace_misses from Stats section to calculate hit rate
		metrics.ConnectedClients = 10  // Placeholder
		metrics.UsedMemory = 1024000   // Placeholder
		metrics.HitRate = 85.0         // Placeholder
	}

	return metrics
}

func (h *HealthHandler) getClaudeMetrics() ClaudeMetrics {
	requestsPerMinute, tokensPerHour, requestLimit, tokenLimit := h.claudeClient.GetUsageStats()
	
	status := "healthy"
	if !h.claudeClient.IsHealthy() {
		status = "unhealthy"
	}

	return ClaudeMetrics{
		Status:              status,
		CircuitBreakerState: h.claudeClient.GetCircuitBreakerState().String(),
		RequestsPerMinute:   requestsPerMinute,
		TokensPerHour:       tokensPerHour,
		RequestLimit:        requestLimit,
		TokenLimit:          tokenLimit,
		SuccessRate:         95.0, // Placeholder - would track actual success rate
	}
}

func (h *HealthHandler) getServiceMetrics() ServiceMetrics {
	uptime := time.Since(startTime)
	
	return ServiceMetrics{
		StartTime:                startTime,
		UptimeSeconds:            int64(uptime.Seconds()),
		TotalRequests:            1000,  // Placeholder - would track actual requests
		ActiveWebSockets:         5,     // Placeholder - would get from WebSocket hub
		RecommendationsGenerated: 500,   // Placeholder - would track actual recommendations
		CacheHitRate:             78.5,  // Placeholder - would get from cache service
	}
}