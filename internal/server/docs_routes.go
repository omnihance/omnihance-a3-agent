package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/omnihance/omnihance-a3-agent/internal/mw"
)

func (s *Server) InitializeDocsRoutes(r *chi.Mux) {
	r.Route("/docs", func(r chi.Router) {
		r.Use(mw.RequireLocalIP)
		r.Get("/", s.docsHandler)
		r.Get("/index.html", s.docsHandler)
		r.Get("/openapi.yml", s.openAPIHandler)
	})
}

func (s *Server) docsHandler(w http.ResponseWriter, r *http.Request) {
	htmlContent, err := s.docsFiles.ReadFile("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write(htmlContent)
}

func (s *Server) openAPIHandler(w http.ResponseWriter, r *http.Request) {
	yamlContent, err := s.docsFiles.ReadFile("openapi.yml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(yamlContent)
}
