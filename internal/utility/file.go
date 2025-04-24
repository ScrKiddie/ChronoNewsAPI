package utility

import (
	"bytes"
	"chrononewsapi/internal/model"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/h2non/bimg"
	"image"
	"math"
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

func CompressImage(fileData model.FileData, tempDir string) (string, error) {
	size, err := bimg.Size(fileData.File)
	if err != nil {
		return "", fmt.Errorf("gagal membaca ukuran gambar: %v", err)
	}

	width, height := CalculateOptimalSize(size.Width, size.Height)
	options := bimg.Options{
		Type:         bimg.WEBP,
		Compression:  6,
		Width:        width,
		Height:       height,
		Force:        true,
		NoAutoRotate: true,
	}

	qualityLevels := []int{80, 70, 50, 30, 20, 10}
	var outputImage []byte

	for _, quality := range qualityLevels {
		options.Quality = quality
		outputImage, err = bimg.NewImage(fileData.File).Process(options)
		if err != nil {
			return "", fmt.Errorf("gagal memproses gambar: %v", err)
		}

		if len(outputImage) <= maxSize {
			break
		}
	}

	outputPath := filepath.Join(tempDir, fileData.Name)
	err = os.WriteFile(outputPath, outputImage, 0644)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan file: %v", err)
	}

	return filepath.Base(outputPath), nil
}

func CalculateOptimalSize(originalWidth, originalHeight int) (int, int) {
	if originalWidth <= maxWidth && originalHeight <= maxHeight {
		return originalWidth, originalHeight
	}

	ratio := math.Min(
		float64(maxWidth)/float64(originalWidth),
		float64(maxHeight)/float64(originalHeight),
	)

	newWidth := int(math.Round(float64(originalWidth) * ratio))
	newHeight := int(math.Round(float64(originalHeight) * ratio))

	return newWidth, newHeight
}
