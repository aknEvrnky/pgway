package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcserver "github.com/aknEvrnky/pgway/internal/adapters/grpc/server"
	"github.com/aknEvrnky/pgway/internal/adapters/pubsub/memory"
	"github.com/aknEvrnky/pgway/internal/adapters/rest"

	badgerrepo "github.com/aknEvrnky/pgway/internal/adapters/repository/badger"
	agentapp "github.com/aknEvrnky/pgway/internal/application/agent"
	"github.com/aknEvrnky/pgway/internal/application/auth"
	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/aknEvrnky/pgway/internal/platform/logger"
	badgerdb "github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "", "path to config file (default: search /etc/pgway, $HOME/.pgway, .)")
	flag.Parse()

	if err := config.Load(*configPath); err != nil {
		zap.L().Fatal("load configuration", zap.Error(err))
	}

	cfg := config.Get()
	if err := logger.SetLevel(cfg.LogLevel); err != nil {
		zap.L().Fatal("set log level", zap.Error(err))
	}

	opts := badgerdb.DefaultOptions(cfg.BadgerPath).WithLogger(badgerrepo.NewBadgerLogger())
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

	pubsub := memory.NewPubSub(10)

	cpService := controlplane.NewService(
		proxyRepo,
		poolRepo,
		lbRepo,
		routerRepo,
		flowRepo,
		epRepo,
		pubsub,
	)

	userRepo := badgerrepo.NewUserRepository(db)
	agentRepo := badgerrepo.NewAgentRepository(db)
	tokenRepo := badgerrepo.NewTokenRepository(db)
	regTokenRepo := badgerrepo.NewRegistrationTokenRepository(db)

	authService := auth.NewService(userRepo, tokenRepo, cfg.TokenTTL)
	authenticator := auth.NewAuthenticator(userRepo, agentRepo, tokenRepo)
	agentCreds := auth.NewAgentCredentialService(tokenRepo, regTokenRepo)
	agentService := agentapp.NewService(agentRepo, agentCreds, cfg.AgentTokenTTL)

	if err := authService.Bootstrap(context.Background()); err != nil {
		zap.L().Fatal("auth bootstrap", zap.Error(err))
	}

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	grpcServer := grpcserver.New(cpService, cpService, authService, authService, authenticator, agentService, pubsub, sigCtx, grpcserver.AgentServerConfig{
		HeartbeatThreshold:   cfg.AgentHeartbeatThreshold,
		AgentTokenTTL:        cfg.AgentTokenTTL,
		RegistrationTokenTTL: cfg.RegistrationTokenTTL,
	})

	lis, err := net.Listen("tcp", cfg.GrpcListenAddr)
	if err != nil {
		zap.L().Fatal("listen", zap.Error(err), zap.String("grpc_listen_addr", cfg.GrpcListenAddr))
	}

	restAdapter := rest.NewRestAdapter(cpService, cfg.RestListenAddr)

	go func() {
		zap.L().Info("control plane started", zap.String("grpc", cfg.GrpcListenAddr))
		err := grpcServer.Serve(lis)
		if err != nil {
			zap.L().Fatal("grpc serve", zap.Error(err))
		}
	}()

	go func() {
		if err := restAdapter.Run(sigCtx); err != nil {
			zap.L().Fatal("rest serve", zap.Error(err))
		}
	}()

	<-sigCtx.Done()
	zap.L().Info("shutting down control plane")
	grpcserver.GracefulStopWithTimeout(grpcServer, 5*time.Second)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := restAdapter.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("rest shutdown", zap.Error(err))
	}
}
