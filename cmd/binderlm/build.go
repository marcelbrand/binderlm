package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/marcelbrand/binderlm/internal/config"
	"github.com/marcelbrand/binderlm/internal/stitcher"
	"github.com/spf13/cobra"
)

var (
	outputPath string
	useStdout  bool
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Assemble local markdown files into a unified master context document",
	Long: `The build command reads the configuration, discovers all matching markdown files,
safely shifts heading levels via Goldmark AST, processes YAML frontmatter,
and compiles a single cohesive context markdown file.`,
	RunE: runBuild,
}

func init() {
	buildCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Override output file path defined in config")
	buildCmd.Flags().BoolVar(&useStdout, "stdout", false, "Write stitched markdown directly to standard output")
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	cfg, err := config.Load(configFile)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[info] Loaded configuration from %s (BaseDir: %s)\n", configFile, cfg.BaseDir)
	}

	assembler := stitcher.NewAssembler(cfg)
	doc, err := assembler.Assemble(cmd.Context())
	if err != nil {
		return fmt.Errorf("assembly failed: %w", err)
	}

	if useStdout {
		_, err := os.Stdout.Write(doc.Content)
		return err
	}

	// Resolve output path
	targetOut := outputPath
	if targetOut == "" {
		targetOut = cfg.Output.Filename
	}
	if !filepath.IsAbs(targetOut) && cfg.BaseDir != "" {
		targetOut = filepath.Join(cfg.BaseDir, targetOut)
	}

	// Ensure output directory exists
	outDir := filepath.Dir(targetOut)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outDir, err)
	}

	if err := os.WriteFile(targetOut, doc.Content, 0644); err != nil {
		return fmt.Errorf("failed to write output file to %s: %w", targetOut, err)
	}

	elapsed := time.Since(startTime).Round(time.Millisecond)
	fmt.Fprintf(os.Stderr, "✓ Successfully assembled %d files (%d headings, %d bytes) into %s in %s\n",
		doc.FileCount, doc.HeadingCount, doc.ByteCount, targetOut, elapsed)

	return nil
}
