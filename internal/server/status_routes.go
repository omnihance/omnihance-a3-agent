package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

func (s *Server) InitializeStatusRoutes(r *chi.Mux) {
	r.Get("/api/status", s.statusHandler)
}

func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	adminUserCount, _ := s.internalDB.GetAdminUserCount()
	setUpDone := false
	if adminUserCount > 0 {
		setUpDone = true
	}

	_ = utils.WriteJSONResponse(w, StatusResponse{
		Name:                "omnihance-a3-agent",
		Version:             s.version,
		SetupDone:           setUpDone,
		NewVersionAvailable: false,
		MetricsEnabled:      s.cfg.MetricsEnabled,
	})
}

type StatusResponse struct {
	Name                string `json:"name"`
	Version             string `json:"version"`
	SetupDone           bool   `json:"setup_done"`
	NewVersionAvailable bool   `json:"new_version_available"`
	MetricsEnabled      bool   `json:"metrics_enabled"`
}
