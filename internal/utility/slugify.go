package utility

import (
	"regexp"
	"strings"
)

func Slugify(text string) string {
	if text == "" {
		return ""
	}
	slug := strings.ToLower(text)
	slug = strings.TrimSpace(slug)
	slug = regexp.MustCompile(`\s+`).ReplaceAllString(slug, "-")
	slug = regexp.MustCompile(`[^\w-]+`).ReplaceAllString(slug, "")
	slug = regexp.MustCompile(`--+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}
