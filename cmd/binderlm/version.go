package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	Version   = "v0.1.0"
	GitCommit = "none"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("binderlm %s (commit: %s, built: %s, runtime: %s %s/%s)\n",
			Version, GitCommit, BuildDate, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
