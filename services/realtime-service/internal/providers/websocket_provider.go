package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"

	"github.com/stitts-dev/dfs-sim/services/realtime-service/internal/models"
	"github.com/stitts-dev/dfs-sim/shared/types"
)

// WebSocketProvider implements real-time data streaming via WebSocket
type WebSocketProvider struct {
	name                string
	wsURL               string
	headers             map[string]string
	supportedSports     []string
	
	// Connection management
	conn                *websocket.Conn
	isConnected         bool
	connMutex           sync.RWMutex
	
	// Subscription management
	subscriptions       map[string]*Subscription
	subMutex            sync.RWMutex
	
	// Event channels
	eventChan           chan models.RealTimeEvent
	ownershipChan       chan models.OwnershipSnapshot
	
	// Configuration
	config              *RealTimeConfig
	logger              *logrus.Logger
	circuitBreaker      *gobreaker.CircuitBreaker
	
	// Metrics
	metrics             *ProviderMetrics
	metricsMutex        sync.Mutex
	
	// Control channels
	stopChan            chan struct{}
	reconnectChan       chan struct{}
	
	// Health monitoring
	lastPing            time.Time
	connectionStartTime time.Time
	reconnectAttempts   int
}

// NewWebSocketProvider creates a new WebSocket real-time provider
func NewWebSocketProvider(name, wsURL string, config *RealTimeConfig, logger *logrus.Logger) *WebSocketProvider {
	// Create circuit breaker
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        fmt.Sprintf("websocket-provider-%s", name),
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
			}).Warn("WebSocket provider circuit breaker state changed")
		},
	})

	return &WebSocketProvider{
		name:            name,
		wsURL:           wsURL,
		headers:         make(map[string]string),
		supportedSports: []string{"golf"}, // Default to golf
		subscriptions:   make(map[string]*Subscription),
		eventChan:       make(chan models.RealTimeEvent, config.EventBufferSize),
		ownershipChan:   make(chan models.OwnershipSnapshot, config.EventBufferSize),
		config:          config,
		logger:          logger,
		circuitBreaker:  cb,
		metrics:         &ProviderMetrics{},
		stopChan:        make(chan struct{}),
		reconnectChan:   make(chan struct{}, 1),
	}
}

// GetProviderName returns the provider name
func (wp *WebSocketProvider) GetProviderName() string {
	return wp.name
}

// GetSupportedSports returns supported sports
func (wp *WebSocketProvider) GetSupportedSports() []string {
	return wp.supportedSports
}

// IsHealthy returns the provider health status
func (wp *WebSocketProvider) IsHealthy() bool {
	wp.connMutex.RLock()
	defer wp.connMutex.RUnlock()
	return wp.isConnected && time.Since(wp.lastPing) < 60*time.Second
}

// Connect establishes WebSocket connection with circuit breaker protection
func (wp *WebSocketProvider) Connect(ctx context.Context) error {
	_, err := wp.circuitBreaker.Execute(func() (interface{}, error) {
		return nil, wp.doConnect(ctx)
	})
	return err
}

// doConnect performs the actual WebSocket connection
func (wp *WebSocketProvider) doConnect(ctx context.Context) error {
	wp.logger.WithField("provider", wp.name).Info("Connecting to WebSocket provider")

	dialer := websocket.Dialer{
		HandshakeTimeout: wp.config.ConnectionTimeout,
	}

	// Add authentication headers
	header := http.Header{}
	for key, value := range wp.headers {
		header.Set(key, value)
	}

	conn, _, err := dialer.DialContext(ctx, wp.wsURL, header)
	if err != nil {
		wp.incrementErrorMetric()
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	wp.connMutex.Lock()
	wp.conn = conn
	wp.isConnected = true
	wp.connectionStartTime = time.Now()
	wp.reconnectAttempts = 0
	wp.lastPing = time.Now()
	wp.connMutex.Unlock()

	wp.logger.WithField("provider", wp.name).Info("WebSocket connection established")

	// Start message handling goroutines
	go wp.readMessages()
	go wp.pingHandler()
	
	return nil
}

// Subscribe creates a new subscription for real-time events
func (wp *WebSocketProvider) Subscribe(ctx context.Context, subscription *Subscription) (<-chan models.RealTimeEvent, error) {
	if !wp.IsHealthy() {
		return nil, fmt.Errorf("provider %s is not healthy", wp.name)
	}

	wp.subMutex.Lock()
	wp.subscriptions[subscription.ID] = subscription
	wp.subMutex.Unlock()

	// Send subscription message to WebSocket
	subMessage := map[string]interface{}{
		"type":        "subscribe",
		"id":          subscription.ID,
		"event_types": subscription.EventTypes,
		"sports":      subscription.Sports,
		"contest_ids": subscription.ContestIDs,
		"player_ids":  subscription.PlayerIDs,
	}

	if err := wp.sendMessage(subMessage); err != nil {
		return nil, fmt.Errorf("failed to send subscription: %w", err)
	}

	wp.logger.WithFields(logrus.Fields{
		"provider":       wp.name,
		"subscription_id": subscription.ID,
		"event_types":    subscription.EventTypes,
	}).Info("Created WebSocket subscription")

	return wp.eventChan, nil
}

// Unsubscribe removes a subscription
func (wp *WebSocketProvider) Unsubscribe(ctx context.Context, subscriptionID string) error {
	wp.subMutex.Lock()
	delete(wp.subscriptions, subscriptionID)
	wp.subMutex.Unlock()

	// Send unsubscribe message
	unsubMessage := map[string]interface{}{
		"type": "unsubscribe",
		"id":   subscriptionID,
	}

	return wp.sendMessage(unsubMessage)
}

// GetActiveSubscriptions returns all active subscriptions
func (wp *WebSocketProvider) GetActiveSubscriptions() []Subscription {
	wp.subMutex.RLock()
	defer wp.subMutex.RUnlock()

	subscriptions := make([]Subscription, 0, len(wp.subscriptions))
	for _, sub := range wp.subscriptions {
		subscriptions = append(subscriptions, *sub)
	}
	return subscriptions
}

// StreamEvents streams real-time events
func (wp *WebSocketProvider) StreamEvents(ctx context.Context, eventTypes []models.EventType) (<-chan models.RealTimeEvent, error) {
	subscription := &Subscription{
		ID:         fmt.Sprintf("stream-%d", time.Now().UnixNano()),
		EventTypes: eventTypes,
		Sports:     wp.supportedSports,
		CreatedAt:  time.Now(),
		IsActive:   true,
	}

	return wp.Subscribe(ctx, subscription)
}

// StreamOwnership streams ownership data
func (wp *WebSocketProvider) StreamOwnership(ctx context.Context, contestIDs []string) (<-chan models.OwnershipSnapshot, error) {
	// Create ownership subscription
	subMessage := map[string]interface{}{
		"type":        "subscribe_ownership",
		"contest_ids": contestIDs,
	}

	if err := wp.sendMessage(subMessage); err != nil {
		return nil, fmt.Errorf("failed to subscribe to ownership: %w", err)
	}

	return wp.ownershipChan, nil
}

// GetConnectionStatus returns connection health information
func (wp *WebSocketProvider) GetConnectionStatus() ConnectionStatus {
	wp.connMutex.RLock()
	defer wp.connMutex.RUnlock()

	status := ConnectionStatus{
		IsConnected:       wp.isConnected,
		LastConnected:     wp.connectionStartTime,
		ReconnectAttempts: wp.reconnectAttempts,
	}

	if wp.isConnected {
		status.ConnectionUptime = time.Since(wp.connectionStartTime)
		status.LatencyMs = float64(time.Since(wp.lastPing).Milliseconds())
	}

	return status
}

// GetMetrics returns provider performance metrics
func (wp *WebSocketProvider) GetMetrics() ProviderMetrics {
	wp.metricsMutex.Lock()
	defer wp.metricsMutex.Unlock()
	return *wp.metrics
}

// readMessages handles incoming WebSocket messages
func (wp *WebSocketProvider) readMessages() {
	defer wp.cleanup()

	for {
		select {
		case <-wp.stopChan:
			return
		default:
			wp.connMutex.RLock()
			conn := wp.conn
			wp.connMutex.RUnlock()

			if conn == nil {
				wp.scheduleReconnect()
				return
			}

			// Set read deadline
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			_, messageBytes, err := conn.ReadMessage()
			if err != nil {
				wp.logger.WithError(err).Error("Failed to read WebSocket message")
				wp.incrementErrorMetric()
				wp.scheduleReconnect()
				return
			}

			wp.lastPing = time.Now()
			wp.processMessage(messageBytes)
		}
	}
}

// processMessage handles individual WebSocket messages
func (wp *WebSocketProvider) processMessage(messageBytes []byte) {
	var message map[string]interface{}
	if err := json.Unmarshal(messageBytes, &message); err != nil {
		wp.logger.WithError(err).Error("Failed to parse WebSocket message")
		wp.incrementErrorMetric()
		return
	}

	messageType, ok := message["type"].(string)
	if !ok {
		wp.logger.Error("Missing message type in WebSocket message")
		return
	}

	switch messageType {
	case "event":
		wp.handleEventMessage(message)
	case "ownership":
		wp.handleOwnershipMessage(message)
	case "ping":
		wp.handlePingMessage()
	case "error":
		wp.handleErrorMessage(message)
	default:
		wp.logger.WithField("type", messageType).Debug("Unknown message type")
	}

	wp.incrementProcessedMetric()
}

// handleEventMessage processes real-time event messages
func (wp *WebSocketProvider) handleEventMessage(message map[string]interface{}) {
	data, ok := message["data"].(map[string]interface{})
	if !ok {
		wp.logger.Error("Invalid event data format")
		return
	}

	event := models.RealTimeEvent{
		EventType:    models.EventType(getString(data, "event_type")),
		PlayerID:     getUintPtr(data, "player_id"),
		GameID:       getStringPtr(data, "game_id"),
		TournamentID: getStringPtr(data, "tournament_id"),
		Timestamp:    time.Now(),
		Source:       wp.name,
		ImpactRating: getFloat64(data, "impact_rating"),
		Confidence:   getFloat64(data, "confidence"),
	}

	// Convert data to JSON
	dataBytes, _ := json.Marshal(data)
	event.Data = dataBytes

	// Send to event channel (non-blocking)
	select {
	case wp.eventChan <- event:
		wp.incrementReceivedMetric()
	default:
		wp.logger.Warn("Event channel full, dropping event")
		wp.incrementErrorMetric()
	}
}

// handleOwnershipMessage processes ownership update messages
func (wp *WebSocketProvider) handleOwnershipMessage(message map[string]interface{}) {
	data, ok := message["data"].(map[string]interface{})
	if !ok {
		wp.logger.Error("Invalid ownership data format")
		return
	}

	snapshot := models.OwnershipSnapshot{
		ContestID:    getString(data, "contest_id"),
		Timestamp:    time.Now(),
		TotalEntries: int(getFloat64(data, "total_entries")),
	}

	// Convert ownership data to JSON
	if playerOwnership, ok := data["player_ownership"]; ok {
		ownershipBytes, _ := json.Marshal(playerOwnership)
		snapshot.PlayerOwnership = ownershipBytes
	}

	// Send to ownership channel (non-blocking)
	select {
	case wp.ownershipChan <- snapshot:
	default:
		wp.logger.Warn("Ownership channel full, dropping update")
	}
}

// Helper functions for data extraction
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getStringPtr(data map[string]interface{}, key string) *string {
	if val, ok := data[key].(string); ok && val != "" {
		return &val
	}
	return nil
}

func getUintPtr(data map[string]interface{}, key string) *uint {
	if val, ok := data[key].(float64); ok {
		uintVal := uint(val)
		return &uintVal
	}
	return nil
}

func getFloat64(data map[string]interface{}, key string) float64 {
	if val, ok := data[key].(float64); ok {
		return val
	}
	return 0.0
}

// Utility methods
func (wp *WebSocketProvider) sendMessage(message interface{}) error {
	wp.connMutex.RLock()
	conn := wp.conn
	wp.connMutex.RUnlock()

	if conn == nil {
		return fmt.Errorf("no active connection")
	}

	return conn.WriteJSON(message)
}

func (wp *WebSocketProvider) handlePingMessage() {
	wp.lastPing = time.Now()
	// Send pong response
	pongMessage := map[string]string{"type": "pong"}
	wp.sendMessage(pongMessage)
}

func (wp *WebSocketProvider) handleErrorMessage(message map[string]interface{}) {
	errorMsg := getString(message, "message")
	wp.logger.WithField("error", errorMsg).Error("Received error from WebSocket provider")
	wp.incrementErrorMetric()
}

func (wp *WebSocketProvider) pingHandler() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wp.stopChan:
			return
		case <-ticker.C:
			if wp.IsHealthy() {
				pingMessage := map[string]string{"type": "ping"}
				if err := wp.sendMessage(pingMessage); err != nil {
					wp.logger.WithError(err).Error("Failed to send ping")
					wp.scheduleReconnect()
					return
				}
			}
		}
	}
}

func (wp *WebSocketProvider) scheduleReconnect() {
	select {
	case wp.reconnectChan <- struct{}{}:
	default:
		// Reconnect already scheduled
	}
}

func (wp *WebSocketProvider) cleanup() {
	wp.connMutex.Lock()
	if wp.conn != nil {
		wp.conn.Close()
		wp.conn = nil
	}
	wp.isConnected = false
	wp.connMutex.Unlock()
}

// Metrics helpers
func (wp *WebSocketProvider) incrementReceivedMetric() {
	wp.metricsMutex.Lock()
	wp.metrics.EventsReceived++
	wp.metrics.LastEventTime = time.Now()
	wp.metricsMutex.Unlock()
}

func (wp *WebSocketProvider) incrementProcessedMetric() {
	wp.metricsMutex.Lock()
	wp.metrics.EventsProcessed++
	wp.metricsMutex.Unlock()
}

func (wp *WebSocketProvider) incrementErrorMetric() {
	wp.metricsMutex.Lock()
	wp.metrics.EventsErrored++
	wp.metricsMutex.Unlock()
}

// Static data methods (implementing DataProvider interface)
func (wp *WebSocketProvider) GetPlayers(sport types.Sport, date string) ([]types.PlayerData, error) {
	return nil, fmt.Errorf("static data methods not implemented for WebSocket provider")
}

func (wp *WebSocketProvider) GetCurrentTournament() (*GolfTournamentData, error) {
	return nil, fmt.Errorf("static data methods not implemented for WebSocket provider")
}

func (wp *WebSocketProvider) GetTournamentSchedule() ([]GolfTournamentData, error) {
	return nil, fmt.Errorf("static data methods not implemented for WebSocket provider")
}