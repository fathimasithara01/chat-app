package repository

import "user-service/internal/domain"

type UserRepository interface {
	CreateUser(user *domain.User) (*domain.User, error)
	GetUserByID(id string) (*domain.User, error)
}
