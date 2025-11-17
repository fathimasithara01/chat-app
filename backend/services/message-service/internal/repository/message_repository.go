package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type Chat struct {
	ID        string    `bson:"_id" json:"id"`
	Members   []string  `bson:"members" json:"members"`
	IsGroup   bool      `bson:"is_group" json:"is_group"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

type MessageRepository struct {
	coll *mongo.Collection
}

func NewMessageRepository(coll *mongo.Collection) *MessageRepository {
	return &MessageRepository{coll: coll}
}

func (r *MessageRepository) InitChat(ctx context.Context, chatID string, members []string, isGroup bool) error {
	chat := &Chat{
		ID:        chatID,
		Members:   members,
		IsGroup:   isGroup,
		CreatedAt: time.Now().UTC(),
	}
	_, err := r.coll.InsertOne(ctx, chat)
	return err
}
