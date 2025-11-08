package models

import "time"

type Chat struct {
	ID           string    `bson:"_id,omitempty"`
	Participants []string  `bson:"participants"`
	IsGroup      bool      `bson:"is_group"`
	GroupName    string    `bson:"group_name,omitempty"`
	CreatedAt    time.Time `bson:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at"`
	Deleted      bool      `bson:"deleted"`
}
