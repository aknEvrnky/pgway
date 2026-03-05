package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aknEvrnky/pgway/internal/adapters/http"
	lbRepo "github.com/aknEvrnky/pgway/internal/adapters/repository/balancer/config"
	epRepo "github.com/aknEvrnky/pgway/internal/adapters/repository/entrypoint/config"
	flowRepo "github.com/aknEvrnky/pgway/internal/adapters/repository/flow/config"
	routerRepo "github.com/aknEvrnky/pgway/internal/adapters/repository/router/config"
	"github.com/aknEvrnky/pgway/internal/application/core/api"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	_ "github.com/aknEvrnky/pgway/internal/platform/logger"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("c", "", "path to config file")
	flag.Parse()

	if err := config.Load(*configPath); err != nil {
		zap.L().Fatal("load configuration file", zap.Error(err))
	}

	entryPointRepository, err := epRepo.NewConfigRepository(config.Get())

	if err != nil {
		zap.L().Fatal("init entrypoints", zap.Error(err))
	}

	routerRepository, err := routerRepo.NewConfigRepository(config.Get())

	if err != nil {
		zap.L().Fatal("init routers", zap.Error(err))
	}

	flowRepository, err := flowRepo.NewConfigRepository(config.Get())

	if err != nil {
		zap.L().Fatal("init flows", zap.Error(err))
	}

	lbRepository, err := lbRepo.NewConfigRepository(config.Get())

	if err != nil {
		zap.L().Fatal("init load balancers", zap.Error(err))
	}

	app := api.NewApplication(entryPointRepository, flowRepository, routerRepository, lbRepository)
	ctx := context.Background()

	err = app.ValidateAll(ctx)
	if err != nil {
		zap.L().Fatal("init application", zap.Error(err))
	}

	httpAdapter, err := http.NewHttpAdapter(ctx, app)

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
