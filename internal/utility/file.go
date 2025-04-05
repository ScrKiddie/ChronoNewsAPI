package utility

import (
	"bytes"
	"chronoverseapi/internal/model"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/chai2010/webp"
	"github.com/google/uuid"
	"golang.org/x/image/draw"
	"image"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

const maxSize = 500 * 1024
const maxWidth = 1980
const maxHeight = 1980

func CreateFileName(file *multipart.FileHeader) string {
	return uuid.New().String() + filepath.Ext(file.Filename)
}

func Base64ToFile(base64File string) ([]byte, string, error) {
	if idx := strings.Index(base64File, ";base64,"); idx != -1 {
		base64File = base64File[idx+8:]
	}

	decodedFile, err := base64.StdEncoding.DecodeString(base64File)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode base64: %v", err)
	}

	imgCfg, format, err := image.DecodeConfig(bytes.NewReader(decodedFile))
	if err != nil || imgCfg.Width == 0 || imgCfg.Height == 0 {
		return nil, "", errors.New("file is not a valid image")
	}

	if format != "jpeg" && format != "png" {
		return nil, "", fmt.Errorf("unsupported image format only JPEG and PNG are allowed")
	}

	fileName := uuid.New().String() + ".webp"
	return decodedFile, fileName, nil
}

func ResizeImage(img image.Image) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	if width <= maxWidth && height <= maxHeight {
		return img
	}

	ratio := float64(maxWidth) / float64(width)
	if float64(height)*ratio > float64(maxHeight) {
		ratio = float64(maxHeight) / float64(height)
	}

	newWidth := int(float64(width) * ratio)
	newHeight := int(float64(height) * ratio)
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	return dst
}

func CompressImage(fileData model.FileData, tempDir string) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(fileData.File))
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %v", err)
	}

	img = ResizeImage(img)

	outputPath := filepath.Join(tempDir, fileData.Name)
	tempFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer tempFile.Close()

	qualityLevels := []float32{90, 70, 40, 20, 10, 5, 0}
	var imgBuffer bytes.Buffer

	for _, quality := range qualityLevels {
		imgBuffer.Reset()
		err = webp.Encode(&imgBuffer, img, &webp.Options{Quality: quality})
		if err != nil {
			return "", fmt.Errorf("failed to compress image: %v", err)
		}
		if imgBuffer.Len() < maxSize {
			_, err = tempFile.Write(imgBuffer.Bytes())
			if err != nil {
				return "", fmt.Errorf("failed to save WebP file: %v", err)
			}
			return filepath.Base(tempFile.Name()), nil
		}
	}

	_, err = tempFile.Write(imgBuffer.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to save WebP file: %v", err)
	}

	return filepath.Base(tempFile.Name()), nil
}
