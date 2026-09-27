package cmd

import (
	"net"
	"strconv"

	"github.com/aknEvrnky/pgway/internal/platform/version"
	"github.com/spf13/cobra"
)

const (
	defaultHost = "127.0.0.1"
	defaultPort = 9090
)

// NewRootCmd builds the pgctl command tree. pgctl is config-file free: the
// control plane address comes from --host/--port flags, the bearer token
// only from the file at --token-path (default $HOME/.pgctl/credentials).
func NewRootCmd(connect ConnectFunc) *cobra.Command {
	deps := &Deps{}

	var host string
	var port int
	var tokenPath string

	root := &cobra.Command{
		Use:     "pgctl",
		Short:   "pgway control plane CLI",
		Version: version.String(),
		// version / help must not dial the control plane.
		SilenceUsage: true,
		// version / help must not dial the control plane.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "version" || cmd.Name() == "help" {
				return nil
			}
			if v, err := cmd.Flags().GetBool("version"); err == nil && v {
				return nil
			}

			if tokenPath == "" {
				p, err := defaultTokenPath()
				if err != nil {
					return err
				}
				tokenPath = p
			}

			token := readTokenFile(tokenPath)
			addr := net.JoinHostPort(host, strconv.Itoa(port))

			client, err := connect(addr, token)
			if err != nil {
				return err
			}

			deps.Client = client
			deps.Token = token
			deps.TokenPath = tokenPath

			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if deps.Client != nil {
				deps.Client.Close()
			}
		},
	}
	root.SetVersionTemplate(version.Line("pgctl") + "\n")

	root.PersistentFlags().StringVarP(&host, "host", "H", defaultHost, "control plane host")
	root.PersistentFlags().IntVarP(&port, "port", "P", defaultPort, "control plane port")
	root.PersistentFlags().StringVar(&tokenPath, "token-path", "", "bearer token file (default: $HOME/.pgctl/credentials)")

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
