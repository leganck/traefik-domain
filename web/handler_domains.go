package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/leganck/traefik-domain/internal/state"
	log "github.com/sirupsen/logrus"
)

type ProviderInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DomainEntry struct {
	Providers map[string]bool              `json:"providers"`
	Records   map[string]*state.RecordInfo `json:"records"`
	InTraefik bool                         `json:"inTraefik"`
}

type DomainResponse struct {
	Domains         map[string]*DomainEntry `json:"domains"`
	Providers       []ProviderInfo          `json:"providers"`
	ProviderGlobals map[string]bool         `json:"providerGlobals"`
}

func (h *Handler) handleGetDomains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	preferences := h.stateStore.GetPreferences()
	discovery := h.stateStore.GetDiscovery()
	records := h.stateStore.GetRecords()
	providers := h.providerStore.GetProviders()

	providerInfos := make([]ProviderInfo, len(providers))
	providerIDs := make([]string, len(providers))
	for i, p := range providers {
		providerInfos[i] = ProviderInfo{ID: p.ProviderID, Name: p.Name}
		providerIDs[i] = p.ProviderID
	}

	domainEntries := make(map[string]*DomainEntry)
	for domainName := range preferences {
		entry := &DomainEntry{
			Providers: h.stateStore.GetEffectiveProviderState(domainName, providerIDs),
			Records:   map[string]*state.RecordInfo{},
		}
		if disc := discovery[domainName]; disc != nil {
			entry.InTraefik = disc.InTraefik
		}
		if cache := records[domainName]; cache != nil {
			entry.Records = cache.Records
		}
		if entry.Records == nil {
			entry.Records = make(map[string]*state.RecordInfo)
		}
		domainEntries[domainName] = entry
	}

	response := DomainResponse{
		Domains:         domainEntries,
		Providers:       providerInfos,
		ProviderGlobals: h.stateStore.GetProviderGlobals(),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("Failed to encode domains response: %v", err)
	}
}

type ToggleRequest struct {
	Domain            string `json:"domain"`
	ProviderID        string `json:"providerId"`
	Enabled           bool   `json:"enabled"`
	OverwriteExisting bool   `json:"overwriteExisting"`
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

	if err := h.stateStore.SetDomainProvider(req.Domain, req.ProviderID, req.Enabled, req.OverwriteExisting); err != nil {
		log.Errorf("Failed to set domain provider %s for domain %s: %v", req.ProviderID, req.Domain, err)
		respondWithJSON(w, http.StatusInternalServerError, ToggleResponse{
			Success: false,
			Message: "Failed to update configuration",
		})
		return
	}

	if h.applyDomainUpdates != nil {
		if err := h.applyDomainUpdates([]DomainUpdateRequest{{
			Domain:            req.Domain,
			ProviderID:        req.ProviderID,
			Enabled:           req.Enabled,
			OverwriteExisting: req.OverwriteExisting,
		}}); err != nil {
			log.Errorf("Failed to apply DNS update for domain %s on provider %s: %v", req.Domain, req.ProviderID, err)
			respondWithJSON(w, http.StatusInternalServerError, ToggleResponse{
				Success: false,
				Message: "配置已更新，但应用 DNS 更新失败",
			})
			return
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

	if err := h.stateStore.SetProviderGlobal(req.ProviderID, req.Enabled); err != nil {
		log.Errorf("Failed to set provider %s global toggle to %v: %v", req.ProviderID, req.Enabled, err)
		respondWithJSON(w, http.StatusInternalServerError, ToggleResponse{
			Success: false,
			Message: "Failed to update configuration",
		})
		return
	}

	if h.applyDomainUpdates != nil {
		domains := h.stateStore.GetPreferences()
		requests := make([]DomainUpdateRequest, 0, len(domains))
		for domainName := range domains {
			requests = append(requests, DomainUpdateRequest{
				Domain:            domainName,
				ProviderID:        req.ProviderID,
				Enabled:           req.Enabled,
				OverwriteExisting: req.OverwriteExisting,
			})
		}
		if err := h.applyDomainUpdates(requests); err != nil {
			log.Errorf("Failed to apply DNS updates for provider %s: %v", req.ProviderID, err)
			respondWithJSON(w, http.StatusInternalServerError, ToggleResponse{
				Success: false,
				Message: "配置已更新，但应用 DNS 更新失败",
			})
			return
		}
	}

	log.Infof("Updated provider %s global toggle to enabled=%v", req.ProviderID, req.Enabled)
	respondWithJSON(w, http.StatusOK, ToggleResponse{Success: true})
}

func (h *Handler) isValidProvider(providerID string) bool {
	providers := h.providerStore.GetProviders()
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
	if disc := h.stateStore.GetDiscovery()[domain]; disc != nil && disc.InTraefik {
		respondWithJSON(w, http.StatusBadRequest, ToggleResponse{
			Success: false,
			Message: "只能删除不在 Traefik 中的域名",
		})
		return
	}

	providers, err := h.stateStore.DeleteDomain(domain)
	if err != nil {
		log.Errorf("Failed to delete domain %s from config: %v", domain, err)
		respondWithJSON(w, http.StatusInternalServerError, ToggleResponse{
			Success: false,
			Message: "删除域名失败",
		})
		return
	}

	if h.applyDomainUpdates != nil {
		requests := make([]DomainUpdateRequest, 0, len(providers))
		for provider, enabled := range providers {
			if enabled {
				requests = append(requests, DomainUpdateRequest{Domain: domain, ProviderID: provider, Enabled: false})
			}
		}
		if err := h.applyDomainUpdates(requests); err != nil {
			log.Errorf("Failed to apply DNS cleanup for %s: %v", domain, err)
			respondWithJSON(w, http.StatusInternalServerError, ToggleResponse{
				Success: false,
				Message: "域名已删除，但应用 DNS 清理失败",
			})
			return
		}
	}

	log.Infof("Deleted domain %s from config and providers", domain)
	respondWithJSON(w, http.StatusOK, ToggleResponse{Success: true})
}
