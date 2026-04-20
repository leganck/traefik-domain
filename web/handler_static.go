package web

import (
	"embed"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

//go:embed static/*
var staticFS embed.FS

func handleIndex(w http.ResponseWriter, r *http.Request) {
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

func handleStatic(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/static/")
	if strings.Contains(path, "..") || strings.Contains(path, "//") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

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

func StartServer(port int, handler *Handler) (*http.Server, error) {
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", port),
		Handler: mux,
	}

	go func() {
		log.Infof("Starting web server on port %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("Web server error: %v", err)
		}
	}()

	return server, nil
}
