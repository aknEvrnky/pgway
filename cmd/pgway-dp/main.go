package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aknEvrnky/pgway/internal/adapters/agentruntime"
	grpcclient "github.com/aknEvrnky/pgway/internal/adapters/grpc/client"
	"github.com/aknEvrnky/pgway/internal/adapters/http"
	"github.com/aknEvrnky/pgway/internal/adapters/proxy/net"
	"github.com/aknEvrnky/pgway/internal/adapters/pubsub/memory"
	"github.com/aknEvrnky/pgway/internal/application/consumer"
	"github.com/aknEvrnky/pgway/internal/application/core/api"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	_ "github.com/aknEvrnky/pgway/internal/platform/logger"
	"go.uber.org/zap"
)

type changeWatcher struct {
	client *grpcclient.Client
	pub    *memory.PubSub
}

func (w changeWatcher) Watch(ctx context.Context) error {
	return w.client.Watch(ctx, w.pub)
}

func main() {
	configPath := flag.String("config", "", "path to config file (default: search /etc/pgway, $HOME/.pgway, .)")
	flag.Parse()

	if err := config.Load(*configPath); err != nil {
		zap.L().Fatal("load configuration", zap.Error(err))
	}

	cfg := config.Get()

	lock, err := agentruntime.AcquireLock(agentruntime.LockPath(cfg.AgentStatePath))
	if err != nil {
		zap.L().Fatal("acquire agent lock", zap.Error(err))
	}
	defer lock.Close()

	cpClient, err := grpcclient.NewClient(cfg.GrpcListenAddr, "")
	if err != nil {
		zap.L().Fatal("connect to control plane", zap.Error(err), zap.String("addr", cfg.GrpcListenAddr))
	}
	defer cpClient.Close()

	agent := domain.Agent{
		Id:      cfg.AgentName,
		Version: "dev",
		Labels:  cfg.AgentLabels,
	}
	if err := agentruntime.EnsureHostname(&agent); err != nil {
		zap.L().Fatal("resolve hostname", zap.Error(err))
	}

	boot, err := agentruntime.BootstrapCredentials(
		context.Background(),
		cfg.AgentStatePath,
		cfg.RegistrationToken,
		agent,
		cpClient,
	)
	if err != nil {
		zap.L().Fatal("agent bootstrap", zap.Error(err))
	}
	cpClient.SetToken(boot.AgentToken)
	zap.L().Info("agent ready", zap.String("agent_id", boot.AgentID))

	app := api.NewApplication(cpClient, cpClient)
	ctx := context.Background()

	if err := app.Bootstrap(ctx); err != nil {
		zap.L().Fatal("bootstrap", zap.Error(err))
	}

	proxyTransport := net.NewAdapter()
	httpAdapter, err := http.NewHttpAdapter(ctx, app, proxyTransport)
	if err != nil {
		zap.L().Fatal("init http adapter", zap.Error(err))
	}

	localBus := memory.NewPubSub(10)
	eventConsumer := consumer.NewConsumer(localBus, app, httpAdapter)

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
		hbErr <- agentruntime.RunHeartbeat(sigCtx, cpClient, cfg.HeartbeatInterval)
	}()

	go func() {
		consumeErr <- eventConsumer.ConsumeEvents(sigCtx)
	}()

	go func() {
		watchErr <- agentruntime.RunWatch(sigCtx, changeWatcher{client: cpClient, pub: localBus}, func(c context.Context) error {
			return app.Bootstrap(c)
		})
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
