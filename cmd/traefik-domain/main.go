package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/leganck/traefik-domain/config"
	internalapp "github.com/leganck/traefik-domain/internal/app"
	log "github.com/sirupsen/logrus"
)

func main() {
	providersConfig := config.NewProvidersConfig()
	if err := providersConfig.LoadFromSources(); err != nil {
		log.WithError(fmt.Errorf("load providers config: %w", err)).Fatal("Failed to initialize providers config")
	}

	if err := providersConfig.StartWatcher(); err != nil {
		log.WithError(fmt.Errorf("start config watcher: %w", err)).Fatal("Failed to start config watcher")
	}
	defer providersConfig.StopWatcher()

	app := internalapp.New(providersConfig)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app.Run(ctx, providersConfig.GetReloadChan())
}
