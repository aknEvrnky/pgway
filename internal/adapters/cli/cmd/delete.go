package cmd

import (
	"fmt"

	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/spf13/cobra"
)

func newDeleteCmd(cp ports.ControlPlane) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete resources",
	}

	cmd.AddCommand(newDeleteProxyCmd(cp))

	return cmd
}

func newDeleteProxyCmd(cp ports.ControlPlane) *cobra.Command {
	return &cobra.Command{
		Use:   "proxy <name>",
		Short: "Delete a proxy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cp.DeleteProxy(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Printf("proxy/%s deleted\n", args[0])
			return nil
		},
	}
}
