package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/spf13/cobra"
)

func newGetCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get resources",
	}

	cmd.AddCommand(newGetProxyCmd(d))
	cmd.AddCommand(newGetPoolCmd(d))
	cmd.AddCommand(newGetBalancerCmd(d))
	cmd.AddCommand(newGetRouterCmd(d))
	cmd.AddCommand(newGetFlowCmd(d))
	cmd.AddCommand(newGetEntrypointCmd(d))

	return cmd
}

func newGetProxyCmd(d *Deps) *cobra.Command {
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
				proxy, err := d.Client.GetProxy(ctx, args[0])
				if err != nil {
					return err
				}
				return enc.Encode(proxy)
			}

			// List
			result, err := d.Client.ListProxies(ctx, domain.ListParams{}, domain.ProxyFilter{})
			if err != nil {
				return err
			}

			if len(result.Items) == 0 {
				fmt.Println("no proxies found")
				return nil
			}

			return enc.Encode(result.Items)
		},
	}
}

func newGetPoolCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:     "pool [name]",
		Short:   "Get pool or list all pools",
		Example: "  pgctl get pool\n  pgctl get pool my-pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")

			if len(args) > 0 {
				pool, err := d.Client.GetPool(ctx, args[0])
				if err != nil {
					return err
				}
				return enc.Encode(pool)
			}

			result, err := d.Client.ListPools(ctx, domain.ListParams{}, domain.PoolFilter{})
			if err != nil {
				return err
			}

			if len(result.Items) == 0 {
				fmt.Println("no pools found")
				return nil
			}

			return enc.Encode(result.Items)
		},
	}
}

func newGetBalancerCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:     "balancer [name]",
		Short:   "Get load balancer or list all load balancers",
		Example: "  pgctl get balancer\n  pgctl get balancer my-balancer",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")

			if len(args) > 0 {
				pool, err := d.Client.GetBalancer(ctx, args[0])
				if err != nil {
					return err
				}
				return enc.Encode(pool)
			}

			result, err := d.Client.ListBalancers(ctx, domain.ListParams{}, domain.BalancerFilter{})
			if err != nil {
				return err
			}

			if len(result.Items) == 0 {
				fmt.Println("no load balancer found")
				return nil
			}

			return enc.Encode(result.Items)
		},
	}
}

func newGetRouterCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:     "router [name]",
		Short:   "Get router or list all routers",
		Example: "  pgctl get router\n  pgctl get router my-router",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")

			if len(args) > 0 {
				pool, err := d.Client.GetRouter(ctx, args[0])
				if err != nil {
					return err
				}
				return enc.Encode(pool)
			}

			result, err := d.Client.ListRouters(ctx, domain.ListParams{}, domain.RouterFilter{})
			if err != nil {
				return err
			}

			if len(result.Items) == 0 {
				fmt.Println("no router found")
				return nil
			}

			return enc.Encode(result.Items)
		},
	}
}

func newGetFlowCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:     "flow [name]",
		Short:   "Get flow or list all flows",
		Example: "  pgctl get flow\n  pgctl get flow my-flow",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")

			if len(args) > 0 {
				flow, err := d.Client.GetFlow(ctx, args[0])
				if err != nil {
					return err
				}
				return enc.Encode(flow)
			}

			result, err := d.Client.ListFlows(ctx, domain.ListParams{}, domain.FlowFilter{})
			if err != nil {
				return err
			}

			if len(result.Items) == 0 {
				fmt.Println("no flow found")
				return nil
			}

			return enc.Encode(result.Items)
		},
	}
}

func newGetEntrypointCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:     "entrypoint [name]",
		Short:   "Get entrypoint or list all entrypoints",
		Example: "  pgctl get entrypoint\n  pgctl get entrypoint my-entrypoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")

			if len(args) > 0 {
				pool, err := d.Client.GetEntrypoint(ctx, args[0])
				if err != nil {
					return err
				}
				return enc.Encode(pool)
			}

			result, err := d.Client.ListEntrypoints(ctx, domain.ListParams{}, domain.EntrypointFilter{})
			if err != nil {
				return err
			}

			if len(result.Items) == 0 {
				fmt.Println("no entrypoint found")
				return nil
			}

			return enc.Encode(result.Items)
		},
	}
}
