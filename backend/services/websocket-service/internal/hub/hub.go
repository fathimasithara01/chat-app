package hub

import (
	"context"
	"sync"
	"time"
)

type Hub struct {
	clientsByConv map[string]map[*Client]struct{}
	// userUUID -> set of clients
	clientsByUser map[string]map[*Client]struct{}
	mu            sync.RWMutex

	// publish function for cross-instance broadcasting (optional)
	PublishToOtherInstances func(ctx context.Context, channel string, payload []byte) error
}

type Client = struct {
	Conn      interface{}
	UserUUID  string
	ConvID    string
	Send      chan []byte
	Connected time.Time
}

func NewHub() *Hub {
	return &Hub{
		clientsByConv: make(map[string]map[*Client]struct{}),
		clientsByUser: make(map[string]map[*Client]struct{}),
	}
}

// Add a client instance
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

// Remove a client
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

// Broadcast to all clients in a conversation (local instance)
func (h *Hub) BroadcastToConversation(ctx context.Context, convID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if set, ok := h.clientsByConv[convID]; ok {
		for c := range set {
			select {
			case c.Send <- msg:
			default:
				// client moving slow — drop or handle backpressure
			}
		}
	}
	// also publish to other instances for multi-instance delivery
	if h.PublishToOtherInstances != nil {
		_ = h.PublishToOtherInstances(ctx, convID, msg)
	}
}

// Send to a user across all their sockets
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
	// also publish for other instances
	if h.PublishToOtherInstances != nil {
		_ = h.PublishToOtherInstances(ctx, "user:"+userUUID, msg)
	}
}
