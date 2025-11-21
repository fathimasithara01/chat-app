package ws

import (
	"github.com/fathima-sithara/websocket/internal/auth"
	"github.com/fathima-sithara/websocket/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// NewServerApp wires the hub and jwt validator into Fiber app.
func NewServerApp(hub *Hub, jv *auth.JWTValidator, cfg *config.Config) *fiber.App {
	app := fiber.New()

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Get("/ws", websocket.New(func(conn *websocket.Conn) {
		// Basic auth extraction
		token := conn.Query("token")
		if token == "" {
			_ = conn.Close()
			return
		}
		sub, err := jv.Validate(token)
		if err != nil {
			_ = conn.Close()
			return
		}
		chatID := conn.Query("chat_id")
		if chatID == "" {
			_ = conn.Close()
			return
		}

		client := NewClient(conn, sub, chatID, hub, cfg.RateLimitPerSec)
		hub.Register(client)

		// writer goroutine & blocking read
		go client.writePump()
		client.readPump()
	}))

	// presence endpoint
	app.Get("/presence/:user_id", func(c *fiber.Ctx) error {
		uid := c.Params("user_id")
		ok := hub.CheckPresence(uid)
		return c.JSON(fiber.Map{"user_id": uid, "online": ok})
	})

	return app
}
