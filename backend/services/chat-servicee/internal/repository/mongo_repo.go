package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/fathima-sithara/chat-service/config"
	"github.com/fathima-sithara/chat-service/internal/models"
)

type MongoRepository struct {
	Client            *mongo.Client
	DB                *mongo.Database
	UserCollection    *mongo.Collection
	ChatCollection    *mongo.Collection
	MessageCollection *mongo.Collection
}

// NewMongoRepository initializes MongoDB connection and collections
func NewMongoRepository(cfg *config.Config) (*MongoRepository, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		return nil, err
	}

	db := client.Database(cfg.MongoDB)

	return &MongoRepository{
		Client:            client,
		DB:                db,
		UserCollection:    db.Collection("users"),
		ChatCollection:    db.Collection("chats"),
		MessageCollection: db.Collection("messages"),
	}, nil
}

// Disconnect closes the MongoDB connection
func (r *MongoRepository) Disconnect(ctx context.Context) error {
	return r.Client.Disconnect(ctx)
}

// Chat methods
func (r *MongoRepository) CreateChat(participantIDs []string, isGroup bool, groupName string) (string, error) {
	chat := models.Chat{
		ID:           primitive.NewObjectID().Hex(),
		Participants: participantIDs,
		IsGroup:      isGroup,
		GroupName:    groupName,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Deleted:      false,
	}

	_, err := r.ChatCollection.InsertOne(context.Background(), chat)
	return chat.ID, err
}

func (r *MongoRepository) GetUserChats(userID string) ([]models.Chat, error) {
	filter := bson.M{"participants": userID, "deleted": false}
	cursor, err := r.ChatCollection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}

	var chats []models.Chat
	if err := cursor.All(context.Background(), &chats); err != nil {
		return nil, err
	}

	return chats, nil
}

func (r *MongoRepository) GetChat(chatID string) (*models.Chat, error) {
	var chat models.Chat
	err := r.ChatCollection.FindOne(
		context.Background(),
		bson.M{"_id": chatID, "deleted": false},
	).Decode(&chat)

	if err != nil {
		return nil, err
	}
	return &chat, nil
}

func (r *MongoRepository) DeleteChat(chatID string) error {
	_, err := r.ChatCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": chatID},
		bson.M{"$set": bson.M{"deleted": true, "updated_at": time.Now()}},
	)
	return err
}

func (r *MongoRepository) SendMessage(msg models.Message) error {
	_, err := r.MessageCollection.InsertOne(context.Background(), msg)
	return err
}

func (r *MongoRepository) GetMessages(chatID string, page, limit int) ([]models.Message, error) {
	skip := int64((page - 1) * limit)
	limit64 := int64(limit)

	opts := options.Find().SetSkip(skip).SetLimit(limit64).SetSort(bson.M{"timestamp": 1})
	cursor, err := r.MessageCollection.Find(context.Background(), bson.M{"chat_id": chatID, "deleted": false}, opts)
	if err != nil {
		return nil, err
	}

	var messages []models.Message
	if err := cursor.All(context.Background(), &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *MongoRepository) EditMessage(messageID, content string) error {
	_, err := r.MessageCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": messageID},
		bson.M{
			"$set": bson.M{
				"content":   content,
				"edited":    true,
				"timestamp": time.Now(),
			},
		},
	)
	return err
}

func (r *MongoRepository) DeleteMessage(messageID string) error {
	_, err := r.MessageCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": messageID},
		bson.M{
			"$set": bson.M{
				"deleted":   true,
				"timestamp": time.Now(),
			},
		},
	)
	return err
}

func (r *MongoRepository) MarkMessagesRead(chatID string, messageIDs []string, userID string) error {
	filter := map[string]interface{}{
		"_id":     map[string]interface{}{"$in": messageIDs},
		"chat_id": chatID,
	}
	update := map[string]interface{}{
		"$addToSet": map[string]interface{}{
			"read_by": userID,
		},
	}

	_, err := r.MessageCollection.UpdateMany(context.Background(), filter, update)
	return err
}

// SetUserOnline marks a user as online
func (r *MongoRepository) SetUserOnline(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.UserCollection.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{
			"$set": bson.M{
				"is_online": true,
				"last_seen": time.Now(),
			},
		},
	)
	return err
}

// SetUserOffline marks a user as offline
func (r *MongoRepository) SetUserOffline(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.UserCollection.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{
			"$set": bson.M{
				"is_online": false,
				"last_seen": time.Now(),
			},
		},
	)
	return err
}

// GetOnlineUsers fetches all currently online users
func (r *MongoRepository) GetOnlineUsers() ([]models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.UserCollection.Find(ctx, bson.M{"is_online": true})
	if err != nil {
		return nil, err
	}

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// GetUser fetches a single user by ID
func (r *MongoRepository) GetUser(userID string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := r.UserCollection.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// SearchUsers searches users by username or email (case-insensitive)
func (r *MongoRepository) SearchUsers(query string) ([]models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"$or": []bson.M{
			{"username": bson.M{"$regex": query, "$options": "i"}},
			{"email": bson.M{"$regex": query, "$options": "i"}},
		},
	}

	cursor, err := r.UserCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// CreateUser inserts a new user into MongoDB
func (r *MongoRepository) CreateUser(username, email, phone string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userID := primitive.NewObjectID().Hex()
	user := models.User{
		ID:        userID,
		Username:  username,
		Email:     email,
		Phone:     phone,
		IsOnline:  false,
		LastSeen:  time.Now(),
		CreatedAt: time.Now(),
	}

	_, err := r.UserCollection.InsertOne(ctx, user)
	return userID, err
}
