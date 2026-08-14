package stitcher

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/marcelbrand/binderlm/internal/config"
	"github.com/marcelbrand/binderlm/internal/parser"
)

// Assembler aggregates sections and markdown files into a master context document.
type Assembler struct {
	cfg     *config.Config
	reader  *parser.FileReader
	shifter *parser.HeadingShifter
}

// NewAssembler creates a new Assembler configured with the provided config.
func NewAssembler(cfg *config.Config) *Assembler {
	maxLevel := cfg.Output.MaxHeadingLevel
	if maxLevel <= 0 {
		maxLevel = 6
	}

	return &Assembler{
		cfg:     cfg,
		reader:  parser.NewFileReader(cfg.BaseDir),
		shifter: parser.NewHeadingShifter(maxLevel),
	}
}

// Assemble compiles the entire document tree and returns the CompiledDocument.
func (a *Assembler) Assemble(ctx context.Context) (*CompiledDocument, error) {
	slugTracker := NewSlugTracker()
	var tocItems []TOCItem
	var bodyBuf bytes.Buffer

	fileCount := 0
	headingCount := 0

	for i, sec := range a.cfg.Sections {
		secLevel := sec.Level
		if secLevel <= 0 {
			secLevel = 1
		}
		// In markdown hierarchy, Document title is H1, so root sections start at H2
		hLevel := secLevel + 1
		if hLevel > a.cfg.Output.MaxHeadingLevel {
			hLevel = a.cfg.Output.MaxHeadingLevel
		}

		if err := a.processSection(ctx, sec, hLevel, slugTracker, &tocItems, &bodyBuf, &fileCount, &headingCount, fmt.Sprintf("section[%d]", i)); err != nil {
			return nil, err
		}
	}

	var finalDoc bytes.Buffer

	// 1. Master Title
	title := a.cfg.Output.Title
	if title == "" {
		title = "Master Context Document"
	}
	finalDoc.WriteString("# " + title + "\n\n")

	// 2. Master Description
	if desc := strings.TrimSpace(a.cfg.Output.Description); desc != "" {
		finalDoc.WriteString(desc + "\n\n")
	}

	// 3. Table of Contents
	if a.cfg.Output.IsGenerateTOC() && len(tocItems) > 0 {
		finalDoc.WriteString("## Table of Contents\n\n")
		tocContent := BuildTOC(tocItems)
		finalDoc.WriteString(tocContent + "\n\n")
		finalDoc.WriteString("---\n\n")
	}

	// 4. Document Body
	finalDoc.Write(bodyBuf.Bytes())

	content := finalDoc.Bytes()
	return &CompiledDocument{
		Title:        title,
		Description:  a.cfg.Output.Description,
		Content:      content,
		FileCount:    fileCount,
		HeadingCount: headingCount,
		ByteCount:    len(content),
	}, nil
}

func (a *Assembler) processSection(
	ctx context.Context,
	sec config.SectionConfig,
	headingLevel int,
	slugTracker *SlugTracker,
	tocItems *[]TOCItem,
	buf *bytes.Buffer,
	fileCount *int,
	headingCount *int,
	pathTrace string,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Emit Section Heading
	secTitle := strings.TrimSpace(sec.Title)
	if secTitle != "" {
		anchor := slugTracker.Slugify(secTitle)
		*tocItems = append(*tocItems, TOCItem{
			Title:  secTitle,
			Level:  headingLevel,
			Anchor: anchor,
		})
		*headingCount++

		headingPrefix := strings.Repeat("#", headingLevel)
		buf.WriteString(fmt.Sprintf("%s %s\n\n", headingPrefix, secTitle))
	}

	// Discover and process files in this section
	files, err := a.reader.DiscoverFiles(sec)
	if err != nil {
		return fmt.Errorf("error in %s (%q): %w", pathTrace, sec.Title, err)
	}

	for _, f := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rawBytes, err := os.ReadFile(f.FullPath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", f.FullPath, err)
		}

		fmResult, err := parser.ProcessFrontmatter(rawBytes, f.RelativePath, a.cfg.Output.FrontmatterMode)
		if err != nil {
			return err
		}

		// Shift headings: imported document headings are shifted by headingLevel
		shiftedBody, headings, err := a.shifter.ShiftHeadings(fmResult.Processed, headingLevel)
		if err != nil {
			return fmt.Errorf("failed to shift headings for %s: %w", f.FullPath, err)
		}

		// Register file headings into TOC
		for _, h := range headings {
			hAnchor := slugTracker.Slugify(h.Text)
			*tocItems = append(*tocItems, TOCItem{
				Title:  h.Text,
				Level:  h.Level,
				Anchor: hAnchor,
			})
			*headingCount++
		}

		// Inset source provenance annotation if enabled
		if a.cfg.Output.IsInjectSourceHints() {
			buf.WriteString(fmt.Sprintf("> *Source: `%s`*\n\n", f.RelativePath))
		}

		// Append file content
		trimmedBody := bytes.TrimSpace(shiftedBody)
		if len(trimmedBody) > 0 {
			buf.Write(trimmedBody)
			buf.WriteString("\n\n")
		}

		*fileCount++
	}

	// Recursively process subsections
	for i, sub := range sec.Subsections {
		subLevel := sub.Level
		if subLevel <= 0 {
			subLevel = headingLevel + 1
		} else {
			subLevel = subLevel + 1 // Offset by H1 master title
		}
		if subLevel > a.cfg.Output.MaxHeadingLevel {
			subLevel = a.cfg.Output.MaxHeadingLevel
		}

		subPathTrace := fmt.Sprintf("%s.subsections[%d]", pathTrace, i)
		if err := a.processSection(ctx, sub, subLevel, slugTracker, tocItems, buf, fileCount, headingCount, subPathTrace); err != nil {
			return err
		}
	}

	return nil
}
