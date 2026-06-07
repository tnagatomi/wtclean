package cli

import (
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/tnagatomi/wtclean/internal/config"
	"github.com/tnagatomi/wtclean/internal/repo"
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
	repos, totalScanned, err := repo.Discover(cfg.Roots, cfg.MaxDepth)
	if err != nil {
		return err
	}
	prog := tea.NewProgram(tui.NewModel(repos, tui.ModelOptions{
		ConfigPath:   path,
		ConfigRoots:  cfg.Roots,
		TotalScanned: totalScanned,
	}))
	_, err = prog.Run()
	return err
}
