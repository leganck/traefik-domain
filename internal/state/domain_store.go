package state

import (
	"fmt"

	"github.com/leganck/traefik-domain/dns/model"
)

func (sc *DomainSyncState) GetPreferences() map[string]*DomainPreference {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	configCopy := make(map[string]*DomainPreference)
	for domain, cfg := range sc.Preferences {
		if cfg != nil {
			configCopy[domain] = copyPreferenceLocked(cfg)
		}
	}
	return configCopy
}

func (sc *DomainSyncState) GetDiscovery() map[string]*DomainDiscovery {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	configCopy := make(map[string]*DomainDiscovery)
	for domain, cfg := range sc.Discovery {
		if cfg != nil {
			copyCfg := *cfg
			configCopy[domain] = &copyCfg
		}
	}

	return configCopy
}

func (sc *DomainSyncState) GetRecords() map[string]*DomainRecordCache {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	configCopy := make(map[string]*DomainRecordCache)
	for domain, cfg := range sc.Records {
		if cfg != nil {
			configCopy[domain] = &DomainRecordCache{Records: copyRecordMap(cfg.Records)}
		}
	}
	return configCopy
}

func (sc *DomainSyncState) DeleteDomain(domain string) (map[string]bool, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	_, exists := sc.Preferences[domain]
	if !exists {
		return nil, fmt.Errorf("domain %s not found", domain)
	}

	providers := make(map[string]bool)
	for providerID := range sc.ProviderGlobals {
		providers[providerID] = sc.providerEnabledLocked(domain, providerID)
	}
	if cfg := sc.Preferences[domain]; cfg != nil {
		for providerID := range cfg.Providers {
			providers[providerID] = sc.providerEnabledLocked(domain, providerID)
		}
	}

	delete(sc.Preferences, domain)
	delete(sc.Discovery, domain)
	delete(sc.Records, domain)
	sc.markDirtyLocked()
	return providers, nil
}

func (sc *DomainSyncState) GetAllMainDomains() []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	seen := make(map[string]bool)
	var result []string
	for domain := range sc.Preferences {
		_, mainDomain, err := model.SplitDomain(domain)
		if err != nil {
			continue
		}
		if !seen[mainDomain] {
			seen[mainDomain] = true
			result = append(result, mainDomain)
		}
	}
	return result
}

func (sc *DomainSyncState) GetDomain(domain string) *DomainConfig {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if _, exists := sc.Preferences[domain]; exists {
		return &DomainConfig{Providers: copyBoolMap(sc.Preferences[domain].Providers), Overrides: copyBoolMap(sc.Preferences[domain].Overrides), InTraefik: sc.Discovery[domain] != nil && sc.Discovery[domain].InTraefik, Records: copyRecordMap(getRecordsLocked(sc.Records, domain))}
	}

	return &DomainConfig{Providers: make(map[string]bool), Overrides: make(map[string]bool)}
}

func getRecordsLocked(records map[string]*DomainRecordCache, domain string) map[string]*RecordInfo {
	if cfg := records[domain]; cfg != nil {
		return cfg.Records
	}
	return nil
}
