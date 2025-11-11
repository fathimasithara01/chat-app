package service

import (
	"context"
	"encoding/base64"
	"time"

	"crypto/cipher"

	"github.com/fathima-sithara/message-service/internal/config"
	"github.com/fathima-sithara/message-service/internal/crypto"
	"github.com/fathima-sithara/message-service/internal/domain"
	repo "github.com/fathima-sithara/message-service/internal/repository"
	"github.com/redis/go-redis/v9"
)

type QueryService struct {
	repo  *repo.MongoRepository
	cache *redis.Client
	aead  cipher.AEAD
	cfg   *config.Config
}

func NewQueryService(r *repo.MongoRepository, rdb *redis.Client, aesKey []byte, cfg *config.Config) *QueryService {
	aead, err := crypto.NewGCM(aesKey)
	if err != nil {
		panic(err)
	}
	return &QueryService{repo: r, cache: rdb, aead: aead, cfg: cfg}
}

func (s *QueryService) GetMessages(ctx context.Context, chatID string, limit int64, before time.Time) ([]*domain.Message, error) {
	msgs, err := s.repo.GetMessages(ctx, chatID, limit, before)
	if err != nil {
		return nil, err
	}
	for _, m := range msgs {
		if m.Encrypted && m.Content != "" {
			ct, err := base64.StdEncoding.DecodeString(m.Content)
			if err != nil {
				continue
			}
			plain, err := crypto.Decrypt(s.aead, ct)
			if err != nil {
				continue
			}
			m.Content = string(plain)
			m.Encrypted = false
		}
	}
	return msgs, nil
}

func (s *QueryService) GetLastMessage(ctx context.Context, chatID string) (*domain.Message, error) {
	m, err := s.repo.GetLastMessage(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if m.Encrypted && m.Content != "" {
		ct, err := base64.StdEncoding.DecodeString(m.Content)
		if err == nil {
			if plain, err := crypto.Decrypt(s.aead, ct); err == nil {
				m.Content = string(plain)
				m.Encrypted = false
			}
		}
	}
	return m, nil
}

func (s *QueryService) GetMessageByID(ctx context.Context, msgID string) (*domain.Message, error) {
	m, err := s.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if m.Encrypted && m.Content != "" {
		ct, err := base64.StdEncoding.DecodeString(m.Content)
		if err == nil {
			if plain, err := crypto.Decrypt(s.aead, ct); err == nil {
				m.Content = string(plain)
				m.Encrypted = false
			}
		}
	}
	return m, nil
}
