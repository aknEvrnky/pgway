package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aknEvrnky/pgway/internal/adapters/agentstate"
	grpcclient "github.com/aknEvrnky/pgway/internal/adapters/grpc/client"
	"github.com/aknEvrnky/pgway/internal/adapters/http"
	"github.com/aknEvrnky/pgway/internal/adapters/proxy/net"
	"github.com/aknEvrnky/pgway/internal/adapters/pubsub/memory"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/application/dataplane/agenthost"
	"github.com/aknEvrnky/pgway/internal/application/dataplane/api"
	"github.com/aknEvrnky/pgway/internal/application/dataplane/consumer"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/aknEvrnky/pgway/internal/platform/logger"
	"go.uber.org/zap"
)

type changeWatcher struct {
	client *grpcclient.Client
	pub    *memory.PubSub
}

func (w changeWatcher) Watch(ctx context.Context, afterConnect func(context.Context) error) error {
	return w.client.Watch(ctx, w.pub, afterConnect)
}

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

	lock, err := agentstate.NewLock(cfg.Agent.StatePath).Acquire()
	if err != nil {
		zap.L().Fatal("acquire agent lock", zap.Error(err))
	}
	defer lock.Close()

	cpClient, err := grpcclient.NewClient(cfg.GRPC.ListenAddr, "", grpcclient.KeepaliveConfig{
		Interval: cfg.GRPC.KeepaliveInterval,
		Timeout:  cfg.GRPC.KeepaliveTimeout,
	})
	if err != nil {
		zap.L().Fatal("connect to control plane", zap.Error(err), zap.String("addr", cfg.GRPC.ListenAddr))
	}
	defer cpClient.Close()

	agent := domain.Agent{
		Id:      cfg.Agent.Name,
		Version: "dev",
		Labels:  cfg.Agent.Labels,
	}
	if agent.Hostname == "" {
		hostname, err := os.Hostname()
		if err != nil {
			zap.L().Fatal("resolve hostname", zap.Error(err))
		}
		agent.Hostname = hostname
	}

	store := agentstate.NewStore(cfg.Agent.StatePath)
	boot, err := agenthost.BootstrapCredentials(
		context.Background(),
		store,
		cfg.Agent.RegistrationToken,
		agent,
		cpClient,
	)
	if err != nil {
		zap.L().Fatal("agent bootstrap", zap.Error(err))
	}
	cpClient.SetToken(boot.AgentToken)
	zap.L().Info("agent ready", zap.String("agent_id", boot.AgentID))

	app := api.NewApplication(cpClient, cpClient, zap.L())
	ctx := context.Background()

	if err := app.Bootstrap(ctx); err != nil {
		zap.L().Fatal("bootstrap", zap.Error(err))
	}

	proxyTransport := net.NewAdapter(net.TransportConfig{
		MaxIdleConns:        cfg.Proxy.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.Proxy.MaxIdleConnsPerHost,
		IdleConnTimeout:     cfg.Proxy.IdleConnTimeout,
		DialTimeout:         cfg.Proxy.DialTimeout,
		DNSCacheEnabled:     cfg.Proxy.DNSCache.Enabled,
		DNSCacheTTL:         cfg.Proxy.DNSCache.TTL,
	})
	httpAdapter, err := http.NewHttpAdapter(ctx, app, proxyTransport, int64(cfg.Proxy.MaxRequestBodyBytes))
	if err != nil {
		zap.L().Fatal("init http adapter", zap.Error(err))
	}

	localBus := memory.NewPubSub(10)
	eventConsumer := consumer.NewConsumer(zap.L(), localBus, app, httpAdapter)

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	runErr := make(chan error, 1)
	hbErr := make(chan error, 1)
	watchErr := make(chan error, 1)
	consumeErr := make(chan error, 1)

	go func() {
		zap.L().Info("gateway started")
		runErr <- httpAdapter.Run(sigCtx)
	}()

	go func() {
		hbErr <- agenthost.RunHeartbeat(sigCtx, zap.L(), cpClient, cfg.Agent.HeartbeatInterval)
	}()

	go func() {
		consumeErr <- eventConsumer.ConsumeEvents(sigCtx)
	}()

	go func() {
		watchErr <- agenthost.RunWatch(sigCtx, zap.L(), changeWatcher{client: cpClient, pub: localBus}, func(c context.Context) error {
			return app.Bootstrap(c)
		}, agenthost.WatchOptions{})
	}()

	select {
	case <-sigCtx.Done():
	case err := <-runErr:
		zap.L().Error("server failed", zap.Error(err))
		stop()
	case err := <-hbErr:
		if err != nil && sigCtx.Err() == nil {
			zap.L().Error("heartbeat stopped", zap.Error(err))
			stop()
		}
	case err := <-watchErr:
		if err != nil && sigCtx.Err() == nil {
			zap.L().Error("watch stopped", zap.Error(err))
			stop()
		}
	case err := <-consumeErr:
		if err != nil && sigCtx.Err() == nil {
			zap.L().Error("event consumer stopped", zap.Error(err))
			stop()
		}
	}

	deregCtx, deregCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := cpClient.Deregister(deregCtx); err != nil {
		zap.L().Warn("deregister agent", zap.Error(err))
	}
	deregCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpAdapter.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("http shutdown", zap.Error(err))
	}
}
