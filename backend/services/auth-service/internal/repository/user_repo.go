package repository

import (
	"context"
	"time"

	"github.com/fathima-sithara/chat-app/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepository interface {
	Create(ctx context.Context, u *models.User) error
	FindByPhone(ctx context.Context, phone string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByUUID(ctx context.Context, uuid string) (*models.User, error)
	Update(ctx context.Context, u *models.User) error
}

type mongoUserRepo struct {
	col *mongo.Collection
}

func NewMongoUserRepo(db *mongo.Database) UserRepository {
	return &mongoUserRepo{
		col: db.Collection("users"),
	}
}

func (r *mongoUserRepo) Create(ctx context.Context, u *models.User) error {
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = u.CreatedAt
	if u.UUID == "" {
		// caller should set UUID
	}
	_, err := r.col.InsertOne(ctx, u)
	return err
}

func (r *mongoUserRepo) FindByPhone(ctx context.Context, phone string) (*models.User, error) {
	var u models.User
	err := r.col.FindOne(ctx, bson.M{"phone": phone}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *mongoUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := r.col.FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *mongoUserRepo) FindByUUID(ctx context.Context, uuid string) (*models.User, error) {
	var u models.User
	err := r.col.FindOne(ctx, bson.M{"uuid": uuid}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *mongoUserRepo) Update(ctx context.Context, u *models.User) error {
	u.UpdatedAt = time.Now().UTC()
	filter := bson.M{"uuid": u.UUID}
	update := bson.M{"$set": u}
	_, err := r.col.UpdateOne(ctx, filter, update, options.Update().SetUpsert(false))
	return err
}
