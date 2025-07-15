package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"

	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/models"
)

// EventPublisher handles publishing events to Redis Streams with reliability and performance features
type EventPublisher struct {
	redisClient    *redis.Client
	logger         *logrus.Logger
	circuitBreaker *gobreaker.CircuitBreaker
	streamName     string
	
	// Batch publishing
	batchSize      int
	batchTimeout   time.Duration
	eventBatch     []models.RealTimeEvent
	batchMutex     sync.Mutex
	batchTimer     *time.Timer
	
	// Metrics
	publishStats   *PublishStats
	
	// Configuration
	config         *PublisherConfig
	
	// Control channels
	stopChan       chan struct{}
	flushChan      chan struct{}
}

// PublishStats tracks publishing metrics
type PublishStats struct {
	EventsPublished    int64     `json:"events_published"`
	EventsFailed       int64     `json:"events_failed"`
	BatchesPublished   int64     `json:"batches_published"`
	AverageLatency     time.Duration `json:"average_latency"`
	LastPublishTime    time.Time `json:"last_publish_time"`
	CircuitBreakerTrips int64    `json:"circuit_breaker_trips"`
	mu                 sync.Mutex
}

// PublisherConfig contains configuration for the event publisher
type PublisherConfig struct {
	StreamName         string        `json:"stream_name"`
	MaxLength          int64         `json:"max_length"`
	BatchSize          int           `json:"batch_size"`
	BatchTimeout       time.Duration `json:"batch_timeout"`
	RetryAttempts      int           `json:"retry_attempts"`
	RetryDelay         time.Duration `json:"retry_delay"`
	CircuitBreakerConfig CircuitBreakerConfig `json:"circuit_breaker"`
}

// CircuitBreakerConfig contains circuit breaker settings
type CircuitBreakerConfig struct {
	MaxRequests     uint32        `json:"max_requests"`
	Interval        time.Duration `json:"interval"`
	Timeout         time.Duration `json:"timeout"`
	FailureRatio    float64       `json:"failure_ratio"`
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(redisClient *redis.Client, config *PublisherConfig, logger *logrus.Logger) *EventPublisher {
	// Set defaults if not provided
	if config.StreamName == "" {
		config.StreamName = "realtime_events"
	}
	if config.MaxLength == 0 {
		config.MaxLength = 10000
	}
	if config.BatchSize == 0 {
		config.BatchSize = 10
	}
	if config.BatchTimeout == 0 {
		config.BatchTimeout = 1 * time.Second
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 100 * time.Millisecond
	}

	// Create circuit breaker
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "event-publisher",
		MaxRequests: config.CircuitBreakerConfig.MaxRequests,
		Interval:    config.CircuitBreakerConfig.Interval,
		Timeout:     config.CircuitBreakerConfig.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= config.CircuitBreakerConfig.FailureRatio
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.WithFields(logrus.Fields{
				"from_state": from,
				"to_state":   to,
			}).Warn("Event publisher circuit breaker state changed")
		},
	})

	publisher := &EventPublisher{
		redisClient:    redisClient,
		logger:         logger,
		circuitBreaker: cb,
		streamName:     config.StreamName,
		batchSize:      config.BatchSize,
		batchTimeout:   config.BatchTimeout,
		eventBatch:     make([]models.RealTimeEvent, 0, config.BatchSize),
		publishStats:   &PublishStats{},
		config:         config,
		stopChan:       make(chan struct{}),
		flushChan:      make(chan struct{}, 1),
	}

	// Start batch processor
	go publisher.processBatches()

	return publisher
}

// PublishEvent publishes a single event (adds to batch)
func (ep *EventPublisher) PublishEvent(ctx context.Context, event *models.RealTimeEvent) error {
	ep.batchMutex.Lock()
	defer ep.batchMutex.Unlock()

	// Add event to batch
	ep.eventBatch = append(ep.eventBatch, *event)

	// Check if batch is full
	if len(ep.eventBatch) >= ep.batchSize {
		ep.triggerFlush()
	} else if ep.batchTimer == nil {
		// Start batch timer if this is the first event in the batch
		ep.batchTimer = time.AfterFunc(ep.batchTimeout, func() {
			ep.triggerFlush()
		})
	}

	return nil
}

// PublishEventImmediate publishes an event immediately without batching
func (ep *EventPublisher) PublishEventImmediate(ctx context.Context, event *models.RealTimeEvent) error {
	startTime := time.Now()

	// Use circuit breaker for reliability
	_, err := ep.circuitBreaker.Execute(func() (interface{}, error) {
		return nil, ep.doPublishSingle(ctx, event)
	})

	if err != nil {
		ep.incrementFailedStats()
		return fmt.Errorf("failed to publish event immediately: %w", err)
	}

	ep.incrementPublishedStats()
	ep.updateLatencyStats(time.Since(startTime))

	return nil
}

// PublishBatch publishes multiple events as a batch
func (ep *EventPublisher) PublishBatch(ctx context.Context, events []models.RealTimeEvent) error {
	if len(events) == 0 {
		return nil
	}

	startTime := time.Now()

	// Use circuit breaker for reliability
	_, err := ep.circuitBreaker.Execute(func() (interface{}, error) {
		return nil, ep.doPublishBatch(ctx, events)
	})

	if err != nil {
		ep.incrementFailedStats()
		return fmt.Errorf("failed to publish batch: %w", err)
	}

	ep.incrementBatchStats()
	ep.publishStats.mu.Lock()
	ep.publishStats.EventsPublished += int64(len(events))
	ep.publishStats.mu.Unlock()
	ep.updateLatencyStats(time.Since(startTime))

	return nil
}

// doPublishSingle performs the actual single event publishing
func (ep *EventPublisher) doPublishSingle(ctx context.Context, event *models.RealTimeEvent) error {
	eventData := ep.serializeEvent(event)

	streamID, err := ep.redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: ep.streamName,
		MaxLen: ep.config.MaxLength,
		Approx: true,
		Values: eventData,
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to add event to stream: %w", err)
	}

	ep.logger.WithFields(logrus.Fields{
		"event_id":   event.EventID,
		"event_type": event.EventType,
		"stream_id":  streamID,
	}).Debug("Event published to Redis Stream")

	return nil
}

// doPublishBatch performs the actual batch publishing using Redis pipeline
func (ep *EventPublisher) doPublishBatch(ctx context.Context, events []models.RealTimeEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Use Redis pipeline for batch operations
	pipe := ep.redisClient.Pipeline()

	for _, event := range events {
		eventData := ep.serializeEvent(&event)
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: ep.streamName,
			MaxLen: ep.config.MaxLength,
			Approx: true,
			Values: eventData,
		})
	}

	// Execute pipeline
	cmders, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute batch publish pipeline: %w", err)
	}

	// Check individual command results
	successCount := 0
	for i, cmd := range cmders {
		if cmd.Err() != nil {
			ep.logger.WithError(cmd.Err()).WithFields(logrus.Fields{
				"event_id":    events[i].EventID,
				"event_type":  events[i].EventType,
				"batch_index": i,
			}).Error("Failed to publish event in batch")
		} else {
			successCount++
		}
	}

	ep.logger.WithFields(logrus.Fields{
		"batch_size":    len(events),
		"success_count": successCount,
		"failed_count":  len(events) - successCount,
	}).Info("Batch publish completed")

	if successCount == 0 {
		return fmt.Errorf("all events in batch failed to publish")
	}

	return nil
}

// serializeEvent converts a RealTimeEvent to Redis stream format
func (ep *EventPublisher) serializeEvent(event *models.RealTimeEvent) map[string]interface{} {
	eventData := map[string]interface{}{
		"event_id":      event.EventID.String(),
		"event_type":    string(event.EventType),
		"timestamp":     event.Timestamp.Format(time.RFC3339),
		"source":        event.Source,
		"data":          string(event.Data),
		"impact_rating": event.ImpactRating,
		"confidence":    event.Confidence,
		"created_at":    event.CreatedAt.Format(time.RFC3339),
	}

	// Add optional fields if present
	if event.PlayerID != nil {
		playerIDBytes, _ := json.Marshal(*event.PlayerID)
		eventData["player_id"] = string(playerIDBytes)
	}

	if event.GameID != nil {
		eventData["game_id"] = *event.GameID
	}

	if event.TournamentID != nil {
		eventData["tournament_id"] = *event.TournamentID
	}

	if event.ExpirationTime != nil {
		eventData["expiration_time"] = event.ExpirationTime.Format(time.RFC3339)
	}

	return eventData
}

// processBatches handles batch processing in a separate goroutine
func (ep *EventPublisher) processBatches() {
	for {
		select {
		case <-ep.stopChan:
			// Flush any remaining events before stopping
			ep.flushBatch()
			return
		case <-ep.flushChan:
			ep.flushBatch()
		}
	}
}

// triggerFlush triggers a batch flush
func (ep *EventPublisher) triggerFlush() {
	select {
	case ep.flushChan <- struct{}{}:
	default:
		// Flush already triggered
	}
}

// flushBatch publishes the current batch
func (ep *EventPublisher) flushBatch() {
	ep.batchMutex.Lock()
	defer ep.batchMutex.Unlock()

	if len(ep.eventBatch) == 0 {
		return
	}

	// Stop the timer if it's running
	if ep.batchTimer != nil {
		ep.batchTimer.Stop()
		ep.batchTimer = nil
	}

	// Create copy of batch for publishing
	batchToPublish := make([]models.RealTimeEvent, len(ep.eventBatch))
	copy(batchToPublish, ep.eventBatch)

	// Clear the batch
	ep.eventBatch = ep.eventBatch[:0]

	// Publish batch (without holding the mutex)
	ep.batchMutex.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ep.PublishBatch(ctx, batchToPublish); err != nil {
		ep.logger.WithError(err).WithField("batch_size", len(batchToPublish)).Error("Failed to publish batch")
	}
	ep.batchMutex.Lock() // Re-acquire for defer unlock
}

// GetStats returns publishing statistics
func (ep *EventPublisher) GetStats() PublishStats {
	ep.publishStats.mu.Lock()
	defer ep.publishStats.mu.Unlock()
	return *ep.publishStats
}

// Stop stops the event publisher
func (ep *EventPublisher) Stop() {
	close(ep.stopChan)
}

// Metrics helpers
func (ep *EventPublisher) incrementPublishedStats() {
	ep.publishStats.mu.Lock()
	ep.publishStats.EventsPublished++
	ep.publishStats.LastPublishTime = time.Now()
	ep.publishStats.mu.Unlock()
}

func (ep *EventPublisher) incrementFailedStats() {
	ep.publishStats.mu.Lock()
	ep.publishStats.EventsFailed++
	ep.publishStats.mu.Unlock()
}

func (ep *EventPublisher) incrementBatchStats() {
	ep.publishStats.mu.Lock()
	ep.publishStats.BatchesPublished++
	ep.publishStats.mu.Unlock()
}

func (ep *EventPublisher) updateLatencyStats(latency time.Duration) {
	ep.publishStats.mu.Lock()
	// Simple moving average
	if ep.publishStats.AverageLatency == 0 {
		ep.publishStats.AverageLatency = latency
	} else {
		ep.publishStats.AverageLatency = (ep.publishStats.AverageLatency + latency) / 2
	}
	ep.publishStats.mu.Unlock()
}

// PublishEventWithRetry publishes an event with retry logic
func (ep *EventPublisher) PublishEventWithRetry(ctx context.Context, event *models.RealTimeEvent) error {
	var lastErr error

	for attempt := 0; attempt < ep.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(ep.config.RetryDelay * time.Duration(attempt)):
			}
		}

		err := ep.PublishEventImmediate(ctx, event)
		if err == nil {
			return nil
		}

		lastErr = err
		ep.logger.WithError(err).WithFields(logrus.Fields{
			"event_id": event.EventID,
			"attempt":  attempt + 1,
		}).Warn("Event publish attempt failed, retrying")
	}

	return fmt.Errorf("failed to publish event after %d attempts: %w", ep.config.RetryAttempts, lastErr)
}

// CreateEventFromData is a helper function to create RealTimeEvent from raw data
func CreateEventFromData(eventType models.EventType, source string, data map[string]interface{}, impactRating float64) *models.RealTimeEvent {
	event := &models.RealTimeEvent{
		EventType:    eventType,
		Timestamp:    time.Now(),
		Source:       source,
		ImpactRating: impactRating,
		Confidence:   1.0,
		CreatedAt:    time.Now(),
	}

	// Convert data to JSON
	if dataBytes, err := json.Marshal(data); err == nil {
		event.Data = dataBytes
	}

	return event
}