package middleware

import (
	"crypto/rsa"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func JWTAuth(publicKeyPath string) fiber.Handler {
	var pub *rsa.PublicKey
	if publicKeyPath != "" {
		b, _ := ioutil.ReadFile(publicKeyPath)
		if len(b) > 0 {
			pub, _ = jwt.ParseRSAPublicKeyFromPEM(b)
		}
	}

	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization"})
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization header"})
		}
		tokenStr := parts[1]

		var parsed *jwt.Token
		var err error
		if pub != nil {
			parsed, err = jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fiber.ErrUnauthorized
				}
				return pub, nil
			})
		} else {
			parsed, _, err = new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
		}

		if err != nil || parsed == nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		if claims, ok := parsed.Claims.(jwt.MapClaims); ok {
			c.Locals("user_claims", claims)
		}
		return c.Next()
	}
}
