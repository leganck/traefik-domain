package web

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// DomainProviderConfig represents the provider enablement for a domain
type DomainProviderConfig struct {
	DNSPod      bool `json:"dnspod"`
	AdGuard     bool `json:"adguard"`
	Cloudflare  bool `json:"cloudflare"`
	OpenWRT     bool `json:"openwrt"`
}

// DomainConfig represents configuration for a single domain
type DomainConfig struct {
	Providers DomainProviderConfig `json:"providers"`
}

// SwitchConfig represents the overall switch configuration with thread-safe access
type SwitchConfig struct {
	mu      sync.RWMutex
	Domains map[string]*DomainConfig `json:"domains"`
	path    string
}

// NewSwitchConfig creates a new SwitchConfig instance
func NewSwitchConfig(path string) *SwitchConfig {
	return &SwitchConfig{
		Domains: make(map[string]*DomainConfig),
		path:    path,
	}
}

// Load reads the configuration from the JSON file
// If the file doesn't exist, creates a default empty config
// If the file is corrupted, backs it up and creates a new default config
func (sc *SwitchConfig) Load() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(sc.path); os.IsNotExist(err) {
		log.Infof("Config file %s does not exist, creating default config", sc.path)
		return sc.saveUnlocked()
	}

	// Read file
	data, err := os.ReadFile(sc.path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var config struct {
		Domains map[string]*DomainConfig `json:"domains"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		// Backup corrupted file
		backupPath := sc.path + ".backup." + time.Now().Format("YYYYMMDD-HHMMSS")
		log.Warnf("Config file is corrupted, backing up to %s", backupPath)
		if err := os.Rename(sc.path, backupPath); err != nil {
			return fmt.Errorf("failed to backup corrupted config: %w", err)
		}
		// Reset to empty config
		sc.Domains = make(map[string]*DomainConfig)
		return sc.saveUnlocked()
	}

	// Ensure all domains have provider configs initialized
	sc.Domains = make(map[string]*DomainConfig)
	for domain, cfg := range config.Domains {
		if cfg == nil {
			cfg = &DomainConfig{
				Providers: DomainProviderConfig{},
			}
		}
		sc.Domains[domain] = cfg
	}

	return nil
}

// Save writes the configuration to the JSON file
func (sc *SwitchConfig) Save() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.saveUnlocked()
}

// saveUnlocked writes the configuration without locking
// Must be called with lock held
func (sc *SwitchConfig) saveUnlocked() error {
	// Ensure directory exists
	dir := filepath.Dir(sc.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal JSON with indentation
	data, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(sc.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfig returns a copy of the current configuration
func (sc *SwitchConfig) GetConfig() map[string]*DomainConfig {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	configCopy := make(map[string]*DomainConfig)
	for domain, cfg := range sc.Domains {
		if cfg != nil {
			configCopy[domain] = &DomainConfig{
				Providers: cfg.Providers,
			}
		}
	}
	return configCopy
}

// GetDomain returns the configuration for a specific domain
// If the domain doesn't exist, returns a new DomainConfig with all providers disabled
func (sc *SwitchConfig) GetDomain(domain string) *DomainConfig {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if cfg, exists := sc.Domains[domain]; exists && cfg != nil {
		return &DomainConfig{
			Providers: cfg.Providers,
		}
	}

	// Return default config with all providers disabled
	return &DomainConfig{
		Providers: DomainProviderConfig{},
	}
}

// SetDomainProvider sets a specific provider enablement for a domain
func (sc *SwitchConfig) SetDomainProvider(domain string, provider string, enabled bool) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Get or create domain config
	cfg, exists := sc.Domains[domain]
	if !exists || cfg == nil {
		cfg = &DomainConfig{
			Providers: DomainProviderConfig{},
		}
		sc.Domains[domain] = cfg
	}

	// Set provider
	switch provider {
	case "dnspod":
		cfg.Providers.DNSPod = enabled
	case "adguard":
		cfg.Providers.AdGuard = enabled
	case "cloudflare":
		cfg.Providers.Cloudflare = enabled
	case "openwrt":
		cfg.Providers.OpenWRT = enabled
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}

	return sc.saveUnlocked()
}

// SetProviderGlobal sets a provider enablement for all domains
func (sc *SwitchConfig) SetProviderGlobal(provider string, enabled bool) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for _, cfg := range sc.Domains {
		if cfg == nil {
			continue
		}
		switch provider {
		case "dnspod":
			cfg.Providers.DNSPod = enabled
		case "adguard":
			cfg.Providers.AdGuard = enabled
		case "cloudflare":
			cfg.Providers.Cloudflare = enabled
		case "openwrt":
			cfg.Providers.OpenWRT = enabled
		default:
			return fmt.Errorf("unknown provider: %s", provider)
		}
	}

	return sc.saveUnlocked()
}

// MergeDomains merges new domains into the configuration
// New domains default to all providers disabled
func (sc *SwitchConfig) MergeDomains(domains []string) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for _, domain := range domains {
		if _, exists := sc.Domains[domain]; !exists {
			sc.Domains[domain] = &DomainConfig{
				Providers: DomainProviderConfig{},
			}
		}
	}

	return sc.saveUnlocked()
}

// ShouldSync checks if a domain should be synced to a specific provider
func (sc *SwitchConfig) ShouldSync(domain string, provider string) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	cfg, exists := sc.Domains[domain]
	if !exists || cfg == nil {
		// Domain not in config, default to not syncing (all disabled)
		return false
	}

	switch provider {
	case "dnspod":
		return cfg.Providers.DNSPod
	case "adguard":
		return cfg.Providers.AdGuard
	case "cloudflare":
		return cfg.Providers.Cloudflare
	case "openwrt":
		return cfg.Providers.OpenWRT
	default:
		return false
	}
}
