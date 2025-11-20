package util

import "github.com/google/uuid"

func NewID() string { return uuid.NewString() }
