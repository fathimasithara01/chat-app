package ws

import (
	"sync"
	"time"
)

type Hub struct {
	rooms map[string]map[*Connection]bool
	mu    sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[*Connection]bool),
	}
}

func (h *Hub) Register(room string, c *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*Connection]bool)
	}
	h.rooms[room][c] = true
}

func (h *Hub) Unregister(room string, c *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.rooms[room]; ok {
		delete(conns, c)
		if len(conns) == 0 {
			delete(h.rooms, room)
		}
	}
}

func (h *Hub) Broadcast(room string, msg interface{}) {
	h.mu.RLock()
	conns := h.rooms[room]
	h.mu.RUnlock()
	if conns == nil {
		return
	}
	for c := range conns {
		select {
		case c.send <- msg:
		case <-time.After(200 * time.Millisecond):
			h.Unregister(room, c)
			close(c.send)
		}
	}
}
