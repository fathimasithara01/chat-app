package service

import (
	"context"
	"errors"
	"time"

	"githhub.com/fathimasithara/user-service/internal/domain"
	"githhub.com/fathimasithara/user-service/internal/repository"
	"githhub.com/fathimasithara/user-service/internal/utils"
	"go.uber.org/zap"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrValidation         = errors.New("validation error")
)

type UserService struct {
	repo       repository.UserRepository
	logger     *zap.Logger
	jwtManager *utils.JWTManager
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewUserService(repo repository.UserRepository, logger *zap.Logger, jwtManager *utils.JWTManager, accessTTLMinutes int, refreshTTLDays int) *UserService {
	return &UserService{
		repo:       repo,
		logger:     logger,
		jwtManager: jwtManager,
		accessTTL:  time.Duration(accessTTLMinutes) * time.Minute,
		refreshTTL: time.Duration(refreshTTLDays) * 24 * time.Hour,
	}
}

func (s *UserService) RegisterUser(ctx context.Context, req *domain.UserRegisterRequest) (*domain.User, error) {
	if req.Name == "" || req.Email == "" || req.Password == "" {
		return nil, ErrValidation
	}

	_, err := s.repo.GetByEmail(ctx, req.Email)
	if err == nil {
		return nil, ErrUserAlreadyExists
	}

	hash, _ := utils.HashPassword(req.Password)

	user := &domain.User{
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		Password: hash,
		Role:     domain.RoleUser,
		Verified: false,
	}

	id, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, err
	}
	user.ID = id
	user.Password = ""
	return user, nil
}

func (s *UserService) LoginUser(ctx context.Context, email, password string) (*domain.AuthTokens, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !utils.CheckPasswordHash(password, user.Password) {
		return nil, ErrInvalidCredentials
	}

	return s.generateTokens(user.ID, user.Role)
}

func (s *UserService) generateTokens(userID string, role domain.UserRole) (*domain.AuthTokens, error) {
	access, err := s.jwtManager.Generate(userID, role, s.accessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := s.jwtManager.Generate(userID, role, s.refreshTTL)
	if err != nil {
		return nil, err
	}
	return &domain.AuthTokens{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *UserService) RefreshAccessToken(refreshToken string) (string, error) {
	claims, err := s.jwtManager.GetClaims(refreshToken)
	if err != nil {
		return "", ErrInvalidToken
	}
	return s.jwtManager.Generate(claims.UserID, claims.Role, s.accessTTL)
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, repository.ErrUserNotFound
	}
	user.Password = "" 
	return user, nil
}

// UpdateUser updates a user's information
func (s *UserService) UpdateUser(ctx context.Context, id string, req *domain.UserUpdateRequest) error {
	update := make(map[string]interface{})
	if req.Name != nil {
		update["name"] = *req.Name
	}
	if req.Email != nil {
		update["email"] = *req.Email
	}
	if req.Phone != nil {
		update["phone"] = *req.Phone
	}
	if len(update) == 0 {
		return ErrValidation
	}
	return s.repo.Update(ctx, id, update)
}

// DeleteUser deletes a user by ID
func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// ListUsers returns paginated users
func (s *UserService) ListUsers(ctx context.Context, limit, offset int64) ([]*domain.User, error) {
	users, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		u.Password = ""
	}
	return users, nil
}
