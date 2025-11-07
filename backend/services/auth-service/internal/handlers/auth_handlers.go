package handlers

import (
	"errors"

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

type errorResp struct {
	Error string `json:"error"`
}

type messageResp struct {
	Message string `json:"message"`
}

type tokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type registerReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register(c *fiber.Ctx) error {
	var req registerReq
	if err := c.BodyParser(&req); err != nil {
		h.log.Error("failed to parse register request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "username, email, and password are required"})
	}

	err := h.svc.InitiateEmailRegistration(c.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		h.log.Error("failed to initiate email registration", zap.Error(err),
			zap.String("username", req.Username), zap.String("email", req.Email))
		if errors.Is(err, services.ErrUserAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(errorResp{Error: err.Error()})
		}
		if errors.Is(err, services.ErrTooManyRequests) {
			return c.Status(fiber.StatusTooManyRequests).JSON(errorResp{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to send verification code"})
	}

	return c.Status(fiber.StatusOK).JSON(messageResp{Message: "verification code sent"})
}

type verifyEmailReq struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
	var req verifyEmailReq
	if err := c.BodyParser(&req); err != nil {
		h.log.Error("failed to parse verify email request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
	}

	if req.Email == "" || req.OTP == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "email and OTP are required"})
	}

	access, refresh, err := h.svc.CompleteEmailVerification(c.Context(), req.Email, req.OTP)
	if err != nil {
		h.log.Error("failed to complete email verification", zap.Error(err), zap.String("email", req.Email))
		if errors.Is(err, services.ErrInvalidOTP) {
			return c.Status(fiber.StatusUnauthorized).JSON(errorResp{Error: err.Error()})
		}
		if errors.Is(err, services.ErrUserNotFound) { // User might not have initiated registration
			return c.Status(fiber.StatusNotFound).JSON(errorResp{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to verify email"})
	}
	return c.JSON(tokenResp{AccessToken: access, RefreshToken: refresh})
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req loginReq
	if err := c.BodyParser(&req); err != nil {
		h.log.Error("failed to parse login request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "email and password are required"})
	}

	access, refresh, err := h.svc.LoginWithPassword(c.Context(), req.Email, req.Password)
	if err != nil {
		h.log.Error("login failed", zap.Error(err), zap.String("email", req.Email))
		if errors.Is(err, services.ErrInvalidCredentials) || errors.Is(err, services.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(errorResp{Error: "invalid email or password"})
		}
		if errors.Is(err, services.ErrUserNotVerified) {
			return c.Status(fiber.StatusForbidden).JSON(errorResp{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to login"})
	}
	return c.JSON(tokenResp{AccessToken: access, RefreshToken: refresh})
}

type requestOTPReq struct {
	Phone string `json:"phone"`
	Email string `json:"email"`
}

func (h *Handler) RequestOTP(c *fiber.Ctx) error {
	var req requestOTPReq
	if err := c.BodyParser(&req); err != nil {
		h.log.Error("failed to parse request OTP request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
	}

	// if req.Phone == "" {
	// 	return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "phone number is required"})
	// }

	if err := h.svc.RequestOTP(c.Context(), req.Phone, req.Email); err != nil {
		if errors.Is(err, services.ErrTooManyRequests) {
			return c.Status(fiber.StatusTooManyRequests).JSON(errorResp{Error: err.Error()})
		}
		h.log.Error("request phone OTP failed", zap.Error(err), zap.String("phone", req.Phone))
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to send OTP"})
	}
	return c.Status(fiber.StatusOK).JSON(messageResp{Message: "OTP sent successfully"})
}

type verifyOTPReq struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	OTP   string `json:"otp"`
}

func (h *Handler) VerifyOTP(c *fiber.Ctx) error {
	var req verifyOTPReq
	if err := c.BodyParser(&req); err != nil {
		h.log.Error("failed to parse verify OTP request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
	}

	if req.Phone == "" || req.OTP == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "phone and OTP are required"})
	}

	access, refresh, err := h.svc.VerifyOTP(c.Context(), req.Phone, req.Email, req.OTP)
	if err != nil {
		if errors.Is(err, services.ErrInvalidOTP) {
			return c.Status(fiber.StatusUnauthorized).JSON(errorResp{Error: err.Error()})
		}
		h.log.Error("verify phone OTP failed", zap.Error(err), zap.String("phone", req.Phone))
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to verify OTP"})
	}
	return c.JSON(tokenResp{AccessToken: access, RefreshToken: refresh})
}

func (h *Handler) Refresh(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&req); err != nil {
		h.log.Error("failed to parse refresh token request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
	}

	access, refresh, err := h.svc.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		h.log.Error("failed to refresh token", zap.Error(err))
		if errors.Is(err, services.ErrInvalidRefreshToken) {
			return c.Status(fiber.StatusUnauthorized).JSON(errorResp{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to refresh token"})
	}

	return c.JSON(tokenResp{AccessToken: access, RefreshToken: refresh})
}

type logoutReq struct {
	AccessToken string `json:"access_token"`
}

func (h *Handler) Logout(c *fiber.Ctx) error {

	userID := c.Locals("userID")
	if userID == nil {
		var req logoutReq
		if err := c.BodyParser(&req); err != nil {
			h.log.Error("failed to parse logout request body", zap.Error(err))
			return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
		}

		parsedUserID, err := h.svc.GetUserIDFromAccessToken(req.AccessToken)
		if err != nil {
			h.log.Warn("Failed to parse access token for logout", zap.Error(err))
			return c.Status(fiber.StatusUnauthorized).JSON(errorResp{Error: "invalid access token"})
		}
		userID = parsedUserID
	}

	uidStr, ok := userID.(string)
	if !ok {
		h.log.Error("userID not found in context or invalid type for logout", zap.Any("userID", userID))
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "authentication context missing"})
	}

	err := h.svc.Logout(c.Context(), uidStr)
	if err != nil {
		h.log.Error("failed to logout user", zap.Error(err), zap.String("userID", uidStr))
		if errors.Is(err, services.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(errorResp{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to logout"})
	}

	return c.Status(fiber.StatusOK).JSON(messageResp{Message: "logged out"})
}

type changePasswordReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) ChangePassword(c *fiber.Ctx) error {
	var req changePasswordReq
	if err := c.BodyParser(&req); err != nil {
		h.log.Error("failed to parse change password request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "old_password and new_password are required"})
	}

	userID := c.Locals("userID")
	if userID == nil {
		h.log.Warn("userID not found in context for change password")
		return c.Status(fiber.StatusUnauthorized).JSON(errorResp{Error: "unauthorized"})
	}

	uidStr, ok := userID.(string)
	if !ok {
		h.log.Error("userID in context is not a string for change password", zap.Any("userID", userID))
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "authentication context error"})
	}

	err := h.svc.ChangePassword(c.Context(), uidStr, req.OldPassword, req.NewPassword)
	if err != nil {
		h.log.Error("failed to change password", zap.Error(err), zap.String("userID", uidStr))
		if errors.Is(err, services.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(errorResp{Error: err.Error()})
		}
		if errors.Is(err, services.ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(errorResp{Error: "incorrect old password"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to change password"})
	}

	return c.Status(fiber.StatusOK).JSON(messageResp{Message: "password changed"})
}
