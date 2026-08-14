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

type authModeKey struct{}

// WithAuthMode returns a context enriched with the desired authentication mode ("user", "sa", "auto").
func WithAuthMode(ctx context.Context, mode string) context.Context {
	return context.WithValue(ctx, authModeKey{}, NormalizeAuthMode(mode))
}

// GetAuthModeFromContext retrieves the configured auth mode from context, environment, or default "auto".
func GetAuthModeFromContext(ctx context.Context) string {
	if v := ctx.Value(authModeKey{}); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	if envMode := strings.TrimSpace(os.Getenv("BINDERLM_AUTH_MODE")); envMode != "" {
		return NormalizeAuthMode(envMode)
	}
	return "auto"
}

// NormalizeAuthMode normalizes aliases for auth modes.
func NormalizeAuthMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "user", "oauth", "personal":
		return "user"
	case "sa", "service_account", "serviceaccount", "service-account":
		return "sa"
	default:
		return "auto"
	}
}

// ResolveClientOption inspects environment variables, local files, and auth mode to return client options.
func ResolveClientOption(ctx context.Context) (option.ClientOption, error) {
	mode := GetAuthModeFromContext(ctx)

	switch mode {
	case "user":
		return resolveUserOAuth(ctx)
	case "sa":
		return resolveServiceAccount(ctx)
	default:
		return resolveAuto(ctx)
	}
}

func resolveUserOAuth(ctx context.Context) (option.ClientOption, error) {
	storage := NewTokenStorage()
	if !storage.Exists() {
		return nil, &AuthError{
			Message: "no cached OAuth user token found. Run 'binderlm login' or 'binderlm setup' to authenticate your personal Google account",
		}
	}

	mgr := NewOAuthManager("", "", storage)
	ts, err := mgr.GetTokenSource(ctx)
	if err != nil {
		return nil, &AuthError{
			Message: "failed to load OAuth token source",
			Err:     err,
		}
	}

	if _, tErr := ts.Token(); tErr != nil {
		return nil, &AuthError{
			Message: "cached OAuth token is invalid or expired. Run 'binderlm login' to re-authenticate",
			Err:     tErr,
		}
	}

	return option.WithTokenSource(ts), nil
}

func resolveServiceAccount(ctx context.Context) (option.ClientOption, error) {
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

	// 2. Credentials file path from environment variable
	if credPath := strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")); credPath != "" {
		if _, err := os.Stat(credPath); err != nil {
			return nil, &AuthError{
				Message: fmt.Sprintf("credentials file not found at %s", credPath),
				Err:     err,
			}
		}
		return option.WithCredentialsFile(credPath), nil
	}

	// 3. Global Service Account file in ~/.config/binderlm/service_account.json
	saPath := GetServiceAccountPath()
	if _, err := os.Stat(saPath); err == nil {
		return option.WithCredentialsFile(saPath), nil
	}

	return nil, &AuthError{
		Message: "no Service Account credentials found. Set GOOGLE_APPLICATION_CREDENTIALS_JSON, GOOGLE_APPLICATION_CREDENTIALS, or run 'binderlm setup'",
	}
}

func resolveAuto(ctx context.Context) (option.ClientOption, error) {
	// 1. In-memory JSON (CI/CD always takes top priority)
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

	// 2. Credentials file path from environment variable
	if credPath := strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")); credPath != "" {
		if _, err := os.Stat(credPath); err != nil {
			return nil, &AuthError{
				Message: fmt.Sprintf("credentials file not found at %s", credPath),
				Err:     err,
			}
		}
		return option.WithCredentialsFile(credPath), nil
	}

	// 3. Cached Developer OAuth Token (Preferred on local machines where developer logged in)
	storage := NewTokenStorage()
	if storage.Exists() {
		mgr := NewOAuthManager("", "", storage)
		ts, err := mgr.GetTokenSource(ctx)
		if err == nil && ts != nil {
			if _, tErr := ts.Token(); tErr == nil {
				return option.WithTokenSource(ts), nil
			}
		}
	}

	// 4. Global Service Account in ~/.config/binderlm/service_account.json
	saPath := GetServiceAccountPath()
	if _, err := os.Stat(saPath); err == nil {
		return option.WithCredentialsFile(saPath), nil
	}

	// 5. Fallback to ADC
	creds, err := google.FindDefaultCredentials(ctx, drive.DriveScope)
	if err == nil && creds != nil {
		return option.WithCredentials(creds), nil
	}

	return nil, &AuthError{
		Message: "no Google Drive credentials found. Run 'binderlm setup' / 'binderlm login' for personal accounts, or set GOOGLE_APPLICATION_CREDENTIALS_JSON / GOOGLE_APPLICATION_CREDENTIALS for service accounts",
		Err:     err,
	}
}

// GetAuthStatus inspects the active authentication method without failing immediately if unauthenticated.
func GetAuthStatus(ctx context.Context) AuthStatus {
	mode := GetAuthModeFromContext(ctx)

	if mode == "user" {
		storage := NewTokenStorage()
		if storage.Exists() {
			tokenPath, _ := storage.TokenPath()
			mgr := NewOAuthManager("", "", storage)
			ts, err := mgr.GetTokenSource(ctx)
			if err == nil && ts != nil {
				email, _ := FetchUserInfo(ctx, ts)
				if email == "" {
					email = "Authenticated User"
				}
				return AuthStatus{
					Type:        AuthTypeOAuthUser,
					Account:     email,
					Description: "Personal Google Account (Interactive OAuth Login)",
					TokenFile:   tokenPath,
				}
			}
		}
		return AuthStatus{
			Type:        AuthTypeNone,
			Description: "Unauthenticated (No personal OAuth token found. Run 'binderlm login')",
		}
	}

	if mode == "sa" {
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

		saPath := GetServiceAccountPath()
		if data, err := os.ReadFile(saPath); err == nil {
			account := saPath
			var js struct {
				ClientEmail string `json:"client_email"`
			}
			if jErr := json.Unmarshal(data, &js); jErr == nil && js.ClientEmail != "" {
				account = js.ClientEmail
			}
			return AuthStatus{
				Type:        AuthTypeServiceAccountFile,
				Account:     account,
				Description: "Global Service Account (~/.config/binderlm/service_account.json)",
			}
		}

		return AuthStatus{
			Type:        AuthTypeNone,
			Description: "Unauthenticated (No Service Account credentials found)",
		}
	}

	// Auto mode
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

	// 3. Cached Developer OAuth Token (Preferred locally)
	storage := NewTokenStorage()
	if storage.Exists() {
		tokenPath, _ := storage.TokenPath()
		mgr := NewOAuthManager("", "", storage)
		ts, err := mgr.GetTokenSource(ctx)
		if err == nil && ts != nil {
			email, _ := FetchUserInfo(ctx, ts)
			if email == "" {
				email = "Authenticated User"
			}
			return AuthStatus{
				Type:        AuthTypeOAuthUser,
				Account:     email,
				Description: "Personal Google Account (Interactive OAuth Login)",
				TokenFile:   tokenPath,
			}
		}
	}

	// 4. Global Service Account file
	saPath := GetServiceAccountPath()
	if data, err := os.ReadFile(saPath); err == nil {
		account := saPath
		var js struct {
			ClientEmail string `json:"client_email"`
		}
		if jErr := json.Unmarshal(data, &js); jErr == nil && js.ClientEmail != "" {
			account = js.ClientEmail
		}
		return AuthStatus{
			Type:        AuthTypeServiceAccountFile,
			Account:     account,
			Description: "Global Service Account (~/.config/binderlm/service_account.json)",
		}
	}

	// 5. ADC
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
		Description: "Unauthenticated (Run 'binderlm setup' / 'binderlm login' or configure Service Account)",
	}
}
