package main

import (
	"fmt"
	"os"

	"github.com/aknEvrnky/pgway/internal/adapters/cli/cmd"
	grpcclient "github.com/aknEvrnky/pgway/internal/adapters/grpc/client"
	"github.com/aknEvrnky/pgway/internal/platform/config"
)

func main() {
	// config is loaded inside the root command's PersistentPreRunE so the
	// --config and --token flags are parsed before the client is dialed
	connect := func(addr, token string) (cmd.Client, error) {
		cfg := config.Get()
		return grpcclient.NewClient(addr, token, grpcclient.KeepaliveConfig{
			Interval: cfg.GRPC.KeepaliveInterval,
			Timeout:  cfg.GRPC.KeepaliveTimeout,
		})
	}

	rootCmd := cmd.NewRootCmd(connect)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "unable to run command:", err)
		os.Exit(1)
	}
}
