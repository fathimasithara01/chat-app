package utils

import (
	"go.uber.org/zap"
)

func NewLogger(dev bool) (*zap.SugaredLogger, error) {
	var z *zap.Logger
	var err error
	if dev {
		cfg := zap.NewDevelopmentConfig()
		z, err = cfg.Build()
	} else {
		z, err = zap.NewProduction()
	}
	if err != nil {
		return nil, err
	}
	return z.Sugar(), nil
}
