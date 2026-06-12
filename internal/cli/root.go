package cli

import (
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/tnagatomi/wtclean/internal/config"
	"github.com/tnagatomi/wtclean/internal/scanner"
	"github.com/tnagatomi/wtclean/internal/tui"
)

var Version = "dev"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "wtclean",
		Short:         "Manage git worktrees across multiple projects",
		Long:          "wtclean is a TUI tool that lists and deletes git worktrees across multiple projects.",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runTUI,
	}
	cmd.AddCommand(newInitCmd())
	return cmd
}

func runTUI(cmd *cobra.Command, args []string) error {
	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	// Discovery is deferred to the TUI's Init so the repository list is
	// scanned asynchronously, sharing the `r` refresh code path; config is
	// still loaded synchronously here since a bad config leaves nothing to
	// show. See docs/adr/0001-async-startup-scan.md.
	scanRoots := make([]scanner.Root, len(cfg.Roots))
	for i, r := range cfg.Roots {
		scanRoots[i] = scanner.Root{Path: r.Path, MaxDepth: r.MaxDepth}
	}
	prog := tea.NewProgram(tui.NewScanning(tui.ModelOptions{
		ConfigPath: path,
		ScanRoots:  scanRoots,
		Skip:       cfg.Skip,
	}))
	_, err = prog.Run()
	return err
}
