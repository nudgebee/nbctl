package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var inited bool

func IsConfigured() bool {
	return viper.GetString("endpoint") != "" &&
		viper.GetString("api-key") != "" &&
		viper.GetString("username") != "" &&
		viper.GetString("account-id") != ""
}

func InitConfig() {
	if inited {
		return
	}
	inited = true

	if os.Getenv("NBCTL_TESTING") == "true" {
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error finding home directory:", err)
	}

	configPath := filepath.Join(home, ".nudgebee")
	viper.AddConfigPath(configPath)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.SetEnvPrefix("NUDGEBEE")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "Error reading config file:", err)
		}
	}

	// Load profile-specific configuration
	currentProfile := viper.GetString("current-profile")
	profiles := viper.GetStringMapString("profiles")

	if currentProfile == "" {
		// If no current profile is set, try to find one
		if len(profiles) == 1 {
			// If only one profile exists, make it current
			for profileName := range profiles {
				currentProfile = profileName
				viper.Set("current-profile", currentProfile)
				break
			}
		} else if len(profiles) > 1 {
			fmt.Fprintln(os.Stderr, "Warning: No current profile set. Please use 'nbctl configure set-profile <profile-name>' to select one.")
		}
	}

	if currentProfile != "" {
		profileSettings := viper.GetStringMapString(fmt.Sprintf("profiles.%s", currentProfile))
		for key, value := range profileSettings {
			viper.Set(key, value)
		}
	}
}