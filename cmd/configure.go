package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// readInput is a helper function to read a line of input from stdin with a prompt.
func readInput(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Manage nbctl configurations and profiles",
	Long: `The configure command allows you to set up and manage different profiles for nbctl.
Each profile can have its own API endpoint, key, username, and default account.
Use 'nbctl configure add' to create or update a profile.
Use 'nbctl configure set-current <profile-name>' to switch the active profile.
Use 'nbctl configure list' to see all configured profiles.
Use 'nbctl configure current' to see the currently active profile.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}
var configureAddCmd = &cobra.Command{
	Use:   "add [profile-name]",
	Short: "Add or update a configuration profile",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := "default"
		if len(args) > 0 {
			profileName = args[0]
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("finding home directory: %w", err)
		}
		configPath := filepath.Join(home, ".nudgebee")
		configFile := filepath.Join(configPath, "config.yaml")
		// Read existing config to check for profile existence
		_ = viper.ReadInConfig() // Ignore error if config doesn't exist yet

		// Check if profile already exists
		if viper.IsSet(fmt.Sprintf("profiles.%s", profileName)) {
			response, err := readInput(fmt.Sprintf("Profile '%s' already exists. Overwrite? (y/N): ", profileName))
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}
			if strings.ToLower(response) != "y" {
				fmt.Println("Profile not overwritten.")
				return nil
			}
		}

		// Get existing profiles or create a new map
		profilesMap := viper.GetStringMap("profiles")
		if profilesMap == nil {
			profilesMap = make(map[string]any)
		}

		// Create a map for the current profile's settings
		profileSettings := make(map[string]string)

		endpoint, err := readInput("Enter Nudgebee API Endpoint (e.g., https://api.nudgebee.com): ")
		if err != nil {
			return fmt.Errorf("reading API endpoint: %w", err)
		}
		profileSettings["endpoint"] = endpoint

		apiKey, err := readInput("Enter Nudgebee API Key: ")
		if err != nil {
			return fmt.Errorf("reading API key: %w", err)
		}
		profileSettings["api-key"] = apiKey

		username, err := readInput("Enter your Nudgebee Username (e.g., your email): ")
		if err != nil {
			return fmt.Errorf("reading username: %w", err)
		}
		profileSettings["username"] = username

		defaultAccount, err := readInput("Enter Default Account ID: ")
		if err != nil {
			return fmt.Errorf("reading default account ID: %w", err)
		}
		profileSettings["account-id"] = defaultAccount

		// Validate the configuration by making a simple API call
		fmt.Println("Validating configuration...")
		gqlClient := client.NewClient(client.WithApiKey(apiKey), client.WithEndpoint(endpoint), client.WithUsername(username))
		req := client.NewRequest(`
			query {
				cloud_accounts: accounts_list(where: {}, limit: 1, offset: 0) {
					rows {
						id
					}
				}
			}`)
		var respData struct {
			CloudAccounts struct {
				Rows []struct {
					ID string `json:"id"`
				} `json:"rows"`
			} `json:"cloud_accounts"`
		}
		if err := gqlClient.Run(context.Background(), req, &respData); err != nil {
			fmt.Println("Configuration validation failed. Please check your credentials and re-run 'nbctl configure add'.")
			return err
		}
		fmt.Println("Configuration validated successfully.")

		// Add/update the profile settings in the profiles map
		profilesMap[profileName] = profileSettings
		viper.Set("profiles", profilesMap) // Update the global viper instance with the modified profiles map

		if err := os.MkdirAll(configPath, 0o700); err != nil {
			return fmt.Errorf("creating config directory: %w", err)
		}

		// Create a new Viper instance to write only profile-related data
		writerViper := viper.New()
		writerViper.Set("profiles", viper.GetStringMap("profiles")) // This will now get the explicitly updated map
		writerViper.Set("current-profile", viper.GetString("current-profile"))

		if err := writerViper.WriteConfigAs(configFile); err != nil {
			return fmt.Errorf("writing config file: %w", err)
		}
		// Config stores plaintext API keys; restrict to the owner.
		if err := os.Chmod(configFile, 0o600); err != nil {
			return fmt.Errorf("restricting config file permissions: %w", err)
		}

		// If this is the first profile, or if current-profile is not set, make it current
		if viper.GetString("current-profile") == "" || viper.GetString("current-profile") == profileName {
			viper.Set("current-profile", profileName)
			// Re-write with updated current-profile
			writerViper.Set("current-profile", profileName)
			if err := writerViper.WriteConfigAs(configFile); err != nil {
				return fmt.Errorf("setting current profile: %w", err)
			}
			if err := os.Chmod(configFile, 0o600); err != nil {
				return fmt.Errorf("restricting config file permissions: %w", err)
			}
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Profile '", profileName, "' saved to", configFile); err != nil {
			_ = err
		}
		return nil
	},
}
var configureSetCurrentCmd = &cobra.Command{
	Use:   "set-current <profile-name>",
	Short: "Set the current active configuration profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]
		_ = viper.ReadInConfig() // Ensure config is read
		if !viper.IsSet(fmt.Sprintf("profiles.%s", profileName)) {
			return fmt.Errorf("profile '%s' not found", profileName)
		}
		viper.Set("current-profile", profileName)
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("finding home directory: %w", err)
		}
		configPath := filepath.Join(home, ".nudgebee")
		configFile := filepath.Join(configPath, "config.yaml")

		// Create a new Viper instance to write only profile-related data
		writerViper := viper.New()
		writerViper.Set("profiles", viper.GetStringMap("profiles"))
		writerViper.Set("current-profile", profileName) // Set the current profile in the writerViper

		if err := writerViper.WriteConfigAs(configFile); err != nil {
			return fmt.Errorf("writing config file: %w", err)
		}
		if err := os.Chmod(configFile, 0o600); err != nil {
			return fmt.Errorf("restricting config file permissions: %w", err)
		}
		fmt.Printf("Current profile set to '%s'.\n", profileName)
		return nil
	},
}
var configureListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = viper.ReadInConfig() // Ensure config is read
		profiles := viper.GetStringMapString("profiles")
		if len(profiles) == 0 {
			fmt.Println("No profiles configured. Use 'nbctl configure add' to create one.")
			return nil
		}
		currentProfile := viper.GetString("current-profile")
		fmt.Println("Configured Profiles:")
		for profileName := range profiles {
			indicator := ""
			if profileName == currentProfile {
				indicator = " (current)"
			}
			fmt.Printf("- %s%s\n", profileName, indicator)
		}
		return nil
	},
}
var configureCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the currently active configuration profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = viper.ReadInConfig() // Ensure config is read
		currentProfile := viper.GetString("current-profile")
		if currentProfile == "" {
			fmt.Println("No current profile set. Use 'nbctl configure set-current <profile-name>' or 'nbctl configure add' to create/set one.")
		} else {
			fmt.Printf("Current active profile: %s\n", currentProfile)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configureCmd)
	configureCmd.AddCommand(configureAddCmd)
	configureCmd.AddCommand(configureSetCurrentCmd)
	configureCmd.AddCommand(configureListCmd)
	configureCmd.AddCommand(configureCurrentCmd)
}
