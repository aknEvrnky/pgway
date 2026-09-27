package main

import (
	"fmt"
	"os"
	"time"

	"github.com/aknEvrnky/pgway/internal/adapters/cli/cmd"
	grpcclient "github.com/aknEvrnky/pgway/internal/adapters/grpc/client"
)

func main() {
	// pgctl is config-file free: address and token are resolved in the root
	// command's PersistentPreRunE from --host/--port/--token-path. Keepalive
	// is hardcoded to the server defaults — the process is short-lived.
	connect := func(addr, token string) (cmd.Client, error) {
		return grpcclient.NewClient(addr, token, grpcclient.KeepaliveConfig{
			Interval: time.Minute,
			Timeout:  20 * time.Second,
		})
	}

	rootCmd := cmd.NewRootCmd(connect)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "unable to run command:", err)
		os.Exit(1)
	}
}
