package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Client represents a WebSocket client connection
type Client struct {
	ID     string
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *GatewayHub
}

// GatewayHub manages WebSocket connections for the API Gateway
type GatewayHub struct {
	// Registered clients
	clients map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to all clients
	broadcast chan []byte

	// User-specific message channels
	userChannels map[string][]*Client

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Logger
	logger *logrus.Logger
}

// Message represents a WebSocket message
type Message struct {
	Type      string      `json:"type"`
	UserID    string      `json:"user_id,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// OptimizationProgress represents optimization progress updates
type OptimizationProgress struct {
	RequestID       string  `json:"request_id"`
	Status          string  `json:"status"`
	Progress        float64 `json:"progress"`
	LineupsGenerated int     `json:"lineups_generated"`
	TotalLineups    int     `json:"total_lineups"`
	Message         string  `json:"message,omitempty"`
	Error           string  `json:"error,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// NewGatewayHub creates a new WebSocket hub
func NewGatewayHub(logger *logrus.Logger) *GatewayHub {
	return &GatewayHub{
		clients:      make(map[*Client]bool),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		broadcast:    make(chan []byte),
		userChannels: make(map[string][]*Client),
		logger:       logger,
	}
}

// Run starts the hub and handles client registration/unregistration
func (h *GatewayHub) Run() {
	h.logger.Info("Starting WebSocket hub")
	
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			
			// Add to user-specific channels
			if client.UserID != "" {
				h.userChannels[client.UserID] = append(h.userChannels[client.UserID], client)
			}
			h.mu.Unlock()
			
			h.logger.WithFields(logrus.Fields{
				"client_id": client.ID,
				"user_id":   client.UserID,
			}).Info("Client registered")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				
				// Remove from user-specific channels
				if client.UserID != "" {
					if clients, exists := h.userChannels[client.UserID]; exists {
						for i, c := range clients {
							if c == client {
								h.userChannels[client.UserID] = append(clients[:i], clients[i+1:]...)
								break
							}
						}
						// Clean up empty slices
						if len(h.userChannels[client.UserID]) == 0 {
							delete(h.userChannels, client.UserID)
						}
					}
				}
			}
			h.mu.Unlock()
			
			h.logger.WithFields(logrus.Fields{
				"client_id": client.ID,
				"user_id":   client.UserID,
			}).Info("Client unregistered")

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// HandleOptimizationProgress handles WebSocket connections for optimization progress
func (h *GatewayHub) HandleOptimizationProgress(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	// Create client
	client := &Client{
		ID:     generateClientID(),
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    h,
	}

	// Register client
	h.register <- client

	// Start goroutines for handling read/write
	go client.writePump()
	go client.readPump()
}

// SendToUser sends a message to all connections for a specific user
func (h *GatewayHub) SendToUser(userID string, message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal message")
		return
	}

	h.mu.RLock()
	clients, exists := h.userChannels[userID]
	h.mu.RUnlock()

	if !exists {
		h.logger.WithField("user_id", userID).Debug("No active connections for user")
		return
	}

	for _, client := range clients {
		select {
		case client.Send <- data:
		default:
			h.logger.WithField("user_id", userID).Warn("Failed to send message to client")
		}
	}
}

// SendOptimizationProgress sends optimization progress to a specific user
func (h *GatewayHub) SendOptimizationProgress(userID string, progress OptimizationProgress) {
	message := Message{
		Type:      "optimization_progress",
		UserID:    userID,
		Data:      progress,
		Timestamp: getCurrentTimestamp(),
	}

	h.SendToUser(userID, message)
}

// Broadcast sends a message to all connected clients
func (h *GatewayHub) Broadcast(message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal broadcast message")
		return
	}

	h.broadcast <- data
}

// readPump handles reading messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	// Set read deadline and pong handler for keepalive
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

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

// writePump handles writing messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
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

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
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

// Helper functions
func generateClientID() string {
	// Simple client ID generation - in production, use UUID
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}