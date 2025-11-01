package models

import "time"

type Message struct {
	ID             string    `bson:"_id,omitempty" json:"id"`
	ConversationID string    `bson:"conversation_id" json:"conversation_id"`
	SenderID       string    `bson:"sender_id" json:"sender_id"`
	Content        string    `bson:"content" json:"content"`
	Type           string    `bson:"type" json:"type"`
	Attachments    []string  `bson:"attachments,omitempty" json:"attachments,omitempty"`
	CreatedAt      time.Time `bson:"created_at" json:"created_at"`
}
