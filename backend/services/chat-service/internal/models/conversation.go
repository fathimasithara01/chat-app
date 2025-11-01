package models

import "time"

type Conversation struct {
	ID        string   `bson:"_id,omitempty" json:"id"`
	Type      string   `bson:"type" json:"type"`
	Members   []string `bson:"members" json:"members"`
	Title     string   `bson:"title,omitempty" json:"title,omitempty"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}
