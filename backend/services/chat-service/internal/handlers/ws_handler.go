package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/yourorg/chat-app/services/chat-service/internal/service"
	"github.com/yourorg/chat-app/services/chat-service/internal/utils"
	"github.com/yourorg/chat-app/services/chat-service/internal/websocket"
)

type WSHandler struct {
	hub       *websocket.Hub
	chatSvc   *service.ChatService
	jwtSecret string
}

func NewWSHandler(h *websocket.Hub, cs *service.ChatService, jwtSecret string) *WSHandler {
	return &WSHandler{hub: h, chatSvc: cs, jwtSecret: jwtSecret}
}

// Expected ws URL: /ws?token=<jwt>&convId=<convid>
func (wsh *WSHandler) Handle(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

func (wsh *WSHandler) WS(c *websocket.Conn) {
	// extract token and convId from query params
	q := c.Query
	token := q("token")
	convID := q("convId")
	if token == "" || convID == "" {
		_ = c.Close()
		return
	}
	claims, err := utils.ParseAndValidateToken(wsh.jwtSecret, token)
	if err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte("invalid token"))
		_ = c.Close()
		return
	}
	userUUID := claims.UserUUID

	cli := websocket.NewClient(c, userUUID, convID)
	wsh.hub.AddClient(cli)
	defer func() {
		wsh.hub.RemoveClient(cli)
		_ = c.Close()
	}()

	// writer goroutine
	go func() {
		for {
			select {
			case b := <-cli.Send:
				_ = c.WriteMessage(websocket.TextMessage, b)
			}
		}
	}()

	// read loop
	for {
		mt, msgBytes, err := c.ReadMessage()
		if err != nil {
			return
		}
		if mt != websocket.TextMessage {
			continue
		}
		// parse envelope
		var envelope map[string]any
		if err := json.Unmarshal(msgBytes, &envelope); err != nil {
			continue
		}
		// handle "message" type
		if t, ok := envelope["type"].(string); ok && t == "message" {
			payload := envelope["payload"].(map[string]any)
			content, _ := payload["content"].(string)
			// build message model
			msg := &models.Message{
				ConversationID: convID,
				SenderID:       userUUID,
				Content:        content,
				Type:           "text",
				CreatedAt:      time.Now().UTC(),
			}
			ins, err := wsh.chatSvc.SendMessage(context.Background(), msg)
			if err != nil {
				// optional: send error back
				continue
			}
			// broadcast to conversation
			out, _ := json.Marshal(map[string]any{"type": "message", "payload": ins})
			wsh.hub.BroadcastToConversation(context.Background(), convID, out)
		}
	}
}
