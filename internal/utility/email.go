package utility

import (
	"bytes"
	"chronoverseapi/internal/model"
	"html/template"
	"log/slog"
)

func GenerateEmailBody(templatePath string, data *model.EmailBodyData) (string, error) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		slog.Error("error loading email template: ", err.Error())
		return "", err
	}
	var bodyContent bytes.Buffer
	err = tmpl.Execute(&bodyContent, data)
	if err != nil {
		slog.Error("error rendering email template: ", err.Error())
		return "", err
	}
	return bodyContent.String(), nil
}
