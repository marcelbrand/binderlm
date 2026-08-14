package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcelbrand/binderlm/internal/config"
	"github.com/marcelbrand/binderlm/internal/drive"
	"github.com/marcelbrand/binderlm/internal/stitcher"
	"google.golang.org/api/option"
)

func TestSyncWithMockDrive(t *testing.T) {
	uploadedContent := ""
	uploadedFilename := ""
	uploadedParents := []string{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// List files query
		if r.Method == http.MethodGet && (r.URL.Path == "/drive/v3/files" || r.URL.Path == "/files") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"files": []interface{}{},
			})
			return
		}

		// Create file (POST)
		if r.Method == http.MethodPost && (strings.HasSuffix(r.URL.Path, "/files") || strings.Contains(r.URL.Path, "/upload")) {
			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body)
			uploadedContent = buf.String()

			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":           "mock-created-id-789",
				"name":         "project_context_latest.md",
				"webViewLink":  "https://drive.google.com/file/d/mock-created-id-789/view",
				"size":         fmt.Sprintf("%d", len(uploadedContent)),
				"modifiedTime": "2026-08-14T15:30:00Z",
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer ts.Close()

	configPath, err := filepath.Abs("../examples/basic/binderlm.yaml")
	if err != nil {
		t.Fatalf("failed to resolve config path: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load basic config: %v", err)
	}

	// 1. Assemble markdown
	assembler := stitcher.NewAssembler(cfg)
	doc, err := assembler.Assemble(context.Background())
	if err != nil {
		t.Fatalf("assembly failed: %v", err)
	}

	// 2. Initialize Drive uploader against mock endpoint
	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
	if err != nil {
		t.Fatalf("failed to init mock drive service: %v", err)
	}

	uploader := drive.NewUploader(srv)
	targetFolder := "test-folder-id"
	result, err := uploader.Sync(ctx, targetFolder, cfg.Output.Filename, bytes.NewReader(doc.Content))
	if err != nil {
		t.Fatalf("uploader.Sync failed: %v", err)
	}

	if result.Action != "created" {
		t.Errorf("expected action 'created', got %q", result.Action)
	}
	if result.FileID != "mock-created-id-789" {
		t.Errorf("expected FileID 'mock-created-id-789', got %q", result.FileID)
	}
	if !strings.Contains(uploadedContent, "Basic Sample Documentation") {
		t.Errorf("expected uploaded content to contain 'Basic Sample Documentation', got %s", uploadedContent)
	}
	_ = uploadedFilename
	_ = uploadedParents
}

func TestSyncMissingCredentials(t *testing.T) {
	// Temporarily clear environment variables
	origJSON := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")
	origFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")
	os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
	defer func() {
		if origJSON != "" {
			os.Setenv("GOOGLE_APPLICATION_CREDENTIALS_JSON", origJSON)
		}
		if origFile != "" {
			os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", origFile)
		}
	}()

	_, err := drive.ResolveClientOption(context.Background())
	if err == nil {
		t.Fatalf("expected AuthError when no credentials are provided, got nil")
	}
	if !drive.IsAuthError(err) {
		t.Errorf("expected error to be AuthError, got: %T (%v)", err, err)
	}
}
