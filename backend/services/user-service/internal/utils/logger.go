package utils

import (
	"go.uber.org/zap"
)

func NewLogger(env string) *zap.Logger {
	if env == "development" {
		l, _ := zap.NewDevelopment()
		return l
	}
	l, _ := zap.NewProduction()
	return l
}
