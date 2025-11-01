package service

import (
	"context"
	"time"

	"github.com/yourorg/chat-app/services/chat-service/internal/kafka"
	"github.com/yourorg/chat-app/services/chat-service/internal/models"
	"github.com/yourorg/chat-app/services/chat-service/internal/repository"
)

type ChatService struct {
	repo repository.Repository
	kp   *kafka.Producer
}

func NewChatService(repo repository.Repository, kp *kafka.Producer) *ChatService {
	return &ChatService{repo: repo, kp: kp}
}

func (s *ChatService) SendMessage(ctx context.Context, m *models.Message) (*models.Message, error) {
	inserted, err := s.repo.InsertMessage(ctx, m)
	if err != nil {
		return nil, err
	}
	_ = s.kp.PublishMessageSent(ctx, map[string]any{
		"message_id": inserted.ID,
		"conversation_id": inserted.ConversationID,
		"sender_id": inserted.SenderID,
		"content": inserted.Content,
		"created_at": inserted.CreatedAt,
	})
	return inserted, nil
}

func (s *ChatService) GetHistory(ctx context.Context, convID string, limit int64, before time.Time) ([]*models.Message, error) {
	return s.repo.GetMessages(ctx, convID, limit, before)
}
