package services

import (
	"fmt"
	"sync"
	"time"
)

// SMSRateLimiter implements rate limiting for SMS services
type SMSRateLimiter struct {
	mu         sync.RWMutex
	requests   map[string][]time.Time
	maxRequests int
	window     time.Duration
}

// NewSMSRateLimiter creates a new SMS rate limiter
// maxRequests: maximum number of requests per window
// window: time window for rate limiting (e.g., 1 hour)
func NewSMSRateLimiter(maxRequests int, window time.Duration) *SMSRateLimiter {
	return &SMSRateLimiter{
		requests:   make(map[string][]time.Time),
		maxRequests: maxRequests,
		window:     window,
	}
}

// Allow checks if the request is allowed for the given phone number
func (rl *SMSRateLimiter) Allow(phoneNumber string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	
	// Clean up old requests
	rl.cleanupOldRequests(phoneNumber, now)
	
	// Check if limit exceeded
	if len(rl.requests[phoneNumber]) >= rl.maxRequests {
		return fmt.Errorf("rate limit exceeded: maximum %d SMS per %v", rl.maxRequests, rl.window)
	}
	
	// Record new request
	if rl.requests[phoneNumber] == nil {
		rl.requests[phoneNumber] = make([]time.Time, 0)
	}
	rl.requests[phoneNumber] = append(rl.requests[phoneNumber], now)
	
	return nil
}

// cleanupOldRequests removes requests outside the time window
func (rl *SMSRateLimiter) cleanupOldRequests(phoneNumber string, now time.Time) {
	if requests, exists := rl.requests[phoneNumber]; exists {
		cutoff := now.Add(-rl.window)
		validRequests := make([]time.Time, 0, len(requests))
		
		for _, req := range requests {
			if req.After(cutoff) {
				validRequests = append(validRequests, req)
			}
		}
		
		if len(validRequests) == 0 {
			delete(rl.requests, phoneNumber)
		} else {
			rl.requests[phoneNumber] = validRequests
		}
	}
}

// GetStats returns rate limiter statistics
func (rl *SMSRateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	return map[string]interface{}{
		"tracked_numbers": len(rl.requests),
		"max_requests":    rl.maxRequests,
		"window":          rl.window.String(),
	}
}

// Reset clears all rate limiting data
func (rl *SMSRateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.requests = make(map[string][]time.Time)
}