package cmd

import (
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/spf13/cobra"
)

// NewRootCmd builds the pgctl command tree. Config is loaded in
// PersistentPreRunE (after flag parsing) so --config takes effect; the token
// resolution order is then: --token flag > config/env (token key or
// PGWAY_TOKEN) > credentials file.
func NewRootCmd(connect ConnectFunc) *cobra.Command {
	deps := &Deps{}

	var tokenFlag string
	var configFlag string

	root := &cobra.Command{
		Use:          "pgctl",
		Short:        "pgway control plane CLI",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Load(configFlag); err != nil {
				return err
			}
			cfg := config.Get()

			token := tokenFlag
			if token == "" {
				token = cfg.Token
			}
			if token == "" {
				token = readCredentials()
			}

			client, err := connect(cfg.GrpcListenAddr, token)
			if err != nil {
				return err
			}

			deps.Client = client
			deps.Token = token

			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if deps.Client != nil {
				deps.Client.Close()
			}
		},
	}

	root.PersistentFlags().StringVar(&tokenFlag, "token", "", "bearer token for control plane authentication")
	root.PersistentFlags().StringVar(&configFlag, "config", "", "path to config file (default: search /etc/pgway, $HOME/.pgway, .)")

	root.AddCommand(
		newApplyCmd(deps),
		newGetCmd(deps),
		newDeleteCmd(deps),
		newInitCmd(deps),
		newLoginCmd(deps),
		newLogoutCmd(deps),
		newUserCmd(deps),
	)

	return root
}
