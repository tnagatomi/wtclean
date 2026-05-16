package cli

import (
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/tnagatomi/wtm/internal/config"
	"github.com/tnagatomi/wtm/internal/repo"
	"github.com/tnagatomi/wtm/internal/tui"
)

var Version = "dev"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "wtm",
		Short:         "Manage git worktrees across multiple projects",
		Long:          "wtm is a TUI tool that lists and deletes git worktrees across multiple projects.",
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
	repos, err := repo.Discover(cfg.Roots, cfg.MaxDepth)
	if err != nil {
		return err
	}
	prog := tea.NewProgram(tui.NewModel(repos))
	_, err = prog.Run()
	return err
}
