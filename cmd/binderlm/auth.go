package main

import (
	"fmt"

	"github.com/marcelbrand/binderlm/internal/drive"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Inspect or manage authentication credentials",
	Long:  `Inspect active authentication credentials and Google Drive authorization state.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuthStatus(cmd, args)
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display active Google Drive authentication method and account",
	RunE:  runAuthStatus,
}

func init() {
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	status := drive.GetAuthStatus(cmd.Context())

	fmt.Println("🔐 binderlm Authentication Status")
	fmt.Println("──────────────────────────────────────────")
	fmt.Printf("Method:      %s\n", status.Description)

	if status.Account != "" {
		fmt.Printf("Account:     %s\n", status.Account)
	}

	if status.TokenFile != "" {
		fmt.Printf("Token File:  %s\n", status.TokenFile)
	}

	if status.Type == drive.AuthTypeNone {
		fmt.Println("\n👉 To authenticate:")
		fmt.Println("   • Local Developer: Run 'binderlm login' to sign in with your Google account.")
		fmt.Println("   • CI/CD / Machine: Set GOOGLE_APPLICATION_CREDENTIALS_JSON or GOOGLE_APPLICATION_CREDENTIALS.")
	}

	return nil
}
