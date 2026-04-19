package web

import (
	"encoding/json"
	"net/http"
)

type TraefikConfigRequest struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tfCfg := h.providersConfig.GetTraefikConfig()
	providers := h.providersConfig.GetProviders()

	for i := range providers {
		providers[i].Secret = maskSecret(providers[i].Secret)
	}

	response := map[string]interface{}{
		"traefik": map[string]string{
			"host":     tfCfg.Host,
			"username": tfCfg.Username,
			"password": maskSecret(tfCfg.Password),
		},
		"providers": providers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleUpdateTraefikConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TraefikConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	existing := h.providersConfig.GetTraefikConfig()
	if req.Host != "" {
		existing.Host = req.Host
	}
	if req.Username != "" {
		existing.Username = req.Username
	}
	if req.Password != "" {
		existing.Password = req.Password
	}

	if err := h.providersConfig.SetTraefikConfig(existing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
