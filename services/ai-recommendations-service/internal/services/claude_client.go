package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
	"github.com/stitts-dev/dfs-sim/shared/pkg/config"
)

// ClaudeClient handles interaction with the Claude API
type ClaudeClient struct {
	httpClient     *http.Client
	cache          *CacheService
	logger         *logrus.Logger
	apiKey         string
	baseURL        string
	rateLimiter    *time.Ticker
	circuitBreaker *gobreaker.CircuitBreaker
	retryAttempts  int
	requestTracker *ClaudeRateLimitTracker
	mu             sync.Mutex
}

// ClaudeRateLimitTracker tracks API usage and limits
type ClaudeRateLimitTracker struct {
	mu               sync.Mutex
	requestsPerMinute int
	tokensPerHour    int64
	lastReset        time.Time
	hourlyTokens     int64
	minuteRequests   int
	requestLimit     int
	tokenLimit       int64
}

// ClaudeConfig represents configuration for Claude API requests
type ClaudeConfig struct {
	Model         string  `json:"model"`           // "claude-sonnet-4-20250514" or latest
	MaxTokens     int     `json:"max_tokens"`      // Dynamic based on complexity
	Temperature   float64 `json:"temperature"`     // Vary by recommendation type (0.3-0.7)
	TopP          float64 `json:"top_p"`           // Default 1.0
	TopK          int     `json:"top_k"`           // Default 0 (disabled)
	Stream        bool    `json:"stream"`          // Default false
	PromptCache   bool    `json:"prompt_cache"`    // Enable prompt caching for cost savings
}

// ClaudeMessage represents a message in the conversation
type ClaudeMessage struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"` // The message content
}

// ClaudeRequest represents the request payload for Claude API
type ClaudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	TopK        int             `json:"top_k,omitempty"`
	Messages    []ClaudeMessage `json:"messages"`
	Stream      bool            `json:"stream,omitempty"`
	System      string          `json:"system,omitempty"`
}

// ClaudeResponse represents the response from Claude API
type ClaudeResponse struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Role         string                 `json:"role"`
	Content      []ClaudeContentBlock   `json:"content"`
	Model        string                 `json:"model"`
	StopReason   string                 `json:"stop_reason"`
	StopSequence string                 `json:"stop_sequence"`
	Usage        ClaudeUsage            `json:"usage"`
}

// ClaudeContentBlock represents content blocks in the response
type ClaudeContentBlock struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// ClaudeUsage represents token usage information
type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ClaudeError represents API error response
type ClaudeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewClaudeClient creates a new Claude API client with rate limiting and circuit breaker
func NewClaudeClient(cfg *config.Config, logger *logrus.Logger) *ClaudeClient {
	// Initialize rate limit tracker
	tracker := &ClaudeRateLimitTracker{
		requestLimit:   60,    // 60 requests per minute
		tokenLimit:     100000, // 100k tokens per hour
		lastReset:      time.Now(),
	}

	// Setup circuit breaker
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "claude-api",
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 3
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.WithFields(logrus.Fields{
				"circuit":    name,
				"from_state": from.String(),
				"to_state":   to.String(),
			}).Info("Claude API circuit breaker state changed")
		},
	})

	return &ClaudeClient{
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Allow longer timeout for AI processing
		},
		logger:         logger,
		apiKey:         cfg.ClaudeAPIKey,
		baseURL:        "https://api.anthropic.com/v1",
		rateLimiter:    time.NewTicker(1 * time.Second), // 1 request per second (safe for 60/min)
		circuitBreaker: cb,
		retryAttempts:  3,
		requestTracker: tracker,
	}
}

// SendMessage sends a message to Claude API with rate limiting and circuit breaker
func (c *ClaudeClient) SendMessage(ctx context.Context, prompt string, systemPrompt string, config ClaudeConfig) (*ClaudeResponse, error) {
	// Check rate limits
	if err := c.checkRateLimits(); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Build request
	request := ClaudeRequest{
		Model:       config.Model,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
		TopP:        config.TopP,
		TopK:        config.TopK,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: config.Stream,
		System: systemPrompt,
	}

	// Use circuit breaker to make the request
	response, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.makeRequest(ctx, request)
	})

	if err != nil {
		return nil, fmt.Errorf("claude API request failed: %w", err)
	}

	claudeResponse := response.(*ClaudeResponse)

	// Track token usage
	c.trackTokenUsage(claudeResponse.Usage.InputTokens + claudeResponse.Usage.OutputTokens)

	return claudeResponse, nil
}

// makeRequest handles the actual HTTP request with retries
func (c *ClaudeClient) makeRequest(ctx context.Context, request ClaudeRequest) (*ClaudeResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Rate limiting
	<-c.rateLimiter.C

	// Track request
	if err := c.trackRequest(); err != nil {
		return nil, err
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < c.retryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(backoff)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewBuffer(requestBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set required headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		// Handle response
		if resp.StatusCode == http.StatusOK {
			var claudeResp ClaudeResponse
			if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			return &claudeResp, nil
		}

		// Handle error responses
		var claudeErr ClaudeError
		if err := json.NewDecoder(resp.Body).Decode(&claudeErr); err != nil {
			lastErr = fmt.Errorf("API request failed with status %d", resp.StatusCode)
			continue
		}

		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("invalid API credentials: %s", claudeErr.Message)
		case http.StatusTooManyRequests:
			lastErr = fmt.Errorf("rate limit exceeded: %s", claudeErr.Message)
		case http.StatusBadRequest:
			return nil, fmt.Errorf("bad request: %s", claudeErr.Message)
		case http.StatusInternalServerError:
			lastErr = fmt.Errorf("claude API error: %s", claudeErr.Message)
		default:
			lastErr = fmt.Errorf("unexpected error (status %d): %s", resp.StatusCode, claudeErr.Message)
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", c.retryAttempts, lastErr)
}

// checkRateLimits checks if we're within rate limits
func (c *ClaudeClient) checkRateLimits() error {
	c.requestTracker.mu.Lock()
	defer c.requestTracker.mu.Unlock()

	now := time.Now()

	// Reset counters if needed
	if now.Minute() != c.requestTracker.lastReset.Minute() {
		c.requestTracker.minuteRequests = 0
		c.requestTracker.lastReset = now
	}

	if now.Hour() != c.requestTracker.lastReset.Hour() {
		c.requestTracker.hourlyTokens = 0
	}

	// Check request limit
	if c.requestTracker.minuteRequests >= c.requestTracker.requestLimit {
		return fmt.Errorf("request rate limit exceeded (%d/%d per minute)",
			c.requestTracker.minuteRequests, c.requestTracker.requestLimit)
	}

	// Check token limit
	if c.requestTracker.hourlyTokens >= c.requestTracker.tokenLimit {
		return fmt.Errorf("token rate limit exceeded (%d/%d per hour)",
			c.requestTracker.hourlyTokens, c.requestTracker.tokenLimit)
	}

	return nil
}

// trackRequest tracks API requests
func (c *ClaudeClient) trackRequest() error {
	c.requestTracker.minuteRequests++

	c.logger.WithFields(logrus.Fields{
		"minute_requests": c.requestTracker.minuteRequests,
		"hourly_tokens":   c.requestTracker.hourlyTokens,
	}).Debug("Tracked Claude API request")

	return nil
}

// trackTokenUsage tracks token consumption
func (c *ClaudeClient) trackTokenUsage(tokens int) {
	c.requestTracker.mu.Lock()
	defer c.requestTracker.mu.Unlock()

	c.requestTracker.hourlyTokens += int64(tokens)

	c.logger.WithFields(logrus.Fields{
		"tokens_used":     tokens,
		"hourly_total":    c.requestTracker.hourlyTokens,
		"hourly_limit":    c.requestTracker.tokenLimit,
	}).Debug("Tracked Claude API token usage")
}

// GetUsageStats returns current usage statistics
func (c *ClaudeClient) GetUsageStats() (requestsPerMinute, tokensPerHour int64, requestLimit, tokenLimit int64) {
	c.requestTracker.mu.Lock()
	defer c.requestTracker.mu.Unlock()

	return int64(c.requestTracker.minuteRequests),
		c.requestTracker.hourlyTokens,
		int64(c.requestTracker.requestLimit),
		c.requestTracker.tokenLimit
}

// IsHealthy checks if the Claude API client is healthy
func (c *ClaudeClient) IsHealthy() bool {
	return c.circuitBreaker.State() == gobreaker.StateClosed
}

// GetCircuitBreakerState returns the current circuit breaker state
func (c *ClaudeClient) GetCircuitBreakerState() gobreaker.State {
	return c.circuitBreaker.State()
}

// BuildDefaultConfig returns default configuration for Claude requests
func (c *ClaudeClient) BuildDefaultConfig(recommendationType string) ClaudeConfig {
	config := ClaudeConfig{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 4000,
		TopP:      1.0,
		TopK:      0,
		Stream:    false,
	}

	// Adjust parameters based on recommendation type
	switch recommendationType {
	case "player_analysis":
		config.Temperature = 0.3 // Lower temperature for factual analysis
		config.MaxTokens = 2000
	case "strategy":
		config.Temperature = 0.5 // Medium temperature for strategic thinking
		config.MaxTokens = 3000
	case "ownership_insights":
		config.Temperature = 0.4 // Slightly higher for creative insights
		config.MaxTokens = 2500
	case "late_swap":
		config.Temperature = 0.2 // Very low for quick, decisive recommendations
		config.MaxTokens = 1500
	default:
		config.Temperature = 0.5 // Balanced default
	}

	return config
}

// GenerateRecommendations is a convenience method for generating DFS recommendations
func (c *ClaudeClient) GenerateRecommendations(ctx context.Context, prompt, systemPrompt string, recommendationType string) (*ClaudeResponse, error) {
	config := c.BuildDefaultConfig(recommendationType)
	return c.SendMessage(ctx, prompt, systemPrompt, config)
}

// SetCacheService sets the cache service for caching responses
func (c *ClaudeClient) SetCacheService(cache *CacheService) {
	c.cache = cache
}

// GetCachedResponse attempts to get a cached response for the given prompt
func (c *ClaudeClient) GetCachedResponse(promptHash string) (*ClaudeResponse, error) {
	if c.cache == nil {
		return nil, fmt.Errorf("cache service not configured")
	}

	cacheKey := fmt.Sprintf("claude:response:%s", promptHash)
	var response ClaudeResponse
	err := c.cache.Get(cacheKey, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// CacheResponse caches a Claude API response
func (c *ClaudeClient) CacheResponse(promptHash string, response *ClaudeResponse, ttl time.Duration) error {
	if c.cache == nil {
		return fmt.Errorf("cache service not configured")
	}

	cacheKey := fmt.Sprintf("claude:response:%s", promptHash)
	return c.cache.Set(cacheKey, response, ttl)
}