package app

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/leganck/traefik-domain/config"
	"github.com/leganck/traefik-domain/internal/service"
	"github.com/leganck/traefik-domain/internal/state"
	"github.com/leganck/traefik-domain/traefik"
	"github.com/leganck/traefik-domain/web"
	log "github.com/sirupsen/logrus"
)

type App struct {
	providersConfig *config.ProvidersConfig
	stateStore      *state.DomainSyncState
	httpServer      *http.Server
	dnsManager      *service.DNSManager
	traefikClient   traefik.Client
	traefikMutex    sync.Mutex
	dnsMutex        sync.Mutex
}

func New(providersConfig *config.ProvidersConfig) *App {
	return &App{providersConfig: providersConfig}
}

func (a *App) Run(ctx context.Context, reloadCh <-chan struct{}) {
	applyLogLevel(a.providersConfig.GetLogLevel())
	a.initRuntime()
	defer a.Shutdown()

	a.runInitialSync(ctx)
	newPollingLoop(a).Run(ctx, reloadCh)
	log.Println("exit")
}

func (a *App) Reload() error {
	if err := a.providersConfig.ReloadFromSources(); err != nil {
		return err
	}
	if a.dnsManager != nil {
		a.reloadProviders()
		log.Infof("Reloaded %d provider(s)", a.dnsManager.ProviderCount())
	}
	return nil
}

func (a *App) initRuntime() {
	a.stateStore = a.initState()
	a.dnsManager = service.NewDNSManager(a.stateStore)
	a.reloadProviders()
	a.httpServer = a.startWebUI(a.stateStore)
	log.Infof("Started with %d provider(s)", a.dnsManager.ProviderCount())
}

func (a *App) initState() *state.DomainSyncState {
	switchConfig := state.NewDomainSyncState()
	if err := switchConfig.Load(); err != nil {
		log.Warnf("Failed to load switch config: %v", err)
	}
	return switchConfig
}

func (a *App) startWebUI(switchConfig *state.DomainSyncState) *http.Server {
	if !a.providersConfig.GetWebEnabled() {
		return nil
	}

	handler := web.NewHandler(switchConfig, a.providersConfig)
	handler.SetApplyDomainUpdatesFunc(a.applyDomainUpdates)
	port := a.providersConfig.GetWebPort()
	httpServer, err := web.StartServer(port, handler)
	if err != nil {
		log.Errorf("Failed to start web server: %v", err)
		return nil
	}

	log.Infof("Web UI enabled on http://localhost:%d", port)
	return httpServer
}

func (a *App) applyDomainUpdates(requests []web.DomainUpdateRequest) error {
	if a.dnsManager == nil {
		return fmt.Errorf("dns manager not initialized")
	}
	return a.dnsManager.Apply(requestsToJobs(requests))
}

func (a *App) runInitialSync(ctx context.Context) {
	a.PollTraefik(ctx)
	a.PollDNS()
	log.Info("Initial synchronization completed")
}

func (a *App) Shutdown() {
	if a.dnsManager != nil {
		a.dnsManager.Stop()
		if err := a.stateStore.Flush(); err != nil {
			log.WithError(err).Warn("Failed to flush switch config on shutdown")
		}
	}
	if a.httpServer != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
			log.WithError(err).Warn("Failed to shutdown web server")
		}
		shutdownCancel()
	}
}

func applyLogLevel(level string) {
	parsedLevel, err := log.ParseLevel(level)
	if err != nil {
		log.WithError(err).Warn("Invalid log level, falling back to info")
		parsedLevel = log.InfoLevel
	}
	log.SetLevel(parsedLevel)
}
