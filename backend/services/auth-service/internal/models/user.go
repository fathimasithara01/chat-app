package models

import (
    "time"

    "go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
    ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UUID         string             `bson:"uuid" json:"uuid"` // stable public id (uuid)
    Phone        string             `bson:"phone,omitempty" json:"phone,omitempty"`
    Email        string             `bson:"email,omitempty" json:"email,omitempty"`
    PasswordHash string             `bson:"password_hash,omitempty" json:"-"`
    IsEmailVerified bool            `bson:"is_email_verified" json:"is_email_verified"`
    IsPhoneVerified bool            `bson:"is_phone_verified" json:"is_phone_verified"`
    CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}
