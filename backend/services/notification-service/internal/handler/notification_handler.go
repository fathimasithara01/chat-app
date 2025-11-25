package handler

import (
	"context"

	"github.com/fathima-sithara/notification-service/internal/model"
	"github.com/fathima-sithara/notification-service/internal/service"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc *service.NotificationService
}

func New(s *service.NotificationService) *Handler {
	return &Handler{s}
}

func (h *Handler) SendNotification(c *fiber.Ctx) error {
	var n model.Notification
	if err := c.BodyParser(&n); err != nil {
		return fiber.ErrBadRequest
	}

	if err := h.svc.Send(context.Background(), &n); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "sent"})
}

func (h *Handler) GetUserNotifications(c *fiber.Ctx) error {
	userID := c.Params("userID")
	notifs, err := h.svc.List(context.Background(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(notifs)
}
