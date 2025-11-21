package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/fathima-sithara/websocket/internal/config"
	"github.com/redis/go-redis/v9"
)

// Hub manages clients and rooms and connects to Redis for pub/sub.
type Hub struct {
	rooms map[string]map[*Client]bool
	users map[string]*Client
	mu    sync.RWMutex
	cfg   *config.Config
	rdb   *redis.Client

	// optional local broadcast queue (not strictly necessary but useful if you want to buffer)
	localBroadcast chan *Envelope
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewHub creates hub and starts redis subscriber.
func NewHub(rdb *redis.Client, cfg *config.Config) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	h := &Hub{
		rooms:          make(map[string]map[*Client]bool),
		users:          make(map[string]*Client),
		cfg:            cfg,
		rdb:            rdb,
		localBroadcast: make(chan *Envelope, 1024),
		ctx:            ctx,
		cancel:         cancel,
	}

	// local dispatcher
	go h.localDispatcher()

	// redis subscriber for cross-node messages
	go h.subscribeRedis("ws:global")

	return h
}

func (h *Hub) localDispatcher() {
	for {
		select {
		case env := <-h.localBroadcast:
			h.BroadcastLocal(env.ChatID, env)
		case <-h.ctx.Done():
			return
		}
	}
}

// subscribeRedis subscribes to a channel and routes payloads to local hub.
func (h *Hub) subscribeRedis(channel string) {
	pubsub := h.rdb.Subscribe(context.Background(), channel)
	ch := pubsub.Channel()
	for {
		select {
		case <-h.ctx.Done():
			_ = pubsub.Close()
			return
		case msg, ok := <-ch:
			if !ok {
				// subscription closed, attempt to resubscribe with delay
				log.Println("redis subscription closed, exiting subscriber goroutine")
				return
			}
			var env Envelope
			if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
				continue
			}
			// deliver locally
			h.BroadcastLocal(env.ChatID, &env)
		}
	}
}

// publishRedis publishes raw bytes to channel (used for cross-node broadcasting).
func (h *Hub) publishRedis(channel string, b []byte) error {
	return h.rdb.Publish(context.Background(), channel, b).Err()
}

// Register registers a client in hub (user -> latest conn, and adds to room map)
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.users[c.uid] = c
	if c.chat != "" {
		if h.rooms[c.chat] == nil {
			h.rooms[c.chat] = make(map[*Client]bool)
		}
		h.rooms[c.chat][c] = true
	}
	_ = h.rdb.Set(context.Background(), "presence:"+c.uid, "online", 60*1).Err()
}

// Unregister removes client from hub and updates presence.
func (h *Hub) Unregister(chatID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// remove latest user mapping if it's this client
	if cur, ok := h.users[c.uid]; ok && cur == c {
		delete(h.users, c.uid)
	}
	if chatID != "" {
		if conns, ok := h.rooms[chatID]; ok {
			delete(conns, c)
			if len(conns) == 0 {
				delete(h.rooms, chatID)
			}
		}
	}
	_ = h.rdb.Del(context.Background(), "presence:"+c.uid).Err()
}

// BroadcastLocal sends to all local connections in the room.
func (h *Hub) BroadcastLocal(chatID string, env *Envelope) {
	h.mu.RLock()
	conns := h.rooms[chatID]
	h.mu.RUnlock()
	if conns == nil {
		return
	}
	b, _ := json.Marshal(env)
	for c := range conns {
		select {
		case c.send <- b:
		default:
			// slow consumer: unregister
			h.Unregister(chatID, c)
			c.close()
		}
	}
}

// Broadcast does local broadcast and publishes to redis for other nodes.
func (h *Hub) Broadcast(chatID string, env *Envelope) error {
	// local fast path
	h.localBroadcast <- env

	// publish for other nodes
	b, _ := json.Marshal(env)
	return h.publishRedis("ws:global", b)
}

// CheckPresence checks presence key in redis.
func (h *Hub) CheckPresence(uid string) bool {
	val, err := h.rdb.Get(context.Background(), "presence:"+uid).Result()
	if err != nil || val != "online" {
		return false
	}
	return true
}

// Shutdown stops background tasks.
func (h *Hub) Shutdown() {
	h.cancel()
	close(h.localBroadcast)
}
