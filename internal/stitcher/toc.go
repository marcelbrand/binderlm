package stitcher

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// SlugTracker keeps track of generated anchor slugs to disambiguate collisions.
type SlugTracker struct {
	counts map[string]int
}

// NewSlugTracker creates a new SlugTracker.
func NewSlugTracker() *SlugTracker {
	return &SlugTracker{
		counts: make(map[string]int),
	}
}

// Slugify creates a GitHub/CommonMark-compatible anchor slug and disambiguates duplicates.
func (st *SlugTracker) Slugify(title string) string {
	slug := sanitizeSlug(title)
	if slug == "" {
		slug = "section"
	}

	count, exists := st.counts[slug]
	if !exists {
		st.counts[slug] = 0
		return slug
	}

	st.counts[slug] = count + 1
	return fmt.Sprintf("%s-%d", slug, count+1)
}

var nonAlphanumericRegex = regexp.MustCompile(`[^\p{L}\p{N}\s\-_]+`)

func sanitizeSlug(s string) string {
	s = strings.ToLower(s)
	// Remove non-alphanumeric except spaces, hyphens, underscores
	s = nonAlphanumericRegex.ReplaceAllString(s, "")
	// Replace spaces with hyphens
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return '-'
		}
		return r
	}, s)
	// Collapse multiple hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

// BuildTOC generates a nested markdown Table of Contents from a slice of TOCItems.
func BuildTOC(items []TOCItem) string {
	if len(items) == 0 {
		return ""
	}

	// Find the minimum level in the items to use as baseline (usually level 2)
	minLevel := 6
	for _, it := range items {
		if it.Level < minLevel {
			minLevel = it.Level
		}
	}

	var sb strings.Builder
	for _, it := range items {
		indentDepth := it.Level - minLevel
		if indentDepth < 0 {
			indentDepth = 0
		}
		indent := strings.Repeat("  ", indentDepth)
		sb.WriteString(fmt.Sprintf("%s* [%s](#%s)\n", indent, it.Title, it.Anchor))
	}

	return strings.TrimRight(sb.String(), "\n")
}
