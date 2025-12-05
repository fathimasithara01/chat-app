package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	models "github.com/fathima-sithara/user-service/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrUserNotFound = errors.New("user not found")
var ErrDuplicateKey = errors.New("duplicate key error")

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByIDAdmin(ctx context.Context, id string) (*models.User, error)
	Update(ctx context.Context, u *models.User) (*models.User, error)
	SoftDelete(ctx context.Context, id string) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	UpdatePassword(ctx context.Context, u *models.User) error
}

type mongoUserRepo struct {
	col *mongo.Collection
}

func NewMongoUserRepo(db *mongo.Database, collection string) UserRepository {
	col := db.Collection(collection)
	_, _ = col.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	})
	return &mongoUserRepo{col: col}
}

func (r *mongoUserRepo) UpdatePassword(ctx context.Context, u *models.User) error {
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

func (r *mongoUserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	var u models.User
	err = r.col.FindOne(ctx, bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&u)

	if err == mongo.ErrNoDocuments {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *mongoUserRepo) GetByIDAdmin(ctx context.Context, id string) (*models.User, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	var u models.User
	err = r.col.FindOne(ctx, bson.M{"_id": objID}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *mongoUserRepo) Update(ctx context.Context, u *models.User) (*models.User, error) {
	if u.ID.IsZero() {
		return nil, errors.New("invalid user ID")
	}
	
	if u.Username != "" {
		count, err := r.col.CountDocuments(ctx, bson.M{
			"username": u.Username,
			"_id":      bson.M{"$ne": u.ID},
		})
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, fmt.Errorf("username already exists")
		}
	}

	if u.Email != "" {
		count, err := r.col.CountDocuments(ctx, bson.M{
			"email": u.Email,
			"_id":   bson.M{"$ne": u.ID},
		})
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, fmt.Errorf("email already exists")
		}
	}

	if u.Phone != "" {
		count, err := r.col.CountDocuments(ctx, bson.M{
			"phone": u.Phone,
			"_id":   bson.M{"$ne": u.ID},
		})
		if err != nil {
			return nil, err
		}
		if count > 0 {	
			return nil, fmt.Errorf("phone already exists")
		}
	}

	u.UpdatedAt = time.Now().UTC()

	updateData := bson.M{
		"updated_at": u.UpdatedAt,
		"username":   u.Username,
		"email":      u.Email,
		"phone":      u.Phone,
	}

	_, err := r.col.UpdateByID(ctx, u.ID, bson.M{"$set": updateData})
	if err != nil {
		return nil, fmt.Errorf("update failed: %w", err)
	}

	return r.GetByID(ctx, u.ID.Hex())
}

func (r *mongoUserRepo) SoftDelete(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	now := time.Now().UTC()
	_, err = r.col.UpdateByID(ctx, objID, bson.M{"$set": bson.M{"deleted_at": now}})
	return err
}

func (r *mongoUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := r.col.FindOne(ctx, bson.M{"email": email, "deleted_at": bson.M{"$exists": false}}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
