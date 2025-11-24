package ws

import (
	"net/http"

	"github.com/fathima-sithara/websocket/internal/auth"
	"github.com/fathima-sithara/websocket/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func NewServer(hub *Hub, jv *auth.JWTValidator, cfg *config.Config) *fiber.App {
	app := fiber.New()

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Get("/ws", websocket.New(func(conn *websocket.Conn) {
		token := conn.Query("token")

		if token == "" {
			token = conn.Subprotocol()
		}

		if token == "" {
			conn.Close()
			return
		}

		sub, err := jv.Validate(token)
		if err != nil {
			conn.Close()
			return
		}

		chatID := conn.Query("chat_id")
		if chatID == "" {
			conn.Close()
			return
		}

		client := NewClient(conn, sub, chatID, hub, cfg.RateLimitPerSec)
		hub.Register(client)

		go client.writePump()
		client.readPump()
	}))

	app.Get("/presence/:user_id", func(c *fiber.Ctx) error {
		uid := c.Params("user_id")
		if uid == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "user_id required"})
		}
		ok := hub.CheckPresence(uid)
		return c.JSON(fiber.Map{"user_id": uid, "online": ok})
	})

	return app
}
