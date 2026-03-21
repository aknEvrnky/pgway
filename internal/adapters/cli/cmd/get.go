package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/spf13/cobra"
)

func newGetCmd(cp ports.ControlPlane) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get resources",
	}

	cmd.AddCommand(newGetProxyCmd(cp))

	return cmd
}

func newGetProxyCmd(cp ports.ControlPlane) *cobra.Command {
	return &cobra.Command{
		Use:   "proxy [name]",
		Short: "Get proxy or list all proxies",
		Example: `  pgctl get proxy           # list all
  pgctl get proxy my-proxy  # get single`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")

			// single
			if len(args) > 0 {
				proxy, err := cp.GetProxy(ctx, args[0])
				if err != nil {
					return err
				}
				return enc.Encode(proxy)
			}

			// List
			proxies, err := cp.ListProxies(ctx)
			if err != nil {
				return err
			}

			if len(proxies) == 0 {
				fmt.Println("no proxies found")
				return nil
			}

			return enc.Encode(proxies)
		},
	}
}
