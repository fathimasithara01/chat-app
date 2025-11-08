package ws

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/fathima-sithara/chat-service/internal/cache"
	"github.com/fathima-sithara/chat-service/internal/kafka"
	"github.com/fathima-sithara/chat-service/internal/repository"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Hub struct {
	clients  map[string]map[string]*websocket.Conn
	mu       sync.RWMutex
	repo     *repository.MongoRepository
	producer *kafka.Producer
	cache    *cache.Client
	stop     chan struct{}
}

func NewHub(repo *repository.MongoRepository, producer *kafka.Producer, cache *cache.Client) *Hub {
	return &Hub{clients: make(map[string]map[string]*websocket.Conn), repo: repo, producer: producer, cache: cache, stop: make(chan struct{})}
}

func (h *Hub) Run() {
	<-h.stop
	log.Info().Msg("hub stopped")
}

func (h *Hub) Close() { close(h.stop) }

func (h *Hub) HandleWebsocket(conn *websocket.Conn) {
	defer conn.Close()
	userID := conn.Query("user_id")
	if userID == "" {
		userID = uuid.NewString()
	}
	cid := uuid.NewString()

	h.mu.Lock()
	if _, ok := h.clients[userID]; !ok {
		h.clients[userID] = make(map[string]*websocket.Conn)
	}
	h.clients[userID][cid] = conn
	h.mu.Unlock()
	_ = h.cache.SetPresence(context.Background(), userID, true)

	log.Info().Str("user", userID).Str("conn", cid).Msg("ws connected")

	for {
		var msg map[string]any
		if err := conn.ReadJSON(&msg); err != nil {
			log.Error().Err(err).Msg("read json")
			break
		}
		if t, ok := msg["type"].(string); ok && t == "message" {
			b, _ := json.Marshal(msg)
			_ = h.producer.PublishMessage(context.Background(), userID, b)
			if to, ok := msg["to"].(string); ok {
				h.SendToUser(to, msg)
			}
		}
	}

	h.mu.Lock()
	delete(h.clients[userID], cid)
	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
		_ = h.cache.SetPresence(context.Background(), userID, false)
	}
	h.mu.Unlock()
}

func (h *Hub) SendToUser(userID string, payload any) {
	h.mu.RLock()
	conns, ok := h.clients[userID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	for _, c := range conns {
		_ = c.WriteJSON(payload)
	}
}

func (h *Hub) BroadcastJSON(payload any) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, conns := range h.clients {
		for _, c := range conns {
			_ = c.WriteJSON(payload)
		}
	}
}
