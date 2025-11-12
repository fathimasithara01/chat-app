package api

import (
	"github.com/fathima-sithara/chat-service/internal/auth"
	"github.com/fathima-sithara/chat-service/internal/config"
	"github.com/fathima-sithara/chat-service/internal/service"
	"github.com/fathima-sithara/chat-service/internal/ws"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

type Server struct {
	cmd *service.CommandService
	qry *service.QueryService
	ws  *ws.Server
}

// NewServer creates a Fiber app with REST + WebSocket routes
func NewServer(cfg *config.Config, cmd *service.CommandService, qry *service.QueryService, wsrv *ws.Server) *fiber.App {
	app := fiber.New()
	app.Use(logger.New())

	s := &Server{cmd: cmd, qry: qry, ws: wsrv} // store the WS server

	api := app.Group("/v1")

	// REST API with JWT middleware
	api.Use(JWTMiddleware(wsrv.JWT())) // use getter to access JWTValidator
	api.Post("/messages", s.sendMessage)
	api.Get("/chats/:chat_id/messages", s.listMessages)
	api.Post("/messages/:msg_id/read", s.markRead)
	api.Patch("/messages/:msg_id", s.editMessage)
	api.Delete("/messages/:msg_id", s.deleteMessage)
	api.Post("/media/upload-url", s.mediaUploadURL)
	api.Get("/chats/:chat_id/last-message", s.lastMessage)

	api.Get("/ws", websocket.New(func(wsConn *websocket.Conn) {
		defer wsConn.Close()

		// token from query param
		token := wsConn.Query("token")
		if token == "" {
			wsConn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "token required"))
			return
		}

		// remove Bearer if present
		const prefix = "Bearer "
		if len(token) > len(prefix) && token[:len(prefix)] == prefix {
			token = token[len(prefix):]
		}

		userID, err := wsrv.JWT().Validate(token)
		if err != nil {
			wsConn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "unauthorized"))
			return
		}

		chatID := wsConn.Query("chat_id")
		if chatID == "" {
			wsConn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "chat_id required"))
			return
		}

		conn := &ws.Connection{
			WS:     wsConn,
			Send:   make(chan interface{}, 256),
			UserID: userID,
			ChatID: chatID,
			Hub:    wsrv.Hub(),
		}

		wsrv.Hub().Register(chatID, conn)
		go conn.WritePump()
		conn.ReadPump()
	}))

	return app
}

// JWTMiddleware protects REST endpoints
func JWTMiddleware(jv *auth.JWTValidator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		h := c.Get("Authorization")
		if h == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing auth"})
		}

		const pref = "Bearer "
		if len(h) <= len(pref) || h[:len(pref)] != pref {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid auth"})
		}

		token := h[len(pref):]
		sub, err := jv.Validate(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		c.Locals("user_id", sub)
		return c.Next()
	}
}
