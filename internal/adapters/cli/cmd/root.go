package cmd

import (
	"github.com/aknEvrnky/pgway/internal/adapters/cli"
	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/spf13/cobra"
)

func NewRootCmd(cp ports.ControlPlane) *cobra.Command {
	dispatcher := cli.NewDispatcher(cp)

	root := &cobra.Command{
		Use:   "pgctl",
		Short: "pgway control plane CLI",
	}

	root.AddCommand(
		newApplyCmd(dispatcher),
		newGetCmd(cp),
		newDeleteCmd(cp),
	)

	return root
}
