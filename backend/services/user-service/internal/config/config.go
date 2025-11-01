package config

import (
	"log"
	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port string
	}
	MongoDB struct {
		URI        string
		Database   string
		Collection string
	}
}

func LoadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config: %v", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Unable to decode config: %v", err)
	}

	return &cfg
}
