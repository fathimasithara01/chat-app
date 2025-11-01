package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fathima-sithara/auth-service/internal/brevo"
	"github.com/fathima-sithara/auth-service/internal/models"
	"github.com/fathima-sithara/auth-service/internal/repository"
	"github.com/fathima-sithara/auth-service/internal/twilio"
	"github.com/fathima-sithara/auth-service/internal/utils"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

const (
	otpPrefix               = "otp:"
	otpRateLimitPrefix      = "otp_rate_limit:"
	emailVerificationPrefix = "email_verify:"
	refreshTokenPrefix      = "refresh_token:"
)

// authService implements the AuthService interface
type authService struct {
	userRepo                    repository.UserRepository
	twilioClient                twilio.Client
	brevoClient                 brevo.Client
	redisClient                 *redis.Client
	jwtSecret                   string
	accessTokenTTLMinutes       int
	refreshTokenTTLDays         int
	otpTTLMinutes               int
	otpRateLimitPerPhonePerHour int
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo repository.UserRepository,
	twilioClient twilio.Client,
	brevoClient brevo.Client,
	redisClient *redis.Client,
	jwtSecret string,
	accessTokenTTLMinutes int,
	refreshTokenTTLDays int,
	otpTTLMinutes int,
	otpRateLimitPerPhonePerHour int,
) AuthService {
	return &authService{
		userRepo:                    userRepo,
		twilioClient:                twilioClient,
		brevoClient:                 brevoClient,
		redisClient:                 redisClient,
		jwtSecret:                   jwtSecret,
		accessTokenTTLMinutes:       accessTokenTTLMinutes,
		refreshTokenTTLDays:         refreshTokenTTLDays,
		otpTTLMinutes:               otpTTLMinutes,
		otpRateLimitPerPhonePerHour: otpRateLimitPerPhonePerHour,
	}
}

// RequestOTP sends an OTP to the given phone number.
func (s *authService) RequestOTP(ctx context.Context, phoneNumber string) error {
	// 1. Check rate limit
	rateLimitKey := otpRateLimitPrefix + phoneNumber
	count, err := s.redisClient.Incr(ctx, rateLimitKey).Result()
	if err != nil {
		return fmt.Errorf("failed to increment OTP rate limit: %w", ErrInternal)
	}

	if count == 1 {
		// Set expiration for the rate limit key if it's the first request
		_, err = s.redisClient.Expire(ctx, rateLimitKey, time.Hour).Result()
		if err != nil {
			return fmt.Errorf("failed to set expiry for OTP rate limit: %w", ErrInternal)
		}
	} else if count > int64(s.otpRateLimitPerPhonePerHour) {
		// If limit exceeded, reset count to avoid continuous increment and return error
		s.redisClient.Decr(ctx, rateLimitKey) // Decrement to reflect only valid attempts
		return ErrOTPRateLimited
	}

	// 2. Generate OTP
	otpCode := utils.GenerateOTP(6) // Assuming 6-digit OTP

	// 3. Store OTP in Redis with expiration
	otpKey := otpPrefix + phoneNumber
	err = s.redisClient.Set(ctx, otpKey, otpCode, time.Duration(s.otpTTLMinutes)*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to store OTP in Redis: %w", ErrInternal)
	}

	// 4. Send OTP via Twilio
	message := fmt.Sprintf("Your Chat App verification code is: %s. It is valid for %d minutes.", otpCode, s.otpTTLMinutes)
	err = s.twilioClient.SendSMS(ctx, phoneNumber, message)
	if err != nil {
		return fmt.Errorf("failed to send OTP via Twilio: %w", ErrInternal)
	}

	return nil
}

// VerifyOTP verifies the provided OTP and logs in the user, or creates one if not exists.
func (s *authService) VerifyOTP(ctx context.Context, phoneNumber, code string) (*models.AuthTokens, error) {
	otpKey := otpPrefix + phoneNumber
	storedOTP, err := s.redisClient.Get(ctx, otpKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrOTPExpired // OTP either expired or never sent
		}
		return nil, fmt.Errorf("failed to get OTP from Redis: %w", ErrInternal)
	}

	if storedOTP != code {
		return nil, ErrInvalidOTP
	}

	// OTP is valid, delete it from Redis
	s.redisClient.Del(ctx, otpKey)

	// Find or create user
	user, err := s.userRepo.FindUserByPhoneNumber(ctx, phoneNumber)
	if err != nil {
		if errors.Is(err, errors.New("user not found")) { // Check for specific "not found" error from repo
			// User doesn't exist, create a new one
			newUser := &models.User{
				PhoneNumber: phoneNumber,
				IsVerified:  true,                  // Phone number is verified through OTP
				FullName:    "User-" + phoneNumber, // Default name, user can update later
			}
			if createErr := s.userRepo.CreateUser(ctx, newUser); createErr != nil {
				return nil, fmt.Errorf("failed to create new user: %w", ErrInternal)
			}
			user = newUser
		} else {
			return nil, fmt.Errorf("failed to find user by phone number: %w", ErrInternal)
		}
	}

	// Generate and return tokens
	tokens, err := s.generateAndStoreTokens(ctx, user.ID.Hex())
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", ErrInternal)
	}

	// Update last login time
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		// Log this error but don't return, as auth was successful
		fmt.Printf("Warning: Failed to update last login time for user %s: %v\n", user.ID.Hex(), err)
	}

	return tokens, nil
}

// RegisterEmail registers a new user with email and password, sending a verification email.
func (s *authService) RegisterEmail(ctx context.Context, email, password, fullName string) (*models.User, error) {
	// Check if user already exists
	_, err := s.userRepo.FindUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrUserAlreadyExists
	}
	if err != nil && !errors.Is(err, errors.New("user not found")) {
		return nil, fmt.Errorf("failed to check existing user by email: %w", ErrInternal)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", ErrInternal)
	}

	newUser := &models.User{
		FullName:     fullName,
		Email:        email,
		PasswordHash: string(hashedPassword),
		IsVerified:   false, // Email needs verification
	}

	if err := s.userRepo.CreateUser(ctx, newUser); err != nil {
		var writeException mongo.WriteException
		if errors.As(err, &writeException) {
			for _, we := range writeException.WriteErrors {
				if we.Code == 11000 { // Duplicate key error
					return nil, ErrUserAlreadyExists
				}
			}
		}
		return nil, fmt.Errorf("failed to create new email user: %w", ErrInternal)
	}

	// Generate email verification code
	verificationCode := utils.GenerateOTP(6) // Reusing OTP generation for email
	emailVerifyKey := emailVerificationPrefix + email
	err = s.redisClient.Set(ctx, emailVerifyKey, verificationCode, time.Duration(s.otpTTLMinutes)*time.Minute).Err()
	if err != nil {
		// Log this but don't block user creation, as email can be resent
		fmt.Printf("Warning: Failed to store email verification code for %s: %v\n", email, err)
	}

	// Send verification email
	go func() {
		emailCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		sendErr := s.brevoClient.SendVerificationEmail(emailCtx, email, fullName, verificationCode)
		if sendErr != nil {
			fmt.Printf("Warning: Failed to send verification email to %s: %v\n", email, sendErr)
		}
	}()

	return newUser, nil
}

// VerifyEmail verifies the provided email verification code.
func (s *authService) VerifyEmail(ctx context.Context, email, code string) (*models.AuthTokens, error) {
	emailVerifyKey := emailVerificationPrefix + email
	storedCode, err := s.redisClient.Get(ctx, emailVerifyKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrVerificationCodeExpired
		}
		return nil, fmt.Errorf("failed to get email verification code from Redis: %w", ErrInternal)
	}

	if storedCode != code {
		return nil, ErrInvalidVerificationCode
	}

	// Code is valid, delete from Redis
	s.redisClient.Del(ctx, emailVerifyKey)

	user, err := s.userRepo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, errors.New("user not found")) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user for email verification: %w", ErrInternal)
	}

	if user.IsVerified {
		return nil, errors.New("email already verified")
	}

	user.IsVerified = true
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user verification status: %w", ErrInternal)
	}

	// Generate and return tokens
	tokens, err := s.generateAndStoreTokens(ctx, user.ID.Hex())
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", ErrInternal)
	}

	// Update last login time
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		fmt.Printf("Warning: Failed to update last login time for user %s: %v\n", user.ID.Hex(), err)
	}

	return tokens, nil
}

// LoginEmail logs in a user with email and password.
func (s *authService) LoginEmail(ctx context.Context, email, password string) (*models.AuthTokens, error) {
	user, err := s.userRepo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, errors.New("user not found")) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to find user during email login: %w", ErrInternal)
	}

	if !user.IsVerified {
		return nil, ErrEmailNotVerified
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate and return tokens
	tokens, err := s.generateAndStoreTokens(ctx, user.ID.Hex())
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", ErrInternal)
	}

	// Update last login time
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		fmt.Printf("Warning: Failed to update last login time for user %s: %v\n", user.ID.Hex(), err)
	}

	return tokens, nil
}

// RefreshTokens refreshes access and refresh tokens.
func (s *authService) RefreshTokens(ctx context.Context, refreshToken string) (*models.AuthTokens, error) {
	claims, err := utils.ParseJWT(refreshToken, s.jwtSecret)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	userID := claims.UserID
	storedRefreshToken, err := s.redisClient.Get(ctx, refreshTokenPrefix+userID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrInvalidRefreshToken // Token not found or expired in Redis
		}
		return nil, fmt.Errorf("failed to get refresh token from Redis: %w", ErrInternal)
	}

	if storedRefreshToken != refreshToken {
		return nil, ErrInvalidRefreshToken // Token mismatch
	}

	// Invalidate the old refresh token immediately
	s.redisClient.Del(ctx, refreshTokenPrefix+userID)

	// Generate new tokens
	newTokens, err := s.generateAndStoreTokens(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new tokens: %w", ErrInternal)
	}

	return newTokens, nil
}

// generateAndStoreTokens is a helper to create JWTs and store refresh token in Redis
func (s *authService) generateAndStoreTokens(ctx context.Context, userID string) (*models.AuthTokens, error) {
	accessToken, accessExp, err := utils.GenerateAccessToken(userID, s.jwtSecret, s.accessTokenTTLMinutes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, _, err := utils.GenerateRefreshToken(userID, s.jwtSecret, s.refreshTokenTTLDays)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token in Redis
	refreshTTL := time.Duration(s.refreshTokenTTLDays) * 24 * time.Hour
	err = s.redisClient.Set(ctx, refreshTokenPrefix+userID, refreshToken, refreshTTL).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token in Redis: %w", ErrInternal)
	}

	return &models.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExp.Unix(),
	}, nil
}
