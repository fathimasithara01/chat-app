package services

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"time"

	"github.com/fathima-sithara/chat-app/internal/brevo"
	"github.com/fathima-sithara/chat-app/internal/models"
	"github.com/fathima-sithara/chat-app/internal/repository"
	"github.com/fathima-sithara/chat-app/internal/twilio"
	"github.com/fathima-sithara/chat-app/internal/utils"
	"github.com/google/uuid"

	"github.com/redis/go-redis/v9"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidCreds = errors.New("invalid credentials")
)

type AuthService struct {
	repo             repository.UserRepository
	twClient         *twilio.Client
	brClient         *brevo.Client
	redis            *redis.Client
	jwtSecret        string
	accessTTL        int
	refreshTTLDays   int
	otpTTLMinutes    int
	otpRateLimitHour int
}

func NewAuthService(repo repository.UserRepository, tw *twilio.Client, br *brevo.Client, rdb *redis.Client, jwtSecret string, accessTTL int, refreshTTLDays int, otpTTL int, otpRateLimit int) *AuthService {
	return &AuthService{
		repo:             repo,
		twClient:         tw,
		brClient:         br,
		redis:            rdb,
		jwtSecret:        jwtSecret,
		accessTTL:        accessTTL,
		refreshTTLDays:   refreshTTLDays,
		otpTTLMinutes:    otpTTL,
		otpRateLimitHour: otpRateLimit,
	}
}

// RegisterByPhone creates or returns existing user and sends OTP
func (s *AuthService) RegisterOrRequestOTP(ctx context.Context, phone string) error {
	// rate limit OTP per phone per hour via Redis
	rlKey := fmt.Sprintf("otp_rl:%s", phone)
	cnt, _ := s.redis.Get(ctx, rlKey).Int()
	if cnt >= s.otpRateLimitHour {
		return fmt.Errorf("otp rate limit exceeded")
	}
	if err := s.redis.Incr(ctx, rlKey).Err(); err == nil {
		s.redis.Expire(ctx, rlKey, time.Hour)
	}

	// generate OTP (6-digit)
	otp := s.generateOTP()
	otpKey := fmt.Sprintf("otp:%s", phone)
	if err := s.redis.Set(ctx, otpKey, otp, time.Duration(s.otpTTLMinutes)*time.Minute).Err(); err != nil {
		return err
	}

	// send SMS via Twilio
	body := fmt.Sprintf("Your Chat App OTP is %s. It will expire in %d minutes.", otp, s.otpTTLMinutes)
	return s.twClient.SendSMS(ctx, phone, body)
}

func (s *AuthService) generateOTP() string {
	// 6 digit numeric
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	// base32 then digits
	enc := base32.StdEncoding.EncodeToString(b)
	// extract digits
	digits := ""
	for i := 0; i < len(enc) && len(digits) < 6; i++ {
		c := enc[i]
		if c >= '0' && c <= '9' {
			digits += string(c)
		} else if c >= 'A' && c <= 'Z' {
			// map letter to digit
			digits += fmt.Sprintf("%d", int(c)%10)
		}
	}
	for len(digits) < 6 {
		digits += "0"
	}
	return digits[:6]
}

func (s *AuthService) VerifyOTPAndLogin(ctx context.Context, phone, otp string) (string /*access token*/, string /*refresh token*/, *models.User, error) {
	otpKey := fmt.Sprintf("otp:%s", phone)
	stored, err := s.redis.Get(ctx, otpKey).Result()
	if err != nil {
		return "", "", nil, fmt.Errorf("otp expired or not found")
	}
	if stored != otp {
		return "", "", nil, fmt.Errorf("invalid otp")
	}

	// find user or create
	u, err := s.repo.FindByPhone(ctx, phone)
	if err != nil {
		return "", "", nil, err
	}
	if u == nil {
		u = &models.User{
			UUID:            uuid.New().String(),
			Phone:           phone,
			IsPhoneVerified: true,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}
		if err := s.repo.Create(ctx, u); err != nil {
			return "", "", nil, err
		}
	} else {
		// mark phone verified if not
		if !u.IsPhoneVerified {
			u.IsPhoneVerified = true
			_ = s.repo.Update(ctx, u)
		}
	}

	// generate tokens
	access, err := utils.GenerateAccessToken(s.jwtSecret, u.UUID, s.accessTTL)
	if err != nil {
		return "", "", nil, err
	}
	refresh, err := s.createAndStoreRefreshToken(ctx, u.UUID)
	if err != nil {
		return "", "", nil, err
	}

	// delete OTP so it can't be reused
	_ = s.redis.Del(ctx, otpKey).Err()

	return access, refresh, u, nil
}

func (s *AuthService) createAndStoreRefreshToken(ctx context.Context, userUUID string) (string, error) {
	// create strong refresh token - random UUID + expiry stored hashed in Redis
	rt := uuid.New().String()
	key := fmt.Sprintf("refresh:%s:%s", userUUID, rt)
	// store as key with TTL equals refreshTTLDays
	if err := s.redis.Set(ctx, key, "1", time.Duration(s.refreshTTLDays)*24*time.Hour).Err(); err != nil {
		return "", err
	}
	return rt, nil
}

func (s *AuthService) RefreshAccessToken(ctx context.Context, userUUID, refreshToken string) (string, string, error) {
	key := fmt.Sprintf("refresh:%s:%s", userUUID, refreshToken)
	ok, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return "", "", err
	}
	if ok == 0 {
		return "", "", fmt.Errorf("invalid refresh token")
	}
	// optionally rotate: delete old and create new
	_ = s.redis.Del(ctx, key).Err()
	newRefresh, err := s.createAndStoreRefreshToken(ctx, userUUID)
	if err != nil {
		return "", "", err
	}
	access, err := utils.GenerateAccessToken(s.jwtSecret, userUUID, s.accessTTL)
	if err != nil {
		return "", "", err
	}
	return access, newRefresh, nil
}

// RegisterWithEmail registers user (email/password) and sends verification email via Brevo.
func (s *AuthService) RegisterWithEmail(ctx context.Context, email, password string) (*models.User, error) {
	existing, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("email already registered")
	}
	hashed, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &models.User{
		UUID:            uuid.New().String(),
		Email:           email,
		PasswordHash:    hashed,
		IsEmailVerified: false,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	// create email verification token (short-lived)
	vToken := uuid.New().String()
	vKey := fmt.Sprintf("email_verify:%s", vToken)
	if err := s.redis.Set(ctx, vKey, u.UUID, time.Hour*24).Err(); err != nil {
		return nil, err
	}

	// send email via Brevo
	verifyURL := fmt.Sprintf("https://your-frontend.example.com/verify-email?token=%s", vToken)
	html := fmt.Sprintf("<p>Welcome! Please verify your email: <a href=\"%s\">Verify Email</a></p>", verifyURL)
	if err := s.brClient.SendEmail(ctx, email, "Verify your email", html, "Verify your email"); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *AuthService) VerifyEmailToken(ctx context.Context, token string) (*models.User, error) {
	key := fmt.Sprintf("email_verify:%s", token)
	uuidVal, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("invalid or expired token")
	}
	u, err := s.repo.FindByUUID(ctx, uuidVal)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}
	u.IsEmailVerified = true
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	_ = s.redis.Del(ctx, key).Err()
	return u, nil
}
