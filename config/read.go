package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// ReadConfig reads the configuration from the specified YAML file.
func ReadConfig(configPath string) (Config, error) {
	var cfg Config

	// Configure viper to use the specified config file
	viper.SetConfigFile(configPath)
	viper.SetConfigType("json")

	// Read in the config file
	if err := viper.ReadInConfig(); err != nil {
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal the file contents into the Config struct
	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

// MustReadConfig reads the configuration or panics if there's an error.
func MustReadConfig(configPath string) Config {
	cfg, err := ReadConfig(configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	return cfg
}
