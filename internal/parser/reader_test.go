package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/marcelbrand/binderlm/internal/config"
	"github.com/marcelbrand/binderlm/internal/parser"
)

func TestFileReaderGlobAndExclude(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file tree:
	//   docs/
	//     a.md
	//     b.md
	//     _drafts/
	//       c.md
	//     CHANGELOG.md
	//     sub/
	//       d.md

	_ = os.MkdirAll(filepath.Join(tempDir, "docs", "_drafts"), 0755)
	_ = os.MkdirAll(filepath.Join(tempDir, "docs", "sub"), 0755)

	_ = os.WriteFile(filepath.Join(tempDir, "docs", "a.md"), []byte("# A"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "docs", "b.md"), []byte("# B"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "docs", "_drafts", "c.md"), []byte("# C"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "docs", "CHANGELOG.md"), []byte("# Log"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "docs", "sub", "d.md"), []byte("# D"), 0644)

	reader := parser.NewFileReader(tempDir)

	sec := config.SectionConfig{
		Title:     "Docs",
		Path:      "docs",
		Recursive: true,
		Exclude: []string{
			"**/_drafts/**",
			"**/CHANGELOG.md",
		},
	}

	files, err := reader.DiscoverFiles(sec)
	if err != nil {
		t.Fatalf("unexpected error discovering files: %v", err)
	}

	// Should match docs/a.md, docs/b.md, docs/sub/d.md (3 files, sorted)
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}

	expected := []string{"docs/a.md", "docs/b.md", "docs/sub/d.md"}
	for i, exp := range expected {
		if files[i].RelativePath != exp {
			t.Errorf("file[%d] expected %s, got %s", i, exp, files[i].RelativePath)
		}
	}
}
