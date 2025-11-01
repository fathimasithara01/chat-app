package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fathima-sithara/auth-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoUserRepo implements UserRepository for MongoDB
type MongoUserRepo struct {
	collection *mongo.Collection
}

// NewMongoUserRepo creates a new MongoUserRepo
func NewMongoUserRepo(db *mongo.Database) UserRepository {
	return &MongoUserRepo{
		collection: db.Collection("users"),
	}
}

// CreateUser inserts a new user into the database
func (r *MongoUserRepo) CreateUser(ctx context.Context, user *models.User) error {
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		// Check for duplicate key error (e.g., unique email constraint)
		var writeException mongo.WriteException
		if errors.As(err, &writeException) {
			for _, we := range writeException.WriteErrors {
				// MongoDB duplicate key error code
				if we.Code == 11000 {
					return errors.New("user with this email or phone number already exists")
				}
			}
		}
		return err
	}
	return nil
}

// FindUserByID retrieves a user by their ID
func (r *MongoUserRepo) FindUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// FindUserByEmail retrieves a user by their email
func (r *MongoUserRepo) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// FindUserByPhoneNumber retrieves a user by their phone number
func (r *MongoUserRepo) FindUserByPhoneNumber(ctx context.Context, phoneNumber string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"phone_number": phoneNumber}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates an existing user in the database
func (r *MongoUserRepo) UpdateUser(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()
	filter := bson.M{"_id": user.ID}
	update := bson.M{"$set": user}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	res := r.collection.FindOneAndUpdate(ctx, filter, update, opts)
	if res.Err() != nil {
		if errors.Is(res.Err(), mongo.ErrNoDocuments) {
			return errors.New("user not found for update")
		}
		return res.Err()
	}

	return res.Decode(user) // Decode updated document back into user object
}

// DeleteUser deletes a user by their ID
func (r *MongoUserRepo) DeleteUser(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("user not found for deletion")
	}
	return nil
}

// EnsureIndexes creates necessary indexes (call this once during app startup if needed)
func (r *MongoUserRepo) EnsureIndexes(ctx context.Context) error {
	// Unique index on email
	emailIndexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true), // Sparse for optional email
	}
	_, err := r.collection.Indexes().CreateOne(ctx, emailIndexModel)
	if err != nil {
		return fmt.Errorf("failed to create email index: %w", err)
	}

	// Unique index on phone_number
	phoneIndexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "phone_number", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true), // Sparse for optional phone number
	}
	_, err = r.collection.Indexes().CreateOne(ctx, phoneIndexModel)
	if err != nil {
		return fmt.Errorf("failed to create phone number index: %w", err)
	}
	return nil
}
