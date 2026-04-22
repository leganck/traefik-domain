package web

import (
	"net/http"

	"github.com/leganck/traefik-domain/config"
	"github.com/leganck/traefik-domain/internal/state"
)

type DomainUpdateRequest struct {
	Domain            string
	ProviderID        string
	Enabled           bool
	OverwriteExisting bool
}

type ApplyDomainUpdatesFunc func(requests []DomainUpdateRequest) error

type DomainStateStore interface {
	GetPreferences() map[string]*state.DomainPreference
	GetDiscovery() map[string]*state.DomainDiscovery
	GetRecords() map[string]*state.DomainRecordCache
	SetDomainProvider(domain string, provider string, enabled bool, overwrite bool) error
	SetProviderGlobal(provider string, enabled bool) error
	GetProviderGlobals() map[string]bool
	DeleteDomain(domain string) (map[string]bool, error)
	RemoveProvider(providerName string)
}

type ProviderStore interface {
	GetProviders() []config.ProviderConfig
	GetProvider(providerID string) (*config.ProviderConfig, bool)
	GetTraefikConfig() config.TraefikConfig
	SetTraefikConfig(cfg config.TraefikConfig) error
	AddProvider(provider config.ProviderConfig) error
	UpdateProvider(providerID string, updates config.ProviderConfig) error
	DeleteProvider(providerID string) error
	FindDuplicateBackendWarning(provider config.ProviderConfig, excludeProviderID string) string
}

type Handler struct {
	stateStore         DomainStateStore
	providerStore      ProviderStore
	applyDomainUpdates ApplyDomainUpdatesFunc
}

func NewHandler(stateStore DomainStateStore, providerStore ProviderStore) *Handler {
	return &Handler{
		stateStore:    stateStore,
		providerStore: providerStore,
	}
}

func (h *Handler) SetApplyDomainUpdatesFunc(fn ApplyDomainUpdatesFunc) {
	h.applyDomainUpdates = fn
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/domains", h.handleGetDomains)
	mux.HandleFunc("/api/domains/{domain}", h.handleDomainDetail)
	mux.HandleFunc("/api/toggle/domain", h.handleToggleDomain)
	mux.HandleFunc("/api/toggle/provider", h.handleToggleProvider)
	mux.HandleFunc("/api/config", h.handleGetConfig)
	mux.HandleFunc("/api/config/traefik", h.handleUpdateTraefikConfig)
	mux.HandleFunc("/api/providers", h.handleProviders)
	mux.HandleFunc("/api/providers/", h.handleProviderDetail)
	mux.HandleFunc("/static/", handleStatic)
	mux.HandleFunc("/", handleIndex)
}
