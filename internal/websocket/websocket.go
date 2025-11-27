package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/models"
	"github.com/abrezinsky/derbyvote/internal/services"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	log        logger.Logger
	clients    map[*Client]bool
	broadcast  chan models.WSMessage
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
	settings   services.SettingsServicer
}

// Client is a middleman between the websocket connection and the hub
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan models.WSMessage
}

// New creates a new Hub instance with injected dependencies
func New(log logger.Logger, settings services.SettingsServicer) *Hub {
	return &Hub{
		log:        log,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan models.WSMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		settings:   settings,
	}
}

// Start begins the hub's main loop in a goroutine
func (h *Hub) Start() {
	go h.run()
}

// run handles client registration/unregistration and message broadcasting
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			h.log.Debug("Client connected", "total_clients", len(h.clients))

			// Send current voting status to new client
			go func() {
				ctx := context.Background()
				votingOpen, _ := h.settings.IsVotingOpen(ctx)
				closeTime, _ := h.settings.GetSetting(ctx, "voting_close_time")

				client.send <- models.WSMessage{
					Type: "voting_status",
					Payload: map[string]interface{}{
						"open":       votingOpen,
						"close_time": closeTime,
					},
				}
			}()

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mutex.Unlock()
			h.log.Debug("Client disconnected", "total_clients", len(h.clients))

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is full, unregister
					go func(c *Client) {
						h.unregister <- c
					}(client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// BroadcastMessage sends a message to all connected clients
func (h *Hub) BroadcastMessage(msgType string, payload interface{}) {
	h.broadcast <- models.WSMessage{
		Type:    msgType,
		Payload: payload,
	}
}

// BroadcastVotingStatus implements services.Broadcaster
func (h *Hub) BroadcastVotingStatus(open bool, closeTime string) {
	h.BroadcastMessage("voting_status", map[string]interface{}{
		"open":       open,
		"close_time": closeTime,
	})
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
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
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.log.Debug("WebSocket error", "error", err)
			}
			break
		}

		// Handle incoming messages if needed
		var msg models.WSMessage
		if err := json.Unmarshal(message, &msg); err == nil {
			// Process message types here if needed
			c.hub.log.Debug("Received message", "type", msg.Type)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
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
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			msgBytes, _ := json.Marshal(message)
			w.Write(msgBytes)

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

// ServeWs handles websocket requests from clients
func (h *Hub) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("WebSocket upgrade error", "error", err)
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan models.WSMessage, 256),
	}
	h.register <- client

	// Allow collection of memory referenced by the caller by doing all work in new goroutines
	go client.writePump()
	go client.readPump()
}

// StartVotingCountdown starts the countdown timer goroutine with context-based cancellation
func (h *Hub) StartVotingCountdown(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.log.Info("Voting countdown stopped")
			return
		case <-ticker.C:
			h.checkAndUpdateCountdown()
		}
	}
}

// checkAndUpdateCountdown checks the timer and broadcasts updates
func (h *Hub) checkAndUpdateCountdown() {
	ctx := context.Background()
	closeTimeStr, err := h.settings.GetSetting(ctx, "voting_close_time")
	if err != nil || closeTimeStr == "" {
		return
	}

	closeTime, err := time.Parse(time.RFC3339, closeTimeStr)
	if err != nil {
		return
	}

	now := time.Now()
	if now.After(closeTime) {
		// Time's up! Close voting
		votingOpen, _ := h.settings.IsVotingOpen(ctx)
		if votingOpen {
			h.settings.SetVotingOpen(ctx, false)
			h.settings.SetSetting(ctx, "voting_close_time", "")
			h.log.Info("Voting automatically closed by timer")

			h.BroadcastMessage("voting_status", map[string]interface{}{
				"open":       false,
				"close_time": "",
			})
		}
	} else {
		// Send countdown update
		remaining := int(closeTime.Sub(now).Seconds())
		h.BroadcastMessage("countdown", map[string]interface{}{
			"seconds_remaining": remaining,
		})
	}
}
