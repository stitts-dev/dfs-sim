package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jstittsworth/dfs-optimizer/internal/services"
	"github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, check against allowed origins
		return true
	},
}

type WebSocketHandler struct {
	hub       *services.WebSocketHub
	jwtSecret string
}

func NewWebSocketHandler(hub *services.WebSocketHub, jwtSecret string) *WebSocketHandler {
	return &WebSocketHandler{
		hub:       hub,
		jwtSecret: jwtSecret,
	}
}

// HandleWebSocket upgrades HTTP connection to WebSocket
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Get user ID from context (set by optional auth middleware)
	var userID uint
	if id, exists := c.Get("user_id"); exists {
		userID = id.(uint)
	} else {
		// Try to get user ID from query parameter for anonymous connections
		if userIDStr := c.Query("user_id"); userIDStr != "" {
			if id, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
				userID = uint(id)
			}
		}
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("Failed to upgrade connection: %v", err)
		return
	}

	// Create client
	client := services.NewClient(h.hub, conn, userID)

	// Register client
	h.hub.Register(client)

	// Send welcome message
	welcomeMsg := map[string]interface{}{
		"type": "welcome",
		"data": map[string]interface{}{
			"message":   "Connected to DFS Optimizer WebSocket",
			"user_id":   userID,
			"timestamp": time.Now().UTC(),
		},
	}

	if err := conn.WriteJSON(welcomeMsg); err != nil {
		logrus.Errorf("Failed to send welcome message: %v", err)
		conn.Close()
		return
	}

	// Allow collection of memory referenced by the caller by doing all work in new goroutines
	go client.WritePump()
	go client.ReadPump()
}

// Example WebSocket message handlers that could be called from other parts of the application

// BroadcastOptimizationProgress sends optimization progress to specific user
func BroadcastOptimizationProgress(hub *services.WebSocketHub, userID uint, progress map[string]interface{}) {
	hub.BroadcastToUser(userID, "optimization_progress", progress)
}

// BroadcastSimulationUpdate sends simulation updates to specific user
func BroadcastSimulationUpdate(hub *services.WebSocketHub, userID uint, update map[string]interface{}) {
	hub.BroadcastToUser(userID, "simulation_update", update)
}

// BroadcastContestUpdate sends contest updates to all subscribed users
func BroadcastContestUpdate(hub *services.WebSocketHub, contestID uint, update map[string]interface{}) {
	topic := "contest_" + strconv.FormatUint(uint64(contestID), 10)
	hub.BroadcastToTopic(topic, "contest_update", update)
}

// BroadcastLeaderboardUpdate sends leaderboard updates to contest participants
func BroadcastLeaderboardUpdate(hub *services.WebSocketHub, contestID uint, leaderboard map[string]interface{}) {
	topic := "leaderboard_" + strconv.FormatUint(uint64(contestID), 10)
	hub.BroadcastToTopic(topic, "leaderboard_update", leaderboard)
}

// BroadcastPlayerUpdate sends player updates (injuries, news) to all users
func BroadcastPlayerUpdate(hub *services.WebSocketHub, playerUpdate map[string]interface{}) {
	hub.BroadcastToTopic("players", "player_update", playerUpdate)
}
