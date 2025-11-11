package ws

import (
	"encoding/json"

	"github.com/fathima-sithara/chat-service/internal/service"
	"github.com/gofiber/websocket/v2"
)

type Server struct {
	hub    *Hub
	cmdSvc *service.CommandService
	qrySvc *service.QueryService
}

func (s *Server) Hub() *Hub {
	return s.hub
}
func NewServer(cmd *service.CommandService, qry *service.QueryService) *Server {
	return &Server{hub: NewHub(), cmdSvc: cmd, qrySvc: qry}
}

// WSHandler returns websocket handler to be used with websocket.New(...)
func (s *Server) WSHandler() func(*websocket.Conn) {
	return func(ws *websocket.Conn) {
		userIDVal := ws.Locals("user_id")
		if userIDVal == nil {
			_ = ws.Close()
			return
		}
		userID, _ := userIDVal.(string)
		chatID := ws.Query("chat_id")
		if chatID == "" {
			_ = ws.Close()
			return
		}
		conn := &Connection{
			ws:     ws,
			send:   make(chan interface{}, 256),
			chatID: chatID,
			hub:    s.hub,
			userID: userID,
		}
		s.hub.Register(chatID, conn)
		go conn.writePump()
		conn.readPump()
	}
}

// HandleEventMessage is called by kafka consumer to forward events to ws hub
func (s *Server) HandleEventMessage(key string, payload []byte) {
	// naive: assume payload contains {"message": {"chat_id":"..."} , "event":"..."}
	// In production parse JSON properly; here we broadcast to all rooms if chat_id present
	var evt map[string]interface{}
	_ = json.Unmarshal(payload, &evt)
	if evt == nil {
		return
	}
	if msgI, ok := evt["message"]; ok {
		if mm, ok := msgI.(map[string]interface{}); ok {
			if chat, ok := mm["chat_id"].(string); ok && chat != "" {
				s.hub.Broadcast(chat, evt)
				return
			}
		}
	}
	// fallback broadcast to empty room or ignore
}
