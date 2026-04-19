package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/leganck/traefik-domain/config"
	log "github.com/sirupsen/logrus"
)

type ProviderInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DomainEntry struct {
	Providers map[string]bool               `json:"providers"`
	Records   map[string]*config.RecordInfo `json:"records"`
	InTraefik bool                          `json:"inTraefik"`
}

type DomainResponse struct {
	Domains   map[string]*DomainEntry `json:"domains"`
	Providers []ProviderInfo           `json:"providers"`
}

func (h *Handler) handleGetDomains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	domains := h.switchConfig.GetConfig()
	providers := h.providersConfig.GetProviders()

	providerInfos := make([]ProviderInfo, len(providers))
	for i, p := range providers {
		providerInfos[i] = ProviderInfo{ID: p.ProviderID, Name: p.Name}
	}

	domainEntries := make(map[string]*DomainEntry)
	for domainName, cfg := range domains {
		entry := &DomainEntry{
			Providers: cfg.Providers,
			Records:   cfg.Records,
			InTraefik: cfg.InTraefik,
		}
		if entry.Records == nil {
			entry.Records = make(map[string]*config.RecordInfo)
		}
		domainEntries[domainName] = entry
	}

	response := DomainResponse{
		Domains:   domainEntries,
		Providers: providerInfos,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("Failed to encode domains response: %v", err)
	}
}

type ToggleRequest struct {
	Domain     string `json:"domain"`
	ProviderID string `json:"providerId"`
	Enabled    bool   `json:"enabled"`
}

type ToggleResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func (h *Handler) handleToggleDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ToggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Errorf("Failed to decode toggle domain request: %v", err)
		respondWithJSON(w, http.StatusBadRequest, ToggleResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if req.Domain == "" {
		respondWithJSON(w, http.StatusBadRequest, ToggleResponse{
			Success: false,
			Message: "Domain is required",
		})
		return
	}

	if !h.isValidProvider(req.ProviderID) {
		respondWithJSON(w, http.StatusBadRequest, ToggleResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid provider: %s", req.ProviderID),
		})
		return
	}

	if err := h.switchConfig.SetDomainProvider(req.Domain, req.ProviderID, req.Enabled); err != nil {
		log.Errorf("Failed to set domain provider %s for domain %s: %v", req.ProviderID, req.Domain, err)
		respondWithJSON(w, http.StatusInternalServerError, ToggleResponse{
			Success: false,
			Message: "Failed to update configuration",
		})
		return
	}

	if !req.Enabled && h.deleteDomain != nil {
		if err := h.deleteDomain(req.Domain, req.ProviderID); err != nil {
			log.Warnf("Failed to delete domain %s from provider %s: %v", req.Domain, req.ProviderID, err)
		}
	}

	log.Infof("Updated domain %s provider %s to enabled=%v", req.Domain, req.ProviderID, req.Enabled)
	respondWithJSON(w, http.StatusOK, ToggleResponse{Success: true})
}

func (h *Handler) handleToggleProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ToggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Errorf("Failed to decode toggle provider request: %v", err)
		respondWithJSON(w, http.StatusBadRequest, ToggleResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if !h.isValidProvider(req.ProviderID) {
		respondWithJSON(w, http.StatusBadRequest, ToggleResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid provider: %s", req.ProviderID),
		})
		return
	}

	if err := h.switchConfig.SetProviderGlobal(req.ProviderID, req.Enabled); err != nil {
		log.Errorf("Failed to set provider %s global toggle to %v: %v", req.ProviderID, req.Enabled, err)
		respondWithJSON(w, http.StatusInternalServerError, ToggleResponse{
			Success: false,
			Message: "Failed to update configuration",
		})
		return
	}

	if !req.Enabled && h.deleteDomain != nil {
		domains := h.switchConfig.GetConfig()
		for domainName := range domains {
			if err := h.deleteDomain(domainName, req.ProviderID); err != nil {
				log.Warnf("Failed to delete domain %s from provider %s: %v", domainName, req.ProviderID, err)
			}
		}
	}

	log.Infof("Updated provider %s global toggle to enabled=%v", req.ProviderID, req.Enabled)
	respondWithJSON(w, http.StatusOK, ToggleResponse{Success: true})
}

func (h *Handler) isValidProvider(providerID string) bool {
	providers := h.providersConfig.GetProviders()
	for _, p := range providers {
		if p.ProviderID == providerID {
			return true
		}
	}
	return false
}

func (h *Handler) handleDomainDetail(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	if domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		h.handleDeleteDomain(w, r, domain)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleDeleteDomain(w http.ResponseWriter, r *http.Request, domain string) {
	cfg := h.switchConfig.GetDomain(domain)
	if cfg.InTraefik {
		respondWithJSON(w, http.StatusBadRequest, ToggleResponse{
			Success: false,
			Message: "只能删除不在 Traefik 中的域名",
		})
		return
	}

	providers, err := h.switchConfig.DeleteDomain(domain)
	if err != nil {
		log.Errorf("Failed to delete domain %s from config: %v", domain, err)
		respondWithJSON(w, http.StatusInternalServerError, ToggleResponse{
			Success: false,
			Message: "删除域名失败",
		})
		return
	}

	if h.deleteDomain != nil {
		for provider, enabled := range providers {
			if enabled {
				if err := h.deleteDomain(domain, provider); err != nil {
					log.Warnf("Failed to delete domain %s from provider %s: %v", domain, provider, err)
				}
			}
		}
	}

	log.Infof("Deleted domain %s from config and providers", domain)
	respondWithJSON(w, http.StatusOK, ToggleResponse{Success: true})
}
