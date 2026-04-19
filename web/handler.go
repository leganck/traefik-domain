package web

import (
	"net/http"

	"github.com/leganck/traefik-domain/config"
)

type DeleteDomainFunc func(domain, provider string) error

type Handler struct {
	switchConfig    *config.SwitchConfig
	providersConfig *config.ProvidersConfig
	deleteDomain    DeleteDomainFunc
}

func NewHandler(switchConfig *config.SwitchConfig, providersConfig *config.ProvidersConfig) *Handler {
	return &Handler{
		switchConfig:    switchConfig,
		providersConfig: providersConfig,
	}
}

func (h *Handler) SetDeleteDomainFunc(fn DeleteDomainFunc) {
	h.deleteDomain = fn
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
