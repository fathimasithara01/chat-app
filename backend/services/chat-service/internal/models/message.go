package models

import (
    "time"

    "go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
    ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    ConversationID primitive.ObjectID `bson:"conversation_id" json:"conversation_id"`
    SenderID       string             `bson:"sender_id" json:"sender_id"`
    ToID           string             `bson:"to_id,omitempty" json:"to_id,omitempty"`
    Content        string             `bson:"content" json:"content"`
    CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
    Delivered      bool               `bson:"delivered" json:"delivered"`
    Read           bool               `bson:"read" json:"read"`
}
