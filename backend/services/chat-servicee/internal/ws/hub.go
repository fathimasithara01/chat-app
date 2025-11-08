package ws

import (
	"crypto/rsa"
	"encoding/json"
	"log"
	"sync"

	"github.com/fathima-sithara/chat-service/internal/cache"
	"github.com/fathima-sithara/chat-service/internal/kafka"
	"github.com/fathima-sithara/chat-service/internal/repository"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v4"
)

type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	mu         sync.Mutex
	Repo       *repository.MongoRepository
	KafkaProd  *kafka.Producer
	Redis      *cache.Client
	PublicKey  *rsa.PublicKey
}

func NewHub(repo *repository.MongoRepository, producer *kafka.Producer, redis *cache.Client, pubKey *rsa.PublicKey) *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Repo:       repo,
		KafkaProd:  producer,
		Redis:      redis,
		PublicKey:  pubKey,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()
		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
		case msg := <-h.Broadcast:
			h.mu.Lock()
			for client := range h.Clients {
				select {
				case client.Send <- msg:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) BroadcastJSON(payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Println("marshal error:", err)
		return
	}
	select {
	case h.Broadcast <- data:
	default:
		log.Println("broadcast channel full, dropping message")
	}
}

func (h *Hub) HandleWebsocket(conn *websocket.Conn) {
	tokenStr := conn.Query("token")
	if tokenStr == "" {
		conn.Close()
		return
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return h.PublicKey, nil
	})
	if err != nil || !token.Valid {
		conn.Close()
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		conn.Close()
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		conn.Close()
		return
	}

	client := &Client{
		ID:   userID,
		Hub:  h,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	h.Register <- client
	go client.WritePump()
	client.ReadPump()
}

func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.Clients {
		close(client.Send)
		client.Conn.Close()
	}
}
