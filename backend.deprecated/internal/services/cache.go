package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type CacheService struct {
	client *redis.Client
}

func NewCacheService(client *redis.Client) *CacheService {
	return &CacheService{
		client: client,
	}
}

func (s *CacheService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if err := s.client.Set(ctx, key, data, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

func (s *CacheService) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found")
		}
		return fmt.Errorf("failed to get cache: %w", err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

func (s *CacheService) Delete(ctx context.Context, keys ...string) error {
	if err := s.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}
	return nil
}

func (s *CacheService) Exists(ctx context.Context, key string) (bool, error) {
	val, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check cache existence: %w", err)
	}
	return val > 0, nil
}

// Cache key generators
func LineupCacheKey(userID uint, contestID uint) string {
	return fmt.Sprintf("lineup:%d:%d", userID, contestID)
}

func OptimizationCacheKey(contestID uint, config string) string {
	return fmt.Sprintf("optimization:%d:%s", contestID, config)
}

func PlayersCacheKey(contestID uint) string {
	return fmt.Sprintf("players:%d", contestID)
}

func SimulationCacheKey(lineupID uint) string {
	return fmt.Sprintf("simulation:%d", lineupID)
}

// Cache with retry logic
func (s *CacheService) SetWithRetry(ctx context.Context, key string, value interface{}, expiration time.Duration, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		if err = s.Set(ctx, key, value, expiration); err == nil {
			return nil
		}
		logrus.Warnf("Cache set failed (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(time.Millisecond * 100 * time.Duration(i+1))
	}
	return err
}

// Convenience methods without context (use background context)
func (s *CacheService) SetSimple(key string, value interface{}, expiration time.Duration) error {
	return s.Set(context.Background(), key, value, expiration)
}

func (s *CacheService) GetSimple(key string, dest interface{}) error {
	return s.Get(context.Background(), key, dest)
}

// Flush clears all cache entries
func (s *CacheService) Flush() error {
	return s.client.FlushDB(context.Background()).Err()
}
