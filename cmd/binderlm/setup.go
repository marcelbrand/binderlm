package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/marcelbrand/binderlm/internal/drive"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard for Google authentication credentials",
	Long: `The setup command launches an interactive wizard to configure OAuth Desktop Client
credentials or Service Account JSON keys in ~/.config/binderlm/.`,
	RunE: runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
	authCmd.AddCommand(&cobra.Command{
		Use:   "setup",
		Short: "Interactive setup wizard for Google authentication credentials",
		RunE:  runSetup,
	})
}

func runSetup(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🔧 binderlm Authentication Setup Wizard")
	fmt.Printf("Config Directory: %s\n", drive.GetConfigDir())
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("Please choose an authentication setup option:")
	fmt.Println("  1) Personal Google Drive Account (OAuth2 Login) [Recommended for individual devs]")
	fmt.Println("  2) Google Cloud Service Account (JSON Key) [Recommended for CI/CD & Teams]")
	fmt.Println("  3) Exit")
	fmt.Print("\nEnter choice [1-3] (default 1): ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = "1"
	}

	switch choice {
	case "1":
		return setupOAuth(cmd, reader)
	case "2":
		return setupServiceAccount(cmd, reader)
	case "3":
		fmt.Println("Setup cancelled.")
		return nil
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}
}

func setupOAuth(cmd *cobra.Command, reader *bufio.Reader) error {
	fmt.Println("\n🔑 Step 1: OAuth Desktop Client Credentials")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("Create an OAuth Client ID in Google Cloud Console:")
	fmt.Println("  1. Go to APIs & Services > Credentials > Create Credentials > OAuth client ID")
	fmt.Println("  2. Application type: Desktop App")
	fmt.Println()

	existingID, existingSecret, _ := drive.LoadClientCredentials("", "")
	defaultIDPrompt := ""
	if existingID != "" && existingID != drive.DefaultClientID {
		defaultIDPrompt = fmt.Sprintf(" [%s...]", existingID[:min(12, len(existingID))])
	}

	fmt.Printf("Enter OAuth Client ID%s: ", defaultIDPrompt)
	clientID, _ := reader.ReadString('\n')
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		clientID = existingID
	}

	fmt.Print("Enter OAuth Client Secret: ")
	clientSecret, _ := reader.ReadString('\n')
	clientSecret = strings.TrimSpace(clientSecret)
	if clientSecret == "" {
		clientSecret = existingSecret
	}

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("Client ID and Client Secret cannot be empty")
	}

	if err := drive.SaveClientCredentials(clientID, clientSecret); err != nil {
		return fmt.Errorf("failed to save client credentials: %w", err)
	}

	fmt.Printf("\n✅ Saved OAuth Client Credentials to %s\n", drive.GetClientCredentialsPath())

	fmt.Print("\nWould you like to log in via browser now? [Y/n]: ")
	loginChoice, _ := reader.ReadString('\n')
	loginChoice = strings.TrimSpace(strings.ToLower(loginChoice))

	if loginChoice == "" || loginChoice == "y" || loginChoice == "yes" {
		fmt.Println()
		return runLogin(cmd, nil)
	}

	fmt.Println("\nYou can log in at any time by running: binderlm login")
	return nil
}

func setupServiceAccount(cmd *cobra.Command, reader *bufio.Reader) error {
	fmt.Println("\n🤖 Step 1: Service Account Key Setup")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("Enter the path to your downloaded Service Account JSON key file")
	fmt.Print("File path: ")

	filePath, _ := reader.ReadString('\n')
	filePath = strings.TrimSpace(filePath)

	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Remove quotes if path was dragged & dropped
	filePath = strings.Trim(filePath, `"'`)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("could not read file at %s: %w", filePath, err)
	}

	var js struct {
		ClientEmail string `json:"client_email"`
		Type        string `json:"type"`
	}
	if err := json.Unmarshal(data, &js); err != nil {
		return fmt.Errorf("invalid Service Account JSON: %w", err)
	}

	if err := drive.SaveServiceAccountKey(data); err != nil {
		return fmt.Errorf("failed to save service account key: %w", err)
	}

	fmt.Printf("\n✅ Saved Service Account Key to %s\n", drive.GetServiceAccountPath())
	if js.ClientEmail != "" {
		fmt.Printf("   Service Account: %s\n", js.ClientEmail)
	}

	fmt.Println("\n👉 Next steps:")
	fmt.Println("   1. Open Google Drive and share your target folder with this Service Account as 'Editor'.")
	fmt.Println("   2. Run 'binderlm sync' to upload documentation.")
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
