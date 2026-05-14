package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/robfig/cron/v3"
)

const (
	githubLatestReleaseURL     = "https://api.github.com/repos/omnihance/omnihance-a3-agent/releases/latest"
	githubAcceptHeader         = "application/vnd.github+json"
	githubAPIVersionHeader     = "2026-03-10"
	versionCheckerHTTPTimeout  = 10 * time.Second
	defaultVersionCheckSeconds = 3600
)

type VersionCheckerService interface {
	Start() error
	Stop() error
	CheckNow(ctx context.Context) error
	GetStatus() VersionCheckStatus
}

type VersionCheckStatus struct {
	LatestVersion       *string
	LatestReleaseURL    *string
	VersionCheckedAt    *time.Time
	NewVersionAvailable bool
}

type versionCheckerService struct {
	cfg            *config.EnvVars
	logger         logger.Logger
	currentVersion string
	httpClient     *http.Client
	releaseURL     string
	cron           *cron.Cron
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
	status         VersionCheckStatus
}

func NewVersionCheckerService(
	cfg *config.EnvVars,
	logger logger.Logger,
	currentVersion string,
) VersionCheckerService {
	return NewVersionCheckerServiceWithClient(
		cfg,
		logger,
		currentVersion,
		githubLatestReleaseURL,
		&http.Client{Timeout: versionCheckerHTTPTimeout},
	)
}

func NewVersionCheckerServiceWithClient(
	cfg *config.EnvVars,
	logger logger.Logger,
	currentVersion string,
	releaseURL string,
	httpClient *http.Client,
) VersionCheckerService {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: versionCheckerHTTPTimeout}
	}

	return &versionCheckerService{
		cfg:            cfg,
		logger:         logger,
		currentVersion: currentVersion,
		httpClient:     httpClient,
		releaseURL:     releaseURL,
	}
}

func (v *versionCheckerService) Start() error {
	v.ctx, v.cancel = context.WithCancel(context.Background())

	v.cron = cron.New(cron.WithSeconds())
	checkSchedule := fmt.Sprintf("@every %ds", v.intervalSeconds())
	_, err := v.cron.AddFunc(checkSchedule, func() {
		if err := v.CheckNow(v.ctx); err != nil {
			v.logger.Warn("version check failed", logger.Field{Key: "error", Value: err})
		}
	})
	if err != nil {
		v.cancel()
		return fmt.Errorf("failed to schedule version check: %w", err)
	}

	v.cron.Start()
	go func() {
		if err := v.CheckNow(v.ctx); err != nil {
			v.logger.Warn("version check failed", logger.Field{Key: "error", Value: err})
		}
	}()

	v.logger.Info(
		"version checker service started",
		logger.Field{Key: "interval_seconds", Value: v.intervalSeconds()},
	)

	return nil
}

func (v *versionCheckerService) Stop() error {
	if v.cron != nil {
		ctx := v.cron.Stop()
		<-ctx.Done()
	}

	if v.cancel != nil {
		v.cancel()
	}

	v.logger.Info("version checker service stopped")

	return nil
}

func (v *versionCheckerService) CheckNow(ctx context.Context) error {
	release, err := v.fetchLatestRelease(ctx)
	checkedAt := time.Now().UTC()
	if err != nil {
		v.setStatus(VersionCheckStatus{VersionCheckedAt: &checkedAt})
		return err
	}

	isNewer, err := shouldReportNewVersion(v.currentVersion, release.TagName)
	if err != nil {
		v.setStatus(VersionCheckStatus{VersionCheckedAt: &checkedAt})
		return err
	}

	latestVersion := release.TagName
	latestReleaseURL := release.HTMLURL
	v.setStatus(VersionCheckStatus{
		LatestVersion:       &latestVersion,
		LatestReleaseURL:    &latestReleaseURL,
		VersionCheckedAt:    &checkedAt,
		NewVersionAvailable: isNewer,
	})

	return nil
}

func (v *versionCheckerService) GetStatus() VersionCheckStatus {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.status
}

func (v *versionCheckerService) fetchLatestRelease(ctx context.Context) (githubReleaseResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.releaseURL, nil)
	if err != nil {
		return githubReleaseResponse{}, fmt.Errorf("failed to create version check request: %w", err)
	}

	req.Header.Set("Accept", githubAcceptHeader)
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersionHeader)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return githubReleaseResponse{}, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return githubReleaseResponse{}, fmt.Errorf("latest release request failed with status %d", resp.StatusCode)
	}

	var release githubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubReleaseResponse{}, fmt.Errorf("failed to decode latest release response: %w", err)
	}

	if strings.TrimSpace(release.TagName) == "" {
		return githubReleaseResponse{}, fmt.Errorf("latest release tag name is empty")
	}

	if strings.TrimSpace(release.HTMLURL) == "" {
		return githubReleaseResponse{}, fmt.Errorf("latest release URL is empty")
	}

	return release, nil
}

func (v *versionCheckerService) setStatus(status VersionCheckStatus) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.status = status
}

func (v *versionCheckerService) intervalSeconds() int {
	if v.cfg == nil || v.cfg.VersionCheckIntervalSeconds <= 0 {
		return defaultVersionCheckSeconds
	}

	return v.cfg.VersionCheckIntervalSeconds
}

func shouldReportNewVersion(currentVersion string, latestVersion string) (bool, error) {
	if strings.EqualFold(strings.TrimSpace(currentVersion), "dev") {
		if _, err := parseSemanticVersion(latestVersion); err != nil {
			return false, err
		}

		return true, nil
	}

	current, err := parseSemanticVersion(currentVersion)
	if err != nil {
		return false, err
	}

	latest, err := parseSemanticVersion(latestVersion)
	if err != nil {
		return false, err
	}

	return compareSemanticVersions(latest, current) > 0, nil
}

func parseSemanticVersion(version string) (semanticVersion, error) {
	normalized := strings.TrimSpace(version)
	normalized = strings.TrimPrefix(normalized, "v")
	normalized = strings.TrimPrefix(normalized, "V")

	if normalized == "" {
		return semanticVersion{}, fmt.Errorf("version is empty")
	}

	if buildIndex := strings.Index(normalized, "+"); buildIndex >= 0 {
		normalized = normalized[:buildIndex]
	}

	preRelease := ""
	if preReleaseIndex := strings.Index(normalized, "-"); preReleaseIndex >= 0 {
		preRelease = normalized[preReleaseIndex+1:]
		normalized = normalized[:preReleaseIndex]
	}

	parts := strings.Split(normalized, ".")
	if len(parts) != 3 {
		return semanticVersion{}, fmt.Errorf("version %q is not semantic version", version)
	}

	major, err := parseVersionNumber(parts[0], version)
	if err != nil {
		return semanticVersion{}, err
	}

	minor, err := parseVersionNumber(parts[1], version)
	if err != nil {
		return semanticVersion{}, err
	}

	patch, err := parseVersionNumber(parts[2], version)
	if err != nil {
		return semanticVersion{}, err
	}

	return semanticVersion{
		major:      major,
		minor:      minor,
		patch:      patch,
		preRelease: preRelease,
	}, nil
}

func parseVersionNumber(part string, version string) (int, error) {
	if part == "" {
		return 0, fmt.Errorf("version %q contains an empty version part", version)
	}

	value, err := strconv.Atoi(part)
	if err != nil {
		return 0, fmt.Errorf("version %q contains a non-numeric version part: %w", version, err)
	}

	if value < 0 {
		return 0, fmt.Errorf("version %q contains a negative version part", version)
	}

	return value, nil
}

func compareSemanticVersions(left semanticVersion, right semanticVersion) int {
	if left.major != right.major {
		return compareInts(left.major, right.major)
	}

	if left.minor != right.minor {
		return compareInts(left.minor, right.minor)
	}

	if left.patch != right.patch {
		return compareInts(left.patch, right.patch)
	}

	return comparePreRelease(left.preRelease, right.preRelease)
}

func compareInts(left int, right int) int {
	if left > right {
		return 1
	}

	if left < right {
		return -1
	}

	return 0
}

func comparePreRelease(left string, right string) int {
	if left == "" && right == "" {
		return 0
	}

	if left == "" {
		return 1
	}

	if right == "" {
		return -1
	}

	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	maxParts := len(leftParts)
	if len(rightParts) > maxParts {
		maxParts = len(rightParts)
	}

	for i := 0; i < maxParts; i++ {
		if i >= len(leftParts) {
			return -1
		}

		if i >= len(rightParts) {
			return 1
		}

		partCompare := comparePreReleasePart(leftParts[i], rightParts[i])
		if partCompare != 0 {
			return partCompare
		}
	}

	return 0
}

func comparePreReleasePart(left string, right string) int {
	leftInt, leftErr := strconv.Atoi(left)
	rightInt, rightErr := strconv.Atoi(right)
	if leftErr == nil && rightErr == nil {
		return compareInts(leftInt, rightInt)
	}

	if leftErr == nil {
		return -1
	}

	if rightErr == nil {
		return 1
	}

	return strings.Compare(left, right)
}

type githubReleaseResponse struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

type semanticVersion struct {
	major      int
	minor      int
	patch      int
	preRelease string
}
