package domain

import "time"

type Message struct {
	ID         string              `bson:"_id" json:"id"`
	ChatID     string              `bson:"chat_id" json:"chat_id"`
	SenderID   string              `bson:"sender_id" json:"sender_id"`
	Content    string              `bson:"content" json:"content"`
	MsgType    string              `bson:"msg_type" json:"msg_type"`
	Encrypted  bool                `bson:"encrypted" json:"encrypted"`
	Metadata   map[string]string   `bson:"metadata,omitempty" json:"metadata,omitempty"`
	ReplyTo    string              `bson:"reply_to,omitempty" json:"reply_to,omitempty"`
	CreatedAt  time.Time           `bson:"created_at" json:"created_at"`
	EditedAt   *time.Time          `bson:"edited_at,omitempty" json:"edited_at,omitempty"`
	Delivered  bool                `bson:"delivered" json:"delivered"`
	ReadBy     []string            `bson:"read_by" json:"read_by"`
	DeletedFor []string            `bson:"deleted_for" json:"deleted_for"`
	Reactions  map[string][]string `bson:"reactions,omitempty" json:"reactions,omitempty"`
}
