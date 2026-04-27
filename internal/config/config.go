package config

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// reservedProfileName is used internally as the cache file key when no
// profile is selected. Configs that define a profile by this name would
// collide with the common-only cache slot, so Load rejects them.
const reservedProfileName = "common"

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

	if _, exists := cfg.Profiles[reservedProfileName]; exists {
		return nil, fmt.Errorf("profile name %q is reserved (used as the cache key when --profile is omitted)", reservedProfileName)
	}

	return &cfg, nil
}

// ResolveProfile merges the common section with the named profile. Profile
// values override common values for the same key.
//
// Pass profile="" to resolve common only — useful for configs that don't need
// per-environment differentiation. If profile is non-empty and not defined in
// the config, ResolveProfile returns an error listing available profiles.
func (c *Config) ResolveProfile(profile string) (Section, error) {
	if profile != "" {
		if _, ok := c.Profiles[profile]; !ok {
			available := c.profileNames()
			if len(available) == 0 {
				return Section{}, fmt.Errorf("profile %q not defined; this config has no profiles. Run without --profile to load only the common section", profile)
			}
			return Section{}, fmt.Errorf("profile %q not defined; available: %s", profile, strings.Join(available, ", "))
		}
	}

	resolved := Section{
		Vars:    make(map[string]string),
		Secrets: make(map[string]string),
	}

	for k, v := range c.Common.Vars {
		resolved.Vars[k] = v
	}
	for k, v := range c.Common.Secrets {
		resolved.Secrets[k] = v
	}

	if profile != "" {
		p := c.Profiles[profile]
		for k, v := range p.Vars {
			resolved.Vars[k] = v
		}
		for k, v := range p.Secrets {
			resolved.Secrets[k] = v
		}
	}

	return resolved, nil
}

// CacheKey returns the file slot used to cache this profile's resolved
// secrets. When profile is empty, returns the reserved common-only slot.
func (c *Config) CacheKey(profile string) string {
	if profile == "" {
		return reservedProfileName
	}
	return profile
}

func (c *Config) profileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
