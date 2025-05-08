package handlers

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time" // เพิ่ม import สำหรับ timesta

	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v2"
	pngquant "github.com/yusukebe/go-pngquant"
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

	clientIP := c.IP()
	userAgent := c.Get("User-Agent")

	// สร้าง slice เพื่อเก็บ URL ของภาพที่อัพโหลด
	var urls []string

	// วนลูปบันทึกไฟล์แต่ละไฟล์
	for _, file := range files {
		originalName := file.Filename
		newName := time.Now().Format("20060102_150405") + ".png"
		savePath := filepath.Join("./uploads", newName)
		err = c.SaveFile(file, savePath)

		checkPngquant()

		if err != nil {
			return c.Status(500).SendString("Cannot save file: " + file.Filename)
		}

		// บีบอัดภาพหลังบันทึก
		err = compressPNG(savePath, savePath)
		if err != nil {
			fmt.Printf("Compress failed for %s: %v\n", newName, err)
		}

		// Logging
		fmt.Printf("UPLOAD [%s] from [%s] - UA: [%s] - original: %s => saved: %s\n",
			time.Now().Format(time.RFC3339), clientIP, userAgent, originalName, newName)

		// เพิ่ม URL ลงใน slice
		url := "https://" + "www.ymt-group.com" + "/pos-image/" + newName
		urls = append(urls, url)
	}

	// ส่ง response กลับไป
	return c.JSON(fiber.Map{
		"message": "Images uploaded successfully",
		"urls":    urls,
	})
}

// https://www.ymt-group.com/image-server/image/1742182975093345996.jpg/resize?w=500&h=300
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

func compressPNG(inputPath, outputPath string) error {
	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %v", err)
	}

	outputData, err := pngquant.CompressBytes(inputData, "5")
	if err != nil {
		return fmt.Errorf("failed to compress PNG: %v", err)
	}

	err = os.WriteFile(outputPath, outputData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}

	return nil
}

func CompressBytes(input []byte, speed string) (output []byte, err error) {
	cmd := exec.Command("pngquant", "-", "--speed", speed)
	cmd.Stdin = bytes.NewReader(input) // เปลี่ยนจาก strings.NewReader เป็น bytes.NewReader
	var o, stderr bytes.Buffer
	cmd.Stdout = &o
	cmd.Stderr = &stderr // จับ stderr เพื่อ debug

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("pngquant failed: %v, stderr: %s", err, stderr.String())
	}

	output = o.Bytes()
	return output, nil
}

func checkPngquant() error {
	_, err := exec.LookPath("pngquant")
	if err != nil {
		return fmt.Errorf("pngquant not found in PATH: %v", err)
	}
	return nil
}
