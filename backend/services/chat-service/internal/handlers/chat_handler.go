package handlers

import (
	"context"
	"time"

	"github.com/fathima-sithara/chat-service/internal/models"
	"github.com/fathima-sithara/chat-service/internal/repository"
	"github.com/fathima-sithara/chat-service/internal/ws"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type ChatHandler struct {
	repo *repository.MongoRepository
	hub  *ws.Hub
}

func NewChatHandler(repo *repository.MongoRepository, hub *ws.Hub) *ChatHandler {
	return &ChatHandler{repo: repo, hub: hub}
}

func (h *ChatHandler) CreateConversation(c *fiber.Ctx) error {
	var body struct {
		Members []string `json:"members"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.ErrBadRequest
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conv, err := h.repo.CreateConversation(ctx, body.Members)
	if err != nil {
		log.Error().Err(err).Msg("create conv")
		return fiber.ErrInternalServerError
	}
	return c.JSON(conv)
}

func (h *ChatHandler) SendMessage(c *fiber.Ctx) error {
	var body models.Message
	if err := c.BodyParser(&body); err != nil {
		return fiber.ErrBadRequest
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.repo.SaveMessage(ctx, &body); err != nil {
		log.Error().Err(err).Msg("save message")
		return fiber.ErrInternalServerError
	}
	// publish to kafka
	// hub will pick up via consumer and forward to connected users
	return c.JSON(fiber.Map{"status": "sent"})
}

func (h *ChatHandler) GetMessages(c *fiber.Ctx) error {
	convID := c.Params("id")
	limit := int64(50)
	skip := int64(0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	msgs, err := h.repo.GetMessages(ctx, convID, limit, skip)
	if err != nil {
		log.Error().Err(err).Msg("get messages")
		return fiber.ErrInternalServerError
	}
	return c.JSON(msgs)
}
