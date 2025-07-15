package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stitts-dev/dfs-sim/services/ai-recommendations-service/internal/models"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now (should be restricted in production)
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client represents a WebSocket client for AI recommendations
type Client struct {
	UserID     string
	ContestIDs []uint // Contests this client is interested in
	Conn       *websocket.Conn
	Send       chan []byte
	Hub        *RecommendationHub
	LastSeen   time.Time
}

// RecommendationHub maintains active WebSocket connections for AI recommendations
type RecommendationHub struct {
	clients        map[*Client]bool
	userClients    map[string][]*Client
	contestClients map[uint][]*Client
	broadcast      chan *models.RecommendationUpdate
	register       chan *Client
	unregister     chan *Client
	logger         *logrus.Logger
	mutex          sync.RWMutex
}

// RecommendationMessage represents different types of messages sent to clients
type RecommendationMessage struct {
	Type      string      `json:"type"`      // "recommendation", "insight", "ownership_alert", "late_swap", "error"
	Data      interface{} `json:"data"`      // The actual message data
	Timestamp time.Time   `json:"timestamp"`
	UserID    string      `json:"user_id,omitempty"`
	ContestID uint        `json:"contest_id,omitempty"`
}

// NewRecommendationHub creates a new AI recommendations WebSocket hub
func NewRecommendationHub(logger *logrus.Logger) *RecommendationHub {
	return &RecommendationHub{
		clients:        make(map[*Client]bool),
		userClients:    make(map[string][]*Client),
		contestClients: make(map[uint][]*Client),
		broadcast:      make(chan *models.RecommendationUpdate, 256),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		logger:         logger,
	}
}

// Run starts the hub and handles client registration/unregistration
func (h *RecommendationHub) Run() {
	ticker := time.NewTicker(30 * time.Second) // Ping clients every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case update := <-h.broadcast:
			h.broadcastUpdate(update)

		case <-ticker.C:
			h.pingClients()
		}
	}
}

// registerClient adds a new client to the hub
func (h *RecommendationHub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.clients[client] = true
	h.userClients[client.UserID] = append(h.userClients[client.UserID], client)

	// Register client for specific contests
	for _, contestID := range client.ContestIDs {
		h.contestClients[contestID] = append(h.contestClients[contestID], client)
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":       client.UserID,
		"contest_ids":   client.ContestIDs,
		"total_clients": len(h.clients),
	}).Info("AI recommendations WebSocket client connected")

	// Send welcome message
	welcomeMsg := &RecommendationMessage{
		Type:      "connected",
		Data:      map[string]interface{}{"message": "Connected to AI recommendations service"},
		Timestamp: time.Now(),
		UserID:    client.UserID,
	}
	h.sendToClient(client, welcomeMsg)
}

// unregisterClient removes a client from the hub
func (h *RecommendationHub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

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

		// Remove from contest clients
		for _, contestID := range client.ContestIDs {
			contestClients := h.contestClients[contestID]
			for i, c := range contestClients {
				if c == client {
					h.contestClients[contestID] = append(contestClients[:i], contestClients[i+1:]...)
					break
				}
			}

			// Clean up empty contest client slice
			if len(h.contestClients[contestID]) == 0 {
				delete(h.contestClients, contestID)
			}
		}

		h.logger.WithFields(logrus.Fields{
			"user_id":       client.UserID,
			"total_clients": len(h.clients),
		}).Info("AI recommendations WebSocket client disconnected")
	}
}

// broadcastUpdate sends updates to relevant clients
func (h *RecommendationHub) broadcastUpdate(update *models.RecommendationUpdate) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	message := &RecommendationMessage{
		Type:      update.Type,
		Data:      update.Data,
		Timestamp: update.Timestamp,
		UserID:    update.UserID,
		ContestID: 0, // Will be set per client
	}

	// Send to specific user if specified
	if update.UserID != "" {
		clients := h.userClients[update.UserID]
		for _, client := range clients {
			h.sendToClient(client, message)
		}
		return
	}

	// Send to all clients (general updates)
	for client := range h.clients {
		h.sendToClient(client, message)
	}
}

// sendToClient sends a message to a specific client
func (h *RecommendationHub) sendToClient(client *Client, message *RecommendationMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal WebSocket message")
		return
	}

	select {
	case client.Send <- data:
		client.LastSeen = time.Now()
	default:
		// Client's send channel is full, close the connection
		h.unregister <- client
	}
}

// pingClients sends ping messages to check client health
func (h *RecommendationHub) pingClients() {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	now := time.Now()
	staleClients := []*Client{}

	for client := range h.clients {
		// Check if client is stale (no activity for 2 minutes)
		if now.Sub(client.LastSeen) > 2*time.Minute {
			staleClients = append(staleClients, client)
		}
	}

	// Remove stale clients
	for _, client := range staleClients {
		h.unregister <- client
	}

	if len(staleClients) > 0 {
		h.logger.WithField("stale_clients", len(staleClients)).Debug("Removed stale WebSocket clients")
	}
}

// HandleWebSocket handles WebSocket connections for AI recommendations
func (h *RecommendationHub) HandleWebSocket(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Parse contest IDs from query parameters
	contestIDsParam := c.Query("contest_ids")
	var contestIDs []uint
	if contestIDsParam != "" {
		// Parse comma-separated contest IDs
		// This is simplified - in production you'd want better parsing
		contestIDs = []uint{1, 2, 3} // Placeholder
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade AI recommendations WebSocket connection")
		return
	}

	client := &Client{
		UserID:     userID,
		ContestIDs: contestIDs,
		Conn:       conn,
		Send:       make(chan []byte, 256),
		Hub:        h,
		LastSeen:   time.Now(),
	}

	client.Hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// BroadcastInsight sends AI insights to specific users
func (h *RecommendationHub) BroadcastInsight(userID string, update *models.RecommendationUpdate) {
	h.broadcast <- update
}

// BroadcastToContest sends updates to all users following a specific contest
func (h *RecommendationHub) BroadcastToContest(contestID uint, update *models.RecommendationUpdate) {
	h.mutex.RLock()
	clients := h.contestClients[contestID]
	h.mutex.RUnlock()

	if len(clients) == 0 {
		return
	}

	message := &RecommendationMessage{
		Type:      update.Type,
		Data:      update.Data,
		Timestamp: update.Timestamp,
		ContestID: contestID,
	}

	for _, client := range clients {
		h.sendToClient(client, message)
	}

	h.logger.WithFields(logrus.Fields{
		"contest_id":   contestID,
		"client_count": len(clients),
		"update_type":  update.Type,
	}).Debug("Broadcast update to contest clients")
}

// BroadcastOwnershipAlert sends ownership change alerts
func (h *RecommendationHub) BroadcastOwnershipAlert(contestID uint, playerID uint, ownershipData interface{}) {
	update := &models.RecommendationUpdate{
		Type:      "ownership_alert",
		PlayerID:  playerID,
		Data:      ownershipData,
		Timestamp: time.Now(),
	}

	h.BroadcastToContest(contestID, update)
}

// BroadcastLateSwapAlert sends late swap recommendations
func (h *RecommendationHub) BroadcastLateSwapAlert(contestID uint, recommendation interface{}) {
	update := &models.RecommendationUpdate{
		Type:      "late_swap",
		Data:      recommendation,
		Timestamp: time.Now(),
	}

	h.BroadcastToContest(contestID, update)
}

// BroadcastPlayerAlert sends player-specific alerts (injury, news, etc.)
func (h *RecommendationHub) BroadcastPlayerAlert(playerID uint, alertType string, alertData interface{}) {
	update := &models.RecommendationUpdate{
		Type:      alertType,
		PlayerID:  playerID,
		Data:      alertData,
		Timestamp: time.Now(),
	}

	h.broadcast <- update
}

// GetConnectedUsers returns the list of currently connected user IDs
func (h *RecommendationHub) GetConnectedUsers() []string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	users := make([]string, 0, len(h.userClients))
	for userID := range h.userClients {
		users = append(users, userID)
	}
	return users
}

// GetConnectionCount returns the total number of active connections
func (h *RecommendationHub) GetConnectionCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// GetContestSubscribers returns the number of clients subscribed to a contest
func (h *RecommendationHub) GetContestSubscribers(contestID uint) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.contestClients[contestID])
}

// GetHubStats returns statistics about the hub
func (h *RecommendationHub) GetHubStats() map[string]interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_clients":     len(h.clients),
		"unique_users":      len(h.userClients),
		"contests_tracked":  len(h.contestClients),
		"uptime_seconds":    time.Now().Unix(), // Placeholder
	}

	// Add per-contest stats
	contestStats := make(map[uint]int)
	for contestID, clients := range h.contestClients {
		contestStats[contestID] = len(clients)
	}
	stats["contest_subscribers"] = contestStats

	return stats
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	// Set read deadline and pong handler for keep-alive
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.LastSeen = time.Now()
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.WithError(err).Error("AI recommendations WebSocket error")
			}
			break
		}

		// Handle incoming messages (client can send subscription updates, etc.)
		c.handleIncomingMessage(message)
		c.LastSeen = time.Now()
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second) // Send ping every 54 seconds
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.Hub.logger.WithError(err).Error("Failed to write AI recommendations WebSocket message")
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleIncomingMessage processes messages sent by the client
func (c *Client) handleIncomingMessage(message []byte) {
	var clientMsg map[string]interface{}
	if err := json.Unmarshal(message, &clientMsg); err != nil {
		c.Hub.logger.WithError(err).Warn("Failed to parse client message")
		return
	}

	msgType, ok := clientMsg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "subscribe_contest":
		// Client wants to subscribe to a specific contest
		if contestIDFloat, ok := clientMsg["contest_id"].(float64); ok {
			contestID := uint(contestIDFloat)
			c.Hub.mutex.Lock()
			// Add to contest clients if not already there
			found := false
			for _, id := range c.ContestIDs {
				if id == contestID {
					found = true
					break
				}
			}
			if !found {
				c.ContestIDs = append(c.ContestIDs, contestID)
				c.Hub.contestClients[contestID] = append(c.Hub.contestClients[contestID], c)
			}
			c.Hub.mutex.Unlock()
			
			c.Hub.logger.WithFields(logrus.Fields{
				"user_id":    c.UserID,
				"contest_id": contestID,
			}).Debug("Client subscribed to contest")
		}

	case "unsubscribe_contest":
		// Client wants to unsubscribe from a specific contest
		if contestIDFloat, ok := clientMsg["contest_id"].(float64); ok {
			contestID := uint(contestIDFloat)
			c.Hub.mutex.Lock()
			
			// Remove from contest IDs
			for i, id := range c.ContestIDs {
				if id == contestID {
					c.ContestIDs = append(c.ContestIDs[:i], c.ContestIDs[i+1:]...)
					break
				}
			}
			
			// Remove from contest clients
			contestClients := c.Hub.contestClients[contestID]
			for i, client := range contestClients {
				if client == c {
					c.Hub.contestClients[contestID] = append(contestClients[:i], contestClients[i+1:]...)
					break
				}
			}
			
			c.Hub.mutex.Unlock()
			
			c.Hub.logger.WithFields(logrus.Fields{
				"user_id":    c.UserID,
				"contest_id": contestID,
			}).Debug("Client unsubscribed from contest")
		}

	case "ping":
		// Respond to client ping
		response := &RecommendationMessage{
			Type:      "pong",
			Data:      map[string]interface{}{"timestamp": time.Now().Unix()},
			Timestamp: time.Now(),
		}
		c.Hub.sendToClient(c, response)
	}
}