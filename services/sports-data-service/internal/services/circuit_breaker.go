package services

import (
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
)

type CircuitBreakerService struct {
	breakers map[string]*gobreaker.CircuitBreaker
	logger   *logrus.Logger
}

func NewCircuitBreakerService(threshold int, timeout time.Duration, logger *logrus.Logger) *CircuitBreakerService {
	settings := gobreaker.Settings{
		Name:        "external-api",
		MaxRequests: uint32(threshold),
		Timeout:     timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.WithFields(logrus.Fields{
				"component": "circuit_breaker",
				"service":   name,
				"from":      from.String(),
				"to":        to.String(),
			}).Info("Circuit breaker state changed")
		},
	}

	// Create separate breakers for each external service
	breakers := map[string]*gobreaker.CircuitBreaker{
		"rapidapi":    gobreaker.NewCircuitBreaker(settings),
		"espn":        gobreaker.NewCircuitBreaker(settings),
		"balldontlie": gobreaker.NewCircuitBreaker(settings),
		"thesportsdb": gobreaker.NewCircuitBreaker(settings),
	}

	return &CircuitBreakerService{
		breakers: breakers,
		logger:   logger,
	}
}

// Execute wraps a function call with circuit breaker protection
func (cb *CircuitBreakerService) Execute(service string, fn func() (interface{}, error)) (interface{}, error) {
	breaker, exists := cb.breakers[service]
	if !exists {
		cb.logger.WithFields(logrus.Fields{
			"component": "circuit_breaker",
			"service":   service,
		}).Warn("No circuit breaker found for service, executing without protection")
		return fn()
	}

	return breaker.Execute(fn)
}

// GetState returns the current state of a circuit breaker
func (cb *CircuitBreakerService) GetState(service string) gobreaker.State {
	if breaker, exists := cb.breakers[service]; exists {
		return breaker.State()
	}
	return gobreaker.StateClosed // Default state
}

// GetCounts returns the current counts for a circuit breaker
func (cb *CircuitBreakerService) GetCounts(service string) gobreaker.Counts {
	if breaker, exists := cb.breakers[service]; exists {
		return breaker.Counts()
	}
	return gobreaker.Counts{} // Empty counts
}
