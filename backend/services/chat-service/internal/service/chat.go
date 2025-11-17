package service

import (
	"context"
	"errors"

	"github.com/fathima-sithara/message-service/internal/repository"
	"github.com/google/uuid"
)

type ChatService struct {
	repo *repository.Repository
}

func NewChatService(r *repository.Repository) *ChatService {
	return &ChatService{repo: r}
}

// Create 1:1 chat (idempotent by participants set)
func (s *ChatService) CreateDM(ctx context.Context, userA, userB, name string) (*repository.Chat, error) {
	if userA == "" || userB == "" || userA == userB {
		return nil, errors.New("invalid participants")
	}
	id := uuid.NewString()
	chat := &repository.Chat{
		ID:      id,
		Name:    name,
		IsGroup: false,
		Members: []string{userA, userB},
	}
	if err := s.repo.CreateChat(ctx, chat); err != nil {
		return nil, err
	}
	return chat, nil
}

func (s *ChatService) CreateGroup(ctx context.Context, owner string, name string, members []string) (*repository.Chat, error) {
	if owner == "" || name == "" {
		return nil, errors.New("invalid request")
	}
	// ensure owner is in members
	found := false
	for _, m := range members {
		if m == owner {
			found = true
			break
		}
	}
	if !found {
		members = append(members, owner)
	}
	id := uuid.NewString()
	chat := &repository.Chat{
		ID:      id,
		IsGroup: true,
		Members: members,
		Name:    name,
	}
	if err := s.repo.CreateChat(ctx, chat); err != nil {
		return nil, err
	}
	return chat, nil
}

func (s *ChatService) GetChat(ctx context.Context, id string) (*repository.Chat, error) {
	return s.repo.GetChat(ctx, id)
}

func (s *ChatService) ListUserChats(ctx context.Context, userID string, limit int64) ([]*repository.Chat, error) {
	return s.repo.ListChatsForUser(ctx, userID, limit)
}

func (s *ChatService) AddMember(ctx context.Context, chatID, userID string) error {
	return s.repo.AddMember(ctx, chatID, userID)
}

func (s *ChatService) RemoveMember(ctx context.Context, chatID, userID string) error {
	return s.repo.RemoveMember(ctx, chatID, userID)
}

func (s *ChatService) UpdateName(ctx context.Context, chatID, name string) error {
	return s.repo.UpdateChatName(ctx, chatID, name)
}
