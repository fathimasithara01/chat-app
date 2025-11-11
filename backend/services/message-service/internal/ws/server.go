package ws

import (
	"github.com/fathima-sithara/message-service/internal/service"
	"github.com/gofiber/websocket/v2"
)

type Server struct {
	Hub    *Hub
	CmdSvc *service.CommandService
	QrySvc *service.QueryService
}

func NewServer(cmd *service.CommandService, qry *service.QueryService) *Server {
	return &Server{
		Hub:    NewHub(),
		CmdSvc: cmd,
		QrySvc: qry,
	}
}

// HandleWS is the websocket.Handler used with websocket.New()
func (s *Server) HandleWS(wsConn *websocket.Conn) {
	// Locals set by JWT middleware preserved through upgrade by fiber/websocket
	userVal := wsConn.Locals("user_id")
	userID, _ := userVal.(string)
	chatID := wsConn.Query("chat_id")
	if userID == "" || chatID == "" {
		_ = wsConn.Close()
		return
	}

	conn := &Connection{
		ws:     wsConn,
		send:   make(chan interface{}, 256),
		chatID: chatID,
		hub:    s.Hub,
		userID: userID,
	}

	s.Hub.Register(chatID, conn)
	go conn.writePump()
	conn.readPump()
}

func (s *Server) BroadcastMessage(chatID string, msg interface{}) {
	s.Hub.Broadcast(chatID, msg)
}
