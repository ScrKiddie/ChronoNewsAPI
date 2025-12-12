package utility

import (
	"chrononewsapi/internal/config"
	"fmt"
	"path/filepath"
	"strings"
)

func BuildImageURL(cfg *config.Config, folderPathFromConfig string, fileName string) string {
	if fileName == "" {
		return ""
	}

	var baseURL string
	if cfg.Storage.Mode == "local" {
		baseURL = cfg.Web.BaseURL
	} else {
		baseURL = cfg.Storage.CdnURL
	}

	cleanInput := strings.TrimLeft(folderPathFromConfig, "/\\.")

	cleanPath := filepath.ToSlash(filepath.Join(".", cleanInput))

	return fmt.Sprintf("%s/%s/%s", baseURL, cleanPath, fileName)
}
