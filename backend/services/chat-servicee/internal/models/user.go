package models

import "time"

type User struct {
	ID        string    `bson:"_id" json:"id"`
	Username  string    `bson:"username" json:"username"`
	Email     string    `bson:"email" json:"email"`
	Phone     string    `bson:"phone,omitempty" json:"phone,omitempty"`
	IsOnline  bool      `bson:"is_online" json:"is_online"`
	LastSeen  time.Time `bson:"last_seen" json:"last_seen"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}
