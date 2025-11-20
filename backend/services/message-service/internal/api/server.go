package api

import (
	"github.com/fathima-sithara/message-service/internal/auth"
	"github.com/fathima-sithara/message-service/internal/config"
	"github.com/fathima-sithara/message-service/internal/events"
	"github.com/fathima-sithara/message-service/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func NewServer(cfg *config.Config, svc *service.MessageService, jv *auth.JWTValidator, pub *events.Publisher) *fiber.App {
	app := fiber.New()
	app.Use(logger.New())
	h := NewHandlers(svc, pub)

	api := app.Group("/v1")

	api.Use(func(c *fiber.Ctx) error {
		hdr := c.Get("Authorization")
		if hdr == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing auth"})
		}
		const pref = "Bearer "
		if len(hdr) <= len(pref) || hdr[:len(pref)] != pref {
			return c.Status(401).JSON(fiber.Map{"error": "invalid auth"})
		}
		token := hdr[len(pref):]
		sub, err := jv.Validate(token)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": err.Error()})
		}
		c.Locals("user_id", sub)
		return c.Next()
	})

	api.Post("/messages", h.sendMessage)
	api.Get("/chats/:chat_id/messages", h.listMessages)
	api.Post("/messages/:msg_id/read", h.markRead)
	api.Patch("/messages/:msg_id", h.editMessage)
	api.Delete("/messages/:msg_id", h.deleteMessage)
	api.Post("/media/upload-url", h.mediaUploadURL)
	api.Get("/chats/:chat_id/last-message", h.lastMessage)

	return app
}
