package config

import (
	"github.com/spf13/viper"
)

type ServerCfg struct {
	Port                string `mapstructure:"port"`
	ReadTimeoutSeconds  int    `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `mapstructure:"write_timeout_seconds"`
}

type JWTCfg struct {
	PublicKeyPath string `mapstructure:"public_key_path"`
}

type LogCfg struct {
	Level string `mapstructure:"level"`
}

type Config struct {
	Server   ServerCfg           `mapstructure:"server"`
	Services map[string][]string `mapstructure:"services"`
	JWT      JWTCfg              `mapstructure:"jwt"`
	Log      LogCfg              `mapstructure:"log"`

	// convenience fields
	Port                string
	ReadTimeoutSeconds  int
	WriteTimeoutSeconds int
}

func LoadConfig(path string) (*Config, error) {
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
	// defaults
	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}
	if cfg.Server.ReadTimeoutSeconds == 0 {
		cfg.Server.ReadTimeoutSeconds = 15
	}
	if cfg.Server.WriteTimeoutSeconds == 0 {
		cfg.Server.WriteTimeoutSeconds = 15
	}
	return &cfg, nil
}
