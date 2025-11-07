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

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByIDAdmin(ctx context.Context, id string) (*models.User, error)
	Update(ctx context.Context, u *models.User) (*models.User, error)
	SoftDelete(ctx context.Context, id string) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
}

type mongoUserRepo struct {
	col *mongo.Collection
}

func NewMongoUserRepo(db *mongo.Database, collection string) UserRepository {
	col := db.Collection(collection)
	// create index on email unique
	_, _ = col.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	})
	return &mongoUserRepo{col: col}
}

func (r *mongoUserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	var u models.User
	err = r.col.FindOne(ctx, bson.M{"_id": objID, "deleted_at": bson.M{"$exists": false}}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// for admin can read soft-deleted optionally
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
	u.UpdatedAt = time.Now().UTC()
	updateData := bson.M{}
	if u.Username != "" {
		updateData["username"] = u.Username
	}
	if u.Email != "" {
		updateData["email"] = u.Email
	}
	if u.Phone != "" {
		updateData["phone"] = u.Phone
	}
	updateData["updated_at"] = u.UpdatedAt

	_, err := r.col.UpdateByID(ctx, u.ID, bson.M{"$set": updateData})
	if err != nil {
		return nil, err
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
