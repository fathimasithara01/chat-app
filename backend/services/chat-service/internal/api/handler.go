package api

import (
	"github.com/fathima-sithara/chat-service/internal/service"
	"github.com/gofiber/fiber/v2"
)

type sendMessageReq struct {
	ChatID  string `json:"chat_id" validate:"required"`
	Content string `json:"content" validate:"required"`
}

func (s *Server) sendMessage(c *fiber.Ctx) error {
	var req sendMessageReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	userID := c.Locals("user_id").(string)

	msg, err := s.cmd.SendMessage(c.Context(), service.SendMessageCommand{
		ChatID:  req.ChatID,
		UserID:  userID,
		Content: req.Content,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	s.ws.Hub().Broadcast(req.ChatID, fiber.Map{
		"event":   "message_created",
		"message": msg,
	})

	return c.JSON(msg)
}

func (s *Server) listMessages(c *fiber.Ctx) error {
	chatID := c.Params("chat_id")
	if chatID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing chat_id")
	}

	msgs, err := s.qry.ListMessages(c.Context(), chatID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"messages": msgs})
}

func (s *Server) markRead(c *fiber.Ctx) error {
	msgID := c.Params("msg_id")
	userID := c.Locals("user_id").(string)

	if err := s.cmd.MarkAsRead(c.Context(), msgID, userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

type editReq struct {
	Content string `json:"content" validate:"required"`
}

func (s *Server) editMessage(c *fiber.Ctx) error {
	msgID := c.Params("msg_id")
	userID := c.Locals("user_id").(string)

	var req editReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	msg, err := s.cmd.EditMessage(c.Context(), msgID, userID, req.Content)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	s.ws.Hub().Broadcast(msg.ChatID, fiber.Map{
		"event":   "message_updated",
		"message": msg,
	})

	return c.JSON(msg)
}

func (s *Server) deleteMessage(c *fiber.Ctx) error {
	msgID := c.Params("msg_id")
	userID := c.Locals("user_id").(string)

	chatID, err := s.cmd.DeleteMessage(c.Context(), msgID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	s.ws.Hub().Broadcast(chatID, fiber.Map{
		"event":     "message_deleted",
		"messageId": msgID,
	})

	return c.JSON(fiber.Map{"status": "deleted"})
}

func (s *Server) mediaUploadURL(c *fiber.Ctx) error {
	url, err := s.cmd.GeneratePresignedUploadURL(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"upload_url": url})
}

func (s *Server) lastMessage(c *fiber.Ctx) error {
	chatID := c.Params("chat_id")
	msg, err := s.qry.LastMessage(c.Context(), chatID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(msg)
}
