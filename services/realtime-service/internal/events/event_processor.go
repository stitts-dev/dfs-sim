package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/models"
)

// EventProcessor handles real-time event processing using Redis Streams
type EventProcessor struct {
	redisClient      *redis.Client
	db               *gorm.DB
	logger           *logrus.Logger
	eventHandlers    map[models.EventType]EventHandler
	consumerGroup    string
	consumerID       string
	streamName       string
	processingStats  *ProcessingStats
	stopChan         chan struct{}
	wg               sync.WaitGroup
	isRunning        bool
	mu               sync.RWMutex
}

// ProcessingStats tracks event processing metrics
type ProcessingStats struct {
	EventsProcessed   int64     `json:"events_processed"`
	EventsFailed      int64     `json:"events_failed"`
	EventsRetried     int64     `json:"events_retried"`
	AverageProcessTime time.Duration `json:"average_process_time"`
	LastProcessTime   time.Time `json:"last_process_time"`
	ErrorRate         float64   `json:"error_rate"`
	mu                sync.Mutex
}

// EventHandler defines the interface for handling specific event types
type EventHandler interface {
	HandleEvent(ctx context.Context, event *models.RealTimeEvent) error
	GetEventType() models.EventType
	GetPriority() int // Higher priority handlers run first
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(redisClient *redis.Client, db *gorm.DB, logger *logrus.Logger) *EventProcessor {
	consumerID := fmt.Sprintf("realtime-service-%d", time.Now().UnixNano())
	
	processor := &EventProcessor{
		redisClient:   redisClient,
		db:            db,
		logger:        logger,
		eventHandlers: make(map[models.EventType]EventHandler),
		consumerGroup: "realtime-service-group",
		consumerID:    consumerID,
		streamName:    "realtime_events",
		processingStats: &ProcessingStats{},
		stopChan:      make(chan struct{}),
	}

	// Register default event handlers
	processor.registerDefaultHandlers()

	return processor
}

// registerDefaultHandlers registers built-in event handlers
func (ep *EventProcessor) registerDefaultHandlers() {
	// Register specific event handlers
	ep.RegisterHandler(NewPlayerInjuryHandler(ep.db, ep.logger))
	ep.RegisterHandler(NewWeatherUpdateHandler(ep.db, ep.logger))
	ep.RegisterHandler(NewOwnershipChangeHandler(ep.db, ep.logger))
	ep.RegisterHandler(NewContestUpdateHandler(ep.db, ep.logger))
	ep.RegisterHandler(NewGenericEventHandler(ep.db, ep.logger))
}

// RegisterHandler registers an event handler for a specific event type
func (ep *EventProcessor) RegisterHandler(handler EventHandler) {
	ep.eventHandlers[handler.GetEventType()] = handler
	ep.logger.WithFields(logrus.Fields{
		"event_type": handler.GetEventType(),
		"priority":   handler.GetPriority(),
	}).Info("Registered event handler")
}

// Start begins event processing
func (ep *EventProcessor) Start(ctx context.Context) error {
	ep.mu.Lock()
	if ep.isRunning {
		ep.mu.Unlock()
		return fmt.Errorf("event processor is already running")
	}
	ep.isRunning = true
	ep.mu.Unlock()

	ep.logger.Info("Starting event processor")

	// Create consumer group if it doesn't exist
	err := ep.redisClient.XGroupCreateMkStream(ctx, ep.streamName, ep.consumerGroup, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	// Start processing goroutines
	ep.wg.Add(2)
	go ep.processEvents(ctx)
	go ep.processPendingEvents(ctx)

	// Wait for stop signal
	<-ep.stopChan
	
	ep.logger.Info("Stopping event processor")
	ep.wg.Wait()
	
	ep.mu.Lock()
	ep.isRunning = false
	ep.mu.Unlock()

	return nil
}

// Stop stops event processing
func (ep *EventProcessor) Stop() {
	ep.mu.RLock()
	if !ep.isRunning {
		ep.mu.RUnlock()
		return
	}
	ep.mu.RUnlock()

	close(ep.stopChan)
}

// PublishEvent publishes an event to the Redis Stream
func (ep *EventProcessor) PublishEvent(ctx context.Context, event *models.RealTimeEvent) error {
	// Prepare event data for Redis Stream
	eventData := map[string]interface{}{
		"event_id":      event.EventID.String(),
		"event_type":    string(event.EventType),
		"player_id":     event.PlayerID,
		"game_id":       event.GameID,
		"tournament_id": event.TournamentID,
		"timestamp":     event.Timestamp.Format(time.RFC3339),
		"source":        event.Source,
		"data":          string(event.Data),
		"impact_rating": event.ImpactRating,
		"confidence":    event.Confidence,
		"expiration_time": formatTimePtr(event.ExpirationTime),
	}

	// Add to Redis Stream
	streamID, err := ep.redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: ep.streamName,
		MaxLen: 10000, // Keep last 10k events
		Approx: true,
		Values: eventData,
	}).Result()

	if err != nil {
		ep.logger.WithError(err).WithField("event_id", event.EventID).Error("Failed to publish event to stream")
		return fmt.Errorf("failed to publish event: %w", err)
	}

	ep.logger.WithFields(logrus.Fields{
		"event_id":   event.EventID,
		"event_type": event.EventType,
		"stream_id":  streamID,
	}).Debug("Published event to Redis Stream")

	return nil
}

// processEvents processes new events from the stream
func (ep *EventProcessor) processEvents(ctx context.Context) {
	defer ep.wg.Done()

	ep.logger.Info("Starting event processing worker")

	for {
		select {
		case <-ep.stopChan:
			return
		default:
			// Read events from stream with consumer group
			streams, err := ep.redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    ep.consumerGroup,
				Consumer: ep.consumerID,
				Streams:  []string{ep.streamName, ">"},
				Count:    10, // Process up to 10 events at once
				Block:    5 * time.Second,
			}).Result()

			if err != nil {
				if err != redis.Nil {
					ep.logger.WithError(err).Error("Failed to read from Redis Stream")
					time.Sleep(time.Second) // Brief pause before retry
				}
				continue
			}

			// Process each stream
			for _, stream := range streams {
				for _, message := range stream.Messages {
					if err := ep.handleStreamMessage(ctx, message); err != nil {
						ep.incrementFailedStats()
						ep.logger.WithError(err).WithField("message_id", message.ID).Error("Failed to process stream message")
						
						// Send to dead letter queue or retry logic
						ep.handleFailedMessage(ctx, message, err)
					} else {
						ep.incrementProcessedStats()
						// Acknowledge successful processing
						ep.redisClient.XAck(ctx, ep.streamName, ep.consumerGroup, message.ID)
					}
				}
			}
		}
	}
}

// processPendingEvents processes any pending events that weren't acknowledged
func (ep *EventProcessor) processPendingEvents(ctx context.Context) {
	defer ep.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ep.logger.Info("Starting pending events processor")

	for {
		select {
		case <-ep.stopChan:
			return
		case <-ticker.C:
			// Check for pending events
			pending, err := ep.redisClient.XPendingExt(ctx, &redis.XPendingExtArgs{
				Stream:   ep.streamName,
				Group:    ep.consumerGroup,
				Start:    "-",
				End:      "+",
				Count:    100,
				Consumer: ep.consumerID,
			}).Result()

			if err != nil {
				ep.logger.WithError(err).Error("Failed to get pending events")
				continue
			}

			for _, msg := range pending {
				// Claim old pending messages (older than 60 seconds)
				if time.Since(msg.LastDelivery) > 60*time.Second {
					claimed, err := ep.redisClient.XClaim(ctx, &redis.XClaimArgs{
						Stream:   ep.streamName,
						Group:    ep.consumerGroup,
						Consumer: ep.consumerID,
						MinIdle:  60 * time.Second,
						Messages: []string{msg.ID},
					}).Result()

					if err != nil {
						ep.logger.WithError(err).Error("Failed to claim pending event")
						continue
					}

					// Process claimed messages
					for _, claimedMsg := range claimed {
						if err := ep.handleStreamMessage(ctx, claimedMsg); err != nil {
							ep.logger.WithError(err).WithField("message_id", claimedMsg.ID).Error("Failed to process claimed message")
							ep.handleFailedMessage(ctx, claimedMsg, err)
						} else {
							ep.redisClient.XAck(ctx, ep.streamName, ep.consumerGroup, claimedMsg.ID)
						}
					}
				}
			}
		}
	}
}

// handleStreamMessage converts Redis Stream message to RealTimeEvent and processes it
func (ep *EventProcessor) handleStreamMessage(ctx context.Context, message redis.XMessage) error {
	startTime := time.Now()
	
	// Parse message into RealTimeEvent
	event, err := ep.parseStreamMessage(message)
	if err != nil {
		return fmt.Errorf("failed to parse stream message: %w", err)
	}

	// Store event in database
	if err := ep.db.Create(event).Error; err != nil {
		ep.logger.WithError(err).WithField("event_id", event.EventID).Error("Failed to store event in database")
		// Continue processing even if DB write fails
	}

	// Find and execute handler
	handler, exists := ep.eventHandlers[event.EventType]
	if !exists {
		// Use generic handler as fallback
		handler = ep.eventHandlers[models.EventType("generic")]
		if handler == nil {
			ep.logger.WithField("event_type", event.EventType).Warn("No handler found for event type")
			return nil
		}
	}

	// Execute handler with timeout
	handlerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := handler.HandleEvent(handlerCtx, event); err != nil {
		return fmt.Errorf("event handler failed: %w", err)
	}

	// Update processing time stats
	ep.updateProcessingTime(time.Since(startTime))

	// Mark event as processed
	now := time.Now()
	event.ProcessedAt = &now
	ep.db.Save(event)

	ep.logger.WithFields(logrus.Fields{
		"event_id":      event.EventID,
		"event_type":    event.EventType,
		"process_time":  time.Since(startTime),
		"handler":       handler.GetEventType(),
	}).Debug("Successfully processed event")

	return nil
}

// parseStreamMessage converts Redis Stream message to RealTimeEvent
func (ep *EventProcessor) parseStreamMessage(message redis.XMessage) (*models.RealTimeEvent, error) {
	event := &models.RealTimeEvent{}

	// Parse event ID
	if eventIDStr, ok := message.Values["event_id"].(string); ok {
		if err := event.EventID.UnmarshalText([]byte(eventIDStr)); err != nil {
			return nil, fmt.Errorf("invalid event_id: %w", err)
		}
	}

	// Parse event type
	if eventTypeStr, ok := message.Values["event_type"].(string); ok {
		event.EventType = models.EventType(eventTypeStr)
	}

	// Parse player ID
	if playerIDStr, ok := message.Values["player_id"].(string); ok && playerIDStr != "" {
		var playerID uint64
		if err := json.Unmarshal([]byte(playerIDStr), &playerID); err == nil {
			pid := uint(playerID)
			event.PlayerID = &pid
		}
	}

	// Parse optional fields
	if gameID, ok := message.Values["game_id"].(string); ok && gameID != "" {
		event.GameID = &gameID
	}

	if tournamentID, ok := message.Values["tournament_id"].(string); ok && tournamentID != "" {
		event.TournamentID = &tournamentID
	}

	// Parse timestamp
	if timestampStr, ok := message.Values["timestamp"].(string); ok {
		if ts, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			event.Timestamp = ts
		} else {
			event.Timestamp = time.Now()
		}
	} else {
		event.Timestamp = time.Now()
	}

	// Parse other fields
	if source, ok := message.Values["source"].(string); ok {
		event.Source = source
	}

	if dataStr, ok := message.Values["data"].(string); ok {
		event.Data = []byte(dataStr)
	}

	if impactStr, ok := message.Values["impact_rating"].(string); ok {
		var impact float64
		if err := json.Unmarshal([]byte(impactStr), &impact); err == nil {
			event.ImpactRating = impact
		}
	}

	if confidenceStr, ok := message.Values["confidence"].(string); ok {
		var confidence float64
		if err := json.Unmarshal([]byte(confidenceStr), &confidence); err == nil {
			event.Confidence = confidence
		}
	}

	// Parse expiration time
	if expirationStr, ok := message.Values["expiration_time"].(string); ok && expirationStr != "" {
		if expTime, err := time.Parse(time.RFC3339, expirationStr); err == nil {
			event.ExpirationTime = &expTime
		}
	}

	return event, nil
}

// handleFailedMessage handles events that failed to process
func (ep *EventProcessor) handleFailedMessage(ctx context.Context, message redis.XMessage, processingError error) {
	// Log the failure
	failureLog := models.EventLog{
		Action:  "failed",
		Details: []byte(fmt.Sprintf(`{"error": "%s", "message_id": "%s"}`, processingError.Error(), message.ID)),
	}

	if eventIDStr, ok := message.Values["event_id"].(string); ok {
		if eventUUID, err := parseUUID(eventIDStr); err == nil {
			failureLog.EventID = eventUUID
		}
	}

	ep.db.Create(&failureLog)

	// For now, just acknowledge the failed message to prevent infinite retries
	// In production, you might want to implement:
	// 1. Dead letter queue
	// 2. Exponential backoff retry
	// 3. Manual retry interface
	ep.redisClient.XAck(ctx, ep.streamName, ep.consumerGroup, message.ID)
}

// GetStats returns processing statistics
func (ep *EventProcessor) GetStats() ProcessingStats {
	ep.processingStats.mu.Lock()
	defer ep.processingStats.mu.Unlock()
	
	stats := *ep.processingStats
	
	// Calculate error rate
	total := stats.EventsProcessed + stats.EventsFailed
	if total > 0 {
		stats.ErrorRate = float64(stats.EventsFailed) / float64(total) * 100
	}
	
	return stats
}

// Metrics helpers
func (ep *EventProcessor) incrementProcessedStats() {
	ep.processingStats.mu.Lock()
	ep.processingStats.EventsProcessed++
	ep.processingStats.LastProcessTime = time.Now()
	ep.processingStats.mu.Unlock()
}

func (ep *EventProcessor) incrementFailedStats() {
	ep.processingStats.mu.Lock()
	ep.processingStats.EventsFailed++
	ep.processingStats.mu.Unlock()
}

func (ep *EventProcessor) updateProcessingTime(duration time.Duration) {
	ep.processingStats.mu.Lock()
	// Simple moving average calculation
	if ep.processingStats.AverageProcessTime == 0 {
		ep.processingStats.AverageProcessTime = duration
	} else {
		ep.processingStats.AverageProcessTime = (ep.processingStats.AverageProcessTime + duration) / 2
	}
	ep.processingStats.mu.Unlock()
}

// Utility functions
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func parseUUID(s string) ([]byte, error) {
	// Simple UUID string to bytes conversion
	return []byte(s), nil
}