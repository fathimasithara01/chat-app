package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNotFound = errors.New("not found")

type Message struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	SenderID  string    `bson:"sender_id" json:"sender_id"`
	Content   string    `bson:"content" json:"content"`
	MsgType   string    `bson:"msg_type" json:"msg_type"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

type Chat struct {
	ID          string    `bson:"_id,omitempty" json:"id"`
	Name        string    `bson:"name,omitempty" json:"name"`
	IsGroup     bool      `bson:"is_group" json:"is_group"`
	Members     []string  `bson:"members" json:"members"` // user IDs only
	LastMessage *Message  `bson:"last_message,omitempty" json:"last_message,omitempty"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}

type Repository struct{ coll *mongo.Collection }

func NewMongoRepository(coll *mongo.Collection) *Repository {
	// index on members array
	idx := mongo.IndexModel{
		Keys:    bson.D{{Key: "members", Value: 1}},
		Options: options.Index().SetBackground(true).SetName("members_idx"),
	}
	_, _ = coll.Indexes().CreateOne(context.Background(), idx)
	return &Repository{coll: coll}
}

func (r *Repository) CreateChat(ctx context.Context, chat *Chat) error {
	now := time.Now().UTC()
	chat.CreatedAt = now
	chat.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, chat)
	return err
}

func (r *Repository) GetChat(ctx context.Context, id string) (*Chat, error) {
	var c Chat
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&c); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repository) ListChatsForUser(ctx context.Context, userID string, limit int64) ([]*Chat, error) {
	filter := bson.M{"members": userID}
	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}}).SetLimit(limit)
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []*Chat
	for cur.Next(ctx) {
		var c Chat
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

func (r *Repository) UpdateChat(ctx context.Context, chat *Chat) error {
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
