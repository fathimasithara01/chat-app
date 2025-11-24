package repository

import (
	"context"
	"errors"
	"time"

	"github.com/fathima-sithara/message-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNotFound = errors.New("not found")

type Repository struct{ coll *mongo.Collection }

func NewMongoRepository(coll *mongo.Collection) *Repository {
	idx := mongo.IndexModel{
		Keys:    bson.D{{Key: "members", Value: 1}},
		Options: options.Index().SetBackground(true).SetName("members_idx"),
	}
	_, _ = coll.Indexes().CreateOne(context.Background(), idx)
	return &Repository{coll: coll}
}

func (r *Repository) CreateChat(ctx context.Context, chat *models.Chat) error {
	now := time.Now().UTC()
	chat.CreatedAt = now
	chat.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, chat)
	return err
}

func (r *Repository) GetChat(ctx context.Context, id string) (*models.Chat, error) {
	var c models.Chat
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&c); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repository) ListChatsForUser(ctx context.Context, userID string, limit int64) ([]*models.Chat, error) {
	filter := bson.M{"members": userID}
	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}}).SetLimit(limit)
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []*models.Chat
	for cur.Next(ctx) {
		var c models.Chat
		if err := cur.Decode(&c); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, nil
}

func (r *Repository) AddMember(ctx context.Context, chatID, userID string) error {
	update := bson.M{"$addToSet": bson.M{"members": userID}, "$set": bson.M{"updated_at": time.Now().UTC()}}
	_, err := r.coll.UpdateByID(ctx, chatID, update)
	return err
}

func (r *Repository) RemoveMember(ctx context.Context, chatID, userID string) error {
	update := bson.M{"$pull": bson.M{"members": userID}, "$set": bson.M{"updated_at": time.Now().UTC()}}
	_, err := r.coll.UpdateByID(ctx, chatID, update)
	return err
}

func (r *Repository) UpdateChat(ctx context.Context, chat *models.Chat) error {
	if chat == nil || chat.ID == "" {
		return errors.New("invalid chat")
	}
	chat.UpdatedAt = time.Now().UTC()
	_, err := r.coll.UpdateByID(ctx, chat.ID, bson.M{"$set": bson.M{
		"name":       chat.Name,
		"members":    chat.Members,
		"updated_at": chat.UpdatedAt,
	}})
	return err
}
