package repository

import (
	"context"
	"errors"
	"time"

	"github.com/fathima-sithara/auth-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	Create(ctx context.Context, u *models.User) error
	FindByPhone(ctx context.Context, phone string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, u *models.User) error
	SetRefreshTokenHash(ctx context.Context, id string, hash string) error
}

type mongoUserRepo struct {
	col *mongo.Collection
}

func NewMongoUserRepo(db *mongo.Database, collection string) UserRepository {
	col := db.Collection(collection)
	// indexes
	_, _ = col.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{Keys: bson.D{{Key: "phone", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
	})
	return &mongoUserRepo{col: col}
}

func (r *mongoUserRepo) Create(ctx context.Context, u *models.User) error {
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = time.Now().UTC()
	_, err := r.col.InsertOne(ctx, u)
	return err
}

func (r *mongoUserRepo) FindByPhone(ctx context.Context, phone string) (*models.User, error) {
	var u models.User
	err := r.col.FindOne(ctx, bson.M{"phone": phone}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, ErrUserNotFound
	}
	return &u, err
}

func (r *mongoUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := r.col.FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, ErrUserNotFound
	}
	return &u, err
}

func (r *mongoUserRepo) Update(ctx context.Context, u *models.User) error {
	u.UpdatedAt = time.Now().UTC()
	_, err := r.col.UpdateByID(ctx, u.ID, bson.M{"$set": u})
	return err
}

func (r *mongoUserRepo) SetRefreshTokenHash(ctx context.Context, id string, hash string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.col.UpdateByID(ctx, objID, bson.M{"$set": bson.M{"refresh_token_hash": hash}})
	return err
}
