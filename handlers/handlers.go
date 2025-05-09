package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time" // เพิ่ม import สำหรับ timesta

	"github.com/google/uuid"

	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v2"
	pngquant "github.com/yusukebe/go-pngquant"
	"golang.org/x/sync/semaphore"
)

func Upload(c *fiber.Ctx) error {
	start := time.Now()
	fmt.Printf("Start receiving request at %v from %s\n", start, c.Get("Origin"))

	form, err := c.MultipartForm()
	if err != nil {
		fmt.Printf("Cannot parse multipart form: %v\n", err)
		return c.Status(400).SendString("Upload failed: " + err.Error())
	}
	fmt.Printf("Parse multipart form took %v\n", time.Since(start))

	files := form.File["image"]
	if len(files) == 0 {
		return c.Status(400).SendString("No images uploaded")
	}
	if len(files) > 10 {
		return c.Status(400).SendString("Too many files uploaded (max 10)")
	}

	userAgent := c.Get("User-Agent")
	var urls []string
	var mu sync.Mutex

	sem := semaphore.NewWeighted(2)
	ctx := context.Background()
	errors := make(chan error, len(files))

	for i, file := range files {
		if err := sem.Acquire(ctx, 1); err != nil {
			fmt.Printf("Cannot acquire semaphore: %v\n", err)
			return c.Status(500).SendString("Server busy")
		}

		go func(i int, file *multipart.FileHeader) {
			defer sem.Release(1)
			loopStart := time.Now()
			originalName := file.Filename
			newName := fmt.Sprintf("%s_%d_%s.png", time.Now().Format("20060102_150405"), i, uuid.New().String()[0:6])
			savePath := filepath.Join("./uploads", newName)

			fileData, err := file.Open()
			if err != nil {
				errors <- fmt.Errorf("Cannot read file %s: %v", originalName, err)
				return
			}
			defer fileData.Close()

			outFile, err := os.Create(savePath)
			if err != nil {
				errors <- fmt.Errorf("Cannot create file %s: %v", originalName, err)
				return
			}
			defer outFile.Close()

			saveStart := time.Now()
			_, err = io.Copy(outFile, fileData)
			if err != nil {
				errors <- fmt.Errorf("Cannot save file %s: %v", originalName, err)
				return
			}
			fmt.Printf("Save file %s took %v\n", newName, time.Since(saveStart))

			fmt.Printf("UPLOAD [%s] - UA: [%s] - original: %s => saved: %s\n",
				time.Now().Format(time.RFC3339), userAgent, originalName, newName)

			url := "https://www.ymt-group.com/pos-image/" + newName
			mu.Lock()
			urls = append(urls, url)
			mu.Unlock()
			fmt.Printf("Loop %d took %v\n", i, time.Since(loopStart))
			errors <- nil
		}(i, file)
	}

	// รอผลลัพธ์จากทุก goroutine พร้อม timeout
	timeout := time.After(30 * time.Second)
	for i := 0; i < len(files); i++ {
		select {
		case err := <-errors:
			if err != nil {
				return c.Status(500).SendString(err.Error())
			}
		case <-timeout:
			return c.Status(500).SendString("Upload timeout: operation took too long")
		}
	}

	fmt.Printf("Total upload took %v\n", time.Since(start))
	return c.Status(200).JSON(fiber.Map{
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
