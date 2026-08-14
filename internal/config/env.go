package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ApplyEnvOverrides applies environment variables to the configuration where applicable.
func ApplyEnvOverrides(cfg *Config) {
	if folderID := strings.TrimSpace(os.Getenv("GDRIVE_FOLDER_ID")); folderID != "" {
		cfg.Drive.FolderID = folderID
	}
}

// LoadEnvFile reads a .env file and sets environment variables for the current process.
// It will not overwrite existing environment variables already present in the environment.
func LoadEnvFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Support "export KEY=VALUE" prefix
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Strip surrounding single or double quotes
		if len(val) >= 2 {
			if (strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`)) ||
				(strings.HasPrefix(val, `'`) && strings.HasSuffix(val, `'`)) {
				val = val[1 : len(val)-1]
			}
		}

		if key != "" {
			// Only set if not already present in environment
			if _, exists := os.LookupEnv(key); !exists {
				if err := os.Setenv(key, val); err != nil {
					return fmt.Errorf("failed setting env var %s on line %d: %w", key, lineNum, err)
				}
			}
		}
	}

	return scanner.Err()
}
