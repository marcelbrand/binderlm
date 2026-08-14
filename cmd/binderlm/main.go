package main

import (
	"fmt"
	"os"

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
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to config file (default: binderlm.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
