package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// AlertRateLimiter manages rate limiting for alert delivery to prevent spam
type AlertRateLimiter struct {
	redisClient     *redis.Client
	logger          *logrus.Logger
	defaultLimit    int           // Default alerts per hour per user
	windowDuration  time.Duration // Rate limiting window duration
	
	// User-specific limits
	userLimits      map[int]int   // user_id -> custom limit
	userLimitsMutex sync.RWMutex
	
	// Rate limiting statistics
	stats           *RateLimitStats
	statsMutex      sync.Mutex
}

// RateLimitStats tracks rate limiting metrics
type RateLimitStats struct {
	AlertsAllowed     int64     `json:"alerts_allowed"`
	AlertsBlocked     int64     `json:"alerts_blocked"`
	UniqueUsers       int       `json:"unique_users"`
	WindowResets      int64     `json:"window_resets"`
	LastResetTime     time.Time `json:"last_reset_time"`
	AverageUserUsage  float64   `json:"average_user_usage"`
	TopUserUsage      int       `json:"top_user_usage"`
}

// UserRateLimit represents a user's current rate limit status
type UserRateLimit struct {
	UserID       int       `json:"user_id"`
	AlertCount   int       `json:"alert_count"`
	Limit        int       `json:"limit"`
	WindowStart  time.Time `json:"window_start"`
	WindowEnd    time.Time `json:"window_end"`
	Remaining    int       `json:"remaining"`
	IsBlocked    bool      `json:"is_blocked"`
	ResetTime    time.Time `json:"reset_time"`
}

// RateLimitRule represents a rate limiting rule
type RateLimitRule struct {
	UserID       int           `json:"user_id"`
	RuleID       string        `json:"rule_id"`
	AlertType    string        `json:"alert_type"`
	Limit        int           `json:"limit"`
	Window       time.Duration `json:"window"`
	Priority     string        `json:"priority"`
	IsActive     bool          `json:"is_active"`
}

// NewAlertRateLimiter creates a new alert rate limiter
func NewAlertRateLimiter(redisClient *redis.Client, defaultLimit int, logger *logrus.Logger) *AlertRateLimiter {
	return &AlertRateLimiter{
		redisClient:    redisClient,
		logger:         logger,
		defaultLimit:   defaultLimit,
		windowDuration: time.Hour, // 1-hour windows by default
		userLimits:     make(map[int]int),
		stats:          &RateLimitStats{},
	}
}

// CanSendAlert checks if an alert can be sent to a user without exceeding rate limits
func (rl *AlertRateLimiter) CanSendAlert(userID int, ruleID string) bool {
	// Get user's rate limit
	limit := rl.getUserLimit(userID)
	
	// Check current usage
	usage, err := rl.getCurrentUsage(userID)
	if err != nil {
		rl.logger.WithError(err).WithField("user_id", userID).Error("Failed to check rate limit usage")
		// Allow the alert on error to avoid blocking legitimate alerts
		return true
	}
	
	// Check if user has exceeded their limit
	if usage >= limit {
		rl.incrementBlockedStats()
		rl.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"rule_id": ruleID,
			"usage":   usage,
			"limit":   limit,
		}).Debug("Alert blocked due to rate limit")
		return false
	}
	
	// Increment usage counter
	if err := rl.incrementUsage(userID); err != nil {
		rl.logger.WithError(err).WithField("user_id", userID).Error("Failed to increment usage counter")
	}
	
	rl.incrementAllowedStats()
	return true
}

// CanSendAlertWithPriority checks rate limits with priority consideration
func (rl *AlertRateLimiter) CanSendAlertWithPriority(userID int, ruleID string, priority string) bool {
	// High priority alerts may have higher limits or bypass certain restrictions
	if priority == "critical" || priority == "high" {
		// Use a higher limit for high-priority alerts
		limit := rl.getUserLimit(userID) * 2 // Double the limit for high-priority
		
		usage, err := rl.getCurrentUsage(userID)
		if err != nil {
			return true // Allow on error
		}
		
		if usage >= limit {
			// Even high-priority alerts have a limit
			rl.incrementBlockedStats()
			return false
		}
	}
	
	return rl.CanSendAlert(userID, ruleID)
}

// GetUserRateLimit returns the current rate limit status for a user
func (rl *AlertRateLimiter) GetUserRateLimit(userID int) (*UserRateLimit, error) {
	limit := rl.getUserLimit(userID)
	usage, err := rl.getCurrentUsage(userID)
	if err != nil {
		return nil, err
	}
	
	windowStart := rl.getCurrentWindowStart()
	windowEnd := windowStart.Add(rl.windowDuration)
	
	return &UserRateLimit{
		UserID:      userID,
		AlertCount:  usage,
		Limit:       limit,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		Remaining:   max(0, limit-usage),
		IsBlocked:   usage >= limit,
		ResetTime:   windowEnd,
	}, nil
}

// SetUserLimit sets a custom rate limit for a specific user
func (rl *AlertRateLimiter) SetUserLimit(userID int, limit int) {
	rl.userLimitsMutex.Lock()
	rl.userLimits[userID] = limit
	rl.userLimitsMutex.Unlock()
	
	rl.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"limit":   limit,
	}).Info("Set custom rate limit for user")
}

// RemoveUserLimit removes a custom rate limit for a user (reverts to default)
func (rl *AlertRateLimiter) RemoveUserLimit(userID int) {
	rl.userLimitsMutex.Lock()
	delete(rl.userLimits, userID)
	rl.userLimitsMutex.Unlock()
	
	rl.logger.WithField("user_id", userID).Info("Removed custom rate limit for user")
}

// getUserLimit returns the rate limit for a specific user
func (rl *AlertRateLimiter) getUserLimit(userID int) int {
	rl.userLimitsMutex.RLock()
	defer rl.userLimitsMutex.RUnlock()
	
	if limit, exists := rl.userLimits[userID]; exists {
		return limit
	}
	
	return rl.defaultLimit
}

// getCurrentUsage returns the current alert count for a user in the current window
func (rl *AlertRateLimiter) getCurrentUsage(userID int) (int, error) {
	ctx := context.Background()
	key := rl.getUserKey(userID)
	
	usageStr, err := rl.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil // No usage recorded yet
		}
		return 0, fmt.Errorf("failed to get usage from Redis: %w", err)
	}
	
	usage, err := strconv.Atoi(usageStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse usage value: %w", err)
	}
	
	return usage, nil
}

// incrementUsage increments the usage counter for a user
func (rl *AlertRateLimiter) incrementUsage(userID int) error {
	ctx := context.Background()
	key := rl.getUserKey(userID)
	
	// Use pipeline for atomic increment and expire
	pipe := rl.redisClient.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, rl.windowDuration)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to increment usage: %w", err)
	}
	
	return nil
}

// getUserKey generates a Redis key for a user's rate limit counter
func (rl *AlertRateLimiter) getUserKey(userID int) string {
	windowStart := rl.getCurrentWindowStart()
	windowID := windowStart.Unix() / int64(rl.windowDuration.Seconds())
	return fmt.Sprintf("alert_rate_limit:user:%d:window:%d", userID, windowID)
}

// getCurrentWindowStart returns the start time of the current rate limiting window
func (rl *AlertRateLimiter) getCurrentWindowStart() time.Time {
	now := time.Now()
	windowSeconds := int64(rl.windowDuration.Seconds())
	windowStart := (now.Unix() / windowSeconds) * windowSeconds
	return time.Unix(windowStart, 0)
}

// ResetUserLimit resets the rate limit counter for a specific user
func (rl *AlertRateLimiter) ResetUserLimit(userID int) error {
	ctx := context.Background()
	key := rl.getUserKey(userID)
	
	err := rl.redisClient.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to reset user limit: %w", err)
	}
	
	rl.logger.WithField("user_id", userID).Info("Reset rate limit for user")
	return nil
}

// GetRateLimitStats returns rate limiting statistics
func (rl *AlertRateLimiter) GetRateLimitStats() RateLimitStats {
	rl.statsMutex.Lock()
	defer rl.statsMutex.Unlock()
	
	stats := *rl.stats
	
	// Calculate additional metrics
	total := stats.AlertsAllowed + stats.AlertsBlocked
	if total > 0 {
		stats.AverageUserUsage = float64(stats.AlertsAllowed) / float64(stats.UniqueUsers)
	}
	
	return stats
}

// GetActiveUsers returns a list of users who have sent alerts in the current window
func (rl *AlertRateLimiter) GetActiveUsers() ([]UserRateLimit, error) {
	ctx := context.Background()
	
	// Find all rate limit keys in current window
	windowStart := rl.getCurrentWindowStart()
	windowID := windowStart.Unix() / int64(rl.windowDuration.Seconds())
	pattern := fmt.Sprintf("alert_rate_limit:user:*:window:%d", windowID)
	
	keys, err := rl.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get active user keys: %w", err)
	}
	
	users := make([]UserRateLimit, 0, len(keys))
	
	for _, key := range keys {
		// Extract user ID from key
		var userID int
		if _, err := fmt.Sscanf(key, "alert_rate_limit:user:%d:window:%d", &userID, &windowID); err != nil {
			continue
		}
		
		userLimit, err := rl.GetUserRateLimit(userID)
		if err != nil {
			rl.logger.WithError(err).WithField("user_id", userID).Warn("Failed to get user rate limit")
			continue
		}
		
		users = append(users, *userLimit)
	}
	
	return users, nil
}

// CleanupExpiredWindows removes expired rate limit data
func (rl *AlertRateLimiter) CleanupExpiredWindows() error {
	ctx := context.Background()
	
	// Find all rate limit keys
	pattern := "alert_rate_limit:user:*:window:*"
	keys, err := rl.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get rate limit keys for cleanup: %w", err)
	}
	
	currentWindowID := rl.getCurrentWindowStart().Unix() / int64(rl.windowDuration.Seconds())
	expiredKeys := make([]string, 0)
	
	for _, key := range keys {
		// Extract window ID from key
		var userID, windowID int64
		if _, err := fmt.Sscanf(key, "alert_rate_limit:user:%d:window:%d", &userID, &windowID); err != nil {
			continue
		}
		
		// Remove windows older than 24 hours
		if currentWindowID-windowID > 24 {
			expiredKeys = append(expiredKeys, key)
		}
	}
	
	if len(expiredKeys) > 0 {
		err := rl.redisClient.Del(ctx, expiredKeys...).Err()
		if err != nil {
			return fmt.Errorf("failed to delete expired keys: %w", err)
		}
		
		rl.logger.WithField("deleted_keys", len(expiredKeys)).Info("Cleaned up expired rate limit windows")
	}
	
	return nil
}

// SetRateLimitRule sets a specific rate limiting rule
func (rl *AlertRateLimiter) SetRateLimitRule(rule RateLimitRule) error {
	// Store rule in Redis for persistence
	ctx := context.Background()
	ruleKey := fmt.Sprintf("rate_limit_rules:user:%d:rule:%s", rule.UserID, rule.RuleID)
	
	ruleJSON, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("failed to marshal rate limit rule: %w", err)
	}
	
	err = rl.redisClient.Set(ctx, ruleKey, ruleJSON, 0).Err() // No expiration
	if err != nil {
		return fmt.Errorf("failed to store rate limit rule: %w", err)
	}
	
	rl.logger.WithFields(logrus.Fields{
		"user_id":    rule.UserID,
		"rule_id":    rule.RuleID,
		"alert_type": rule.AlertType,
		"limit":      rule.Limit,
	}).Info("Set rate limit rule")
	
	return nil
}

// StartCleanupWorker starts a background worker to clean up expired data
func (rl *AlertRateLimiter) StartCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Hour) // Cleanup every hour
	defer ticker.Stop()
	
	rl.logger.Info("Starting rate limiter cleanup worker")
	
	for {
		select {
		case <-ctx.Done():
			rl.logger.Info("Rate limiter cleanup worker stopped")
			return
		case <-ticker.C:
			if err := rl.CleanupExpiredWindows(); err != nil {
				rl.logger.WithError(err).Error("Failed to cleanup expired rate limit windows")
			}
		}
	}
}

// Statistics helpers
func (rl *AlertRateLimiter) incrementAllowedStats() {
	rl.statsMutex.Lock()
	rl.stats.AlertsAllowed++
	rl.statsMutex.Unlock()
}

func (rl *AlertRateLimiter) incrementBlockedStats() {
	rl.statsMutex.Lock()
	rl.stats.AlertsBlocked++
	rl.statsMutex.Unlock()
}

// Utility functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

