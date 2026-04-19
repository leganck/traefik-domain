package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/leganck/traefik-domain/config"
	log "github.com/sirupsen/logrus"
)

func (h *Handler) handleProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getProviders(w, r)
	case http.MethodPost:
		h.createProvider(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) getProviders(w http.ResponseWriter, r *http.Request) {
	providers := h.providersConfig.GetProviders()
	for i := range providers {
		providers[i].Secret = maskSecret(providers[i].Secret)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"providers": providers})
}

func (h *Handler) createProvider(w http.ResponseWriter, r *http.Request) {
	var provider config.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&provider); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if provider.Name == "" || provider.Type == "" || provider.Secret == "" {
		http.Error(w, "name, type and secret are required", http.StatusBadRequest)
		return
	}

	provider.ProviderID = config.GenerateProviderID()

	if err := h.providersConfig.AddProvider(provider); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

func (h *Handler) handleProviderDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/providers/")
	if id == "" {
		http.Error(w, "Provider ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.updateProvider(w, r, id)
	case http.MethodDelete:
		h.deleteProvider(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) updateProvider(w http.ResponseWriter, r *http.Request, providerID string) {
	var updates config.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.providersConfig.UpdateProvider(providerID, updates); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *Handler) deleteProvider(w http.ResponseWriter, r *http.Request, providerID string) {
	if h.switchConfig != nil {
		h.switchConfig.RemoveProvider(providerID)
	}

	if err := h.providersConfig.DeleteProvider(providerID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func maskSecret(secret string) string {
	if len(secret) <= 4 {
		return "****"
	}
	return "****" + secret[len(secret)-4:]
}

func respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Errorf("Failed to encode JSON response: %v", err)
	}
}
