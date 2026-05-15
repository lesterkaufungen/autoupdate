package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed frontend/dist/*
var uiFS embed.FS

func (s *Server) UIHandler() http.Handler {
	distFS, err := fs.Sub(uiFS, "frontend/dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "UI not found", http.StatusNotFound)
		})
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// If path starts with /ui/, strip it for the file server
		if strings.HasPrefix(path, "/ui/") {
			r.URL.Path = strings.TrimPrefix(path, "/ui/")
		} else if path == "/ui" {
			http.Redirect(w, r, "/ui/", http.StatusMovedPermanently)
			return
		}

		// Re-evaluate path after stripping
		path = r.URL.Path

		// Serve index.html for SPA routes or if directory is requested
		if path == "" || path == "/" || !strings.Contains(path, ".") {
			r.URL.Path = "/"
		}

		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) serveUI(mux *http.ServeMux) {
	if !s.config.UI {
		return
	}
	mux.Handle("/ui/", s.UIHandler())
}
