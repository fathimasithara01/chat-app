package handler

import (
	"strconv"

	"githhub.com/fathimasithara/user-service/internal/domain"
	"githhub.com/fathimasithara/user-service/internal/service"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type UserHandler struct {
	userService *service.UserService
	logger      *zap.Logger
}

func NewUserHandler(userService *service.UserService, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// RegisterRoutes registers all user-related routes
func (h *UserHandler) RegisterRoutes(app *fiber.App, jwtMiddleware fiber.Handler) {
	api := app.Group("/api/users")

	api.Post("/register", h.Register)
	api.Post("/login", h.Login)
	api.Post("/refresh", h.RefreshToken)

	secured := api.Group("/", jwtMiddleware)
	secured.Get("/", h.List)
	secured.Get("/:id", h.Get)
	secured.Put("/:id", h.Update)
	secured.Delete("/:id", h.Delete)
}

// Register handles user registration
func (h *UserHandler) Register(c *fiber.Ctx) error {
	var req domain.UserRegisterRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Warn("invalid register request", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	user, err := h.userService.RegisterUser(c.Context(), &req)
	if err != nil {
		h.logger.Warn("registration failed", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user": user})
}

// Login handles user authentication
func (h *UserHandler) Login(c *fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		h.logger.Warn("invalid login request", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	tokens, err := h.userService.LoginUser(c.Context(), req.Email, req.Password)
	if err != nil {
		h.logger.Warn("login failed", zap.Error(err))
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"tokens": tokens})
}

// RefreshToken generates a new access token using refresh token
func (h *UserHandler) RefreshToken(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&req); err != nil {
		h.logger.Warn("invalid refresh token request", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	accessToken, err := h.userService.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		h.logger.Warn("refresh token failed", zap.Error(err))
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"access_token": accessToken})
}

// Get retrieves a user by ID
func (h *UserHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	user, err := h.userService.GetUserByID(c.Context(), id)
	if err != nil {
		h.logger.Warn("get user failed", zap.String("userID", id), zap.Error(err))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"user": user})
}

// Update updates a user's information
func (h *UserHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var req domain.UserUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Warn("invalid update request", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := h.userService.UpdateUser(c.Context(), id, &req); err != nil {
		h.logger.Warn("update user failed", zap.String("userID", id), zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Delete deletes a user by ID
func (h *UserHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.userService.DeleteUser(c.Context(), id); err != nil {
		h.logger.Warn("delete user failed", zap.String("userID", id), zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// List retrieves a paginated list of users
func (h *UserHandler) List(c *fiber.Ctx) error {
	limit, _ := strconv.ParseInt(c.Query("limit", "10"), 10, 64)
	offset, _ := strconv.ParseInt(c.Query("offset", "0"), 10, 64)

	users, err := h.userService.ListUsers(c.Context(), limit, offset)
	if err != nil {
		h.logger.Warn("list users failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"users": users, "limit": limit, "offset": offset})
}
