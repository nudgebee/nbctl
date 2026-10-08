package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

var inited bool

// Reset is for testing purposes only to reset the init guard and viper.
func Reset() {
	inited = false
	viper.Reset()
}

// IsConfigured reports whether the settings every API call needs are present.
// username is not required: the API key authenticates on its own.
func IsConfigured() bool {
	return viper.GetString("endpoint") != "" &&
		viper.GetString("api-key") != "" &&
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
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "Error reading config file:", err)
		}
	}

	// Load profile-specific configuration
	profile := viper.GetString("profile")
	if profile != "" {
		profiles := viper.GetStringMapString("profiles")
		if _, ok := profiles[profile]; !ok {
			fmt.Fprintf(os.Stderr, "Error: profile '%s' not found\n", profile)
			os.Exit(1)
		}
		applyProfile(profile)
		return
	}

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
		applyProfile(currentProfile)
	}
}

// applyProfile copies a profile's settings into viper. A setting also given as
// a NUDGEBEE_* environment variable keeps the env value, so an environment that
// configures nbctl by env is not overridden by a config file in $HOME.
func applyProfile(name string) {
	for key, value := range viper.GetStringMapString(fmt.Sprintf("profiles.%s", name)) {
		envName := "NUDGEBEE_" + strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
		// Non-empty only: viper ignores an empty env var, so skipping the
		// profile for one would leave the setting blank.
		if os.Getenv(envName) != "" {
			continue
		}
		viper.Set(key, value)
	}
}
