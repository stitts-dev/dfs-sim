package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"

	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/models"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// SSEProvider implements real-time data streaming via Server-Sent Events
type SSEProvider struct {
	name              string
	sseURL            string
	headers           map[string]string
	supportedSports   []string
	
	// Connection management
	httpClient        *http.Client
	response          *http.Response
	scanner           *bufio.Scanner
	isConnected       bool
	connMutex         sync.RWMutex
	
	// Event channels
	eventChan         chan models.RealTimeEvent
	ownershipChan     chan models.OwnershipSnapshot
	
	// Configuration
	config            *RealTimeConfig
	logger            *logrus.Logger
	circuitBreaker    *gobreaker.CircuitBreaker
	
	// Metrics
	metrics           *ProviderMetrics
	metricsMutex      sync.Mutex
	
	// Control channels
	stopChan          chan struct{}
	reconnectChan     chan struct{}
	
	// Health monitoring
	lastEventTime     time.Time
	connectionStartTime time.Time
	reconnectAttempts int
	
	// SSE-specific
	lastEventID       string
	retryInterval     time.Duration
}

// NewSSEProvider creates a new Server-Sent Events real-time provider
func NewSSEProvider(name, sseURL string, config *RealTimeConfig, logger *logrus.Logger) *SSEProvider {
	// Create circuit breaker
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        fmt.Sprintf("sse-provider-%s", name),
		MaxRequests: uint32(config.MaxReconnectAttempts),
		Interval:    config.RateLimitWindow,
		Timeout:     config.RecoveryTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.WithFields(logrus.Fields{
				"provider":   name,
				"from_state": from,
				"to_state":   to,
			}).Warn("SSE provider circuit breaker state changed")
		},
	})

	httpClient := &http.Client{
		Timeout: config.ConnectionTimeout,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: true, // Important for SSE
		},
	}

	return &SSEProvider{
		name:            name,
		sseURL:          sseURL,
		headers:         make(map[string]string),
		supportedSports: []string{"golf"},
		httpClient:      httpClient,
		eventChan:       make(chan models.RealTimeEvent, config.EventBufferSize),
		ownershipChan:   make(chan models.OwnershipSnapshot, config.EventBufferSize),
		config:          config,
		logger:          logger,
		circuitBreaker:  cb,
		metrics:         &ProviderMetrics{},
		stopChan:        make(chan struct{}),
		reconnectChan:   make(chan struct{}, 1),
		retryInterval:   5 * time.Second, // Default SSE retry interval
	}
}

// GetProviderName returns the provider name
func (sp *SSEProvider) GetProviderName() string {
	return sp.name
}

// GetSupportedSports returns supported sports
func (sp *SSEProvider) GetSupportedSports() []string {
	return sp.supportedSports
}

// IsHealthy returns the provider health status
func (sp *SSEProvider) IsHealthy() bool {
	sp.connMutex.RLock()
	defer sp.connMutex.RUnlock()
	return sp.isConnected && time.Since(sp.lastEventTime) < 5*time.Minute
}

// Connect establishes SSE connection with circuit breaker protection
func (sp *SSEProvider) Connect(ctx context.Context) error {
	_, err := sp.circuitBreaker.Execute(func() (interface{}, error) {
		return nil, sp.doConnect(ctx)
	})
	return err
}

// doConnect performs the actual SSE connection
func (sp *SSEProvider) doConnect(ctx context.Context) error {
	sp.logger.WithField("provider", sp.name).Info("Connecting to SSE provider")

	// Build request
	req, err := http.NewRequestWithContext(ctx, "GET", sp.sseURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create SSE request: %w", err)
	}

	// Set SSE headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	
	// Add custom headers
	for key, value := range sp.headers {
		req.Header.Set(key, value)
	}

	// Add Last-Event-ID if available
	if sp.lastEventID != "" {
		req.Header.Set("Last-Event-ID", sp.lastEventID)
	}

	// Execute request
	resp, err := sp.httpClient.Do(req)
	if err != nil {
		sp.incrementErrorMetric()
		return fmt.Errorf("failed to connect to SSE endpoint: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		sp.incrementErrorMetric()
		return fmt.Errorf("SSE connection failed with status %d", resp.StatusCode)
	}

	sp.connMutex.Lock()
	sp.response = resp
	sp.scanner = bufio.NewScanner(resp.Body)
	sp.isConnected = true
	sp.connectionStartTime = time.Now()
	sp.reconnectAttempts = 0
	sp.lastEventTime = time.Now()
	sp.connMutex.Unlock()

	sp.logger.WithField("provider", sp.name).Info("SSE connection established")

	// Start reading events
	go sp.readEvents()
	
	return nil
}

// Subscribe creates a subscription (SSE providers typically auto-stream all events)
func (sp *SSEProvider) Subscribe(ctx context.Context, subscription *Subscription) (<-chan models.RealTimeEvent, error) {
	if !sp.IsHealthy() {
		return nil, fmt.Errorf("provider %s is not healthy", sp.name)
	}

	// For SSE, we typically can't send subscription parameters
	// Events are filtered client-side based on subscription criteria
	sp.logger.WithFields(logrus.Fields{
		"provider":       sp.name,
		"subscription_id": subscription.ID,
		"event_types":    subscription.EventTypes,
	}).Info("Created SSE subscription (client-side filtering)")

	return sp.eventChan, nil
}

// Unsubscribe removes a subscription (no-op for SSE)
func (sp *SSEProvider) Unsubscribe(ctx context.Context, subscriptionID string) error {
	sp.logger.WithField("subscription_id", subscriptionID).Info("SSE unsubscribe (no server action required)")
	return nil
}

// GetActiveSubscriptions returns empty slice (SSE doesn't track server-side subscriptions)
func (sp *SSEProvider) GetActiveSubscriptions() []Subscription {
	return []Subscription{}
}

// StreamEvents streams real-time events
func (sp *SSEProvider) StreamEvents(ctx context.Context, eventTypes []models.EventType) (<-chan models.RealTimeEvent, error) {
	if !sp.IsHealthy() {
		if err := sp.Connect(ctx); err != nil {
			return nil, fmt.Errorf("failed to connect for event streaming: %w", err)
		}
	}
	return sp.eventChan, nil
}

// StreamOwnership streams ownership data
func (sp *SSEProvider) StreamOwnership(ctx context.Context, contestIDs []string) (<-chan models.OwnershipSnapshot, error) {
	if !sp.IsHealthy() {
		if err := sp.Connect(ctx); err != nil {
			return nil, fmt.Errorf("failed to connect for ownership streaming: %w", err)
		}
	}
	return sp.ownershipChan, nil
}

// GetConnectionStatus returns connection health information
func (sp *SSEProvider) GetConnectionStatus() ConnectionStatus {
	sp.connMutex.RLock()
	defer sp.connMutex.RUnlock()

	status := ConnectionStatus{
		IsConnected:       sp.isConnected,
		LastConnected:     sp.connectionStartTime,
		ReconnectAttempts: sp.reconnectAttempts,
	}

	if sp.isConnected {
		status.ConnectionUptime = time.Since(sp.connectionStartTime)
		status.LatencyMs = float64(time.Since(sp.lastEventTime).Milliseconds())
	}

	return status
}

// GetMetrics returns provider performance metrics
func (sp *SSEProvider) GetMetrics() ProviderMetrics {
	sp.metricsMutex.Lock()
	defer sp.metricsMutex.Unlock()
	return *sp.metrics
}

// readEvents handles incoming SSE events
func (sp *SSEProvider) readEvents() {
	defer sp.cleanup()

	var event *SSEEvent
	
	for {
		select {
		case <-sp.stopChan:
			return
		default:
			sp.connMutex.RLock()
			scanner := sp.scanner
			sp.connMutex.RUnlock()

			if scanner == nil {
				sp.scheduleReconnect()
				return
			}

			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					sp.logger.WithError(err).Error("SSE scanner error")
					sp.incrementErrorMetric()
				}
				sp.scheduleReconnect()
				return
			}

			line := scanner.Text()
			sp.lastEventTime = time.Now()

			// Parse SSE line
			if newEvent := sp.parseSSELine(line, event); newEvent != nil {
				event = newEvent
			}

			// Process complete event
			if event != nil && sp.isCompleteEvent(event) {
				sp.processSSEEvent(event)
				event = nil // Reset for next event
			}
		}
	}
}

// SSEEvent represents a parsed Server-Sent Event
type SSEEvent struct {
	ID    string
	Event string
	Data  string
	Retry string
}

// parseSSELine parses individual SSE lines
func (sp *SSEProvider) parseSSELine(line string, currentEvent *SSEEvent) *SSEEvent {
	// Empty line indicates end of event
	if line == "" {
		return currentEvent
	}

	// Initialize event if needed
	if currentEvent == nil {
		currentEvent = &SSEEvent{}
	}

	// Parse field
	if strings.HasPrefix(line, ":") {
		// Comment line, ignore
		return currentEvent
	}

	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return currentEvent
	}

	field := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	switch field {
	case "id":
		currentEvent.ID = value
		sp.lastEventID = value
	case "event":
		currentEvent.Event = value
	case "data":
		if currentEvent.Data != "" {
			currentEvent.Data += "\n"
		}
		currentEvent.Data += value
	case "retry":
		currentEvent.Retry = value
		if retryMs, err := time.ParseDuration(value + "ms"); err == nil {
			sp.retryInterval = retryMs
		}
	}

	return currentEvent
}

// isCompleteEvent checks if an SSE event is complete
func (sp *SSEProvider) isCompleteEvent(event *SSEEvent) bool {
	return event != nil && event.Data != ""
}

// processSSEEvent processes a complete SSE event
func (sp *SSEProvider) processSSEEvent(sseEvent *SSEEvent) {
	// Default event type is "message"
	eventType := sseEvent.Event
	if eventType == "" {
		eventType = "message"
	}

	switch eventType {
	case "event", "message":
		sp.handleEventData(sseEvent.Data)
	case "ownership":
		sp.handleOwnershipData(sseEvent.Data)
	case "ping", "heartbeat":
		sp.handleHeartbeat()
	default:
		sp.logger.WithField("event_type", eventType).Debug("Unknown SSE event type")
	}

	sp.incrementProcessedMetric()
}

// handleEventData processes real-time event data
func (sp *SSEProvider) handleEventData(data string) {
	var eventData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &eventData); err != nil {
		sp.logger.WithError(err).Error("Failed to parse SSE event data")
		sp.incrementErrorMetric()
		return
	}

	event := models.RealTimeEvent{
		EventType:    models.EventType(getString(eventData, "event_type")),
		PlayerID:     getUintPtr(eventData, "player_id"),
		GameID:       getStringPtr(eventData, "game_id"),
		TournamentID: getStringPtr(eventData, "tournament_id"),
		Timestamp:    time.Now(),
		Source:       sp.name,
		ImpactRating: getFloat64(eventData, "impact_rating"),
		Confidence:   getFloat64(eventData, "confidence"),
	}

	// Convert data to JSON
	dataBytes, _ := json.Marshal(eventData)
	event.Data = dataBytes

	// Send to event channel (non-blocking)
	select {
	case sp.eventChan <- event:
		sp.incrementReceivedMetric()
	default:
		sp.logger.Warn("SSE event channel full, dropping event")
		sp.incrementErrorMetric()
	}
}

// handleOwnershipData processes ownership update data
func (sp *SSEProvider) handleOwnershipData(data string) {
	var ownershipData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &ownershipData); err != nil {
		sp.logger.WithError(err).Error("Failed to parse SSE ownership data")
		sp.incrementErrorMetric()
		return
	}

	snapshot := models.OwnershipSnapshot{
		ContestID:    getString(ownershipData, "contest_id"),
		Timestamp:    time.Now(),
		TotalEntries: int(getFloat64(ownershipData, "total_entries")),
	}

	// Convert ownership data to JSON
	if playerOwnership, ok := ownershipData["player_ownership"]; ok {
		ownershipBytes, _ := json.Marshal(playerOwnership)
		snapshot.PlayerOwnership = ownershipBytes
	}

	// Send to ownership channel (non-blocking)
	select {
	case sp.ownershipChan <- snapshot:
	default:
		sp.logger.Warn("SSE ownership channel full, dropping update")
	}
}

// handleHeartbeat processes heartbeat/ping events
func (sp *SSEProvider) handleHeartbeat() {
	sp.lastEventTime = time.Now()
	sp.logger.Debug("Received SSE heartbeat")
}

// scheduleReconnect schedules a reconnection attempt
func (sp *SSEProvider) scheduleReconnect() {
	select {
	case sp.reconnectChan <- struct{}{}:
	default:
		// Reconnect already scheduled
	}
}

// cleanup closes the SSE connection
func (sp *SSEProvider) cleanup() {
	sp.connMutex.Lock()
	defer sp.connMutex.Unlock()

	if sp.response != nil {
		sp.response.Body.Close()
		sp.response = nil
	}
	sp.scanner = nil
	sp.isConnected = false
}

// Metrics helpers
func (sp *SSEProvider) incrementReceivedMetric() {
	sp.metricsMutex.Lock()
	sp.metrics.EventsReceived++
	sp.metrics.LastEventTime = time.Now()
	sp.metricsMutex.Unlock()
}

func (sp *SSEProvider) incrementProcessedMetric() {
	sp.metricsMutex.Lock()
	sp.metrics.EventsProcessed++
	sp.metricsMutex.Unlock()
}

func (sp *SSEProvider) incrementErrorMetric() {
	sp.metricsMutex.Lock()
	sp.metrics.EventsErrored++
	sp.metricsMutex.Unlock()
}

// Static data methods (implementing DataProvider interface)
func (sp *SSEProvider) GetPlayers(sport types.Sport, date string) ([]types.PlayerData, error) {
	return nil, fmt.Errorf("static data methods not implemented for SSE provider")
}

func (sp *SSEProvider) GetCurrentTournament() (*GolfTournamentData, error) {
	return nil, fmt.Errorf("static data methods not implemented for SSE provider")
}

func (sp *SSEProvider) GetTournamentSchedule() ([]GolfTournamentData, error) {
	return nil, fmt.Errorf("static data methods not implemented for SSE provider")
}