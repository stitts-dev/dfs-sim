package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// CacheService provides caching functionality for AI recommendations
type CacheService struct {
	client *redis.Client
	logger *logrus.Logger
	ctx    context.Context
}

// Cache TTL constants as defined in the PRP
const (
	PlayerInsightTTL      = 6 * time.Hour     // During active games
	HistoricalAnalysisTTL = 7 * 24 * time.Hour // Weekly refresh
	ModelResponseTTL      = 24 * time.Hour     // Daily refresh
	OwnershipSnapshotTTL  = 5 * time.Minute   // Real-time updates
	UserPreferencesTTL    = 1 * time.Hour     // Hourly refresh
	TournamentDataTTL     = 12 * time.Hour    // Twice daily refresh
	WeatherDataTTL        = 30 * time.Minute  // Weather changes frequently
	InjuryReportTTL       = 15 * time.Minute  // Injury status can change quickly
	NewsAlertTTL          = 10 * time.Minute  // News is time-sensitive
)

// NewCacheService creates a new cache service instance
func NewCacheService(redisClient *redis.Client, logger *logrus.Logger) *CacheService {
	return &CacheService{
		client: redisClient,
		logger: logger,
		ctx:    context.Background(),
	}
}

// buildCacheKey constructs consistent cache keys for AI recommendations
func (c *CacheService) buildCacheKey(elements ...string) string {
	return fmt.Sprintf("ai-recommendations:%s", strings.Join(elements, ":"))
}

// Set stores a value in cache with TTL
func (c *CacheService) Set(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	err = c.client.Set(c.ctx, key, data, ttl).Err()
	if err != nil {
		c.logger.WithError(err).WithField("key", key).Error("Failed to set cache value")
		return err
	}

	c.logger.WithFields(logrus.Fields{
		"key": key,
		"ttl": ttl.String(),
	}).Debug("Cached value successfully")

	return nil
}

// Get retrieves a value from cache
func (c *CacheService) Get(key string, dest interface{}) error {
	data, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("cache miss")
		}
		c.logger.WithError(err).WithField("key", key).Error("Failed to get cache value")
		return err
	}

	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		c.logger.WithError(err).WithField("key", key).Error("Failed to unmarshal cache value")
		return err
	}

	c.logger.WithField("key", key).Debug("Cache hit")
	return nil
}

// Delete removes a value from cache
func (c *CacheService) Delete(key string) error {
	err := c.client.Del(c.ctx, key).Err()
	if err != nil {
		c.logger.WithError(err).WithField("key", key).Error("Failed to delete cache value")
		return err
	}

	c.logger.WithField("key", key).Debug("Deleted cache value")
	return nil
}

// Exists checks if a key exists in cache
func (c *CacheService) Exists(key string) (bool, error) {
	count, err := c.client.Exists(c.ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SetWithNX sets a value only if the key doesn't exist (for distributed locking)
func (c *CacheService) SetWithNX(key string, value interface{}, ttl time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal cache value: %w", err)
	}

	success, err := c.client.SetNX(c.ctx, key, data, ttl).Result()
	if err != nil {
		c.logger.WithError(err).WithField("key", key).Error("Failed to set cache value with NX")
		return false, err
	}

	return success, nil
}

// Increment atomically increments a numeric value
func (c *CacheService) Increment(key string, delta int64) (int64, error) {
	result, err := c.client.IncrBy(c.ctx, key, delta).Result()
	if err != nil {
		c.logger.WithError(err).WithField("key", key).Error("Failed to increment cache value")
		return 0, err
	}

	return result, nil
}

// PlayerInsight caching methods
func (c *CacheService) SetPlayerInsight(playerID uint, contestID uint, insight interface{}) error {
	key := c.buildCacheKey("player", fmt.Sprintf("%d", playerID), "insight", fmt.Sprintf("%d", contestID))
	return c.Set(key, insight, PlayerInsightTTL)
}

func (c *CacheService) GetPlayerInsight(playerID uint, contestID uint, dest interface{}) error {
	key := c.buildCacheKey("player", fmt.Sprintf("%d", playerID), "insight", fmt.Sprintf("%d", contestID))
	return c.Get(key, dest)
}

// Ownership snapshot caching methods
func (c *CacheService) SetOwnershipSnapshot(contestID uint, snapshot interface{}) error {
	key := c.buildCacheKey("contest", fmt.Sprintf("%d", contestID), "ownership", "snapshot")
	return c.Set(key, snapshot, OwnershipSnapshotTTL)
}

func (c *CacheService) GetOwnershipSnapshot(contestID uint, dest interface{}) error {
	key := c.buildCacheKey("contest", fmt.Sprintf("%d", contestID), "ownership", "snapshot")
	return c.Get(key, dest)
}

// Model response caching methods
func (c *CacheService) SetModelResponse(promptHash string, response interface{}) error {
	key := c.buildCacheKey("model", "response", promptHash)
	return c.Set(key, response, ModelResponseTTL)
}

func (c *CacheService) GetModelResponse(promptHash string, dest interface{}) error {
	key := c.buildCacheKey("model", "response", promptHash)
	return c.Get(key, dest)
}

// Historical analysis caching methods
func (c *CacheService) SetHistoricalAnalysis(sport string, analysisType string, data interface{}) error {
	key := c.buildCacheKey("historical", sport, analysisType)
	return c.Set(key, data, HistoricalAnalysisTTL)
}

func (c *CacheService) GetHistoricalAnalysis(sport string, analysisType string, dest interface{}) error {
	key := c.buildCacheKey("historical", sport, analysisType)
	return c.Get(key, dest)
}

// User preferences caching methods
func (c *CacheService) SetUserPreferences(userID uint, preferences interface{}) error {
	key := c.buildCacheKey("user", fmt.Sprintf("%d", userID), "preferences")
	return c.Set(key, preferences, UserPreferencesTTL)
}

func (c *CacheService) GetUserPreferences(userID uint, dest interface{}) error {
	key := c.buildCacheKey("user", fmt.Sprintf("%d", userID), "preferences")
	return c.Get(key, dest)
}

// Real-time data caching methods
func (c *CacheService) SetWeatherData(location string, data interface{}) error {
	key := c.buildCacheKey("weather", location)
	return c.Set(key, data, WeatherDataTTL)
}

func (c *CacheService) GetWeatherData(location string, dest interface{}) error {
	key := c.buildCacheKey("weather", location)
	return c.Get(key, dest)
}

func (c *CacheService) SetInjuryReport(playerID uint, report interface{}) error {
	key := c.buildCacheKey("injury", fmt.Sprintf("%d", playerID))
	return c.Set(key, report, InjuryReportTTL)
}

func (c *CacheService) GetInjuryReport(playerID uint, dest interface{}) error {
	key := c.buildCacheKey("injury", fmt.Sprintf("%d", playerID))
	return c.Get(key, dest)
}

func (c *CacheService) SetNewsAlert(alertID string, alert interface{}) error {
	key := c.buildCacheKey("news", alertID)
	return c.Set(key, alert, NewsAlertTTL)
}

func (c *CacheService) GetNewsAlert(alertID string, dest interface{}) error {
	key := c.buildCacheKey("news", alertID)
	return c.Get(key, dest)
}

// Tournament data caching methods
func (c *CacheService) SetTournamentData(tournamentID string, data interface{}) error {
	key := c.buildCacheKey("tournament", tournamentID)
	return c.Set(key, data, TournamentDataTTL)
}

func (c *CacheService) GetTournamentData(tournamentID string, dest interface{}) error {
	key := c.buildCacheKey("tournament", tournamentID)
	return c.Get(key, dest)
}

// Bulk operations for efficiency
func (c *CacheService) SetMultiple(items map[string]interface{}, ttl time.Duration) error {
	pipe := c.client.Pipeline()
	
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
		}
		pipe.Set(c.ctx, key, data, ttl)
	}

	_, err := pipe.Exec(c.ctx)
	if err != nil {
		c.logger.WithError(err).Error("Failed to execute bulk cache set")
		return err
	}

	c.logger.WithField("count", len(items)).Debug("Bulk cached values successfully")
	return nil
}

func (c *CacheService) GetMultiple(keys []string) (map[string]interface{}, error) {
	pipe := c.client.Pipeline()
	
	for _, key := range keys {
		pipe.Get(c.ctx, key)
	}

	cmders, err := pipe.Exec(c.ctx)
	if err != nil && err != redis.Nil {
		c.logger.WithError(err).Error("Failed to execute bulk cache get")
		return nil, err
	}

	results := make(map[string]interface{})
	for i, cmder := range cmders {
		cmd := cmder.(*redis.StringCmd)
		val, err := cmd.Result()
		if err == nil {
			var data interface{}
			if json.Unmarshal([]byte(val), &data) == nil {
				results[keys[i]] = data
			}
		}
	}

	c.logger.WithFields(logrus.Fields{
		"requested": len(keys),
		"found":     len(results),
	}).Debug("Bulk cache get completed")

	return results, nil
}

// Cache warming methods
func (c *CacheService) WarmCache(contestID uint, playerIDs []uint) error {
	c.logger.WithField("contest_id", contestID).Info("Starting cache warming")

	// Pre-load common data that will be frequently accessed
	warmupKeys := []string{
		c.buildCacheKey("contest", fmt.Sprintf("%d", contestID), "ownership", "snapshot"),
		c.buildCacheKey("contest", fmt.Sprintf("%d", contestID), "weather"),
		c.buildCacheKey("contest", fmt.Sprintf("%d", contestID), "news"),
	}

	// Add player-specific cache keys
	for _, playerID := range playerIDs {
		warmupKeys = append(warmupKeys,
			c.buildCacheKey("player", fmt.Sprintf("%d", playerID), "insight", fmt.Sprintf("%d", contestID)),
			c.buildCacheKey("injury", fmt.Sprintf("%d", playerID)),
		)
	}

	// Check which keys exist
	existing, err := c.GetMultiple(warmupKeys)
	if err != nil {
		c.logger.WithError(err).Warn("Cache warming check failed")
	}

	c.logger.WithFields(logrus.Fields{
		"contest_id":   contestID,
		"total_keys":   len(warmupKeys),
		"cached_keys":  len(existing),
		"missing_keys": len(warmupKeys) - len(existing),
	}).Info("Cache warming completed")

	return nil
}

// Cleanup methods
func (c *CacheService) CleanupExpiredData() error {
	// Redis handles TTL expiration automatically, but we can do additional cleanup here
	pattern := c.buildCacheKey("*")
	
	keys, err := c.client.Keys(c.ctx, pattern).Result()
	if err != nil {
		return err
	}

	c.logger.WithField("key_count", len(keys)).Debug("Cache cleanup scan completed")
	return nil
}

// Invalidate cache for specific entities
func (c *CacheService) InvalidatePlayer(playerID uint) error {
	pattern := c.buildCacheKey("player", fmt.Sprintf("%d", playerID), "*")
	return c.deleteByPattern(pattern)
}

func (c *CacheService) InvalidateContest(contestID uint) error {
	pattern := c.buildCacheKey("contest", fmt.Sprintf("%d", contestID), "*")
	return c.deleteByPattern(pattern)
}

func (c *CacheService) InvalidateUser(userID uint) error {
	pattern := c.buildCacheKey("user", fmt.Sprintf("%d", userID), "*")
	return c.deleteByPattern(pattern)
}

// Helper method to delete keys by pattern
func (c *CacheService) deleteByPattern(pattern string) error {
	keys, err := c.client.Keys(c.ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		err = c.client.Del(c.ctx, keys...).Err()
		if err != nil {
			c.logger.WithError(err).WithField("pattern", pattern).Error("Failed to delete keys by pattern")
			return err
		}

		c.logger.WithFields(logrus.Fields{
			"pattern": pattern,
			"count":   len(keys),
		}).Debug("Deleted keys by pattern")
	}

	return nil
}

// Health check method
func (c *CacheService) IsHealthy() bool {
	err := c.client.Ping(c.ctx).Err()
	return err == nil
}

// Get cache statistics
func (c *CacheService) GetStats() (map[string]interface{}, error) {
	info, err := c.client.Info(c.ctx, "memory", "stats").Result()
	if err != nil {
		return nil, err
	}

	// Parse Redis info for relevant metrics
	stats := make(map[string]interface{})
	
	// Add AI recommendations specific metrics
	pattern := c.buildCacheKey("*")
	keys, err := c.client.Keys(c.ctx, pattern).Result()
	if err == nil {
		stats["ai_recommendation_keys"] = len(keys)
	}

	stats["redis_info"] = info
	
	return stats, nil
}