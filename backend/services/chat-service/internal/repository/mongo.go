package repository

import (
	"context"
	"errors"
	"time"

	"github.com/fathima-sithara/message-service/internal/config"
	"github.com/fathima-sithara/message-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNotFound = errors.New("not found")

type MongoRepository struct {
	coll *mongo.Collection
}

func NewMongoClient(cfg *config.Config) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		return nil, err
	}
	return client, nil
}

func NewMongoRepository(coll *mongo.Collection) *MongoRepository {
	ix := mongo.IndexModel{
		Keys:    bson.D{{Key: "chat_id", Value: 1}, {Key: "created_at", Value: -1}},
		Options: options.Index().SetBackground(true).SetName("chat_created_idx"),
	}
	_, _ = coll.Indexes().CreateOne(context.Background(), ix)
	return &MongoRepository{coll: coll}
}

func (r *MongoRepository) SaveMessage(ctx context.Context, m *domain.Message) error {
	filter := bson.M{"_id": m.ID}
	update := bson.M{"$setOnInsert": m}
	opts := options.Update().SetUpsert(true)
	_, err := r.coll.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *MongoRepository) GetMessageByID(ctx context.Context, id string) (*domain.Message, error) {
	var m domain.Message
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&m); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (r *MongoRepository) GetMessages(ctx context.Context, chatID string, limit int64, before time.Time) ([]*domain.Message, error) {
	filter := bson.M{"chat_id": chatID}
	if !before.IsZero() {
		filter["created_at"] = bson.M{"$lt": before}
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit)
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := []*domain.Message{}
	for cur.Next(ctx) {
		var m domain.Message
		if err := cur.Decode(&m); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, nil
}

func (r *MongoRepository) GetLastMessage(ctx context.Context, chatID string) (*domain.Message, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})
	var m domain.Message
	if err := r.coll.FindOne(ctx, bson.M{"chat_id": chatID}, opts).Decode(&m); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// MarkRead updates read_by and returns chatID
func (r *MongoRepository) MarkRead(ctx context.Context, messageID, userID string) (string, error) {
	// get chat id
	var res domain.Message
	if err := r.coll.FindOne(ctx, bson.M{"_id": messageID}).Decode(&res); err != nil {
		return "", err
	}
	_, err := r.coll.UpdateByID(ctx, messageID, bson.M{"$addToSet": bson.M{"read_by": userID}})
	if err != nil {
		return "", err
	}
	return res.ChatID, nil
}

// EditMessage updates content and returns chatID
func (r *MongoRepository) EditMessage(ctx context.Context, messageID, encContent string, editedAt time.Time) (string, error) {
	// fetch chat id first
	var res domain.Message
	if err := r.coll.FindOne(ctx, bson.M{"_id": messageID}).Decode(&res); err != nil {
		return "", err
	}
	_, err := r.coll.UpdateByID(ctx, messageID, bson.M{"$set": bson.M{"content": encContent, "edited_at": editedAt}})
	if err != nil {
		return "", err
	}
	return res.ChatID, nil
}

func (r *MongoRepository) SoftDeleteMessage(ctx context.Context, messageID, userID string) error {
	_, err := r.coll.UpdateByID(ctx, messageID, bson.M{"$addToSet": bson.M{"deleted_for": userID}})
	return err
}

func (r *MongoRepository) DeleteMessageForAll(ctx context.Context, messageID string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": messageID})
	return err
}

func (r *MongoRepository) AddReaction(ctx context.Context, messageID, emoji, userID string) error {
	_, err := r.coll.UpdateByID(ctx, messageID, bson.M{"$addToSet": bson.M{"reactions." + emoji: userID}})
	return err
}

// GetChatIDByMessage: helper
func (r *MongoRepository) GetChatIDByMessage(ctx context.Context, messageID string) (string, error) {
	var res domain.Message
	if err := r.coll.FindOne(ctx, bson.M{"_id": messageID}).Decode(&res); err != nil {
		return "", err
	}
	return res.ChatID, nil
}
