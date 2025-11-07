package utils

import "go.uber.org/zap"

func NewLogger(env string) *zap.Logger {
	var log *zap.Logger
	var err error
	if env == "development" {
		log, err = zap.NewDevelopment()
	} else {
		log, err = zap.NewProduction()
	}
	if err != nil {
		panic(err)
	}
	return log
}
