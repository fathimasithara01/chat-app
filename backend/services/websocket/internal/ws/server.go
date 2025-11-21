package ws

import (
	"github.com/fathima-sithara/websocket/internal/auth"
	"github.com/fathima-sithara/websocket/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	"net/http"
)

type Server struct {
	hub *Hub
	jv  *auth.JWTValidator
	cfg *config.Config
	app *fiber.App
}

func NewServer(hub *Hub, jv *auth.JWTValidator, cfg *config.Config) *fiber.App {
	app := fiber.New()
	s := &Server{hub: hub, jv: jv, cfg: cfg, app: app}

	// health
	app.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })

	// websocket endpoint (upgrade)
	app.Get("/ws", websocket.New(s.handleWS))

	// presence endpoint
	app.Get("/presence/:user_id", func(c *fiber.Ctx) error {
		uid := c.Params("user_id")
		if uid == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "user_id required"})
		}
		ok := s.hub.CheckPresence(uid)
		return c.JSON(fiber.Map{"user_id": uid, "online": ok})
	})

	return app
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }

func (s *Server) handleWS(conn *websocket.Conn) {
	// simple auth: token query param or Authorization header
	token := conn.Query("token")
	if token == "" {
		ah := conn.UnderlyingConn().RemoteAddr()
		_ = ah // keep for logs if needed
		_ = conn.Close()
		return
	}
	sub, err := s.jv.Validate(token)
	if err != nil {
		_ = conn.Close()
		return
	}
	chatID := conn.Query("chat_id")
	if chatID == "" {
		_ = conn.Close()
		return
	}
	// create client
	hub := s.hub
	client := NewClient(conn, sub, chatID, hub, s.cfg.RateLimitPerSec)
	hub.Register(client)
	// start pumps
	go client.writePump()
	client.readPump()
}
