package ws

import (
	"github.com/fathima-sithara/websocket-service/internal/auth"
	"github.com/gofiber/websocket/v2"
)

// Server wires hub and jwt validator
type Server struct {
	hub *Hub
	jv  *auth.JWTValidator
}

func NewServer(jv *auth.JWTValidator) *Server {
	return &Server{
		hub: NewHub(),
		jv:  jv,
	}
}

// HandleWS returns a func that matches websocket.New handler signature
func (s *Server) HandleWS() func(*websocket.Conn) {
	return func(conn *websocket.Conn) {
		// query params token & chat_id (room)
		token := conn.Query("token")
		room := conn.Query("chat_id")
		if token == "" || room == "" {
			_ = conn.Close()
			return
		}
		uid, err := s.jv.Validate(token)
		if err != nil {
			_ = conn.Close()
			return
		}

		c := NewConnection(conn, uid, room, s.hub)
		s.hub.Register(room, c)
		go c.writePump()
		c.readPump()
	}
}
