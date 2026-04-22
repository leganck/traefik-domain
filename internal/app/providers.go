package app

import (
	"github.com/leganck/traefik-domain/dns"
	"github.com/leganck/traefik-domain/internal/service"
	"github.com/leganck/traefik-domain/traefik"
	"github.com/leganck/traefik-domain/web"
	log "github.com/sirupsen/logrus"
)

func (a *App) reloadProviders() {
	if a.dnsManager == nil {
		return
	}
	a.dnsManager.SetProviders(a.buildProviderInstances())
	a.reloadTraefikClient()
}

func (a *App) buildProviderInstances() []*service.ProviderInstance {
	configs := a.providersConfig.GetProviders()
	instances := make([]*service.ProviderInstance, 0, len(configs))

	for _, cfg := range configs {
		cfgCopy := cfg
		providerInstance, err := dns.NewDNSProvider(&cfgCopy, a.stateStore, log.WithFields(log.Fields{"provider": cfg.Name, "provider_id": cfg.ProviderID}))
		if err != nil {
			log.WithError(err).WithFields(log.Fields{"provider": cfg.Name, "provider_id": cfg.ProviderID}).Error("Failed to initialize provider")
			continue
		}

		instances = append(instances, &service.ProviderInstance{
			ID:       cfg.ProviderID,
			Name:     cfg.Name,
			Provider: providerInstance,
		})
		log.WithFields(log.Fields{"provider": cfg.Name, "provider_id": cfg.ProviderID}).Info("Provider initialized")
	}

	return instances
}

func requestsToJobs(requests []web.DomainUpdateRequest) []service.DNSJob {
	jobs := make([]service.DNSJob, 0, len(requests))
	for _, req := range requests {
		jobs = append(jobs, service.DNSJob{
			Domain:            req.Domain,
			ProviderID:        req.ProviderID,
			Enabled:           req.Enabled,
			OverwriteExisting: req.OverwriteExisting,
		})
	}
	return jobs
}

func (a *App) reloadTraefikClient() {
	tfCfg := a.providersConfig.GetTraefikConfig()
	a.traefikClient = traefik.NewHTTPClient(tfCfg.Host, tfCfg.Username, tfCfg.Password, nil)
}
