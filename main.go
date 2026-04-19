package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/leganck/traefik-domain/config"
	"github.com/leganck/traefik-domain/dns"
	"github.com/leganck/traefik-domain/dns/model"
	"github.com/leganck/traefik-domain/dns/provider"
	"github.com/leganck/traefik-domain/traefik"
	"github.com/leganck/traefik-domain/web"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type ProviderInstance struct {
	id       string
	name     string
	provider *dns.Provider
}

var (
	activeProviders []*ProviderInstance
	providersMu     sync.RWMutex
	mutex           = sync.Mutex{}
)

func main() {
	providersConfig := config.NewProvidersConfig()
	exists := true
	if err := providersConfig.Load(); err != nil {
		if os.IsNotExist(err) {
			exists = false
		} else {
			log.WithError(err).Fatal("Failed to load providers config")
		}
	}

	applyEnvVarsToConfig(providersConfig)

	var conf struct {
		PollInterval int
		WebEnabled   bool
		WebPort      int
		LogLevel     string
	}
	conf.PollInterval = providersConfig.GetPollInterval()
	conf.WebEnabled = providersConfig.GetWebEnabled()
	conf.WebPort = providersConfig.GetWebPort()
	conf.LogLevel = providersConfig.GetLogLevel()

	logLevel, _ := log.ParseLevel(conf.LogLevel)
	log.SetLevel(logLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if !exists {
		initProvidersFromEnvVars(providersConfig)
	}

	var switchConfig *config.SwitchConfig
	var httpServer *http.Server
	var err error
	if conf.WebEnabled {
		switchConfig = config.NewSwitchConfig()
		if err := switchConfig.Load(); err != nil {
			log.Warnf("Failed to load switch config: %v", err)
		}
		handler := web.NewHandler(switchConfig, providersConfig)
		handler.SetDeleteDomainFunc(func(domain, providerID string) error {
			providersMu.RLock()
			defer providersMu.RUnlock()
			for _, pi := range activeProviders {
				if pi.id == providerID {
					return pi.provider.DeleteDomain(domain)
				}
			}
			return fmt.Errorf("provider %s not found", providerID)
		})
		httpServer, err = web.StartServer(conf.WebPort, handler)
		if err != nil {
			log.Errorf("Failed to start web server: %v", err)
		} else {
			log.Infof("Web UI enabled on http://localhost:%d", conf.WebPort)
		}
	}

	activeProviders = initProviders(providersConfig, switchConfig)

	log.Infof("Started with %d provider(s)", len(activeProviders))

	ticker := time.NewTicker(time.Duration(conf.PollInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !mutex.TryLock() {
				log.Debug("previous sync still running, skipping this round")
				continue
			}

			tfCfg := providersConfig.GetTraefikConfig()
			if tfCfg.Host == "" {
				mutex.Unlock()
				log.Info("Traefik not configured, skipping sync")
				continue
			}

			domains, err := traefik.TraefikDomains(tfCfg.Host, tfCfg.Username, tfCfg.Password)
			if err != nil {
				mutex.Unlock()
				log.Errorf("traefik domains error: %v", err)
				continue
			}

			if switchConfig != nil {
				var domainList []string
				for _, domainObjs := range domains {
					for _, d := range domainObjs {
						domainList = append(domainList, d.CustomDomain)
					}
				}
				providerIDs := providersConfig.GetProviderIDs()
				if err := switchConfig.MergeDomains(domainList, providerIDs); err != nil {
					log.Warnf("Failed to merge domains to switch config: %v", err)
				}
			}

			providersMu.RLock()
			providers := activeProviders
			providersMu.RUnlock()

			if len(providers) == 0 {
				mutex.Unlock()
				log.Info("No DNS providers configured, skipping sync")
				continue
			}

			syncFromCache(providers, switchConfig)
			updateRecordCache(providers, switchConfig)
			mutex.Unlock()

		case <-providersConfig.GetReloadChan():
			log.Info("Reloading providers config")
			providersMu.Lock()
			activeProviders = initProviders(providersConfig, switchConfig)
			providersMu.Unlock()
			log.Infof("Reloaded %d provider(s)", len(activeProviders))

		case <-sigChan:
			log.Println("received shutdown signal, exiting...")
			if httpServer != nil {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				httpServer.Shutdown(shutdownCtx)
				shutdownCancel()
			}
			cancel()
			return

		case <-ctx.Done():
			log.Println("exit")
			return
		}
	}
}

func applyEnvVarsToConfig(pc *config.ProvidersConfig) {
	pollInterval := 5
	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		if n, _ := fmt.Sscanf(v, "%d", &pollInterval); n != 1 || pollInterval <= 0 {
			pollInterval = 5
		}
	}

	webEnabled := true
	if v := os.Getenv("WEB_ENABLED"); v != "" {
		webEnabled = strings.ToLower(v) == "true"
	}

	webPort := 8080
	if v := os.Getenv("WEB_PORT"); v != "" {
		if n, _ := fmt.Sscanf(v, "%d", &webPort); n != 1 || webPort <= 0 {
			webPort = 8080
		}
	}

	logLevel := "info"
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		logLevel = v
	}

	pc.SetAppConfig(pollInterval, webEnabled, webPort, logLevel)
	log.Infof("Applied app config from env vars: poll_interval=%d, web_enabled=%v, web_port=%d, log_level=%s",
		pollInterval, webEnabled, webPort, logLevel)
}

func initProvidersFromEnvVars(pc *config.ProvidersConfig) {
	traefikHost := os.Getenv("TRAEFIK_HOST")
	if traefikHost != "" {
		pc.SetTraefikConfig(config.TraefikConfig{
			Host:     traefikHost,
			Username: os.Getenv("TRAEFIK_USERNAME"),
			Password: os.Getenv("TRAEFIK_PASSWORD"),
		})
		log.Info("Initialized Traefik config from env vars")
	}

	dnsName := os.Getenv("DNS_NAME")
	dnsID := os.Getenv("DNS_ID")
	dnsSecret := os.Getenv("DNS_SECRET")
	if dnsName != "" && dnsID != "" && dnsSecret != "" {
		providerCfg := config.ProviderConfig{
			ProviderID:  config.GenerateProviderID(),
			Name:        dnsName,
			Type:        dnsName,
			ID:          dnsID,
			Secret:      dnsSecret,
			RecordValue: os.Getenv("DNS_RECORD_VALUE"),
		}
		switch strings.ToLower(dnsName) {
		case "adguard":
			providerCfg.Host = os.Getenv("ADGUARD_HOST")
		case "openwrt":
			providerCfg.Host = os.Getenv("OPENWRT_HOST")
		}
		if err := pc.AddProvider(providerCfg); err != nil {
			log.WithError(err).Warn("Failed to add provider from env vars")
		} else {
			log.Infof("Initialized provider %s from env vars", dnsName)
		}
	}
}

func initProviders(pc *config.ProvidersConfig, switchConfig *config.SwitchConfig) []*ProviderInstance {
	configs := pc.GetProviders()
	var instances []*ProviderInstance

	for _, cfg := range configs {
		initCfg := &provider.ProviderConfig{
			ProviderID:  cfg.ProviderID,
			Name:        cfg.Name,
			Type:        cfg.Type,
			ID:          cfg.ID,
			Secret:      cfg.Secret,
			Host:        cfg.Host,
			RecordValue: cfg.RecordValue,
		}

		providerInstance, err := dns.NewDNSProvider(initCfg, switchConfig, log.WithField("provider", cfg.Name))
		if err != nil {
			log.WithError(err).WithField("provider", cfg.Name).Error("Failed to initialize provider")
			continue
		}

		instances = append(instances, &ProviderInstance{
			id:       cfg.ProviderID,
			name:     cfg.Name,
			provider: providerInstance,
		})
		log.WithField("provider", cfg.Name).Info("Provider initialized")
	}

	return instances
}

func syncFromCache(providers []*ProviderInstance, switchConfig *config.SwitchConfig) {
	var wg sync.WaitGroup
	for _, pi := range providers {
		wg.Add(1)
		go func(p *ProviderInstance) {
			defer wg.Done()
			enabledDomains := switchConfig.GetEnabledDomains(p.id)
			if len(enabledDomains) == 0 {
				log.WithField("provider", p.name).Debug("No enabled domains to sync")
				return
			}
			log.WithField("provider", p.name).Info("Syncing domains from cache")
			domainMap := make(map[string][]*traefik.Domain)
			for _, customDomain := range enabledDomains {
				subDomain, mainDomain, err := model.SplitDomain(customDomain)
				if err != nil {
					log.WithError(err).Warnf("Failed to split domain %s", customDomain)
					continue
				}
				domainMap[mainDomain] = append(domainMap[mainDomain], &traefik.Domain{
					MainDomain:   mainDomain,
					SubDomain:    subDomain,
					CustomDomain: customDomain,
				})
			}
			for mainDomain, domains := range domainMap {
				if err := p.provider.AddOrUpdateCname(mainDomain, domains); err != nil {
					log.WithError(err).WithField("provider", p.name).Error("Failed to sync domains")
				}
			}
		}(pi)
	}
	wg.Wait()
}

func updateRecordCache(providers []*ProviderInstance, switchConfig *config.SwitchConfig) {
	mainDomains := switchConfig.GetAllMainDomains()
	if len(mainDomains) == 0 {
		return
	}

	for _, pi := range providers {
		providerMap := make(map[string]*config.RecordInfo)
		for _, mainDomain := range mainDomains {
			records, err := pi.provider.ListRecords(mainDomain)
			if err != nil {
				log.WithError(err).WithField("provider", pi.name).Warnf("Failed to list records for cache: %s", mainDomain)
				continue
			}
			for _, r := range records {
				if r.CustomDomain != "" {
					providerMap[r.CustomDomain] = &config.RecordInfo{
						Value: r.Value,
						Type:  r.Type,
					}
				}
			}
		}
		switchConfig.UpdateRecords(pi.id, providerMap)
	}

	if err := switchConfig.Save(); err != nil {
		log.Warnf("Failed to save record cache: %v", err)
	}
}
