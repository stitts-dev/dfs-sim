package websocket

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now (should be restricted in production)
	},
}

// Client represents a WebSocket client
type Client struct {
	UserID uuid.UUID
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *Hub
}

// AnalyticsEvent represents real-time analytics events
type AnalyticsEvent struct {
	Type        string                 `json:"type"`
	UserID      uuid.UUID              `json:"user_id"`
	EventID     string                 `json:"event_id"`
	Category    string                 `json:"category"` // "portfolio", "ml", "performance"
	Data        interface{}            `json:"data"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   int64                  `json:"timestamp"`
	Progress    *ProgressData          `json:"progress,omitempty"`
}

// ProgressData represents progress information for long-running analytics
type ProgressData struct {
	Percentage int    `json:"percentage"`
	Stage      string `json:"stage"`
	Message    string `json:"message"`
	ETA        int64  `json:"eta_seconds,omitempty"`
}

// AnalyticsSubscription represents user subscription to analytics events
type AnalyticsSubscription struct {
	UserID     uuid.UUID `json:"user_id"`
	EventTypes []string  `json:"event_types"`
	Categories []string  `json:"categories"`
}

// Message represents a WebSocket message
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	userClients map[uuid.UUID][]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	logger     *logrus.Logger
	mutex      sync.RWMutex
	
	// Analytics-specific channels
	analyticsEvents    chan AnalyticsEvent
	subscriptions      map[uuid.UUID]map[string]bool // userID -> eventType -> subscribed
	eventBuffer        map[uuid.UUID][]AnalyticsEvent // Buffer events for offline users
	analyticsEnabled   bool
}

// NewHub creates a new WebSocket hub
func NewHub(logger *logrus.Logger) *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		userClients: make(map[uuid.UUID][]*Client),
		broadcast:   make(chan []byte, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		logger:      logger,
		
		// Analytics-specific initialization
		analyticsEvents:  make(chan AnalyticsEvent, 512),
		subscriptions:    make(map[uuid.UUID]map[string]bool),
		eventBuffer:      make(map[uuid.UUID][]AnalyticsEvent),
		analyticsEnabled: true,
	}
}

// Run starts the hub and handles client registration/unregistration
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.userClients[client.UserID] = append(h.userClients[client.UserID], client)
			h.mutex.Unlock()
			
			h.logger.WithFields(logrus.Fields{
				"user_id": client.UserID,
				"total_clients": len(h.clients),
			}).Info("WebSocket client connected")

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				
				// Remove from user clients
				userClients := h.userClients[client.UserID]
				for i, c := range userClients {
					if c == client {
						h.userClients[client.UserID] = append(userClients[:i], userClients[i+1:]...)
						break
					}
				}
				
				// Clean up empty user client slice
				if len(h.userClients[client.UserID]) == 0 {
					delete(h.userClients, client.UserID)
				}
			}
			h.mutex.Unlock()
			
			h.logger.WithFields(logrus.Fields{
				"user_id": client.UserID,
				"total_clients": len(h.clients),
			}).Info("WebSocket client disconnected")

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
			
		case analyticsEvent := <-h.analyticsEvents:
			if h.analyticsEnabled {
				h.handleAnalyticsEvent(analyticsEvent)
			}
		}
	}
}

// HandleWebSocket handles WebSocket connections
func (h *Hub) HandleWebSocket(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	client := &Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    h,
	}

	client.Hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// BroadcastToUser sends a message to all connections for a specific user
func (h *Hub) BroadcastToUser(userID uuid.UUID, message interface{}) {
	h.mutex.RLock()
	clients := h.userClients[userID]
	h.mutex.RUnlock()

	if len(clients) == 0 {
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal WebSocket message")
		return
	}

	h.mutex.RLock()
	for _, client := range clients {
		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(h.clients, client)
		}
	}
	h.mutex.RUnlock()
}

// BroadcastToAll sends a message to all connected clients
func (h *Hub) BroadcastToAll(message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal WebSocket message")
		return
	}

	h.broadcast <- data
}

// GetConnectedUsers returns the list of currently connected user IDs
func (h *Hub) GetConnectedUsers() []uuid.UUID {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	users := make([]uuid.UUID, 0, len(h.userClients))
	for userID := range h.userClients {
		users = append(users, userID)
	}
	return users
}

// GetConnectionCount returns the total number of active connections
func (h *Hub) GetConnectionCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.WithError(err).Error("WebSocket error")
			}
			break
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	defer c.Conn.Close()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.Hub.logger.WithError(err).Error("Failed to write WebSocket message")
				return
			}
		}
	}
}

// Analytics-specific methods

// SendAnalyticsEvent sends an analytics event to subscribed users
func (h *Hub) SendAnalyticsEvent(event AnalyticsEvent) {
	if !h.analyticsEnabled {
		return
	}
	
	select {
	case h.analyticsEvents <- event:
	default:
		h.logger.Warn("Analytics events channel is full, dropping event")
	}
}

// SubscribeToAnalytics subscribes a user to specific analytics event types
func (h *Hub) SubscribeToAnalytics(userID uuid.UUID, eventTypes []string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	if h.subscriptions[userID] == nil {
		h.subscriptions[userID] = make(map[string]bool)
	}
	
	for _, eventType := range eventTypes {
		h.subscriptions[userID][eventType] = true
	}
	
	h.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"event_types": eventTypes,
	}).Info("User subscribed to analytics events")
	
	// Send buffered events if any
	h.sendBufferedEvents(userID)
}

// UnsubscribeFromAnalytics unsubscribes a user from analytics events
func (h *Hub) UnsubscribeFromAnalytics(userID uuid.UUID, eventTypes []string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	if h.subscriptions[userID] == nil {
		return
	}
	
	for _, eventType := range eventTypes {
		delete(h.subscriptions[userID], eventType)
	}
	
	// Clean up empty subscription
	if len(h.subscriptions[userID]) == 0 {
		delete(h.subscriptions, userID)
	}
	
	h.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"event_types": eventTypes,
	}).Info("User unsubscribed from analytics events")
}

// handleAnalyticsEvent processes analytics events and sends to subscribed users
func (h *Hub) handleAnalyticsEvent(event AnalyticsEvent) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	// Check if specific user event
	if event.UserID != uuid.Nil {
		h.sendEventToUser(event.UserID, event)
		return
	}
	
	// Broadcast to all subscribed users
	for userID, subscriptions := range h.subscriptions {
		if subscriptions[event.Type] || subscriptions["all"] {
			h.sendEventToUser(userID, event)
		}
	}
}

// sendEventToUser sends an analytics event to a specific user
func (h *Hub) sendEventToUser(userID uuid.UUID, event AnalyticsEvent) {
	// Check if user is online
	clients := h.userClients[userID]
	if len(clients) == 0 {
		// Buffer event for offline user
		h.bufferEventForUser(userID, event)
		return
	}
	
	// Send to all user connections
	data, err := json.Marshal(Message{
		Type:    "analytics_event",
		Payload: event,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal analytics event")
		return
	}
	
	for _, client := range clients {
		select {
		case client.Send <- data:
		default:
			h.logger.WithField("user_id", userID).Warn("Client send channel full, dropping analytics event")
		}
	}
}

// bufferEventForUser buffers an analytics event for offline users
func (h *Hub) bufferEventForUser(userID uuid.UUID, event AnalyticsEvent) {
	const maxBufferSize = 100
	
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	buffer := h.eventBuffer[userID]
	buffer = append(buffer, event)
	
	// Keep only the most recent events
	if len(buffer) > maxBufferSize {
		buffer = buffer[len(buffer)-maxBufferSize:]
	}
	
	h.eventBuffer[userID] = buffer
}

// sendBufferedEvents sends buffered events to a user who just came online
func (h *Hub) sendBufferedEvents(userID uuid.UUID) {
	buffer := h.eventBuffer[userID]
	if len(buffer) == 0 {
		return
	}
	
	// Send all buffered events
	for _, event := range buffer {
		h.sendEventToUser(userID, event)
	}
	
	// Clear buffer
	delete(h.eventBuffer, userID)
	
	h.logger.WithFields(logrus.Fields{
		"user_id":       userID,
		"buffered_events": len(buffer),
	}).Info("Sent buffered analytics events to user")
}

// EnableAnalytics enables analytics event processing
func (h *Hub) EnableAnalytics() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.analyticsEnabled = true
	h.logger.Info("Analytics event processing enabled")
}

// DisableAnalytics disables analytics event processing
func (h *Hub) DisableAnalytics() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.analyticsEnabled = false
	h.logger.Info("Analytics event processing disabled")
}

// GetAnalyticsStats returns analytics hub statistics
func (h *Hub) GetAnalyticsStats() map[string]interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	totalBuffered := 0
	for _, buffer := range h.eventBuffer {
		totalBuffered += len(buffer)
	}
	
	return map[string]interface{}{
		"analytics_enabled":    h.analyticsEnabled,
		"subscribed_users":     len(h.subscriptions),
		"buffered_events":      totalBuffered,
		"analytics_channel_len": len(h.analyticsEvents),
	}
}