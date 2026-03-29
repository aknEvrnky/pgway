package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
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
	cmd.AddCommand(newGetRouterCmd(cp))
	cmd.AddCommand(newGetFlowCmd(cp))
	cmd.AddCommand(newGetEntrypointCmd(cp))

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
			result, err := cp.ListProxies(ctx, domain.ListParams{})
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

			result, err := cp.ListPools(ctx, domain.ListParams{})
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

func newGetBalancerCmd(cp ports.ControlPlane) *cobra.Command {
	return &cobra.Command{
		Use:     "balancer [name]",
		Short:   "Get load balancer or list all load balancers",
		Example: "  pgctl get balancer\n  pgctl get balancer my-balancer",
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

			result, err := cp.ListBalancers(ctx, domain.ListParams{})
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

func newGetRouterCmd(cp ports.ControlPlane) *cobra.Command {
	return &cobra.Command{
		Use:     "router [name]",
		Short:   "Get router or list all routers",
		Example: "  pgctl get router\n  pgctl get router my-router",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")

			if len(args) > 0 {
				pool, err := cp.GetRouter(ctx, args[0])
				if err != nil {
					return err
				}
				return enc.Encode(pool)
			}

			result, err := cp.ListRouters(ctx, domain.ListParams{})
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

func newGetFlowCmd(cp ports.ControlPlane) *cobra.Command {
	return &cobra.Command{
		Use:     "flow [name]",
		Short:   "Get flow or list all flows",
		Example: "  pgctl get flow\n  pgctl get flow my-flow",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")

			if len(args) > 0 {
				pool, err := cp.GetRouter(ctx, args[0])
				if err != nil {
					return err
				}
				return enc.Encode(pool)
			}

			result, err := cp.ListFlows(ctx, domain.ListParams{})
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

func newGetEntrypointCmd(cp ports.ControlPlane) *cobra.Command {
	return &cobra.Command{
		Use:     "entrypoint [name]",
		Short:   "Get entrypoint or list all entrypoints",
		Example: "  pgctl get entrypoint\n  pgctl get entrypoint my-entrypoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")

			if len(args) > 0 {
				pool, err := cp.GetEntrypoint(ctx, args[0])
				if err != nil {
					return err
				}
				return enc.Encode(pool)
			}

			result, err := cp.ListEntrypoints(ctx, domain.ListParams{})
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
