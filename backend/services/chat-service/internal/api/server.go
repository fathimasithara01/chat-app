package api

import (
	"github.com/fathima-sithara/message-service/internal/auth"
	"github.com/fathima-sithara/message-service/internal/config"

	"github.com/fathima-sithara/message-service/internal/service"
	"github.com/fathima-sithara/message-service/internal/ws"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

type Server struct {
	svc  *service.ChatService
	wsrv *ws.Server
	app  *fiber.App
}

func NewServer(cfg *config.Config, svc *service.ChatService, wsrv *ws.Server, jv *auth.JWTValidator) *fiber.App {
	app := fiber.New()
	app.Use(logger.New())

	s := &Server{svc: svc, wsrv: wsrv, app: app}

	api := app.Group("/v1")

	api.Use(func(c *fiber.Ctx) error {
		h := c.Get("Authorization")
		if h == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing auth"})
		}
		const pref = "Bearer "
		if len(h) <= len(pref) || h[:len(pref)] != pref {
			return c.Status(401).JSON(fiber.Map{"error": "invalid auth"})
		}
		token := h[len(pref):]
		sub, err := jv.Validate(token)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": err.Error()})
		}
		c.Locals("user_id", sub)
		return c.Next()
	})

	api.Post("/chats", s.createChat)
	api.Post("/groups", s.createGroup)
	api.Get("/chats", s.listChats)
	api.Get("/chats/:chat_id", s.getChat)
	api.Post("/groups/:chat_id/members", s.addMember)
	api.Delete("/groups/:chat_id/members/:user_id", s.removeMember)
	api.Patch("/chats/:chat_id", s.updateChat)

	api.Get("/ws", websocket.New(wsrv.HandleWS()))

	return app
}

func (s *Server) createChat(c *fiber.Ctx) error {
	var body struct {
		ParticipantID string `json:"participant_id"`
		Name          string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "invalid request"})
	}
	user := c.Locals("user_id").(string)
	chat, err := s.svc.CreateDM(c.Context(), user, body.ParticipantID, body.Name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"status": "success", "data": chat})
}

func (s *Server) createGroup(c *fiber.Ctx) error {
	var body struct {
		Name    string   `json:"name"`
		Members []string `json:"members"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "invalid request"})
	}
	user := c.Locals("user_id").(string)
	chat, err := s.svc.CreateGroup(c.Context(), user, body.Name, body.Members)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"status": "success", "data": chat})
}

	func (s *Server) listChats(c *fiber.Ctx) error {
		user := c.Locals("user_id").(string)
		chats, err := s.svc.ListUserChats(c.Context(), user, 50)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(chats)
	}

func (s *Server) getChat(c *fiber.Ctx) error {
	id := c.Params("chat_id")
	ch, err := s.svc.GetChat(c.Context(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(ch)
}

func (s *Server) addMember(c *fiber.Ctx) error {
	chatID := c.Params("chat_id")
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := c.BodyParser(&body); err != nil || body.UserID == "" {
		return c.Status(400).JSON(fiber.Map{
			"status":  "error",
			"message": "invalid request body",
		})
	}

	// Pass only the UserID string
	if err := s.svc.AddMember(c.Context(), chatID, body.UserID); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "member added",
	})
}

func (s *Server) removeMember(c *fiber.Ctx) error {
	id := c.Params("chat_id")
	userID := c.Params("user_id")
	if err := s.svc.RemoveMember(c.Context(), id, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "success", "message": "member removed"})
}

func (s *Server) updateChat(c *fiber.Ctx) error {
	id := c.Params("chat_id")
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid"})
	}
	if err := s.svc.UpdateName(c.Context(), id, body.Name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "updated"})
}
