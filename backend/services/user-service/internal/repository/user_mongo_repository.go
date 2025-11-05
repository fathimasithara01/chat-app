package repository

import (
	"context"
	"errors"

	"githhub.com/fathimasithara/user-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) (string, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, id string, update interface{}) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int64) ([]*domain.User, error)
}

type MongoUserRepo struct {
	collection *mongo.Collection
}

func NewMongoUserRepo(db *mongo.Database, collectionName string) *MongoUserRepo {
	return &MongoUserRepo{
		collection: db.Collection(collectionName),
	}
}

func (r *MongoUserRepo) Create(ctx context.Context, user *domain.User) (string, error) {
	res, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		return "", err
	}
	return res.InsertedID.(string), nil
}

func (r *MongoUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func (r *MongoUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func (r *MongoUserRepo) Update(ctx context.Context, id string, update interface{}) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	return err
}

func (r *MongoUserRepo) Delete(ctx context.Context, id string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *MongoUserRepo) List(ctx context.Context, limit, offset int64) ([]*domain.User, error) {
	opts := options.Find().SetLimit(limit).SetSkip(offset)
	cur, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var users []*domain.User
	for cur.Next(ctx) {
		var u domain.User
		if err := cur.Decode(&u); err != nil {
			continue
		}
		users = append(users, &u)
	}
	return users, nil
}
