package main

import (
	"fmt"

	"github.com/marcelbrand/binderlm/internal/drive"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove cached Google OAuth2 credentials",
	Long:  `The logout command deletes the local OAuth token stored in ~/.config/binderlm/token.json.`,
	RunE:  runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command, args []string) error {
	storage := drive.NewTokenStorage()
	tokenPath, _ := storage.TokenPath()

	if !storage.Exists() {
		fmt.Println("ℹ️  No cached OAuth token found. You are already logged out.")
		return nil
	}

	if err := storage.Delete(); err != nil {
		return fmt.Errorf("failed to remove token at %s: %w", tokenPath, err)
	}

	fmt.Printf("✅ Successfully logged out. Removed cached credentials from %s\n", tokenPath)
	return nil
}
