package repository

import (
	"context"
	"time"

	"github.com/fathima-sithara/notification-service/internal/model"
	"go.mongodb.org/mongo-driver/mongo"
)

type NotificationRepo struct {
	col *mongo.Collection
}

func NewNotificationRepo(db *mongo.Database) *NotificationRepo {
	return &NotificationRepo{
		col: db.Collection("notifications"),
	}
}

func (r *NotificationRepo) Create(ctx context.Context, n *model.Notification) error {
	n.CreatedAt = time.Now()
	_, err := r.col.InsertOne(ctx, n)
	return err
}

func (r *NotificationRepo) GetUserNotifications(ctx context.Context, userID string) ([]model.Notification, error) {
	var notifs []model.Notification
	cursor, err := r.col.Find(ctx, map[string]interface{}{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &notifs); err != nil {
		return nil, err
	}
	return notifs, nil
}
