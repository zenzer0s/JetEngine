package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
// Values are read by viper from a config file or environment variables.
type Config struct {
	TelegramBotToken string `mapstructure:"TELEGRAM_BOT_TOKEN"`
	BadgerDBPath     string `mapstructure:"BADGERDB_PATH"`
	// Add other configuration fields as needed
	// e.g., LogLevel string `mapstructure:"LOG_LEVEL"`
	// e.g., ServerPort string `mapstructure:"SERVER_PORT"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	// Set the path to look for the config file in
	viper.AddConfigPath(path)
	// Set the name of the config file (without extension)
	viper.SetConfigName("config")
	// Set the type of the config file
	viper.SetConfigType("yaml") // or json, toml, etc.

	// Allow reading from environment variables
	viper.AutomaticEnv()
	// Optional: Set a prefix for environment variables to avoid conflicts
	// viper.SetEnvPrefix("JETENGINE")
	// Optional: Replace dots with underscores in env var names
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Attempt to read the config file
	err = viper.ReadInConfig()
	if err != nil {
		// Handle errors reading the config file
		// If the file is not found, viper might still find env vars,
		// so we only return error for other types of errors.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; ignore error if env vars are expected
		// Or log this information: fmt.Println("Config file not found, using environment variables.")
	}

	// Unmarshal the config into the Config struct
	err = viper.Unmarshal(&config)
	if err != nil {
		return Config{}, fmt.Errorf("unable to decode into struct: %w", err)
	}

	// --- Add Validation or Default Values Here ---
	if config.TelegramBotToken == "" {
		return Config{}, fmt.Errorf("TELEGRAM_BOT_TOKEN is not set")
	}
	if config.BadgerDBPath == "" {
		// Set a default path if not provided
		config.BadgerDBPath = "./badger_data"
		fmt.Println("BADGERDB_PATH not set, using default:", config.BadgerDBPath)
	}
	// --- End Validation ---

	return config, nil
}

