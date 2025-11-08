package handlers

import (
	"github.com/fathima-sithara/chat-service/internal/cache"
	"github.com/fathima-sithara/chat-service/internal/repository"
	"github.com/fathima-sithara/chat-service/internal/ws"
	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	repo        *repository.MongoRepository
	hub         *ws.Hub
	redisClient *cache.Client
}

func NewUserHandler(repo *repository.MongoRepository, hub *ws.Hub, redisClient *cache.Client) *UserHandler {
	return &UserHandler{repo: repo, hub: hub, redisClient: redisClient}
}

// GetOnlineUsers fetches currently online users from Redis
func (h *UserHandler) GetOnlineUsers(c *fiber.Ctx) error {
	users, err := h.redisClient.GetOnlineUsers()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"users": users})
}

// SearchUsers searches users by username/email
func (h *UserHandler) SearchUsers(c *fiber.Ctx) error {
	query := c.Query("query")
	if query == "" {
		return c.Status(400).JSON(fiber.Map{"error": "query required"})
	}

	users, err := h.repo.SearchUsers(query)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"users": users})
}
