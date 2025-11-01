package logger

import (
	"go.uber.org/zap"
	"sync"
)

var (
	instance *zap.SugaredLogger
	once     sync.Once
)

type Config struct {
	Development bool
}

func New(cfg Config) (*zap.SugaredLogger, error) {
	var err error
	once.Do(func() {
		var l *zap.Logger
		if cfg.Development {
			l, err = zap.NewDevelopment()
		} else {
			l, err = zap.NewProduction()
		}
		if err != nil {
			return
		}
		instance = l.Sugar()
	})
	return instance, err
}
