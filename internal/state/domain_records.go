package state

import "reflect"

func (sc *DomainSyncState) UpdateRecords(providerName string, records map[string]*RecordInfo) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	changed := false

	for domain, cfg := range sc.Records {
		if cfg == nil {
			continue
		}
		if cfg.Records == nil {
			cfg.Records = make(map[string]*RecordInfo)
		}
		if info, ok := records[domain]; ok {
			if existing, exists := cfg.Records[providerName]; !exists || !reflect.DeepEqual(existing, info) {
				cfg.Records[providerName] = info
				changed = true
			}
			if info.Managed {
				if pref := sc.Preferences[domain]; pref != nil && pref.Overrides != nil {
					if _, exists := pref.Overrides[providerName]; exists {
						delete(pref.Overrides, providerName)
						changed = true
					}
				}
			}
		} else {
			if _, exists := cfg.Records[providerName]; exists {
				changed = true
			}
			delete(cfg.Records, providerName)
			if pref := sc.Preferences[domain]; pref != nil && pref.Overrides != nil {
				if _, exists := pref.Overrides[providerName]; exists {
					delete(pref.Overrides, providerName)
					changed = true
				}
			}
		}
	}
	if changed {
		sc.markDirtyLocked()
	}
	return nil
}
