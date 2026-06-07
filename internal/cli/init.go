package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/tnagatomi/wtclean/internal/config"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Write a starter configuration file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.DefaultPath()
			if err != nil {
				return fmt.Errorf("resolve config path: %w", err)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fmt.Errorf("create config directory: %w", err)
			}
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
			if errors.Is(err, os.ErrExist) {
				return fmt.Errorf("config file already exists at %s", path)
			}
			if err != nil {
				return fmt.Errorf("create config file: %w", err)
			}
			defer func() { _ = f.Close() }()
			if _, err := f.WriteString(config.StarterContent); err != nil {
				return fmt.Errorf("write config: %w", err)
			}
			cmd.Printf("Wrote starter config to %s\n", path)
			return nil
		},
	}
}
