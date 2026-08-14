package drive

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetConfigDir returns the base directory for storing binderlm global configurations and tokens.
// Precedence:
// 1. BINDERLM_CONFIG_DIR environment variable (if specified)
// 2. ~/.config/binderlm
// 3. os.UserConfigDir()/binderlm fallback
func GetConfigDir() string {
	if dir := os.Getenv("BINDERLM_CONFIG_DIR"); dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".config", "binderlm")
	}

	userConfig, err := os.UserConfigDir()
	if err == nil && userConfig != "" {
		return filepath.Join(userConfig, "binderlm")
	}

	return ".binderlm"
}

// GetClientCredentialsPath returns the path to client.json.
func GetClientCredentialsPath() string {
	return filepath.Join(GetConfigDir(), "client.json")
}

// GetServiceAccountPath returns the path to service_account.json.
func GetServiceAccountPath() string {
	return filepath.Join(GetConfigDir(), "service_account.json")
}

// ClientCredentials holds OAuth client id and secret.
type ClientCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// GoogleOAuthClientJSON represents the raw format exported from Google Cloud Console.
type GoogleOAuthClientJSON struct {
	Installed *ClientCredentials `json:"installed"`
	Web       *ClientCredentials `json:"web"`
	ClientID     string          `json:"client_id"`
	ClientSecret string          `json:"client_secret"`
}

// LoadClientCredentials loads OAuth client credentials from disk or environment.
func LoadClientCredentials(cliID, cliSecret string) (string, string, error) {
	// 1. Direct CLI flags
	if cliID != "" && cliSecret != "" {
		return cliID, cliSecret, nil
	}

	// 2. Environment variables
	envID := strings.TrimSpace(os.Getenv("GDRIVE_OAUTH_CLIENT_ID"))
	envSecret := strings.TrimSpace(os.Getenv("GDRIVE_OAUTH_CLIENT_SECRET"))
	if cliID == "" {
		cliID = envID
	}
	if cliSecret == "" {
		cliSecret = envSecret
	}
	if cliID != "" && cliSecret != "" {
		return cliID, cliSecret, nil
	}

	// 3. ~/.config/binderlm/client.json
	path := GetClientCredentialsPath()
	if data, err := os.ReadFile(path); err == nil {
		var gCreds GoogleOAuthClientJSON
		if err := json.Unmarshal(data, &gCreds); err == nil {
			if gCreds.Installed != nil && gCreds.Installed.ClientID != "" {
				if cliID == "" {
					cliID = gCreds.Installed.ClientID
				}
				if cliSecret == "" {
					cliSecret = gCreds.Installed.ClientSecret
				}
			} else if gCreds.Web != nil && gCreds.Web.ClientID != "" {
				if cliID == "" {
					cliID = gCreds.Web.ClientID
				}
				if cliSecret == "" {
					cliSecret = gCreds.Web.ClientSecret
				}
			} else if gCreds.ClientID != "" {
				if cliID == "" {
					cliID = gCreds.ClientID
				}
				if cliSecret == "" {
					cliSecret = gCreds.ClientSecret
				}
			}
		}
	}

	// 4. Default fallback if still empty
	if cliID == "" {
		cliID = DefaultClientID
	}
	if cliSecret == "" {
		cliSecret = DefaultClientSecret
	}

	return cliID, cliSecret, nil
}

// SaveClientCredentials writes client credentials to ~/.config/binderlm/client.json.
func SaveClientCredentials(clientID, clientSecret string) error {
	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	creds := ClientCredentials{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode client credentials: %w", err)
	}

	path := GetClientCredentialsPath()
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write client credentials to %s: %w", path, err)
	}

	return nil
}

// SaveServiceAccountKey writes a service account key to ~/.config/binderlm/service_account.json.
func SaveServiceAccountKey(jsonData []byte) error {
	var js map[string]interface{}
	if err := json.Unmarshal(jsonData, &js); err != nil {
		return fmt.Errorf("invalid Service Account JSON: %w", err)
	}

	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	path := GetServiceAccountPath()
	if err := os.WriteFile(path, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write service account key to %s: %w", path, err)
	}

	return nil
}
