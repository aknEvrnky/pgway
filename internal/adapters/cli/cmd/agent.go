package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/spf13/cobra"
)

func newAgentCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage data-plane agents",
	}

	cmd.AddCommand(newAgentTokenCmd(d))
	cmd.AddCommand(newAgentListCmd(d))
	cmd.AddCommand(newAgentDeleteCmd(d))

	return cmd
}

func newAgentTokenCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Manage agent registration tokens",
	}
	cmd.AddCommand(newAgentTokenCreateCmd(d))
	return cmd
}

func newAgentTokenCreateCmd(d *Deps) *cobra.Command {
	var ttl time.Duration

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a single-use agent registration token (admin only)",
		Long:  "Prints the plaintext registration token once. Exchange it via AgentService/Register.",
		Example: `  pgctl agent token create
  pgctl agent token create --ttl 2h`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ttl <= 0 {
				ttl = config.Get().RegistrationTokenTTL
			}

			token, err := d.Client.CreateRegistrationToken(cmd.Context(), ttl)
			if err != nil {
				return err
			}

			fmt.Printf("registration token (shown once): %s\n", token)
			fmt.Printf("expires in: %s\n", ttl)
			return nil
		},
	}

	cmd.Flags().DurationVar(&ttl, "ttl", 0, "token lifetime (default: registration_token_ttl from config)")

	return cmd
}

type agentListRow struct {
	Name          string            `json:"name"`
	Status        string            `json:"status"`
	Hostname      string            `json:"hostname,omitempty"`
	Version       string            `json:"version,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	LastHeartbeat *time.Time        `json:"last_heartbeat,omitempty"`
}

func newAgentListCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := d.Client.ListAgents(cmd.Context(), domain.ListParams{}, domain.AgentFilter{})
			if err != nil {
				return err
			}

			if len(result.Items) == 0 {
				fmt.Println("no agents found")
				return nil
			}

			threshold := config.Get().AgentHeartbeatThreshold
			now := time.Now()
			rows := make([]agentListRow, 0, len(result.Items))
			for _, a := range result.Items {
				rows = append(rows, agentListRow{
					Name:          a.Id,
					Status:        string(a.Status(now, threshold)),
					Hostname:      a.Hostname,
					Version:       a.Version,
					Labels:        a.Labels,
					LastHeartbeat: a.LastHeartbeat,
				})
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(rows)
		},
	}
}

func newAgentDeleteCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an agent and revoke its tokens (admin only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.Client.DeleteAgent(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Printf("agent/%s deleted\n", args[0])
			return nil
		},
	}
}
