package state

import (
	"sync"
	"time"
)

const DomainSyncStatePath = "./data/domain_sync_state.json"

const (
	DomainPreferencesPath = "./data/domain_preferences.json"
	DomainDiscoveryPath   = "./data/domain_discovery.json"
	DomainRecordsPath     = "./data/domain_records.json"
)

type RecordInfo struct {
	ID      string `json:"id,omitempty"`
	Value   string `json:"value"`
	Type    string `json:"type"`
	Managed bool   `json:"managed"`
}

type DomainPreference struct {
	Providers map[string]bool `json:"providers"`
	Overrides map[string]bool `json:"overrides,omitempty"`
}

type DomainDiscovery struct {
	InTraefik bool `json:"inTraefik"`
}

type DomainRecordCache struct {
	Records map[string]*RecordInfo `json:"records"`
}

// DomainConfig is kept as a compatibility view while callers migrate to the split state types.
type DomainConfig struct {
	Providers map[string]bool        `json:"providers"`
	Overrides map[string]bool        `json:"overrides,omitempty"`
	InTraefik bool                   `json:"inTraefik"`
	Records   map[string]*RecordInfo `json:"records"`
}

type DomainSyncState struct {
	mu              sync.RWMutex
	Preferences     map[string]*DomainPreference  `json:"preferences"`
	Discovery       map[string]*DomainDiscovery   `json:"discovery"`
	Records         map[string]*DomainRecordCache `json:"records"`
	ProviderGlobals map[string]bool               `json:"providerGlobals,omitempty"`
	path            string
	preferencesPath string
	discoveryPath   string
	recordsPath     string
	dirty           bool
	saveDelay       time.Duration
	saveTimer       *time.Timer
}
