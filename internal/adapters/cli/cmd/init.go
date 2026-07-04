package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newInitCmd(d *Deps) *cobra.Command {
	var bootstrapToken, username, password string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create the first admin user with the server's bootstrap token",
		Long: `Initializes pgway by creating the first admin user.

The bootstrap token is printed to the server log on first start. Pass it via
--bootstrap-token or the PGWAY_BOOTSTRAP_TOKEN environment variable.`,
		Example: "  pgctl init --bootstrap-token pgw_...",
		RunE: func(cmd *cobra.Command, args []string) error {
			// read via os.Getenv on purpose: this is a one-shot secret, not
			// configuration — it must not live in a config file
			if bootstrapToken == "" {
				bootstrapToken = os.Getenv("PGWAY_BOOTSTRAP_TOKEN")
			}
			if bootstrapToken == "" {
				return fmt.Errorf("bootstrap token is required (--bootstrap-token or PGWAY_BOOTSTRAP_TOKEN)")
			}

			var err error
			if username == "" {
				if username, err = promptLine("admin username"); err != nil {
					return err
				}
			}

			if password == "" {
				if password, err = promptPassword("admin password"); err != nil {
					return err
				}
			}

			user, token, err := d.Client.InitAdmin(cmd.Context(), bootstrapToken, username, password)
			if err != nil {
				return err
			}

			if err := writeCredentials(token); err != nil {
				return fmt.Errorf("admin created but storing token failed: %w", err)
			}

			fmt.Printf("admin user %q created\n", user.Id)
			fmt.Printf("token stored in ~/.pgway/credentials\n")
			fmt.Printf("token: %s\n", token)

			return nil
		},
	}

	cmd.Flags().StringVar(&bootstrapToken, "bootstrap-token", "", "one-time bootstrap token from the server log")
	cmd.Flags().StringVar(&username, "username", "", "admin username (prompted when omitted)")
	cmd.Flags().StringVar(&password, "password", "", "admin password (prompted when omitted)")

	return cmd
}
