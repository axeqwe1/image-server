package main

import (
	"log"
	"os"

	"github.com/axeqwe1/image-server/handlers"
	"github.com/axeqwe1/image-server/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		log.Fatal("Cannot create uploads directory:", err)
	}

	app := fiber.New(fiber.Config{
		BodyLimit:         100 * 1024 * 1024, // 100MB
		StreamRequestBody: true,              // Stream แทนการเก็บใน memory
	})
	// ใช้ CORS middleware
	app.Use(middleware.Cors())
	// เพิ่ม route สำหรับ "/"
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome to Go Image Server!")
	})
	app.Post("/upload", handlers.Upload)
	app.Get("/image/:filename/resize", handlers.Resize)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Fatal(app.Listen("127.0.0.1:" + port))
}
