package handlers

import (
	"errors"

	"github.com/fathima-sithara/auth-service/internal/repository"
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

// Standardized error response structure
type errorResp struct {
	Error string `json:"error"`
}

// Standardized success message structure
type messageResp struct {
	Message string `json:"message"`
}

// tokenResp defines the structure for token responses.
type tokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Refresh handles refreshing access and refresh tokens.
func (h *Handler) Refresh(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"` // Renamed field for clarity
	}
	if err := c.BodyParser(&req); err != nil {
		h.log.Error("failed to parse refresh token request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
	}

	access, refresh, err := h.svc.RefreshToken(c.Context(), req.RefreshToken) // Use c.Context()
	if err != nil {
		h.log.Error("failed to refresh token", zap.Error(err))
		if errors.Is(err, services.ErrInvalidRefreshToken) {
			return c.Status(fiber.StatusUnauthorized).JSON(errorResp{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to refresh token"})
	}

	return c.JSON(tokenResp{AccessToken: access, RefreshToken: refresh})
}

// requestOTPReq defines the structure for an OTP request.
type requestOTPReq struct {
	Phone string `json:"phone"`
}

// RequestOTP handles sending an OTP to a phone number.
func (h *Handler) RequestOTP(c *fiber.Ctx) error {
	var req requestOTPReq
	if err := c.BodyParser(&req); err != nil {
		h.log.Error("failed to parse request OTP request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
	}

	// Basic validation
	if req.Phone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "phone number is required"})
	}

	if err := h.svc.RequestOTP(c.Context(), req.Phone); err != nil { // Use c.Context()
		if errors.Is(err, services.ErrTooManyRequests) {
			return c.Status(fiber.StatusTooManyRequests).JSON(errorResp{Error: err.Error()})
		}
		h.log.Error("request OTP failed", zap.Error(err), zap.String("phone", req.Phone))
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to send OTP"})
	}
	return c.Status(fiber.StatusOK).JSON(messageResp{Message: "OTP sent successfully"})
}

// verifyOTPReq defines the structure for OTP verification.
type verifyOTPReq struct {
	Phone string `json:"phone"`
	OTP   string `json:"otp"`
}

// VerifyOTP handles verifying an OTP and logging in/registering the user.
func (h *Handler) VerifyOTP(c *fiber.Ctx) error {
	var req verifyOTPReq
	if err := c.BodyParser(&req); err != nil {
		h.log.Error("failed to parse verify OTP request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
	}

	if req.Phone == "" || req.OTP == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "phone and OTP are required"})
	}

	access, refresh, err := h.svc.VerifyOTP(c.Context(), req.Phone, req.OTP) // Use c.Context()
	if err != nil {
		if errors.Is(err, services.ErrInvalidOTP) {
			return c.Status(fiber.StatusUnauthorized).JSON(errorResp{Error: err.Error()})
		}
		h.log.Error("verify OTP failed", zap.Error(err), zap.String("phone", req.Phone))
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to verify OTP"})
	}
	return c.JSON(tokenResp{AccessToken: access, RefreshToken: refresh})
}

// registerEmailReq defines the structure for requesting email OTP.
type registerEmailReq struct {
	Email string `json:"email"`
}

// RegisterEmail handles sending an OTP to an email address for verification.
func (h *Handler) RegisterEmail(c *fiber.Ctx) error {
	var req registerEmailReq
	if err := c.BodyParser(&req); err != nil {
		h.log.Error("failed to parse register email request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
	}

	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "email is required"})
	}

	if err := h.svc.RegisterEmail(c.Context(), req.Email); err != nil { // Use c.Context()
		h.log.Error("register email failed", zap.Error(err), zap.String("email", req.Email))
		if errors.Is(err, services.ErrTooManyRequests) {
			return c.Status(fiber.StatusTooManyRequests).JSON(errorResp{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to send email verification code"})
	}
	return c.JSON(messageResp{Message: "email verification code sent"})
}

// verifyEmailReq defines the structure for email OTP verification.
type verifyEmailReq struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

// VerifyEmail handles verifying an email OTP and logging in/registering the user.
func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
	var req verifyEmailReq
	if err := c.BodyParser(&req); err != nil {
		h.log.Error("failed to parse verify email request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
	}

	if req.Email == "" || req.OTP == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "email and OTP are required"})
	}

	access, refresh, err := h.svc.VerifyEmail(c.Context(), req.Email, req.OTP) // Use c.Context()
	if err != nil {
		if errors.Is(err, services.ErrInvalidOTP) {
			return c.Status(fiber.StatusUnauthorized).JSON(errorResp{Error: err.Error()})
		}
		h.log.Error("verify email failed", zap.Error(err), zap.String("email", req.Email))
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to verify email"})
	}
	return c.JSON(tokenResp{AccessToken: access, RefreshToken: refresh})
}

// registerWithPasswordReq defines the structure for email/password registration.
type registerWithPasswordReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterWithPassword handles user registration with email and password.
func (h *Handler) RegisterWithPassword(c *fiber.Ctx) error {
	var req registerWithPasswordReq
	if err := c.BodyParser(&req); err != nil {
		h.log.Error("failed to parse register with password request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "username, email, and password are required"})
	}

	access, refresh, err := h.svc.RegisterWithPassword(c.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		h.log.Error("register with password failed", zap.Error(err), zap.String("email", req.Email))
		if errors.Is(err, services.ErrUserAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(errorResp{Error: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to register user"})
	}
	return c.Status(fiber.StatusCreated).JSON(tokenResp{AccessToken: access, RefreshToken: refresh})
}

// loginWithPasswordReq defines the structure for email/password login.
type loginWithPasswordReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginWithPassword handles user login with email and password.
func (h *Handler) LoginWithPassword(c *fiber.Ctx) error {
	var req loginWithPasswordReq
	if err := c.BodyParser(&req); err != nil {
		h.log.Error("failed to parse login with password request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorResp{Error: "email and password are required"})
	}

	access, refresh, err := h.svc.LoginWithPassword(c.Context(), req.Email, req.Password)
	if err != nil {
		h.log.Error("login with password failed", zap.Error(err), zap.String("email", req.Email))
		if errors.Is(err, services.ErrInvalidCredentials) || errors.Is(err, repository.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(errorResp{Error: "invalid email or password"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(errorResp{Error: "failed to login"})
	}
	return c.JSON(tokenResp{AccessToken: access, RefreshToken: refresh})
}
