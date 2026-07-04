package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newLoginCmd(d *Deps) *cobra.Command {
	var username, password string
	var ttl time.Duration
	var noExpiry bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Exchange credentials for a bearer token",
		Example: `  pgctl login --username alice
  pgctl login --username dp-agent --no-expiry   # long-lived token for automation`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if username == "" {
				if username, err = promptLine("username"); err != nil {
					return err
				}
			}

			if password == "" {
				if password, err = promptPassword("password"); err != nil {
					return err
				}
			}

			token, err := d.Client.Login(cmd.Context(), username, password, ttl, noExpiry)
			if err != nil {
				return err
			}

			if err := writeCredentials(token); err != nil {
				return fmt.Errorf("login succeeded but storing token failed: %w", err)
			}

			fmt.Printf("logged in as %q\n", username)
			fmt.Printf("token stored in ~/.pgway/credentials\n")
			fmt.Printf("token: %s\n", token)

			return nil
		},
	}

	cmd.Flags().StringVar(&username, "username", "", "username (prompted when omitted)")
	cmd.Flags().StringVar(&password, "password", "", "password (prompted when omitted)")
	cmd.Flags().DurationVar(&ttl, "ttl", 0, "token lifetime, e.g. 24h (0 = server default)")
	cmd.Flags().BoolVar(&noExpiry, "no-expiry", false, "issue a token that never expires")

	return cmd
}

func newLogoutCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Revoke the current token and remove stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if d.Token == "" {
				return fmt.Errorf("no token to revoke")
			}

			if err := d.Client.Logout(cmd.Context(), d.Token); err != nil {
				return err
			}

			if err := removeCredentials(); err != nil {
				return err
			}

			fmt.Println("logged out")
			return nil
		},
	}
}
