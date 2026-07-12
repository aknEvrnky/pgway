package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aknEvrnky/pgway/internal/adapters/grpc/server"
	"github.com/aknEvrnky/pgway/internal/adapters/http"
	proxyadapter "github.com/aknEvrnky/pgway/internal/adapters/proxy/net"
	"github.com/aknEvrnky/pgway/internal/adapters/pubsub/memory"
	badgerrepo "github.com/aknEvrnky/pgway/internal/adapters/repository/badger"
	"github.com/aknEvrnky/pgway/internal/adapters/rest"
	"github.com/aknEvrnky/pgway/internal/application/auth"
	"github.com/aknEvrnky/pgway/internal/application/consumer"
	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/aknEvrnky/pgway/internal/application/core/api"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	_ "github.com/aknEvrnky/pgway/internal/platform/logger"
	badgerdb "github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

func main() {
	if err := config.Load(""); err != nil {
		zap.L().Fatal("load configuration", zap.Error(err))
	}

	cfg := config.Get()

	// BadgerDB
	opts := badgerdb.DefaultOptions(cfg.BadgerPath).WithLogger(badgerrepo.NewBadgerLogger())
	db, err := badgerdb.Open(opts)
	if err != nil {
		zap.L().Fatal("open badger", zap.Error(err), zap.String("path", cfg.BadgerPath))
	}
	defer db.Close()

	pubSub := memory.NewPubSub(10)

	// Control Plane — single service, used in both gRPC and data-plane
	cpService := controlplane.NewService(
		badgerrepo.NewProxyRepository(db),
		badgerrepo.NewPoolRepository(db),
		badgerrepo.NewBalancerRepository(db),
		badgerrepo.NewRouterRepository(db),
		badgerrepo.NewFlowRepository(db),
		badgerrepo.NewEntrypointRepository(db),
		pubSub,
	)

	// Auth service — users, tokens, bootstrap flow
	authService := auth.NewService(
		badgerrepo.NewUserRepository(db),
		badgerrepo.NewTokenRepository(db),
		cfg.TokenTTL,
	)

	if err := authService.Bootstrap(context.Background()); err != nil {
		zap.L().Fatal("auth bootstrap", zap.Error(err))
	}

	// gRPC server — cli's command bus, auth enforced
	grpcServer := server.New(cpService, cpService, authService, authService, authService)

	lis, err := net.Listen("tcp", cfg.GrpcListenAddr)
	if err != nil {
		zap.L().Fatal("grpc listen", zap.Error(err), zap.String("addr", cfg.GrpcListenAddr))
	}

	// Data Plane — cpService as read only service
	app := api.NewApplication(cpService, cpService)
	ctx := context.Background()

	if err := app.Bootstrap(ctx); err != nil {
		zap.L().Fatal("bootstrap", zap.Error(err))
	}

	proxyTransport := proxyadapter.NewAdapter()
	httpAdapter, err := http.NewHttpAdapter(ctx, app, proxyTransport)
	if err != nil {
		zap.L().Fatal("init http adapter", zap.Error(err))
	}

	// REST adapter
	restAdapter := rest.NewRestAdapter(cpService, cfg.RestListenAddr)

	// event consumer — handler order matters: app refreshes the cache first,
	// then the http adapter reads the refreshed cache
	eventConsumer := consumer.NewConsumer(pubSub, app, httpAdapter)

	// Start
	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	runErr := make(chan error, 4)

	go func() {
		zap.L().Info("grpc started", zap.String("addr", cfg.GrpcListenAddr))
		runErr <- grpcServer.Serve(lis)
	}()

	go func() {
		zap.L().Info("gateway started")
		runErr <- httpAdapter.Run(sigCtx)
	}()

	go func() {
		zap.L().Info("restapi started")
		runErr <- restAdapter.Run(sigCtx)
	}()

	go func() {
		zap.L().Info("event consumer started")
		if err := eventConsumer.ConsumeEvents(sigCtx); err != nil {
			runErr <- err
		}
	}()

	select {
	case <-sigCtx.Done():
	case err := <-runErr:
		zap.L().Error("server failed", zap.Error(err))
	}

	// Graceful shutdown
	zap.L().Info("shutting down")
	grpcServer.GracefulStop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpAdapter.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("http shutdown", zap.Error(err))
	}

	if err := restAdapter.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("rest shutdown", zap.Error(err))
	}
}
