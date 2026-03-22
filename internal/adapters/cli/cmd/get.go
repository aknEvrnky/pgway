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
	cmd.AddCommand(newGetPoolCmd(cp))
	cmd.AddCommand(newGetBalancerCmd(cp))

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

func newGetPoolCmd(cp ports.ControlPlane) *cobra.Command {
	return &cobra.Command{
		Use:     "pool [name]",
		Short:   "Get pool or list all pools",
		Example: "  pgctl get pool\n  pgctl get pool my-pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")

			if len(args) > 0 {
				pool, err := cp.GetPool(ctx, args[0])
				if err != nil {
					return err
				}
				return enc.Encode(pool)
			}

			pools, err := cp.ListPools(ctx)
			if err != nil {
				return err
			}

			if len(pools) == 0 {
				fmt.Println("no pools found")
				return nil
			}

			return enc.Encode(pools)
		},
	}
}

func newGetBalancerCmd(cp ports.ControlPlane) *cobra.Command {
	return &cobra.Command{
		Use:     "balancer [name]",
		Short:   "Get load balancer or list all load balancers",
		Example: "  pgctl get balancer\n  pgctl get balancer my-pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")

			if len(args) > 0 {
				pool, err := cp.GetBalancer(ctx, args[0])
				if err != nil {
					return err
				}
				return enc.Encode(pool)
			}

			balancers, err := cp.ListBalancers(ctx)
			if err != nil {
				return err
			}

			if len(balancers) == 0 {
				fmt.Println("no load balancer found")
				return nil
			}

			return enc.Encode(balancers)
		},
	}
}
