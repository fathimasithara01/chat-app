package ws

import (
	"sync"

	"github.com/gofiber/websocket/v2"
)

// Hub manages connected clients and chat rooms
type Hub struct {
	clients   map[string]*Client           // userID -> Client
	rooms     map[string]map[string]bool   // chatID -> map[userID]bool
	mu        sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]*Client),
		rooms:   make(map[string]map[string]bool),
	}
}

func (h *Hub) AddClient(userID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[userID] = c
}

func (h *Hub) RemoveClient(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, userID)
	for _, members := range h.rooms {
		delete(members, userID)
	}
}

func (h *Hub) JoinRoom(chatID, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.rooms[chatID]; !ok {
		h.rooms[chatID] = make(map[string]bool)
	}
	h.rooms[chatID][userID] = true
}

func (h *Hub) LeaveRoom(chatID, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if members, ok := h.rooms[chatID]; ok {
		delete(members, userID)
	}
}

func (h *Hub) Broadcast(chatID string, message any) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if members, ok := h.rooms[chatID]; ok {
		for userID := range members {
			if client, ok := h.clients[userID]; ok {
				client.Send(message)
			}
		}
	}
}

// Client represents a connected websocket client
type Client struct {
	UserID string
	Conn   *websocket.Conn
	send   chan any
}

func NewClient(userID string, conn *websocket.Conn) *Client {
	return &Client{
		UserID: userID,
		Conn:   conn,
		send:   make(chan any),
	}
}

func (c *Client) Send(msg any) {
	select {
	case c.send <- msg:
	default:
		// drop if blocked
	}
}

func (c *Client) WritePump() {
	for msg := range c.send {
		c.Conn.WriteJSON(msg)
	}
}
