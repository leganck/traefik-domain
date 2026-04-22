package config

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (pc *ProvidersConfig) LoadFromSources() error {
	exists, err := pc.fileExists()
	if err != nil {
		return err
	}
	if err := pc.Load(); err != nil {
		return err
	}
	if !exists {
		if err := pc.initFromEnvVars(); err != nil {
			return err
		}
	}
	pc.ApplyEnvVars()
	return nil
}

func (pc *ProvidersConfig) ReloadFromSources() error {
	if err := pc.Load(); err != nil {
		return err
	}
	pc.ApplyEnvVars()
	return nil
}

func (pc *ProvidersConfig) fileExists() (bool, error) {
	_, err := os.Stat(pc.path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to stat providers config: %w", err)
}

func (pc *ProvidersConfig) ApplyEnvVars() {
	currentPollInterval := pc.GetPollInterval()
	currentTraefikPollInterval := pc.GetTraefikPollInterval()
	currentDNSPollInterval := pc.GetDNSPollInterval()
	currentWebEnabled := pc.GetWebEnabled()
	currentWebPort := pc.GetWebPort()
	currentLogLevel := pc.GetLogLevel()

	pollInterval := envInt("POLL_INTERVAL", currentPollInterval, 5)
	traefikPollInterval := envInt("TRAEFIK_POLL_INTERVAL", currentTraefikPollInterval, 30)
	dnsPollInterval := envInt("DNS_POLL_INTERVAL", currentDNSPollInterval, 300)
	webEnabled := envBool("WEB_ENABLED", currentWebEnabled)
	webPort := envInt("WEB_PORT", currentWebPort, 8080)
	logLevel := envString("LOG_LEVEL", currentLogLevel)

	pc.mu.Lock()
	pc.setAppConfigUnlocked(pollInterval, traefikPollInterval, dnsPollInterval, webEnabled, webPort, logLevel)
	pc.mu.Unlock()

	if pollInterval != currentPollInterval || traefikPollInterval != currentTraefikPollInterval || dnsPollInterval != currentDNSPollInterval || webEnabled != currentWebEnabled || webPort != currentWebPort || logLevel != currentLogLevel {
		log.Infof("Applied config from env vars: poll_interval=%d, traefik_poll_interval=%d, dns_poll_interval=%d, web_enabled=%v, web_port=%d, log_level=%s",
			pollInterval, traefikPollInterval, dnsPollInterval, webEnabled, webPort, logLevel)
	}
}

func (pc *ProvidersConfig) initFromEnvVars() error {
	traefikHost := os.Getenv("TRAEFIK_HOST")
	if traefikHost != "" {
		if err := pc.SetTraefikConfig(TraefikConfig{
			Host:     traefikHost,
			Username: os.Getenv("TRAEFIK_USERNAME"),
			Password: os.Getenv("TRAEFIK_PASSWORD"),
		}); err != nil {
			return err
		}
		log.Info("Initialized Traefik config from env vars")
	}

	dnsName := os.Getenv("DNS_NAME")
	dnsID := os.Getenv("DNS_ID")
	dnsSecret := os.Getenv("DNS_SECRET")
	if dnsName == "" || dnsID == "" || dnsSecret == "" {
		return nil
	}

	providerCfg := ProviderConfig{
		ProviderID:  GenerateProviderID(),
		Name:        dnsName,
		Type:        dnsName,
		ID:          dnsID,
		Secret:      dnsSecret,
		RecordValue: os.Getenv("DNS_RECORD_VALUE"),
	}
	switch strings.ToLower(dnsName) {
	case "adguard":
		providerCfg.Host = os.Getenv("ADGUARD_HOST")
	case "openwrt":
		providerCfg.Host = os.Getenv("OPENWRT_HOST")
	}

	if err := pc.AddProvider(providerCfg); err != nil {
		return err
	}
	log.Infof("Initialized provider %s from env vars", dnsName)
	return nil
}

func envInt(name string, current int, fallback int) int {
	value := current
	if v := os.Getenv(name); v != "" {
		if n, _ := fmt.Sscanf(v, "%d", &value); n != 1 || value <= 0 {
			value = fallback
		}
	}
	return value
}

func envBool(name string, current bool) bool {
	if v := os.Getenv(name); v != "" {
		return strings.ToLower(v) == "true"
	}
	return current
}

func envString(name string, current string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return current
}
