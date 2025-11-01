package websocket

import (
	"context"
	"sync"
)

type Hub struct {
	clientsByConv map[string]map[*Client]struct{} // convID -> set of clients
	clientsByUser map[string]map[*Client]struct{} // userUUID -> set of clients
	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clientsByConv: make(map[string]map[*Client]struct{}),
		clientsByUser: make(map[string]map[*Client]struct{}),
	}
}

func (h *Hub) AddClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clientsByConv[c.ConvID]; !ok {
		h.clientsByConv[c.ConvID] = make(map[*Client]struct{})
	}
	h.clientsByConv[c.ConvID][c] = struct{}{}

	if _, ok := h.clientsByUser[c.UserUUID]; !ok {
		h.clientsByUser[c.UserUUID] = make(map[*Client]struct{})
	}
	h.clientsByUser[c.UserUUID][c] = struct{}{}
}

func (h *Hub) RemoveClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clientsByConv[c.ConvID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clientsByConv, c.ConvID)
		}
	}
	if set, ok := h.clientsByUser[c.UserUUID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clientsByUser, c.UserUUID)
		}
	}
}

func (h *Hub) BroadcastToConversation(ctx context.Context, convID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if set, ok := h.clientsByConv[convID]; ok {
		for c := range set {
			select {
			case c.Send <- msg:
			default:
				// drop slow clients (could close)
			}
		}
	}
}

func (h *Hub) SendToUser(ctx context.Context, userUUID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if set, ok := h.clientsByUser[userUUID]; ok {
		for c := range set {
			select {
			case c.Send <- msg:
			default:
			}
		}
	}
}
