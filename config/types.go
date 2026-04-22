package config

import (
	"math/rand"
)

const (
	DomainSyncStatePath = "./data/domain_sync_state.json"
	ProvidersPath       = "./data/providers.json"
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
	Traefik             TraefikConfig    `json:"traefik"`
	Providers           []ProviderConfig `json:"providers"`
	PollInterval        int              `json:"poll_interval"`
	TraefikPollInterval int              `json:"traefik_poll_interval"`
	DNSPollInterval     int              `json:"dns_poll_interval"`
	WebEnabled          bool             `json:"web_enabled"`
	WebPort             int              `json:"web_port"`
	LogLevel            string           `json:"log_level"`
}
