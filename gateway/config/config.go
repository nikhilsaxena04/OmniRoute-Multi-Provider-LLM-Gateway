package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type ProviderConfig struct {
	BaseURL   string `yaml:"base_url"`
	Model     string `yaml:"model"`
	APIKeyEnv string `yaml:"api_key_env"`
	APIKey    string `yaml:"-"` // Loaded from env, not yaml
}

type RoutingConfig struct {
	Priorities    []string `yaml:"priorities"`
	CostThreshold float64  `yaml:"cost_threshold"`
}

type Config struct {
	Providers map[string]*ProviderConfig `yaml:"providers"`
	Routing   RoutingConfig              `yaml:"-"` // Loaded separately
	Port      string                     `yaml:"-"` // Loaded from env
}

// Load reads .env and parses the YAML configs
func Load(providersPath, routingPath string) (*Config, error) {
	// Attempt to load .env file from the project root or current dir
	// We ignore the error because .env might not exist in prod (env vars injected directly)
	_ = godotenv.Load(".env")
	_ = godotenv.Load(filepath.Join("..", ".env")) // if running from gateway/

	data, err := os.ReadFile(providersPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read providers config %s: %w", providersPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse providers yaml: %w", err)
	}

	routingData, err := os.ReadFile(routingPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read routing config %s: %w", routingPath, err)
	}
	if err := yaml.Unmarshal(routingData, &cfg.Routing); err != nil {
		return nil, fmt.Errorf("failed to parse routing yaml: %w", err)
	}

	// Load secrets and environment-specific settings
	cfg.Port = os.Getenv("GATEWAY_PORT")
	if cfg.Port == "" {
		cfg.Port = "8787" // default
	}

	for name, p := range cfg.Providers {
		p.APIKey = os.Getenv(p.APIKeyEnv)
		if p.APIKey == "" {
			return nil, fmt.Errorf("missing required environment variable %s for provider %s", p.APIKeyEnv, name)
		}
	}

	return &cfg, nil
}
