package parser

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// HeadingInfo represents a heading found in the document.
type HeadingInfo struct {
	Level    int
	Original int
	Text     string
	Line     int // 0-indexed line number in source
}

// HeadingShifter shifts Markdown headings using Goldmark AST to guarantee zero corruption of code blocks.
type HeadingShifter struct {
	maxLevel int
}

// NewHeadingShifter creates a HeadingShifter with maximum heading level (typically 6).
func NewHeadingShifter(maxLevel int) *HeadingShifter {
	if maxLevel < 1 || maxLevel > 6 {
		maxLevel = 6
	}
	return &HeadingShifter{maxLevel: maxLevel}
}

// ShiftHeadings parses markdown content with goldmark AST, shifts true heading nodes by offset,
// and returns the modified content along with extracted heading metadata.
func (s *HeadingShifter) ShiftHeadings(source []byte, offset int) ([]byte, []HeadingInfo, error) {
	if len(source) == 0 || offset <= 0 {
		headings := s.ExtractHeadings(source)
		return source, headings, nil
	}

	parser := goldmark.DefaultParser()
	doc := parser.Parse(text.NewReader(source))

	type headingReplacement struct {
		lineIndex int
		newLevel  int
		text      string
		origLevel int
	}

	var replacements []headingReplacement

	// Split source into lines for precise line-by-line reconstruction
	lines := bytes.Split(source, []byte("\n"))

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if h, ok := n.(*ast.Heading); ok {
			headingText := extractHeadingText(h, source)
			newLevel := h.Level + offset
			if newLevel > s.maxLevel {
				newLevel = s.maxLevel
			}

			// Get the line number of the heading
			if h.Lines().Len() > 0 {
				lineSegment := h.Lines().At(0)
				lineIdx := findLineIndex(source, lineSegment.Start)
				if lineIdx >= 0 && lineIdx < len(lines) {
					replacements = append(replacements, headingReplacement{
						lineIndex: lineIdx,
						newLevel:  newLevel,
						text:      headingText,
						origLevel: h.Level,
					})
				}
			}
		}

		return ast.WalkContinue, nil
	})

	// Apply replacements to lines
	for _, rep := range replacements {
		originalLine := string(lines[rep.lineIndex])
		trimmed := strings.TrimLeft(originalLine, " \t")
		leadingWhitespace := originalLine[:len(originalLine)-len(trimmed)]

		// If ATX heading (starts with #)
		if strings.HasPrefix(trimmed, "#") {
			afterHash := strings.TrimLeft(trimmed, "#")
			cleanContent := strings.TrimLeft(afterHash, " \t")
			newHeading := leadingWhitespace + strings.Repeat("#", rep.newLevel) + " " + cleanContent
			lines[rep.lineIndex] = []byte(newHeading)
		} else {
			// Setext heading (underlined with === or ---)
			newHeading := leadingWhitespace + strings.Repeat("#", rep.newLevel) + " " + rep.text
			lines[rep.lineIndex] = []byte(newHeading)
			if rep.lineIndex+1 < len(lines) {
				nextLine := strings.TrimSpace(string(lines[rep.lineIndex+1]))
				if isSetextUnderline(nextLine) {
					lines = append(lines[:rep.lineIndex+1], lines[rep.lineIndex+2:]...)
				}
			}
		}
	}

	result := bytes.Join(lines, []byte("\n"))
	headings := s.ExtractHeadings(result)
	return result, headings, nil
}

// ExtractHeadings collects all AST headings from markdown source without modifying it.
func (s *HeadingShifter) ExtractHeadings(source []byte) []HeadingInfo {
	if len(source) == 0 {
		return nil
	}

	parser := goldmark.DefaultParser()
	doc := parser.Parse(text.NewReader(source))

	var headings []HeadingInfo

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if h, ok := n.(*ast.Heading); ok {
			headingText := extractHeadingText(h, source)
			lineIdx := 0
			if h.Lines().Len() > 0 {
				lineIdx = findLineIndex(source, h.Lines().At(0).Start)
			}

			headings = append(headings, HeadingInfo{
				Level:    h.Level,
				Original: h.Level,
				Text:     headingText,
				Line:     lineIdx,
			})
		}
		return ast.WalkContinue, nil
	})

	return headings
}

func extractHeadingText(h *ast.Heading, source []byte) string {
	var buf bytes.Buffer
	for child := h.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			if t.Segment.Start >= 0 && t.Segment.Stop <= len(source) {
				buf.Write(source[t.Segment.Start:t.Segment.Stop])
			}
		} else if c, ok := child.(*ast.CodeSpan); ok {
			buf.WriteByte('`')
			buf.Write(c.Text(source))
			buf.WriteByte('`')
		} else if a, ok := child.(*ast.AutoLink); ok {
			buf.Write(a.Label(source))
		}
	}

	text := strings.TrimSpace(buf.String())
	if text == "" && h.Lines().Len() > 0 {
		seg := h.Lines().At(0)
		if seg.Start >= 0 && seg.Stop <= len(source) {
			raw := string(source[seg.Start:seg.Stop])
			text = strings.TrimSpace(strings.TrimLeft(raw, "# \t"))
		}
	}
	return text
}

func findLineIndex(source []byte, byteOffset int) int {
	if byteOffset < 0 || byteOffset > len(source) {
		return 0
	}
	line := 0
	for i := 0; i < byteOffset; i++ {
		if source[i] == '\n' {
			line++
		}
	}
	return line
}

func isSetextUnderline(s string) bool {
	if len(s) == 0 {
		return false
	}
	char := s[0]
	if char != '=' && char != '-' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] != char {
			return false
		}
	}
	return true
}
