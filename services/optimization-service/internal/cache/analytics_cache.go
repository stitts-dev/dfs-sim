package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stitts-dev/dfs-sim/services/optimization-service/internal/optimizer"
	"github.com/stitts-dev/dfs-sim/shared/types"
	"github.com/sirupsen/logrus"
)

// AnalyticsCache handles Redis caching for computed analytics
type AnalyticsCache struct {
	client     *redis.Client
	defaultTTL time.Duration
	keyPrefix  string
	logger     *logrus.Entry
}

// CacheConfig contains configuration for the analytics cache
type CacheConfig struct {
	RedisURL     string        `json:"redis_url"`
	Database     int           `json:"database"`
	DefaultTTL   time.Duration `json:"default_ttl"`
	KeyPrefix    string        `json:"key_prefix"`
	MaxRetries   int           `json:"max_retries"`
	PoolSize     int           `json:"pool_size"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
}

// CacheStats represents cache performance statistics
type CacheStats struct {
	Hits              int64         `json:"hits"`
	Misses            int64         `json:"misses"`
	Sets              int64         `json:"sets"`
	Deletes           int64         `json:"deletes"`
	Errors            int64         `json:"errors"`
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	HitRate           float64       `json:"hit_rate"`
	TotalSize         int64         `json:"total_size_bytes"`
	ActiveConnections int           `json:"active_connections"`
}

// NewAnalyticsCache creates a new analytics cache instance
func NewAnalyticsCache(config CacheConfig) (*AnalyticsCache, error) {
	// Parse Redis options
	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Override configuration
	opt.DB = config.Database
	opt.MaxRetries = config.MaxRetries
	opt.PoolSize = config.PoolSize
	opt.ReadTimeout = config.ReadTimeout
	opt.WriteTimeout = config.WriteTimeout

	// Create Redis client
	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	cache := &AnalyticsCache{
		client:     client,
		defaultTTL: config.DefaultTTL,
		keyPrefix:  config.KeyPrefix,
		logger:     logrus.WithField("component", "analytics_cache"),
	}

	cache.logger.WithFields(logrus.Fields{
		"redis_url":     config.RedisURL,
		"database":      config.Database,
		"default_ttl":   config.DefaultTTL,
		"key_prefix":    config.KeyPrefix,
	}).Info("Analytics cache initialized")

	return cache, nil
}

// GetPlayerAnalytics retrieves cached player analytics
func (ac *AnalyticsCache) GetPlayerAnalytics(ctx context.Context, playerID uuid.UUID, date time.Time) (*optimizer.PlayerAnalytics, error) {
	key := ac.buildPlayerKey(playerID, date)
	start := time.Now()

	result, err := ac.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			ac.logger.WithField("key", key).Debug("Cache miss for player analytics")
			return nil, nil // Cache miss
		}
		ac.logger.WithError(err).WithField("key", key).Error("Failed to get player analytics from cache")
		return nil, err
	}

	var analytics optimizer.PlayerAnalytics
	if err := json.Unmarshal([]byte(result), &analytics); err != nil {
		ac.logger.WithError(err).WithField("key", key).Error("Failed to unmarshal player analytics")
		return nil, err
	}

	ac.logger.WithFields(logrus.Fields{
		"key":           key,
		"response_time": time.Since(start),
	}).Debug("Cache hit for player analytics")

	return &analytics, nil
}

// SetPlayerAnalytics caches player analytics
func (ac *AnalyticsCache) SetPlayerAnalytics(ctx context.Context, playerID uuid.UUID, date time.Time, analytics *optimizer.PlayerAnalytics) error {
	if analytics == nil {
		return fmt.Errorf("analytics cannot be nil")
	}

	key := ac.buildPlayerKey(playerID, date)
	start := time.Now()

	data, err := json.Marshal(analytics)
	if err != nil {
		ac.logger.WithError(err).WithField("player_id", playerID).Error("Failed to marshal player analytics")
		return err
	}

	// Set with TTL
	err = ac.client.Set(ctx, key, data, ac.defaultTTL).Err()
	if err != nil {
		ac.logger.WithError(err).WithField("key", key).Error("Failed to set player analytics in cache")
		return err
	}

	ac.logger.WithFields(logrus.Fields{
		"key":           key,
		"ttl":           ac.defaultTTL,
		"response_time": time.Since(start),
		"size_bytes":    len(data),
	}).Debug("Cached player analytics")

	return nil
}

// GetBulkPlayerAnalytics retrieves multiple player analytics in a single operation
func (ac *AnalyticsCache) GetBulkPlayerAnalytics(ctx context.Context, playerIDs []uuid.UUID, date time.Time) (map[uuid.UUID]*optimizer.PlayerAnalytics, error) {
	if len(playerIDs) == 0 {
		return make(map[uuid.UUID]*optimizer.PlayerAnalytics), nil
	}

	// Build keys
	keys := make([]string, len(playerIDs))
	keyToID := make(map[string]uuid.UUID)
	for i, playerID := range playerIDs {
		key := ac.buildPlayerKey(playerID, date)
		keys[i] = key
		keyToID[key] = playerID
	}

	start := time.Now()

	// Multi-get
	results, err := ac.client.MGet(ctx, keys...).Result()
	if err != nil {
		ac.logger.WithError(err).Error("Failed to perform bulk get for player analytics")
		return nil, err
	}

	// Parse results
	analytics := make(map[uuid.UUID]*optimizer.PlayerAnalytics)
	hits := 0
	misses := 0

	for i, result := range results {
		if result == nil {
			misses++
			continue
		}

		var playerAnalytics optimizer.PlayerAnalytics
		if err := json.Unmarshal([]byte(result.(string)), &playerAnalytics); err != nil {
			ac.logger.WithError(err).WithField("key", keys[i]).Warn("Failed to unmarshal player analytics in bulk operation")
			misses++
			continue
		}

		playerID := keyToID[keys[i]]
		analytics[playerID] = &playerAnalytics
		hits++
	}

	ac.logger.WithFields(logrus.Fields{
		"total_requested": len(playerIDs),
		"hits":           hits,
		"misses":         misses,
		"hit_rate":       float64(hits) / float64(len(playerIDs)) * 100,
		"response_time":  time.Since(start),
	}).Debug("Bulk player analytics retrieval completed")

	return analytics, nil
}

// SetBulkPlayerAnalytics caches multiple player analytics in a single operation
func (ac *AnalyticsCache) SetBulkPlayerAnalytics(ctx context.Context, analyticsMap map[uuid.UUID]*optimizer.PlayerAnalytics, date time.Time) error {
	if len(analyticsMap) == 0 {
		return nil
	}

	start := time.Now()
	pipe := ac.client.Pipeline()

	// Add all sets to pipeline
	for playerID, analytics := range analyticsMap {
		if analytics == nil {
			continue
		}

		key := ac.buildPlayerKey(playerID, date)
		data, err := json.Marshal(analytics)
		if err != nil {
			ac.logger.WithError(err).WithField("player_id", playerID).Warn("Failed to marshal player analytics in bulk operation")
			continue
		}

		pipe.Set(ctx, key, data, ac.defaultTTL)
	}

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		ac.logger.WithError(err).Error("Failed to execute bulk set for player analytics")
		return err
	}

	ac.logger.WithFields(logrus.Fields{
		"total_set":     len(analyticsMap),
		"response_time": time.Since(start),
	}).Debug("Bulk player analytics caching completed")

	return nil
}

// GetOptimizationResult retrieves cached optimization result
func (ac *AnalyticsCache) GetOptimizationResult(ctx context.Context, cacheKey string) (*optimizer.DPResult, error) {
	key := ac.buildOptimizationKey(cacheKey)
	start := time.Now()

	result, err := ac.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			ac.logger.WithField("key", key).Debug("Cache miss for optimization result")
			return nil, nil
		}
		ac.logger.WithError(err).WithField("key", key).Error("Failed to get optimization result from cache")
		return nil, err
	}

	var dpResult optimizer.DPResult
	if err := json.Unmarshal([]byte(result), &dpResult); err != nil {
		ac.logger.WithError(err).WithField("key", key).Error("Failed to unmarshal optimization result")
		return nil, err
	}

	ac.logger.WithFields(logrus.Fields{
		"key":           key,
		"response_time": time.Since(start),
	}).Debug("Cache hit for optimization result")

	return &dpResult, nil
}

// SetOptimizationResult caches optimization result
func (ac *AnalyticsCache) SetOptimizationResult(ctx context.Context, cacheKey string, result *optimizer.DPResult, ttl time.Duration) error {
	if result == nil {
		return fmt.Errorf("optimization result cannot be nil")
	}

	key := ac.buildOptimizationKey(cacheKey)
	start := time.Now()

	data, err := json.Marshal(result)
	if err != nil {
		ac.logger.WithError(err).Error("Failed to marshal optimization result")
		return err
	}

	if ttl == 0 {
		ttl = ac.defaultTTL
	}

	err = ac.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		ac.logger.WithError(err).WithField("key", key).Error("Failed to set optimization result in cache")
		return err
	}

	ac.logger.WithFields(logrus.Fields{
		"key":           key,
		"ttl":           ttl,
		"response_time": time.Since(start),
		"size_bytes":    len(data),
	}).Debug("Cached optimization result")

	return nil
}

// WarmCache preloads cache with commonly accessed data
func (ac *AnalyticsCache) WarmCache(ctx context.Context, players []types.Player, date time.Time) error {
	ac.logger.WithFields(logrus.Fields{
		"player_count": len(players),
		"date":         date.Format("2006-01-02"),
	}).Info("Starting cache warming")

	start := time.Now()

	// Create analytics engine for warm-up
	analyticsEngine := optimizer.NewAnalyticsEngine()

	// Generate analytics for all players
	analyticsMap := make(map[uuid.UUID]*optimizer.PlayerAnalytics)
	successCount := 0
	errorCount := 0

	for _, player := range players {
		analytics, err := analyticsEngine.CalculatePlayerAnalytics(player, []optimizer.PerformanceData{})
		if err != nil {
			ac.logger.WithError(err).WithField("player_id", player.ID).Warn("Failed to calculate analytics during cache warming")
			errorCount++
			continue
		}

		analyticsMap[player.ID] = analytics
		successCount++
	}

	// Bulk cache the analytics
	if len(analyticsMap) > 0 {
		err := ac.SetBulkPlayerAnalytics(ctx, analyticsMap, date)
		if err != nil {
			ac.logger.WithError(err).Error("Failed to bulk cache analytics during warming")
			return err
		}
	}

	ac.logger.WithFields(logrus.Fields{
		"total_players":   len(players),
		"success_count":   successCount,
		"error_count":     errorCount,
		"cache_size":      len(analyticsMap),
		"warming_time":    time.Since(start),
	}).Info("Cache warming completed")

	return nil
}

// InvalidatePlayer removes cached data for a specific player
func (ac *AnalyticsCache) InvalidatePlayer(ctx context.Context, playerID uuid.UUID, dates ...time.Time) error {
	if len(dates) == 0 {
		dates = []time.Time{time.Now()}
	}

	keys := make([]string, len(dates))
	for i, date := range dates {
		keys[i] = ac.buildPlayerKey(playerID, date)
	}

	deleted, err := ac.client.Del(ctx, keys...).Result()
	if err != nil {
		ac.logger.WithError(err).WithField("player_id", playerID).Error("Failed to invalidate player cache")
		return err
	}

	ac.logger.WithFields(logrus.Fields{
		"player_id":     playerID,
		"keys_deleted":  deleted,
		"total_keys":    len(keys),
	}).Debug("Player cache invalidated")

	return nil
}

// GetCacheStats returns cache statistics
func (ac *AnalyticsCache) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	_, err := ac.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	// Parse Redis INFO stats
	stats := &CacheStats{}
	
	// This would need to be implemented to parse Redis INFO output
	// For now, providing a basic structure
	
	dbSize, err := ac.client.DBSize(ctx).Result()
	if err == nil {
		stats.TotalSize = dbSize
	}

	// Basic pool stats
	poolStats := ac.client.PoolStats()
	stats.ActiveConnections = int(poolStats.TotalConns)

	ac.logger.WithFields(logrus.Fields{
		"total_size":         stats.TotalSize,
		"active_connections": stats.ActiveConnections,
	}).Debug("Cache stats retrieved")

	return stats, nil
}

// ClearExpiredKeys removes expired analytics keys
func (ac *AnalyticsCache) ClearExpiredKeys(ctx context.Context) error {
	pattern := ac.keyPrefix + "player:*"
	
	iter := ac.client.Scan(ctx, 0, pattern, 0).Iterator()
	keysToCheck := make([]string, 0)
	
	for iter.Next(ctx) {
		keysToCheck = append(keysToCheck, iter.Val())
	}
	
	if err := iter.Err(); err != nil {
		return err
	}

	// Check TTL and remove expired keys
	expiredCount := 0
	for _, key := range keysToCheck {
		ttl, err := ac.client.TTL(ctx, key).Result()
		if err != nil {
			continue
		}
		
		// If TTL is -1 (no expiry) or positive, keep it
		// If TTL is -2 (expired) or 0, it should be cleaned up
		if ttl < 0 && ttl != -1 {
			ac.client.Del(ctx, key)
			expiredCount++
		}
	}

	ac.logger.WithFields(logrus.Fields{
		"keys_checked":  len(keysToCheck),
		"keys_expired":  expiredCount,
	}).Info("Expired keys cleanup completed")

	return nil
}

// Close closes the Redis connection
func (ac *AnalyticsCache) Close() error {
	return ac.client.Close()
}

// Helper functions

// buildPlayerKey creates a cache key for player analytics
func (ac *AnalyticsCache) buildPlayerKey(playerID uuid.UUID, date time.Time) string {
	dateStr := date.Format("2006-01-02")
	return fmt.Sprintf("%splayer:%s:date:%s", ac.keyPrefix, playerID.String(), dateStr)
}

// buildOptimizationKey creates a cache key for optimization results
func (ac *AnalyticsCache) buildOptimizationKey(cacheKey string) string {
	return fmt.Sprintf("%soptimization:%s", ac.keyPrefix, cacheKey)
}

// GenerateOptimizationCacheKey creates a cache key for optimization results
func GenerateOptimizationCacheKey(config optimizer.OptimizeConfig, playerIDs []uuid.UUID) string {
	// Create a deterministic key based on optimization parameters
	key := fmt.Sprintf("salary:%d_lineups:%d_diff:%d", 
		config.SalaryCap, config.NumLineups, config.MinDifferentPlayers)
	
	if config.UseCorrelations {
		key += fmt.Sprintf("_corr:%.2f", config.CorrelationWeight)
	}
	
	if len(config.StackingRules) > 0 {
		key += fmt.Sprintf("_stacks:%d", len(config.StackingRules))
	}
	
	// Add hash of player IDs for uniqueness
	hash := calculatePlayerHash(playerIDs)
	key += fmt.Sprintf("_players:%s", hash)
	
	return key
}

// calculatePlayerHash creates a hash from player IDs
func calculatePlayerHash(playerIDs []uuid.UUID) string {
	if len(playerIDs) == 0 {
		return "empty"
	}
	
	// Simple hash based on first and last player IDs
	// In production, you might want a more sophisticated hash
	first := playerIDs[0].String()
	last := playerIDs[len(playerIDs)-1].String()
	
	return fmt.Sprintf("%s-%s-%d", first[:8], last[:8], len(playerIDs))
}