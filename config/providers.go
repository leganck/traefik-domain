package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

type ProvidersConfig struct {
	path     string
	data     *ProvidersData
	mu       sync.RWMutex
	reloadCh chan struct{}
	watcher  *fsnotify.Watcher
}

func NewProvidersConfig() *ProvidersConfig {
	return &ProvidersConfig{
		path:     ProvidersPath,
		data:     &ProvidersData{Providers: []ProviderConfig{}, PollInterval: 5, TraefikPollInterval: 30, DNSPollInterval: 300, WebEnabled: true, WebPort: 8080, LogLevel: "info"},
		reloadCh: make(chan struct{}, 1),
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
	if pc.data.TraefikPollInterval <= 0 {
		pc.data.TraefikPollInterval = 30
	}
	if pc.data.DNSPollInterval <= 0 {
		pc.data.DNSPollInterval = 300
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
	return pc.reloadCh
}

func (pc *ProvidersConfig) notifyReload() {
	select {
	case pc.reloadCh <- struct{}{}:
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

func (pc *ProvidersConfig) GetTraefikPollInterval() int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	if pc.data.TraefikPollInterval <= 0 {
		return 30
	}
	return pc.data.TraefikPollInterval
}

func (pc *ProvidersConfig) GetDNSPollInterval() int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	if pc.data.DNSPollInterval <= 0 {
		return 300
	}
	return pc.data.DNSPollInterval
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

func (pc *ProvidersConfig) setAppConfigUnlocked(pollInterval int, traefikPollInterval int, dnsPollInterval int, webEnabled bool, webPort int, logLevel string) {
	pc.data.PollInterval = pollInterval
	pc.data.TraefikPollInterval = traefikPollInterval
	pc.data.DNSPollInterval = dnsPollInterval
	pc.data.WebEnabled = webEnabled
	pc.data.WebPort = webPort
	pc.data.LogLevel = logLevel
}

func (pc *ProvidersConfig) SetAppConfig(pollInterval int, traefikPollInterval int, dnsPollInterval int, webEnabled bool, webPort int, logLevel string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.setAppConfigUnlocked(pollInterval, traefikPollInterval, dnsPollInterval, webEnabled, webPort, logLevel)
	return pc.saveUnlocked()
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
			merged := p
			if updates.Name != "" {
				merged.Name = updates.Name
			}
			if updates.Type != "" {
				merged.Type = updates.Type
			}
			if updates.ID != "" {
				merged.ID = updates.ID
			}
			if updates.Secret != "" {
				merged.Secret = updates.Secret
			}
			if updates.Host != "" {
				merged.Host = updates.Host
			}
			if updates.RecordValue != "" {
				merged.RecordValue = updates.RecordValue
			}
			pc.data.Providers[i] = merged
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

func (pc *ProvidersConfig) FindDuplicateBackendWarning(provider ProviderConfig, excludeProviderID string) string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	candidateType := strings.ToLower(strings.TrimSpace(provider.Type))
	candidateHost := normalizeProviderHost(provider.Host)
	candidateID := strings.TrimSpace(provider.ID)
	candidateSecret := strings.TrimSpace(provider.Secret)

	for _, existing := range pc.data.Providers {
		if existing.ProviderID == excludeProviderID {
			continue
		}
		if strings.ToLower(strings.TrimSpace(existing.Type)) != candidateType {
			continue
		}

		sameHost := candidateHost != "" && normalizeProviderHost(existing.Host) == candidateHost
		sameID := candidateID != "" && strings.TrimSpace(existing.ID) == candidateID
		sameSecret := candidateSecret != "" && strings.TrimSpace(existing.Secret) == candidateSecret

		if sameHost || sameID || sameSecret {
			return fmt.Sprintf("检测到提供商 %q 可能指向同一个 DNS 后端，多个实例会共享或覆盖同一批记录，请确认是否为预期配置", existing.Name)
		}
	}

	return ""
}

func normalizeProviderHost(host string) string {
	return strings.TrimRight(strings.ToLower(strings.TrimSpace(host)), "/")
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
		pc.watcher = nil
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
