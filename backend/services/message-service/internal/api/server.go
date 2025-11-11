package api

import (
	"strconv"
	"time"

	"github.com/fathima-sithara/message-service/internal/config"
	"github.com/fathima-sithara/message-service/internal/crypto"
	"github.com/fathima-sithara/message-service/internal/service"
	"github.com/fathima-sithara/message-service/internal/ws"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

type Server struct {
	cmd *service.CommandService
	qry *service.QueryService
	ws  *ws.Server
	app *fiber.App
}

func NewServer(cfg *config.Config, cmd *service.CommandService, qry *service.QueryService, wsrv *ws.Server, jwtValidator *crypto.JWTValidator) *fiber.App {
	s := &Server{cmd: cmd, qry: qry, ws: wsrv, app: fiber.New()}
	s.app.Use(logger.New())

	api := s.app.Group("/v1")
	api.Use(JWTAuthMiddleware(jwtValidator))

	api.Post("/messages", s.sendMessage)
	api.Get("/chats/:chat_id/messages", s.listMessages)
	api.Post("/messages/:msg_id/read", s.markRead)
	api.Patch("/messages/:msg_id", s.editMessage)
	api.Delete("/messages/:msg_id", s.deleteMessage)
	api.Get("/ws", websocket.New(s.ws.HandleWS))
	api.Post("/media/upload-url", s.mediaUploadURL)
	api.Get("/chats/:chat_id/last-message", s.lastMessage)

	return s.app
}

type sendReq struct {
	ChatID   string            `json:"chat_id"`
	Content  string            `json:"content"`
	MsgType  string            `json:"msg_type"`
	MsgID    string            `json:"msg_id,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func (s *Server) sendMessage(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var req sendReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	dto := &service.SendMessageDTO{
		ChatID:   req.ChatID,
		SenderID: userID,
		Content:  req.Content,
		MsgType:  req.MsgType,
		MsgID:    req.MsgID,
		Metadata: req.Metadata,
	}
	m, err := s.cmd.CreateMessage(c.Context(), dto)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	s.ws.BroadcastMessage(req.ChatID, map[string]interface{}{
		"event": "message.new",
		"data":  m,
	})
	return c.Status(201).JSON(m)
}

func (s *Server) listMessages(c *fiber.Ctx) error {
	chatID := c.Params("chat_id")
	limitQ := c.Query("limit", "50")
	beforeQ := c.Query("before", "")
	limit, _ := strconv.ParseInt(limitQ, 10, 64)
	var before time.Time
	if beforeQ != "" {
		if t, err := time.Parse(time.RFC3339, beforeQ); err == nil {
			before = t
		}
	}
	msgs, err := s.qry.GetMessages(c.Context(), chatID, limit, before)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(msgs)
}

func (s *Server) markRead(c *fiber.Ctx) error {
	msgID := c.Params("msg_id")
	userID := c.Locals("user_id").(string)
	chatID, err := s.cmd.MarkRead(c.Context(), msgID, userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	s.ws.BroadcastMessage(chatID, map[string]interface{}{
		"event":  "message.read",
		"msg_id": msgID,
		"user":   userID,
	})
	return c.JSON(fiber.Map{"status": "ok"})
}

func (s *Server) editMessage(c *fiber.Ctx) error {
	msgID := c.Params("msg_id")
	var body struct {
		NewContent string `json:"new_content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	m, chatID, err := s.cmd.EditMessage(c.Context(), msgID, body.NewContent)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	// get decrypted version for broadcast
	if msgFull, err2 := s.qry.GetMessageByID(c.Context(), msgID); err2 == nil {
		s.ws.BroadcastMessage(chatID, map[string]interface{}{"event": "message.edited", "msg": msgFull})
	} else {
		s.ws.BroadcastMessage(chatID, map[string]interface{}{"event": "message.edited", "msg": m})
	}
	return c.JSON(m)
}

func (s *Server) deleteMessage(c *fiber.Ctx) error {
	msgID := c.Params("msg_id")
	forParam := c.Query("for", "me")
	userID := c.Locals("user_id").(string)
	// get chat id first (so we can broadcast)
	chatID, _ := s.qry.GetMessageByID(c.Context(), msgID)
	chatIDStr := ""
	if chatID != nil {
		chatIDStr = chatID.ChatID
	}
	chatIDRes, err := s.cmd.DeleteMessage(c.Context(), msgID, userID, forParam)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if chatIDStr == "" {
		chatIDStr = chatIDRes
	}
	if chatIDStr != "" {
		s.ws.BroadcastMessage(chatIDStr, map[string]interface{}{
			"event":  "message.deleted",
			"msg_id": msgID,
			"for":    forParam,
		})
	}
	return c.JSON(fiber.Map{"status": "deleted"})
}

func (s *Server) mediaUploadURL(c *fiber.Ctx) error {
	var body struct {
		FileType string `json:"file_type"`
		FileSize int64  `json:"file_size"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	url, err := s.cmd.GetMediaUploadURL(c.Context(), body.FileType, body.FileSize)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"upload_url": url})
}

func (s *Server) lastMessage(c *fiber.Ctx) error {
	chatID := c.Params("chat_id")
	m, err := s.qry.GetLastMessage(c.Context(), chatID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(m)
}
