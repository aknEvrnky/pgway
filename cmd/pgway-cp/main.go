package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	grpcserver "github.com/aknEvrnky/pgway/internal/adapters/grpc/server"

	badgerrepo "github.com/aknEvrnky/pgway/internal/adapters/repository/badger"
	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/aknEvrnky/pgway/internal/platform/badger"
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

	opts := badgerdb.DefaultOptions(cfg.BadgerPath).WithLogger(badger.NewBadgerLogger())
	db, err := badgerdb.Open(opts)
	if err != nil {
		zap.L().Fatal("open badger", zap.Error(err), zap.String("path", cfg.BadgerPath))
	}
	defer db.Close()

	proxyRepo := badgerrepo.NewProxyRepository(db)
	poolRepo := badgerrepo.NewPoolRepository(db)
	lbRepo := badgerrepo.NewBalancerRepository(db)
	routerRepo := badgerrepo.NewRouterRepository(db)
	flowRepo := badgerrepo.NewFlowRepository(db)
	epRepo := badgerrepo.NewEntrypointRepository(db)

	// control plane service
	cpService := controlplane.NewService(proxyRepo, poolRepo, lbRepo, routerRepo, flowRepo, epRepo)

	// grpc server
	grpcServer := grpc.NewServer()
	cpGrpcServer := grpcserver.NewControlPlaneServer(cpService)
	controlplanev1.RegisterProxyServiceServer(grpcServer, cpGrpcServer)
	controlplanev1.RegisterPoolServiceServer(grpcServer, cpGrpcServer)
	controlplanev1.RegisterRouterServiceServer(grpcServer, cpGrpcServer)
	controlplanev1.RegisterBalancerServiceServer(grpcServer, cpGrpcServer)
	controlplanev1.RegisterEntrypointServiceServer(grpcServer, cpGrpcServer)
	controlplanev1.RegisterFlowServiceServer(grpcServer, cpGrpcServer)

	// tcp port
	lis, err := net.Listen("tcp", cfg.GrpcListenAddr)

	if err != nil {
		zap.L().Fatal("listen", zap.Error(err), zap.String("grpc_listen_addr", cfg.GrpcListenAddr))
	}

	// graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		zap.L().Info("control plane started", zap.String("grpc", cfg.GrpcListenAddr))
		err := grpcServer.Serve(lis)
		if err != nil {
			zap.L().Fatal("grpc serve", zap.Error(err))
		}
	}()

	<-sigCh
	zap.L().Info("shutting down control plane")
	grpcServer.GracefulStop()
}
