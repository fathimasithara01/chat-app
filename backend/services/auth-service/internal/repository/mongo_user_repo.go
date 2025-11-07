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

// ErrUserNotFound is returned when a user with the given criteria is not found.
var ErrUserNotFound = errors.New("user not found")

// ErrDuplicateKey is returned when a unique constraint is violated during user creation or update.
var ErrDuplicateKey = errors.New("duplicate key error")

// UserRepository defines the interface for user data operations.
type UserRepository interface {
	Create(ctx context.Context, u *models.User) error
	FindByPhone(ctx context.Context, phone string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	Update(ctx context.Context, u *models.User) error
	SetRefreshTokenHash(ctx context.Context, id string, hash string) error
	FindByUsername(ctx context.Context, username string) (*models.User, error)
}

// mongoUserRepo implements UserRepository for MongoDB.
type mongoUserRepo struct {
	col *mongo.Collection
}

// NewMongoUserRepo creates a new UserRepository instance backed by MongoDB.
// It also ensures necessary indexes are created.
func NewMongoUserRepo(db *mongo.Database, collection string) UserRepository {
	col := db.Collection(collection)
	// Ensure indexes for unique fields (phone, email, username)
	// SetSparse(true) allows documents without these fields to be inserted without violating unique constraint.
	// This is important because a user might register only with phone OR email initially.
	_, err := col.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{Keys: bson.D{{Key: "phone", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
		{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)}, // Added index for username
	})
	if err != nil {
		fmt.Printf("Warning: Failed to create MongoDB indexes: %v\n", err)
	}
	return &mongoUserRepo{col: col}
}

// Create inserts a new user into the database.
func (r *mongoUserRepo) Create(ctx context.Context, u *models.User) error {
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = time.Now().UTC()
	result, err := r.col.InsertOne(ctx, u)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%w: %v", ErrDuplicateKey, err)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		u.ID = oid
	}
	return nil
}

// FindByPhone retrieves a user by their phone number.
func (r *mongoUserRepo) FindByPhone(ctx context.Context, phone string) (*models.User, error) {
	var u models.User
	err := r.col.FindOne(ctx, bson.M{"phone": phone}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by phone: %w", err)
	}
	return &u, nil
}

// FindByEmail retrieves a user by their email address.
func (r *mongoUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := r.col.FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return &u, nil
}

// FindByID retrieves a user by their MongoDB ObjectID string.
func (r *mongoUserRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	var u models.User
	err = r.col.FindOne(ctx, bson.M{"_id": objID}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}
	return &u, nil
}

// FindByUsername retrieves a user by their username.
func (r *mongoUserRepo) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var u models.User
	err := r.col.FindOne(ctx, bson.M{"username": username}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by username: %w", err)
	}
	return &u, nil
}

// Update updates an existing user's document in the database.
func (r *mongoUserRepo) Update(ctx context.Context, u *models.User) error {
	u.UpdatedAt = time.Now().UTC()
	result, err := r.col.UpdateByID(ctx, u.ID, bson.M{"$set": u})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%w: %v", ErrDuplicateKey, err)
		}
		return fmt.Errorf("failed to update user: %w", err)
	}
	if result.MatchedCount == 0 {
		return ErrUserNotFound
	}
	return nil
}

// SetRefreshTokenHash updates only the refresh token hash for a given user ID.
func (r *mongoUserRepo) SetRefreshTokenHash(ctx context.Context, id string, hash string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}
	result, err := r.col.UpdateByID(ctx, objID, bson.M{"$set": bson.M{"refresh_token_hash": hash, "updated_at": time.Now().UTC()}})
	if err != nil {
		return fmt.Errorf("failed to set refresh token hash for user %s: %w", id, err)
	}
	if result.MatchedCount == 0 {
		return ErrUserNotFound
	}
	return nil
}
