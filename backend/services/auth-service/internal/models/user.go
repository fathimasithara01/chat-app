package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the system
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	FullName     string             `bson:"full_name" json:"full_name"`
	Email        string             `bson:"email" json:"email"`
	PhoneNumber  string             `bson:"phone_number" json:"phone_number"` // Stored without country code
	PasswordHash string             `bson:"password_hash" json:"-"`           // Hashed password
	IsVerified   bool               `bson:"is_verified" json:"is_verified"`   // For email verification
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
	LastLoginAt  *time.Time         `bson:"last_login_at,omitempty" json:"last_login_at,omitempty"`
}

// LoginRequest defines the structure for a login request
type LoginRequest struct {
	Identifier string `json:"identifier" validate:"required"` // Can be email or phone number
	Password   string `json:"password" validate:"required"`
}

// RegisterEmailRequest defines the structure for email registration
type RegisterEmailRequest struct {
	FullName string `json:"full_name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// RefreshTokenRequest defines the structure for refreshing tokens
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// AuthTokens holds access and refresh tokens
type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"` // Unix timestamp for access token expiry
}
