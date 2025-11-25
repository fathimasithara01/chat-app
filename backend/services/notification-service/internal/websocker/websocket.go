package websocket

import (
	"sync"

	"github.com/gofiber/websocket/v2"
)

type Hub struct {
	Clients map[string]*websocket.Conn
	mu      sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		Clients: make(map[string]*websocket.Conn),
	}
}

func (h *Hub) Add(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	h.Clients[userID] = conn
	h.mu.Unlock()
}

func (h *Hub) Send(userID string, msg []byte) {
	h.mu.Lock()
	conn, ok := h.Clients[userID]
	h.mu.Unlock()

	if ok {
		conn.WriteMessage(websocket.TextMessage, msg)
	}
}
