package stitcher_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcelbrand/binderlm/internal/config"
	"github.com/marcelbrand/binderlm/internal/stitcher"
)

func TestAssemblerEndToEnd(t *testing.T) {
	tempDir := t.TempDir()

	// Create test structure:
	// docs/
	//   arch.md
	// services/auth/
	//   login.md

	_ = os.MkdirAll(filepath.Join(tempDir, "docs"), 0755)
	_ = os.MkdirAll(filepath.Join(tempDir, "services", "auth"), 0755)

	archMD := `---
title: "Core Architecture"
---
# Architecture

The system is modular.

## Data Layer

PostgreSQL is used.
`
	loginMD := `# Login Flow

Authentication uses JWT.
`

	_ = os.WriteFile(filepath.Join(tempDir, "docs", "arch.md"), []byte(archMD), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "services", "auth", "login.md"), []byte(loginMD), 0644)

	cfg := config.DefaultConfig()
	cfg.BaseDir = tempDir
	cfg.Output.Title = "Full System Specification"
	cfg.Output.Description = "Generated context document for testing."
	cfg.Sections = []config.SectionConfig{
		{
			Title: "Architecture & Foundation",
			Level: 1,
			Files: []string{"docs/arch.md"},
		},
		{
			Title: "Services",
			Level: 1,
			Subsections: []config.SectionConfig{
				{
					Title: "Auth Service",
					Level: 2,
					Path:  "services/auth",
				},
			},
		},
	}

	assembler := stitcher.NewAssembler(cfg)
	doc, err := assembler.Assemble(context.Background())
	if err != nil {
		t.Fatalf("unexpected error during assemble: %v", err)
	}

	content := string(doc.Content)

	// Check master title & description
	if !strings.Contains(content, "# Full System Specification") {
		t.Errorf("missing master title in output:\n%s", content)
	}
	if !strings.Contains(content, "Generated context document for testing.") {
		t.Errorf("missing description in output:\n%s", content)
	}

	// Check TOC
	if !strings.Contains(content, "## Table of Contents") {
		t.Errorf("missing TOC header in output:\n%s", content)
	}
	if !strings.Contains(content, "* [Architecture & Foundation](#architecture-foundation)") {
		t.Errorf("missing TOC item for root section:\n%s", content)
	}
	if !strings.Contains(content, "  * [Auth Service](#auth-service)") {
		t.Errorf("missing TOC item for subsection:\n%s", content)
	}

	// Check section headings
	if !strings.Contains(content, "## Architecture & Foundation") {
		t.Errorf("missing H2 section heading in output:\n%s", content)
	}
	if !strings.Contains(content, "### Auth Service") {
		t.Errorf("missing H3 subsection heading in output:\n%s", content)
	}

	// Check source provenance hints
	if !strings.Contains(content, "> *Source: `docs/arch.md`*") {
		t.Errorf("missing source hint for arch.md:\n%s", content)
	}
	if !strings.Contains(content, "> *Source: `services/auth/login.md`*") {
		t.Errorf("missing source hint for login.md:\n%s", content)
	}

	// Check shifted headings:
	// In arch.md (section level 2), # Architecture should become ### Architecture (H3)
	if !strings.Contains(content, "### Architecture") {
		t.Errorf("expected shifted '### Architecture' in output:\n%s", content)
	}
	// In arch.md, ## Data Layer should become #### Data Layer (H4)
	if !strings.Contains(content, "#### Data Layer") {
		t.Errorf("expected shifted '#### Data Layer' in output:\n%s", content)
	}

	// In login.md (subsection level 3), # Login Flow should become #### Login Flow (H4)
	if !strings.Contains(content, "#### Login Flow") {
		t.Errorf("expected shifted '#### Login Flow' in output:\n%s", content)
	}

	if doc.FileCount != 2 {
		t.Errorf("expected FileCount = 2, got %d", doc.FileCount)
	}
}
