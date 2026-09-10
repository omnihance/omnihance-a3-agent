package main

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/omnihance/omnihance-a3-agent/internal/config"
	"github.com/omnihance/omnihance-a3-agent/internal/db"
	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/omnihance/omnihance-a3-agent/internal/server"
	"github.com/omnihance/omnihance-a3-agent/internal/services"
)

//go:embed omnihance-a3-agent-ui/dist/*
var frontendFiles embed.FS

//go:embed docs/*
var docsFiles embed.FS

var version = "dev"

const shutdownTimeout = 30 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	context.AfterFunc(ctx, stop)
	code := run(ctx)
	stop()
	os.Exit(code)
}

func run(ctx context.Context) int {
	if ctx.Err() != nil {
		return 0
	}

	cfg := config.New()
	log := logger.NewZerologFileLogger("omnihance-a3-agent", cfg.LogDir, cfg.GetLogLevel())
	defer func() {
		_ = log.Close()
	}()

	if ctx.Err() != nil {
		return 0
	}

	internalDB := db.NewSQLiteDB(cfg.DatabaseURL, log)
	if err := internalDB.Connect(); err != nil {
		log.Error("Could not connect to internal database", logger.Field{Key: "error", Value: err})
		return 1
	}

	defer func() {
		_ = internalDB.Close()
	}()

	if ctx.Err() != nil {
		return 0
	}

	if err := internalDB.MigrateUp(); err != nil {
		log.Error("Could not migrate internal database", logger.Field{Key: "error", Value: err})
		return 1
	}

	if ctx.Err() != nil {
		return 0
	}

	if cfg.MetricsEnabled {
		metricsCollector := services.NewMetricsCollectorService(cfg, log, internalDB)
		if err := metricsCollector.Start(); err != nil {
			log.Error("Could not start metrics collector service", logger.Field{Key: "error", Value: err})
			return 1
		}

		defer func() {
			_ = metricsCollector.Stop()
		}()
	}

	if ctx.Err() != nil {
		return 0
	}

	log.Info(
		"Starting Omnihance A3 Agent on port "+cfg.Port,
		logger.Field{Key: "port", Value: cfg.Port},
		logger.Field{Key: "log_level", Value: cfg.GetLogLevel().String()},
		logger.Field{Key: "version", Value: version},
	)

	fileEditor := services.NewFileEditorService(log)
	processService := services.NewProcessService(log)
	serverManagerService := services.NewServerManagerService(internalDB, processService, log)
	backupService := services.NewBackupService(cfg, log, internalDB, fileEditor)
	if err := backupService.Start(); err != nil {
		log.Error("Could not start backup service", logger.Field{Key: "error", Value: err})
		return 1
	}

	defer func() {
		_ = backupService.Stop()
	}()

	if ctx.Err() != nil {
		return 0
	}

	serverViewService := services.NewServerViewService(log, internalDB, fileEditor)
	if err := serverViewService.Start(); err != nil {
		log.Error("Could not start server view service", logger.Field{Key: "error", Value: err})
		return 1
	}

	if ctx.Err() != nil {
		return 0
	}

	versionChecker := services.NewVersionCheckerService(cfg, log, version)
	if err := versionChecker.Start(); err != nil {
		log.Error("Could not start version checker service", logger.Field{Key: "error", Value: err})
		return 1
	}

	defer func() {
		_ = versionChecker.Stop()
	}()

	if ctx.Err() != nil {
		return 0
	}

	server := server.NewServer(
		cfg, log,
		frontendFiles,
		docsFiles,
		version,
		internalDB,
		fileEditor,
		processService,
		serverManagerService,
		versionChecker,
		backupService,
		serverViewService,
	)
	defer func() {
		_ = server.Close()
	}()

	serveErr := make(chan error, 1)
	if ctx.Err() == nil {
		go func() {
			serveErr <- server.ListenAndServe()
		}()
	}

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("Could not start Omnihance A3 Agent server", logger.Field{Key: "error", Value: err})
			return 1
		}
	case <-ctx.Done():
	}

	log.Info("Omnihance A3 Agent shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("Could not shut down Omnihance A3 Agent server", logger.Field{Key: "error", Value: err})
		return 1
	}

	return 0
}
