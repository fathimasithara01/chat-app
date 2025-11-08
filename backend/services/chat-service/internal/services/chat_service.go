package services

import (
	"context"
	"encoding/json"

	"github.com/fathima-sithara/chat-service/internal/kafka"
	"github.com/fathima-sithara/chat-service/internal/models"
	"github.com/fathima-sithara/chat-service/internal/repository"
	"github.com/rs/zerolog/log"
)

type ChatService struct {
	repo     *repository.MongoRepository
	producer *kafka.Producer
}

func NewChatService(r *repository.MongoRepository, p *kafka.Producer) *ChatService {
	return &ChatService{repo: r, producer: p}
}

func (s *ChatService) SendMessage(ctx context.Context, msg *models.Message) error {
	if err := s.repo.SaveMessage(ctx, msg); err != nil {
		return err
	}
	b, _ := json.Marshal(msg)
	if err := s.producer.PublishMessage(ctx, msg.SenderID, b); err != nil {
		log.Error().Err(err).Msg("kafka publish")
	}
	return nil
}
