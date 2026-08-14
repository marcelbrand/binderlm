package test

import (
	"context"
	"strings"
	"testing"

	"github.com/marcelbrand/binderlm/internal/config"
	"github.com/marcelbrand/binderlm/internal/parser"
	"github.com/marcelbrand/binderlm/internal/stitcher"
)

func TestAST_CodeBlockIntegrity(t *testing.T) {
	inputMarkdown := `# Outer Title

Here is a bash code snippet:
` + "```bash" + `
# This is a comment inside bash
# Not a heading
echo "Hello # world"
` + "```" + `

And indented code block:
    # Another comment in 4-space code block
    echo "test"

And quadruple-fenced block:
` + "````markdown" + `
# Nested Markdown Heading in Code Block
## Subheading
` + "````" + `

## Next Section
Some content.
`

	shifter := parser.NewHeadingShifter(6)
	// Demote by offset 2 (H1 -> H3, H2 -> H4)
	shifted, headings, err := shifter.ShiftHeadings([]byte(inputMarkdown), 2)
	if err != nil {
		t.Fatalf("ShiftHeadings failed: %v", err)
	}

	shiftedStr := string(shifted)

	// 1. Verify outer H1 became H3
	if !strings.Contains(shiftedStr, "### Outer Title") {
		t.Errorf("expected '### Outer Title', got:\n%s", shiftedStr)
	}

	// 2. Verify outer H2 became H4
	if !strings.Contains(shiftedStr, "#### Next Section") {
		t.Errorf("expected '#### Next Section', got:\n%s", shiftedStr)
	}

	// 3. Verify bash comment was untouched
	if !strings.Contains(shiftedStr, "# This is a comment inside bash") {
		t.Errorf("bash comment was corrupted:\n%s", shiftedStr)
	}

	// 4. Verify indented code block comment was untouched
	if !strings.Contains(shiftedStr, "# Another comment in 4-space code block") {
		t.Errorf("indented code comment was corrupted:\n%s", shiftedStr)
	}

	// 5. Verify quad-fenced code block was untouched
	if !strings.Contains(shiftedStr, "# Nested Markdown Heading in Code Block") {
		t.Errorf("quad-fenced code content was corrupted:\n%s", shiftedStr)
	}

	// Verify extracted headings count
	if len(headings) != 2 {
		t.Errorf("expected exactly 2 headings, got %d", len(headings))
	}
}

func TestAST_HeadingClampingAtH6(t *testing.T) {
	inputMarkdown := `# Level 1
## Level 2
### Level 3
#### Level 4
##### Level 5
###### Level 6
`

	shifter := parser.NewHeadingShifter(6)
	// Demote with large offset 4 (H1->H5, H2->H6, H3->H6, H4->H6, H5->H6, H6->H6)
	shifted, headings, err := shifter.ShiftHeadings([]byte(inputMarkdown), 4)
	if err != nil {
		t.Fatalf("ShiftHeadings failed: %v", err)
	}

	shiftedStr := string(shifted)

	if !strings.Contains(shiftedStr, "##### Level 1") {
		t.Errorf("expected '##### Level 1', got:\n%s", shiftedStr)
	}
	if !strings.Contains(shiftedStr, "###### Level 2") {
		t.Errorf("expected '###### Level 2', got:\n%s", shiftedStr)
	}
	if !strings.Contains(shiftedStr, "###### Level 3") {
		t.Errorf("expected clamped '###### Level 3', got:\n%s", shiftedStr)
	}
	if !strings.Contains(shiftedStr, "###### Level 6") {
		t.Errorf("expected clamped '###### Level 6', got:\n%s", shiftedStr)
	}

	for _, h := range headings {
		if h.Level > 6 {
			t.Errorf("heading level exceeded max level 6: %+v", h)
		}
	}
}

func TestTOC_DisambiguationAndSlugs(t *testing.T) {
	st := stitcher.NewSlugTracker()

	headings := []struct {
		Level int
		Title string
	}{
		{Level: 2, Title: "Getting Started"},
		{Level: 2, Title: "Configuration"},
		{Level: 3, Title: "Overview"},
		{Level: 2, Title: "Services"},
		{Level: 3, Title: "Overview"}, // Duplicate slug -> overview-1
		{Level: 3, Title: "Overview"}, // Duplicate slug -> overview-2
		{Level: 2, Title: "API & Reference!"},
	}

	var items []stitcher.TOCItem
	for _, h := range headings {
		items = append(items, stitcher.TOCItem{
			Level:  h.Level,
			Title:  h.Title,
			Anchor: st.Slugify(h.Title),
		})
	}

	tocMarkdown := stitcher.BuildTOC(items)

	expectedEntries := []string{
		"* [Getting Started](#getting-started)",
		"* [Configuration](#configuration)",
		"  * [Overview](#overview)",
		"* [Services](#services)",
		"  * [Overview](#overview-1)",
		"  * [Overview](#overview-2)",
		"* [API & Reference!](#api-reference)",
	}

	for _, entry := range expectedEntries {
		if !strings.Contains(tocMarkdown, entry) {
			t.Errorf("expected TOC to contain %q, got:\n%s", entry, tocMarkdown)
		}
	}
}

func TestFrontmatter_Modes(t *testing.T) {
	doc := `---
title: "Custom Auth System"
version: "2.1"
author: "Security Team"
status: "active"
---
# Main Header

Body text goes here.
`

	// 1. Strip mode
	resStrip, err := parser.ProcessFrontmatter([]byte(doc), "test.md", "strip")
	if err != nil {
		t.Fatalf("strip mode failed: %v", err)
	}
	if resStrip.Title != "Custom Auth System" {
		t.Errorf("expected title 'Custom Auth System', got %q", resStrip.Title)
	}
	if strings.Contains(string(resStrip.Processed), "author: \"Security Team\"") {
		t.Errorf("strip mode did not remove frontmatter:\n%s", string(resStrip.Processed))
	}

	// 2. Keep mode
	resKeep, err := parser.ProcessFrontmatter([]byte(doc), "test.md", "keep")
	if err != nil {
		t.Fatalf("keep mode failed: %v", err)
	}
	if !strings.Contains(string(resKeep.Processed), "author: \"Security Team\"") {
		t.Errorf("keep mode lost frontmatter:\n%s", string(resKeep.Processed))
	}

	// 3. Table mode
	resTable, err := parser.ProcessFrontmatter([]byte(doc), "test.md", "table")
	if err != nil {
		t.Fatalf("table mode failed: %v", err)
	}
	tableStr := string(resTable.Processed)
	if !strings.Contains(tableStr, "| Property | Value |") {
		t.Errorf("table mode missing table header:\n%s", tableStr)
	}
	if !strings.Contains(tableStr, "| **author** | Security Team |") {
		t.Errorf("table mode missing author row:\n%s", tableStr)
	}
}

func TestAssembler_SourceProvenanceInjection(t *testing.T) {
	trueVal := true
	cfg := &config.Config{
		Version: "1",
		BaseDir: "../examples/basic",
		Output: config.OutputConfig{
			Title:             "Master Architecture",
			Filename:          "output.md",
			GenerateTOC:       &trueVal,
			InjectSourceHints: &trueVal,
			FrontmatterMode:   "strip",
			MaxHeadingLevel:   6,
		},
		Sections: []config.SectionConfig{
			{
				Title: "Introduction",
				Files: []string{"./docs/intro.md"},
			},
		},
	}

	assembler := stitcher.NewAssembler(cfg)
	doc, err := assembler.Assemble(context.Background())
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	docStr := string(doc.Content)
	if !strings.Contains(docStr, "> *Source: `docs/intro.md`*") {
		t.Errorf("expected source hint injection, got:\n%s", docStr)
	}
}
