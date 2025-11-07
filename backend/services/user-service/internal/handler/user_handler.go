package handlers

import (
	"errors"

	"github.com/fathima-sithara/user-service/internal/repository"
	"github.com/fathima-sithara/user-service/internal/service"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Handler struct {
	svc *service.UserService
	log *zap.Logger
}

func NewHandler(svc *service.UserService, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

type updateProfileReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

type changePasswordReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// GetProfile returns current user's profile (userID from middleware)
func (h *Handler) GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthenticated"})
	}
	uid := userID.(string)
	u, err := h.svc.GetProfile(c.Context(), uid)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
		}
		h.log.Error("get profile failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(u)
}

// UpdateProfile updates username/email/phone
func (h *Handler) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthenticated"})
	}
	uid := userID.(string)
	var req updateProfileReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	u, err := h.svc.UpdateProfile(c.Context(), uid, req.Username, req.Email, req.Phone)
	if err != nil {
		h.log.Error("update profile failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update profile"})
	}
	return c.JSON(u)
}

// ChangePassword proxies to auth-service
func (h *Handler) ChangePassword(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
	}
	var req changePasswordReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "old and new password required"})
	}
	if err := h.svc.ChangePassword(c.Context(), authHeader, req.OldPassword, req.NewPassword); err != nil {
		h.log.Error("change password error", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to change password"})
	}
	return c.JSON(fiber.Map{"message": "password changed"})
}

// Admin: Get user by id
func (h *Handler) GetUserByID(c *fiber.Ctx) error {
	id := c.Params("id")
	u, err := h.svc.GetByIDAdmin(c.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
		}
		h.log.Error("get user by id failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(u)
}

// Admin: soft delete
func (h *Handler) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.svc.DeleteUser(c.Context(), id); err != nil {
		h.log.Error("delete user failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete user"})
	}
	return c.JSON(fiber.Map{"message": "user deleted"})
}
