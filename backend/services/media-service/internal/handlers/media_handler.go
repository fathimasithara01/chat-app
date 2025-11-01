package handlers

import (
	"context"
	"media-service/internal/auth"
	service "media-service/internal/services"
	utils "media-service/internal/utis"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Handler struct {
	verifier *auth.JWTVerifier
	svc      *service.MediaService
}

func NewHandler(v *auth.JWTVerifier, svc *service.MediaService) *Handler {
	return &Handler{verifier: v, svc: svc}
}

// POST /upload (multipart/form-data 'file')
func (h *Handler) Upload(c *fiber.Ctx) error {
	// auth: Authorization: Bearer <token>
	token := c.Get("Authorization")
	if token == "" {
		return utils.JSONError(c, fiber.StatusUnauthorized, "missing auth")
	}
	// token may be "Bearer x", strip
	if strings.HasPrefix(token, "Bearer ") {
		token = strings.TrimPrefix(token, "Bearer ")
	}
	userID, err := h.verifier.VerifyToken(token)
	if err != nil {
		return utils.JSONError(c, fiber.StatusUnauthorized, "invalid token")
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return utils.JSONError(c, fiber.StatusBadRequest, "file missing")
	}
	f, err := fileHeader.Open()
	if err != nil {
		return utils.JSONError(c, fiber.StatusInternalServerError, "cannot open file")
	}
	defer f.Close()
	data := make([]byte, fileHeader.Size)
	_, _ = f.Read(data)

	// basic content-type detection
	ct := fileHeader.Header.Get("Content-Type")
	if ct == "" {
		ct = http.DetectContentType(data)
	}

	// for this implementation handle images only for thumbnail
	if strings.HasPrefix(ct, "image/") {
		media, err := h.svc.UploadImage(context.Background(), userID, fileHeader.Filename, ct, data)
		if err != nil {
			return utils.JSONError(c, fiber.StatusInternalServerError, err.Error())
		}
		return utils.JSONSuccess(c, fiber.StatusCreated, media)
	}

	// fallback - store as generic file
	media, err := h.svc.UploadFile(context.Background(), userID, fileHeader.Filename, ct, data)
	if err != nil {
		return utils.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.JSONSuccess(c, fiber.StatusCreated, media)
}

// GET /media/:id/url -> returns presigned URL
func (h *Handler) GetSignedURL(c *fiber.Ctx) error {
	id := c.Params("id")
	// validate id
	if _, err := primitive.ObjectIDFromHex(id); err == nil {
		// if storing Mongo ObjectIDs, but we used UUIDs - keep simple
	}
	// get media by id
	media, err := h.svc.GetByID(context.Background(), id)
	if err != nil {
		return utils.JSONError(c, fiber.StatusNotFound, "not found")
	}
	// if object has public URL and configured publicRead, return it
	if media.URL != "" {
		// if media.URL empty, use presign
		return utils.JSONSuccess(c, fiber.StatusOK, fiber.Map{"url": media.URL})
	}
	// presign using key
	presigned, err := h.svc.GetPresignedURL(context.Background(), media.Key)
	if err != nil {
		return utils.JSONError(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.JSONSuccess(c, fiber.StatusOK, fiber.Map{"url": presigned})
}
