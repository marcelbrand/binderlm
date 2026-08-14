package drive_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcelbrand/binderlm/internal/drive"
	"google.golang.org/api/option"
)

func TestAuthError(t *testing.T) {
	err := &drive.AuthError{Message: "auth failed"}
	if !drive.IsAuthError(err) {
		t.Errorf("expected IsAuthError to be true")
	}
	if err.Error() != "auth failed" {
		t.Errorf("unexpected error string: %s", err.Error())
	}
}

func TestResolveClientOption_EnvJSON(t *testing.T) {
	validJSON := `{"type": "service_account", "project_id": "test-project"}`
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS_JSON", validJSON)
	defer os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")

	opt, err := drive.ResolveClientOption(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if opt == nil {
		t.Fatalf("expected non-nil option")
	}
}

func TestResolveClientOption_InvalidEnvJSON(t *testing.T) {
	invalidJSON := `not-a-json`
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS_JSON", invalidJSON)
	defer os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")

	_, err := drive.ResolveClientOption(context.Background())
	if err == nil {
		t.Fatalf("expected error for invalid JSON, got nil")
	}
	if !drive.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T", err)
	}
}

func TestResolveClientOption_EnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "sa.json")
	if err := os.WriteFile(credFile, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credFile)
	defer os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")

	opt, err := drive.ResolveClientOption(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if opt == nil {
		t.Fatalf("expected non-nil option")
	}
}

func TestResolveClientOption_MissingFile(t *testing.T) {
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/path/to/nonexistent/key.json")
	defer os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")

	_, err := drive.ResolveClientOption(context.Background())
	if err == nil {
		t.Fatalf("expected error for nonexistent file, got nil")
	}
	if !drive.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T", err)
	}
}

func TestDriveUploader_SyncCreateAndUpdate(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++

		// Query Files List
		if r.Method == http.MethodGet && (r.URL.Path == "/drive/v3/files" || r.URL.Path == "/files") {
			q := r.URL.Query().Get("q")
			if q == "" {
				t.Errorf("expected query param 'q' in list request")
			}
			// Simulate empty response on first search, found on second
			if callCount == 1 {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"files": []interface{}{},
				})
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"files": []map[string]interface{}{
						{
							"id":           "file-123",
							"name":         "test.md",
							"webViewLink":  "https://drive.google.com/file/d/file-123/view",
							"size":         "42",
							"modifiedTime": "2026-08-14T15:00:00Z",
						},
					},
				})
			}
			return
		}

		// Create file (POST)
		if r.Method == http.MethodPost && (strings.HasSuffix(r.URL.Path, "/files") || strings.Contains(r.URL.Path, "/upload")) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":           "created-456",
				"name":         "test.md",
				"webViewLink":  "https://drive.google.com/file/d/created-456/view",
				"size":         "100",
				"modifiedTime": "2026-08-14T15:01:00Z",
			})
			return
		}

		// Update file (PATCH)
		if r.Method == http.MethodPatch && (strings.HasSuffix(r.URL.Path, "/file-123") || strings.Contains(r.URL.Path, "file-123")) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":           "file-123",
				"name":         "test.md",
				"webViewLink":  "https://drive.google.com/file/d/file-123/view",
				"size":         "200",
				"modifiedTime": "2026-08-14T15:02:00Z",
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer ts.Close()

	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	uploader := drive.NewUploader(srv)

	// 1. First sync -> Should Create
	res1, err := uploader.Sync(ctx, "folder-abc", "test.md", bytes.NewReader([]byte("# New Content")))
	if err != nil {
		t.Fatalf("Sync (create) failed: %v", err)
	}
	if res1.Action != "created" || res1.FileID != "created-456" {
		t.Errorf("unexpected create result: %+v", res1)
	}

	// 2. Second sync -> Should Update (mock returns existing file)
	res2, err := uploader.Sync(ctx, "folder-abc", "test.md", bytes.NewReader([]byte("# Updated Content")))
	if err != nil {
		t.Fatalf("Sync (update) failed: %v", err)
	}
	if res2.Action != "updated" || res2.FileID != "file-123" {
		t.Errorf("unexpected update result: %+v", res2)
	}
}

func TestDriveUploader_DryRun(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && (r.URL.Path == "/drive/v3/files" || r.URL.Path == "/files") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"files": []map[string]interface{}{
					{
						"id":          "existing-file-id",
						"name":        "doc.md",
						"webViewLink": "https://drive.google.com/file/d/existing-file-id/view",
						"size":        "500",
					},
				},
			})
			return
		}
		// If any write method is called in dry run, fail the test
		t.Errorf("unexpected mutation request in dry run mode: %s %s", r.Method, r.URL.Path)
	}))
	defer ts.Close()

	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	uploader := drive.NewUploader(srv, drive.WithDryRun(true))
	res, err := uploader.Sync(ctx, "folder-xyz", "doc.md", bytes.NewReader([]byte("# Dry Run Content")))
	if err != nil {
		t.Fatalf("Dry run sync failed: %v", err)
	}

	if res.Action != "dry_run (update)" || res.FileID != "existing-file-id" {
		t.Errorf("unexpected dry run result: %+v", res)
	}
}

func TestDriveUploader_QuotaExceededFriendlyError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			// Return empty list so it attempts create
			json.NewEncoder(w).Encode(map[string]interface{}{"files": []interface{}{}})
			return
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    403,
					"message": "Service Accounts do not have storage quota. Leverage shared drives",
					"errors": []map[string]interface{}{
						{
							"reason":  "storageQuotaExceeded",
							"message": "Service Accounts do not have storage quota.",
						},
					},
				},
			})
			return
		}
	}))
	defer ts.Close()

	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	uploader := drive.NewUploader(srv)
	_, err = uploader.Sync(ctx, "folder-test-id", "doc.md", bytes.NewReader([]byte("# Content")))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "Quick Solutions") || !strings.Contains(err.Error(), "folder-test-id") {
		t.Errorf("expected friendly error message with solutions, got:\n%s", err.Error())
	}
}

