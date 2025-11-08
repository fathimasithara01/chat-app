package middleware

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

func JWTAuth(publicKeyPath string) fiber.Handler {
	pubBytes, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		log.Fatal().Err(err).Msg("read jwt pubkey")
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubBytes)
	if err != nil {
		log.Fatal().Err(err).Msg("parse pubkey")
	}

	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "missing auth"})
		}
		parts := strings.Split(auth, " ")
		if len(parts) != 2 {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid auth"})
		}
		tokenStr := parts[1]
		_, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) { return pubKey, nil })
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Next()
	}
}
