package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	RedisHost     string
	RedisPassword string
	RedisDB       string
	PublicKeyPath string
}

func Load() *Config {
	_ = godotenv.Load("./configs/.env")

	cfg := &Config{
		Port:          os.Getenv("PORT"),
		RedisHost:     os.Getenv("REDIS_HOST"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       os.Getenv("REDIS_DB"),
		PublicKeyPath: os.Getenv("PUBLIC_KEY_PATH"),
	}

	if cfg.Port == "" {
		log.Fatal("PORT missing")
	}

	return cfg
}
