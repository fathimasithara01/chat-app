package utils

import "github.com/gofiber/fiber/v2"

func JSONSuccess(c *fiber.Ctx, status int, payload interface{}) error {
	return c.Status(status).JSON(fiber.Map{"status": "ok", "data": payload})
}

func JSONError(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(fiber.Map{"status": "error", "message": msg})
}
