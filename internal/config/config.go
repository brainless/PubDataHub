package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	StoragePath string `mapstructure:"storage_path"`
}

var AppConfig Config

func InitConfig() error {
	configName := "config"
	configType := "json"
	configPath := os.Getenv("PUBDATAHUB_CONFIG_PATH")
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, ".pubdatahub")
	}

	viper.AddConfigPath(configPath)
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)

	viper.SetDefault("storage_path", filepath.Join(configPath, "data"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, create a default one
			fmt.Printf("Config file not found, creating default at %s/%s.%s\n", configPath, configName, configType)
			if err := os.MkdirAll(configPath, 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}
			if err := viper.WriteConfigAs(filepath.Join(configPath, fmt.Sprintf("%s.%s", configName, configType))); err != nil {
				return fmt.Errorf("failed to write default config file: %w", err)
			}
		} else {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Ensure storage path exists
	if err := os.MkdirAll(AppConfig.StoragePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	return nil
}

func SetStoragePath(path string) error {
	viper.Set("storage_path", path)
	return viper.WriteConfig()
}
