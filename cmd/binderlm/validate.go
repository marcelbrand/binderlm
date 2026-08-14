package main

import (
	"fmt"
	"os"

	"github.com/marcelbrand/binderlm/internal/config"
	"github.com/marcelbrand/binderlm/internal/validator"
	"github.com/spf13/cobra"
)

var (
	strictValidation bool
	checkDrive       bool
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration, file paths, frontmatter, and Google Drive auth",
	Long: `The validate command performs static analysis on binderlm.yaml, verifies
file existence and glob resolution, checks YAML frontmatter syntax across all
referenced markdown files, and verifies Google Drive configuration and credentials.`,
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().BoolVar(&strictValidation, "strict", false, "Treat warnings as errors (e.g. globs matching 0 files)")
	validateCmd.Flags().BoolVar(&checkDrive, "check-drive", false, "Verify Google Drive authentication and folder configuration even if drive is disabled")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	opts := validator.ValidationOptions{
		CheckDrive: checkDrive,
		Strict:     strictValidation,
	}

	result, err := validator.Validate(cmd.Context(), cfg, opts)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Print diagnostics
	for _, d := range result.Diagnostics {
		switch d.Severity {
		case validator.SeverityError:
			fmt.Fprintf(os.Stderr, "  %s\n", d.String())
		case validator.SeverityWarning:
			fmt.Fprintf(os.Stderr, "  %s\n", d.String())
		case validator.SeverityInfo:
			if verbose {
				fmt.Fprintf(os.Stderr, "  %s\n", d.String())
			}
		}
	}

	// Print Summary
	cfgPath := configFile
	if cfgPath == "" {
		cfgPath = "binderlm.yaml"
	}
	fmt.Fprintf(os.Stderr, "\nValidation Summary:\n")
	fmt.Fprintf(os.Stderr, "  Config File:    %s\n", cfgPath)
	fmt.Fprintf(os.Stderr, "  Sections:       %d\n", result.SectionCount)
	fmt.Fprintf(os.Stderr, "  Files Resolved: %d\n", result.FileCount)
	if result.DriveChecked {
		if result.DriveValid {
			fmt.Fprintf(os.Stderr, "  Google Drive:   ✓ Authenticated (Folder ID: %s)\n", cfg.Drive.FolderID)
		} else {
			fmt.Fprintf(os.Stderr, "  Google Drive:   ✗ Auth / Config Issue\n")
		}
	}
	fmt.Fprintf(os.Stderr, "  Warnings:       %d\n", result.WarningCount)
	fmt.Fprintf(os.Stderr, "  Errors:         %d\n", result.ErrorCount)

	if !result.Valid {
		if result.HasAuthError && result.AuthErr != nil {
			return result.AuthErr
		}
		if result.ErrorCount > 0 {
			return fmt.Errorf("validation failed with %d error(s)", result.ErrorCount)
		}
		if strictValidation && result.WarningCount > 0 {
			return fmt.Errorf("validation failed in strict mode with %d warning(s)", result.WarningCount)
		}
	}

	fmt.Fprintf(os.Stderr, "\n✓ Configuration and Markdown sources are valid!\n")
	return nil
}
