package config

import (
	"time"

	"github.com/spf13/viper"
)

type ServerCfg struct {
	Port                string `mapstructure:"port"`
	ReadTimeoutSeconds  int    `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `mapstructure:"write_timeout_seconds"`
}

type RedisCfg struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	Prefix   string `mapstructure:"prefix"`
}

type JwtCfg struct {
	PublicKeyPath string `mapstructure:"public_key_path"`
	SigningMethod string `mapstructure:"signing_method"`
}

type Config struct {
	Server ServerCfg `mapstructure:"server"`
	Redis  RedisCfg  `mapstructure:"redis"`
	JWT    JwtCfg    `mapstructure:"jwt"`
	// Derived
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.AutomaticEnv()
	v.SetEnvPrefix("APP")
	// allow nested override: APP_SERVER_PORT etc.
	v.SetEnvKeyReplacer(nil)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if cfg.Server.ReadTimeoutSeconds == 0 {
		cfg.Server.ReadTimeoutSeconds = 15
	}
	if cfg.Server.WriteTimeoutSeconds == 0 {
		cfg.Server.WriteTimeoutSeconds = 15
	}
	cfg.ReadTimeout = time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second
	cfg.WriteTimeout = time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second
	return &cfg, nil
}
