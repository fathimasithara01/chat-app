package repository

import (
	"context"
	"time"

	"github.com/fathima-sithara/chat-service/config"
	"github.com/fathima-sithara/chat-service/internal/models"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepository struct {
	client *mongo.Client
	db     *mongo.Database
}

func NewMongoRepository(cfg *config.Config) *MongoRepository {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal().Err(err).Msg("mongo connect")
	}
	return &MongoRepository{client: client, db: client.Database(cfg.MongoDB)}
}

func (r *MongoRepository) Disconnect(ctx context.Context) error {
	return r.client.Disconnect(ctx)
}

func (r *MongoRepository) SaveMessage(ctx context.Context, m *models.Message) error {
	m.CreatedAt = time.Now().UTC()
	coll := r.db.Collection("messages")
	_, err := coll.InsertOne(ctx, m)
	return err
}

func (r *MongoRepository) GetMessages(ctx context.Context, convID string, limit int64, skip int64) ([]*models.Message, error) {
	coll := r.db.Collection("messages")
	objID, _ := primitive.ObjectIDFromHex(convID)
	cur, err := coll.Find(ctx, bson.M{"conversation_id": objID}, &options.FindOptions{Limit: &limit, Skip: &skip})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var res []*models.Message
	for cur.Next(ctx) {
		var m models.Message
		if err := cur.Decode(&m); err != nil {
			continue
		}
		res = append(res, &m)
	}
	return res, nil
}

func (r *MongoRepository) CreateConversation(ctx context.Context, members []string) (*models.Conversation, error) {
	c := &models.Conversation{Members: members, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	coll := r.db.Collection("conversations")
	res, err := coll.InsertOne(ctx, c)
	if err != nil {
		return nil, err
	}
	c.ID = res.InsertedID.(primitive.ObjectID)
	return c, nil
}
