package service

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/fathima-sithara/chat-service/internal/config"
	"github.com/fathima-sithara/chat-service/internal/domain"
	repo "github.com/fathima-sithara/chat-service/internal/repository"
	"github.com/redis/go-redis/v9"
)

type QueryService struct {
	repo  *repo.MongoRepository
	cache *redis.Client
	cfg   *config.Config
}

func NewQueryService(r *repo.MongoRepository, rdb *redis.Client, cfg *config.Config) *QueryService {
	return &QueryService{repo: r, cache: rdb, cfg: cfg}
}

func (s *QueryService) GetMessages(ctx context.Context, chatID string, limit int64, before time.Time) ([]*domain.Message, error) {
	msgs, err := s.repo.GetMessages(ctx, chatID, limit, before)
	if err != nil {
		return nil, err
	}
	// decode base64 content before returning
	for _, m := range msgs {
		if m.Content != "" {
			if b, err := base64.StdEncoding.DecodeString(m.Content); err == nil {
				m.Content = string(b)
			}
		}
	}
	return msgs, nil
}

func (s *QueryService) GetLastMessage(ctx context.Context, chatID string) (*domain.Message, error) {
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

func (s *QueryService) ListMessages(ctx context.Context, chatID string) ([]*domain.Message, error) {
	msgs, err := s.repo.GetMessages(ctx, chatID, 50, time.Now())
	if err != nil {
		return nil, err
	}

	for _, m := range msgs {
		if b, err := base64.StdEncoding.DecodeString(m.Content); err == nil {
			m.Content = string(b)
		}
	}

	return msgs, nil
}

func (s *QueryService) LastMessage(ctx context.Context, chatID string) (*domain.Message, error) {
	return s.GetLastMessage(ctx, chatID)
}
