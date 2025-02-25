package utility

import (
	"bytes"
	"chronoverseapi/internal/model"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/chai2010/webp"
	"github.com/google/uuid"
	"image"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

const maxSize = 500 * 1024 // 500KB

func CreateFileName(file *multipart.FileHeader) string {
	return uuid.New().String() + filepath.Ext(file.Filename)
}

func Base64ToFile(base64File string) ([]byte, string, error) {
	if idx := strings.Index(base64File, ";base64,"); idx != -1 {
		base64File = base64File[idx+8:]
	}

	decodedFile, err := base64.StdEncoding.DecodeString(base64File)
	if err != nil {
		return nil, "", errors.New("gagal mendekode base64")
	}

	imgCfg, format, err := image.DecodeConfig(bytes.NewReader(decodedFile))
	if err != nil || imgCfg.Width == 0 || imgCfg.Height == 0 {
		return nil, "", errors.New("file bukan gambar yang valid")
	}

	if format != "jpeg" && format != "png" {
		return nil, "", fmt.Errorf("format gambar tidak didukung, hanya JPEG dan PNG yang diizinkan")
	}

	fileName := uuid.New().String() + ".webp"

	return decodedFile, fileName, nil
}

func CompressImage(fileData model.FileData, tempDir string) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(fileData.File))
	if err != nil {
		return "", fmt.Errorf("gagal mendekode gambar: %v", err)
	}

	outputPath := filepath.Join(tempDir, fileData.Name)

	tempFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("gagal membuat file sementara: %v", err)
	}
	defer tempFile.Close()

	quality := 90
	var imgBuffer bytes.Buffer

	for quality >= 10 {
		imgBuffer.Reset()
		err = webp.Encode(&imgBuffer, img, &webp.Options{Quality: float32(quality)})
		if err != nil {
			return "", fmt.Errorf("gagal mengompresi gambar: %v", err)
		}
		if imgBuffer.Len() < maxSize {
			_, err = tempFile.Write(imgBuffer.Bytes())
			if err != nil {
				return "", fmt.Errorf("gagal menyimpan file WebP: %v", err)
			}
			return filepath.Base(tempFile.Name()), nil
		}
		quality -= 10
	}

	_, err = tempFile.Write(imgBuffer.Bytes())
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan file WebP: %v", err)
	}

	return filepath.Base(tempFile.Name()), nil
}
