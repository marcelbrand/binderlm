package drive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// AuthError represents a failure in resolving or validating Google authentication credentials.
type AuthError struct {
	Message string
	Err     error
}

func (e *AuthError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AuthError) Unwrap() error {
	return e.Err
}

// IsAuthError checks if an error is or wraps an AuthError.
func IsAuthError(err error) bool {
	var authErr *AuthError
	return errors.As(err, &authErr)
}

// AuthType represents the resolved authentication method.
type AuthType string

const (
	AuthTypeServiceAccountJSON AuthType = "service_account_json"
	AuthTypeServiceAccountFile AuthType = "service_account_file"
	AuthTypeOAuthUser          AuthType = "oauth_user"
	AuthTypeADC                AuthType = "application_default_credentials"
	AuthTypeNone               AuthType = "none"
)

// AuthStatus encapsulates the current active credentials configuration.
type AuthStatus struct {
	Type        AuthType
	Account     string // Email address or principal name if resolvable
	Description string // Human-readable description of source
	TokenFile   string // Path to token file if OAuth
}

// ResolveClientOption inspects environment variables and local environment to find Google credentials.
// Precedence:
// 1. GOOGLE_APPLICATION_CREDENTIALS_JSON (in-memory raw JSON string, ideal for CI/CD)
// 2. GOOGLE_APPLICATION_CREDENTIALS (file path to service account JSON)
// 3. Cached OAuth User Token (~/.config/binderlm/token.json via `binderlm login`)
// 4. Application Default Credentials (ADC)
func ResolveClientOption(ctx context.Context) (option.ClientOption, error) {
	// 1. In-memory JSON
	if jsonCreds := strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")); jsonCreds != "" {
		var js map[string]interface{}
		if err := json.Unmarshal([]byte(jsonCreds), &js); err != nil {
			return nil, &AuthError{
				Message: "invalid JSON in GOOGLE_APPLICATION_CREDENTIALS_JSON",
				Err:     err,
			}
		}
		return option.WithCredentialsJSON([]byte(jsonCreds)), nil
	}

	// 2. Credentials file path
	if credPath := strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")); credPath != "" {
		if _, err := os.Stat(credPath); err != nil {
			return nil, &AuthError{
				Message: fmt.Sprintf("credentials file not found at %s", credPath),
				Err:     err,
			}
		}
		return option.WithCredentialsFile(credPath), nil
	}

	// 3. Cached Developer OAuth Token
	storage := NewTokenStorage()
	if storage.Exists() {
		mgr := NewOAuthManager("", "", storage)
		ts, err := mgr.GetTokenSource(ctx)
		if err == nil && ts != nil {
			// Verify token works
			if _, tErr := ts.Token(); tErr == nil {
				return option.WithTokenSource(ts), nil
			}
		}
	}

	// 4. Fallback to ADC
	creds, err := google.FindDefaultCredentials(ctx, drive.DriveScope)
	if err == nil && creds != nil {
		return option.WithCredentials(creds), nil
	}

	return nil, &AuthError{
		Message: "no Google Drive credentials found. Run 'binderlm login' for personal accounts, or set GOOGLE_APPLICATION_CREDENTIALS_JSON / GOOGLE_APPLICATION_CREDENTIALS for service accounts",
		Err:     err,
	}
}

// GetAuthStatus inspects the active authentication method without failing immediately if unauthenticated.
func GetAuthStatus(ctx context.Context) AuthStatus {
	// 1. In-memory JSON
	if jsonCreds := strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")); jsonCreds != "" {
		var js struct {
			ClientEmail string `json:"client_email"`
		}
		_ = json.Unmarshal([]byte(jsonCreds), &js)
		account := js.ClientEmail
		if account == "" {
			account = "Service Account (in-memory JSON)"
		}
		return AuthStatus{
			Type:        AuthTypeServiceAccountJSON,
			Account:     account,
			Description: "Google Cloud Service Account (GOOGLE_APPLICATION_CREDENTIALS_JSON)",
		}
	}

	// 2. Credentials file path
	if credPath := strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")); credPath != "" {
		account := credPath
		if data, err := os.ReadFile(credPath); err == nil {
			var js struct {
				ClientEmail string `json:"client_email"`
			}
			if jErr := json.Unmarshal(data, &js); jErr == nil && js.ClientEmail != "" {
				account = js.ClientEmail
			}
		}
		return AuthStatus{
			Type:        AuthTypeServiceAccountFile,
			Account:     account,
			Description: fmt.Sprintf("Google Cloud Service Account File (%s)", credPath),
		}
	}

	// 3. Cached Developer OAuth Token
	storage := NewTokenStorage()
	if storage.Exists() {
		tokenPath, _ := storage.TokenPath()
		mgr := NewOAuthManager("", "", storage)
		ts, err := mgr.GetTokenSource(ctx)
		if err == nil && ts != nil {
			email, err := FetchUserInfo(ctx, ts)
			if err == nil && email != "" {
				return AuthStatus{
					Type:        AuthTypeOAuthUser,
					Account:     email,
					Description: "Personal Google Account (Interactive OAuth Login)",
					TokenFile:   tokenPath,
				}
			}
			return AuthStatus{
				Type:        AuthTypeOAuthUser,
				Account:     "Authenticated User",
				Description: "Personal Google Account (Interactive OAuth Login)",
				TokenFile:   tokenPath,
			}
		}
	}

	// 4. ADC
	creds, err := google.FindDefaultCredentials(ctx, drive.DriveScope)
	if err == nil && creds != nil {
		return AuthStatus{
			Type:        AuthTypeADC,
			Account:     "Application Default Credentials",
			Description: "Google Application Default Credentials (gcloud/ADC)",
		}
	}

	return AuthStatus{
		Type:        AuthTypeNone,
		Account:     "",
		Description: "Unauthenticated (Run 'binderlm login' or configure Service Account)",
	}
}
