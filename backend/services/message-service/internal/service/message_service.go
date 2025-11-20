package service

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/fathima-sithara/message-service/internal/domain"
	"github.com/fathima-sithara/message-service/internal/repository"
	"github.com/fathima-sithara/message-service/internal/util"
	"github.com/redis/go-redis/v9"
)

type MessageService struct {
	repo  *repository.MongoRepository
	cache *redis.Client // optional
}

func NewMessageService(r *repository.MongoRepository, c *redis.Client) *MessageService {
	return &MessageService{repo: r, cache: c}
}

func (s *MessageService) SendMessage(ctx context.Context, chatID, senderID, content, msgType string) (*domain.Message, error) {
	if chatID == "" || senderID == "" {
		return nil, errors.New("chat_id and sender_id required")
	}
	id := util.NewID()
	enc := base64.StdEncoding.EncodeToString([]byte(content))

	m := &domain.Message{
		ID:         id,
		ChatID:     chatID,
		SenderID:   senderID,
		Content:    enc,
		MsgType:    msgType,
		CreatedAt:  time.Now().UTC(),
		ReadBy:     []string{}, // initialize arrays/maps
		DeletedFor: []string{},
		Reactions:  map[string][]string{},
	}

	if err := s.repo.SaveMessage(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *MessageService) ListMessages(ctx context.Context, chatID string, limit int64, before time.Time) ([]*domain.Message, error) {
	msgs, err := s.repo.GetMessages(ctx, chatID, limit, before)
	if err != nil {
		return nil, err
	}
	for _, m := range msgs {
		if m.Content != "" {
			if b, err := base64.StdEncoding.DecodeString(m.Content); err == nil {
				m.Content = string(b)
			}
		}
	}
	return msgs, nil
}

func (s *MessageService) MarkRead(ctx context.Context, messageID, userID string) (string, error) {
	return s.repo.MarkRead(ctx, messageID, userID)
}

func (s *MessageService) EditMessage(ctx context.Context, messageID, userID, newContent string) (string, error) {
	enc := base64.StdEncoding.EncodeToString([]byte(newContent))
	return s.repo.EditMessage(ctx, messageID, enc, time.Now().UTC())
}

func (s *MessageService) DeleteMessageForUser(ctx context.Context, messageID, userID string) (string, error) {
	return s.repo.SoftDeleteMessage(ctx, messageID, userID)
}

func (s *MessageService) DeleteMessageForAll(ctx context.Context, messageID, userID string) (string, error) {
	return s.repo.DeleteMessageForAll(ctx, messageID)
}

func (s *MessageService) AddReaction(ctx context.Context, messageID, emoji, userID string) (string, error) {
	return s.repo.AddReaction(ctx, messageID, emoji, userID)
}

func (s *MessageService) GetLastMessage(ctx context.Context, chatID string) (*domain.Message, error) {
	m, err := s.repo.GetLastMessage(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if m.Content != "" {
		if b, err := base64.StdEncoding.DecodeString(m.Content); err == nil {
			m.Content = string(b)
		}
	}
	return m, nil
}
