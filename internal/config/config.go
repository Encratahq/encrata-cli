package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	DefaultBaseURL = "https://api.encrata.com"
	EnvAPIKey      = "ENCRATA_API_KEY"
	EnvBaseURL     = "ENCRATA_BASE_URL"
)

type Config struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
	Output  string `mapstructure:"output"`
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(filepath.Join(home, ".encrata"))
	}
	viper.AddConfigPath(".")

	viper.SetDefault("base_url", DefaultBaseURL)
	viper.SetDefault("output", "table")

	viper.SetEnvPrefix("ENCRATA")
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()

	cfg := &Config{
		APIKey:  viper.GetString("api_key"),
		BaseURL: viper.GetString("base_url"),
		Output:  viper.GetString("output"),
	}

	// Env vars override config file
	if key := os.Getenv(EnvAPIKey); key != "" {
		cfg.APIKey = key
	}
	if url := os.Getenv(EnvBaseURL); url != "" {
		cfg.BaseURL = url
	}

	return cfg
}

func Save(cfg *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".encrata")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	viper.Set("api_key", cfg.APIKey)
	viper.Set("base_url", cfg.BaseURL)
	viper.Set("output", cfg.Output)

	configPath := filepath.Join(configDir, "config.yaml")
	return viper.WriteConfigAs(configPath)
}

func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API key required. Set via ENCRATA_API_KEY env var or run: encrata config set-key <key>")
	}
	return nil
}
