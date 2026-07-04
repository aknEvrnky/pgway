package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDeleteCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete resources",
	}

	cmd.AddCommand(newDeleteProxyCmd(d))
	cmd.AddCommand(newDeletePoolCmd(d))
	cmd.AddCommand(newDeleteBalancerCmd(d))
	cmd.AddCommand(newDeleteRouterCmd(d))
	cmd.AddCommand(newDeleteFlowCmd(d))
	cmd.AddCommand(newDeleteEntrypointCmd(d))

	return cmd
}

func newDeleteProxyCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "proxy <name>",
		Short: "Delete a proxy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.Client.DeleteProxy(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Printf("proxy/%s deleted\n", args[0])
			return nil
		},
	}
}

func newDeletePoolCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "pool <name>",
		Short: "Delete a pool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.Client.DeletePool(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Printf("pool/%s deleted\n", args[0])
			return nil
		},
	}
}

func newDeleteBalancerCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "balancer <name>",
		Short: "Delete a load balancer",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.Client.DeleteBalancer(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Printf("balancer/%s deleted\n", args[0])
			return nil
		},
	}
}

func newDeleteRouterCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "router <name>",
		Short: "Delete a router",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.Client.DeleteRouter(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Printf("router/%s deleted\n", args[0])
			return nil
		},
	}
}

func newDeleteFlowCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "flow <name>",
		Short: "Delete a flow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.Client.DeleteFlow(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Printf("flow/%s deleted\n", args[0])
			return nil
		},
	}
}

func newDeleteEntrypointCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "entrypoint <name>",
		Short: "Delete an entrypoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.Client.DeleteEntrypoint(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Printf("entrypoint/%s deleted\n", args[0])
			return nil
		},
	}
}
