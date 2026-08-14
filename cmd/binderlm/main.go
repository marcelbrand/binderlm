package main

import (
	"fmt"
	"os"

	"github.com/marcelbrand/binderlm/internal/drive"
	"github.com/spf13/cobra"
)

var (
	configFile string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "binderlm",
	Short: "binderlm — Git-to-NotebookLM Markdown Aggregator & Sync Tool",
	Long: `binderlm consolidates decentralized Markdown documentation across repositories,
microservices, and packages into a unified, hierarchically structured context document
for Google NotebookLM and enterprise RAG pipelines.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to config file (default: binderlm.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if drive.IsAuthError(err) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

