package api

import (
	"github.com/fathima-sithara/websocket-service/internal/auth"
	"github.com/fathima-sithara/websocket-service/internal/config"
	"github.com/fathima-sithara/websocket-service/internal/store"
	"github.com/fathima-sithara/websocket-service/internal/ws"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

type Server struct {
	app   *fiber.App
	wsrv  *ws.Server
	store store.Store
	jv    *auth.JWTValidator
	cfg   *config.Config
}

func NewServer(cfg *config.Config, wsrv *ws.Server, st store.Store, jv *auth.JWTValidator) *fiber.App {
	app := fiber.New()
	app.Use(logger.New())
	s := &Server{app: app, wsrv: wsrv, store: st, jv: jv, cfg: cfg}

	api := app.Group("/v1")

	api.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })

	api.Get("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	api.Get("/ws", websocket.New(wsrv.HandleWS()))

	api.Get("/chats/:chat_id/messages", s.getLatestMessages)

	return app
}

func (s *Server) getLatestMessages(c *fiber.Ctx) error {
	chatID := c.Params("chat_id")
	msgs, _ := s.store.GetLatestMessages(chatID, 50)
	return c.JSON(fiber.Map{"status": "success", "data": msgs})
}
