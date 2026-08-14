package parser_test

import (
	"strings"
	"testing"

	"github.com/marcelbrand/binderlm/internal/parser"
)

func TestFrontmatterModes(t *testing.T) {
	doc := `---
title: "User Authentication Guide"
version: 1.2
author: "Engineering Team"
---
# Welcome

This is the auth documentation.
`

	// 1. Strip Mode
	resStrip, err := parser.ProcessFrontmatter([]byte(doc), "docs/auth.md", "strip")
	if err != nil {
		t.Fatalf("unexpected error in strip mode: %v", err)
	}
	if resStrip.Title != "User Authentication Guide" {
		t.Errorf("expected title 'User Authentication Guide', got '%s'", resStrip.Title)
	}
	if strings.Contains(string(resStrip.Processed), "---") {
		t.Errorf("expected frontmatter to be stripped, got:\n%s", string(resStrip.Processed))
	}
	if !strings.Contains(string(resStrip.Processed), "# Welcome") {
		t.Errorf("expected body to contain '# Welcome'")
	}

	// 2. Table Mode
	resTable, err := parser.ProcessFrontmatter([]byte(doc), "docs/auth.md", "table")
	if err != nil {
		t.Fatalf("unexpected error in table mode: %v", err)
	}
	if !strings.Contains(string(resTable.Processed), "| Property | Value |") {
		t.Errorf("expected table header in output, got:\n%s", string(resTable.Processed))
	}
	if !strings.Contains(string(resTable.Processed), "| **title** | User Authentication Guide |") {
		t.Errorf("expected table row in output, got:\n%s", string(resTable.Processed))
	}

	// 3. Keep Mode
	resKeep, err := parser.ProcessFrontmatter([]byte(doc), "docs/auth.md", "keep")
	if err != nil {
		t.Fatalf("unexpected error in keep mode: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(resKeep.Processed)), "---") {
		t.Errorf("expected raw frontmatter in keep mode")
	}
}

func TestTitleResolutionFallbacks(t *testing.T) {
	// Fallback 1: First H1 in body if no frontmatter title
	docNoTitle := `---
author: "Dev"
---
# Database Migrations

Details about migrations.
`
	res, err := parser.ProcessFrontmatter([]byte(docNoTitle), "services/db/migrate.md", "strip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Title != "Database Migrations" {
		t.Errorf("expected title 'Database Migrations', got '%s'", res.Title)
	}

	// Fallback 2: Formatted filename if no H1 and no frontmatter
	docNoH1 := `
Just some notes without headings.
`
	res2, err := parser.ProcessFrontmatter([]byte(docNoH1), "services/billing/payment_flow-v2.md", "strip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.Title != "Payment Flow V2" {
		t.Errorf("expected title 'Payment Flow V2', got '%s'", res2.Title)
	}
}
