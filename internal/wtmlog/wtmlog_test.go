package wtmlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathUsesXDGStateHomeWhenSet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	got, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	want := filepath.Join(dir, "wtm", "wtm.log")
	if got != want {
		t.Errorf("path: got %q, want %q", got, want)
	}
	// The wtm/ parent should exist after the call.
	if _, err := os.Stat(filepath.Join(dir, "wtm")); err != nil {
		t.Errorf("Path should create the parent dir: %v", err)
	}
}

func TestPathFallsBackToLocalState(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", dir)
	got, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	want := filepath.Join(dir, ".local", "state", "wtm", "wtm.log")
	if got != want {
		t.Errorf("path: got %q, want %q", got, want)
	}
}

func TestAppendWritesTimestampedLine(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	if err := Append("delete /repo/wt/a: boom"); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := Append("fetch /repo: timeout"); err != nil {
		t.Fatalf("Append (second): %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "wtm", "wtm.log"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 log lines (append should not overwrite), got %d: %q", len(lines), data)
	}
	if !strings.Contains(lines[0], "delete /repo/wt/a: boom") {
		t.Errorf("first line missing payload: %q", lines[0])
	}
	if !strings.HasPrefix(lines[0], "20") {
		t.Errorf("first line should start with an RFC 3339 year: %q", lines[0])
	}
}
