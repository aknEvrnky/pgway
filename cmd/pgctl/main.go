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

	// the client is created after flag parsing so --token can take effect
	connect := func(token string) (cmd.Client, error) {
		return grpcclient.NewClient(cfg.GrpcListenAddr, token)
	}

	rootCmd := cmd.NewRootCmd(connect, cfg.Token)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "unable to run command:", err)
		os.Exit(1)
	}
}
