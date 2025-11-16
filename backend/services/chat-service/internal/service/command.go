package service

import (
	"context"
	"encoding/base64"
	"time"

	"crypto/cipher"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/fathima-sithara/message-service/internal/config"
	"github.com/fathima-sithara/message-service/internal/crypto"
	"github.com/fathima-sithara/message-service/internal/domain"
	"github.com/fathima-sithara/message-service/internal/kafka"
	repo "github.com/fathima-sithara/message-service/internal/repository"
)

type CommandService struct {
	repo  *repo.MongoRepository
	cache *redis.Client
	prod  *kafka.Producer
	aead  cipher.AEAD
	cfg   *config.Config
}

func NewCommandService(r *repo.MongoRepository, rdb *redis.Client, prod *kafka.Producer, aesKey []byte, cfg *config.Config) *CommandService {
	aead, err := crypto.NewGCM(aesKey)
	if err != nil {
		panic(err)
	}
	return &CommandService{
		repo:  r,
		cache: rdb,
		prod:  prod,
		aead:  aead,
		cfg:   cfg,
	}
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

	ct, err := crypto.Encrypt(s.aead, []byte(dto.Content))
	if err != nil {
		return nil, err
	}
	encBase64 := base64.StdEncoding.EncodeToString(ct)

	m := &domain.Message{
		ID:         id,
		ChatID:     dto.ChatID,
		SenderID:   dto.SenderID,
		Content:    encBase64,
		MsgType:    dto.MsgType,
		Encrypted:  true,
		Metadata:   dto.Metadata,
		ReplyTo:    dto.ReplyTo,
		CreatedAt:  now,
		ReadBy:     []string{},
		DeletedFor: []string{},
		Reactions:  map[string][]string{},
	}

	if err := s.repo.SaveMessage(ctx, m); err != nil {
		return nil, err
	}

	cacheKey := "chat:" + dto.ChatID + ":recent"
	_ = s.cache.LPush(ctx, cacheKey, encBase64).Err()
	_ = s.cache.LTrim(ctx, cacheKey, 0, 99).Err()
	_ = s.cache.Expire(ctx, cacheKey, 24*time.Hour).Err()

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
	_ = s.prod.PublishMessage(ctx, msgID, map[string]interface{}{
		"event":  "message.read",
		"msg_id": msgID,
		"user":   userID,
		"chat":   chatID,
	})
	return chatID, nil
}

func (s *CommandService) EditMessage(ctx context.Context, msgID, newContent string) (*domain.Message, string, error) {
	ct, err := crypto.Encrypt(s.aead, []byte(newContent))
	if err != nil {
		return nil, "", err
	}
	enc := base64.StdEncoding.EncodeToString(ct)
	now := time.Now().UTC()
	chatID, err := s.repo.EditMessage(ctx, msgID, enc, now)
	if err != nil {
		return nil, "", err
	}
	msg := &domain.Message{ID: msgID, Content: enc, Encrypted: true}
	_ = s.prod.PublishMessage(ctx, msgID, map[string]interface{}{
		"event":  "message.edited",
		"msg_id": msgID,
		"chat":   chatID,
	})
	return msg, chatID, nil
}

func (s *CommandService) DeleteMessage(ctx context.Context, msgID, userID, forParam string) (string, error) {
	chatID, err := s.repo.GetChatIDByMessage(ctx, msgID)
	if err != nil {
		return "", err
	}
	if forParam == "me" {
		if err := s.repo.SoftDeleteMessage(ctx, msgID, userID); err != nil {
			return "", err
		}
		_ = s.prod.PublishMessage(ctx, msgID, map[string]interface{}{
			"event":  "message.deleted",
			"msg_id": msgID,
			"for":    "me",
			"user":   userID,
			"chat":   chatID,
		})
		return chatID, nil
	}
	if err := s.repo.DeleteMessageForAll(ctx, msgID); err != nil {
		return "", err
	}
	_ = s.prod.PublishMessage(ctx, msgID, map[string]interface{}{
		"event":  "message.deleted",
		"msg_id": msgID,
		"for":    "all",
		"chat":   chatID,
	})
	return chatID, nil
}

func (s *CommandService) AddReaction(ctx context.Context, msgID, emoji, userID string) error {
	if err := s.repo.AddReaction(ctx, msgID, emoji, userID); err != nil {
		return err
	}
	_ = s.prod.PublishMessage(ctx, msgID, map[string]interface{}{
		"event": "message.reaction", "msg_id": msgID, "emoji": emoji, "user": userID,
	})
	return nil
}

func (s *CommandService) GetMediaUploadURL(ctx context.Context, fileType string, fileSize int64) (string, error) {
	// stub: replace with AWS S3/MinIO implementation
	return "https://example-storage.local/upload/" + time.Now().Format("20060102150405"), nil
}
