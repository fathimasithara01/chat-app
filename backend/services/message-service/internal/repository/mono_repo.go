package repository

import (
	"context"
	"errors"
	"time"

	"github.com/fathima-sithara/message-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNotFound = errors.New("not found")

type MongoRepository struct {
	db      *mongo.Database
	msgColl *mongo.Collection
	chatCol *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	r := &MongoRepository{
		db:      db,
		msgColl: db.Collection("messages"),
		chatCol: db.Collection("chats"),
	}
	_, _ = r.msgColl.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "chat_id", Value: 1}, {Key: "created_at", Value: -1}},
		Options: options.Index().SetBackground(true),
	})
	return r
}

func (r *MongoRepository) InitChat(ctx context.Context, chatID string, members []string, isGroup bool) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	doc := bson.M{"_id": chatID, "members": members, "is_group": isGroup, "created_at": time.Now().UTC()}
	_, err := r.chatCol.UpdateByID(ctx, chatID, bson.M{"$setOnInsert": doc}, options.Update().SetUpsert(true))
	return err
}

func (r *MongoRepository) SaveMessage(ctx context.Context, m *domain.Message) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if m.ReadBy == nil {
		m.ReadBy = []string{}
	}
	if m.DeletedFor == nil {
		m.DeletedFor = []string{}
	}
	if m.Reactions == nil {
		m.Reactions = map[string][]string{}
	}

	filter := bson.M{"_id": m.ID}
	update := bson.M{"$setOnInsert": m}
	_, err := r.msgColl.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func (r *MongoRepository) GetMessages(ctx context.Context, chatID string, limit int64, before time.Time) ([]*domain.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"chat_id": chatID}
	if !before.IsZero() {
		filter["created_at"] = bson.M{"$lt": before}
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit)
	cur, err := r.msgColl.Find(ctx, filter, opts)
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
		if m.ReadBy == nil {
			m.ReadBy = []string{}
		}
		if m.DeletedFor == nil {
			m.DeletedFor = []string{}
		}
		if m.Reactions == nil {
			m.Reactions = map[string][]string{}
		}
		out = append(out, &m)
	}
	return out, nil
}

func (r *MongoRepository) GetMessageByID(ctx context.Context, messageID string) (*domain.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var m domain.Message
	if err := r.msgColl.FindOne(ctx, bson.M{"_id": messageID}).Decode(&m); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if m.ReadBy == nil {
		m.ReadBy = []string{}
	}
	if m.DeletedFor == nil {
		m.DeletedFor = []string{}
	}
	if m.Reactions == nil {
		m.Reactions = map[string][]string{}
	}
	return &m, nil
}

func (r *MongoRepository) EditMessage(ctx context.Context, messageID, newContent string, now time.Time) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	res := r.msgColl.FindOneAndUpdate(
		ctx,
		bson.M{"_id": messageID},
		bson.M{"$set": bson.M{"content": newContent, "edited_at": now}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	var m domain.Message
	if err := res.Decode(&m); err != nil {
		return "", err
	}
	return m.ChatID, nil
}

func (r *MongoRepository) SoftDeleteMessage(ctx context.Context, messageID, userID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, _ = r.msgColl.UpdateOne(
		ctx,
		bson.M{
			"_id": messageID,
			"$or": []bson.M{
				{"deleted_for": bson.M{"$exists": false}},
				{"deleted_for": nil},
				{"deleted_for": bson.M{"$not": bson.M{"$type": "array"}}},
			},
		},
		bson.M{
			"$set": bson.M{"deleted_for": []string{}},
		},
	)

	res := r.msgColl.FindOneAndUpdate(
		ctx,
		bson.M{"_id": messageID},
		bson.M{"$addToSet": bson.M{"deleted_for": userID}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var m domain.Message
	if err := res.Decode(&m); err != nil {
		return "", err
	}

	return m.ChatID, nil
}

func (r *MongoRepository) DeleteMessageForAll(ctx context.Context, messageID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var m domain.Message
	if err := r.msgColl.FindOneAndDelete(ctx, bson.M{"_id": messageID}).Decode(&m); err != nil {
		return "", err
	}
	return m.ChatID, nil
}

func (r *MongoRepository) MarkRead(ctx context.Context, messageID, userID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, _ = r.msgColl.UpdateOne(ctx,
		bson.M{
			"_id": messageID,
			"$or": []bson.M{
				{"read_by": bson.M{"$exists": false}},
				{"read_by": nil},
				{"read_by": bson.M{"$not": bson.M{"$type": "array"}}},
			},
		},
		bson.M{
			"$set": bson.M{"read_by": []string{}},
		},
	)

	res := r.msgColl.FindOneAndUpdate(
		ctx,
		bson.M{"_id": messageID},
		bson.M{"$addToSet": bson.M{"read_by": userID}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var m domain.Message
	if err := res.Decode(&m); err != nil {
		return "", err
	}

	return m.ChatID, nil
}

func (r *MongoRepository) AddReaction(ctx context.Context, messageID, emoji, userID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	update := bson.M{
		"$setOnInsert": bson.M{"reactions": bson.M{}},
		"$addToSet":    bson.M{"reactions." + emoji: userID},
	}
	_, err := r.msgColl.UpdateOne(ctx, bson.M{"_id": messageID}, update, options.Update().SetUpsert(true))
	if err != nil {
		return "", err
	}

	var m domain.Message
	if err := r.msgColl.FindOne(ctx, bson.M{"_id": messageID}).Decode(&m); err != nil {
		return "", err
	}
	if m.Reactions == nil {
		m.Reactions = map[string][]string{}
	}
	return m.ChatID, nil
}

func (r *MongoRepository) GetLastMessage(ctx context.Context, chatID string) (*domain.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})
	var m domain.Message
	if err := r.msgColl.FindOne(ctx, bson.M{"chat_id": chatID}, opts).Decode(&m); err != nil {
		return nil, err
	}
	if m.ReadBy == nil {
		m.ReadBy = []string{}
	}
	if m.DeletedFor == nil {
		m.DeletedFor = []string{}
	}
	if m.Reactions == nil {
		m.Reactions = map[string][]string{}
	}
	return &m, nil
}
