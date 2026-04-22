package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
)

type persistedDomainPreferences struct {
	Domains         map[string]*persistedDomainPreference `json:"domains"`
	ProviderGlobals map[string]bool                       `json:"providerGlobals"`
}

type persistedDomainPreference struct {
	Providers map[string]bool `json:"providers"`
	Overrides map[string]bool `json:"overrides,omitempty"`
}

type persistedDomainDiscovery struct {
	Domains map[string]*persistedDomainDiscoveryEntry `json:"domains"`
}

type persistedDomainDiscoveryEntry struct {
	InTraefik bool `json:"inTraefik"`
}

type persistedDomainRecords struct {
	Domains map[string]*persistedDomainRecordsEntry `json:"domains"`
}

type persistedDomainRecordsEntry struct {
	Records map[string]*RecordInfo `json:"records"`
}

func NewDomainSyncState() *DomainSyncState {
	return &DomainSyncState{
		Preferences:     make(map[string]*DomainPreference),
		Discovery:       make(map[string]*DomainDiscovery),
		Records:         make(map[string]*DomainRecordCache),
		ProviderGlobals: make(map[string]bool),
		path:            DomainSyncStatePath,
		preferencesPath: DomainPreferencesPath,
		discoveryPath:   DomainDiscoveryPath,
		recordsPath:     DomainRecordsPath,
		saveDelay:       time.Second,
	}
}

func (sc *DomainSyncState) Load() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	ensureStatePathsLocked(sc)

	prefs, prefsNeedsSave, err := loadDomainPreferences(sc.preferencesPath)
	if err != nil {
		return err
	}
	discovery, discoveryNeedsSave, err := loadDomainDiscovery(sc.discoveryPath)
	if err != nil {
		return err
	}
	records, recordsNeedsSave, err := loadDomainRecords(sc.recordsPath)
	if err != nil {
		return err
	}

	applyPersistedSplitStateLocked(sc, prefs, discovery, records)
	if prefsNeedsSave || discoveryNeedsSave || recordsNeedsSave {
		return sc.saveUnlocked()
	}
	return nil
}

func (sc *DomainSyncState) Save() error {
	return sc.Flush()
}

func (sc *DomainSyncState) Flush() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.flushUnlocked()
}

func ensureStatePathsLocked(sc *DomainSyncState) {
	if sc.preferencesPath == "" {
		sc.preferencesPath = DomainPreferencesPath
	}
	if sc.discoveryPath == "" {
		sc.discoveryPath = DomainDiscoveryPath
	}
	if sc.recordsPath == "" {
		sc.recordsPath = DomainRecordsPath
	}
}

func loadDomainPreferences(path string) (*persistedDomainPreferences, bool, error) {
	data, missing, err := readStateFile(path)
	if err != nil {
		return nil, false, err
	}
	if missing {
		return &persistedDomainPreferences{Domains: make(map[string]*persistedDomainPreference), ProviderGlobals: make(map[string]bool)}, true, nil
	}
	var persisted persistedDomainPreferences
	if err := json.Unmarshal(data, &persisted); err != nil {
		backupCorruptedState(path)
		return &persistedDomainPreferences{Domains: make(map[string]*persistedDomainPreference), ProviderGlobals: make(map[string]bool)}, true, nil
	}
	if persisted.Domains == nil {
		persisted.Domains = make(map[string]*persistedDomainPreference)
	}
	if persisted.ProviderGlobals == nil {
		persisted.ProviderGlobals = make(map[string]bool)
	}
	return &persisted, false, nil
}

func loadDomainDiscovery(path string) (*persistedDomainDiscovery, bool, error) {
	data, missing, err := readStateFile(path)
	if err != nil {
		return nil, false, err
	}
	if missing {
		return &persistedDomainDiscovery{Domains: make(map[string]*persistedDomainDiscoveryEntry)}, true, nil
	}
	var persisted persistedDomainDiscovery
	if err := json.Unmarshal(data, &persisted); err != nil {
		backupCorruptedState(path)
		return &persistedDomainDiscovery{Domains: make(map[string]*persistedDomainDiscoveryEntry)}, true, nil
	}
	if persisted.Domains == nil {
		persisted.Domains = make(map[string]*persistedDomainDiscoveryEntry)
	}
	return &persisted, false, nil
}

func loadDomainRecords(path string) (*persistedDomainRecords, bool, error) {
	data, missing, err := readStateFile(path)
	if err != nil {
		return nil, false, err
	}
	if missing {
		return &persistedDomainRecords{Domains: make(map[string]*persistedDomainRecordsEntry)}, true, nil
	}
	var persisted persistedDomainRecords
	if err := json.Unmarshal(data, &persisted); err != nil {
		backupCorruptedState(path)
		return &persistedDomainRecords{Domains: make(map[string]*persistedDomainRecordsEntry)}, true, nil
	}
	if persisted.Domains == nil {
		persisted.Domains = make(map[string]*persistedDomainRecordsEntry)
	}
	return &persisted, false, nil
}

func readStateFile(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("failed to read state file %s: %w", path, err)
	}
	return data, false, nil
}

func backupCorruptedState(path string) {
	backupPath := path + ".backup." + time.Now().Format("20060102-150405")
	log.Warnf("State file corrupted, backing up to %s", backupPath)
	if err := os.Rename(path, backupPath); err != nil {
		log.Warnf("Failed to backup corrupted state file %s: %v", path, err)
	}
}

func applyPersistedSplitStateLocked(sc *DomainSyncState, prefs *persistedDomainPreferences, discovery *persistedDomainDiscovery, records *persistedDomainRecords) {
	resetStateUnlocked(sc)

	for domain, persisted := range prefs.Domains {
		cfg := normalizePreference(persisted)
		sc.Preferences[domain] = cfg
	}
	for providerID, enabled := range prefs.ProviderGlobals {
		sc.ProviderGlobals[providerID] = enabled
	}
	for domain, persisted := range discovery.Domains {
		cfg := normalizeDiscovery(persisted)
		sc.Discovery[domain] = cfg
		ensureEmptyPreferenceLocked(sc, domain)
	}
	for domain, persisted := range records.Domains {
		cfg := normalizeRecords(persisted)
		sc.Records[domain] = cfg
		ensureEmptyPreferenceLocked(sc, domain)
	}
	for domain := range sc.Preferences {
		ensureEmptyDiscoveryLocked(sc, domain)
		ensureEmptyRecordsLocked(sc, domain)
	}
}

func resetStateUnlocked(sc *DomainSyncState) {
	sc.Preferences = make(map[string]*DomainPreference)
	sc.Discovery = make(map[string]*DomainDiscovery)
	sc.Records = make(map[string]*DomainRecordCache)
	sc.ProviderGlobals = make(map[string]bool)
	sc.dirty = false
	if sc.saveTimer != nil {
		sc.saveTimer.Stop()
		sc.saveTimer = nil
	}
}

func ensureEmptyPreferenceLocked(sc *DomainSyncState, domain string) {
	if _, exists := sc.Preferences[domain]; !exists {
		sc.Preferences[domain] = &DomainPreference{Providers: make(map[string]bool), Overrides: make(map[string]bool)}
	}
}

func ensureEmptyDiscoveryLocked(sc *DomainSyncState, domain string) {
	if _, exists := sc.Discovery[domain]; !exists {
		sc.Discovery[domain] = &DomainDiscovery{}
	}
}

func ensureEmptyRecordsLocked(sc *DomainSyncState, domain string) {
	if _, exists := sc.Records[domain]; !exists {
		sc.Records[domain] = &DomainRecordCache{Records: make(map[string]*RecordInfo)}
	}
}

func normalizePreference(cfg *persistedDomainPreference) *DomainPreference {
	if cfg == nil {
		return &DomainPreference{Providers: make(map[string]bool), Overrides: make(map[string]bool)}
	}
	return &DomainPreference{Providers: copyBoolMap(cfg.Providers), Overrides: copyBoolMap(cfg.Overrides)}
}

func normalizeDiscovery(cfg *persistedDomainDiscoveryEntry) *DomainDiscovery {
	if cfg == nil {
		return &DomainDiscovery{}
	}
	return &DomainDiscovery{InTraefik: cfg.InTraefik}
}

func normalizeRecords(cfg *persistedDomainRecordsEntry) *DomainRecordCache {
	if cfg == nil {
		return &DomainRecordCache{Records: make(map[string]*RecordInfo)}
	}
	return &DomainRecordCache{Records: copyRecordMap(cfg.Records)}
}

func saveSplitState(sc *DomainSyncState) error {
	if err := saveStateFile(sc.preferencesPath, buildPersistedPreferences(sc)); err != nil {
		return err
	}
	if err := saveStateFile(sc.discoveryPath, buildPersistedDiscovery(sc)); err != nil {
		return err
	}
	if err := saveStateFile(sc.recordsPath, buildPersistedRecords(sc)); err != nil {
		return err
	}
	return nil
}

func saveStateFile(path string, value any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state file %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write state file %s: %w", path, err)
	}
	return nil
}

func buildPersistedPreferences(sc *DomainSyncState) persistedDomainPreferences {
	persisted := persistedDomainPreferences{
		Domains:         make(map[string]*persistedDomainPreference),
		ProviderGlobals: copyBoolMap(sc.ProviderGlobals),
	}
	for domain, cfg := range sc.Preferences {
		if cfg == nil {
			continue
		}
		persisted.Domains[domain] = &persistedDomainPreference{Providers: copyBoolMap(cfg.Providers), Overrides: copyBoolMap(cfg.Overrides)}
	}
	return persisted
}

func buildPersistedDiscovery(sc *DomainSyncState) persistedDomainDiscovery {
	persisted := persistedDomainDiscovery{Domains: make(map[string]*persistedDomainDiscoveryEntry)}
	for domain, cfg := range sc.Discovery {
		if cfg == nil {
			continue
		}
		persisted.Domains[domain] = &persistedDomainDiscoveryEntry{InTraefik: cfg.InTraefik}
	}
	return persisted
}

func buildPersistedRecords(sc *DomainSyncState) persistedDomainRecords {
	persisted := persistedDomainRecords{Domains: make(map[string]*persistedDomainRecordsEntry)}
	for domain, cfg := range sc.Records {
		if cfg == nil {
			continue
		}
		persisted.Domains[domain] = &persistedDomainRecordsEntry{Records: copyRecordMap(cfg.Records)}
	}
	return persisted
}

func (sc *DomainSyncState) saveUnlocked() error {
	return saveSplitState(sc)
}

func (sc *DomainSyncState) flushUnlocked() error {
	if !sc.dirty {
		return nil
	}
	if sc.saveTimer != nil {
		sc.saveTimer.Stop()
		sc.saveTimer = nil
	}
	if err := sc.saveUnlocked(); err != nil {
		return err
	}
	sc.dirty = false
	return nil
}

func (sc *DomainSyncState) markDirtyLocked() {
	sc.dirty = true
	if sc.saveTimer != nil {
		sc.saveTimer.Stop()
	}
	sc.saveTimer = time.AfterFunc(sc.saveDelay, func() {
		if err := sc.Flush(); err != nil {
			log.Warnf("Failed to flush domain sync state: %v", err)
		}
	})
}
