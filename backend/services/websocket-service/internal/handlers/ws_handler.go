package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/websocket/v2"
	// "github.com/gofiber/fiber/v2"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/yourorg/chat-app/services/websocket-service/internal/auth"
	"github.com/yourorg/chat-app/services/websocket-service/internal/connection"
	"github.com/yourorg/chat-app/services/websocket-service/internal/hub"
	"github.com/yourorg/chat-app/services/websocket-service/internal/kafka"
	"github.com/yourorg/chat-app/services/websocket-service/internal/redis"
)

// Envelope for WS JSON messages
type Envelope struct {
	Type    string         `json:"type"` // "message","typing","presence","ack"
	Payload map[string]any `json:"payload"`
}

type WSHandler struct {
	hub       *hub.Hub
	prod      *kafka.Producer
	store     *redis.Store
	jwtSecret string
	logger    *zap.SugaredLogger

	// config values
	pingInterval  time.Duration
	writeDeadline time.Duration
	maxMsgSize    int64
}

func NewWSHandler(h *hub.Hub, p *kafka.Producer, s *redis.Store, jwtSecret string, pingInterval, writeDeadline time.Duration, maxMsgSize int64, logger *zap.SugaredLogger) *WSHandler {
	return &WSHandler{
		hub: h, prod: p, store: s, jwtSecret: jwtSecret,
		pingInterval: pingInterval, writeDeadline: writeDeadline, maxMsgSize: maxMsgSize, logger: logger,
	}
}

// Fiber route for upgrade: /ws?token=<jwt>&convId=<convId>
// It must be mounted with fiber/websocket middleware so this handler will run for ws connections
func (w *WSHandler) WS(c *websocket.Conn) {
	// parse token and convId from query
	q := c.Params // fiber/websocket exposes query via Query? use c.Query
	// actually use c.Query
	token := c.Query("token")
	convID := c.Query("convId")
	if token == "" || convID == "" {
		_ = c.WriteMessage(websocket.TextMessage, []byte("missing token or convId"))
		_ = c.Close()
		return
	}
	claims, err := auth.ParseAndValidateToken(w.jwtSecret, token)
	if err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte("invalid token"))
		_ = c.Close()
		return
	}
	userUUID := claims.UserUUID

	// create local client
	client := connection.NewClient(c, userUUID, convID)
	socketID := uuid.New().String()

	// register in-hub and redis presence
	w.hub.AddClient((*hub.Client)(client)) // adapt types in final code
	_ = w.store.AddConnection(context.Background(), userUUID, socketID, convID, 24*time.Hour)

	// start writer goroutine
	go func() {
		ticker := time.NewTicker(w.pingInterval)
		defer ticker.Stop()
		for {
			select {
			case b, ok := <-client.Send:
				if !ok {
					// channel closed
					_ = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
					return
				}
				_ = c.SetWriteDeadline(time.Now().Add(w.writeDeadline))
				if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
					w.logger.Warnf("write msg error: %v", err)
					return
				}
			case <-ticker.C:
				_ = c.SetWriteDeadline(time.Now().Add(w.writeDeadline))
				if err := c.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					w.logger.Warnf("ping error: %v", err)
					return
				}
			}
		}
	}()

	// reader loop
	c.SetReadLimit(w.maxMsgSize)
	for {
		mt, msg, err := c.ReadMessage()
		if err != nil {
			// connection closed or error
			break
		}
		if mt != websocket.TextMessage {
			continue
		}
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			continue
		}
		switch env.Type {
		case "message":
			// expected payload: { "content": "...", "conversation_id": "..." }
			content, _ := env.Payload["content"].(string)
			convIDPayload, _ := env.Payload["conversation_id"].(string)
			if convIDPayload == "" {
				convIDPayload = convID
			}
			// build event to publish
			event := map[string]any{
				"type": "message",
				"payload": map[string]any{
					"conversation_id": convIDPayload,
					"sender_id":       userUUID,
					"content":         content,
					"sent_at":         time.Now().UTC().Format(time.RFC3339),
				},
			}
			// 1) publish to kafka for persistence + notifications
			_ = w.prod.PublishMessageSent(context.Background(), event)
			// 2) broadcast to local clients in conversation
			out, _ := json.Marshal(map[string]any{"type": "message", "payload": event["payload"]})
			w.hub.BroadcastToConversation(context.Background(), convIDPayload, out)

		case "typing":
			// typing indicator - broadcast locally and to other instances via pubsub
			out, _ := json.Marshal(map[string]any{"type": "typing", "payload": env.Payload})
			w.hub.BroadcastToConversation(context.Background(), convID, out)

		case "presence":
			// client sets custom presence - we may update Redis
			status, _ := env.Payload["status"].(string)
			_ = w.store.Publish(context.Background(), "presence:"+userUUID, []byte(fmt.Sprintf(`{"status":"%s","at":%d}`, status, time.Now().Unix())))
		default:
			// ignore unknown
		}
	}

	// cleanup on disconnect
	w.hub.RemoveClient((*hub.Client)(client))
	_ = w.store.RemoveConnection(context.Background(), userUUID, socketID)
	client.Close()
}
