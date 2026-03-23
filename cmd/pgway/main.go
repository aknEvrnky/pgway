package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/adapters/grpc/server"
	"github.com/aknEvrnky/pgway/internal/adapters/http"
	proxyadapter "github.com/aknEvrnky/pgway/internal/adapters/proxy/net"
	badgerrepo "github.com/aknEvrnky/pgway/internal/adapters/repository/badger"
	"github.com/aknEvrnky/pgway/internal/adapters/rest"
	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/aknEvrnky/pgway/internal/application/core/api"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	_ "github.com/aknEvrnky/pgway/internal/platform/logger"
	badgerdb "github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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

	// Control Plane — single service, used in both gRPC and data-plane
	cpService := controlplane.NewService(
		badgerrepo.NewProxyRepository(db),
		badgerrepo.NewPoolRepository(db),
		badgerrepo.NewBalancerRepository(db),
		badgerrepo.NewRouterRepository(db),
		badgerrepo.NewFlowRepository(db),
		badgerrepo.NewEntrypointRepository(db),
	)

	// gRPC server — cli's command bus
	grpcServer := grpc.NewServer()
	cpGrpcServer := server.NewControlPlaneServer(cpService)
	controlplanev1.RegisterProxyServiceServer(grpcServer, cpGrpcServer)
	controlplanev1.RegisterPoolServiceServer(grpcServer, cpGrpcServer)
	controlplanev1.RegisterBalancerServiceServer(grpcServer, cpGrpcServer)
	controlplanev1.RegisterRouterServiceServer(grpcServer, cpGrpcServer)
	controlplanev1.RegisterFlowServiceServer(grpcServer, cpGrpcServer)
	controlplanev1.RegisterEntrypointServiceServer(grpcServer, cpGrpcServer)

	lis, err := net.Listen("tcp", cfg.GrpcListenAddr)
	if err != nil {
		zap.L().Fatal("grpc listen", zap.Error(err), zap.String("addr", cfg.GrpcListenAddr))
	}

	// Data Plane — cpService as read only service
	app := api.NewApplication(cpService)
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

	// Start
	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	runErr := make(chan error, 3)

	go func() {
		zap.L().Info("grpc started", zap.String("addr", cfg.GrpcListenAddr))
		runErr <- grpcServer.Serve(lis)
	}()

	go func() {
		zap.L().Info("gateway started")
		runErr <- httpAdapter.Run(sigCtx)
	}()

	go func() {
		runErr <- restAdapter.Run(sigCtx)
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
