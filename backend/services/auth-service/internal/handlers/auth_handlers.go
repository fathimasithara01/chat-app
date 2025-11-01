package handlers

import (
	"github.com/fathima-sithara/chat-app/internal/services"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc *services.AuthService
}

func NewHandler(svc *services.AuthService) *Handler {
	return &Handler{svc: svc}
}

type requestOTPReq struct {
	Phone string `json:"phone"`
}

func (h *Handler) RequestOTP(c *fiber.Ctx) error {
	var req requestOTPReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	if req.Phone == "" {
		return fiber.NewError(fiber.StatusBadRequest, "phone required")
	}
	if err := h.svc.RegisterOrRequestOTP(c.Context(), req.Phone); err != nil {
		return fiber.NewError(fiber.StatusTooManyRequests, err.Error())
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "otp_sent"})
}

type verifyOTPReq struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

func (h *Handler) VerifyOTP(c *fiber.Ctx) error {
	var req verifyOTPReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	access, refresh, user, err := h.svc.VerifyOTPAndLogin(c.Context(), req.Phone, req.Code)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}
	return c.JSON(fiber.Map{
		"access_token":  access,
		"refresh_token": refresh,
		"user":          user,
	})
}

type registerEmailReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) RegisterEmail(c *fiber.Ctx) error {
	var req registerEmailReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	user, err := h.svc.RegisterWithEmail(c.Context(), req.Email, req.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user": user})
}

type verifyEmailReq struct {
	Token string `json:"token"`
}

func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
	var req verifyEmailReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	user, err := h.svc.VerifyEmailToken(c.Context(), req.Token)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"user": user})
}

type refreshReq struct {
	UserUUID     string `json:"user_uuid"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(c *fiber.Ctx) error {
	var req refreshReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	access, newRefresh, err := h.svc.RefreshAccessToken(c.Context(), req.UserUUID, req.RefreshToken)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}
	return c.JSON(fiber.Map{
		"access_token":  access,
		"refresh_token": newRefresh,
	})
}
