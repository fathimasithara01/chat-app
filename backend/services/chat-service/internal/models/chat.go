package models

import "time"

type Message struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	SenderID  string    `bson:"sender_id" json:"sender_id"`
	Content   string    `bson:"content" json:"content"`
	MsgType   string    `bson:"msg_type" json:"msg_type"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

type Chat struct {
	ID          string    `bson:"_id,omitempty" json:"id"`
	Name        string    `bson:"name,omitempty" json:"name"`
	IsGroup     bool      `bson:"is_group" json:"is_group"`
	Members     []string  `bson:"members" json:"members"` // user IDs only
	LastMessage *Message  `bson:"last_message,omitempty" json:"last_message,omitempty"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}
