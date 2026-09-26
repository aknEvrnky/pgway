package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aknEvrnky/pgway/internal/adapters/grpc/server"
	"github.com/aknEvrnky/pgway/internal/adapters/http"
	"github.com/aknEvrnky/pgway/internal/adapters/probes"
	proxyadapter "github.com/aknEvrnky/pgway/internal/adapters/proxy/net"
	"github.com/aknEvrnky/pgway/internal/adapters/pubsub/memory"
	badgerrepo "github.com/aknEvrnky/pgway/internal/adapters/repository/badger"
	"github.com/aknEvrnky/pgway/internal/adapters/rest"
	agentapp "github.com/aknEvrnky/pgway/internal/application/agent"
	"github.com/aknEvrnky/pgway/internal/application/auth"
	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/aknEvrnky/pgway/internal/application/dataplane/agenthost"
	"github.com/aknEvrnky/pgway/internal/application/dataplane/api"
	"github.com/aknEvrnky/pgway/internal/application/dataplane/consumer"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/aknEvrnky/pgway/internal/platform/logger"
	"github.com/aknEvrnky/pgway/internal/platform/metrics"
	"github.com/aknEvrnky/pgway/internal/platform/tracing"
	"github.com/aknEvrnky/pgway/internal/platform/version"
	badgerdb "github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "", "path to config file (default: search /etc/pgway, $HOME/.pgway, .)")
	showVersion := flag.Bool("v", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		_, _ = fmt.Fprintln(os.Stdout, version.Line("pgway"))
		return
	}

	if err := config.Load(*configPath); err != nil {
		zap.L().Fatal("load configuration", zap.Error(err))
	}

	cfg := config.Get()
	if err := logger.SetLevel(cfg.LogLevel); err != nil {
		zap.L().Fatal("set log level", zap.Error(err))
	}
	zap.L().Info("starting", zap.String("version", version.String()), zap.String("commit", version.Commit))

	stopMetrics, err := metrics.Setup(context.Background(), metrics.Config{
		Enabled:        cfg.Otel.Enabled,
		Endpoint:       cfg.Otel.Endpoint,
		ServiceName:    cfg.Otel.ServiceName,
		ExportInterval: cfg.Otel.ExportInterval,
		Insecure:       cfg.Otel.Insecure,
		Version:        version.String(),
	}, "pgway")
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
		Version:     version.String(),
	}, "pgway")
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

	// BadgerDB
	opts := badgerdb.DefaultOptions(cfg.Badger.Path).WithLogger(badgerrepo.NewBadgerLogger())
	db, err := badgerdb.Open(opts)
	if err != nil {
		zap.L().Fatal("open badger", zap.Error(err), zap.String("path", cfg.Badger.Path))
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

	userRepo := badgerrepo.NewUserRepository(db)
	agentRepo := badgerrepo.NewAgentRepository(db)
	tokenRepo := badgerrepo.NewTokenRepository(db)
	regTokenRepo := badgerrepo.NewRegistrationTokenRepository(db)

	// Auth — Service owns user accounts/sessions; Authenticator resolves
	// bearer tokens to principals for the transports.
	authService := auth.NewService(userRepo, tokenRepo, cfg.Auth.TokenTTL)
	authenticator := auth.NewAuthenticator(userRepo, agentRepo, tokenRepo)
	agentCreds := auth.NewAgentCredentialService(tokenRepo, regTokenRepo)
	agentService := agentapp.NewService(agentRepo, agentCreds, cfg.Auth.AgentTokenTTL)

	if err := authService.Bootstrap(context.Background()); err != nil {
		zap.L().Fatal("auth bootstrap", zap.Error(err))
	}

	ctx := context.Background()
	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	go badgerrepo.RunValueLogGC(sigCtx, db, cfg.Badger.GCInterval, zap.L())

	// gRPC server — cli's command bus, auth enforced
	grpcServer := server.New(cpService, cpService, authService, authService, authenticator, agentService, pubSub, sigCtx, server.AgentServerConfig{
		HeartbeatThreshold:   cfg.Agent.HeartbeatThreshold,
		AgentTokenTTL:        cfg.Auth.AgentTokenTTL,
		RegistrationTokenTTL: cfg.Auth.RegistrationTokenTTL,
	}, server.KeepaliveConfig{
		Interval: cfg.GRPC.KeepaliveInterval,
		Timeout:  cfg.GRPC.KeepaliveTimeout,
	}, server.RateLimitConfig{
		RPS:   cfg.GRPC.RateLimitRPS,
		Burst: cfg.GRPC.RateLimitBurst,
	})

	lis, err := net.Listen("tcp", cfg.GRPC.ListenAddr)
	if err != nil {
		zap.L().Fatal("grpc listen", zap.Error(err), zap.String("addr", cfg.GRPC.ListenAddr))
	}

	gate := probes.NewReadyGate(probes.ReadyGateConfig{
		Storage:     badgerrepo.NewStoragePinger(db),
		RequireGRPC: true,
	})
	gate.MarkGRPCServing(true)

	// Data Plane — cpService as read only service
	app := api.NewApplication(cpService, cpService, zap.L())

	if err := app.Bootstrap(ctx); err != nil {
		zap.L().Fatal("bootstrap", zap.Error(err))
	}

	proxyTransport := proxyadapter.NewAdapter(proxyadapter.TransportConfig{
		MaxIdleConns:        cfg.Proxy.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.Proxy.MaxIdleConnsPerHost,
		IdleConnTimeout:     cfg.Proxy.IdleConnTimeout,
		DialTimeout:         cfg.Proxy.DialTimeout,
		DNSCacheEnabled:     cfg.Proxy.DNSCache.Enabled,
		DNSCacheTTL:         cfg.Proxy.DNSCache.TTL,
	})
	httpAdapter, err := http.NewHttpAdapter(ctx, app, proxyTransport, int64(cfg.Proxy.MaxRequestBodyBytes), nil)
	if err != nil {
		zap.L().Fatal("init http adapter", zap.Error(err))
	}

	// REST adapter
	var restAdapter *rest.Adapter
	if cfg.Rest.Enabled {
		restAdapter = rest.NewRestAdapter(cpService, authenticator, cfg.Rest)
	}

	var probeAdapter *probes.Adapter
	if cfg.Probes.Enabled {
		probeAdapter = probes.New(cfg.Probes.ListenAddr, gate)
	}

	// event consumer — handler order matters: app refreshes the cache first,
	// then the http adapter reads the refreshed cache
	eventConsumer := consumer.NewConsumer(zap.L(), pubSub, consumer.CoalesceConfig{
		Window:    cfg.Dataplane.EventCoalesceWindow,
		MaxBuffer: cfg.Dataplane.EventCoalesceMaxBuffer,
	}, app, httpAdapter)

	runErr := make(chan error, 5)

	go func() {
		zap.L().Info("grpc started", zap.String("addr", cfg.GRPC.ListenAddr))
		runErr <- grpcServer.Serve(lis)
	}()

	go func() {
		zap.L().Info("gateway started")
		runErr <- httpAdapter.Run(sigCtx)
	}()

	if restAdapter != nil {
		go func() {
			zap.L().Info("restapi started")
			runErr <- restAdapter.Run(sigCtx)
		}()
	}

	go func() {
		zap.L().Info("event consumer started")
		if err := eventConsumer.ConsumeEvents(sigCtx); err != nil {
			runErr <- err
		}
	}()

	if probeAdapter != nil {
		go func() {
			runErr <- probeAdapter.Run()
		}()
	}

	if cfg.Dataplane.EventResyncInterval > 0 {
		go func() {
			ticker := time.NewTicker(cfg.Dataplane.EventResyncInterval)
			defer ticker.Stop()
			for {
				select {
				case <-sigCtx.Done():
					return
				case <-ticker.C:
					if err := agenthost.Resync(sigCtx, app, httpAdapter); err != nil {
						zap.L().Warn("periodic resync", zap.Error(err))
					}
				}
			}
		}()
	}

	select {
	case <-sigCtx.Done():
	case err := <-runErr:
		zap.L().Error("server failed", zap.Error(err))
		stop()
	}

	// Graceful shutdown
	zap.L().Info("shutting down")
	gate.MarkShuttingDown()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if probeAdapter != nil {
		if err := probeAdapter.Shutdown(shutdownCtx); err != nil {
			zap.L().Error("probes shutdown", zap.Error(err))
		}
	}

	server.GracefulStopWithTimeout(grpcServer, 5*time.Second)

	if err := httpAdapter.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("http shutdown", zap.Error(err))
	}

	if restAdapter != nil {
		if err := restAdapter.Shutdown(shutdownCtx); err != nil {
			zap.L().Error("rest shutdown", zap.Error(err))
		}
	}
}
