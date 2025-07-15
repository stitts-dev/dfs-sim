package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/stitts-dev/dfs-sim/shared/types"
)

// OptimizationCacheService handles caching for optimization results
type OptimizationCacheService struct {
	client *redis.Client
	logger *logrus.Logger
}

// NewOptimizationCacheService creates a new optimization cache service
func NewOptimizationCacheService(client *redis.Client, logger *logrus.Logger) *OptimizationCacheService {
	return &OptimizationCacheService{
		client: client,
		logger: logger,
	}
}

// SetOptimizationResult stores an optimization result in cache
func (c *OptimizationCacheService) SetOptimizationResult(ctx context.Context, key string, result *types.OptimizationResult, expiration time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal optimization result: %w", err)
	}

	fullKey := fmt.Sprintf("optimization:%s", key)
	if err := c.client.Set(ctx, fullKey, data, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set optimization result in cache: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"cache_key": fullKey,
		"expiration": expiration,
		"lineups_count": len(result.Lineups),
	}).Debug("Cached optimization result")

	return nil
}

// GetOptimizationResult retrieves an optimization result from cache
func (c *OptimizationCacheService) GetOptimizationResult(ctx context.Context, key string) (*types.OptimizationResult, error) {
	fullKey := fmt.Sprintf("optimization:%s", key)
	data, err := c.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("optimization result not found in cache")
		}
		return nil, fmt.Errorf("failed to get optimization result from cache: %w", err)
	}

	var result types.OptimizationResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal optimization result: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"cache_key": fullKey,
		"lineups_count": len(result.Lineups),
	}).Debug("Retrieved optimization result from cache")

	return &result, nil
}

// SetSimulationResult stores a simulation result in cache
func (c *OptimizationCacheService) SetSimulationResult(ctx context.Context, key string, result *types.SimulationResult, expiration time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal simulation result: %w", err)
	}

	fullKey := fmt.Sprintf("simulation:%s", key)
	if err := c.client.Set(ctx, fullKey, data, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set simulation result in cache: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"cache_key": fullKey,
		"expiration": expiration,
		"iterations": result.Iterations,
	}).Debug("Cached simulation result")

	return nil
}

// GetSimulationResult retrieves a simulation result from cache
func (c *OptimizationCacheService) GetSimulationResult(ctx context.Context, key string) (*types.SimulationResult, error) {
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("simulation result not found in cache")
		}
		return nil, fmt.Errorf("failed to get simulation result from cache: %w", err)
	}

	var result types.SimulationResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal simulation result: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"cache_key": key,
		"iterations": result.Iterations,
	}).Debug("Retrieved simulation result from cache")

	return &result, nil
}

// DeleteOptimizationResult removes an optimization result from cache
func (c *OptimizationCacheService) DeleteOptimizationResult(ctx context.Context, key string) error {
	fullKey := fmt.Sprintf("optimization:%s", key)
	if err := c.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("failed to delete optimization result from cache: %w", err)
	}

	c.logger.WithField("cache_key", fullKey).Debug("Deleted optimization result from cache")
	return nil
}

// GetStatus returns cache statistics
func (c *OptimizationCacheService) GetStatus(ctx context.Context) map[string]interface{} {
	info := c.client.Info(ctx)
	dbSize := c.client.DBSize(ctx)
	memory := c.client.MemoryUsage(ctx, "")

	status := map[string]interface{}{
		"service": "optimization-cache",
		"timestamp": time.Now(),
		"connected": true,
	}

	if dbSize.Err() == nil {
		status["db_size"] = dbSize.Val()
	}

	if memory.Err() == nil {
		status["memory_usage"] = memory.Val()
	}

	if info.Err() == nil {
		status["redis_info"] = "available"
	}

	// Get optimization-specific cache stats
	optimizationKeys, err := c.client.Keys(ctx, "optimization:*").Result()
	if err == nil {
		status["optimization_keys"] = len(optimizationKeys)
	}

	simulationKeys, err := c.client.Keys(ctx, "simulation:*").Result()
	if err == nil {
		status["simulation_keys"] = len(simulationKeys)
	}

	return status
}

// FlushOptimizationCache clears all optimization results from cache
func (c *OptimizationCacheService) FlushOptimizationCache(ctx context.Context) error {
	keys, err := c.client.Keys(ctx, "optimization:*").Result()
	if err != nil {
		return fmt.Errorf("failed to get optimization keys: %w", err)
	}

	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete optimization keys: %w", err)
		}
	}

	c.logger.WithField("deleted_keys", len(keys)).Info("Flushed optimization cache")
	return nil
}

// FlushSimulationCache clears all simulation results from cache
func (c *OptimizationCacheService) FlushSimulationCache(ctx context.Context) error {
	keys, err := c.client.Keys(ctx, "simulation:*").Result()
	if err != nil {
		return fmt.Errorf("failed to get simulation keys: %w", err)
	}

	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete simulation keys: %w", err)
		}
	}

	c.logger.WithField("deleted_keys", len(keys)).Info("Flushed simulation cache")
	return nil
}

// SetWithRetry attempts to set a cache entry with retries
func (c *OptimizationCacheService) SetWithRetry(ctx context.Context, key string, value interface{}, expiration time.Duration, maxRetries int) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := c.client.Set(ctx, key, data, expiration).Err(); err != nil {
			lastErr = err
			c.logger.WithError(err).WithField("attempt", i+1).Warn("Cache set attempt failed")
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond) // Exponential backoff
			continue
		}
		return nil
	}

	return fmt.Errorf("failed to set cache after %d retries: %w", maxRetries, lastErr)
}