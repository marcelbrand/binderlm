package config

import (
	"fmt"
	"strings"
)

var allowedFrontmatterModes = map[string]bool{
	"strip": true,
	"table": true,
	"keep":  true,
}

// Validate checks the configuration for semantic correctness and required fields.
func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if cfg.Version != "1" && cfg.Version != "1.0" {
		return fmt.Errorf("unsupported config version %q (expected \"1\")", cfg.Version)
	}

	if strings.TrimSpace(cfg.Output.Filename) == "" {
		return fmt.Errorf("output.filename must not be empty")
	}

	if cfg.Output.FrontmatterMode == "" {
		cfg.Output.FrontmatterMode = "strip"
	} else if !allowedFrontmatterModes[cfg.Output.FrontmatterMode] {
		return fmt.Errorf("invalid output.frontmatter_mode %q (must be one of: strip, table, keep)", cfg.Output.FrontmatterMode)
	}

	if cfg.Output.MaxHeadingLevel < 1 || cfg.Output.MaxHeadingLevel > 6 {
		return fmt.Errorf("output.max_heading_level must be between 1 and 6 (got %d)", cfg.Output.MaxHeadingLevel)
	}

	if len(cfg.Sections) == 0 {
		return fmt.Errorf("sections list must contain at least one section")
	}

	for i, sec := range cfg.Sections {
		if err := validateSection(sec, fmt.Sprintf("sections[%d]", i), 1); err != nil {
			return err
		}
	}

	return nil
}

func validateSection(sec SectionConfig, path string, defaultLevel int) error {
	if strings.TrimSpace(sec.Title) == "" {
		return fmt.Errorf("%s: section title must not be empty", path)
	}

	hasFiles := len(sec.Files) > 0
	hasPath := strings.TrimSpace(sec.Path) != ""
	hasSubsections := len(sec.Subsections) > 0

	if !hasFiles && !hasPath && !hasSubsections {
		return fmt.Errorf("%s (%q): section must specify 'files', 'path', or 'subsections'", path, sec.Title)
	}

	for i, sub := range sec.Subsections {
		subPath := fmt.Sprintf("%s.subsections[%d]", path, i)
		if err := validateSection(sub, subPath, defaultLevel+1); err != nil {
			return err
		}
	}

	return nil
}
