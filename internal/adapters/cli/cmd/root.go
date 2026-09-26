package cmd

import (
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/aknEvrnky/pgway/internal/platform/version"
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
		Use:     "pgctl",
		Short:   "pgway control plane CLI",
		Version: version.String(),
		// version / help must not dial the control plane.
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "version" || cmd.Name() == "help" {
				return nil
			}
			if v, err := cmd.Flags().GetBool("version"); err == nil && v {
				return nil
			}
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

			client, err := connect(cfg.GRPC.DialTarget(), token)
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
	root.SetVersionTemplate(version.Line("pgctl") + "\n")

	root.PersistentFlags().StringVar(&tokenFlag, "token", "", "bearer token for control plane authentication")
	root.PersistentFlags().StringVar(&configFlag, "config", "", "path to config file (default: search /etc/pgway, $HOME/.pgway, .)")

	root.AddCommand(
		newVersionCmd(),
		newApplyCmd(deps),
		newGetCmd(deps),
		newDeleteCmd(deps),
		newInitCmd(deps),
		newLoginCmd(deps),
		newLogoutCmd(deps),
		newUserCmd(deps),
		newAgentCmd(deps),
	)

	return root
}
