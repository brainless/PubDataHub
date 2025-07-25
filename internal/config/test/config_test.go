package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brainless/PubDataHub/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestInitConfig(t *testing.T) {
	// Clean up any existing config file before test
	homeDir, _ := os.UserHomeDir()
	testConfigPath := filepath.Join(homeDir, ".pubdatahub_test")
	os.RemoveAll(testConfigPath)

	// Set a temporary config path for testing
	os.Setenv("PUBDATAHUB_CONFIG_PATH", testConfigPath)
	defer os.Unsetenv("PUBDATAHUB_CONFIG_PATH")

	// Reset viper for clean state
	viper.Reset()

	// Test case 1: No config file exists, should create default
	err := config.InitConfig()
	assert.NoError(t, err)
	assert.NotEmpty(t, config.AppConfig.StoragePath)
	assert.True(t, fileExists(filepath.Join(testConfigPath, "config.json")))
	assert.True(t, dirExists(config.AppConfig.StoragePath))

	// Test case 2: Config file exists, should read it
	viper.Reset()
	// Modify the default config file to have a custom storage path
	customStoragePath := filepath.Join(testConfigPath, "custom_data")
	viper.Set("storage_path", customStoragePath)
	viper.WriteConfigAs(filepath.Join(testConfigPath, "config.json"))

	err = config.InitConfig()
	assert.NoError(t, err)
	assert.Equal(t, customStoragePath, config.AppConfig.StoragePath)
	assert.True(t, dirExists(customStoragePath))

	// Clean up after test
	os.RemoveAll(testConfigPath)
}

func TestSetStoragePath(t *testing.T) {
	// Setup similar to InitConfig
	homeDir, _ := os.UserHomeDir()
	testConfigPath := filepath.Join(homeDir, ".pubdatahub_test_set")
	os.RemoveAll(testConfigPath)
	os.Setenv("PUBDATAHUB_CONFIG_PATH", testConfigPath)
	defer os.Unsetenv("PUBDATAHUB_CONFIG_PATH")
	viper.Reset()

	// Initialize config first to ensure a config file exists
	config.InitConfig()

	newPath := filepath.Join(testConfigPath, "new_storage")
	err := config.SetStoragePath(newPath)
	assert.NoError(t, err)

	// Verify by re-initializing config and checking the path
	viper.Reset()
	config.InitConfig()
	assert.Equal(t, newPath, config.AppConfig.StoragePath)
	assert.True(t, dirExists(newPath))

	// Clean up
	os.RemoveAll(testConfigPath)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return !os.IsNotExist(err) && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return !os.IsNotExist(err) && info.IsDir()
}
