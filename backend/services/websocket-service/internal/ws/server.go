package ws

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type Server struct {
	hub *Hub
}

func NewServer(h *Hub) *Server {
	return &Server{hub: h}
}

func (s *Server) HandleWS() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		userID := c.Query("user_id")
		if userID == "" {
			c.Close()
			return
		}
		client := NewClient(userID, c)
		s.hub.AddClient(userID, client)
		defer s.hub.RemoveClient(userID)

		go client.WritePump()

		for {
			var msg map[string]any
			if err := c.ReadJSON(&msg); err != nil {
				log.Println("ws read error:", err)
				break
			}
			event := msg["event"].(string)
			data := msg["data"]
			s.HandleEvent(client, event, data)
		}
	})
}

func (s *Server) HandleEvent(client *Client, event string, data any) {
	switch event {
	case "join_chat":
		chatID := data.(map[string]any)["chat_id"].(string)
		s.hub.JoinRoom(chatID, client.UserID)
	case "leave_chat":
		chatID := data.(map[string]any)["chat_id"].(string)
		s.hub.LeaveRoom(chatID, client.UserID)
	case "send_message":
		d := data.(map[string]any)
		chatID := d["chat_id"].(string)
		content := d["content"].(string)
		// Broadcast message to all members in chat
		s.hub.Broadcast(chatID, map[string]any{
			"event": "new_message",
			"data": map[string]any{
				"chat_id":   chatID,
				"sender_id": client.UserID,
				"content":   content,
			},
		})
	case "typing_start", "typing_stop":
		d := data.(map[string]any)
		chatID := d["chat_id"].(string)
		s.hub.Broadcast(chatID, map[string]any{
			"event": event,
			"data": map[string]any{
				"chat_id": chatID,
				"user_id": client.UserID,
			},
		})
	}
}
