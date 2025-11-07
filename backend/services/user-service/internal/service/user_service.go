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

type UserService struct {
	repo       repository.UserRepository
	authSvcURL string // for change-password proxy
	log        *zap.Logger
	httpClient *http.Client
}

func NewUserService(repo repository.UserRepository, authSvcURL string, logger *zap.Logger) *UserService {
	if authSvcURL == "" {
		// try env fallback
		authSvcURL = "http://localhost:8080"
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

// ChangePassword proxies the request to auth-service's change-password endpoint.
// It requires the access token to be sent along (the caller should set Authorization header).
func (s *UserService) ChangePassword(ctx context.Context, authHeader, oldPassword, newPassword string) error {
	if authHeader == "" {
		return fmt.Errorf("authorization header required")
	}
	payload := map[string]string{
		"old_password": oldPassword,
		"new_password": newPassword,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.authSvcURL+"/api/v1/auth/change-password", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("auth-service returned status %d", resp.StatusCode)
	}
	return nil
}

func (s *UserService) GetByIDAdmin(ctx context.Context, id string) (*models.User, error) {
	return s.repo.GetByIDAdmin(ctx, id)
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	return s.repo.SoftDelete(ctx, id)
}
