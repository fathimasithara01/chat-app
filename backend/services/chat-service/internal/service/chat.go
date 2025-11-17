package service

import (
	"context"
	"errors"
	"time"

	"github.com/fathima-sithara/message-service/internal/events"
	"github.com/fathima-sithara/message-service/internal/repository"
	"github.com/google/uuid"
)

type ChatService struct {
	repo      *repository.Repository
	publisher *events.Publisher
}

func NewChatService(r *repository.Repository, p *events.Publisher) *ChatService {
	return &ChatService{repo: r, publisher: p}
}

// helper to check if a user exists in slice
func contains(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

// CreateDM creates a 1:1 chat
func (s *ChatService) CreateDM(ctx context.Context, userA, userB, name string) (*repository.Chat, error) {
	if userA == "" || userB == "" || userA == userB {
		return nil, errors.New("invalid participants")
	}

	id := uuid.NewString()
	chat := &repository.Chat{
		ID:        id,
		Name:      name,
		IsGroup:   false,
		Members:   []string{userA, userB},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.repo.CreateChat(ctx, chat); err != nil {
		return nil, err
	}

	if s.publisher != nil {
		_ = s.publisher.PublishChatCreated(chat.ID, chat.Name, chat.Members, chat.IsGroup)
	}

	return chat, nil
}

// CreateGroup creates a group chat
func (s *ChatService) CreateGroup(ctx context.Context, owner, name string, members []string) (*repository.Chat, error) {
	if owner == "" || name == "" {
		return nil, errors.New("invalid request")
	}

	if !contains(members, owner) {
		members = append(members, owner)
	}

	id := uuid.NewString()
	chat := &repository.Chat{
		ID:        id,
		Name:      name,
		IsGroup:   true,
		Members:   members,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.repo.CreateChat(ctx, chat); err != nil {
		return nil, err
	}

	if s.publisher != nil {
		_ = s.publisher.PublishChatCreated(chat.ID, chat.Name, chat.Members, chat.IsGroup)
	}

	return chat, nil
}

// GetChat returns a chat by ID
func (s *ChatService) GetChat(ctx context.Context, id string) (*repository.Chat, error) {
	return s.repo.GetChat(ctx, id)
}

// ListUserChats returns all chats for a user
//
//	func (s *ChatService) ListUserChats(ctx context.Context, userID string, limit int64) ([]*repository.Chat, error) {
//		return s.repo.ListChatsForUser(ctx, userID, limit)
//	}
func (s *ChatService) ListUserChats(ctx context.Context, userID string, limit int64) ([]*repository.Chat, error) {
	return s.repo.ListChatsForUser(ctx, userID, limit)
}

// AddMember adds a user to a chat
func (s *ChatService) AddMember(ctx context.Context, chatID, userID string) error {
	chat, err := s.repo.GetChat(ctx, chatID)
	if err != nil {
		return err
	}

	if contains(chat.Members, userID) {
		return errors.New("user already a member")
	}

	chat.Members = append(chat.Members, userID)
	chat.UpdatedAt = time.Now().UTC()

	return s.repo.UpdateChat(ctx, chat)
}

// RemoveMember removes a user from a chat
func (s *ChatService) RemoveMember(ctx context.Context, chatID, userID string) error {
	chat, err := s.repo.GetChat(ctx, chatID)
	if err != nil {
		return err
	}

	newMembers := []string{}
	for _, m := range chat.Members {
		if m != userID {
			newMembers = append(newMembers, m)
		}
	}

	chat.Members = newMembers
	chat.UpdatedAt = time.Now().UTC()

	return s.repo.UpdateChat(ctx, chat)
}

// UpdateName updates chat name
func (s *ChatService) UpdateName(ctx context.Context, chatID, name string) error {
	chat, err := s.repo.GetChat(ctx, chatID)
	if err != nil {
		return err
	}

	chat.Name = name
	chat.UpdatedAt = time.Now().UTC()

	return s.repo.UpdateChat(ctx, chat)
}
