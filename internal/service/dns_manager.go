package service

import (
	"fmt"
	"strings"
	"sync"

	"github.com/leganck/traefik-domain/dns"
	"github.com/leganck/traefik-domain/internal/state"
	log "github.com/sirupsen/logrus"
)

type ProviderInstance struct {
	ID       string
	Name     string
	Provider *dns.Provider
}

type DNSJob struct {
	Domain            string
	ProviderID        string
	Enabled           bool
	OverwriteExisting bool
}

type DNSManager struct {
	switchConfig *state.DomainSyncState
	mu           sync.RWMutex
	providers    map[string]*ProviderInstance
}

func NewDNSManager(switchConfig *state.DomainSyncState) *DNSManager {
	return &DNSManager{
		switchConfig: switchConfig,
		providers:    make(map[string]*ProviderInstance),
	}
}

func (m *DNSManager) Stop() {}

func (m *DNSManager) SetProviders(providers []*ProviderInstance) {
	providerMap := make(map[string]*ProviderInstance, len(providers))
	for _, pi := range providers {
		providerMap[pi.ID] = pi
	}

	m.mu.Lock()
	m.providers = providerMap
	m.mu.Unlock()
}

func (m *DNSManager) Apply(jobs []DNSJob) error {
	if len(jobs) == 0 {
		return nil
	}

	grouped := make(map[string][]DNSJob)
	for _, job := range jobs {
		grouped[job.ProviderID] = append(grouped[job.ProviderID], job)
	}

	providers := make(map[string]*ProviderInstance, len(grouped))
	m.mu.RLock()
	for providerID := range grouped {
		provider, ok := m.providers[providerID]
		if !ok {
			m.mu.RUnlock()
			return fmt.Errorf("provider %s not found", providerID)
		}
		providers[providerID] = provider
	}
	m.mu.RUnlock()

	for providerID, batch := range grouped {
		if err := m.processBatch(providers[providerID], batch); err != nil {
			return err
		}
	}

	return nil
}

func (m *DNSManager) RefreshAllStates() error {
	if m.switchConfig == nil {
		return nil
	}

	for _, provider := range m.providersSnapshot() {
		if err := refreshProviderRecordState(provider, m.switchConfig); err != nil {
			return err
		}
	}
	return nil
}

func (m *DNSManager) ProviderCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.providers)
}

func (m *DNSManager) providersSnapshot() []*ProviderInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]*ProviderInstance, 0, len(m.providers))
	for _, provider := range m.providers {
		providers = append(providers, provider)
	}
	return providers
}

func (m *DNSManager) processBatch(provider *ProviderInstance, batch []DNSJob) error {
	if len(batch) == 0 {
		return nil
	}

	latestByDomain := make(map[string]DNSJob, len(batch))
	for _, job := range batch {
		latestByDomain[job.Domain] = job
	}

	var errs []string

	for _, job := range latestByDomain {
		var err error
		if job.Enabled {
			err = provider.Provider.EnsureDomain(job.Domain, job.OverwriteExisting)
		} else {
			err = provider.Provider.DeleteManagedDomain(job.Domain)
		}
		if err != nil {
			log.WithError(err).WithFields(log.Fields{"provider": provider.Name, "provider_id": provider.ID, "domain": job.Domain}).Error("Failed to process DNS job")
			errs = append(errs, fmt.Sprintf("%s: %v", job.Domain, err))
		}
	}

	if m.switchConfig != nil {
		if err := refreshProviderRecordState(provider, m.switchConfig); err != nil {
			log.WithError(err).WithFields(log.Fields{"provider": provider.Name, "provider_id": provider.ID}).Warn("Failed to refresh provider state after DNS jobs")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("provider %s apply failed: %s", provider.Name, strings.Join(errs, "; "))
	}

	return nil
}

func refreshProviderRecordState(pi *ProviderInstance, switchConfig *state.DomainSyncState) error {
	mainDomains := switchConfig.GetAllMainDomains()
	providerMap := make(map[string]*state.RecordInfo)
	for _, mainDomain := range mainDomains {
		records, err := pi.Provider.ListRecords(mainDomain)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{"provider": pi.Name, "provider_id": pi.ID}).Warnf("Failed to list records for cache: %s", mainDomain)
			continue
		}
		for _, r := range records {
			if r.CustomDomain != "" {
				providerMap[r.CustomDomain] = &state.RecordInfo{
					ID:      r.Id,
					Value:   r.Value,
					Type:    r.Type,
					Managed: r.Managed,
				}
			}
		}
	}
	if err := switchConfig.UpdateRecords(pi.ID, providerMap); err != nil {
		log.WithError(err).WithFields(log.Fields{"provider": pi.Name, "provider_id": pi.ID}).Warn("Failed to update record cache")
		return err
	}
	return nil
}
