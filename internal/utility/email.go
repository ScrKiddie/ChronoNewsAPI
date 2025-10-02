package utility

import (
	"bytes"
	"chrononewsapi/internal/model"
	"embed"
	"html/template"
)

func GenerateEmailBody(fs embed.FS, templatePath string, data *model.EmailBodyData) (string, error) {
	tmpl, err := template.ParseFS(fs, templatePath)
	if err != nil {
		return "", err
	}

	var bodyContent bytes.Buffer
	err = tmpl.Execute(&bodyContent, data)
	if err != nil {
		return "", err
	}

	return bodyContent.String(), nil
}
