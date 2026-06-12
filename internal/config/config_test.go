package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestLoad(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		name         string
		content      string
		wantRoots    []Root
		wantMaxDepth int
		wantSkip     []string
		wantErr      string
	}{
		{
			name: "valid with explicit max_depth",
			content: `
roots:
  - /abs/path
  - ~/relative
max_depth: 3
`,
			wantRoots: []Root{
				{Path: "/abs/path", MaxDepth: 3},
				{Path: filepath.Join(home, "relative"), MaxDepth: 3},
			},
			wantMaxDepth: 3,
		},
		{
			name: "missing max_depth falls back to default",
			content: `
roots:
  - /only
`,
			wantRoots:    []Root{{Path: "/only", MaxDepth: DefaultMaxDepth}},
			wantMaxDepth: DefaultMaxDepth,
		},
		{
			name: "tilde alone expands to home",
			content: `
roots:
  - "~"
`,
			wantRoots:    []Root{{Path: home, MaxDepth: DefaultMaxDepth}},
			wantMaxDepth: DefaultMaxDepth,
		},
		{
			name: "per-root max_depth overrides global, others inherit",
			content: `
roots:
  - ~/src
  - path: ~/Downloads
    max_depth: 2
max_depth: 5
`,
			wantRoots: []Root{
				{Path: filepath.Join(home, "src"), MaxDepth: 5},
				{Path: filepath.Join(home, "Downloads"), MaxDepth: 2},
			},
			wantMaxDepth: 5,
		},
		{
			name: "root mapping without path errors",
			content: `
roots:
  - max_depth: 2
`,
			wantErr: "non-empty path",
		},
		{
			name: "skip globs are parsed",
			content: `
roots:
  - /only
skip:
  - node_modules
  - "*.egg-info"
`,
			wantRoots:    []Root{{Path: "/only", MaxDepth: DefaultMaxDepth}},
			wantMaxDepth: DefaultMaxDepth,
			wantSkip:     []string{"node_modules", "*.egg-info"},
		},
		{
			name: "invalid skip pattern errors",
			content: `
roots:
  - /only
skip:
  - "[bad"
`,
			wantErr: "invalid skip pattern",
		},
		{
			name:    "empty roots errors",
			content: `max_depth: 4`,
			wantErr: "no roots",
		},
		{
			name:    "invalid yaml errors",
			content: "roots: [unterminated\n",
			wantErr: "parse",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yml")
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := cfg.MaxDepth; got != tc.wantMaxDepth {
				t.Errorf("MaxDepth: got %d, want %d", got, tc.wantMaxDepth)
			}
			if !slices.Equal(cfg.Roots, tc.wantRoots) {
				t.Errorf("Roots: got %v, want %v", cfg.Roots, tc.wantRoots)
			}
			if !slices.Equal(cfg.Skip, tc.wantSkip) {
				t.Errorf("Skip: got %v, want %v", cfg.Skip, tc.wantSkip)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nonexistent.yml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "wtclean init") {
		t.Errorf("error should suggest `wtclean init`, got: %v", err)
	}
}

func TestDefaultPath(t *testing.T) {
	t.Run("respects XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")
		got, err := DefaultPath()
		if err != nil {
			t.Fatal(err)
		}
		want := "/custom/xdg/wtclean/config.yml"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("falls back to ~/.config", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "/home/test")
		got, err := DefaultPath()
		if err != nil {
			t.Fatal(err)
		}
		want := "/home/test/.config/wtclean/config.yml"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestStarterContent(t *testing.T) {
	var cfg Config
	if err := yaml.Unmarshal([]byte(StarterContent), &cfg); err != nil {
		t.Fatalf("starter content should be valid YAML: %v", err)
	}
	if cfg.MaxDepth != DefaultMaxDepth {
		t.Errorf("starter MaxDepth: got %d, want %d", cfg.MaxDepth, DefaultMaxDepth)
	}
	if len(cfg.Roots) != 0 {
		t.Errorf("starter should have empty roots, got %v", cfg.Roots)
	}
	// The starter ships representative per-language skip globs so a fresh
	// install scans quickly out of the box; node_modules is the canonical one.
	if !slices.Contains(cfg.Skip, "node_modules") {
		t.Errorf("starter Skip should include node_modules, got %v", cfg.Skip)
	}

	// Loading the starter should fail with the empty-roots error so users
	// know they must edit the file before wtclean is usable.
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(StarterContent), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "no roots") {
		t.Errorf("starter Load should report no roots, got: %v", err)
	}
}
