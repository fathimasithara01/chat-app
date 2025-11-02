package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/fathima-sithara/auth-service/internal/brevo"
	"github.com/fathima-sithara/auth-service/internal/models"
	"github.com/fathima-sithara/auth-service/internal/repository"
	"github.com/fathima-sithara/auth-service/internal/twilio"
	"github.com/fathima-sithara/auth-service/internal/utils"
	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidOTP      = errors.New("invalid or expired otp")
	ErrTooManyRequests = errors.New("too many otp requests, try later")
)

type AuthService struct {
	userRepo     repository.UserRepository
	tw           *twilio.Client
	br           *brevo.Client
	redis        *redis.Client
	jm           *utils.JWTManager
	otpTTL       time.Duration
	otpRateLimit int
}

func NewAuthService(userRepo repository.UserRepository, tw *twilio.Client, br *brevo.Client, rdb *redis.Client, jwtSecret string, accessMins int, refreshDays int, otpTTLMin int, rateLimit int) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		tw:           tw,
		br:           br,
		redis:        rdb,
		jm:           utils.NewJWTManager(jwtSecret, accessMins, refreshDays),
		otpTTL:       time.Duration(otpTTLMin) * time.Minute,
		otpRateLimit: rateLimit,
	}
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	// Parse refresh token to extract userID
	userID, err := s.jm.ParseRefresh(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token")
	}

	// Generate new Access Token
	access, _, err := s.jm.GenerateAccess(userID)
	if err != nil {
		return "", "", err
	}

	// Generate new Refresh Token
	refresh, _, err := s.jm.GenerateRefresh(userID)
	if err != nil {
		return "", "", err
	}

	// Hash refresh token before saving in DB
	h := sha256.Sum256([]byte(refresh))
	rhash := hex.EncodeToString(h[:])
	if err := s.userRepo.SetRefreshTokenHash(ctx, userID, rhash); err != nil {
		return "", "", err
	}

	return access, refresh, nil
}

func (s *AuthService) genOTP() string {
	// simple 6-digit random (time-based) — good enough for OTP
	t := time.Now().UnixNano()
	v := (t % 1000000)
	return fmt.Sprintf("%06d", v)
}

func (s *AuthService) RequestOTP(ctx context.Context, phone string) error {
	rlKey := fmt.Sprintf("otp:rl:%s", phone)
	cnt, _ := s.redis.Get(ctx, rlKey).Int()
	if cnt >= s.otpRateLimit && s.otpRateLimit > 0 {
		return ErrTooManyRequests
	}

	otp := s.genOTP()
	otpKey := fmt.Sprintf("otp:%s", phone)
	if err := s.redis.Set(ctx, otpKey, otp, s.otpTTL).Err(); err != nil {
		return err
	}

	// rate-limit counter (1 hour)
	if err := s.redis.Incr(ctx, rlKey).Err(); err == nil {
		s.redis.Expire(ctx, rlKey, time.Hour)
	}

	// dispatch SMS via Twilio (noop if not configured)
	if s.tw != nil {
		body := fmt.Sprintf("Your verification code: %s", otp)
		_ = s.tw.SendSMS(ctx, phone, body)
	}
	return nil
}

func (s *AuthService) VerifyOTP(ctx context.Context, phone, otp string) (string, string, error) {
	otpKey := fmt.Sprintf("otp:%s", phone)
	v, err := s.redis.Get(ctx, otpKey).Result()
	if err != nil || v != otp {
		return "", "", ErrInvalidOTP
	}
	_ = s.redis.Del(ctx, otpKey)

	// find or create user
	u, err := s.userRepo.FindByPhone(ctx, phone)
	if err == repository.ErrUserNotFound {
		newU := &models.User{
			Phone:     phone,
			Verified:  true,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		if err := s.userRepo.Create(ctx, newU); err != nil {
			return "", "", err
		}
		u, _ = s.userRepo.FindByPhone(ctx, phone)
	} else if err != nil {
		return "", "", err
	}

	uid := u.ID.Hex()
	access, _, err := s.jm.GenerateAccess(uid)
	if err != nil {
		return "", "", err
	}
	refresh, _, err := s.jm.GenerateRefresh(uid)
	if err != nil {
		return "", "", err
	}

	// store hashed refresh
	h := sha256.Sum256([]byte(refresh))
	rhash := hex.EncodeToString(h[:])
	_ = s.userRepo.SetRefreshTokenHash(ctx, uid, rhash)
	return access, refresh, nil
}

func (s *AuthService) RegisterEmail(ctx context.Context, email string) error {
	otp := s.genOTP()
	key := "emailotp:" + email
	if err := s.redis.Set(ctx, key, otp, s.otpTTL).Err(); err != nil {
		return err
	}
	if s.br != nil && s.br.APIKey != "" {
		subj := "Your verification code"
		html := "<p>Your verification code is <b>" + otp + "</b></p>"
		_ = s.br.SendEmail(ctx, email, subj, html)
	}
	return nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, email, otp string) (string, string, error) {
	key := "emailotp:" + email
	v, err := s.redis.Get(ctx, key).Result()
	if err != nil || v != otp {
		return "", "", ErrInvalidOTP
	}
	_ = s.redis.Del(ctx, key)

	u, err := s.userRepo.FindByEmail(ctx, email)
	if err == repository.ErrUserNotFound {
		newU := &models.User{
			Email:     email,
			Verified:  true,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		if err := s.userRepo.Create(ctx, newU); err != nil {
			return "", "", err
		}
		u, _ = s.userRepo.FindByEmail(ctx, email)
	} else if err != nil {
		return "", "", err
	}

	uid := u.ID.Hex()
	access, _, err := s.jm.GenerateAccess(uid)
	if err != nil {
		return "", "", err
	}
	refresh, _, err := s.jm.GenerateRefresh(uid)
	if err != nil {
		return "", "", err
	}
	h := sha256.Sum256([]byte(refresh))
	_ = s.userRepo.SetRefreshTokenHash(ctx, uid, hex.EncodeToString(h[:]))
	return access, refresh, nil
}
