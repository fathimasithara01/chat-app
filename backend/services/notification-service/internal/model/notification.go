package model

import "time"

type Notification struct {
	ID        string    `json:"id" bson:"_id,omitempty"`
	UserID    string    `json:"user_id" bson:"user_id"`
	Title     string    `json:"title" bson:"title"`
	Message   string    `json:"message" bson:"message"`
	Type      string    `json:"type" bson:"type"`
	Read      bool      `json:"read" bson:"read"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}
