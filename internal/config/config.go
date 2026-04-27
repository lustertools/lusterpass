package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Section struct {
	Vars    map[string]string `yaml:"vars"`
	Secrets map[string]string `yaml:"secrets"`
}

type Config struct {
	Account  string             `yaml:"account"`
	Project  string             `yaml:"project"`
	Common   Section            `yaml:"common"`
	Profiles map[string]Section `yaml:"profiles"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// ResolveProfile merges common + profile sections.
// Profile values override common values for the same key.
func (c *Config) ResolveProfile(profile string) Section {
	resolved := Section{
		Vars:    make(map[string]string),
		Secrets: make(map[string]string),
	}

	// Layer 1: common
	for k, v := range c.Common.Vars {
		resolved.Vars[k] = v
	}
	for k, v := range c.Common.Secrets {
		resolved.Secrets[k] = v
	}

	// Layer 2: profile overrides
	if p, ok := c.Profiles[profile]; ok {
		for k, v := range p.Vars {
			resolved.Vars[k] = v
		}
		for k, v := range p.Secrets {
			resolved.Secrets[k] = v
		}
	}

	return resolved
}
