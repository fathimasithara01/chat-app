package service

import (
	"context"

	"github.com/fathima-sithara/notification-service/internal/model"
	"github.com/fathima-sithara/notification-service/internal/repository"
)

type NotificationService struct {
	repo *repository.NotificationRepo
}

func New(repo *repository.NotificationRepo) *NotificationService {
	return &NotificationService{repo}
}

func (s *NotificationService) Send(ctx context.Context, n *model.Notification) error {
	return s.repo.Create(ctx, n)
}

func (s *NotificationService) List(ctx context.Context, userID string) ([]model.Notification, error) {
	return s.repo.GetUserNotifications(ctx, userID)
}
