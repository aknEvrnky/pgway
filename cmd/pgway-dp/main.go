package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcclient "github.com/aknEvrnky/pgway/internal/adapters/grpc/client"
	"github.com/aknEvrnky/pgway/internal/adapters/http"
	"github.com/aknEvrnky/pgway/internal/adapters/proxy/net"
	"github.com/aknEvrnky/pgway/internal/application/core/api"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	_ "github.com/aknEvrnky/pgway/internal/platform/logger"
	"go.uber.org/zap"
)

func main() {
	if err := config.Load(""); err != nil {
		zap.L().Fatal("load configuration", zap.Error(err))
	}

	cfg := config.Get()

	// connect CP through gRPC
	cpClient, err := grpcclient.NewClient(cfg.GrpcListenAddr)
	if err != nil {
		zap.L().Fatal("connect to control plane", zap.Error(err), zap.String("addr", cfg.GrpcListenAddr))
	}
	defer cpClient.Close()

	// app takes cp client as read-only
	app := api.NewApplication(cpClient)
	ctx := context.Background()

	if err := app.Bootstrap(ctx); err != nil {
		zap.L().Fatal("bootstrap", zap.Error(err))
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
		zap.L().Info("gateway started")
		runErr <- httpAdapter.Run(sigCtx)
	}()

	select {
	case <-sigCtx.Done():
	case err := <-runErr:
		zap.L().Error("server failed", zap.Error(err))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpAdapter.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("http shutdown", zap.Error(err))
	}
}
