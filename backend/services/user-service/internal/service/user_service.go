package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	models "github.com/fathima-sithara/user-service/internal/model"
	"github.com/fathima-sithara/user-service/internal/repository"
	"go.uber.org/zap"
)

var ErrNotFound = errors.New("not found")
var ErrInvalidCredentials = errors.New("invalid credentials")

type UserService struct {
	repo       repository.UserRepository
	authSvcURL string
	log        *zap.Logger
	httpClient *http.Client
}

func NewUserService(repo repository.UserRepository, authSvcURL string, logger *zap.Logger) *UserService {
	if authSvcURL == "" {
		authSvcURL = "http://localhost:8081"
	}

	return &UserService{
		repo:       repo,
		authSvcURL: strings.TrimRight(authSvcURL, "/"),
		log:        logger,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *UserService) GetProfile(ctx context.Context, userID string) (*models.User, error) {
	return s.repo.GetByID(ctx, userID)
}

func (s *UserService) UpdateProfile(ctx context.Context, userID string, username, email, phone string) (*models.User, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if username != "" {
		u.Username = username
	}
	if email != "" {
		u.Email = email
	}
	if phone != "" {
		u.Phone = phone
	}

	return s.repo.Update(ctx, u)
}

func (s *UserService) ChangePassword(ctx context.Context, token, oldPass, newPass string) error {
	body, _ := json.Marshal(map[string]string{
		"old_password": newPass,
		"new_password": newPass,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", s.authSvcURL+"/api/v1/auth/change-password", strings.NewReader(string(body)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("auth-service rejected change-password")
	}
	return nil
}

func (s *UserService) GetByIDAdmin(ctx context.Context, id string) (*models.User, error) {
	return s.repo.GetByIDAdmin(ctx, id)
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	return s.repo.SoftDelete(ctx, id)
}
