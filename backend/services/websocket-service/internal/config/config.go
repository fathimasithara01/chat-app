package config

import (
	"log"
	"os"
)

type Config struct {
	JWTSecret string
	Port      string
	NATSURL   string
}

func Load() *Config {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	return &Config{
		JWTSecret: secret,
		Port:      port,
		NATSURL:   natsURL,
	}
}
