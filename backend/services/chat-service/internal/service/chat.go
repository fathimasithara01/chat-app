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
	repo *repository.Repository
	pub  *events.Publisher
}

func NewChatService(r *repository.Repository, p *events.Publisher) *ChatService {
	return &ChatService{repo: r, pub: p}
}

func contains(arr []string, id string) bool {
	for _, x := range arr {
		if x == id {
			return true
		}
	}
	return false
}

func (s *ChatService) CreateDM(ctx context.Context, a, b, name string) (*repository.Chat, error) {
	if a == "" || b == "" || a == b {
		return nil, errors.New("invalid participants")
	}
	chat := &repository.Chat{
		ID:        uuid.NewString(),
		Name:      name,
		IsGroup:   false,
		Members:   []string{a, b},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.repo.CreateChat(ctx, chat); err != nil {
		return nil, err
	}
	if s.pub != nil {
		_ = s.pub.PublishChatCreated(chat.ID, chat.Name, chat.Members, chat.IsGroup)
	}
	return chat, nil
}

func (s *ChatService) CreateGroup(ctx context.Context, owner, name string, members []string) (*repository.Chat, error) {
	if owner == "" || name == "" {
		return nil, errors.New("invalid data")
	}
	if !contains(members, owner) {
		members = append(members, owner)
	}
	chat := &repository.Chat{ID: uuid.NewString(), Name: name, IsGroup: true, Members: members, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := s.repo.CreateChat(ctx, chat); err != nil {
		return nil, err
	}
	if s.pub != nil {
		_ = s.pub.PublishChatCreated(chat.ID, chat.Name, chat.Members, chat.IsGroup)
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
	chat, err := s.repo.GetChat(ctx, chatID)
	if err != nil {
		return err
	}
	if contains(chat.Members, userID) {
		return errors.New("already a member")
	}
	chat.Members = append(chat.Members, userID)
	chat.UpdatedAt = time.Now().UTC()
	return s.repo.UpdateChat(ctx, chat)
}

func (s *ChatService) RemoveMember(ctx context.Context, chatID, userID string) error {
	chat, err := s.repo.GetChat(ctx, chatID)
	if err != nil {
		return err
	}
	newList := []string{}
	for _, m := range chat.Members {
		if m != userID {
			newList = append(newList, m)
		}
	}
	chat.Members = newList
	chat.UpdatedAt = time.Now().UTC()
	return s.repo.UpdateChat(ctx, chat)
}

func (s *ChatService) UpdateName(ctx context.Context, chatID, name string) error {
	chat, err := s.repo.GetChat(ctx, chatID)
	if err != nil {
		return err
	}
	chat.Name = name
	chat.UpdatedAt = time.Now().UTC()
	return s.repo.UpdateChat(ctx, chat)
}
