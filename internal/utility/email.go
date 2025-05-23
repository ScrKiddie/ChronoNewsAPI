package utility

import (
	"bytes"
	"chrononewsapi/internal/model"
	"embed"
	"html/template"
	"log/slog"
)

func GenerateEmailBody(fs embed.FS, templatePath string, data *model.EmailBodyData) (string, error) {
	tmpl, err := template.ParseFS(fs, templatePath)
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
