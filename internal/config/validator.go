package config

import (
	"github.com/go-playground/validator/v10"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"reflect"
	"regexp"
)

func NewValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.RegisterValidation("exclusiveor", ExclusiveOr); err != nil {
		log.Fatalf(err.Error())
	}
	if err := v.RegisterValidation("image", Image); err != nil {
		log.Fatalf(err.Error())
	}
	if err := v.RegisterValidation("passwordformat", PasswordFormat); err != nil {
		log.Fatalf(err.Error())
	}
	if err := v.RegisterValidation("uniquemembers", UniqueMembers); err != nil {
		log.Fatalf(err.Error())
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

	allowedExtensions := []string{".png", ".jpg", ".jpeg"}
	allowedContentTypes := []string{"image/png", "image/jpeg"}
	var maxSize int64 = 1024 * 1024
	maxWidth, maxHeight := 320, 320

	file, ok := fl.Field().Interface().(multipart.FileHeader)
	if !ok {
		return false
	}

	if file.Size > maxSize {
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
		slog.Error(err.Error())
		return false
	}
	defer fileOpened.Close()

	fileHeader := make([]byte, 512)
	if _, err := fileOpened.Read(fileHeader); err != nil {
		slog.Error(err.Error())
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
		slog.Error(err.Error())
		return false
	}

	img, _, err := image.DecodeConfig(fileOpened)
	if err != nil {
		return false
	}

	if img.Width != maxWidth || img.Height != maxHeight {
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
