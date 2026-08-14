package validator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/marcelbrand/binderlm/internal/config"
	"github.com/marcelbrand/binderlm/internal/drive"
	"github.com/marcelbrand/binderlm/internal/parser"
)

// DiagnosticSeverity represents the severity level of a validation diagnostic.
type DiagnosticSeverity string

const (
	SeverityInfo    DiagnosticSeverity = "INFO"
	SeverityWarning DiagnosticSeverity = "WARN"
	SeverityError   DiagnosticSeverity = "ERROR"
)

// Diagnostic represents an individual finding during validation.
type Diagnostic struct {
	Severity DiagnosticSeverity
	Section  string
	Path     string
	Message  string
}

func (d Diagnostic) String() string {
	var prefix string
	switch d.Severity {
	case SeverityInfo:
		prefix = "ℹ"
	case SeverityWarning:
		prefix = "⚠"
	case SeverityError:
		prefix = "✗"
	}

	loc := ""
	if d.Section != "" && d.Path != "" {
		loc = fmt.Sprintf(" [%s | %s]", d.Section, d.Path)
	} else if d.Section != "" {
		loc = fmt.Sprintf(" [%s]", d.Section)
	} else if d.Path != "" {
		loc = fmt.Sprintf(" [%s]", d.Path)
	}

	return fmt.Sprintf("%s%s %s", prefix, loc, d.Message)
}

// ValidationOptions controls validator behavior.
type ValidationOptions struct {
	CheckDrive bool
	Strict     bool
}

// ValidationResult summarizes the outcome of validating a configuration.
type ValidationResult struct {
	Valid          bool
	HasAuthError   bool
	AuthErr        error
	Diagnostics    []Diagnostic
	SectionCount   int
	FileCount      int
	WarningCount   int
	ErrorCount     int
	DriveChecked   bool
	DriveValid     bool
	ResolvedFiles  []string
}

// Validate performs deep static analysis and file resolution for a binderlm configuration.
func Validate(ctx context.Context, cfg *config.Config, opts ValidationOptions) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:         true,
		Diagnostics:   make([]Diagnostic, 0),
		ResolvedFiles: make([]string, 0),
	}

	if cfg == nil {
		result.Valid = false
		result.ErrorCount = 1
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  "configuration is nil",
		})
		return result, nil
	}

	// 1. Semantic configuration validation
	if err := config.Validate(cfg); err != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  fmt.Sprintf("schema validation failed: %v", err),
		})
		result.ErrorCount++
	}

	seenFiles := make(map[string]bool)
	fileReader := parser.NewFileReader(cfg.BaseDir)

	// 2. Validate sections recursively
	for i, sec := range cfg.Sections {
		validateSectionDeep(sec, fmt.Sprintf("sections[%d]", i), cfg, fileReader, seenFiles, result)
	}

	result.FileCount = len(seenFiles)

	// 3. Validate Google Drive configuration & credentials if requested/enabled
	if cfg.Drive.Enabled || opts.CheckDrive {
		result.DriveChecked = true
		if strings.TrimSpace(cfg.Drive.FolderID) == "" {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  "Google Drive is enabled but folder_id is not specified (provide --folder-id, set drive.folder_id in config, or GDRIVE_FOLDER_ID env var)",
			})
			result.ErrorCount++
		}

		_, authErr := drive.ResolveClientOption(ctx)
		if authErr != nil {
			result.HasAuthError = true
			result.AuthErr = authErr
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  fmt.Sprintf("Google Drive authentication failed: %v", authErr),
			})
			result.ErrorCount++
		} else {
			result.DriveValid = true
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityInfo,
				Message:  "Google Drive credentials resolved successfully",
			})
		}
	}

	// Calculate validity based on errors and strict mode
	if result.ErrorCount > 0 {
		result.Valid = false
	} else if opts.Strict && result.WarningCount > 0 {
		result.Valid = false
	} else {
		result.Valid = true
	}

	return result, nil
}

func validateSectionDeep(
	sec config.SectionConfig,
	secPath string,
	cfg *config.Config,
	reader *parser.FileReader,
	seenFiles map[string]bool,
	result *ValidationResult,
) {
	result.SectionCount++
	secLabel := sec.Title
	if secLabel == "" {
		secLabel = secPath
	}

	// Check explicit files
	for _, f := range sec.Files {
		fullPath := resolvePath(cfg.BaseDir, f)
		fi, err := os.Stat(fullPath)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Section:  secLabel,
				Path:     f,
				Message:  fmt.Sprintf("explicit file does not exist: %v", err),
			})
			result.ErrorCount++
			continue
		}

		if fi.IsDir() {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Section:  secLabel,
				Path:     f,
				Message:  "explicit file path points to a directory, not a file",
			})
			result.ErrorCount++
			continue
		}

		validateFileContent(fullPath, f, secLabel, cfg.Output.FrontmatterMode, seenFiles, result)
	}

	// Check directory path and pattern
	if strings.TrimSpace(sec.Path) != "" {
		dirPath := resolvePath(cfg.BaseDir, sec.Path)
		fi, err := os.Stat(dirPath)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Section:  secLabel,
				Path:     sec.Path,
				Message:  fmt.Sprintf("section directory path does not exist: %v", err),
			})
			result.ErrorCount++
		} else if !fi.IsDir() {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Section:  secLabel,
				Path:     sec.Path,
				Message:  "section path must be a directory",
			})
			result.ErrorCount++
		} else {
			// Discover files via reader
			files, err := reader.DiscoverFiles(sec)
			if err != nil {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					Severity: SeverityError,
					Section:  secLabel,
					Path:     sec.Path,
					Message:  fmt.Sprintf("failed to expand pattern: %v", err),
				})
				result.ErrorCount++
			} else if len(files) == 0 && len(sec.Files) == 0 {
				pat := sec.Pattern
				if pat == "" {
					if sec.Recursive {
						pat = "**/*.md"
					} else {
						pat = "*.md"
					}
				}
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					Severity: SeverityWarning,
					Section:  secLabel,
					Path:     sec.Path,
					Message:  fmt.Sprintf("pattern %q in %s matched 0 files", pat, sec.Path),
				})
				result.WarningCount++
			} else {
				for _, df := range files {
					validateFileContent(df.FullPath, df.RelativePath, secLabel, cfg.Output.FrontmatterMode, seenFiles, result)
				}
			}
		}
	}

	// Validate subsections recursively
	for i, sub := range sec.Subsections {
		validateSectionDeep(sub, fmt.Sprintf("%s.subsections[%d]", secPath, i), cfg, reader, seenFiles, result)
	}
}

func validateFileContent(
	fullPath string,
	displayPath string,
	secLabel string,
	fmMode string,
	seenFiles map[string]bool,
	result *ValidationResult,
) {
	if seenFiles[fullPath] {
		return
	}
	seenFiles[fullPath] = true
	result.ResolvedFiles = append(result.ResolvedFiles, displayPath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Section:  secLabel,
			Path:     displayPath,
			Message:  fmt.Sprintf("failed to read file: %v", err),
		})
		result.ErrorCount++
		return
	}

	// Validate frontmatter and title resolution
	_, fmErr := parser.ProcessFrontmatter(content, fullPath, fmMode)
	if fmErr != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Section:  secLabel,
			Path:     displayPath,
			Message:  fmt.Sprintf("frontmatter error: %v", fmErr),
		})
		result.ErrorCount++
		return
	}
}

func resolvePath(baseDir string, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	if baseDir == "" {
		return p
	}
	return filepath.Join(baseDir, p)
}
