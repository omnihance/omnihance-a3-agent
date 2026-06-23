package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
	"github.com/stretchr/testify/require"
)

func TestStatusHandlerReturnsCachedVersionCheckStatus(t *testing.T) {
	internalDB := newTestInternalDB(t)
	latestVersion := "v1.1.0"
	latestReleaseURL := "https://github.com/omnihance/omnihance-a3-agent/releases/tag/v1.1.0"
	checkedAt := time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)
	server := &Server{
		cfg:        &config.EnvVars{MetricsEnabled: true, MaxFileUploadSizeMb: 512},
		version:    "1.0.0",
		internalDB: internalDB,
		versionChecker: fakeVersionCheckerService{
			status: services.VersionCheckStatus{
				LatestVersion:       &latestVersion,
				LatestReleaseURL:    &latestReleaseURL,
				VersionCheckedAt:    &checkedAt,
				NewVersionAvailable: true,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rr := httptest.NewRecorder()

	server.statusHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response StatusResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
	require.Equal(t, "omnihance-a3-agent", response.Name)
	require.Equal(t, "1.0.0", response.Version)
	require.True(t, response.NewVersionAvailable)
	require.NotNil(t, response.LatestVersion)
	require.Equal(t, latestVersion, *response.LatestVersion)
	require.NotNil(t, response.LatestReleaseURL)
	require.Equal(t, latestReleaseURL, *response.LatestReleaseURL)
	require.NotNil(t, response.VersionCheckedAt)
	require.Equal(t, checkedAt, *response.VersionCheckedAt)
	require.True(t, response.MetricsEnabled)
	require.Equal(t, int64(512*1024*1024), response.MaxFileUploadSizeBytes)
}

type fakeVersionCheckerService struct {
	status services.VersionCheckStatus
}

func (f fakeVersionCheckerService) Start() error {
	return nil
}

func (f fakeVersionCheckerService) Stop() error {
	return nil
}

func (f fakeVersionCheckerService) CheckNow(ctx context.Context) error {
	return nil
}

func (f fakeVersionCheckerService) GetStatus() services.VersionCheckStatus {
	return f.status
}
