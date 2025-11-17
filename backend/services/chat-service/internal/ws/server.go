package ws

import (
	"github.com/fathima-sithara/message-service/internal/auth"
	"github.com/gofiber/websocket/v2"
)

type Server struct {
	hub *Hub
	svc interface{}
	jv  *auth.JWTValidator
}

func NewServer(svc interface{}, jv *auth.JWTValidator) *Server {
	return &Server{hub: NewHub(), svc: svc, jv: jv}
}

func (s *Server) HandleWS() func(*websocket.Conn) {
	return func(conn *websocket.Conn) {
		token := conn.Query("token")
		if token == "" {
			_ = conn.Close()
			return
		}
		uid, err := s.jv.Validate(token)
		if err != nil {
			_ = conn.Close()
			return
		}
		chatID := conn.Query("chat_id")
		if chatID == "" {
			_ = conn.Close()
			return
		}

		c := &Connection{
			ws:   conn,
			send: make(chan interface{}, 256),
			chat: chatID,
			uid:  uid,
			hub:  s.hub,
		}
		s.hub.Register(chatID, c)
		go c.writePump()
		c.readPump()
	}
}
