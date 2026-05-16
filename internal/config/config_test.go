package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		name         string
		content      string
		wantRoots    []string
		wantMaxDepth int
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
			wantRoots:    []string{"/abs/path", filepath.Join(home, "relative")},
			wantMaxDepth: 3,
		},
		{
			name: "missing max_depth falls back to default",
			content: `
roots:
  - /only
`,
			wantRoots:    []string{"/only"},
			wantMaxDepth: DefaultMaxDepth,
		},
		{
			name: "tilde alone expands to home",
			content: `
roots:
  - "~"
`,
			wantRoots:    []string{home},
			wantMaxDepth: DefaultMaxDepth,
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
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nonexistent.yml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "wtm init") {
		t.Errorf("error should suggest `wtm init`, got: %v", err)
	}
}

func TestDefaultPath(t *testing.T) {
	t.Run("respects XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")
		got, err := DefaultPath()
		if err != nil {
			t.Fatal(err)
		}
		want := "/custom/xdg/wtm/config.yml"
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
		want := "/home/test/.config/wtm/config.yml"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestStarterContentParses(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(StarterContent), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("starter content should parse: %v", err)
	}
	if cfg.MaxDepth != DefaultMaxDepth {
		t.Errorf("starter MaxDepth: got %d, want %d", cfg.MaxDepth, DefaultMaxDepth)
	}
	if len(cfg.Roots) == 0 {
		t.Error("starter should have at least one root")
	}
}
