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
	app *fiber.App
}

func NewServer(cfg *config.Config, cmd *service.CommandService, qry *service.QueryService, wsrv *ws.Server) *fiber.App {
	app := fiber.New()
	// load JWT validator
	jv, err := auth.NewJWTValidator(cfg.JWT.PublicKeyPath)
	if err != nil {
		panic("jwt validator: " + err.Error())
	}

	s := &Server{cmd: cmd, qry: qry, ws: wsrv, app: app}
	app.Use(logger.New())

	api := app.Group("/v1")

	api.Use(JWTAuthMiddleware(jv))

	api.Post("/messages", s.sendMessage)
	api.Get("/chats/:chat_id/messages", s.listMessages)
	api.Post("/messages/:msg_id/read", s.markRead)
	api.Patch("/messages/:msg_id", s.editMessage)
	api.Delete("/messages/:msg_id", s.deleteMessage)
	api.Get("/ws", websocket.New(s.ws.WSHandler()))
	api.Post("/media/upload-url", s.mediaUploadURL)
	api.Get("/chats/:chat_id/last-message", s.lastMessage)

	return app
}

func JWTAuthMiddleware(jv *auth.JWTValidator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		h := c.Get("Authorization")
		if h == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing auth"})
		}
		// Expect "Bearer <token>"
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

// Handlers (sendMessage, listMessages, markRead, editMessage, deleteMessage, mediaUploadURL, lastMessage)
// Implement similarly to message-service: call cmd/qry and broadcast via ws
// For brevity, reuse your message-service handler patterns.

