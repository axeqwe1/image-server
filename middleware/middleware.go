package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// Cors middleware เพื่อจัดการ Cross-Origin Resource Sharing
func Cors() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		c.Set("Access-Control-Max-Age", "86400")

		if c.Method() == fiber.MethodOptions {
			fmt.Println("Preflight request received")
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}
