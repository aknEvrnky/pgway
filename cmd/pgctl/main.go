package main

import (
	"github.com/aknEvrnky/pgway/internal/adapters/cli/cmd"
	badgerrepo "github.com/aknEvrnky/pgway/internal/adapters/repository/badger"
	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/aknEvrnky/pgway/internal/platform/badger"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	badgerdb "github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

func main() {
	if err := config.Load(""); err != nil {
		zap.L().Fatal("load configuration", zap.Error(err))
	}

	cfg := config.Get()

	opts := badgerdb.DefaultOptions(cfg.BadgerPath).WithLogger(badger.NewBadgerLogger())
	db, err := badgerdb.Open(opts)
	if err != nil {
		zap.L().Fatal("open badger", zap.Error(err), zap.String("path", cfg.BadgerPath))
	}
	defer db.Close()

	proxyRepo := badgerrepo.NewProxyRepository(db)
	poolRepo := badgerrepo.NewPoolRepository(db)
	lbRepo := badgerrepo.NewBalancerRepository(db)

	cpService := controlplane.NewService(proxyRepo, poolRepo, lbRepo)

	rootCmd := cmd.NewRootCmd(cpService)
	if err := rootCmd.Execute(); err != nil {
		zap.L().Fatal("unable to run command", zap.Error(err))
	}
}
