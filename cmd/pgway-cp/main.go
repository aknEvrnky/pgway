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
	"github.com/aknEvrnky/pgway/internal/adapters/probes"
	"github.com/aknEvrnky/pgway/internal/adapters/pubsub/memory"
	"github.com/aknEvrnky/pgway/internal/adapters/rest"

	badgerrepo "github.com/aknEvrnky/pgway/internal/adapters/repository/badger"
	agentapp "github.com/aknEvrnky/pgway/internal/application/agent"
	"github.com/aknEvrnky/pgway/internal/application/auth"
	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/aknEvrnky/pgway/internal/platform/logger"
	"github.com/aknEvrnky/pgway/internal/platform/metrics"
	"github.com/aknEvrnky/pgway/internal/platform/tracing"
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

	stopMetrics, err := metrics.Setup(context.Background(), metrics.Config{
		Enabled:        cfg.Otel.Enabled,
		Endpoint:       cfg.Otel.Endpoint,
		ServiceName:    cfg.Otel.ServiceName,
		ExportInterval: cfg.Otel.ExportInterval,
		Insecure:       cfg.Otel.Insecure,
	}, "pgway-cp")
	if err != nil {
		zap.L().Fatal("init metrics", zap.Error(err))
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := stopMetrics(shutdownCtx); err != nil {
			zap.L().Warn("metrics shutdown", zap.Error(err))
		}
	}()

	stopTracing, err := tracing.Setup(context.Background(), tracing.Config{
		Enabled:     cfg.Otel.Enabled && cfg.Otel.TracesEnabled,
		Endpoint:    cfg.Otel.Endpoint,
		ServiceName: cfg.Otel.ServiceName,
		Insecure:    cfg.Otel.Insecure,
		SampleRatio: cfg.Otel.TraceSampleRatio,
	}, "pgway-cp")
	if err != nil {
		zap.L().Fatal("init tracing", zap.Error(err))
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := stopTracing(shutdownCtx); err != nil {
			zap.L().Warn("tracing shutdown", zap.Error(err))
		}
	}()

	opts := badgerdb.DefaultOptions(cfg.Badger.Path).WithLogger(badgerrepo.NewBadgerLogger())
	db, err := badgerdb.Open(opts)
	if err != nil {
		zap.L().Fatal("open badger", zap.Error(err), zap.String("path", cfg.Badger.Path))
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

	authService := auth.NewService(userRepo, tokenRepo, cfg.Auth.TokenTTL)
	authenticator := auth.NewAuthenticator(userRepo, agentRepo, tokenRepo)
	agentCreds := auth.NewAgentCredentialService(tokenRepo, regTokenRepo)
	agentService := agentapp.NewService(agentRepo, agentCreds, cfg.Auth.AgentTokenTTL)

	if err := authService.Bootstrap(context.Background()); err != nil {
		zap.L().Fatal("auth bootstrap", zap.Error(err))
	}

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go badgerrepo.RunValueLogGC(sigCtx, db, cfg.Badger.GCInterval, zap.L())

	gate := probes.NewReadyGate(probes.ReadyGateConfig{
		Storage:     badgerrepo.NewStoragePinger(db),
		RequireGRPC: true,
	})

	grpcServer := grpcserver.New(cpService, cpService, authService, authService, authenticator, agentService, pubsub, sigCtx, grpcserver.AgentServerConfig{
		HeartbeatThreshold:   cfg.Agent.HeartbeatThreshold,
		AgentTokenTTL:        cfg.Auth.AgentTokenTTL,
		RegistrationTokenTTL: cfg.Auth.RegistrationTokenTTL,
	}, grpcserver.KeepaliveConfig{
		Interval: cfg.GRPC.KeepaliveInterval,
		Timeout:  cfg.GRPC.KeepaliveTimeout,
	}, grpcserver.RateLimitConfig{
		RPS:   cfg.GRPC.RateLimitRPS,
		Burst: cfg.GRPC.RateLimitBurst,
	})

	lis, err := net.Listen("tcp", cfg.GRPC.ListenAddr)
	if err != nil {
		zap.L().Fatal("listen", zap.Error(err), zap.String("grpc_listen_addr", cfg.GRPC.ListenAddr))
	}
	gate.MarkGRPCServing(true)

	restAdapter := rest.NewRestAdapter(cpService, cfg.Rest.ListenAddr)

	var probeAdapter *probes.Adapter
	if cfg.Probes.Enabled {
		probeAdapter = probes.New(cfg.Probes.ListenAddr, gate)
		go func() {
			if err := probeAdapter.Run(); err != nil {
				zap.L().Fatal("probes serve", zap.Error(err))
			}
		}()
	}

	go func() {
		zap.L().Info("control plane started", zap.String("grpc", cfg.GRPC.ListenAddr))
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
	gate.MarkShuttingDown()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if probeAdapter != nil {
		if err := probeAdapter.Shutdown(shutdownCtx); err != nil {
			zap.L().Error("probes shutdown", zap.Error(err))
		}
	}

	grpcserver.GracefulStopWithTimeout(grpcServer, 5*time.Second)

	if err := restAdapter.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("rest shutdown", zap.Error(err))
	}
}
