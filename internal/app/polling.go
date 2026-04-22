package app

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

type pollingLoop struct {
	app           *App
	traefikTicker *time.Ticker
	dnsTicker     *time.Ticker
}

func newPollingLoop(app *App) *pollingLoop {
	return &pollingLoop{app: app}
}

func (l *pollingLoop) Run(ctx context.Context, reloadCh <-chan struct{}) {
	l.resetTickers()
	defer l.stop()

	for {
		select {
		case <-l.traefikTicker.C:
			l.app.PollTraefik(ctx)

		case <-l.dnsTicker.C:
			l.app.PollDNS()

		case <-reloadCh:
			log.Info("Reloading providers config")
			if err := l.app.Reload(); err != nil {
				log.WithError(err).Warn("Failed to reload providers config from disk")
				continue
			}
			applyLogLevel(l.app.providersConfig.GetLogLevel())
			l.resetTickers()

		case <-ctx.Done():
			return
		}
	}
}

func (l *pollingLoop) resetTickers() {
	l.stop()
	l.traefikTicker, l.dnsTicker = l.app.NewPollTickers()
}

func (l *pollingLoop) stop() {
	if l.traefikTicker != nil {
		l.traefikTicker.Stop()
		l.traefikTicker = nil
	}
	if l.dnsTicker != nil {
		l.dnsTicker.Stop()
		l.dnsTicker = nil
	}
}

func (a *App) NewPollTickers() (*time.Ticker, *time.Ticker) {
	traefikPollInterval := time.Duration(a.providersConfig.GetTraefikPollInterval()) * time.Second
	dnsPollInterval := time.Duration(a.providersConfig.GetDNSPollInterval()) * time.Second
	log.Infof("Traefik poll interval: %v, DNS poll interval: %v", traefikPollInterval, dnsPollInterval)
	return time.NewTicker(traefikPollInterval), time.NewTicker(dnsPollInterval)
}

func (a *App) PollTraefik(ctx context.Context) {
	if !a.traefikMutex.TryLock() {
		log.Debug("previous traefik poll still running, skipping this round")
		return
	}
	defer a.traefikMutex.Unlock()

	tfCfg := a.providersConfig.GetTraefikConfig()
	if tfCfg.Host == "" {
		log.Info("Traefik not configured, skipping traefik poll")
		return
	}

	if a.traefikClient == nil {
		a.reloadTraefikClient()
	}

	domains, err := a.traefikClient.Domains(ctx)
	if err != nil {
		log.Errorf("traefik domains error: %v", err)
		return
	}

	if a.stateStore == nil {
		return
	}

	var domainList []string
	for _, domainObjs := range domains {
		for _, d := range domainObjs {
			domainList = append(domainList, d.CustomDomain)
		}
	}
	if err := a.stateStore.MergeDomains(domainList); err != nil {
		log.Warnf("Failed to merge domains to switch config: %v", err)
	}
	log.Infof("Traefik poll: discovered %d domains", len(domainList))
}

func (a *App) PollDNS() {
	if !a.dnsMutex.TryLock() {
		log.Debug("previous DNS poll still running, skipping this round")
		return
	}
	defer a.dnsMutex.Unlock()
	_ = a.pollDNSLocked()
}

func (a *App) pollDNSLocked() error {
	if a.stateStore == nil {
		log.Debug("switch config not enabled, skipping DNS poll")
		return nil
	}

	if a.dnsManager == nil {
		log.Debug("dns manager not initialized, skipping DNS poll")
		return nil
	}

	if a.dnsManager.ProviderCount() == 0 {
		log.Info("No DNS providers configured, skipping DNS poll")
		return nil
	}

	return a.dnsManager.RefreshAllStates()
}
