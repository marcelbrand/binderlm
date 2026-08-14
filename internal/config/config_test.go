package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/marcelbrand/binderlm/internal/config"
)

func TestConfigDefaultsAndValidation(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "binderlm.yaml")

	validYAML := `
version: "1"
output:
  filename: "compiled_docs.md"
  title: "Test Project"
  frontmatter_mode: "table"
sections:
  - title: "Overview"
    files:
      - "docs/intro.md"
`
	if err := os.WriteFile(configPath, []byte(validYAML), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("expected config.Load to succeed, got error: %v", err)
	}

	if cfg.Output.Filename != "compiled_docs.md" {
		t.Errorf("expected filename compiled_docs.md, got %s", cfg.Output.Filename)
	}
	if !cfg.Output.IsGenerateTOC() {
		t.Errorf("expected generate_toc to default to true")
	}
	if !cfg.Output.IsInjectSourceHints() {
		t.Errorf("expected inject_source_hints to default to true")
	}
	if cfg.Output.FrontmatterMode != "table" {
		t.Errorf("expected frontmatter_mode to be table, got %s", cfg.Output.FrontmatterMode)
	}
	if cfg.Output.MaxHeadingLevel != 6 {
		t.Errorf("expected max_heading_level to default to 6, got %d", cfg.Output.MaxHeadingLevel)
	}
	if len(cfg.Sections) != 1 {
		t.Errorf("expected 1 section, got %d", len(cfg.Sections))
	}
}

func TestConfigValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "invalid version",
			yaml: `version: "2"` + "\nsections:\n  - title: \"A\"\n    files: [\"a.md\"]",
		},
		{
			name: "empty filename",
			yaml: `version: "1"` + "\noutput:\n  filename: \"\"\nsections:\n  - title: \"A\"\n    files: [\"a.md\"]",
		},
		{
			name: "invalid frontmatter mode",
			yaml: `version: "1"` + "\noutput:\n  frontmatter_mode: \"invalid\"\nsections:\n  - title: \"A\"\n    files: [\"a.md\"]",
		},
		{
			name: "empty sections",
			yaml: `version: "1"` + "\nsections: []",
		},
		{
			name: "empty section title",
			yaml: `version: "1"` + "\nsections:\n  - title: \"\"\n    files: [\"a.md\"]",
		},
		{
			name: "section missing files and path",
			yaml: `version: "1"` + "\nsections:\n  - title: \"Empty Sec\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "binderlm.yaml")
			_ = os.WriteFile(configPath, []byte(tt.yaml), 0644)

			_, err := config.Load(configPath)
			if err == nil {
				t.Errorf("expected error for case %q, but got nil", tt.name)
			}
		})
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("GDRIVE_FOLDER_ID", "env_folder_12345")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "binderlm.yaml")

	validYAML := `
version: "1"
drive:
  folder_id: "config_folder_xyz"
sections:
  - title: "Overview"
    files:
      - "docs/intro.md"
`
	_ = os.WriteFile(configPath, []byte(validYAML), 0644)

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Drive.FolderID != "env_folder_12345" {
		t.Errorf("expected folder_id to be overridden to env_folder_12345, got %s", cfg.Drive.FolderID)
	}
}

func TestLoadEnvFile(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	content := `
# Sample comment
GDRIVE_OAUTH_CLIENT_ID=test-client-id-123
export GDRIVE_OAUTH_CLIENT_SECRET="test-secret-456"
TEST_QUOTED_VAR='quoted-val'
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test env: %v", err)
	}

	if err := config.LoadEnvFile(envPath); err != nil {
		t.Fatalf("unexpected error loading env file: %v", err)
	}

	if os.Getenv("GDRIVE_OAUTH_CLIENT_ID") != "test-client-id-123" {
		t.Errorf("expected test-client-id-123, got %s", os.Getenv("GDRIVE_OAUTH_CLIENT_ID"))
	}
	if os.Getenv("GDRIVE_OAUTH_CLIENT_SECRET") != "test-secret-456" {
		t.Errorf("expected test-secret-456, got %s", os.Getenv("GDRIVE_OAUTH_CLIENT_SECRET"))
	}
	if os.Getenv("TEST_QUOTED_VAR") != "quoted-val" {
		t.Errorf("expected quoted-val, got %s", os.Getenv("TEST_QUOTED_VAR"))
	}
}

