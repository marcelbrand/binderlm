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

// ResolveClientOption inspects environment variables and local environment to find Google credentials.
// Precedence:
// 1. GOOGLE_APPLICATION_CREDENTIALS_JSON (in-memory raw JSON string)
// 2. GOOGLE_APPLICATION_CREDENTIALS (file path)
// 3. Application Default Credentials (ADC)
func ResolveClientOption(ctx context.Context) (option.ClientOption, error) {
	// 1. In-memory JSON
	if jsonCreds := strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")); jsonCreds != "" {
		// Basic JSON validation
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

	// 3. Fallback to ADC
	creds, err := google.FindDefaultCredentials(ctx, drive.DriveScope)
	if err != nil || creds == nil {
		return nil, &AuthError{
			Message: "no Google Cloud credentials found. Please set GOOGLE_APPLICATION_CREDENTIALS_JSON or GOOGLE_APPLICATION_CREDENTIALS",
			Err:     err,
		}
	}

	return option.WithCredentials(creds), nil
}
