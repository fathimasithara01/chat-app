package handlers

import (
	"context"
	"time"

	"github.com/fathima-sithara/auth-service/internal/services"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Handler struct {
	svc *services.AuthService
	log *zap.Logger
}

func NewHandler(svc *services.AuthService, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, log: logger}
}

type requestOTPReq struct {
	Phone string `json:"phone"`
}

type verifyOTPReq struct {
	Phone string `json:"phone"`
	OTP   string `json:"otp"`
}

type tokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(c *fiber.Ctx) error {
	var req struct {
		Refresh string `json:"refresh"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	access, refresh, err := h.svc.RefreshToken(c.Context(), req.Refresh)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"access_token":  access,
		"refresh_token": refresh,
	})
}

func (h *Handler) RequestOTP(c *fiber.Ctx) error {
	var req requestOTPReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.svc.RequestOTP(ctx, req.Phone); err != nil {
		if err == services.ErrTooManyRequests {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": err.Error()})
		}
		h.log.Error("request otp failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "otp sent"})
}

func (h *Handler) VerifyOTP(c *fiber.Ctx) error {
	var req verifyOTPReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	access, refresh, err := h.svc.VerifyOTP(ctx, req.Phone, req.OTP)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid otp"})
	}
	return c.JSON(tokenResp{AccessToken: access, RefreshToken: refresh})
}

type registerEmailReq struct {
	Email string `json:"email"`
}

func (h *Handler) RegisterEmail(c *fiber.Ctx) error {
	var req registerEmailReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.svc.RegisterEmail(ctx, req.Email); err != nil {
		h.log.Error("register email failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed"})
	}
	return c.JSON(fiber.Map{"message": "email otp sent"})
}

type verifyEmailReq struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
	var req verifyEmailReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	access, refresh, err := h.svc.VerifyEmail(ctx, req.Email, req.OTP)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid otp"})
	}
	return c.JSON(tokenResp{AccessToken: access, RefreshToken: refresh})
}
