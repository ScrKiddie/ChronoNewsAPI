package utility

import (
	"strings"
)

func NormalizeEmail(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))

	localPart, domainPart, found := strings.Cut(email, "@")
	if !found {
		return email
	}

	if idx := strings.Index(localPart, "+"); idx != -1 {
		localPart = localPart[:idx]
	}

	if domainPart == "gmail.com" || domainPart == "googlemail.com" {
		domainPart = "gmail.com"
		localPart = strings.ReplaceAll(localPart, ".", "")
	}

	return localPart + "@" + domainPart
}
