package config

import (
	"time"

	"github.com/spf13/viper"
)

type AppConf struct {
	Env            string `mapstructure:"env"`
	Port           int    `mapstructure:"port"`
	ShutdownSecond int    `mapstructure:"shutdown_seconds"`
}

type MongoConf struct {
	URI        string `mapstructure:"uri"`
	Database   string `mapstructure:"database"`
	Collection string `mapstructure:"collection"`
}

type AWSConf struct {
	Region   string `mapstructure:"region"`
	Bucket   string `mapstructure:"bucket"`
	Endpoint string `mapstructure:"endpoint"`
}

type S3Conf struct {
	PublicRead bool `mapstructure:"public_read"`
	PresignTTL int  `mapstructure:"presign_ttl_seconds"`
}

type RedisConf struct {
	Addr      string `mapstructure:"addr"`
	Password  string `mapstructure:"password"`
	DB        int    `mapstructure:"db"`
	SignedTTL int    `mapstructure:"signed_url_cache_ttl_seconds"`
}

type JWTConf struct {
	PublicKeyPath string `mapstructure:"public_key_path"`
}

type Config struct {
	App   AppConf `mapstructure:"app"`
	Mongo MongoConf `mapstructure:"mongodb"`
	AWS   AWSConf `mapstructure:"aws"`
	S3    S3Conf `mapstructure:"s3"`
	Redis RedisConf `mapstructure:"redis"`
	JWT   JWTConf `mapstructure:"jwt"`
	Log   struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`

	// derived
	ShutdownTimeout time.Duration
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	if cfg.App.ShutdownSecond == 0 {
		cfg.App.ShutdownSecond = 15
	}
	cfg.ShutdownTimeout = time.Duration(cfg.App.ShutdownSecond) * time.Second
	if cfg.S3.PresignTTL == 0 {
		cfg.S3.PresignTTL = 600
	}
	if cfg.Redis.SignedTTL == 0 {
		cfg.Redis.SignedTTL = cfg.S3.PresignTTL
	}
	return &cfg, nil
}
