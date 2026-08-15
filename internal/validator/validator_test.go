package validator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcelbrand/binderlm/internal/config"
)

func TestValidate_ValidBasic(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	doc1 := filepath.Join(tmpDir, "doc1.md")
	if err := os.WriteFile(doc1, []byte("# Document One\nContent here.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Version: "1",
		BaseDir: tmpDir,
		Output: config.OutputConfig{
			Filename:         "output.md",
			FrontmatterMode:  "strip",
			MaxHeadingLevel:  6,
		},
		Sections: []config.SectionConfig{
			{
				Title: "Section 1",
				Files: []string{"doc1.md"},
			},
		},
	}

	res, err := Validate(context.Background(), cfg, ValidationOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Valid {
		t.Errorf("expected config to be valid, got errors: %+v", res.Diagnostics)
	}
	if res.FileCount != 1 {
		t.Errorf("expected 1 file count, got %d", res.FileCount)
	}
	if res.ErrorCount != 0 {
		t.Errorf("expected 0 errors, got %d", res.ErrorCount)
	}
	if res.WarningCount != 0 {
		t.Errorf("expected 0 warnings, got %d", res.WarningCount)
	}
}

func TestValidate_MissingExplicitFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Version: "1",
		BaseDir: tmpDir,
		Output: config.OutputConfig{
			Filename:        "output.md",
			FrontmatterMode: "strip",
			MaxHeadingLevel: 6,
		},
		Sections: []config.SectionConfig{
			{
				Title: "Section 1",
				Files: []string{"nonexistent.md"},
			},
		},
	}

	res, err := Validate(context.Background(), cfg, ValidationOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Valid {
		t.Errorf("expected config to be invalid due to missing file")
	}
	if res.ErrorCount != 1 {
		t.Errorf("expected 1 error, got %d", res.ErrorCount)
	}
}

func TestValidate_MissingDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Version: "1",
		BaseDir: tmpDir,
		Output: config.OutputConfig{
			Filename:        "output.md",
			FrontmatterMode: "strip",
			MaxHeadingLevel: 6,
		},
		Sections: []config.SectionConfig{
			{
				Title: "Section 1",
				Path:  "nonexistent_dir",
			},
		},
	}

	res, err := Validate(context.Background(), cfg, ValidationOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Valid {
		t.Errorf("expected config to be invalid due to missing dir")
	}
	if res.ErrorCount != 1 {
		t.Errorf("expected 1 error, got %d", res.ErrorCount)
	}
}

func TestValidate_EmptyGlobMatch_WarningAndStrict(t *testing.T) {
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Version: "1",
		BaseDir: tmpDir,
		Output: config.OutputConfig{
			Filename:        "output.md",
			FrontmatterMode: "strip",
			MaxHeadingLevel: 6,
		},
		Sections: []config.SectionConfig{
			{
				Title:   "Section 1",
				Path:    "docs",
				Pattern: "*.md",
			},
		},
	}

	// Default mode: warning, but valid = true
	res, err := Validate(context.Background(), cfg, ValidationOptions{Strict: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Valid {
		t.Errorf("expected config to be valid in non-strict mode despite warning")
	}
	if res.WarningCount != 1 {
		t.Errorf("expected 1 warning, got %d", res.WarningCount)
	}

	// Strict mode: valid = false
	resStrict, err := Validate(context.Background(), cfg, ValidationOptions{Strict: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resStrict.Valid {
		t.Errorf("expected config to be invalid in strict mode with warnings")
	}
}

func TestValidate_MalformedFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	badDoc := filepath.Join(tmpDir, "bad.md")
	// Malformed YAML with unaligned tabs/colons
	malformedContent := "---\ntitle: [unclosed list\nauthor: test\n---\n# Header\n"
	if err := os.WriteFile(badDoc, []byte(malformedContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Version: "1",
		BaseDir: tmpDir,
		Output: config.OutputConfig{
			Filename:        "output.md",
			FrontmatterMode: "strip",
			MaxHeadingLevel: 6,
		},
		Sections: []config.SectionConfig{
			{
				Title: "Section 1",
				Files: []string{"bad.md"},
			},
		},
	}

	res, err := Validate(context.Background(), cfg, ValidationOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Valid {
		t.Errorf("expected config to be invalid due to malformed frontmatter")
	}
	if res.ErrorCount != 1 {
		t.Errorf("expected 1 error for malformed frontmatter, got %d", res.ErrorCount)
	}
}

func TestValidate_DriveCheckWithoutCreds(t *testing.T) {
	tmpDir := t.TempDir()
	doc := filepath.Join(tmpDir, "doc.md")
	if err := os.WriteFile(doc, []byte("# Valid doc\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Temporarily clear Google env vars and global config dir
	t.Setenv("BINDERLM_CONFIG_DIR", tmpDir)
	t.Setenv("BINDERLM_TOKEN_FILE", filepath.Join(tmpDir, "nonexistent_token.json"))
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

	cfg := &config.Config{
		Version: "1",
		BaseDir: tmpDir,
		Output: config.OutputConfig{
			Filename:        "output.md",
			FrontmatterMode: "strip",
			MaxHeadingLevel: 6,
		},
		Drive: config.DriveConfig{
			Enabled:  true,
			FolderID: "sample-folder-id",
		},
		Sections: []config.SectionConfig{
			{
				Title: "Section 1",
				Files: []string{"doc.md"},
			},
		},
	}

	// Default mode (CheckDrive=false): should be valid even with drive enabled and no credentials
	resDefault, err := Validate(context.Background(), cfg, ValidationOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resDefault.Valid {
		t.Errorf("expected config to be valid by default without checking Drive, got: %+v", resDefault.Diagnostics)
	}
	if resDefault.DriveChecked {
		t.Errorf("expected DriveChecked to be false by default")
	}

	// Explicit CheckDrive=true without credentials: should fail
	resCheck, err := Validate(context.Background(), cfg, ValidationOptions{CheckDrive: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resCheck.Valid {
		t.Errorf("expected config to be invalid when CheckDrive is true and credentials are missing")
	}
	if !resCheck.HasAuthError {
		t.Errorf("expected HasAuthError to be true when CheckDrive is true")
	}
	if !resCheck.DriveChecked {
		t.Errorf("expected DriveChecked to be true when CheckDrive is true")
	}
}
