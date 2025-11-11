package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/fathima-sithara/chat-service/internal/config"
	"github.com/fathima-sithara/chat-service/internal/domain"
	"github.com/fathima-sithara/chat-service/internal/kafka"
	repo "github.com/fathima-sithara/chat-service/internal/repository"
)

type CommandService struct {
	repo  *repo.MongoRepository
	cache *redis.Client
	prod  *kafka.Producer
	cfg   *config.Config
}

func NewCommandService(r *repo.MongoRepository, rdb *redis.Client, prod *kafka.Producer, cfg *config.Config) *CommandService {
	return &CommandService{repo: r, cache: rdb, prod: prod, cfg: cfg}
}

type SendMessageDTO struct {
	ChatID   string
	SenderID string
	Content  string
	MsgType  string
	MsgID    string
	Metadata map[string]string
	ReplyTo  string
}

func (s *CommandService) CreateMessage(ctx context.Context, dto *SendMessageDTO) (*domain.Message, error) {
	id := dto.MsgID
	if id == "" {
		id = uuid.NewString()
	}
	now := time.Now().UTC()

	enc := base64.StdEncoding.EncodeToString([]byte(dto.Content)) // store base64; encryption handled by message-service if needed

	m := &domain.Message{
		ID:         id,
		ChatID:     dto.ChatID,
		SenderID:   dto.SenderID,
		Content:    enc,
		MsgType:    dto.MsgType,
		Encrypted:  false,
		Metadata:   dto.Metadata,
		ReplyTo:    dto.ReplyTo,
		CreatedAt:  now,
		Delivered:  false,
		ReadBy:     []string{},
		DeletedFor: []string{},
		Reactions:  map[string][]string{},
	}

	if err := s.repo.SaveMessage(ctx, m); err != nil {
		return nil, err
	}

	// cache top-N (recent) for quick fetch
	cacheKey := "chat:" + dto.ChatID + ":recent"
	_ = s.cache.LPush(ctx, cacheKey, enc).Err()
	_ = s.cache.LTrim(ctx, cacheKey, 0, 99).Err()
	_ = s.cache.Expire(ctx, cacheKey, 24*time.Hour).Err()

	// publish event to TopicOut
	_ = s.prod.PublishMessage(ctx, id, map[string]interface{}{
		"event":   "message.new",
		"message": m,
	})

	return m, nil
}

func (s *CommandService) MarkRead(ctx context.Context, msgID, userID string) (string, error) {
	chatID, err := s.repo.MarkRead(ctx, msgID, userID)
	if err != nil {
		return "", err
	}
	_ = s.prod.PublishMessage(ctx, msgID, map[string]interface{}{"event": "message.read", "msg_id": msgID, "user": userID})
	return chatID, nil
}

// func (s *CommandService) DeleteMessage(ctx context.Context, msgID, userID, forParam string) (string, error) {
// 	if forParam == "me" {
// 		chatID, err := s.repo.SoftDeleteMessage(ctx, msgID, userID)
// 		if err != nil {
// 			return "", err
// 		}
// 		_ = s.prod.PublishMessage(ctx, msgID, map[string]interface{}{"event": "message.deleted", "msg_id": msgID, "for": "me", "user": userID})
// 		return chatID, nil
// 	}
// 	chatID, err := s.repo.DeleteMessageForAll(ctx, msgID)
// 	if err != nil {
// 		return "", err
// 	}
// 	_ = s.prod.PublishMessage(ctx, msgID, map[string]interface{}{"event": "message.deleted", "msg_id": msgID, "for": "all"})
// 	return chatID, nil
// }

func (s *CommandService) AddReaction(ctx context.Context, msgID, emoji, userID string) (string, error) {
	chatID, err := s.repo.AddReaction(ctx, msgID, emoji, userID)
	if err != nil {
		return "", err
	}
	_ = s.prod.PublishMessage(ctx, msgID, map[string]interface{}{"event": "message.reaction", "msg_id": msgID, "emoji": emoji, "user": userID})
	return chatID, nil
}

func (s *CommandService) GetMediaUploadURL(ctx context.Context, fileType string, fileSize int64) (string, error) {
	// stub
	return "https://example-storage.local/upload/" + time.Now().Format("20060102150405"), nil
}

// Public API used by HTTP handlers

type SendMessageCommand struct {
	ChatID  string
	UserID  string
	Content string
}

func (s *CommandService) SendMessage(ctx context.Context, cmd SendMessageCommand) (*domain.Message, error) {
	return s.CreateMessage(ctx, &SendMessageDTO{
		ChatID:   cmd.ChatID,
		SenderID: cmd.UserID,
		Content:  cmd.Content,
		MsgType:  "text",
	})
}

func (s *CommandService) MarkAsRead(ctx context.Context, msgID, userID string) error {
	_, err := s.MarkRead(ctx, msgID, userID)
	return err
}

// func (s *CommandService) EditMessage(ctx context.Context, msgID, userID, content string) (*domain.Message, error) {
// 	// Ensure message belongs to user before edit
// 	// (optional authorization check via repo)
// 	_, m, err := s.EditMessage(ctx, msgID, content)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return m, nil
// }

func (s *CommandService) EditMessage(ctx context.Context, msgID, userID, newContent string) (*domain.Message, error) {
	// 1) fetch existing message
	m, err := s.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return nil, err
	}

	// 2) ownership check
	if m.SenderID != userID {
		return nil, fmt.Errorf("unauthorized: only sender can edit message")
	}

	// 3) encrypt/encode new content (here we base64-encode as in other places)
	enc := base64.StdEncoding.EncodeToString([]byte(newContent))
	now := time.Now().UTC()

	// 4) update via repo.EditMessage (this updates content and edited_at, returns chatID)
	_, err = s.repo.EditMessage(ctx, msgID, enc, now)
	if err != nil {
		return nil, err
	}

	// 5) prepare updated message to return
	// Option A: re-fetch from DB for the canonical updated document
	updated, err := s.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		// repo updated but re-fetch failed; return fallback constructed object
		return &domain.Message{
			ID:         msgID,
			ChatID:     m.ChatID,
			SenderID:   m.SenderID,
			Content:    newContent, // decoded already
			MsgType:    m.MsgType,
			Encrypted:  m.Encrypted,
			Metadata:   m.Metadata,
			ReplyTo:    m.ReplyTo,
			CreatedAt:  m.CreatedAt,
			EditedAt:   &now,
			Delivered:  m.Delivered,
			ReadBy:     m.ReadBy,
			DeletedFor: m.DeletedFor,
			Reactions:  m.Reactions,
		}, nil
	}

	// decode base64 content before returning
	if updated.Content != "" {
		if b, err := base64.StdEncoding.DecodeString(updated.Content); err == nil {
			updated.Content = string(b)
		}
	}

	// 6) publish event (message.edited) with the updated message
	_ = s.prod.PublishMessage(ctx, updated.ID, map[string]interface{}{
		"event":   "message.edited",
		"message": updated,
	})

	return updated, nil
}
func (s *CommandService) GeneratePresignedUploadURL(ctx context.Context) (string, error) {
	return s.GetMediaUploadURL(ctx, "generic", 0)
}

func (s *CommandService) DeleteMessage(ctx context.Context, msgID, userID string) (string, error) {
	chatID, err := s.repo.SoftDeleteMessage(ctx, msgID, userID)
	if err != nil {
		return "", err
	}

	_ = s.prod.PublishMessage(ctx, msgID, map[string]interface{}{
		"event": "message.deleted",
		"msgId": msgID,
		"user":  userID,
	})

	return chatID, nil
}
