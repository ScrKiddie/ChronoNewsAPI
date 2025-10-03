package config

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

func NewValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.RegisterValidation("exclusiveor", ExclusiveOr); err != nil {
		slog.Error("Failed to register 'exclusiveor' validator", "err", err)
		os.Exit(1)
	}
	if err := v.RegisterValidation("image", Image); err != nil {
		slog.Error("Failed to register 'image' validator", "err", err)
		os.Exit(1)
	}
	if err := v.RegisterValidation("passwordformat", PasswordFormat); err != nil {
		slog.Error("Failed to register 'passwordformat' validator", "err", err)
		os.Exit(1)
	}
	if err := v.RegisterValidation("uniquemembers", UniqueMembers); err != nil {
		slog.Error("Failed to register 'uniquemembers' validator", "err", err)
		os.Exit(1)
	}
	return v
}

func ExclusiveOr(fl validator.FieldLevel) bool {
	param := fl.Param()
	field1 := fl.Field().String()
	field2 := fl.Parent().FieldByName(param).String()
	if (field1 != "" && field2 != "") || (field1 == "" && field2 == "") {
		return false
	}
	return true
}

func Image(fl validator.FieldLevel) bool {

	allowedExtensions := []string{".png", ".jpg", ".jpeg", ".jpe", ".jfif", ".jif", ".jfi"}
	allowedContentTypes := []string{"image/png", "image/jpeg", "image/pjpeg", "image/apng"}
	var defaultMaxSize int64 = 2

	defaultMaxWidth, defaultMaxHeight := 800, 800

	params := fl.Param()
	maxWidth, maxHeight := defaultMaxWidth, defaultMaxHeight
	maxSize := defaultMaxSize

	if params != "" {
		parts := strings.Split(params, "_")
		if len(parts) >= 2 {
			if w, err := strconv.Atoi(parts[0]); err == nil {
				maxWidth = w
			}
			if h, err := strconv.Atoi(parts[1]); err == nil {
				maxHeight = h
			}
			if len(parts) == 3 {
				if s, err := strconv.ParseInt(parts[2], 10, 64); err == nil {
					maxSize = s
				}
			}
		}
	}

	file, ok := fl.Field().Interface().(multipart.FileHeader)
	if !ok {
		return false
	}

	if file.Size > maxSize*1024*1024 {
		return false
	}

	extension := filepath.Ext(file.Filename)
	for i, allowedExtension := range allowedExtensions {
		if extension == allowedExtension {
			break
		} else if i == len(allowedExtensions)-1 {
			return false
		}
	}

	fileOpened, err := file.Open()
	if err != nil {
		slog.Error("Failed to open file for image validation", "err", err)
		return false
	}
	defer func(fileOpened multipart.File) {
		err := fileOpened.Close()
		if err != nil {
			slog.Error("Failed to close file after image validation", "err", err)
		}
	}(fileOpened)

	fileHeader := make([]byte, 512)
	if _, err := fileOpened.Read(fileHeader); err != nil {
		slog.Error("Failed to read file header for image validation", "err", err)
		return false
	}

	contentType := http.DetectContentType(fileHeader)
	for i, allowedContentType := range allowedContentTypes {
		if contentType == allowedContentType {
			break
		} else if i == len(allowedContentTypes)-1 {
			return false
		}
	}

	if _, err := fileOpened.Seek(0, 0); err != nil {
		slog.Error("Failed to seek file for image validation", "err", err)
		return false
	}

	img, _, err := image.DecodeConfig(fileOpened)
	if err != nil {
		return false
	}

	if img.Width > maxWidth || img.Height > maxHeight {
		return false
	}

	return true
}

func PasswordFormat(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#\$%\^&\*\(\)_\+\-=\[\]\{\};:'",.<>\/?\\|~]`).MatchString(password)
	return hasLower && hasUpper && hasNumber && hasSpecial
}

func UniqueMembers(fl validator.FieldLevel) bool {
	slice := fl.Field().Interface().([]int32)
	current := reflect.ValueOf(fl.Parent().Interface()).FieldByName("ID").Int()

	seen := make(map[int32]bool)

	for _, value := range slice {
		if value == int32(current) {
			return false
		}
		if seen[value] {
			return false
		}
		seen[value] = true
	}

	return true
}
