package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the authentication system.
type User struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username         string             `bson:"username,omitempty" json:"username,omitempty"` // Added for chat app
	Phone            string             `bson:"phone,omitempty" json:"phone,omitempty"`
	Email            string             `bson:"email,omitempty" json:"email,omitempty"`
	PasswordHash     string             `bson:"password_hash,omitempty" json:"-"`            // Stored hashed password
	RefreshTokenHash string             `bson:"refresh_token_hash,omitempty" json:"-"`       // Stored hashed refresh token
	Verified         bool               `bson:"verified" json:"verified"`                    // Indicates if phone/email is verified
	CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time          `bson:"updated_at" json:"updated_at"`
}