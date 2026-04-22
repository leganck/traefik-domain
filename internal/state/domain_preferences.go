package state

func (sc *DomainSyncState) SetDomainProvider(domain string, provider string, enabled bool, overwrite bool) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	cfg, exists := sc.Preferences[domain]
	if !exists || cfg == nil {
		cfg = &DomainPreference{Providers: make(map[string]bool), Overrides: make(map[string]bool)}
		sc.Preferences[domain] = cfg
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]bool)
	}
	if cfg.Overrides == nil {
		cfg.Overrides = make(map[string]bool)
	}
	currentEnabled := sc.providerEnabledLocked(domain, provider)
	currentOverwrite := cfg.Overrides[provider]
	if currentEnabled == enabled && currentOverwrite == overwrite {
		return nil
	}

	if enabled == sc.ProviderGlobals[provider] {
		delete(cfg.Providers, provider)
	} else {
		cfg.Providers[provider] = enabled
	}
	if enabled && overwrite {
		cfg.Overrides[provider] = true
	} else {
		delete(cfg.Overrides, provider)
	}
	sc.markDirtyLocked()
	return nil
}

func (sc *DomainSyncState) SetProviderGlobal(provider string, enabled bool) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	changed := false
	if sc.ProviderGlobals == nil {
		sc.ProviderGlobals = make(map[string]bool)
	}
	if current, ok := sc.ProviderGlobals[provider]; !ok || current != enabled {
		sc.ProviderGlobals[provider] = enabled
		changed = true
	}

	for _, cfg := range sc.Preferences {
		if cfg == nil {
			continue
		}
		if cfg.Providers == nil {
			cfg.Providers = make(map[string]bool)
		}
		if cfg.Overrides == nil {
			cfg.Overrides = make(map[string]bool)
		}
		if _, exists := cfg.Providers[provider]; exists {
			delete(cfg.Providers, provider)
			changed = true
		}
		if _, exists := cfg.Overrides[provider]; exists {
			delete(cfg.Overrides, provider)
			changed = true
		}
	}

	if changed {
		sc.markDirtyLocked()
	}
	return nil
}

func (sc *DomainSyncState) GetProviderGlobals() map[string]bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	globals := make(map[string]bool, len(sc.ProviderGlobals))
	for providerID, enabled := range sc.ProviderGlobals {
		globals[providerID] = enabled
	}
	return globals
}

func (sc *DomainSyncState) ShouldSync(domain string, provider string) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return sc.providerEnabledLocked(domain, provider)
}

func (sc *DomainSyncState) ShouldOverwrite(domain string, provider string) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	cfg, exists := sc.Preferences[domain]
	if !exists || cfg == nil {
		return false
	}

	return cfg.Overrides[provider]
}

func (sc *DomainSyncState) GetEnabledDomains(provider string) []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var result []string
	for domain := range sc.Preferences {
		if sc.providerEnabledLocked(domain, provider) {
			result = append(result, domain)
		}
	}
	return result
}

func (sc *DomainSyncState) RemoveProvider(providerName string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	changed := false
	if sc.ProviderGlobals != nil {
		if _, exists := sc.ProviderGlobals[providerName]; exists {
			delete(sc.ProviderGlobals, providerName)
			changed = true
		}
	}

	for _, cfg := range sc.Preferences {
		if cfg != nil {
			if cfg.Providers != nil {
				if _, exists := cfg.Providers[providerName]; exists {
					changed = true
				}
				delete(cfg.Providers, providerName)
			}
			if cfg.Overrides != nil {
				if _, exists := cfg.Overrides[providerName]; exists {
					changed = true
				}
				delete(cfg.Overrides, providerName)
			}
		}
	}

	if changed {
		sc.markDirtyLocked()
	}
}

func (sc *DomainSyncState) providerEnabledLocked(domain string, provider string) bool {
	if cfg, exists := sc.Preferences[domain]; exists && cfg != nil && cfg.Providers != nil {
		if enabled, exists := cfg.Providers[provider]; exists {
			return enabled
		}
	}
	if sc.ProviderGlobals != nil {
		return sc.ProviderGlobals[provider]
	}
	return false
}

func copyPreferenceLocked(cfg *DomainPreference) *DomainPreference {
	if cfg == nil {
		return nil
	}
	return &DomainPreference{Providers: copyBoolMap(cfg.Providers), Overrides: copyBoolMap(cfg.Overrides)}
}
