package drive

import (
	"os"
	"testing"
)

func TestClientCredentials_Lifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("BINDERLM_CONFIG_DIR", tmpDir)

	// Test Save
	err := SaveClientCredentials("test-client-id.apps.googleusercontent.com", "test-client-secret-123")
	if err != nil {
		t.Fatalf("SaveClientCredentials failed: %v", err)
	}

	// Verify file exists
	path := GetClientCredentialsPath()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist, got error: %v", path, err)
	}

	// Test Load
	id, secret, err := LoadClientCredentials("", "")
	if err != nil {
		t.Fatalf("LoadClientCredentials failed: %v", err)
	}

	if id != "test-client-id.apps.googleusercontent.com" {
		t.Errorf("expected client id, got %s", id)
	}
	if secret != "test-client-secret-123" {
		t.Errorf("expected secret, got %s", secret)
	}
}

func TestServiceAccount_Lifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("BINDERLM_CONFIG_DIR", tmpDir)

	validSAJSON := []byte(`{"type":"service_account","client_email":"test@proj.iam.gserviceaccount.com"}`)

	err := SaveServiceAccountKey(validSAJSON)
	if err != nil {
		t.Fatalf("SaveServiceAccountKey failed: %v", err)
	}

	path := GetServiceAccountPath()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist, got error: %v", path, err)
	}

	// Verify invalid JSON fails
	err = SaveServiceAccountKey([]byte("invalid json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestGetConfigDir(t *testing.T) {
	t.Setenv("BINDERLM_CONFIG_DIR", "/custom/config/dir")
	if dir := GetConfigDir(); dir != "/custom/config/dir" {
		t.Errorf("expected /custom/config/dir, got %s", dir)
	}
}
