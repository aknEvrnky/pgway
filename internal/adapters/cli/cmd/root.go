package cmd

import (
	"github.com/spf13/cobra"
)

// NewRootCmd builds the pgctl command tree. configToken is the token from
// config/env; the --token flag and the credentials file complete the
// resolution order: flag > config/env > credentials file.
func NewRootCmd(connect ConnectFunc, configToken string) *cobra.Command {
	deps := &Deps{}

	var tokenFlag string

	root := &cobra.Command{
		Use:          "pgctl",
		Short:        "pgway control plane CLI",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			token := tokenFlag
			if token == "" {
				token = configToken
			}
			if token == "" {
				token = readCredentials()
			}

			client, err := connect(token)
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
