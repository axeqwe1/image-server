package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// Cors middleware เพื่อจัดการ Cross-Origin Resource Sharing
func Cors() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// ตั้งค่า header พื้นฐานสำหรับ CORS
		c.Set("Access-Control-Allow-Origin", "*") // อนุญาตทุก origin (ปรับได้)
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Set("Access-Control-Max-Age", "86400") // Cache preflight request 24 ชม.

		// ถ้าเป็น preflight request (OPTIONS) ให้ตอบกลับทันที
		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}

		// ดำเนินการต่อไปยัง handler ถัดไป
		return c.Next()
	}
}
