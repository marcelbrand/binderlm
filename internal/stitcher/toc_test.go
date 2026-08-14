package stitcher_test

import (
	"strings"
	"testing"

	"github.com/marcelbrand/binderlm/internal/stitcher"
)

func TestSlugTrackerDisambiguation(t *testing.T) {
	st := stitcher.NewSlugTracker()

	slug1 := st.Slugify("Architecture Overview")
	slug2 := st.Slugify("Architecture Overview")
	slug3 := st.Slugify("Architecture Overview")
	slugSpecial := st.Slugify("Special & Characters?! (1.0)")

	if slug1 != "architecture-overview" {
		t.Errorf("expected 'architecture-overview', got '%s'", slug1)
	}
	if slug2 != "architecture-overview-1" {
		t.Errorf("expected 'architecture-overview-1', got '%s'", slug2)
	}
	if slug3 != "architecture-overview-2" {
		t.Errorf("expected 'architecture-overview-2', got '%s'", slug3)
	}
	if slugSpecial != "special-characters-10" {
		t.Errorf("expected 'special-characters-10', got '%s'", slugSpecial)
	}
}

func TestBuildTOC(t *testing.T) {
	items := []stitcher.TOCItem{
		{Title: "Section 1", Level: 2, Anchor: "section-1"},
		{Title: "Subsection 1.1", Level: 3, Anchor: "subsection-11"},
		{Title: "Detail", Level: 4, Anchor: "detail"},
		{Title: "Section 2", Level: 2, Anchor: "section-2"},
	}

	toc := stitcher.BuildTOC(items)

	expectedLines := []string{
		"* [Section 1](#section-1)",
		"  * [Subsection 1.1](#subsection-11)",
		"    * [Detail](#detail)",
		"* [Section 2](#section-2)",
	}

	expected := strings.Join(expectedLines, "\n")
	if toc != expected {
		t.Errorf("TOC mismatch.\nExpected:\n%s\nGot:\n%s", expected, toc)
	}
}
