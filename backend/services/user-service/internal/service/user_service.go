package service

import (
	"user-service/internal/domain"
	"user-service/internal/repository"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo}
}

func (s *UserService) CreateUser(req domain.CreateUserRequest) (*domain.UserResponse, error) {
	user := &domain.User{
		Name:  req.Name,
		Email: req.Email,
		Phone: req.Phone,
	}
	result, err := s.repo.CreateUser(user)
	if err != nil {
		return nil, err
	}

	return &domain.UserResponse{
		ID:    result.ID,
		Name:  result.Name,
		Email: result.Email,
		Phone: result.Phone,
	}, nil
}
