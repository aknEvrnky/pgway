package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/spf13/cobra"
)

func newUserCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}

	cmd.AddCommand(newUserCreateCmd(d))
	cmd.AddCommand(newUserListCmd(d))
	cmd.AddCommand(newUserDeleteCmd(d))
	cmd.AddCommand(newUserChangePasswordCmd(d))

	return cmd
}

func newUserCreateCmd(d *Deps) *cobra.Command {
	var password, role string

	cmd := &cobra.Command{
		Use:   "create <username>",
		Short: "Create a user (admin only)",
		Long:  "Creates a user. Without --password a temporary password is generated and printed once.",
		Args:  cobra.ExactArgs(1),
		Example: `  pgctl user create bob
  pgctl user create alice --role admin`,
		RunE: func(cmd *cobra.Command, args []string) error {
			user, generated, err := d.Client.CreateUser(cmd.Context(), args[0], password, domain.Role(role))
			if err != nil {
				return err
			}

			fmt.Printf("user/%s created (role: %s)\n", user.Id, user.Role)
			if generated != "" {
				fmt.Printf("temporary password (shown once): %s\n", generated)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&password, "password", "", "initial password (generated when omitted)")
	cmd.Flags().StringVar(&role, "role", string(domain.RoleMember), "user role: admin or member")

	return cmd
}

func newUserListCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List users (admin only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := d.Client.ListUsers(cmd.Context(), domain.ListParams{}, domain.UserFilter{})
			if err != nil {
				return err
			}

			if len(result.Items) == 0 {
				fmt.Println("no users found")
				return nil
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result.Items)
		},
	}
}

func newUserDeleteCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <username>",
		Short: "Delete a user (admin only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.Client.DeleteUser(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Printf("user/%s deleted\n", args[0])
			return nil
		},
	}
}

func newUserChangePasswordCmd(d *Deps) *cobra.Command {
	var oldPassword, newPassword string

	cmd := &cobra.Command{
		Use:   "change-password [username]",
		Short: "Change your own password, or another user's as admin",
		Args:  cobra.MaximumNArgs(1),
		Example: `  pgctl user change-password          # own password
  pgctl user change-password bob      # admin reset`,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) > 0 {
				target = args[0]
			}

			var err error
			if newPassword == "" {
				if newPassword, err = promptPassword("new password"); err != nil {
					return err
				}
			}

			// another user's password is an admin reset; your own requires
			// the current one
			if target != "" {
				if err := d.Client.ResetPassword(cmd.Context(), target, newPassword); err != nil {
					return err
				}
			} else {
				if oldPassword == "" {
					if oldPassword, err = promptPassword("current password"); err != nil {
						return err
					}
				}

				if err := d.Client.ChangePassword(cmd.Context(), "", oldPassword, newPassword); err != nil {
					return err
				}
			}

			fmt.Println("password changed")
			return nil
		},
	}

	cmd.Flags().StringVar(&oldPassword, "old-password", "", "current password (prompted for own account when omitted)")
	cmd.Flags().StringVar(&newPassword, "new-password", "", "new password (prompted when omitted)")

	return cmd
}
