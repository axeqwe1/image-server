package handlers

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time" // เพิ่ม import สำหรับ timestamp

	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func Upload(c *fiber.Ctx) error {
	// รับ multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).SendString("Upload failed")
	}

	// ดึงไฟล์ทั้งหมดจาก key "image" (สามารถอัพโหลดหลายไฟล์ได้)
	files := form.File["image"]
	if len(files) == 0 {
		return c.Status(400).SendString("No images uploaded")
	}

	// สร้าง slice เพื่อเก็บ URL ของภาพที่อัพโหลด
	var urls []string

	// วนลูปบันทึกไฟล์แต่ละไฟล์
	for _, file := range files {

		filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(file.Filename))
		savePath := filepath.Join("./uploads", filename)
		err = c.SaveFile(file, savePath)
		if err != nil {
			return c.Status(500).SendString("Cannot save file: " + file.Filename)
		}

		// เพิ่ม URL ลงใน slice
		url := "https://" + c.Hostname() + "/images/" + filename
		urls = append(urls, url)
	}

	// ส่ง response กลับไป
	return c.JSON(fiber.Map{
		"message": "Images uploaded successfully",
		"urls":    urls,
	})
}

func Resize(c *fiber.Ctx) error {
	filename := c.Params("filename")
	width := c.QueryInt("w", 100)
	height := c.QueryInt("h", 100)

	// สร้างชื่อไฟล์สำหรับรูปที่ resize
	resizedFilename := fmt.Sprintf("%s_%dx%d.jpg", strings.TrimSuffix(filename, filepath.Ext(filename)), width, height)
	resizedPath := filepath.Join("./uploads", resizedFilename)

	// ตรวจสอบว่ามีไฟล์อยู่แล้วหรือไม่
	if _, err := os.Stat(resizedPath); err == nil {
		return c.Type("jpg").SendFile(resizedPath)
	}

	imgPath := filepath.Join("./uploads", filename)
	img, err := imaging.Open(imgPath)
	if err != nil {
		return c.Status(404).SendString("Image not found")
	}

	resized := imaging.Resize(img, width, height, imaging.Lanczos)

	// บันทึกไฟล์ที่ resize ไว้
	err = imaging.Save(resized, resizedPath, imaging.JPEGQuality(85))
	if err != nil {
		return c.Status(500).SendString("Error saving resized image")
	}

	return c.Type("jpg").SendFile(resizedPath)
}

func TestUpload(t *testing.T) {
	app := fiber.New()
	app.Post("/upload", Upload)

	req := httptest.NewRequest("POST", "/upload", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}
