package cmd

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/aknEvrnky/pgway/internal/platform/version"
	"github.com/spf13/cobra"
)

const (
	defaultHost = "127.0.0.1"
	defaultPort = 9090

	// tokenlessAnnotation marks commands whose RPCs are auth-exempt (login,
	// init): they dial without a token file, and login rewrites it.
	tokenlessAnnotation = "pgctl/tokenless"
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
		Use:          "pgctl",
		Short:        "pgway control plane CLI",
		Version:      version.String(),
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if skipsDial(cmd) {
				return nil
			}
			if v, err := cmd.Flags().GetBool("version"); err == nil && v {
				return nil
			}

			if port < 1 || port > 65535 {
				return fmt.Errorf("--port must be between 1 and 65535, got %d", port)
			}
			// Bare IPv6 literals pass JoinHostPort untouched (it brackets
			// them); anything else carrying a colon would smuggle a port or
			// brackets into the dial target and fail at the first RPC.
			if strings.Contains(host, ":") && net.ParseIP(host) == nil {
				return fmt.Errorf("--host must be a bare host or IP (got %q; set the port with --port)", host)
			}

			if tokenPath == "" {
				p, err := defaultTokenPath()
				if err != nil {
					return err
				}
				tokenPath = p
			}

			var token string
			if cmd.Annotations[tokenlessAnnotation] == "true" {
				token = readTokenFile(tokenPath)
			} else {
				var err error
				token, err = requireTokenFile(tokenPath)
				if err != nil {
					return err
				}
			}

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

// skipsDial reports whether cmd is client-side only: version/help and the
// cobra-generated completion machinery never touch the control plane.
func skipsDial(cmd *cobra.Command) bool {
	switch cmd.Name() {
	case "version", "help", "completion":
		return true
	}
	if strings.HasPrefix(cmd.Name(), "__") { // cobra's hidden __complete*
		return true
	}
	return cmd.Parent() != nil && cmd.Parent().Name() == "completion"
}
