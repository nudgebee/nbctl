package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nudgebee.com/nbctl/pkg/testutil"
)

func TestInitConfig(t *testing.T) {
	t.Run("does not re-initialize", func(t *testing.T) {
		InitConfig()
		// We can't directly test the return, but we can ensure that subsequent calls don't panic.
		InitConfig()
	})

	t.Run("loads config from file", func(t *testing.T) {
		// Create a temporary config file
		tmpdir := t.TempDir()
		configPath := filepath.Join(tmpdir, ".nudgebee")
		require.NoError(t, os.MkdirAll(configPath, 0755))
		configFilePath := filepath.Join(configPath, "config.yaml")
		require.NoError(t, os.WriteFile(configFilePath, []byte("endpoint: http://test.com\napi-key: test-key\nusername: test-user\naccount-id: test-account"), 0644))

		// Point home to our temp dir
		t.Setenv("HOME", tmpdir)

		// Reset viper before running the test
		Reset()

		// Run InitConfig
		InitConfig()

		// Check if the values were loaded
		assert.Equal(t, "http://test.com", viper.GetString("endpoint"))
		assert.Equal(t, "test-key", viper.GetString("api-key"))
		assert.Equal(t, "test-user", viper.GetString("username"))
		assert.Equal(t, "test-account", viper.GetString("account-id"))
	})

	t.Run("loads config from environment variables", func(t *testing.T) {
		// Set environment variables
		t.Setenv("NUDGEBEE_ENDPOINT", "http://env.com")
		t.Setenv("NUDGEBEE_API_KEY", "env-key")
		t.Setenv("NUDGEBEE_USERNAME", "env-user")
		t.Setenv("NUDGEBEE_ACCOUNT_ID", "env-account")

		// Reset viper before running the test
		Reset()

		// Run InitConfig
		InitConfig()

		// Check if the values were loaded
		assert.Equal(t, "http://env.com", viper.GetString("endpoint"))
		assert.Equal(t, "env-key", viper.GetString("api-key"))
		assert.Equal(t, "env-user", viper.GetString("username"))
		assert.Equal(t, "env-account", viper.GetString("account-id"))
	})

	t.Run("loads profile", func(t *testing.T) {
		// Create a temporary config file with profiles
		tmpdir := t.TempDir()
		configPath := filepath.Join(tmpdir, ".nudgebee")
		require.NoError(t, os.MkdirAll(configPath, 0755))
		configFilePath := filepath.Join(configPath, "config.yaml")
		require.NoError(t, os.WriteFile(configFilePath, []byte(`
profiles:
  prof1:
    endpoint: http://prof1.com
    api-key: prof1-key
    username: prof1-user
    account-id: prof1-account
current-profile: prof1
`), 0644))

		// Point home to our temp dir
		t.Setenv("HOME", tmpdir)

		// Reset viper before running the test
		Reset()

		// Run InitConfig
		InitConfig()

		// Check if the values from the profile were loaded
		assert.Equal(t, "http://prof1.com", viper.GetString("endpoint"))
		assert.Equal(t, "prof1-key", viper.GetString("api-key"))
		assert.Equal(t, "prof1-user", viper.GetString("username"))
		assert.Equal(t, "prof1-account", viper.GetString("account-id"))
	})
}

func TestIsConfigured(t *testing.T) {
	t.Run("returns true when all required fields are set", func(t *testing.T) {
		defer testutil.WithViper(map[string]any{
			"endpoint":   "http://test.com",
			"api-key":    "test-key",
			"username":   "test-user",
			"account-id": "test-account",
		})()

		assert.True(t, IsConfigured())
	})

	t.Run("returns false when endpoint is missing", func(t *testing.T) {
		defer testutil.WithViper(map[string]any{
			"endpoint":   "",
			"api-key":    "test-key",
			"username":   "test-user",
			"account-id": "test-account",
		})()

		assert.False(t, IsConfigured())
	})

	t.Run("returns false when api-key is missing", func(t *testing.T) {
		defer testutil.WithViper(map[string]any{
			"endpoint":   "http://test.com",
			"api-key":    "",
			"username":   "test-user",
			"account-id": "test-account",
		})()

		assert.False(t, IsConfigured())
	})

	t.Run("returns false when username is missing", func(t *testing.T) {
		defer testutil.WithViper(map[string]any{
			"endpoint":   "http://test.com",
			"api-key":    "test-key",
			"username":   "",
			"account-id": "test-account",
		})()

		assert.False(t, IsConfigured())
	})

	t.Run("returns false when account-id is missing", func(t *testing.T) {
		defer testutil.WithViper(map[string]any{
			"endpoint":   "http://test.com",
			"api-key":    "test-key",
			"username":   "test-user",
			"account-id": "",
		})()

		assert.False(t, IsConfigured())
	})
}
