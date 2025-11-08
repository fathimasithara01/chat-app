package handlers

import (
	"net/http"
	"time"

	"github.com/fathima-sithara/chat-service/internal/kafka"
	"github.com/fathima-sithara/chat-service/internal/models"
	"github.com/fathima-sithara/chat-service/internal/repository"
	"github.com/fathima-sithara/chat-service/internal/ws"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MessageHandler struct {
	repo     *repository.MongoRepository
	hub      *ws.Hub
	producer *kafka.Producer
}

func NewMessageHandler(repo *repository.MongoRepository, hub *ws.Hub, producer *kafka.Producer) *MessageHandler {
	return &MessageHandler{repo: repo, hub: hub, producer: producer}
}

// SendMessage sends a message to a chat
func (h *MessageHandler) SendMessage(c *fiber.Ctx) error {
	type Req struct {
		ChatID  string `json:"chat_id"`
		Content string `json:"content"`
		Type    string `json:"type"`
	}

	var req Req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	userID := c.Locals("user_id").(string)
	msgID := primitive.NewObjectID().Hex()
	timestamp := time.Now()

	message := models.Message{
		ID:        msgID,
		ChatID:    req.ChatID,
		SenderID:  userID,
		Content:   req.Content,
		Type:      req.Type,
		Timestamp: timestamp,
	}

	if err := h.repo.SendMessage(message); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Broadcast via WebSocket
	h.hub.BroadcastJSON(fiber.Map{"event": "message:receive", "data": message})

	// Produce Kafka event
	h.producer.Publish("chat_messages_out", message)

	return c.JSON(fiber.Map{"message_id": msgID, "timestamp": timestamp})
}

// GetMessages fetches messages of a chat
func (h *MessageHandler) GetMessages(c *fiber.Ctx) error {
	chatID := c.Params("chat_id")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)

	messages, err := h.repo.GetMessages(chatID, page, limit)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"messages": messages})
}

// EditMessage edits a message
func (h *MessageHandler) EditMessage(c *fiber.Ctx) error {
	messageID := c.Params("message_id")
	type Req struct {
		Content string `json:"content"`
	}

	var req Req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if err := h.repo.EditMessage(messageID, req.Content); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "updated"})
}

// DeleteMessage deletes a message
func (h *MessageHandler) DeleteMessage(c *fiber.Ctx) error {
	messageID := c.Params("message_id")
	if err := h.repo.DeleteMessage(messageID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "deleted"})
}
