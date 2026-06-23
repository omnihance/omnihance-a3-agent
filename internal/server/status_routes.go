package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

func (s *Server) InitializeStatusRoutes(r *chi.Mux) {
	r.Get("/api/status", s.statusHandler)
}

func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	adminUserCount, _ := s.internalDB.GetAdminUserCount()
	setUpDone := adminUserCount > 0
	versionCheckStatus := s.getVersionCheckStatus()

	_ = utils.WriteJSONResponse(w, StatusResponse{
		Name:                   "omnihance-a3-agent",
		Version:                s.version,
		SetupDone:              setUpDone,
		NewVersionAvailable:    versionCheckStatus.NewVersionAvailable,
		LatestVersion:          versionCheckStatus.LatestVersion,
		LatestReleaseURL:       versionCheckStatus.LatestReleaseURL,
		VersionCheckedAt:       versionCheckStatus.VersionCheckedAt,
		MetricsEnabled:         s.cfg.MetricsEnabled,
		MaxFileUploadSizeBytes: s.cfg.MaxFileUploadSizeBytes(),
	})
}

type StatusResponse struct {
	Name                   string     `json:"name"`
	Version                string     `json:"version"`
	SetupDone              bool       `json:"setup_done"`
	NewVersionAvailable    bool       `json:"new_version_available"`
	LatestVersion          *string    `json:"latest_version"`
	LatestReleaseURL       *string    `json:"latest_release_url"`
	VersionCheckedAt       *time.Time `json:"version_checked_at"`
	MetricsEnabled         bool       `json:"metrics_enabled"`
	MaxFileUploadSizeBytes int64      `json:"max_file_upload_size_bytes"`
}

func (s *Server) getVersionCheckStatus() services.VersionCheckStatus {
	if s.versionChecker == nil {
		return services.VersionCheckStatus{}
	}

	return s.versionChecker.GetStatus()
}
