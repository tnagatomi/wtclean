// Package config loads and validates the wtm configuration file.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

const DefaultMaxDepth = 5

type Config struct {
	Roots    []string `yaml:"roots"`
	MaxDepth int      `yaml:"max_depth"`
}

// DefaultPath returns the canonical config file path, honoring XDG_CONFIG_HOME.
// macOS's os.UserConfigDir points at ~/Library/Application Support, which the
// spec explicitly avoids, so we resolve XDG ourselves.
func DefaultPath() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "wtm", "config.yml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "wtm", "config.yml"), nil
}

// Load reads and validates the config file at path. Tilde-prefixed roots are
// expanded to absolute paths. Missing max_depth falls back to DefaultMaxDepth.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file not found at %q: run `wtm init` to generate a starter file", path)
		}
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}
	if len(cfg.Roots) == 0 {
		return nil, fmt.Errorf("config %q has no roots: add at least one root directory", path)
	}
	if cfg.MaxDepth <= 0 {
		cfg.MaxDepth = DefaultMaxDepth
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	for i, r := range cfg.Roots {
		cfg.Roots[i] = expandTilde(r, home)
	}
	return &cfg, nil
}

func expandTilde(p, home string) string {
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

// StarterContent is the body of the config file written by `wtm init`.
const StarterContent = `# wtm configuration file.
# See https://github.com/tnagatomi/wtm for details.

# Root directories to scan for git repositories.
# Each root is walked recursively. Tilde (~) expands to your home directory.
# Example:
#   roots:
#     - ~/src
#     - ~/work
roots: []

# Maximum recursion depth from each root.
max_depth: 5
`
