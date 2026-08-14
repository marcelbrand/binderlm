package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/marcelbrand/binderlm/internal/config"
	"github.com/marcelbrand/binderlm/internal/drive"
	"github.com/marcelbrand/binderlm/internal/stitcher"
	"github.com/spf13/cobra"
)

var (
	folderIDFlag string
	dryRun       bool
	keepLocal    bool
	syncOutput   string
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Assemble markdown documentation and synchronize directly to Google Drive",
	Long: `The sync command reads the configuration, compiles the master context document,
and idempotently uploads or updates the file in the designated Google Drive folder
for consumption by Google NotebookLM and enterprise RAG pipelines.`,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().StringVar(&folderIDFlag, "folder-id", "", "Override target Google Drive folder ID")
	syncCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Perform all parsing and remote checks without uploading")
	syncCmd.Flags().BoolVar(&keepLocal, "keep-local", false, "Save the compiled master document locally as well as syncing")
	syncCmd.Flags().StringVarP(&syncOutput, "output", "o", "", "Override local output file path when using --keep-local")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	cfg, err := config.Load(configFile)
	if err != nil {
		return err
	}

	// Apply CLI folder ID override if specified
	if folderIDFlag != "" {
		cfg.Drive.FolderID = folderIDFlag
	}

	// Ensure folder ID is configured
	if strings.TrimSpace(cfg.Drive.FolderID) == "" {
		return fmt.Errorf("Google Drive folder ID is not specified. Provide --folder-id or set drive.folder_id in config / GDRIVE_FOLDER_ID environment variable")
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[info] Loaded configuration from %s (BaseDir: %s)\n", configFile, cfg.BaseDir)
		fmt.Fprintf(os.Stderr, "[info] Target Google Drive folder ID: %s (dry-run: %t)\n", cfg.Drive.FolderID, dryRun)
	}

	// Authenticate and initialize Google Drive client
	srv, err := drive.NewService(cmd.Context())
	if err != nil {
		return err
	}

	// Assemble master markdown in-memory
	assembler := stitcher.NewAssembler(cfg)
	doc, err := assembler.Assemble(cmd.Context())
	if err != nil {
		return fmt.Errorf("assembly failed: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[info] Assembled %d files (%d headings, %d bytes)\n",
			doc.FileCount, doc.HeadingCount, doc.ByteCount)
	}

	// If --keep-local requested, write to disk
	if keepLocal {
		targetOut := syncOutput
		if targetOut == "" {
			targetOut = cfg.Output.Filename
		}
		if !filepath.IsAbs(targetOut) && cfg.BaseDir != "" {
			targetOut = filepath.Join(cfg.BaseDir, targetOut)
		}

		outDir := filepath.Dir(targetOut)
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("failed to create local output directory %s: %w", outDir, err)
		}

		if err := os.WriteFile(targetOut, doc.Content, 0644); err != nil {
			return fmt.Errorf("failed to save local copy to %s: %w", targetOut, err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "[info] Saved local copy to %s\n", targetOut)
		}
	}

	// Perform idempotent Drive sync
	uploader := drive.NewUploader(srv,
		drive.WithMimeType(cfg.Drive.MimeType),
		drive.WithDryRun(dryRun),
	)

	result, err := uploader.Sync(cmd.Context(), cfg.Drive.FolderID, cfg.Output.Filename, bytes.NewReader(doc.Content))
	if err != nil {
		return fmt.Errorf("drive sync failed: %w", err)
	}

	elapsed := time.Since(startTime).Round(time.Millisecond)

	if dryRun {
		fmt.Fprintf(os.Stderr, "[dry-run] %s: %q in folder %s (no mutations made)\n",
			result.Action, result.Filename, result.FolderID)
		if result.FileID != "" {
			fmt.Fprintf(os.Stderr, "          Existing File ID: %s (WebViewLink: %s)\n", result.FileID, result.WebLink)
		}
		return nil
	}

	fmt.Fprintf(os.Stderr, "✓ Successfully synchronized %q to Google Drive (%s, ID: %s, %d bytes) in %s\n",
		result.Filename, result.Action, result.FileID, result.Size, elapsed)
	if result.WebLink != "" {
		fmt.Fprintf(os.Stderr, "  View in Drive: %s\n", result.WebLink)
	}

	return nil
}
