package cli

import (
	"github.com/spf13/cobra"
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
	}
	cmd.AddCommand(newInitCmd())
	return cmd
}
