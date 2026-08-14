package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcelbrand/binderlm/internal/config"
	"github.com/marcelbrand/binderlm/internal/validator"
)

func TestValidateIntegration_BasicExample(t *testing.T) {
	configPath, err := filepath.Abs("../examples/basic/binderlm.yaml")
	if err != nil {
		t.Fatalf("failed to get abs path: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load basic config: %v", err)
	}

	res, err := validator.Validate(context.Background(), cfg, validator.ValidationOptions{})
	if err != nil {
		t.Fatalf("validation failed with error: %v", err)
	}

	if !res.Valid {
		t.Errorf("expected basic example to be valid, got diagnostics: %+v", res.Diagnostics)
	}
	if res.FileCount != 2 {
		t.Errorf("expected 2 resolved files, got %d", res.FileCount)
	}
	if res.SectionCount != 2 {
		t.Errorf("expected 2 sections, got %d", res.SectionCount)
	}
	if res.ErrorCount != 0 {
		t.Errorf("expected 0 errors, got %d", res.ErrorCount)
	}
}

func TestValidateIntegration_MicroservicesExample(t *testing.T) {
	configPath, err := filepath.Abs("../examples/microservices/binderlm.yaml")
	if err != nil {
		t.Fatalf("failed to get abs path: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load microservices config: %v", err)
	}

	res, err := validator.Validate(context.Background(), cfg, validator.ValidationOptions{})
	if err != nil {
		t.Fatalf("validation failed with error: %v", err)
	}

	if !res.Valid {
		t.Errorf("expected microservices example to be valid, got diagnostics: %+v", res.Diagnostics)
	}
	if res.FileCount != 3 {
		t.Errorf("expected 3 resolved files, got %d", res.FileCount)
	}
	if res.SectionCount != 4 {
		t.Errorf("expected 4 sections, got %d", res.SectionCount)
	}
}

func TestValidateIntegration_StrictFailsOnEmptyGlob(t *testing.T) {
	tmpDir := t.TempDir()
	emptyDocs := filepath.Join(tmpDir, "emptydocs")
	if err := os.MkdirAll(emptyDocs, 0755); err != nil {
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
				Title:   "Empty Section",
				Path:    "emptydocs",
				Pattern: "*.md",
			},
		},
	}

	resNonStrict, err := validator.Validate(context.Background(), cfg, validator.ValidationOptions{Strict: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resNonStrict.Valid {
		t.Errorf("expected non-strict mode to pass with warnings")
	}

	resStrict, err := validator.Validate(context.Background(), cfg, validator.ValidationOptions{Strict: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resStrict.Valid {
		t.Errorf("expected strict mode to fail when warnings exist")
	}
}
