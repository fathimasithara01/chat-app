package repository

import (
	"context"

	"github.com/fathima-sithara/auth-service/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	FindUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error)
	FindUserByEmail(ctx context.Context, email string) (*models.User, error)
	FindUserByPhoneNumber(ctx context.Context, phoneNumber string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id primitive.ObjectID) error
}
