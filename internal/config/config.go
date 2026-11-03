package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	DefaultBaseURL = "https://cli.encrata.com"
	DefaultAppURL  = "https://app.encrata.com"
	EnvAPIKey      = "ENCRATA_API_KEY"
	EnvBaseURL     = "ENCRATA_BASE_URL"
	EnvAppURL      = "ENCRATA_APP_URL"
)

// NotifyEvents are the job outcomes a desktop notification can be sent for.
var NotifyEvents = []string{"completed", "failed", "cancelled"}

type Config struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
	// AppURL is the dashboard that notification "View results" links open.
	AppURL string `mapstructure:"app_url"`
	Output string `mapstructure:"output"`
	// Notifications are local, per-machine desktop notification preferences.
	// They are off until the user opts in; nothing is stored server-side.
	Notifications NotifyPrefs `mapstructure:"notifications"`
}

// NotifyPrefs controls desktop notifications for finished bulk jobs.
type NotifyPrefs struct {
	Enabled bool     `mapstructure:"enabled"`
	On      []string `mapstructure:"on"` // subset of NotifyEvents
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(filepath.Join(home, ".encrata"))
	}
	viper.AddConfigPath(".")

	viper.SetDefault("base_url", DefaultBaseURL)
	viper.SetDefault("app_url", DefaultAppURL)
	viper.SetDefault("output", "table")
	viper.SetDefault("notifications.enabled", false)
	viper.SetDefault("notifications.on", NotifyEvents)

	viper.SetEnvPrefix("ENCRATA")
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()

	cfg := &Config{
		APIKey:  viper.GetString("api_key"),
		BaseURL: viper.GetString("base_url"),
		AppURL:  viper.GetString("app_url"),
		Output:  viper.GetString("output"),
		Notifications: NotifyPrefs{
			Enabled: viper.GetBool("notifications.enabled"),
			On:      viper.GetStringSlice("notifications.on"),
		},
	}

	// Env vars override config file
	if key := os.Getenv(EnvAPIKey); key != "" {
		cfg.APIKey = key
	}
	if url := os.Getenv(EnvBaseURL); url != "" {
		cfg.BaseURL = url
	}
	if url := os.Getenv(EnvAppURL); url != "" {
		cfg.AppURL = url
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
	viper.Set("notifications.enabled", cfg.Notifications.Enabled)
	viper.Set("notifications.on", cfg.Notifications.On)

	configPath := filepath.Join(configDir, "config.yaml")
	if err := viper.WriteConfigAs(configPath); err != nil {
		return err
	}
	// The config stores the API key in plaintext, so restrict it to the owner.
	return os.Chmod(configPath, 0o600)
}

func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API key required. Set via ENCRATA_API_KEY env var or run: encrata config set-key <key>")
	}
	return nil
}
