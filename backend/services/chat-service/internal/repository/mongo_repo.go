package repository

import (
	"context"
	"time"

	"github.com/yourorg/chat-app/services/chat-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Repository interface {
	InsertMessage(ctx context.Context, m *models.Message) (*models.Message, error)
	GetMessages(ctx context.Context, convID string, limit int64, beforeTime time.Time) ([]*models.Message, error)
	CreateConversationIfNotExists(ctx context.Context, c *models.Conversation) (*models.Conversation, error)
}

type mongoRepo struct {
	msgCol  *mongo.Collection
	convCol *mongo.Collection
}

func NewMongoRepo(msgCol, convCol *mongo.Collection) Repository {
	return &mongoRepo{msgCol: msgCol, convCol: convCol}
}

func (r *mongoRepo) InsertMessage(ctx context.Context, m *models.Message) (*models.Message, error) {
	m.CreatedAt = time.Now().UTC()
	res, err := r.msgCol.InsertOne(ctx, m)
	if err != nil {
		return nil, err
	}
	oid := res.InsertedID.(primitive.ObjectID)
	m.ID = oid.Hex()
	// update conversation updated_at - optional
	_, _ = r.convCol.UpdateOne(ctx, bson.M{"_id": primitive.ObjectIDHex(m.ConversationID)}, bson.M{"$set": bson.M{"updated_at": time.Now().UTC()}})
	return m, nil
}

func (r *mongoRepo) GetMessages(ctx context.Context, convID string, limit int64, beforeTime time.Time) ([]*models.Message, error) {
	filter := bson.M{"conversation_id": convID}
	if !beforeTime.IsZero() {
		filter["created_at"] = bson.M{"$lt": beforeTime}
	}
	cur, err := r.msgCol.Find(ctx, filter, &mongo.FindOptions{
		Sort:  bson.D{{Key: "created_at", Value: -1}},
		Limit: &limit,
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*models.Message
	for cur.Next(ctx) {
		var m models.Message
		if err := cur.Decode(&m); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	// return in chronological order
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (r *mongoRepo) CreateConversationIfNotExists(ctx context.Context, c *models.Conversation) (*models.Conversation, error) {
	// naive create-if-not-exists (add proper unique keys/index in production)
	c.CreatedAt = time.Now().UTC()
	res, err := r.convCol.InsertOne(ctx, c)
	if err != nil {
		// if duplicate key, try to find it
		if writeErr, ok := err.(mongo.WriteException); ok && len(writeErr.WriteErrors) > 0 {
			// fall through to find
		} else {
			return nil, err
		}
	}
	if res != nil && res.InsertedID != nil {
		c.ID = res.InsertedID.(primitive.ObjectID).Hex()
		return c, nil
	}
	// if insertion failed due to duplicate, attempt find by members
	var found models.Conversation
	err = r.convCol.FindOne(ctx, bson.M{"members": c.Members}).Decode(&found)
	if err == nil {
		return &found, nil
	}
	return c, nil
}
