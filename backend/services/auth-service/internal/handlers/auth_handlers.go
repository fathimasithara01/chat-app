package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/fathima-sithara/auth-service/internal/models"
	"github.com/fathima-sithara/auth-service/internal/services"
	"github.com/fathima-sithara/auth-service/internal/utils"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// Global validator instance
var validate = validator.New()

// ErrorResponse defines a standard structure for API error messages
type ErrorResponse struct {
	Message    string      `json:"message"`
	StatusCode int         `json:"status_code"`
	Timestamp  string      `json:"timestamp"`
	Errors     interface{} `json:"errors,omitempty"` // For validation errors
}

// NewErrorResponse creates a new ErrorResponse instance
func NewErrorResponse(statusCode int, message string, errs ...interface{}) ErrorResponse {
	response := ErrorResponse{
		Message:    message,
		StatusCode: statusCode,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
	if len(errs) > 0 && errs[0] != nil {
		response.Errors = errs[0]
	}
	return response
}

// Handler holds dependencies for HTTP handlers
type Handler struct {
	authService services.AuthService
}

// NewHandler creates a new Handler instance
func NewHandler(authService services.AuthService) *Handler {
	return &Handler{
		authService: authService,
	}
}

// RequestOTP handles requests for OTP via phone
func (h *Handler) RequestOTP(c *fiber.Ctx) error {
	req := new(models.OTPRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(NewErrorResponse(http.StatusBadRequest, "Invalid request payload"))
	}

	// Validate the request struct
	if err := validate.Struct(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(NewErrorResponse(http.StatusBadRequest, "Validation failed", utils.FormatValidationErrors(err)))
	}

	if err := h.authService.RequestOTP(c.Context(), req.PhoneNumber); err != nil {
		if errors.Is(err, services.ErrOTPRateLimited) {
			return c.Status(http.StatusTooManyRequests).JSON(NewErrorResponse(http.StatusTooManyRequests, err.Error()))
		}
		// Log the underlying error in your service layer or here if necessary for debugging
		return c.Status(http.StatusInternalServerError).JSON(NewErrorResponse(http.StatusInternalServerError, "Failed to request OTP"))
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "OTP requested successfully"})
}

// VerifyOTP handles OTP verification
func (h *Handler) VerifyOTP(c *fiber.Ctx) error {
	req := new(models.OTPVerification)
	if err := c.BodyParser(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(NewErrorResponse(http.StatusBadRequest, "Invalid request payload"))
	}

	// Validate the request struct
	if err := validate.Struct(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(NewErrorResponse(http.StatusBadRequest, "Validation failed", utils.FormatValidationErrors(err)))
	}

	tokens, err := h.authService.VerifyOTP(c.Context(), req.PhoneNumber, req.Code)
	if err != nil {
		if errors.Is(err, services.ErrInvalidOTP) || errors.Is(err, services.ErrOTPExpired) {
			return c.Status(http.StatusUnauthorized).JSON(NewErrorResponse(http.StatusUnauthorized, err.Error()))
		}
		return c.Status(http.StatusInternalServerError).JSON(NewErrorResponse(http.StatusInternalServerError, "Failed to verify OTP"))
	}
	return c.Status(http.StatusOK).JSON(tokens)
}

// RegisterEmail handles new user registration with email and password
func (h *Handler) RegisterEmail(c *fiber.Ctx) error {
	req := new(models.RegisterEmailRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(NewErrorResponse(http.StatusBadRequest, "Invalid request payload"))
	}

	// Validate the request struct
	if err := validate.Struct(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(NewErrorResponse(http.StatusBadRequest, "Validation failed", utils.FormatValidationErrors(err)))
	}

	user, err := h.authService.RegisterEmail(c.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		if errors.Is(err, services.ErrUserAlreadyExists) {
			return c.Status(http.StatusConflict).JSON(NewErrorResponse(http.StatusConflict, err.Error()))
		}
		return c.Status(http.StatusInternalServerError).JSON(NewErrorResponse(http.StatusInternalServerError, "Failed to register user"))
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "User registered. Please check your email for verification.",
		"user_id": user.ID.Hex(),
	})
}

// VerifyEmail handles email verification using a code
func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
	req := new(models.VerifyEmailRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(NewErrorResponse(http.StatusBadRequest, "Invalid request payload"))
	}

	// Validate the request struct
	if err := validate.Struct(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(NewErrorResponse(http.StatusBadRequest, "Validation failed", utils.FormatValidationErrors(err)))
	}

	tokens, err := h.authService.VerifyEmail(c.Context(), req.Email, req.Code)
	if err != nil {
		if errors.Is(err, services.ErrInvalidVerificationCode) || errors.Is(err, services.ErrVerificationCodeExpired) {
			return c.Status(http.StatusUnauthorized).JSON(NewErrorResponse(http.StatusUnauthorized, err.Error()))
		}
		return c.Status(http.StatusInternalServerError).JSON(NewErrorResponse(http.StatusInternalServerError, "Failed to verify email"))
	}
	return c.Status(http.StatusOK).JSON(tokens)
}

// Refresh handles token refresh requests
func (h *Handler) Refresh(c *fiber.Ctx) error {
	req := new(models.RefreshTokenRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(NewErrorResponse(http.StatusBadRequest, "Invalid request payload"))
	}

	// Validate the request struct
	if err := validate.Struct(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(NewErrorResponse(http.StatusBadRequest, "Validation failed", utils.FormatValidationErrors(err)))
	}

	tokens, err := h.authService.RefreshTokens(c.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, services.ErrInvalidRefreshToken) {
			return c.Status(http.StatusUnauthorized).JSON(NewErrorResponse(http.StatusUnauthorized, err.Error()))
		}
		return c.Status(http.StatusInternalServerError).JSON(NewErrorResponse(http.StatusInternalServerError, "Failed to refresh tokens"))
	}
	return c.Status(http.StatusOK).JSON(tokens)
}
