package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aknEvrnky/pgway/internal/adapters/http"
	"github.com/aknEvrnky/pgway/internal/adapters/proxy/net"
	badgerrepo "github.com/aknEvrnky/pgway/internal/adapters/repository/badger"
	epRepo "github.com/aknEvrnky/pgway/internal/adapters/repository/config/entrypoint"
	flowRepo "github.com/aknEvrnky/pgway/internal/adapters/repository/config/flow"
	"github.com/aknEvrnky/pgway/internal/application/core/api"
	"github.com/aknEvrnky/pgway/internal/platform/badger"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	_ "github.com/aknEvrnky/pgway/internal/platform/logger"
	badgerdb "github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("c", "", "path to config file")
	flag.Parse()

	if err := config.Load(*configPath); err != nil {
		zap.L().Fatal("load configuration file", zap.Error(err))
	}

	cfg := config.Get()

	opts := badgerdb.DefaultOptions(cfg.BadgerPath).WithLogger(badger.NewBadgerLogger())
	db, err := badgerdb.Open(opts)
	if err != nil {
		zap.L().Fatal("unable to init badger", zap.Error(err), zap.String("path", cfg.BadgerPath))
	}

	defer db.Close()

	entryPointRepository, err := epRepo.NewConfigRepository(cfg)

	if err != nil {
		zap.L().Fatal("init entrypoints", zap.Error(err))
	}

	flowRepository, err := flowRepo.NewConfigRepository(cfg)

	if err != nil {
		zap.L().Fatal("init flows", zap.Error(err))
	}

	routerRepository := badgerrepo.NewRouterRepository(db)
	lbRepository := badgerrepo.NewBalancerRepository(db)
	poolRepository := badgerrepo.NewPoolRepository(db)
	proxyRepository := badgerrepo.NewProxyRepository(db)

	app := api.NewApplication(
		entryPointRepository,
		flowRepository,
		routerRepository,
		lbRepository,
		poolRepository,
		proxyRepository,
	)
	ctx := context.Background()

	err = app.Bootstrap(ctx)
	if err != nil {
		zap.L().Fatal("bootstrapping application", zap.Error(err))
	}

	proxyTransport := net.NewAdapter()

	httpAdapter, err := http.NewHttpAdapter(ctx, app, proxyTransport)

	if err != nil {
		zap.L().Fatal("init http adapter", zap.Error(err))
	}

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	runErr := make(chan error, 1)

	go func() {
		zap.L().Info("Starting http adapter")
		runErr <- httpAdapter.Run(sigCtx)
	}()

	select {
	// Wait for term signal
	case <-sigCtx.Done():
	case err := <-runErr:
		zap.L().Error("server failed", zap.Error(err))
	}

	// graceful shutdown, wait for 30 sec
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err = httpAdapter.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("graceful shutdown", zap.Error(err))
	}
}
