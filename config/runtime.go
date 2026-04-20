package config

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/leganck/traefik-domain/dns/model"
	log "github.com/sirupsen/logrus"
)

const (
	SwitchesPath   = "./data/switches.json"
	ProvidersPath  = "./data/providers.json"
)

type ProviderConfig struct {
	ProviderID  string `json:"provider_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	ID          string `json:"id"`
	Secret      string `json:"secret"`
	Host        string `json:"host"`
	RecordValue string `json:"record_value"`
}

func GenerateProviderID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return "p_" + string(b)
}

type TraefikConfig struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type ProvidersData struct {
	Traefik      TraefikConfig    `json:"traefik"`
	Providers    []ProviderConfig `json:"providers"`
	PollInterval int               `json:"poll_interval"`
	WebEnabled   bool              `json:"web_enabled"`
	WebPort      int               `json:"web_port"`
	LogLevel     string            `json:"log_level"`
}

type ProvidersConfig struct {
	path       string
	data       *ProvidersData
	mu         sync.RWMutex
	reloadChan chan struct{}
	watcher    *fsnotify.Watcher
}

func NewProvidersConfig() *ProvidersConfig {
	return &ProvidersConfig{
		path:       ProvidersPath,
		data:       &ProvidersData{Providers: []ProviderConfig{}, PollInterval: 5, WebEnabled: true, WebPort: 8080, LogLevel: "info"},
		reloadChan: make(chan struct{}, 1),
	}
}

func (pc *ProvidersConfig) StartWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	pc.watcher = watcher

	dir := filepath.Dir(pc.path)
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					if event.Name == pc.path {
						log.Info("Providers config file changed, triggering reload")
						pc.notifyReload()
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Errorf("file watcher error: %v", err)
			}
		}
	}()

	log.Infof("Started watching config directory: %s", dir)
	return nil
}

func (pc *ProvidersConfig) StopWatcher() {
	if pc.watcher != nil {
		pc.watcher.Close()
		pc.watcher = nil
	}
}

func (pc *ProvidersConfig) Load() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	data, err := os.ReadFile(pc.path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Infof("Providers config %s does not exist, creating default", pc.path)
			return pc.saveUnlocked()
		}
		return fmt.Errorf("failed to read providers config: %w", err)
	}

	var providersData ProvidersData
	if err := json.Unmarshal(data, &providersData); err != nil {
		backupPath := pc.path + ".backup." + time.Now().Format("20060102-150405")
		log.Warnf("Providers config corrupted, backing up to %s", backupPath)
		os.Rename(pc.path, backupPath)
		return pc.saveUnlocked()
	}

	pc.data = &providersData
	return nil
}

func (pc *ProvidersConfig) saveUnlocked() error {
	dir := filepath.Dir(pc.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create providers config directory: %w", err)
	}

	if pc.data.PollInterval <= 0 {
		pc.data.PollInterval = 5
	}
	if pc.data.WebPort <= 0 {
		pc.data.WebPort = 8080
	}
	if pc.data.LogLevel == "" {
		pc.data.LogLevel = "info"
	}

	data, err := json.MarshalIndent(pc.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal providers config: %w", err)
	}

	return os.WriteFile(pc.path, data, 0644)
}

func (pc *ProvidersConfig) GetReloadChan() chan struct{} {
	return pc.reloadChan
}

func (pc *ProvidersConfig) notifyReload() {
	select {
	case pc.reloadChan <- struct{}{}:
	default:
	}
}

func (pc *ProvidersConfig) GetTraefikConfig() TraefikConfig {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.data.Traefik
}

func (pc *ProvidersConfig) GetPollInterval() int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	if pc.data.PollInterval <= 0 {
		return 5
	}
	return pc.data.PollInterval
}

func (pc *ProvidersConfig) GetWebEnabled() bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.data.WebEnabled
}

func (pc *ProvidersConfig) GetWebPort() int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	if pc.data.WebPort <= 0 {
		return 8080
	}
	return pc.data.WebPort
}

func (pc *ProvidersConfig) GetLogLevel() string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	if pc.data.LogLevel == "" {
		return "info"
	}
	return pc.data.LogLevel
}

func (pc *ProvidersConfig) SetAppConfig(pollInterval int, webEnabled bool, webPort int, logLevel string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.data.PollInterval = pollInterval
	pc.data.WebEnabled = webEnabled
	pc.data.WebPort = webPort
	pc.data.LogLevel = logLevel
	pc.saveUnlocked()
}

func (pc *ProvidersConfig) GetProviders() []ProviderConfig {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	result := make([]ProviderConfig, len(pc.data.Providers))
	copy(result, pc.data.Providers)
	return result
}

func (pc *ProvidersConfig) GetProvider(providerID string) (*ProviderConfig, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	for _, p := range pc.data.Providers {
		if p.ProviderID == providerID {
			pCopy := p
			return &pCopy, true
		}
	}
	return nil, false
}

func (pc *ProvidersConfig) ProviderExists(providerID string) bool {
	_, exists := pc.GetProvider(providerID)
	return exists
}

func (pc *ProvidersConfig) GetProviderIDs() []string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	ids := make([]string, len(pc.data.Providers))
	for i, p := range pc.data.Providers {
		ids[i] = p.ProviderID
	}
	return ids
}

func (pc *ProvidersConfig) SetTraefikConfig(cfg TraefikConfig) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.data.Traefik = cfg
	if err := pc.saveUnlocked(); err != nil {
		return err
	}
	pc.notifyReload()
	return nil
}

func (pc *ProvidersConfig) AddProvider(provider ProviderConfig) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if provider.ProviderID == "" {
		provider.ProviderID = GenerateProviderID()
	}

	for _, p := range pc.data.Providers {
		if p.ProviderID == provider.ProviderID {
			return fmt.Errorf("provider with id '%s' already exists", provider.ProviderID)
		}
	}

	pc.data.Providers = append(pc.data.Providers, provider)
	if err := pc.saveUnlocked(); err != nil {
		return err
	}
	pc.notifyReload()
	return nil
}

func (pc *ProvidersConfig) UpdateProvider(providerID string, updates ProviderConfig) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	for i, p := range pc.data.Providers {
		if p.ProviderID == providerID {
			if updates.Secret == "" {
				updates.Secret = p.Secret
			}
			updates.ProviderID = providerID
			pc.data.Providers[i] = updates
			if err := pc.saveUnlocked(); err != nil {
				return err
			}
			pc.notifyReload()
			return nil
		}
	}
	return fmt.Errorf("provider '%s' not found", providerID)
}

func (pc *ProvidersConfig) DeleteProvider(providerID string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	for i, p := range pc.data.Providers {
		if p.ProviderID == providerID {
			pc.data.Providers = append(pc.data.Providers[:i], pc.data.Providers[i+1:]...)
			if err := pc.saveUnlocked(); err != nil {
				return err
			}
			pc.notifyReload()
			return nil
		}
	}
	return fmt.Errorf("provider '%s' not found", providerID)
}

type RecordInfo struct {
	Value   string `json:"value"`
	Type    string `json:"type"`
	Managed bool   `json:"managed"`
}

type DomainConfig struct {
	Providers map[string]bool           `json:"providers"`
	InTraefik bool                      `json:"inTraefik"`
	Records   map[string]*RecordInfo    `json:"records"`
}

type SwitchConfig struct {
	mu      sync.RWMutex
	Domains map[string]*DomainConfig `json:"domains"`
	path    string
}

func NewSwitchConfig() *SwitchConfig {
	return &SwitchConfig{
		Domains: make(map[string]*DomainConfig),
		path:    SwitchesPath,
	}
}

func (sc *SwitchConfig) Load() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if _, err := os.Stat(sc.path); os.IsNotExist(err) {
		log.Infof("Config file %s does not exist, creating default config", sc.path)
		return sc.saveUnlocked()
	}

	data, err := os.ReadFile(sc.path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config struct {
		Domains map[string]*DomainConfig `json:"domains"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		backupPath := sc.path + ".backup." + time.Now().Format("20060102-150405")
		log.Warnf("Config file is corrupted, backing up to %s", backupPath)
		if err := os.Rename(sc.path, backupPath); err != nil {
			return fmt.Errorf("failed to backup corrupted config: %w", err)
		}
		sc.Domains = make(map[string]*DomainConfig)
		return sc.saveUnlocked()
	}

	sc.Domains = make(map[string]*DomainConfig)
	for domain, cfg := range config.Domains {
		if cfg == nil {
			cfg = &DomainConfig{
				Providers: make(map[string]bool),
			}
		}
		if cfg.Providers == nil {
			cfg.Providers = make(map[string]bool)
		}
		if cfg.Records == nil {
			cfg.Records = make(map[string]*RecordInfo)
		}
		sc.Domains[domain] = cfg
	}

	return nil
}

func (sc *SwitchConfig) Save() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.saveUnlocked()
}

func (sc *SwitchConfig) saveUnlocked() error {
	dir := filepath.Dir(sc.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(sc.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (sc *SwitchConfig) GetConfig() map[string]*DomainConfig {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	configCopy := make(map[string]*DomainConfig)
	for domain, cfg := range sc.Domains {
		if cfg != nil {
			providersCopy := make(map[string]bool)
			for k, v := range cfg.Providers {
				providersCopy[k] = v
			}
			recordsCopy := make(map[string]*RecordInfo)
			for k, v := range cfg.Records {
				if v != nil {
					vCopy := *v
					recordsCopy[k] = &vCopy
				}
			}
			configCopy[domain] = &DomainConfig{
				Providers: providersCopy,
				InTraefik: cfg.InTraefik,
				Records:   recordsCopy,
			}
		}
	}
	return configCopy
}

func (sc *SwitchConfig) GetDomain(domain string) *DomainConfig {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if cfg, exists := sc.Domains[domain]; exists && cfg != nil {
		providersCopy := make(map[string]bool)
		for k, v := range cfg.Providers {
			providersCopy[k] = v
		}
		recordsCopy := make(map[string]*RecordInfo)
		for k, v := range cfg.Records {
			if v != nil {
				vCopy := *v
				recordsCopy[k] = &vCopy
			}
		}
		return &DomainConfig{
			Providers: providersCopy,
			InTraefik: cfg.InTraefik,
			Records:   recordsCopy,
		}
	}

	return &DomainConfig{
		Providers: make(map[string]bool),
	}
}

func (sc *SwitchConfig) SetDomainProvider(domain string, provider string, enabled bool) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	cfg, exists := sc.Domains[domain]
	if !exists || cfg == nil {
		cfg = &DomainConfig{
			Providers: make(map[string]bool),
			Records:   make(map[string]*RecordInfo),
		}
		sc.Domains[domain] = cfg
	}
	if cfg.Records == nil {
		cfg.Records = make(map[string]*RecordInfo)
	}

	cfg.Providers[provider] = enabled
	return sc.saveUnlocked()
}

func (sc *SwitchConfig) SetProviderGlobal(provider string, enabled bool) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for _, cfg := range sc.Domains {
		if cfg == nil {
			continue
		}
		cfg.Providers[provider] = enabled
	}

	return sc.saveUnlocked()
}

func (sc *SwitchConfig) MergeDomains(domains []string, providerNames []string) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for _, cfg := range sc.Domains {
		if cfg != nil {
			cfg.InTraefik = false
		}
	}

	for _, domain := range domains {
		if _, exists := sc.Domains[domain]; !exists {
			providers := make(map[string]bool)
			for _, name := range providerNames {
				providers[name] = false
			}
			sc.Domains[domain] = &DomainConfig{
				Providers: providers,
				InTraefik: true,
			}
		} else {
			cfg := sc.Domains[domain]
			if cfg.Providers == nil {
				cfg.Providers = make(map[string]bool)
			}
			cfg.InTraefik = true
			for _, name := range providerNames {
				if _, exists := cfg.Providers[name]; !exists {
					cfg.Providers[name] = false
				}
			}
		}
	}

	return sc.saveUnlocked()
}

func (sc *SwitchConfig) ShouldSync(domain string, provider string) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	cfg, exists := sc.Domains[domain]
	if !exists || cfg == nil {
		return false
	}

	return cfg.Providers[provider]
}

func (sc *SwitchConfig) GetEnabledDomains(provider string) []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var result []string
	for domain, cfg := range sc.Domains {
		if cfg != nil && cfg.Providers[provider] {
			result = append(result, domain)
		}
	}
	return result
}

func (sc *SwitchConfig) GetAllMainDomains() []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	seen := make(map[string]bool)
	var result []string
	for domain := range sc.Domains {
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

func (sc *SwitchConfig) DeleteDomain(domain string) (map[string]bool, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	cfg, exists := sc.Domains[domain]
	if !exists {
		return nil, fmt.Errorf("domain %s not found", domain)
	}

	providers := make(map[string]bool)
	if cfg != nil {
		for k, v := range cfg.Providers {
			providers[k] = v
		}
	}

	delete(sc.Domains, domain)
	return providers, sc.saveUnlocked()
}

func (sc *SwitchConfig) RemoveProvider(providerName string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for _, cfg := range sc.Domains {
		if cfg != nil {
			if cfg.Providers != nil {
				delete(cfg.Providers, providerName)
			}
			if cfg.Records != nil {
				delete(cfg.Records, providerName)
			}
		}
	}

	sc.saveUnlocked()
}

func (sc *SwitchConfig) UpdateRecords(providerName string, records map[string]*RecordInfo) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for domain, cfg := range sc.Domains {
		if cfg == nil {
			continue
		}
		if cfg.Records == nil {
			cfg.Records = make(map[string]*RecordInfo)
		}
		if info, ok := records[domain]; ok {
			cfg.Records[providerName] = info
		} else {
			delete(cfg.Records, providerName)
		}
	}
}
