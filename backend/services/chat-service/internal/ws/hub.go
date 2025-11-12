package ws

import "sync"

type Hub struct {
	Rooms map[string]map[*Connection]bool
	mu    sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{Rooms: make(map[string]map[*Connection]bool)}
}

func (h *Hub) Register(chatID string, c *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.Rooms[chatID] == nil {
		h.Rooms[chatID] = make(map[*Connection]bool)
	}
	h.Rooms[chatID][c] = true
}

func (h *Hub) Unregister(chatID string, c *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.Rooms[chatID]; ok {
		delete(conns, c)
		if len(conns) == 0 {
			delete(h.Rooms, chatID)
		}
	}
}

func (h *Hub) Broadcast(chatID string, msg interface{}) {
	h.mu.RLock()
	conns := h.Rooms[chatID]
	h.mu.RUnlock()

	if conns == nil {
		return
	}

	for c := range conns {
		select {
		case c.Send <- msg:
		default:
			h.Unregister(chatID, c)
			close(c.Send)
		}
	}
}
