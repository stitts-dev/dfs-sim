package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"

	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// ServiceProxy handles proxying requests to microservices
type ServiceProxy struct {
	golfClient              *ServiceClient
	optimizationClient      *ServiceClient
	userClient              *ServiceClient
	aiRecommendationsClient *ServiceClient
	realtimeClient          *ServiceClient
	circuitBreakers         map[string]*gobreaker.CircuitBreaker
	logger                  *logrus.Logger
}

// ServiceClient represents an HTTP client for a specific service
type ServiceClient struct {
	baseURL    string
	httpClient *http.Client
	service    string
	logger     *logrus.Logger
}

// NewServiceProxy creates a new service proxy
func NewServiceProxy(cfg *config.Config, logger *logrus.Logger) *ServiceProxy {
	// Create HTTP clients for each service
	golfClient := NewServiceClient(cfg.GolfServiceURL, "sports-data-service", logger)
	optimizationClient := NewServiceClient(cfg.OptimizationServiceURL, "optimization-service", logger)
	userClient := NewServiceClient(cfg.UserServiceURL, "user-service", logger)
	aiRecommendationsClient := NewServiceClient(cfg.AIRecommendationsServiceURL, "ai-recommendations-service", logger)
	realtimeClient := NewServiceClient(cfg.RealtimeServiceURL, "realtime-service", logger)

	// Create circuit breakers for each service
	circuitBreakers := make(map[string]*gobreaker.CircuitBreaker)
	
	golfCB := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "sports-data-service",
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.WithFields(logrus.Fields{
				"service":    name,
				"from_state": from,
				"to_state":   to,
			}).Warn("Circuit breaker state changed")
		},
	})

	optimizationCB := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "optimization-service",
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second, // Longer timeout for optimization
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.WithFields(logrus.Fields{
				"service":    name,
				"from_state": from,
				"to_state":   to,
			}).Warn("Circuit breaker state changed")
		},
	})

	userCB := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "user-service",
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.WithFields(logrus.Fields{
				"service":    name,
				"from_state": from,
				"to_state":   to,
			}).Warn("Circuit breaker state changed")
		},
	})

	aiRecommendationsCB := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "ai-recommendations-service",
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second, // Longer timeout for AI processing
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.WithFields(logrus.Fields{
				"service":    name,
				"from_state": from,
				"to_state":   to,
			}).Warn("Circuit breaker state changed")
		},
	})

	realtimeCB := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "realtime-service",
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     15 * time.Second, // Medium timeout for real-time operations
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.WithFields(logrus.Fields{
				"service":    name,
				"from_state": from,
				"to_state":   to,
			}).Warn("Circuit breaker state changed")
		},
	})

	circuitBreakers["sports-data-service"] = golfCB
	circuitBreakers["optimization-service"] = optimizationCB
	circuitBreakers["user-service"] = userCB
	circuitBreakers["ai-recommendations-service"] = aiRecommendationsCB
	circuitBreakers["realtime-service"] = realtimeCB

	return &ServiceProxy{
		golfClient:              golfClient,
		optimizationClient:      optimizationClient,
		userClient:              userClient,
		aiRecommendationsClient: aiRecommendationsClient,
		realtimeClient:          realtimeClient,
		circuitBreakers:         circuitBreakers,
		logger:                  logger,
	}
}

// NewServiceClient creates a new HTTP client for a service
func NewServiceClient(baseURL, serviceName string, logger *logrus.Logger) *ServiceClient {
	return &ServiceClient{
		baseURL: baseURL,
		service: serviceName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     30 * time.Second,
			},
		},
		logger: logger,
	}
}

// ProxyGolfRequest proxies requests to the golf service
func (sp *ServiceProxy) ProxyGolfRequest(c *gin.Context) {
	sp.proxyRequest(c, sp.golfClient, "sports-data-service")
}

// ProxyOptimizationRequest proxies requests to the optimization service
func (sp *ServiceProxy) ProxyOptimizationRequest(c *gin.Context) {
	sp.proxyRequest(c, sp.optimizationClient, "optimization-service")
}

// ProxyUserRequest proxies requests to the user service
func (sp *ServiceProxy) ProxyUserRequest(c *gin.Context) {
	sp.proxyRequest(c, sp.userClient, "user-service")
}

// ProxyAIRecommendationsRequest proxies requests to the AI recommendations service
func (sp *ServiceProxy) ProxyAIRecommendationsRequest(c *gin.Context) {
	sp.proxyRequest(c, sp.aiRecommendationsClient, "ai-recommendations-service")
}

// ProxyRealtimeRequest proxies requests to the realtime service
func (sp *ServiceProxy) ProxyRealtimeRequest(c *gin.Context) {
	sp.proxyRequest(c, sp.realtimeClient, "realtime-service")
}

// proxyRequest handles the actual request proxying with circuit breaker
func (sp *ServiceProxy) proxyRequest(c *gin.Context, client *ServiceClient, serviceName string) {
	cb := sp.circuitBreakers[serviceName]
	
	result, err := cb.Execute(func() (interface{}, error) {
		return client.ForwardRequest(c)
	})

	if err != nil {
		sp.logger.WithError(err).WithField("service", serviceName).Error("Service request failed")
		
		// Handle circuit breaker errors
		if err == gobreaker.ErrOpenState {
			c.JSON(http.StatusServiceUnavailable, types.ErrorResponse{
				Error: fmt.Sprintf("%s is currently unavailable", serviceName),
				Code:  "SERVICE_UNAVAILABLE",
				Details: map[string]string{
					"service": serviceName,
					"reason":  "circuit_breaker_open",
				},
			})
			return
		}

		c.JSON(http.StatusBadGateway, types.ErrorResponse{
			Error: fmt.Sprintf("Failed to communicate with %s", serviceName),
			Code:  "SERVICE_ERROR",
			Details: map[string]string{
				"service": serviceName,
				"error":   err.Error(),
			},
		})
		return
	}

	response := result.(*types.ServiceResponse)
	
	// Copy headers from service response
	for key, value := range response.Headers {
		c.Header(key, value)
	}

	c.JSON(response.StatusCode, response.Body)
}

// ForwardRequest forwards an HTTP request to the service
func (sc *ServiceClient) ForwardRequest(c *gin.Context) (*types.ServiceResponse, error) {
	// Build the target URL
	targetURL := fmt.Sprintf("%s%s", sc.baseURL, c.Request.URL.RequestURI())

	// Read request body
	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// Create new request
	req, err := http.NewRequestWithContext(
		c.Request.Context(),
		c.Request.Method,
		targetURL,
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers (exclude hop-by-hop headers)
	for key, values := range c.Request.Header {
		if !isHopByHopHeader(key) {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	// Add service identification header
	req.Header.Set("X-Forwarded-By", "api-gateway")
	req.Header.Set("X-Original-Host", c.Request.Host)

	// Execute request
	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request to %s: %w", sc.service, err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response body as JSON
	var body interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &body); err != nil {
			// If JSON parsing fails, return as string
			body = string(respBody)
		}
	}

	// Copy response headers
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if !isHopByHopHeader(key) && len(values) > 0 {
			headers[key] = values[0]
		}
	}

	sc.logger.WithFields(logrus.Fields{
		"service":     sc.service,
		"method":      c.Request.Method,
		"path":        c.Request.URL.Path,
		"status_code": resp.StatusCode,
	}).Debug("Forwarded request to service")

	return &types.ServiceResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    headers,
	}, nil
}

// GetServiceHealth checks the health of a specific service
func (sc *ServiceClient) GetServiceHealth(ctx context.Context) (*types.HealthStatus, error) {
	healthURL := fmt.Sprintf("%s/health", sc.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check service health: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service health check failed with status %d", resp.StatusCode)
	}

	var health types.HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}

	return &health, nil
}

// GetCircuitBreakerStatus returns the status of all circuit breakers
func (sp *ServiceProxy) GetCircuitBreakerStatus() map[string]interface{} {
	status := make(map[string]interface{})
	
	for name, cb := range sp.circuitBreakers {
		counts := cb.Counts()
		status[name] = map[string]interface{}{
			"state":           cb.State().String(),
			"requests":        counts.Requests,
			"total_successes": counts.TotalSuccesses,
			"total_failures":  counts.TotalFailures,
			"consecutive_successes": counts.ConsecutiveSuccesses,
			"consecutive_failures":  counts.ConsecutiveFailures,
		}
	}

	return status
}

// isHopByHopHeader checks if a header is a hop-by-hop header
func isHopByHopHeader(header string) bool {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}

	for _, h := range hopByHopHeaders {
		if header == h {
			return true
		}
	}
	return false
}