package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the runtime settings for the download proxy.
type Config struct {
	Listen           string   `yaml:"listen"`
	Secret           string   `yaml:"secret"`
	MaxExpireSeconds int64    `yaml:"max_expire_seconds"`
	AllowedDomains   []string `yaml:"allowed_domains"`
}

// Load reads and parses the proxy configuration from the given YAML file.
func Load(path string) (*Config, error) {
	// Load and parse the YAML configuration file from disk.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
