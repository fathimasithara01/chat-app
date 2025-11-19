package ws

import (
	"sync"
	"time"
)

type Hub struct {
	rooms map[string]map[*Connection]bool
	mu    sync.RWMutex
}

func NewHub() *Hub { return &Hub{rooms: make(map[string]map[*Connection]bool)} }

func (h *Hub) Register(chatID string, c *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[chatID] == nil {
		h.rooms[chatID] = make(map[*Connection]bool)
	}
	h.rooms[chatID][c] = true
}

func (h *Hub) Unregister(chatID string, c *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.rooms[chatID]; ok {
		delete(conns, c)
		if len(conns) == 0 {
			delete(h.rooms, chatID)
		}
	}
}

func (h *Hub) Broadcast(chatID string, msg interface{}) {
	h.mu.RLock()
	conns := h.rooms[chatID]
	h.mu.RUnlock()
	if conns == nil {
		return
	}
	for c := range conns {
		select {
		case c.send <- msg:
		case <-time.After(200 * time.Millisecond):
			// slow consumer — drop and unregister
			h.Unregister(chatID, c)
			close(c.send)
		}
	}
}
