// Package wtcleanlog writes per-failure one-line records to a state-home
// log file. The TUI surfaces a short summary on-screen while the full
// detail (including timestamps and error text) goes here for later
// debugging.
package wtcleanlog

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Path returns the log-file path, creating its parent directory if it
// does not already exist. Resolution follows the XDG Base Directory
// spec: $XDG_STATE_HOME/wtclean/wtclean.log, falling back to
// ~/.local/state/wtclean/wtclean.log when XDG_STATE_HOME is unset or empty.
func Path() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		base = filepath.Join(home, ".local", "state")
	}
	path := filepath.Join(base, "wtclean", "wtclean.log")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create log directory: %w", err)
	}
	return path, nil
}

// Append writes a single record to the log, prefixing the current time
// in RFC 3339 format. Returns an error from Path resolution or file
// I/O. Each record is terminated with a newline so consumers can split
// on \n without ambiguity.
func Append(line string) error {
	path, err := Path()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	defer func() { _ = f.Close() }()
	_, err = fmt.Fprintf(f, "%s %s\n", time.Now().UTC().Format(time.RFC3339), line)
	if err != nil {
		return fmt.Errorf("write log: %w", err)
	}
	return nil
}
