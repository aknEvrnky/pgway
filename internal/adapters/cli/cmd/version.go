package cmd

import (
	"fmt"

	"github.com/aknEvrnky/pgway/internal/platform/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print pgctl version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), version.Line("pgctl"))
		},
	}
}
