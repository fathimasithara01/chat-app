package config

import (
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type AppConfig struct {
	Env  string
	Port int
	JWT  struct {
		Secret           string
		AccessTTLMinutes int
		RefreshTTLDays   int
	}
}

type MongoConfig struct {
	URI      string
	Database string
}

type UserConfig struct {
	Collection string
}

type Config struct {
	App   AppConfig
	Mongo MongoConfig
	User  UserConfig
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	_ = godotenv.Load()
	viper.AutomaticEnv()

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if secret := viper.GetString("JWT_SECRET"); secret != "" {
		cfg.App.JWT.Secret = secret
	}
	if viper.GetString("JWT_ACCESS_TTL_MINUTES") != "" {
		cfg.App.JWT.AccessTTLMinutes = viper.GetInt("JWT_ACCESS_TTL_MINUTES")
	}
	if viper.GetString("JWT_REFRESH_TTL_DAYS") != "" {
		cfg.App.JWT.RefreshTTLDays = viper.GetInt("JWT_REFRESH_TTL_DAYS")
	}

	return cfg, nil
}
