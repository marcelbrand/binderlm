package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/marcelbrand/binderlm/internal/config"
)

// DiscoveredFile contains metadata about a discovered markdown file.
type DiscoveredFile struct {
	FullPath     string
	RelativePath string
}

// FileReader handles file discovery and reading.
type FileReader struct {
	baseDir string
}

// NewFileReader creates a FileReader using baseDir to resolve relative paths.
func NewFileReader(baseDir string) *FileReader {
	if baseDir == "" {
		baseDir = "."
	}
	return &FileReader{baseDir: baseDir}
}

// DiscoverFiles finds all matching markdown files for a section configuration.
func (r *FileReader) DiscoverFiles(sec config.SectionConfig) ([]DiscoveredFile, error) {
	seen := make(map[string]bool)
	var discovered []DiscoveredFile

	// 1. Process explicit files
	for _, f := range sec.Files {
		fullPath := r.resolvePath(f)
		if _, err := os.Stat(fullPath); err != nil {
			return nil, fmt.Errorf("file not found: %s (%w)", f, err)
		}

		relPath, _ := filepath.Rel(r.baseDir, fullPath)
		if relPath == "" {
			relPath = f
		}

		if !seen[fullPath] {
			seen[fullPath] = true
			discovered = append(discovered, DiscoveredFile{
				FullPath:     fullPath,
				RelativePath: filepath.ToSlash(relPath),
			})
		}
	}

	// 2. Process path + pattern / recursive
	if strings.TrimSpace(sec.Path) != "" {
		dirPath := r.resolvePath(sec.Path)
		pattern := sec.Pattern
		if pattern == "" {
			if sec.Recursive {
				pattern = "**/*.md"
			} else {
				pattern = "*.md"
			}
		}

		fsys := os.DirFS(dirPath)
		matches, err := doublestar.Glob(fsys, pattern)
		if err != nil {
			return nil, fmt.Errorf("glob error in path %s with pattern %s: %w", sec.Path, pattern, err)
		}

		var matchedInDir []DiscoveredFile
		for _, m := range matches {
			fullPath := filepath.Join(dirPath, filepath.FromSlash(m))

			// Check if directory
			fi, err := os.Stat(fullPath)
			if err != nil || fi.IsDir() {
				continue
			}

			// Check exclusions
			if r.isExcluded(m, sec.Exclude) {
				continue
			}

			relPath, _ := filepath.Rel(r.baseDir, fullPath)
			if relPath == "" {
				relPath = fullPath
			}

			if !seen[fullPath] {
				seen[fullPath] = true
				matchedInDir = append(matchedInDir, DiscoveredFile{
					FullPath:     fullPath,
					RelativePath: filepath.ToSlash(relPath),
				})
			}
		}

		// Sort glob matches deterministically by relative path
		sort.Slice(matchedInDir, func(i, j int) bool {
			return matchedInDir[i].RelativePath < matchedInDir[j].RelativePath
		})

		discovered = append(discovered, matchedInDir...)
	}

	return discovered, nil
}

func (r *FileReader) resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(r.baseDir, p)
}

func (r *FileReader) isExcluded(relPath string, excludes []string) bool {
	norm := filepath.ToSlash(relPath)
	for _, excl := range excludes {
		exclNorm := filepath.ToSlash(excl)
		matched, err := doublestar.Match(exclNorm, norm)
		if err == nil && matched {
			return true
		}
		// Also check against basename
		matchedBase, err := doublestar.Match(exclNorm, filepath.Base(norm))
		if err == nil && matchedBase {
			return true
		}
	}
	return false
}
