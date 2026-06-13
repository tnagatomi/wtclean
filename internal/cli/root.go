package cli

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/tnagatomi/wtclean/internal/config"
	"github.com/tnagatomi/wtclean/internal/repo"
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
	// --cwd is a local flag: it selects the single-repository mode for the
	// root TUI command and is meaningless for subcommands such as `init`,
	// so it must not be inherited.
	cmd.Flags().Bool("cwd", false, "Operate only on the git repository containing the current directory, skipping config and the configured roots")
	cmd.AddCommand(newInitCmd())
	return cmd
}

func runTUI(cmd *cobra.Command, args []string) error {
	// --cwd mode targets the single repository containing the working
	// directory. It deliberately skips config entirely (no roots to scan),
	// so `wtclean --cwd` works without an `init`'d config file.
	if useCwd, _ := cmd.Flags().GetBool("cwd"); useCwd {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		m, err := newCwdModel(wd)
		if err != nil {
			return err
		}
		_, err = tea.NewProgram(m).Run()
		return err
	}

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

// newCwdModel builds the --cwd model for the repository containing dir. It
// resolves the primary worktree, loads that one repository, and seeds the
// single-repo TUI. An error is returned when dir is not inside a git
// repository, so the caller can fail before launching the TUI.
func newCwdModel(dir string) (tui.Model, error) {
	primary, err := repo.ResolvePrimaryDir(dir)
	if err != nil {
		return tui.Model{}, err
	}
	r, err := repo.Load(primary)
	if err != nil {
		return tui.Model{}, err
	}
	return tui.NewSingleRepo(r, tui.ModelOptions{}), nil
}
