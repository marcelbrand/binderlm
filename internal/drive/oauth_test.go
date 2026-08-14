package drive

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestTokenStorage_Lifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "binderlm", "token.json")

	storage := NewTokenStorage(tokenPath)

	if storage.Exists() {
		t.Fatal("expected token to not exist initially")
	}

	testToken := &oauth2.Token{
		AccessToken:  "test-access-token-12345",
		TokenType:    "Bearer",
		RefreshToken: "test-refresh-token-67890",
		Expiry:       time.Now().Add(1 * time.Hour).Truncate(time.Second),
	}

	// 1. Save
	if err := storage.Save(testToken); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	if !storage.Exists() {
		t.Fatal("expected token file to exist after save")
	}

	// Check file permissions
	info, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("failed to stat token file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected permissions 0600, got %o", perm)
	}

	// 2. Load
	loaded, err := storage.Load()
	if err != nil {
		t.Fatalf("failed to load token: %v", err)
	}

	if loaded.AccessToken != testToken.AccessToken {
		t.Errorf("expected access token %s, got %s", testToken.AccessToken, loaded.AccessToken)
	}
	if loaded.RefreshToken != testToken.RefreshToken {
		t.Errorf("expected refresh token %s, got %s", testToken.RefreshToken, loaded.RefreshToken)
	}

	// 3. Delete
	if err := storage.Delete(); err != nil {
		t.Fatalf("failed to delete token: %v", err)
	}

	if storage.Exists() {
		t.Fatal("expected token file to not exist after delete")
	}
}

func TestGetAuthStatus_ServiceAccountJSON(t *testing.T) {
	orig := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")
	defer os.Setenv("GOOGLE_APPLICATION_CREDENTIALS_JSON", orig)

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS_JSON", `{"client_email":"test-sa@project.iam.gserviceaccount.com"}`)

	status := GetAuthStatus(context.Background())
	if status.Type != AuthTypeServiceAccountJSON {
		t.Errorf("expected %s, got %s", AuthTypeServiceAccountJSON, status.Type)
	}
	if status.Account != "test-sa@project.iam.gserviceaccount.com" {
		t.Errorf("unexpected account: %s", status.Account)
	}
}

func TestGetAuthStatus_Unauthenticated(t *testing.T) {
	t.Setenv("BINDERLM_CONFIG_DIR", t.TempDir())
	t.Setenv("BINDERLM_TOKEN_FILE", filepath.Join(t.TempDir(), "nonexistent.json"))
	origJSON := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")
	origFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	defer func() {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS_JSON", origJSON)
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", origFile)
	}()

	os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")
	os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")

	status := GetAuthStatus(context.Background())
	if status.Type != AuthTypeNone && status.Type != AuthTypeADC {
		t.Errorf("expected None or ADC, got %s", status.Type)
	}
}

func TestAuthMode_Selection(t *testing.T) {
	if NormalizeAuthMode("user") != "user" || NormalizeAuthMode("oauth") != "user" || NormalizeAuthMode("personal") != "user" {
		t.Errorf("failed normalizing user auth mode")
	}
	if NormalizeAuthMode("sa") != "sa" || NormalizeAuthMode("service_account") != "sa" || NormalizeAuthMode("service-account") != "sa" {
		t.Errorf("failed normalizing sa auth mode")
	}
	if NormalizeAuthMode("auto") != "auto" || NormalizeAuthMode("unknown") != "auto" {
		t.Errorf("failed normalizing auto auth mode")
	}

	ctxUser := WithAuthMode(context.Background(), "user")
	if GetAuthModeFromContext(ctxUser) != "user" {
		t.Errorf("expected user from context")
	}

	ctxSA := WithAuthMode(context.Background(), "sa")
	if GetAuthModeFromContext(ctxSA) != "sa" {
		t.Errorf("expected sa from context")
	}
}

