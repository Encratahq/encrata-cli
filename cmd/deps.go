package cmd

import (
	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/config"
)

// appDeps is the single, injectable dependency seam for the command layer.
// Replacing package-scattered globals with one struct keeps commands testable
// (swap newAPI/cfg in tests) while preserving Cobra's closure-based handlers.
type appDeps struct {
	cfg    *config.Config
	newAPI func(*config.Config) (api.API, error)
}

var app = &appDeps{newAPI: defaultNewAPI}

// defaultNewAPI validates configuration and returns a ready API client.
func defaultNewAPI(cfg *config.Config) (api.API, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return api.New(cfg.BaseURL, cfg.APIKey), nil
}
