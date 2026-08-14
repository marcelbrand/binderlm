package main

import (
	"fmt"
	"time"

	"github.com/marcelbrand/binderlm/internal/drive"
	"github.com/spf13/cobra"
)

var (
	loginPort         int
	loginNoBrowser    bool
	loginClientID     string
	loginClientSecret string
	loginTimeout      time.Duration
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in with your personal Google Drive account via OAuth2",
	Long: `The login command opens an interactive Google OAuth2 consent screen in your browser
and saves the access and refresh tokens locally in ~/.config/binderlm/token.json.

This enables you to run 'binderlm sync' directly to your personal Google Drive
without configuring a Google Cloud Service Account.`,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().IntVar(&loginPort, "port", 8085, "Local callback port for OAuth loopback listener")
	loginCmd.Flags().BoolVar(&loginNoBrowser, "no-browser", false, "Do not automatically open browser; display URL only")
	loginCmd.Flags().StringVar(&loginClientID, "client-id", "", "Custom Google OAuth Client ID (optional override)")
	loginCmd.Flags().StringVar(&loginClientSecret, "client-secret", "", "Custom Google OAuth Client Secret (optional override)")
	loginCmd.Flags().DurationVar(&loginTimeout, "timeout", 3*time.Minute, "Maximum wait time for OAuth authorization")
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	mgr := drive.NewOAuthManager(loginClientID, loginClientSecret, nil)

	fmt.Println("🚀 Initiating Google Drive OAuth2 Login...")

	res, err := mgr.Login(cmd.Context(), drive.LoginOptions{
		Port:        loginPort,
		OpenBrowser: !loginNoBrowser,
		Timeout:     loginTimeout,
	})
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	tokenPath, _ := mgr.Storage.TokenPath()

	fmt.Println("\n✅ Successfully authenticated!")
	if res.Email != "" {
		fmt.Printf("   Account:    %s\n", res.Email)
	}
	fmt.Printf("   Token Path: %s\n", tokenPath)
	fmt.Println("\nYou can now run 'binderlm sync' to upload documents to your personal Google Drive.")
	return nil
}
