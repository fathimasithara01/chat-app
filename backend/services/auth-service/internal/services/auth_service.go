package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/fathima-sithara/auth-service/internal/emailJS"
	"github.com/fathima-sithara/auth-service/internal/models"
	"github.com/fathima-sithara/auth-service/internal/repository"
	"github.com/fathima-sithara/auth-service/internal/twilio"
	"github.com/fathima-sithara/auth-service/internal/utils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidOTP          = errors.New("invalid or expired OTP")
	ErrTooManyRequests     = errors.New("too many OTP requests, please try again later")
	ErrUserAlreadyExists   = errors.New("user with this email or username already exists")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotVerified     = errors.New("user not verified")
	ErrUserNotFound        = errors.New("user not found")
	ErrRegistrationPending = errors.New("email registration initiated, please verify OTP to complete")
)

const (
	emailRegisterPrefix = "email_reg:"
)

type AuthService struct {
	userRepo         repository.UserRepository
	tw               *twilio.Client
	ej               *emailJS.Client
	redis            *redis.Client
	jwtMgr           *utils.JWTManager
	otpTTL           time.Duration
	otpRateLimit     int
	passwordHashCost int
	log              *zap.Logger
}

func NewAuthService(
	userRepo repository.UserRepository,
	tw *twilio.Client,
	ej *emailJS.Client,
	rdb *redis.Client,
	jwtMgr *utils.JWTManager,
	otpTTLMin int,
	rateLimit int,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		tw:               tw,
		ej:               ej,
		redis:            rdb,
		jwtMgr:           jwtMgr,
		otpTTL:           time.Duration(otpTTLMin) * time.Minute,
		otpRateLimit:     rateLimit,
		passwordHashCost: bcrypt.DefaultCost,
		log:              logger,
	}
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	userID, err := s.jwtMgr.ParseRefresh(refreshToken)
	if err != nil {
		s.log.Warn("Failed to parse refresh token", zap.Error(err))
		return "", "", ErrInvalidRefreshToken
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.log.Error("Failed to find user by ID during refresh", zap.Error(err), zap.String("userID", userID))
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", "", ErrInvalidRefreshToken
		}
		return "", "", fmt.Errorf("database error: %w", err)
	}

	providedTokenHash := sha256.Sum256([]byte(refreshToken))
	if user.RefreshTokenHash == "" || user.RefreshTokenHash != hex.EncodeToString(providedTokenHash[:]) {
		s.log.Warn("Provided refresh token hash does not match stored hash",
			zap.String("userID", userID),
			zap.String("storedHash", user.RefreshTokenHash),
			zap.String("providedHash", hex.EncodeToString(providedTokenHash[:])),
		)
		return "", "", ErrInvalidRefreshToken
	}

	access, _, err := s.jwtMgr.GenerateRefreshToken(userID)
	if err != nil {
		s.log.Error("Failed to generate access token during refresh", zap.Error(err), zap.String("userID", userID))
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refresh, _, err := s.jwtMgr.GenerateRefreshToken(userID)
	if err != nil {
		s.log.Error("Failed to generate new refresh token during refresh", zap.Error(err), zap.String("userID", userID))
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	newRefreshTokenHash := sha256.Sum256([]byte(refresh))
	if err := s.userRepo.SetRefreshTokenHash(ctx, userID, hex.EncodeToString(newRefreshTokenHash[:])); err != nil {
		s.log.Error("Failed to set new refresh token hash", zap.Error(err), zap.String("userID", userID))
		return "", "", fmt.Errorf("failed to update refresh token: %w", err)
	}

	return access, refresh, nil
}

func (s *AuthService) InitiateEmailRegistration(ctx context.Context, username, email, password string) error {
	_, errEmail := s.userRepo.FindByEmail(ctx, email)
	if errEmail == nil {
		return ErrUserAlreadyExists
	}
	if errEmail != nil && !errors.Is(errEmail, repository.ErrUserNotFound) {
		s.log.Error("Database error while checking for existing email", zap.Error(errEmail), zap.String("email", email))
		return fmt.Errorf("database error: %w", errEmail)
	}

	_, errUsername := s.userRepo.FindByUsername(ctx, username)
	if errUsername == nil {
		return ErrUserAlreadyExists
	}
	if errUsername != nil && !errors.Is(errUsername, repository.ErrUserNotFound) {
		s.log.Error("Database error while checking for existing username", zap.Error(errUsername), zap.String("username", username))
		return fmt.Errorf("database error: %w", errUsername)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), s.passwordHashCost)
	if err != nil {
		s.log.Error("Failed to hash password for pending registration", zap.Error(err))
		return fmt.Errorf("failed to hash password: %w", err)
	}

	regKey := emailRegisterPrefix + email
	pendingData := map[string]string{
		"username":     username,
		"passwordHash": string(hashedPassword),
	}
	if err := s.redis.HSet(ctx, regKey, pendingData).Err(); err != nil {
		s.log.Error("Failed to store pending email registration data in Redis", zap.Error(err), zap.String("email", email))
		return fmt.Errorf("failed to store registration data: %w", err)
	}
	if err := s.redis.Expire(ctx, regKey, s.otpTTL).Err(); err != nil {
		s.log.Error("Failed to set expiry for pending email registration data in Redis", zap.Error(err), zap.String("email", email))
	}

	return s.SendEmailVerificationOTP(ctx, email)
}

func (s *AuthService) SendEmailVerificationOTP(ctx context.Context, email string) error {
	rlKey := fmt.Sprintf("emailotp:rl:%s", email)
	cnt, err := s.redis.Get(ctx, rlKey).Int()
	if err != nil && !errors.Is(err, redis.Nil) {
		s.log.Error("Failed to get email OTP rate limit from Redis", zap.Error(err), zap.String("email", email))
	}
	if cnt >= s.otpRateLimit && s.otpRateLimit > 0 {
		return ErrTooManyRequests
	}

	otp := utils.GenerateOTP()
	emailOtpKey := fmt.Sprintf("emailotp:%s", email)

	if err := s.redis.Set(ctx, emailOtpKey, otp, s.otpTTL).Err(); err != nil {
		s.log.Error("Failed to set email OTP in Redis", zap.Error(err), zap.String("email", email))
		return fmt.Errorf("failed to store email OTP: %w", err)
	}

	if err := s.redis.Incr(ctx, rlKey).Err(); err != nil {
		s.log.Error("Failed to increment email OTP rate limit in Redis", zap.Error(err), zap.String("email", email))
	}
	if err := s.redis.Expire(ctx, rlKey, time.Hour).Err(); err != nil {
		s.log.Error("Failed to set expiry for email OTP rate limit in Redis", zap.Error(err), zap.String("email", email))
	}

	if s.ej != nil && s.ej.IsConfigured() {
		if err := s.ej.SendEmail(ctx, email, otp); err != nil {
			s.log.Error("Failed to send 	 OTP via EmailJS", zap.Error(err), zap.String("email", email))
			return fmt.Errorf("failed to send email: %w", err)
		}
		s.log.Info("OTP email sent", zap.String("email", email))
	} else {
		s.log.Warn("EmailJS client not configured, OTP email will not be sent", zap.String("email", email))
		if s.log.Core().Enabled(zap.DebugLevel) {
			s.log.Debug("DEBUG: OTP for email", zap.String("email", email), zap.String("otp", otp))
		}
	}
	return nil
}

func (s *AuthService) CompleteEmailVerification(ctx context.Context, email, otp string) (string, string, error) {
	emailOtpKey := fmt.Sprintf("emailotp:%s", email)
	storedOTP, err := s.redis.Get(ctx, emailOtpKey).Result()
	if err != nil {
		s.log.Warn("Failed to retrieve email OTP from Redis or OTP expired", zap.Error(err), zap.String("email", email))
		return "", "", ErrInvalidOTP
	}
	if storedOTP != otp {
		s.log.Warn("Invalid OTP provided for email", zap.String("email", email))
		return "", "", ErrInvalidOTP
	}

	if err := s.redis.Del(ctx, emailOtpKey).Err(); err != nil {
		s.log.Error("Failed to delete email OTP from Redis after verification", zap.Error(err), zap.String("email", email))
	}

	u, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		s.log.Error("Failed to find user by email during email OTP verification", zap.Error(err), zap.String("email", email))
		return "", "", fmt.Errorf("database error: %w", err)
	}

	if u != nil {
		if !u.Verified {
			u.Verified = true
			if err := s.userRepo.Update(ctx, u); err != nil {
				s.log.Error("Failed to update user email verification status after OTP", zap.Error(err), zap.String("email", email), zap.String("userID", u.ID.Hex()))
			}
		}
	} else {
		regKey := emailRegisterPrefix + email
		pendingData, err := s.redis.HGetAll(ctx, regKey).Result()
		if err != nil || len(pendingData) == 0 {
			s.log.Warn("No pending registration data found for email, or Redis error", zap.Error(err), zap.String("email", email))
			return "", "", ErrRegistrationPending
		}

		username := pendingData["username"]
		passwordHash := pendingData["passwordHash"]

		if username == "" || passwordHash == "" {
			s.log.Error("Incomplete pending registration data for email", zap.String("email", email))
			return "", "", errors.New("incomplete registration data, please try registering again")
		}

		newU := &models.User{
			Username:     username,
			Email:        email,
			PasswordHash: passwordHash,
			Verified:     true,
		}
		if err := s.userRepo.Create(ctx, newU); err != nil {
			if errors.Is(err, repository.ErrDuplicateKey) {
				return "", "", ErrUserAlreadyExists
			}
			s.log.Error("Failed to create new user on email OTP verification", zap.Error(err), zap.String("email", email))
			return "", "", fmt.Errorf("failed to create user: %w", err)
		}
		u = newU

		if err := s.redis.Del(ctx, regKey).Err(); err != nil {
			s.log.Error("Failed to delete pending email registration data from Redis", zap.Error(err), zap.String("email", email))
		}
	}

	uid := u.ID.Hex()
	access, _, err := s.jwtMgr.GenerateAccessToken(uid)
	if err != nil {
		s.log.Error("Failed to generate access token for email user after OTP verification", zap.Error(err), zap.String("userID", uid))
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}
	refresh, _, err := s.jwtMgr.GenerateRefreshToken(uid)
	if err != nil {
		s.log.Error("Failed to generate refresh token for email user after OTP verification", zap.Error(err), zap.String("userID", uid))
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	refreshHash := sha256.Sum256([]byte(refresh))
	if err := s.userRepo.SetRefreshTokenHash(ctx, uid, hex.EncodeToString(refreshHash[:])); err != nil {
		s.log.Error("Failed to set refresh token hash for email user after OTP verification", zap.Error(err), zap.String("userID", uid))
	}
	return access, refresh, nil
}

func (s *AuthService) LoginWithPassword(ctx context.Context, email, password string) (string, string, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		s.log.Warn("Attempted login with non-existent email", zap.String("email", email))
		return "", "", ErrInvalidCredentials
	}

	if !user.Verified {
		return "", "", ErrUserNotVerified
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.log.Warn("Failed password comparison for user", zap.String("email", email), zap.String("userID", user.ID.Hex()))
		return "", "", ErrInvalidCredentials
	}

	uid := user.ID.Hex()
	access, _, err := s.jwtMgr.GenerateAccessToken(uid)
	if err != nil {
		s.log.Error("Failed to generate access token after password login", zap.Error(err), zap.String("userID", uid))
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}
	refresh, _, err := s.jwtMgr.GenerateRefreshToken(uid)
	if err != nil {
		s.log.Error("Failed to generate refresh token after password login", zap.Error(err), zap.String("userID", uid))
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	refreshHash := sha256.Sum256([]byte(refresh))
	if err := s.userRepo.SetRefreshTokenHash(ctx, uid, hex.EncodeToString(refreshHash[:])); err != nil {
		s.log.Error("Failed to set refresh token hash after password login", zap.Error(err), zap.String("userID", uid))
	}

	return access, refresh, nil
}

func (s *AuthService) RequestOTP(ctx context.Context, phone, email string) error {
	if phone != "" && s.otpRateLimit > 0 {
		rlKey := fmt.Sprintf("otp:rl:%s", phone)
		cnt, err := s.redis.Get(ctx, rlKey).Int()
		if err != nil && !errors.Is(err, redis.Nil) {
			s.log.Error("Failed to get OTP rate limit from Redis", zap.Error(err), zap.String("phone", phone))
		}

		if cnt >= s.otpRateLimit {
			return ErrTooManyRequests
		}

		if err := s.redis.Incr(ctx, rlKey).Err(); err != nil {
			s.log.Error("Failed to increment OTP rate limit", zap.Error(err), zap.String("phone", phone))
		}
		_ = s.redis.Expire(ctx, rlKey, time.Hour).Err()
	}

	otp := utils.GenerateOTP()

	if phone != "" {
		otpKey := fmt.Sprintf("otp:phone:%s", phone)
		if err := s.redis.Set(ctx, otpKey, otp, s.otpTTL).Err(); err != nil {
			s.log.Error("Failed to store phone OTP in Redis", zap.Error(err), zap.String("phone", phone))
			return fmt.Errorf("failed to store phone OTP: %w", err)
		}
	}

	if email != "" {
		otpKey := fmt.Sprintf("otp:email:%s", email)
		if err := s.redis.Set(ctx, otpKey, otp, s.otpTTL).Err(); err != nil {
			s.log.Error("Failed to store email OTP in Redis", zap.Error(err), zap.String("email", email))
			return fmt.Errorf("failed to store email OTP: %w", err)
		}
	}

	if phone != "" {
		if s.tw != nil && s.tw.IsConfigured() {
			body := fmt.Sprintf("Your verification code is: %s", otp)
			if err := s.tw.SendSMS(ctx, phone, body); err != nil {
				s.log.Error("Failed to send OTP SMS via Twilio", zap.Error(err), zap.String("phone", phone))
				return fmt.Errorf("failed to send SMS: %w", err)
			}
			s.log.Info("OTP SMS sent", zap.String("phone", phone))
		} else {
			s.log.Warn("Twilio client not configured, OTP SMS will not be sent", zap.String("phone", phone))
			s.log.Debug("DEBUG: OTP for phone", zap.String("phone", phone), zap.String("otp", otp))
		}
	}

	if email != "" {
		if s.ej != nil && s.ej.IsConfigured() {
			if err := s.ej.SendEmail(ctx, email, otp); err != nil {
				s.log.Error("Failed to send OTP via EmailJS", zap.Error(err), zap.String("email", email))
				return fmt.Errorf("failed to send email: %w", err)
			}
			s.log.Info("OTP email sent", zap.String("email", email))
		} else {
			s.log.Warn("EmailJS client not configured, OTP email will not be sent", zap.String("email", email))
			s.log.Debug("DEBUG: OTP for email", zap.String("email", email), zap.String("otp", otp))
		}
	}

	return nil
}

func (s *AuthService) VerifyOTP(ctx context.Context, phone, email, otp string) (string, string, error) {
	var key string
	var identifier string

	switch {
	case phone != "":
		key = fmt.Sprintf("otp:phone:%s", phone)
		identifier = phone
	case email != "":
		key = fmt.Sprintf("otp:email:%s", email)
		identifier = email
	default:
		return "", "", fmt.Errorf("phone or email must be provided")
	}

	storedOTP, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		s.log.Warn("Failed to retrieve OTP from Redis or OTP expired", zap.Error(err), zap.String("identifier", identifier))
		return "", "", ErrInvalidOTP
	}

	if storedOTP != otp {
		s.log.Warn("Invalid OTP provided", zap.String("identifier", identifier))
		return "", "", ErrInvalidOTP
	}

	if err := s.redis.Del(ctx, key).Err(); err != nil {
		s.log.Error("Failed to delete OTP from Redis after verification", zap.Error(err), zap.String("identifier", identifier))
	}

	var u *models.User
	if phone != "" {
		u, err = s.userRepo.FindByPhone(ctx, phone)
	} else {
		u, err = s.userRepo.FindByEmail(ctx, email)
	}

	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u = &models.User{
				Phone:    phone,
				Email:    email,
				Verified: true,
			}
			if err := s.userRepo.Create(ctx, u); err != nil {
				s.log.Error("Failed to create new user after OTP verification", zap.Error(err), zap.String("identifier", identifier))
				return "", "", fmt.Errorf("failed to create user: %w", err)
			}
		} else {
			s.log.Error("Failed to find user during OTP verification", zap.Error(err), zap.String("identifier", identifier))
			return "", "", fmt.Errorf("database error: %w", err)
		}
	}

	if !u.Verified {
		u.Verified = true
		if err := s.userRepo.Update(ctx, u); err != nil {
			s.log.Error("Failed to update user verification status", zap.Error(err), zap.String("identifier", identifier), zap.String("userID", u.ID.Hex()))
		}
	}

	uid := u.ID.Hex()

	access, _, err := s.jwtMgr.GenerateAccessToken(uid)
	if err != nil {
		s.log.Error("Failed to generate access token", zap.Error(err), zap.String("userID", uid))
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refresh, _, err := s.jwtMgr.GenerateRefreshToken(uid)
	if err != nil {
		s.log.Error("Failed to generate refresh token", zap.Error(err), zap.String("userID", uid))
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	refreshHash := sha256.Sum256([]byte(refresh))
	if err := s.userRepo.SetRefreshTokenHash(ctx, uid, hex.EncodeToString(refreshHash[:])); err != nil {
		s.log.Error("Failed to store refresh token hash", zap.Error(err), zap.String("userID", uid))
	}

	return access, refresh, nil
}

func (s *AuthService) Logout(ctx context.Context, userID string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.log.Error("User not found for logout", zap.Error(err), zap.String("userID", userID))
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("database error: %w", err)
	}

	if user.RefreshTokenHash != "" {
		if err := s.userRepo.SetRefreshTokenHash(ctx, userID, ""); err != nil {
			s.log.Error("Failed to clear refresh token hash during logout", zap.Error(err), zap.String("userID", userID))
			return fmt.Errorf("failed to clear refresh token: %w", err)
		}
	}

	s.log.Info("User logged out successfully (refresh token cleared)", zap.String("userID", userID))
	return nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.log.Error("User not found for password change", zap.Error(err), zap.String("userID", userID))
		return ErrUserNotFound
	}

	if user.PasswordHash == "" {
		s.log.Warn("User attempting to change password has no password hash set", zap.String("userID", userID))
		return errors.New("cannot change password, no password set for this account")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		s.log.Warn("Old password mismatch during password change", zap.String("userID", userID))
		return ErrInvalidCredentials
	}

	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.passwordHashCost)
	if err != nil {
		s.log.Error("Failed to hash new password", zap.Error(err), zap.String("userID", userID))
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	user.PasswordHash = string(hashedNewPassword)
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.log.Error("Failed to update user's password in DB", zap.Error(err), zap.String("userID", userID))
		return fmt.Errorf("failed to update password: %w", err)
	}

	s.log.Info("User password changed successfully", zap.String("userID", userID))
	return nil
}

func (s *AuthService) GetUserIDFromAccessToken(accessToken string) (string, error) {
	return s.jwtMgr.ParseAccess(accessToken)
}
