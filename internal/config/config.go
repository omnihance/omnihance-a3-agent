package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/joho/godotenv/autoload"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
	"github.com/rs/zerolog"
)

type EnvVars struct {
	Port                             string
	LogLevel                         string
	LogDir                           string
	DatabaseURL                      string
	MetricsCollectionIntervalSeconds int
	MetricsRetentionDays             int
	MetricsCleanupIntervalSeconds    int
	RevisionsDirectory               string
	MetricsEnabled                   bool
	SessionTimeoutSeconds            int
	CookieSecret                     string
	MaxFileUploadSizeMb              int
}

var defaultEnvVars = map[string]string{
	"PORT":                                "8080",
	"LOG_LEVEL":                           "info",
	"LOG_DIR":                             "logs",
	"DATABASE_URL":                        "file:omnihance-a3-agent.db?cache=shared&mode=rwc",
	"METRICS_COLLECTION_INTERVAL_SECONDS": "60",
	"METRICS_RETENTION_DAYS":              "7",
	"METRICS_CLEANUP_INTERVAL_SECONDS":    "3600",
	"REVISIONS_DIRECTORY":                 ".revisions",
	"METRICS_ENABLED":                     "true",
	"SESSION_TIMEOUT_SECONDS":             fmt.Sprintf("%d", 60*60*24*30),
	"COOKIE_SECRET":                       utils.GenerateRandomToken(32),
}

func New() *EnvVars {
	GenerateEnvFile()
	for key, value := range defaultEnvVars {
		if _, ok := os.LookupEnv(key); !ok {
			err := os.Setenv(key, value)
			if err != nil {
				slog.Info("Could not set default " + key + "! " + err.Error())
			}
		}
	}

	metricsCollectionIntervalSeconds, err := strconv.Atoi(os.Getenv("METRICS_COLLECTION_INTERVAL_SECONDS"))
	if err != nil {
		slog.Warn("Could not get metrics collection interval seconds: " + err.Error())
		metricsCollectionIntervalSeconds = 60
	}

	metricsRetentionDays, err := strconv.Atoi(os.Getenv("METRICS_RETENTION_DAYS"))
	if err != nil {
		slog.Warn("Could not get metrics retention days: " + err.Error())
		metricsRetentionDays = 30
	}

	metricsCleanupIntervalSeconds, err := strconv.Atoi(os.Getenv("METRICS_CLEANUP_INTERVAL_SECONDS"))
	if err != nil {
		slog.Warn("Could not get metrics cleanup interval seconds: " + err.Error())
		metricsCleanupIntervalSeconds = 3600
	}

	metricsEnabled, err := strconv.ParseBool(os.Getenv("METRICS_ENABLED"))
	if err != nil {
		slog.Warn("Could not get metrics enabled: " + err.Error())
		metricsEnabled = true
	}

	sessionTimeoutSeconds, err := strconv.Atoi(os.Getenv("SESSION_TIMEOUT_SECONDS"))
	if err != nil {
		slog.Warn("Could not get session timeout seconds: " + err.Error())
		sessionTimeoutSeconds = 60 * 60 * 24 * 30
	}

	cookieSecret := os.Getenv("COOKIE_SECRET")
	if cookieSecret == "" {
		slog.Warn("Cookie secret is not set, generating a new one")
		cookieSecret = utils.GenerateRandomToken(32)
	}

	maxFileUploadSizeMb, err := strconv.Atoi(os.Getenv("MAX_FILE_UPLOAD_SIZE_MB"))
	if err != nil {
		slog.Warn("Could not get max file upload size: " + err.Error())
		maxFileUploadSizeMb = 2
	}

	return &EnvVars{
		Port:                             os.Getenv("PORT"),
		LogLevel:                         os.Getenv("LOG_LEVEL"),
		LogDir:                           os.Getenv("LOG_DIR"),
		DatabaseURL:                      os.Getenv("DATABASE_URL"),
		MetricsCollectionIntervalSeconds: metricsCollectionIntervalSeconds,
		MetricsRetentionDays:             metricsRetentionDays,
		MetricsCleanupIntervalSeconds:    metricsCleanupIntervalSeconds,
		RevisionsDirectory:               os.Getenv("REVISIONS_DIRECTORY"),
		MetricsEnabled:                   metricsEnabled,
		SessionTimeoutSeconds:            sessionTimeoutSeconds,
		CookieSecret:                     cookieSecret,
		MaxFileUploadSizeMb:              maxFileUploadSizeMb,
	}
}

func GenerateEnvFile() {
	envPath := filepath.Join(".", ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if err := writeEnvFile(envPath, defaultEnvVars); err != nil {
			slog.Warn("Could not create .env file: " + err.Error())
		} else {
			slog.Info("Created .env file with default values")
		}
	}
}

func (e *EnvVars) GetLogLevel() zerolog.Level {
	switch e.LogLevel {
	case "verbose":
		fallthrough
	case "silly":
		fallthrough
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

func writeEnvFile(path string, envVars map[string]string) error {
	var builder strings.Builder
	for key, value := range envVars {
		builder.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	return os.WriteFile(path, []byte(builder.String()), 0644)
}
