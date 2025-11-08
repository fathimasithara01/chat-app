package handlers

import (
	"net/http"

	"github.com/fathima-sithara/chat-service/internal/cache"
	"github.com/fathima-sithara/chat-service/internal/kafka"
	"github.com/fathima-sithara/chat-service/internal/repository"
	"github.com/fathima-sithara/chat-service/internal/ws"
	"github.com/gofiber/fiber/v2"
)

type ChatHandler struct {
	repo        *repository.MongoRepository
	hub         *ws.Hub
	producer    *kafka.Producer
	redisClient *cache.Client
}

func NewChatHandler(repo *repository.MongoRepository, hub *ws.Hub, producer *kafka.Producer, redisClient *cache.Client) *ChatHandler {
	return &ChatHandler{
		repo:        repo,
		hub:         hub,
		producer:    producer,
		redisClient: redisClient,
	}
}

// CreateChat handles creating one-on-one or group chats
func (h *ChatHandler) CreateChat(c *fiber.Ctx) error {
	type Req struct {
		ParticipantIDs []string `json:"participant_ids"`
		IsGroup        bool     `json:"is_group"`
		GroupName      string   `json:"group_name"`
	}

	var req Req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	chatID, err := h.repo.CreateChat(req.ParticipantIDs, req.IsGroup, req.GroupName)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"chat_id": chatID, "participants": req.ParticipantIDs})
}

// GetUserChats fetches all chats for the logged-in user
func (h *ChatHandler) GetUserChats(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	chats, err := h.repo.GetUserChats(userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"chats": chats})
}

// GetChatDetails fetches chat info
func (h *ChatHandler) GetChatDetails(c *fiber.Ctx) error {
	chatID := c.Params("chat_id")
	if chatID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "chat_id required"})
	}

	chat, err := h.repo.GetChat(chatID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"chat": chat})
}

// DeleteChat soft-deletes a chat
func (h *ChatHandler) DeleteChat(c *fiber.Ctx) error {
	chatID := c.Params("chat_id")
	if chatID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "chat_id required"})
	}

	if err := h.repo.DeleteChat(chatID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "chat deleted"})
}
