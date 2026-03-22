package main

import (
	"fmt"
	"os"

	"github.com/aknEvrnky/pgway/internal/adapters/cli/cmd"
	grpcclient "github.com/aknEvrnky/pgway/internal/adapters/grpc/client"
	"github.com/aknEvrnky/pgway/internal/platform/config"
)

func main() {
	if err := config.Load(""); err != nil {
		fmt.Fprintln(os.Stderr, "load configuration:", err)
		os.Exit(1)
	}

	cfg := config.Get()

	client, err := grpcclient.NewClient(cfg.GrpcListenAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect to control plane:", err)
		os.Exit(1)
	}
	defer client.Close()

	rootCmd := cmd.NewRootCmd(client)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "unable to run command:", err)
		os.Exit(1)
	}
}
