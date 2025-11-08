package models

import "time"

type Message struct {
	ID        string    `bson:"_id" json:"id"`
	ChatID    string    `bson:"chat_id" json:"chat_id"`
	SenderID  string    `bson:"sender_id" json:"sender_id"`
	Content   string    `bson:"content" json:"content"`
	Type      string    `bson:"type" json:"type"` // text, image, etc
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
	Edited    bool      `bson:"edited" json:"edited"`
	Deleted   bool      `bson:"deleted" json:"deleted"`
}
	