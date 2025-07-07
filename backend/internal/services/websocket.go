package services

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type WebSocketHub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

type Client struct {
	hub    *WebSocketHub
	conn   *websocket.Conn
	send   chan []byte
	userID uint
	topics map[string]bool
}

type WebSocketMessage struct {
	Type      string          `json:"type"`
	Topic     string          `json:"topic"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

type Subscription struct {
	Action string   `json:"action"` // "subscribe" or "unsubscribe"
	Topics []string `json:"topics"`
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			logrus.Infof("Client registered: user_id=%d", client.userID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.mu.Unlock()
				logrus.Infof("Client unregistered: user_id=%d", client.userID)
			} else {
				h.mu.Unlock()
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is full, close it
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a new client to the hub
func (h *WebSocketHub) Register(client *Client) {
	h.register <- client
}

func (h *WebSocketHub) BroadcastToTopic(topic string, messageType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	message := WebSocketMessage{
		Type:      messageType,
		Topic:     topic,
		Data:      jsonData,
		Timestamp: time.Now().UTC(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.IsSubscribedTo(topic) {
			select {
			case client.send <- messageBytes:
			default:
				// Skip if client's buffer is full
			}
		}
	}

	return nil
}

func (h *WebSocketHub) BroadcastToUser(userID uint, messageType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	message := WebSocketMessage{
		Type:      messageType,
		Topic:     "user",
		Data:      jsonData,
		Timestamp: time.Now().UTC(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.userID == userID {
			select {
			case client.send <- messageBytes:
			default:
				// Skip if client's buffer is full
			}
		}
	}

	return nil
}

func NewClient(hub *WebSocketHub, conn *websocket.Conn, userID uint) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
		topics: make(map[string]bool),
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var sub Subscription
		err := c.conn.ReadJSON(&sub)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Errorf("WebSocket error: %v", err)
			}
			break
		}

		// Handle subscription changes
		if sub.Action == "subscribe" {
			for _, topic := range sub.Topics {
				c.topics[topic] = true
			}
		} else if sub.Action == "unsubscribe" {
			for _, topic := range sub.Topics {
				delete(c.topics, topic)
			}
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) IsSubscribedTo(topic string) bool {
	return c.topics[topic] || c.topics["*"] // "*" subscribes to all topics
}
