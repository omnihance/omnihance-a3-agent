package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestShouldReportNewVersion(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		latestVersion  string
		expected       bool
		expectError    bool
	}{
		{
			name:           "dev reports latest release",
			currentVersion: "dev",
			latestVersion:  "v1.2.3",
			expected:       true,
		},
		{
			name:           "newer semantic version reports update",
			currentVersion: "1.0.0",
			latestVersion:  "1.1.0",
			expected:       true,
		},
		{
			name:           "matching versions with v prefix report no update",
			currentVersion: "v1.1.0",
			latestVersion:  "1.1.0",
			expected:       false,
		},
		{
			name:           "older latest version reports no update",
			currentVersion: "1.2.0",
			latestVersion:  "1.1.0",
			expected:       false,
		},
		{
			name:           "release version beats prerelease version",
			currentVersion: "1.2.0-beta.1",
			latestVersion:  "1.2.0",
			expected:       true,
		},
		{
			name:           "invalid current version fails closed",
			currentVersion: "not-a-version",
			latestVersion:  "1.2.0",
			expectError:    true,
		},
		{
			name:           "invalid latest version fails closed",
			currentVersion: "1.2.0",
			latestVersion:  "latest",
			expectError:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := shouldReportNewVersion(test.currentVersion, test.latestVersion)
			if test.expectError {
				require.Error(t, err)
				require.False(t, actual)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		})
	}
}

func TestVersionCheckerCheckNowSuccess(t *testing.T) {
	requestCount := 0
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		require.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))
		require.Equal(t, "2026-03-10", r.Header.Get("X-GitHub-Api-Version"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"tag_name":"v1.1.0","html_url":"https://github.com/omnihance/omnihance-a3-agent/releases/tag/v1.1.0"}`)
	}))
	defer testServer.Close()

	service := newTestVersionCheckerService("1.0.0", testServer.URL, testServer.Client())
	err := service.CheckNow(context.Background())
	require.NoError(t, err)

	status := service.GetStatus()
	require.True(t, status.NewVersionAvailable)
	require.NotNil(t, status.LatestVersion)
	require.Equal(t, "v1.1.0", *status.LatestVersion)
	require.NotNil(t, status.LatestReleaseURL)
	require.Equal(t, "https://github.com/omnihance/omnihance-a3-agent/releases/tag/v1.1.0", *status.LatestReleaseURL)
	require.NotNil(t, status.VersionCheckedAt)
	require.Equal(t, 1, requestCount)
}

func TestVersionCheckerCheckNowDevVersion(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"tag_name":"v2.0.0","html_url":"https://github.com/omnihance/omnihance-a3-agent/releases/tag/v2.0.0"}`)
	}))
	defer testServer.Close()

	service := newTestVersionCheckerService("dev", testServer.URL, testServer.Client())
	err := service.CheckNow(context.Background())
	require.NoError(t, err)

	status := service.GetStatus()
	require.True(t, status.NewVersionAvailable)
	require.NotNil(t, status.LatestVersion)
	require.Equal(t, "v2.0.0", *status.LatestVersion)
}

func TestVersionCheckerCheckNowFailureClearsStatus(t *testing.T) {
	shouldFail := false
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldFail {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"tag_name":"v1.1.0","html_url":"https://github.com/omnihance/omnihance-a3-agent/releases/tag/v1.1.0"}`)
	}))
	defer testServer.Close()

	service := newTestVersionCheckerService("1.0.0", testServer.URL, testServer.Client())
	require.NoError(t, service.CheckNow(context.Background()))
	require.True(t, service.GetStatus().NewVersionAvailable)

	shouldFail = true
	err := service.CheckNow(context.Background())
	require.Error(t, err)

	status := service.GetStatus()
	require.False(t, status.NewVersionAvailable)
	require.Nil(t, status.LatestVersion)
	require.Nil(t, status.LatestReleaseURL)
	require.NotNil(t, status.VersionCheckedAt)
}

func TestVersionCheckerStartUsesConfiguredInterval(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"tag_name":"v1.1.0","html_url":"https://github.com/omnihance/omnihance-a3-agent/releases/tag/v1.1.0"}`)
	}))
	defer testServer.Close()

	service := NewVersionCheckerServiceWithClient(
		&config.EnvVars{VersionCheckIntervalSeconds: 1},
		newTestVersionCheckerLogger(),
		"1.0.0",
		testServer.URL,
		testServer.Client(),
	)

	require.NoError(t, service.Start())
	t.Cleanup(func() {
		_ = service.Stop()
	})

	require.Eventually(t, func() bool {
		status := service.GetStatus()
		return status.NewVersionAvailable && status.VersionCheckedAt != nil
	}, 2*time.Second, 25*time.Millisecond)
}

func newTestVersionCheckerService(
	currentVersion string,
	releaseURL string,
	httpClient *http.Client,
) VersionCheckerService {
	return NewVersionCheckerServiceWithClient(
		&config.EnvVars{VersionCheckIntervalSeconds: 3600},
		newTestVersionCheckerLogger(),
		currentVersion,
		releaseURL,
		httpClient,
	)
}

func newTestVersionCheckerLogger() logger.Logger {
	return logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
}
