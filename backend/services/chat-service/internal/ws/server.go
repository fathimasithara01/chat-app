package ws

import (
	"encoding/json"
	"log"

	"github.com/fathima-sithara/chat-service/internal/auth"
	"github.com/fathima-sithara/chat-service/internal/service"
	"github.com/gofiber/websocket/v2"
)

type Server struct {
	HubSvc *Hub
	CmdSvc *service.CommandService
	QrySvc *service.QueryService
	jv     *auth.JWTValidator
}

func NewServer(cmd *service.CommandService, qry *service.QueryService, jv *auth.JWTValidator) *Server {
	return &Server{
		HubSvc: NewHub(),
		CmdSvc: cmd,
		QrySvc: qry,
		jv:     jv,
	}
}

func (s *Server) Hub() *Hub {
	return s.HubSvc
}

func (s *Server) JWT() *auth.JWTValidator {
	return s.jv
}

// Forward messages from Kafka or other sources
func (s *Server) HandleEventMessage(key string, payload []byte) {
	var evt map[string]interface{}
	if err := json.Unmarshal(payload, &evt); err != nil {
		log.Println("invalid event:", err)
		return
	}
	if msgI, ok := evt["message"]; ok {
		if mm, ok := msgI.(map[string]interface{}); ok {
			if chat, ok := mm["chat_id"].(string); ok && chat != "" {
				s.HubSvc.Broadcast(chat, evt)
			}
		}
	}
}

// Fiber WS handler
func (s *Server) WSHandler() func(*websocket.Conn) {
	return func(ws *websocket.Conn) {
		defer ws.Close()

		token := ws.Query("token")
		const prefix = "Bearer "
		if len(token) > len(prefix) && token[:len(prefix)] == prefix {
			token = token[len(prefix):]
		}

		userID, err := s.jv.Validate(token)
		if err != nil {
			log.Println("ws auth failed:", err)
			return
		}

		chatID := ws.Query("chat_id")
		if chatID == "" {
			log.Println("chat_id required")
			return
		}

		conn := &Connection{
			WS:     ws,
			Send:   make(chan interface{}, 256),
			ChatID: chatID,
			Hub:    s.HubSvc,
			UserID: userID,
		}

		s.HubSvc.Register(chatID, conn)
		go conn.WritePump()
		conn.ReadPump()
	}
}
