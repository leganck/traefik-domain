package state

func (sc *DomainSyncState) MergeDomains(domains []string) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	changed := false
	if sc.ProviderGlobals == nil {
		sc.ProviderGlobals = make(map[string]bool)
	}

	for domain, cfg := range sc.Discovery {
		if cfg != nil && cfg.InTraefik {
			cfg.InTraefik = false
			changed = true
		}
		if sc.Preferences[domain] == nil {
			// keep maps aligned when discovery exists for a domain that has no preference entry yet
			sc.Preferences[domain] = &DomainPreference{Providers: make(map[string]bool), Overrides: make(map[string]bool)}
		}
	}

	for _, domain := range domains {
		if _, exists := sc.Discovery[domain]; !exists {
			sc.Discovery[domain] = &DomainDiscovery{InTraefik: true}
			changed = true
		} else if !sc.Discovery[domain].InTraefik {
			sc.Discovery[domain].InTraefik = true
			changed = true
		}
		if _, exists := sc.Preferences[domain]; !exists {
			sc.Preferences[domain] = &DomainPreference{Providers: make(map[string]bool), Overrides: make(map[string]bool)}
			changed = true
		}
		if _, exists := sc.Records[domain]; !exists {
			sc.Records[domain] = &DomainRecordCache{Records: make(map[string]*RecordInfo)}
		}
	}

	if changed {
		sc.markDirtyLocked()
	}
	return nil
}

func (sc *DomainSyncState) GetEnabledTraefikDomains(provider string) []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var result []string
	for domain, cfg := range sc.Discovery {
		if cfg != nil && cfg.InTraefik && sc.providerEnabledLocked(domain, provider) {
			result = append(result, domain)
		}
	}
	return result
}
