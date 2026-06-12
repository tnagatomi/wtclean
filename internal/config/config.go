// Package config loads and validates the wtclean configuration file.
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
	Roots    []Root   `yaml:"roots"`
	MaxDepth int      `yaml:"max_depth"`
	Skip     []string `yaml:"skip"`
}

// Root is a configured scan root. After Load it carries an absolute-ready
// (tilde-expanded) Path and a resolved MaxDepth: either the per-root override
// or, when none was given, the global max_depth.
type Root struct {
	Path     string
	MaxDepth int
}

// UnmarshalYAML accepts a root written either as a bare string (just the path,
// inheriting the global max_depth) or as a mapping with an explicit per-root
// depth. This keeps existing `roots: [~/src]` configs working while letting a
// noisy root be scanned shallowly:
//
//	roots:
//	  - ~/src
//	  - path: ~/Downloads
//	    max_depth: 2
func (r *Root) UnmarshalYAML(unmarshal func(any) error) error {
	var path string
	if err := unmarshal(&path); err == nil {
		r.Path = path
		return nil
	}
	var aux struct {
		Path     string `yaml:"path"`
		MaxDepth int    `yaml:"max_depth"`
	}
	if err := unmarshal(&aux); err != nil {
		return err
	}
	if aux.Path == "" {
		return errors.New("root entry must have a non-empty path")
	}
	r.Path = aux.Path
	r.MaxDepth = aux.MaxDepth
	return nil
}

// DefaultPath returns the canonical config file path, honoring XDG_CONFIG_HOME.
// macOS's os.UserConfigDir points at ~/Library/Application Support, which the
// spec explicitly avoids, so we resolve XDG ourselves.
func DefaultPath() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "wtclean", "config.yml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "wtclean", "config.yml"), nil
}

// Load reads and validates the config file at path. Tilde-prefixed roots are
// expanded to absolute paths. Missing max_depth falls back to DefaultMaxDepth.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file not found at %q: run `wtclean init` to generate a starter file", path)
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
	for _, pat := range cfg.Skip {
		if _, err := filepath.Match(pat, ""); err != nil {
			return nil, fmt.Errorf("config %q has an invalid skip pattern %q: %w", path, pat, err)
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	for i := range cfg.Roots {
		cfg.Roots[i].Path = expandTilde(cfg.Roots[i].Path, home)
		// A root without its own max_depth (the common case) inherits the
		// global one, which has already fallen back to DefaultMaxDepth above.
		if cfg.Roots[i].MaxDepth <= 0 {
			cfg.Roots[i].MaxDepth = cfg.MaxDepth
		}
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

// StarterContent is the body of the config file written by `wtclean init`.
const StarterContent = `# wtclean configuration file.
# See https://github.com/tnagatomi/wtclean for details.

# Root directories to scan for git repositories.
# Each root is walked recursively. Tilde (~) expands to your home directory.
# A root may be a plain path (using the global max_depth below) or a mapping
# with its own max_depth, handy for a broad, shallow directory.
# Example:
#   roots:
#     - ~/src
#     - ~/work
#     - path: ~/Downloads
#       max_depth: 2
roots:

# Default maximum recursion depth, applied to every root without its own.
max_depth: 5

# Directory names to prune during the scan. Any directory whose base name
# matches one of these globs (filepath.Match syntax) is skipped together with
# everything beneath it, keeping the walk out of large dependency and build
# trees. This speeds up scanning, at the cost of not discovering repositories
# nested inside a skipped directory. Trim the list to the languages you use.
skip:
  # Node.js / JavaScript / TypeScript
  - node_modules
  # Rust / Maven
  - target
  # Python
  - .venv
  - venv
  - __pycache__
  - .mypy_cache
  - .pytest_cache
  - .tox
  - "*.egg-info"
  # Go / PHP / Ruby
  - vendor
  # Java / Kotlin / Gradle
  - .gradle
  # .NET
  - bin
  - obj
  # Swift / Xcode
  - .build
  - DerivedData
  # Common build output and caches
  - build
  - dist
  - out
  - .cache
`
