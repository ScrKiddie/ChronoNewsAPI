package utility

import (
	"bytes"
	"encoding/base64"
	"github.com/google/uuid"
	"image"
	"image/jpeg"
	"mime/multipart"
	"os"
	"path/filepath"
)

func CreateFileName(file *multipart.FileHeader) string {
	return uuid.New().String() + filepath.Ext(file.Filename)
}

func WriteBase64File(base64File string) ([]byte, error) {
	file, err := base64.StdEncoding.DecodeString(base64File)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func CompressImage(file []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(file))
	if err != nil {
		return "", err
	}

	tempFile, err := os.CreateTemp("", "tempImage-*.jpg")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	options := jpeg.Options{Quality: 80}
	var imgBuffer bytes.Buffer
	err = jpeg.Encode(&imgBuffer, img, &options)
	if err != nil {
		return "", err
	}

	maxSize := 500 * 1024
	for imgBuffer.Len() > maxSize && options.Quality > 10 {
		options.Quality -= 10
		imgBuffer.Reset()
		err = jpeg.Encode(&imgBuffer, img, &options)
		if err != nil {
			return "", err
		}
	}

	_, err = tempFile.Write(imgBuffer.Bytes())
	if err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}
