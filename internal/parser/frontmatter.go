package parser

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// FrontmatterResult represents extracted metadata and processed content.
type FrontmatterResult struct {
	Metadata  map[string]interface{}
	Title     string
	Body      []byte
	Processed []byte // Content formatted according to the requested frontmatter mode
}

// ProcessFrontmatter parses frontmatter from source and formats it according to mode ("strip", "table", "keep").
func ProcessFrontmatter(source []byte, filePath string, mode string) (*FrontmatterResult, error) {
	if mode == "" {
		mode = "strip"
	}

	rawFM, body, hasFM := extractRawFrontmatter(source)

	var meta map[string]interface{}
	if hasFM && len(rawFM) > 0 {
		meta = make(map[string]interface{})
		if err := yaml.Unmarshal(rawFM, &meta); err != nil {
			return nil, fmt.Errorf("invalid YAML frontmatter in %s: %w", filePath, err)
		}
	}

	title := resolveTitle(meta, body, filePath)

	var processed []byte
	switch mode {
	case "strip":
		processed = body
	case "keep":
		processed = source
	case "table":
		if len(meta) > 0 {
			table := renderFrontmatterTable(meta)
			var buf bytes.Buffer
			buf.WriteString(table)
			buf.WriteString("\n\n")
			buf.Write(body)
			processed = buf.Bytes()
		} else {
			processed = body
		}
	default:
		processed = body
	}

	return &FrontmatterResult{
		Metadata:  meta,
		Title:     title,
		Body:      body,
		Processed: processed,
	}, nil
}

func extractRawFrontmatter(source []byte) ([]byte, []byte, bool) {
	trimmed := bytes.TrimLeft(source, " \t\r\n")
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return nil, source, false
	}

	// Find the end of the opening delimiter line
	firstNewline := bytes.IndexByte(trimmed, '\n')
	if firstNewline == -1 {
		return nil, source, false
	}

	rest := trimmed[firstNewline+1:]

	// Search for closing delimiter "---" or "..." at the start of a line
	lines := bytes.Split(rest, []byte("\n"))
	var fmLines [][]byte
	closingIndex := -1

	for i, line := range lines {
		lineTrimmed := bytes.TrimSpace(line)
		if bytes.Equal(lineTrimmed, []byte("---")) || bytes.Equal(lineTrimmed, []byte("...")) {
			closingIndex = i
			break
		}
		fmLines = append(fmLines, line)
	}

	if closingIndex == -1 {
		// No closing delimiter found, treat whole document as body
		return nil, source, false
	}

	rawFM := bytes.Join(fmLines, []byte("\n"))

	// Body starts after the closing delimiter line
	var bodyLines [][]byte
	if closingIndex+1 < len(lines) {
		bodyLines = lines[closingIndex+1:]
	}
	body := bytes.TrimLeft(bytes.Join(bodyLines, []byte("\n")), "\r\n")

	return rawFM, body, true
}

func resolveTitle(meta map[string]interface{}, body []byte, filePath string) string {
	// 1. Check title in frontmatter
	if meta != nil {
		if t, ok := meta["title"]; ok {
			if str, ok := t.(string); ok && strings.TrimSpace(str) != "" {
				return strings.TrimSpace(str)
			}
		}
	}

	// 2. Check first H1 in body
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			if title != "" {
				return title
			}
		}
	}

	// 3. Fallback to formatted filename
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return formatFilenameAsTitle(name)
}

func formatFilenameAsTitle(name string) string {
	// Replace hyphens and underscores with spaces
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")

	words := strings.Fields(name)
	for i, w := range words {
		if len(w) > 0 {
			runes := []rune(w)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

func renderFrontmatterTable(meta map[string]interface{}) string {
	if len(meta) == 0 {
		return ""
	}

	keys := make([]string, 0, len(meta))
	for k := range meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString("| Property | Value |\n")
	sb.WriteString("| :--- | :--- |\n")

	for _, k := range keys {
		val := fmt.Sprintf("%v", meta[k])
		// Sanitize pipe characters inside markdown table
		val = strings.ReplaceAll(val, "|", "\\|")
		sb.WriteString(fmt.Sprintf("| **%s** | %s |\n", k, val))
	}

	return strings.TrimRight(sb.String(), "\n")
}
