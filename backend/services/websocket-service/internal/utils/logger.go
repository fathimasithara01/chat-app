package utils

import (
	"go.uber.org/zap"
)

func NewLogger(dev bool) (*zap.Logger, error) {
	if dev {
		cfg := zap.NewDevelopmentConfig()
		return cfg.Build()
	}
	return zap.NewProduction()
}
