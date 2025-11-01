package services

import (
	"context"
	"errors"

	"github.com/fathima-sithara/auth-service/internal/models"
)

var (
	ErrUserAlreadyExists       = errors.New("user with this email or phone number already exists")
	ErrUserNotFound            = errors.New("user not found")
	ErrInvalidCredentials      = errors.New("invalid email/password or phone/OTP")
	ErrInvalidOTP              = errors.New("invalid or expired OTP code")
	ErrOTPExpired              = errors.New("OTP code has expired")
	ErrOTPRateLimited          = errors.New("too many OTP requests, please try again later")
	ErrInvalidVerificationCode = errors.New("invalid or expired verification code")
	ErrVerificationCodeExpired = errors.New("email verification code has expired")
	ErrInvalidRefreshToken     = errors.New("invalid or expired refresh token")
	ErrEmailNotVerified        = errors.New("email not verified")
	ErrForbidden               = errors.New("forbidden")
	ErrInternal                = errors.New("internal server error")
)

// AuthService defines the interface for authentication operations
type AuthService interface {
	RequestOTP(ctx context.Context, phoneNumber string) error
	VerifyOTP(ctx context.Context, phoneNumber, code string) (*models.AuthTokens, error)
	RegisterEmail(ctx context.Context, email, password, fullName string) (*models.User, error)
	VerifyEmail(ctx context.Context, email, code string) (*models.AuthTokens, error)
	LoginEmail(ctx context.Context, email, password string) (*models.AuthTokens, error) // Added for completeness
	RefreshTokens(ctx context.Context, refreshToken string) (*models.AuthTokens, error)
}
