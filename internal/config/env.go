package config

import (
	"os"
	"strings"
)

// ApplyEnvOverrides applies environment variables to the configuration where applicable.
func ApplyEnvOverrides(cfg *Config) {
	if folderID := strings.TrimSpace(os.Getenv("GDRIVE_FOLDER_ID")); folderID != "" {
		cfg.Drive.FolderID = folderID
	}
}
