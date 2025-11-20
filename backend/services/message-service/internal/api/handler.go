package api

import (
	"context"
	"time"

	"github.com/fathima-sithara/message-service/internal/events"
	"github.com/fathima-sithara/message-service/internal/service"
	"github.com/gofiber/fiber/v2"
)

type Handlers struct {
	svc *service.MessageService
	pub *events.Publisher
}

func NewHandlers(svc *service.MessageService, pub *events.Publisher) *Handlers {
	return &Handlers{svc: svc, pub: pub}
}

func (h *Handlers) sendMessage(c *fiber.Ctx) error {
	var req struct {
		ChatID  string `json:"chat_id"`
		Content string `json:"content"`
		MsgType string `json:"msg_type"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	user := c.Locals("user_id").(string)
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()
	msg, err := h.svc.SendMessage(ctx, req.ChatID, user, req.Content, req.MsgType)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if h.pub != nil {
		h.pub.PublishMessageCreated(req.ChatID, msg)
	}
	return c.Status(201).JSON(fiber.Map{"status": "ok", "data": msg})
}

func (h *Handlers) listMessages(c *fiber.Ctx) error {
	chatID := c.Params("chat_id")
	limit := int64(50)
	msgs, err := h.svc.ListMessages(c.Context(), chatID, limit, time.Now())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok", "data": msgs})
}

func (h *Handlers) markRead(c *fiber.Ctx) error {
	msgID := c.Params("msg_id")
	user := c.Locals("user_id").(string)
	chatID, err := h.svc.MarkRead(c.Context(), msgID, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok", "chat_id": chatID})
}

func (h *Handlers) editMessage(c *fiber.Ctx) error {
	msgID := c.Params("msg_id")
	var body struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid"})
	}
	user := c.Locals("user_id").(string)
	chatID, err := h.svc.EditMessage(c.Context(), msgID, user, body.Content)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok", "chat_id": chatID})
}

func (h *Handlers) deleteMessage(c *fiber.Ctx) error {
	msgID := c.Params("msg_id")
	user := c.Locals("user_id").(string)
	delType := c.Query("type", "user")
	if delType == "all" {
		chatID, err := h.svc.DeleteMessageForAll(c.Context(), msgID, user)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "ok", "chat_id": chatID})
	}
	chatID, err := h.svc.DeleteMessageForUser(c.Context(), msgID, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok", "chat_id": chatID})
}

func (h *Handlers) mediaUploadURL(c *fiber.Ctx) error {
	var body struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid"})
	}
	uploadURL := "https://fake-s3.local/upload/" + body.Filename + "?signature=stub"
	fileURL := "https://cdn.local/" + body.Filename
	return c.JSON(fiber.Map{"upload_url": uploadURL, "file_url": fileURL})
}

func (h *Handlers) lastMessage(c *fiber.Ctx) error {
	chatID := c.Params("chat_id")
	m, err := h.svc.GetLastMessage(c.Context(), chatID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok", "data": m})
}
