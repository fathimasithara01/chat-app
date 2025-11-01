package models

import "time"

type Media struct {
	ID          string    `bson:"_id" json:"id"`
	UserID      string    `bson:"user_id" json:"user_id"`
	Key         string    `bson:"key" json:"key"` // S3 object key
	URL         string    `bson:"url" json:"url"` // optional public URL
	Thumbnail   string    `bson:"thumbnail,omitempty" json:"thumbnail,omitempty"`
	Type        string    `bson:"type" json:"type"` // image|video|file
	Size        int64     `bson:"size" json:"size"`
	ContentType string    `bson:"content_type" json:"content_type"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
}
