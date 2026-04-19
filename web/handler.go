package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

//go:embed static/*
var staticFS embed.FS

// Handler handles HTTP requests for the web UI
type Handler struct {
	switchConfig *SwitchConfig
	providers    []string
}

// NewHandler creates a new HTTP handler instance
func NewHandler(switchConfig *SwitchConfig, providers []string) *Handler {
	return &Handler{
		switchConfig: switchConfig,
		providers:    providers,
	}
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// API routes
	mux.HandleFunc("/api/domains", h.handleGetDomains)
	mux.HandleFunc("/api/toggle/domain", h.handleToggleDomain)
	mux.HandleFunc("/api/toggle/provider", h.handleToggleProvider)

	// Static files
	mux.HandleFunc("/static/", h.handleStatic)

	// Index page (catch-all)
	mux.HandleFunc("/", h.handleIndex)
}

// handleIndex serves the index.html from embedded files
func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		log.Errorf("Failed to read index.html: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleStatic serves CSS/JS files with correct Content-Type
func (h *Handler) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Security: prevent directory traversal
	path := strings.TrimPrefix(r.URL.Path, "/static/")
	if strings.Contains(path, "..") || strings.Contains(path, "//") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Clean the path
	cleanPath := filepath.Clean(path)
	if cleanPath == "." || cleanPath == "/" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	fullPath := "static/" + cleanPath
	data, err := staticFS.ReadFile(fullPath)
	if err != nil {
		log.Errorf("Failed to read static file %s: %v", fullPath, err)
		http.NotFound(w, r)
		return
	}

	// Set correct Content-Type based on file extension
	ext := strings.ToLower(filepath.Ext(cleanPath))
	switch ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// DomainResponse represents the response for GET /api/domains
type DomainResponse struct {
	Domains   map[string]*DomainConfig `json:"domains"`
	Providers []string                 `json:"providers"`
}

// handleGetDomains returns domains and providers config
func (h *Handler) handleGetDomains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	domains := h.switchConfig.GetConfig()

	response := DomainResponse{
		Domains:   domains,
		Providers: h.providers,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("Failed to encode domains response: %v", err)
	}
}

// ToggleDomainRequest represents the request for POST /api/toggle/domain
type ToggleDomainRequest struct {
	Domain   string `json:"domain"`
	Provider string `json:"provider"`
	Enabled  bool   `json:"enabled"`
}

// ToggleDomainResponse represents the response for toggle domain endpoint
type ToggleDomainResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// handleToggleDomain updates domain-provider toggle
func (h *Handler) handleToggleDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ToggleDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Errorf("Failed to decode toggle domain request: %v", err)
		respondWithJSON(w, http.StatusBadRequest, ToggleDomainResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Validate domain
	if req.Domain == "" {
		respondWithJSON(w, http.StatusBadRequest, ToggleDomainResponse{
			Success: false,
			Message: "Domain is required",
		})
		return
	}

	// Validate provider
	if !h.isValidProvider(req.Provider) {
		respondWithJSON(w, http.StatusBadRequest, ToggleDomainResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid provider: %s", req.Provider),
		})
		return
	}

	// Update configuration
	if err := h.switchConfig.SetDomainProvider(req.Domain, req.Provider, req.Enabled); err != nil {
		log.Errorf("Failed to set domain provider %s for domain %s: %v", req.Provider, req.Domain, err)
		respondWithJSON(w, http.StatusInternalServerError, ToggleDomainResponse{
			Success: false,
			Message: "Failed to update configuration",
		})
		return
	}

	log.Infof("Updated domain %s provider %s to enabled=%v", req.Domain, req.Provider, req.Enabled)
	respondWithJSON(w, http.StatusOK, ToggleDomainResponse{
		Success: true,
	})
}

// ToggleProviderRequest represents the request for POST /api/toggle/provider
type ToggleProviderRequest struct {
	Provider string `json:"provider"`
	Enabled  bool   `json:"enabled"`
}

// ToggleProviderResponse represents the response for toggle provider endpoint
type ToggleProviderResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// handleToggleProvider updates provider global toggle
func (h *Handler) handleToggleProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ToggleProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Errorf("Failed to decode toggle provider request: %v", err)
		respondWithJSON(w, http.StatusBadRequest, ToggleProviderResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Validate provider
	if !h.isValidProvider(req.Provider) {
		respondWithJSON(w, http.StatusBadRequest, ToggleProviderResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid provider: %s", req.Provider),
		})
		return
	}

	// Update configuration
	if err := h.switchConfig.SetProviderGlobal(req.Provider, req.Enabled); err != nil {
		log.Errorf("Failed to set provider %s global toggle to %v: %v", req.Provider, req.Enabled, err)
		respondWithJSON(w, http.StatusInternalServerError, ToggleProviderResponse{
			Success: false,
			Message: "Failed to update configuration",
		})
		return
	}

	log.Infof("Updated provider %s global toggle to enabled=%v", req.Provider, req.Enabled)
	respondWithJSON(w, http.StatusOK, ToggleProviderResponse{
		Success: true,
	})
}

// isValidProvider checks if a provider name is valid
func (h *Handler) isValidProvider(provider string) bool {
	for _, p := range h.providers {
		if p == provider {
			return true
		}
	}
	return false
}

// respondWithJSON sends a JSON response with the given status code
func respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Errorf("Failed to encode JSON response: %v", err)
	}
}

// StartServer starts the HTTP server in a background goroutine
func StartServer(port int, handler *Handler) (*http.Server, error) {
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// Start server in background goroutine
	go func() {
		log.Infof("Starting web server on port %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("Web server error: %v", err)
		}
	}()

	return server, nil
}
