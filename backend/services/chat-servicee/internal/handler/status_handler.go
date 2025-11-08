package handlers

import (
	"github.com/fathima-sithara/chat-service/internal/repository"
	"github.com/fathima-sithara/chat-service/internal/ws"
	"github.com/gofiber/fiber/v2"
)

type StatusHandler struct {
	hub  *ws.Hub
	repo *repository.MongoRepository
}

func NewStatusHandler(hub *ws.Hub, repo *repository.MongoRepository) *StatusHandler {
	return &StatusHandler{hub: hub, repo: repo}
}

// UpdateTypingStatus updates typing indicator
func (h *StatusHandler) UpdateTypingStatus(c *fiber.Ctx) error {
	type Req struct {
		ChatID   string `json:"chat_id"`
		IsTyping bool   `json:"is_typing"`
	}

	var req Req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	h.hub.BroadcastJSON(fiber.Map{
		"event": "typing",
		"data":  req,
	})
	return c.JSON(fiber.Map{"message": "status updated"})
}

func (h *StatusHandler) MarkAsRead(c *fiber.Ctx) error {
	type Req struct {
		ChatID     string   `json:"chat_id"`
		MessageIDs []string `json:"message_ids"`
	}

	var req Req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	userID := c.Locals("user_id").(string) // extract from JWT
	if err := h.repo.MarkMessagesRead(req.ChatID, req.MessageIDs, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Optional: broadcast "message:read" to WS clients
	h.hub.BroadcastJSON(fiber.Map{
		"event": "message:read",
		"data": map[string]any{
			"chat_id":     req.ChatID,
			"message_ids": req.MessageIDs,
			"user_id":     userID,
		},
	})

	return c.JSON(fiber.Map{"message": "read status updated"})
}
